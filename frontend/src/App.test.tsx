import { describe, it, expect, vi, beforeEach } from 'vitest';
import { useState } from 'react';
import { render, screen, fireEvent, waitFor, within } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import App from './App';
import { AppProviders } from './test/providers';

const { logout, teamPanel } = vi.hoisted(() => ({
  logout: vi.fn(),
  teamPanel: { shouldThrow: false },
}));

vi.mock('./contexts/AuthContext', () => ({
  useAuth: () => ({ isAuthenticated: true, isLoading: false, user: { id: 1, name: '홍길동' }, logout }),
}));

// 입력하면 onDirtyChange(true)를 알리는 가짜 작성 폼
vi.mock('./components/ReportForm', () => ({
  default: ({ onDirtyChange }: { onDirtyChange?: (d: boolean) => void }) => {
    const [value, setValue] = useState('');
    return (
      <input
        aria-label="보고서 입력"
        value={value}
        onChange={(e) => { setValue(e.target.value); onDirtyChange?.(e.target.value !== ''); }}
      />
    );
  },
}));
vi.mock('./components/SiteReportForm', () => ({ default: () => <div>사이트 폼</div> }));
vi.mock('./components/TeamPanel', () => ({
  default: () => {
    if (teamPanel.shouldThrow) throw new Error('팀 화면 오류');
    return <div>팀 화면</div>;
  },
}));

const clickTab = (name: string) => fireEvent.click(screen.getAllByRole('tab', { name })[0]);

function renderApp() {
  return render(<MemoryRouter><App /></MemoryRouter>, { wrapper: AppProviders });
}

describe('탭 전환 시 작성 내용 보존', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.restoreAllMocks();
    teamPanel.shouldThrow = false;
  });

  it('저장 안 된 변경이 있으면 다른 탭에 다녀와도 입력이 남아 있다', async () => {
    renderApp();
    fireEvent.change(await screen.findByLabelText('보고서 입력'), { target: { value: '작성 중' } });

    clickTab('팀');
    expect(await screen.findByText('팀 화면')).toBeInTheDocument();
    expect(screen.getByLabelText('보고서 입력')).not.toBeVisible();

    clickTab('보고서 작성');
    expect(screen.getByLabelText('보고서 입력')).toHaveValue('작성 중');
    expect(screen.queryByText('팀 화면')).not.toBeInTheDocument();
  });

  it('변경이 없으면 기존처럼 언마운트된다', async () => {
    renderApp();
    await screen.findByLabelText('보고서 입력');

    clickTab('팀');
    await screen.findByText('팀 화면');
    expect(screen.queryByLabelText('보고서 입력')).not.toBeInTheDocument();
  });

  it('다른 탭에서 렌더링 오류가 나도 숨겨둔 작성 폼은 유지된다', async () => {
    vi.spyOn(console, 'error').mockImplementation(() => {});
    teamPanel.shouldThrow = true;
    renderApp();
    fireEvent.change(await screen.findByLabelText('보고서 입력'), { target: { value: '작성 중' } });

    clickTab('팀');
    expect(screen.queryByText('팀 화면')).not.toBeInTheDocument();

    clickTab('보고서 작성');
    expect(screen.getByLabelText('보고서 입력')).toHaveValue('작성 중');
  });

  it('저장 안 된 변경이 있을 때만 페이지 이탈을 경고한다', async () => {
    renderApp();
    const input = await screen.findByLabelText('보고서 입력');

    const before = new Event('beforeunload', { cancelable: true });
    window.dispatchEvent(before);
    expect(before.defaultPrevented).toBe(false);

    fireEvent.change(input, { target: { value: '작성 중' } });
    const after = new Event('beforeunload', { cancelable: true });
    window.dispatchEvent(after);
    expect(after.defaultPrevented).toBe(true);
  });

  it('저장 안 된 변경이 없으면 바로 로그아웃하고, 이 사용자의 임시 초안을 지운다', async () => {
    localStorage.setItem('jugan_draft_1_2026-10-06', '{}');
    localStorage.setItem('jugan_draft_2_2026-10-06', '{}');
    renderApp();
    await screen.findByLabelText('보고서 입력');

    fireEvent.click(screen.getByRole('button', { name: '로그아웃' }));
    await waitFor(() => expect(logout).toHaveBeenCalledTimes(1));
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument();
    expect(localStorage.getItem('jugan_draft_1_2026-10-06')).toBeNull();
    expect(localStorage.getItem('jugan_draft_2_2026-10-06')).not.toBeNull();
  });

  it('저장 안 된 변경이 있으면 로그아웃 전에 확인하고, 취소하면 남는다', async () => {
    renderApp();
    fireEvent.change(await screen.findByLabelText('보고서 입력'), { target: { value: '작성 중' } });

    fireEvent.click(screen.getByRole('button', { name: '로그아웃' }));
    const dialog = await screen.findByRole('alertdialog', { name: '저장하지 않은 내용이 있습니다' });
    fireEvent.click(within(dialog).getByRole('button', { name: '취소' }));
    await waitFor(() => expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument());
    expect(logout).not.toHaveBeenCalled();

    fireEvent.click(screen.getByRole('button', { name: '로그아웃' }));
    fireEvent.click(within(await screen.findByRole('alertdialog')).getByRole('button', { name: '로그아웃' }));
    await waitFor(() => expect(logout).toHaveBeenCalledTimes(1));
  });
});
