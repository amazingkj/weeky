import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor, act, within } from '@testing-library/react';
import ReportForm from './ReportForm';
import * as api from '../services/api';
import { AppProviders } from '../test/providers';
import { saveDraft, loadDraft } from '../utils/reportDraft';
import { toLocalYMD } from '../utils/date';

vi.mock('../services/api', () => ({
  getConfig: vi.fn(),
  saveReport: vi.fn(),
  getMyTeams: vi.fn(),
  getReports: vi.fn(),
  getMySubmission: vi.fn(),
  submitReport: vi.fn(),
  unsubmitReport: vi.fn(),
  getTeamProjects: vi.fn(),
  autoCreateTeamProject: vi.fn(),
}));
vi.mock('../contexts/AuthContext', () => ({
  useAuth: () => ({ user: { id: 1, name: '홍길동' } }),
}));
vi.mock('../utils/pptGenerator', () => ({ generatePPT: vi.fn() }));
vi.mock('./SyncPanel', () => ({ default: () => null }));
vi.mock('./PptPreview', () => ({ default: () => null }));

const mocked = api as unknown as Record<string, ReturnType<typeof vi.fn>>;

const lastDirty = (fn: ReturnType<typeof vi.fn>) => fn.mock.calls[fn.mock.calls.length - 1]?.[0];

describe('ReportForm 저장 안 된 변경 감지', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
    mocked.getConfig.mockResolvedValue({});
    mocked.getMyTeams.mockResolvedValue([{ id: 1, name: '플랫폼팀' }]);
    mocked.getMySubmission.mockResolvedValue({ submitted: false });
    mocked.getTeamProjects.mockResolvedValue([]);
    // 지난주 차주계획 → 이번 주 금주실적으로 이월됨
    const lastWeek = new Date();
    lastWeek.setDate(lastWeek.getDate() - 7);
    const y = lastWeek.getFullYear();
    const m = String(lastWeek.getMonth() + 1).padStart(2, '0');
    const d = String(lastWeek.getDate()).padStart(2, '0');
    mocked.getReports.mockResolvedValue([
      {
        id: 9, team_name: '플랫폼팀', author_name: '홍길동', report_date: `${y}-${m}-${d}`,
        this_week: [], next_week: [{ title: '이월 업무', details: '', due_date: '2026-12-31', progress: 0 }],
        issues: '', notes: '', next_issues: '', next_notes: '', template_id: 0,
      },
    ]);
  });

  it('불러온 직후(이월 포함)는 변경 없음, 입력하면 변경 있음, 저장하면 다시 변경 없음', async () => {
    const onDirtyChange = vi.fn();
    render(<ReportForm onDirtyChange={onDirtyChange} />, { wrapper: AppProviders });

    await screen.findByDisplayValue('이월 업무');
    expect(lastDirty(onDirtyChange)).toBe(false);

    fireEvent.change(screen.getByPlaceholderText('홍길동'), { target: { value: '김철수' } });
    expect(lastDirty(onDirtyChange)).toBe(true);

    mocked.saveReport.mockImplementation(async (r) => ({ ...r, id: 10 }));
    fireEvent.click(screen.getByRole('button', { name: /저장/ }));
    await waitFor(() => expect(lastDirty(onDirtyChange)).toBe(false));
    expect(mocked.saveReport).toHaveBeenCalledTimes(1);
  });

  it('보고일자를 바꿔 다른 보고서로 교체될 때 변경 있음이 잠깐이라도 튀지 않는다', async () => {
    const onDirtyChange = vi.fn();
    const { container } = render(<ReportForm onDirtyChange={onDirtyChange} />, { wrapper: AppProviders });
    await screen.findByDisplayValue('이월 업무');

    fireEvent.change(container.querySelector('input[type="date"]')!, { target: { value: '2026-01-05' } });
    await waitFor(() => expect(screen.queryByDisplayValue('이월 업무')).not.toBeInTheDocument());
    expect(onDirtyChange.mock.calls).not.toContainEqual([true]);
  });

  it('작성 중에 보고일자를 바꾸면 확인하고, 취소하면 내용을 유지한다', async () => {
    const { container } = render(<ReportForm />, { wrapper: AppProviders });
    await screen.findByDisplayValue('이월 업무');
    fireEvent.change(screen.getByPlaceholderText('홍길동'), { target: { value: '김철수' } });
    const dateInput = container.querySelector('input[type="date"]') as HTMLInputElement;
    const before = dateInput.value;

    fireEvent.change(dateInput, { target: { value: '2026-01-05' } });
    const dialog = await screen.findByRole('alertdialog', { name: '저장하지 않은 내용이 있습니다' });
    fireEvent.click(within(dialog).getByRole('button', { name: '취소' }));
    await waitFor(() => expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument());
    expect(dateInput.value).toBe(before);
    expect(screen.getByDisplayValue('김철수')).toBeInTheDocument();

    fireEvent.change(dateInput, { target: { value: '2026-01-05' } });
    fireEvent.click(within(await screen.findByRole('alertdialog')).getByRole('button', { name: '바꾸기' }));
    await waitFor(() => expect(screen.queryByDisplayValue('이월 업무')).not.toBeInTheDocument());
    expect(dateInput.value).toBe('2026-01-05');
  });

  it('저장 요청 중에 더 입력한 내용은 저장 응답으로 덮이지 않고 변경 있음으로 남는다', async () => {
    const onDirtyChange = vi.fn();
    render(<ReportForm onDirtyChange={onDirtyChange} />, { wrapper: AppProviders });
    await screen.findByDisplayValue('이월 업무');
    const nameInput = screen.getByPlaceholderText('홍길동');
    fireEvent.change(nameInput, { target: { value: '김철수' } });

    let resolveSave!: (r: unknown) => void;
    mocked.saveReport.mockImplementation((r) => new Promise((resolve) => { resolveSave = () => resolve({ ...r, id: 10 }); }));
    fireEvent.click(screen.getByRole('button', { name: /저장/ }));
    fireEvent.change(nameInput, { target: { value: '김철수2' } });
    await act(async () => { resolveSave(undefined); });

    expect(nameInput).toHaveValue('김철수2');
    expect(lastDirty(onDirtyChange)).toBe(true);

    // id는 반영돼서 다음 저장은 같은 보고서 갱신
    mocked.saveReport.mockImplementation(async (r) => ({ ...r }));
    fireEvent.click(screen.getByRole('button', { name: /저장/ }));
    await waitFor(() => expect(lastDirty(onDirtyChange)).toBe(false));
    expect(mocked.saveReport.mock.calls[1][0]).toMatchObject({ id: 10, author_name: '김철수2' });
  });

  it('숨김 유지 중 선택했던 팀이 없어지면 선택을 해제해 제출 버튼을 숨긴다', async () => {
    const { rerender } = render(<ReportForm active />, { wrapper: AppProviders });
    await screen.findByDisplayValue('이월 업무');
    expect(screen.getByRole('button', { name: /^제출$/ })).toBeInTheDocument();

    rerender(<ReportForm active={false} />);
    mocked.getMyTeams.mockResolvedValue([{ id: 2, name: '다른팀' }]);
    rerender(<ReportForm active />);

    await waitFor(() => expect(screen.queryByRole('button', { name: /^제출$/ })).not.toBeInTheDocument());
  });

  it('숨김 유지 후 다시 보이면 설정·팀 목록을 새로 불러온다', async () => {
    const { rerender } = render(<ReportForm active />, { wrapper: AppProviders });
    await screen.findByDisplayValue('이월 업무');
    expect(mocked.getConfig).toHaveBeenCalledTimes(1);
    expect(mocked.getMyTeams).toHaveBeenCalledTimes(1);

    rerender(<ReportForm active={false} />);
    expect(mocked.getConfig).toHaveBeenCalledTimes(1);

    mocked.getMyTeams.mockResolvedValue([{ id: 1, name: '플랫폼팀' }, { id: 2, name: '새팀' }]);
    rerender(<ReportForm active />);
    expect(mocked.getConfig).toHaveBeenCalledTimes(2);
    expect(await screen.findByRole('option', { name: '새팀' })).toBeInTheDocument();
    // 작성 중인 내용은 그대로
    expect(screen.getByDisplayValue('이월 업무')).toBeInTheDocument();
  });

  it('완료예정일이 빈 항목이 있으면 제출하지 않고 해당 항목을 표시·포커스하며, 채우면 표시가 사라진다', async () => {
    mocked.getReports.mockResolvedValue([]);
    // jsdom에는 scrollIntoView가 없음
    const scrollIntoView = vi.fn();
    Element.prototype.scrollIntoView = scrollIntoView;
    const { container } = render(<ReportForm />, { wrapper: AppProviders });
    const submit = await screen.findByRole('button', { name: /제출/ });

    // 날짜 있는 항목 1개 + 빈 항목 1개
    fireEvent.click(screen.getAllByText('금주 업무를 추가해주세요')[0]);
    fireEvent.click(screen.getAllByRole('button', { name: /항목 추가/ })[0]);
    const dueInputs = () => screen.getAllByLabelText('완료일');
    fireEvent.change(dueInputs()[0], { target: { value: '2026-10-10' } });

    fireEvent.click(submit);
    expect(await screen.findByText(/완료예정일이 비어 있는 항목이 1건/)).toBeInTheDocument();
    expect(mocked.saveReport).not.toHaveBeenCalled();
    expect(dueInputs()[0]).not.toHaveAttribute('aria-invalid');
    expect(dueInputs()[1]).toHaveAttribute('aria-invalid', 'true');
    await waitFor(() => expect(dueInputs()[1]).toHaveFocus());
    expect(scrollIntoView.mock.contexts[0]).toBe(dueInputs()[1]);

    fireEvent.change(dueInputs()[1], { target: { value: '2026-10-11' } });
    expect(container.querySelector('[aria-invalid]')).toBeNull();

    // 다 채운 뒤 새로 추가한 빈 항목은 바로 빨갛게 표시하지 않음
    fireEvent.click(screen.getAllByRole('button', { name: /항목 추가/ })[0]);
    expect(container.querySelector('[aria-invalid]')).toBeNull();
  });

  it('누락 강조 중에 보고일자를 바꾸면 새 보고서에는 강조가 남지 않는다', async () => {
    Element.prototype.scrollIntoView = vi.fn();
    // 2026-01-05 주로 바꾸면 그 전주(2025-12-29) 차주계획이 날짜 없이 이월됨
    mocked.getReports.mockResolvedValue([{
      id: 5, team_name: '플랫폼팀', author_name: '홍길동', report_date: '2025-12-29',
      this_week: [], next_week: [{ title: '지난주 계획', details: '', due_date: '', progress: 0 }],
      issues: '', notes: '', next_issues: '', next_notes: '', template_id: 0,
    }]);
    const { container } = render(<ReportForm />, { wrapper: AppProviders });
    const submit = await screen.findByRole('button', { name: /^제출$/ });

    // 날짜 빈 항목으로 제출 → 강조 켜짐
    fireEvent.click(screen.getAllByText('금주 업무를 추가해주세요')[0]);
    fireEvent.click(submit);
    await waitFor(() => expect(container.querySelector('[aria-invalid]')).not.toBeNull());

    // 다른 날짜로 바꾸면 날짜 없는 이월 항목이 생기지만, 아직 제출을 시도하지 않았으니 강조하지 않아야 함
    fireEvent.change(container.querySelector('input[type="date"]')!, { target: { value: '2026-01-05' } });
    fireEvent.click(within(await screen.findByRole('alertdialog')).getByRole('button', { name: '바꾸기' }));
    expect(await screen.findByDisplayValue('지난주 계획')).toBeInTheDocument();
    expect(container.querySelector('[aria-invalid]')).toBeNull();
  });

  it('저장에 실패하면 변경 있음 상태를 유지한다', async () => {
    const onDirtyChange = vi.fn();
    render(<ReportForm onDirtyChange={onDirtyChange} />, { wrapper: AppProviders });
    await screen.findByDisplayValue('이월 업무');

    fireEvent.change(screen.getByPlaceholderText('홍길동'), { target: { value: '김철수' } });
    mocked.saveReport.mockRejectedValue(new Error('서버 오류'));
    fireEvent.click(screen.getByRole('button', { name: /저장/ }));

    await screen.findByText('서버 오류');
    expect(lastDirty(onDirtyChange)).toBe(true);
  });
  it('입력하면 잠시 뒤 이 브라우저에 초안을 보관하고, 저장하면 지우고 저장 시각을 보여준다', async () => {
    render(<ReportForm />, { wrapper: AppProviders });
    await screen.findByDisplayValue('이월 업무');
    expect(screen.queryByText('저장 안 됨')).not.toBeInTheDocument();

    fireEvent.change(screen.getByPlaceholderText('홍길동'), { target: { value: '김철수' } });
    expect(screen.getByText('저장 안 됨')).toBeInTheDocument();
    await waitFor(() => expect(loadDraft(1, toLocalYMD())?.report.author_name).toBe('김철수'), { timeout: 2000 });

    mocked.saveReport.mockImplementation(async (r) => ({ ...r, id: 10 }));
    fireEvent.click(screen.getByRole('button', { name: /저장/ }));
    expect(await screen.findByText('방금 저장됨')).toBeInTheDocument();
    expect(loadDraft(1, toLocalYMD())).toBeNull();
  });

  it('다시 열었을 때 불러온 내용과 다른 초안이 있으면 복원을 제안하고, 복원하면 변경 있음 상태가 된다', async () => {
    saveDraft(1, {
      team_name: '플랫폼팀', author_name: '홍길동', report_date: toLocalYMD(),
      this_week: [{ title: '초안 업무', details: '', due_date: '', progress: 30 }], next_week: [],
      issues: '', notes: '', next_issues: '', next_notes: '', template_id: 0,
    });
    const onDirtyChange = vi.fn();
    render(<ReportForm onDirtyChange={onDirtyChange} />, { wrapper: AppProviders });

    expect(await screen.findByText(/저장하지 않은 초안이 있습니다/)).toBeInTheDocument();
    expect(screen.getByDisplayValue('이월 업무')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: '복원' }));
    expect(screen.getByDisplayValue('초안 업무')).toBeInTheDocument();
    expect(screen.queryByText(/저장하지 않은 초안이 있습니다/)).not.toBeInTheDocument();
    expect(lastDirty(onDirtyChange)).toBe(true);
  });

  it('초안을 버리면 지우고, 불러온 내용과 같은 초안은 제안하지 않고 정리한다', async () => {
    saveDraft(1, {
      team_name: '플랫폼팀', author_name: '홍길동', report_date: toLocalYMD(),
      this_week: [{ title: '초안 업무', details: '', due_date: '', progress: 30 }], next_week: [],
      issues: '', notes: '', next_issues: '', next_notes: '', template_id: 0,
    });
    const first = render(<ReportForm />, { wrapper: AppProviders });
    await screen.findByText(/저장하지 않은 초안이 있습니다/);
    fireEvent.click(screen.getByRole('button', { name: '버리기' }));
    expect(loadDraft(1, toLocalYMD())).toBeNull();
    first.unmount();

    // 오늘 보고서가 서버에 이미 있고 초안이 그것과 같다면 → 제안 없이 정리
    const existing = {
      id: 11, team_name: '플랫폼팀', author_name: '홍길동', report_date: toLocalYMD(),
      this_week: [{ title: '저장된 업무', details: '', due_date: '2026-12-31', progress: 50 }], next_week: [],
      issues: '', notes: '', next_issues: '', next_notes: '', template_id: 0,
    };
    mocked.getReports.mockResolvedValue([existing]);
    saveDraft(1, existing);
    render(<ReportForm />, { wrapper: AppProviders });
    await screen.findByDisplayValue('저장된 업무');
    expect(screen.queryByText(/저장하지 않은 초안이 있습니다/)).not.toBeInTheDocument();
    expect(loadDraft(1, toLocalYMD())).toBeNull();
  });

  it('Ctrl+S로 저장하고, 숨겨진 상태에서는 가로채지 않는다', async () => {
    mocked.saveReport.mockImplementation(async (r) => ({ ...r, id: 10 }));
    const { rerender } = render(<ReportForm active />, { wrapper: AppProviders });
    await screen.findByDisplayValue('이월 업무');

    const event = new KeyboardEvent('keydown', { key: 's', code: 'KeyS', ctrlKey: true, cancelable: true });
    window.dispatchEvent(event);
    expect(event.defaultPrevented).toBe(true);
    await waitFor(() => expect(mocked.saveReport).toHaveBeenCalledTimes(1));

    rerender(<ReportForm active={false} />);
    const hidden = new KeyboardEvent('keydown', { key: 's', code: 'KeyS', ctrlKey: true, cancelable: true });
    window.dispatchEvent(hidden);
    expect(hidden.defaultPrevented).toBe(false);
    expect(mocked.saveReport).toHaveBeenCalledTimes(1);
  });
  it('배너를 둔 채 입력하다 버리기를 눌러도 새 입력의 초안은 지우지 않는다', async () => {
    saveDraft(1, {
      team_name: '플랫폼팀', author_name: '홍길동', report_date: toLocalYMD(),
      this_week: [{ title: '초안 업무', details: '', due_date: '', progress: 30 }], next_week: [],
      issues: '', notes: '', next_issues: '', next_notes: '', template_id: 0,
    });
    render(<ReportForm />, { wrapper: AppProviders });
    await screen.findByText(/저장하지 않은 초안이 있습니다/);

    fireEvent.change(screen.getByPlaceholderText('홍길동'), { target: { value: '새입력' } });
    await waitFor(() => expect(loadDraft(1, toLocalYMD())?.report.author_name).toBe('새입력'), { timeout: 2000 });

    fireEvent.click(screen.getByRole('button', { name: '버리기' }));
    expect(loadDraft(1, toLocalYMD())?.report.author_name).toBe('새입력');
  });

  it('초안을 쓴 뒤 서버 보고서가 바뀌었으면 복원 시 덮어쓴다고 경고한다', async () => {
    const base = {
      id: 11, team_name: '플랫폼팀', author_name: '홍길동', report_date: toLocalYMD(),
      this_week: [{ title: '처음 내용', details: '', due_date: '2026-12-31', progress: 50 }], next_week: [],
      issues: '', notes: '', next_issues: '', next_notes: '', template_id: 0,
    };
    saveDraft(1, { ...base, notes: '이 PC에서 쓰던 초안' }, JSON.stringify(base));
    // 다른 PC에서 저장해 서버 내용이 바뀐 상태
    mocked.getReports.mockResolvedValue([{ ...base, this_week: [{ ...base.this_week[0], title: '다른 PC 내용' }] }]);
    render(<ReportForm />, { wrapper: AppProviders });

    expect(await screen.findByText(/다른 곳에서 이 보고서가 저장됐습니다/)).toBeInTheDocument();
  });

  it('작성 중 보고일자를 바꾸기로 하면 이전 날짜의 임시 초안도 지운다', async () => {
    const { container } = render(<ReportForm />, { wrapper: AppProviders });
    await screen.findByDisplayValue('이월 업무');
    fireEvent.change(screen.getByPlaceholderText('홍길동'), { target: { value: '김철수' } });
    await waitFor(() => expect(loadDraft(1, toLocalYMD())).not.toBeNull(), { timeout: 2000 });

    const dateInput = container.querySelector('input[type="date"]') as HTMLInputElement;
    fireEvent.change(dateInput, { target: { value: '2026-01-05' } });
    fireEvent.blur(dateInput);
    fireEvent.click(within(await screen.findByRole('alertdialog')).getByRole('button', { name: '바꾸기' }));
    await waitFor(() => expect(dateInput.value).toBe('2026-01-05'));
    expect(loadDraft(1, toLocalYMD())).toBeNull();
  });
  it('저장 응답을 기다리는 동안 다른 날짜로 바꾸면, 늦게 온 응답이 새 날짜 보고서를 건드리지 않는다', async () => {
    const onDirtyChange = vi.fn();
    const { container } = render(<ReportForm onDirtyChange={onDirtyChange} />, { wrapper: AppProviders });
    await screen.findByDisplayValue('이월 업무');
    fireEvent.change(screen.getByPlaceholderText('홍길동'), { target: { value: '김철수' } });

    let resolveSave!: () => void;
    mocked.saveReport.mockImplementation((r) => new Promise((resolve) => { resolveSave = () => resolve({ ...r, id: 10 }); }));
    fireEvent.click(screen.getByRole('button', { name: /저장/ }));

    const dateInput = container.querySelector('input[type="date"]') as HTMLInputElement;
    fireEvent.change(dateInput, { target: { value: '2026-01-05' } });
    fireEvent.blur(dateInput);
    fireEvent.click(within(await screen.findByRole('alertdialog')).getByRole('button', { name: '바꾸기' }));
    await waitFor(() => expect(dateInput.value).toBe('2026-01-05'));
    expect(lastDirty(onDirtyChange)).toBe(false);

    await act(async () => { resolveSave(); });
    expect(dateInput.value).toBe('2026-01-05');
    expect(lastDirty(onDirtyChange)).toBe(false);
    // 다음 저장은 새 날짜 보고서로 (이전 보고서 id가 섞이지 않음)
    mocked.saveReport.mockImplementation(async (r) => ({ ...r, id: 20 }));
    fireEvent.click(screen.getAllByText('금주 업무를 추가해주세요')[0]);
    fireEvent.click(screen.getByRole('button', { name: /저장/ }));
    await waitFor(() => expect(mocked.saveReport).toHaveBeenCalledTimes(2));
    expect(mocked.saveReport.mock.calls[1][0]).toMatchObject({ report_date: '2026-01-05', id: undefined });
  });

  it('1초를 기다리지 못하고 사라져도(언마운트) 마지막 입력을 초안으로 남긴다', async () => {
    const { unmount } = render(<ReportForm />, { wrapper: AppProviders });
    await screen.findByDisplayValue('이월 업무');
    fireEvent.change(screen.getByPlaceholderText('홍길동'), { target: { value: '마지막입력' } });
    unmount();
    expect(loadDraft(1, toLocalYMD())?.report.author_name).toBe('마지막입력');
  });

  it('편집했다가 원래대로 되돌리면 중간 초안을 지운다', async () => {
    render(<ReportForm />, { wrapper: AppProviders });
    await screen.findByDisplayValue('이월 업무');
    const nameInput = screen.getByPlaceholderText('홍길동');
    const original = (nameInput as HTMLInputElement).value;

    fireEvent.change(nameInput, { target: { value: '김철수' } });
    await waitFor(() => expect(loadDraft(1, toLocalYMD())).not.toBeNull(), { timeout: 2000 });
    fireEvent.change(nameInput, { target: { value: original } });
    await waitFor(() => expect(loadDraft(1, toLocalYMD())).toBeNull());
  });

  it('첫 로드가 끝나기 전에는 초안을 남기지 않는다', async () => {
    mocked.getReports.mockImplementation(() => new Promise(() => {}));
    render(<ReportForm />, { wrapper: AppProviders });
    fireEvent.change(screen.getByPlaceholderText('홍길동'), { target: { value: '로드전입력' } });
    await new Promise((r) => setTimeout(r, 1200));
    expect(loadDraft(1, toLocalYMD())).toBeNull();
  });
});
