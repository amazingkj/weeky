import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import ClientHistoryPanel from './ClientHistoryPanel';
import * as api from '../services/api';

vi.mock('../services/api', () => ({
  getClientHistory: vi.fn(),
}));

const mocked = api as unknown as Record<string, ReturnType<typeof vi.fn>>;

const entry = (over: Record<string, unknown>) => ({
  client: '삼성카드', report_date: '2026-09-25', kind: 'report', section: 'this_week',
  authors: ['홍길동'], project: 'CruzAPIM', work: '인증 개선', progress: '50', due_date: '', ...over,
});

describe('ClientHistoryPanel', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocked.getClientHistory.mockResolvedValue({
      from: '2026-07-06', to: '2026-10-04',
      entries: [
        entry({}),
        entry({ section: 'next_week', work: '운영 배포', progress: '' }),
        entry({ client: '도로교통공단', report_date: '2026-09-18', kind: 'site', project: '도공 유지보수', work: '장애 대응', authors: ['김', '이'], progress: '100' }),
        entry({ report_date: '2026-09-18', authors: ['이순신'], work: 'API 설계', progress: '30' }),
      ],
    });
  });

  it('최근 고객사를 기본 선택하고 주차별 금주/차주로 나눠 보여준다', async () => {
    render(<ClientHistoryPanel teamId={1} />);

    expect(await screen.findByText('인증 개선')).toBeInTheDocument();
    expect(screen.getByText('운영 배포')).toBeInTheDocument();
    expect(screen.getByText('API 설계')).toBeInTheDocument();
    expect(screen.getAllByText('금주실적')).toHaveLength(2); // 9/25 주, 9/18 주
    expect(screen.getByText('차주계획')).toBeInTheDocument();
    expect(screen.queryByText('장애 대응')).not.toBeInTheDocument();
  });

  it('고객사를 누르면 해당 고객사 업무만 보인다', async () => {
    render(<ClientHistoryPanel teamId={1} />);
    fireEvent.click(await screen.findByText('도로교통공단'));

    expect(screen.getByText('장애 대응')).toBeInTheDocument();
    expect(screen.getByText('사이트')).toBeInTheDocument();
    expect(screen.getByText('김, 이')).toBeInTheDocument();
    // 작성자 2명은 각각 집계
    expect(screen.getByText('1건 · 2명')).toBeInTheDocument();
    expect(screen.queryByText('인증 개선')).not.toBeInTheDocument();
  });

  it('기간을 바꾸면 해당 주 수로 다시 조회한다', async () => {
    render(<ClientHistoryPanel teamId={1} />);
    fireEvent.change(await screen.findByLabelText('조회 기간'), { target: { value: '24' } });
    expect(mocked.getClientHistory).toHaveBeenLastCalledWith(1, 24);
  });
});
