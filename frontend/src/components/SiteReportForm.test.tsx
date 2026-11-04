import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, act, waitFor, within } from '@testing-library/react';
import SiteReportForm from './SiteReportForm';
import * as api from '../services/api';
import { AppProviders } from '../test/providers';

vi.mock('../services/api', () => ({
  getMyTeams: vi.fn(),
  getMySiteProjects: vi.fn(),
  getMySiteReport: vi.fn(),
  saveSiteReport: vi.fn(),
}));

const mocked = api as unknown as Record<string, ReturnType<typeof vi.fn>>;

const lastDirty = (fn: ReturnType<typeof vi.fn>) => fn.mock.calls[fn.mock.calls.length - 1]?.[0];

const reportWithNotes = (notes: string) => ({
  exists: true,
  report: { report_date_text: '', this_week: [], next_week: [], notes },
});

const dateInput = (container: HTMLElement) => container.querySelector('input[type="date"]') as HTMLInputElement;

describe('SiteReportForm 저장 안 된 변경 감지', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocked.getMyTeams.mockResolvedValue([{ id: 1, name: '플랫폼팀' }]);
    mocked.getMySiteProjects.mockResolvedValue([{ id: 7, team_id: 1, project_name: '도공 사이트', authors: [] }]);
  });

  it('불러온 직후 변경 없음, 입력하면 변경 있음, 저장하면 변경 없음', async () => {
    mocked.getMySiteReport.mockResolvedValue(reportWithNotes('기존 메모'));
    mocked.saveSiteReport.mockResolvedValue({});
    const onDirtyChange = vi.fn();
    render(<SiteReportForm onDirtyChange={onDirtyChange} />, { wrapper: AppProviders });

    const notes = await screen.findByDisplayValue('기존 메모');
    expect(lastDirty(onDirtyChange)).toBe(false);

    fireEvent.change(notes, { target: { value: '수정한 메모' } });
    expect(lastDirty(onDirtyChange)).toBe(true);

    fireEvent.click(screen.getByRole('button', { name: '저장' }));
    await screen.findByText('저장되었습니다.');
    expect(lastDirty(onDirtyChange)).toBe(false);
  });

  it('대상 주를 빠르게 바꾸면 늦게 도착한 이전 응답은 무시한다', async () => {
    const pending: Record<string, (v: unknown) => void> = {};
    mocked.getMySiteReport.mockImplementation(
      (_t: number, _p: number, date: string) => new Promise((resolve) => { pending[date] = resolve; }),
    );
    const onDirtyChange = vi.fn();
    const { container } = render(<SiteReportForm onDirtyChange={onDirtyChange} />, { wrapper: AppProviders });
    await screen.findByText('도공 사이트');

    await act(async () => { fireEvent.change(dateInput(container), { target: { value: '2026-09-04' } }); fireEvent.blur(dateInput(container)); });
    await act(async () => { fireEvent.change(dateInput(container), { target: { value: '2026-09-11' } }); fireEvent.blur(dateInput(container)); });

    await act(async () => { pending['2026-09-11'](reportWithNotes('11일 주 메모')); });
    await act(async () => { pending['2026-09-04'](reportWithNotes('4일 주 메모')); });

    expect(screen.getByDisplayValue('11일 주 메모')).toBeInTheDocument();
    expect(screen.queryByDisplayValue('4일 주 메모')).not.toBeInTheDocument();
    expect(lastDirty(onDirtyChange)).toBe(false);
  });

  it('보고서를 불러오지 못하면 저장을 막는다 (이전 내용으로 다른 보고서를 덮어쓰지 않도록)', async () => {
    mocked.getMySiteReport.mockResolvedValueOnce(reportWithNotes('기존 메모'));
    const { container } = render(<SiteReportForm />, { wrapper: AppProviders });
    await screen.findByDisplayValue('기존 메모');

    mocked.getMySiteReport.mockRejectedValueOnce(new Error('network'));
    await act(async () => { fireEvent.change(dateInput(container), { target: { value: '2026-09-04' } }); });

    expect(await screen.findByText(/보고서를 불러오지 못했습니다/)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '저장' })).toBeDisabled();

    // 다른 주를 골라 정상 조회되면 안내가 사라지고 저장 가능
    mocked.getMySiteReport.mockResolvedValueOnce(reportWithNotes('11일 주 메모'));
    await act(async () => { fireEvent.change(dateInput(container), { target: { value: '2026-09-11' } }); fireEvent.blur(dateInput(container)); });
    await screen.findByDisplayValue('11일 주 메모');
    expect(screen.queryByText(/보고서를 불러오지 못했습니다/)).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: '저장' })).toBeEnabled();
  });

  it('작성 중에 대상 주를 바꾸면 확인하고, 취소하면 내용을 유지한다', async () => {
    mocked.getMySiteReport.mockResolvedValue(reportWithNotes('기존 메모'));
    const { container } = render(<SiteReportForm />, { wrapper: AppProviders });
    fireEvent.change(await screen.findByDisplayValue('기존 메모'), { target: { value: '작성 중' } });
    const before = dateInput(container).value;

    fireEvent.change(dateInput(container), { target: { value: '2026-09-04' } });
    const dialog = await screen.findByRole('alertdialog', { name: '저장하지 않은 내용이 있습니다' });
    fireEvent.click(within(dialog).getByRole('button', { name: '취소' }));

    await waitFor(() => expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument());
    expect(dateInput(container).value).toBe(before);
    expect(screen.getByDisplayValue('작성 중')).toBeInTheDocument();
    expect(mocked.getMySiteReport).toHaveBeenCalledTimes(1);
  });

  it('숨김 유지 중 작성 권한이 없어지면 알리고 변경 상태를 풀어준다', async () => {
    mocked.getMySiteReport.mockResolvedValue(reportWithNotes('기존 메모'));
    const onDirtyChange = vi.fn();
    const { rerender } = render(<SiteReportForm onDirtyChange={onDirtyChange} active />, { wrapper: AppProviders });
    fireEvent.change(await screen.findByDisplayValue('기존 메모'), { target: { value: '작성 중' } });
    expect(lastDirty(onDirtyChange)).toBe(true);

    rerender(<SiteReportForm onDirtyChange={onDirtyChange} active={false} />);
    mocked.getMySiteProjects.mockResolvedValue([{ id: 8, team_id: 1, project_name: '다른 사이트', authors: [] }]);
    rerender(<SiteReportForm onDirtyChange={onDirtyChange} active />);

    expect(await screen.findByText(/작성 권한이 없어져 저장할 수 없습니다/)).toBeInTheDocument();
    expect(lastDirty(onDirtyChange)).toBe(false);
    expect(screen.getByDisplayValue('작성 중')).toBeInTheDocument();
  });

  it('숨김 유지 중 유일한 배정까지 없어지면 안내 화면 대신 작성 내용과 오류를 보여준다', async () => {
    mocked.getMySiteReport.mockResolvedValue(reportWithNotes('기존 메모'));
    const { rerender } = render(<SiteReportForm active />, { wrapper: AppProviders });
    fireEvent.change(await screen.findByDisplayValue('기존 메모'), { target: { value: '작성 중' } });

    rerender(<SiteReportForm active={false} />);
    mocked.getMySiteProjects.mockResolvedValue([]);
    rerender(<SiteReportForm active />);

    expect(await screen.findByText(/작성 권한이 없어져 저장할 수 없습니다/)).toBeInTheDocument();
    expect(screen.getByDisplayValue('작성 중')).toBeInTheDocument();
    expect(screen.queryByText('작성자로 등록된 사이트 프로젝트가 없습니다.')).not.toBeInTheDocument();
  });

  it('대상 주를 키보드로 입력하는 중간값으로는 보고서를 다시 불러오지 않는다', async () => {
    mocked.getMySiteReport.mockResolvedValue(reportWithNotes('기존 메모'));
    const { container } = render(<SiteReportForm />, { wrapper: AppProviders });
    await screen.findByDisplayValue('기존 메모');

    await act(async () => { fireEvent.change(dateInput(container), { target: { value: '0002-09-04' } }); });
    expect(dateInput(container).value).toBe('0002-09-04');
    expect(mocked.getMySiteReport).toHaveBeenCalledTimes(1);

    await act(async () => { fireEvent.change(dateInput(container), { target: { value: '2026-09-04' } }); fireEvent.blur(dateInput(container)); });
    expect(mocked.getMySiteReport).toHaveBeenCalledTimes(2);
  });

  it('조회에 실패하면 변경 상태를 풀어준다 (저장할 수 없는 상태로 붙잡아 두지 않음)', async () => {
    mocked.getMySiteReport.mockResolvedValueOnce(reportWithNotes('기존 메모'));
    const onDirtyChange = vi.fn();
    const { container } = render(<SiteReportForm onDirtyChange={onDirtyChange} />, { wrapper: AppProviders });
    fireEvent.change(await screen.findByDisplayValue('기존 메모'), { target: { value: '작성 중' } });
    expect(lastDirty(onDirtyChange)).toBe(true);

    mocked.getMySiteReport.mockRejectedValueOnce(new Error('network'));
    fireEvent.change(dateInput(container), { target: { value: '2026-09-04' } });
    fireEvent.click(within(await screen.findByRole('alertdialog')).getByRole('button', { name: '바꾸기' }));

    await screen.findByText(/보고서를 불러오지 못했습니다/);
    expect(lastDirty(onDirtyChange)).toBe(false);
  });
});
