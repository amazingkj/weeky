import { useState, useEffect, useCallback, useMemo, useRef } from 'react';
import { Team, SiteProject, SiteTask, SiteNextTask } from '../types';
import { getMyTeams, getMySiteProjects, getMySiteReport, saveSiteReport } from '../services/api';
import Loading from './ui/Loading';
import Alert from './ui/Alert';
import { useConfirm } from './ui/ConfirmDialog';
import DateCommitInput from './ui/DateCommitInput';

const inputCls = 'w-full px-2 py-1 text-sm border border-neutral-200 rounded focus:outline-none focus:border-neutral-400';

// 기본값: 가장 최근에 지나간 금요일 (오늘이 금요일이면 일주일 전 금요일).
// 사이트 보고서는 통상 한 주(월~금)가 끝난 뒤 작성하므로 "지난 주 금요일"이 기본 보고 기준일이 됨.
const getDefaultDate = (): string => {
  const d = new Date();
  const day = d.getDay(); // 0=일 ~ 6=토
  // 오늘 기준 가장 가까운 "지난" 금요일까지의 일수
  // Sun=0→2, Mon=1→3, Tue=2→4, Wed=3→5, Thu=4→6, Fri=5→7(일주일전), Sat=6→1
  const offset = day === 5 ? 7 : day === 6 ? 1 : day + 2;
  d.setDate(d.getDate() - offset);
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
};

interface TeamWithSiteProjects {
  team: Team;
  projects: SiteProject[];
}

interface SiteReportFormState {
  reportDate: string;
  reportDateText: string;
  thisWeek: SiteTask[];
  nextWeek: SiteNextTask[];
  notes: string;
}

const emptyState = (): SiteReportFormState => ({
  reportDate: getDefaultDate(),
  reportDateText: '',
  thisWeek: [],
  nextWeek: [],
  notes: '',
});

// 저장 안 된 변경 비교용 — 대상 주(reportDate)는 선택값이라 제외
const contentKey = (s: SiteReportFormState) =>
  JSON.stringify([s.reportDateText, s.thisWeek, s.nextWeek, s.notes]);

// 모든 팀에서 내가 작성자로 등록된 사이트 프로젝트 수집
async function fetchMySiteTeamProjects(): Promise<TeamWithSiteProjects[]> {
  const teams = await getMyTeams();
  const results = await Promise.all(
    teams.map(async (t) => {
      try {
        const ps = await getMySiteProjects(t.id);
        return { team: t, projects: ps };
      } catch {
        return { team: t, projects: [] };
      }
    })
  );
  return results.filter((r) => r.projects.length > 0);
}

interface SiteReportFormProps {
  // 저장 안 된 변경 여부 — App이 탭 전환 시 폼을 유지할지 판단
  onDirtyChange?: (dirty: boolean) => void;
  // 현재 보이는 탭인지 (숨김 유지 중이면 false)
  active?: boolean;
}

export default function SiteReportForm({ onDirtyChange, active = true }: SiteReportFormProps) {
  const [loading, setLoading] = useState(true);
  const [teamProjects, setTeamProjects] = useState<TeamWithSiteProjects[]>([]);
  const [selectedTeamId, setSelectedTeamId] = useState<number | null>(null);
  const [selectedProjectId, setSelectedProjectId] = useState<number | null>(null);
  const [state, setState] = useState<SiteReportFormState>(emptyState);
  const [reportLoading, setReportLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  // 보고서 조회 실패 — 화면에 이전 보고서 내용이 남아 있어 저장하면 다른 보고서를 덮어쓰므로 저장 차단
  const [loadFailed, setLoadFailed] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const successTimerRef = useRef<ReturnType<typeof setTimeout>>();
  // 서버에서 불러오거나 저장한 내용 — 이것과 다르면 저장 안 된 변경
  const [baseline, setBaseline] = useState<string>(() => contentKey(emptyState()));
  const stateKey = useMemo(() => contentKey(state), [state]);
  const isDirty = stateKey !== baseline;
  const stateRef = useRef(state);
  stateRef.current = state;
  const confirm = useConfirm();
  // 팀·프로젝트·대상 주를 바꾸면 해당 보고서로 교체되므로 작성 중인 내용이 있으면 확인
  const confirmDiscard = async () => !isDirty || confirm({
    title: '저장하지 않은 내용이 있습니다',
    message: '다른 보고서로 바꾸면 작성 중인 내용이 사라집니다.',
    confirmLabel: '바꾸기',
    danger: true,
  });

  useEffect(() => {
    onDirtyChange?.(isDirty);
  }, [isDirty, onDirtyChange]);
  useEffect(() => () => onDirtyChange?.(false), [onDirtyChange]);

  useEffect(() => {
    return () => {
      if (successTimerRef.current) clearTimeout(successTimerRef.current);
    };
  }, []);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const filtered = await fetchMySiteTeamProjects();
        if (cancelled) return;
        setTeamProjects(filtered);
        if (filtered.length > 0) {
          setSelectedTeamId(filtered[0].team.id);
          setSelectedProjectId(filtered[0].projects[0].id);
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => { cancelled = true; };
  }, []);

  const selected = useMemo(() => {
    const tp = teamProjects.find((x) => x.team.id === selectedTeamId);
    if (!tp) return null;
    const project = tp.projects.find((p) => p.id === selectedProjectId);
    return project ? { team: tp.team, project } : null;
  }, [teamProjects, selectedTeamId, selectedProjectId]);

  // 숨김 유지 중 팀 탭에서 사이트 프로젝트 배정이 바뀌었을 수 있어 다시 보이면 목록만 갱신.
  // 선택은 유지 — 바꾸면 작성 중인 내용이 다른 프로젝트 보고서로 교체됨
  const wasActiveRef = useRef(active);
  useEffect(() => {
    if (active && !wasActiveRef.current) {
      fetchMySiteTeamProjects().then((list) => {
        setTeamProjects(list);
        const stillAssigned = list.some((tp) => tp.team.id === selectedTeamId && tp.projects.some((p) => p.id === selectedProjectId));
        if (!stillAssigned) {
          // 저장할 곳이 없어졌으므로 변경 상태를 풀어 계속 붙잡아 두지 않음 (알림 후 내용은 화면에 남김)
          setError('선택했던 사이트 프로젝트의 작성 권한이 없어져 저장할 수 없습니다. 필요한 내용은 따로 복사해두세요.');
          setBaseline(contentKey(stateRef.current));
        }
      }).catch(() => {});
    }
    wasActiveRef.current = active;
  }, [active, selectedTeamId, selectedProjectId]);

  // 지금 화면에 띄운 (project, date) — 저장 완료 시 그사이 다른 보고서로 바뀌었는지 확인용
  const loadKey = `${selected?.project.id}|${state.reportDate}`;
  const loadKeyRef = useRef(loadKey);
  loadKeyRef.current = loadKey;

  // 선택된 (team, project, date)에 대한 기존 보고서 로드
  useEffect(() => {
    if (!selected) return;
    // 빠르게 바꾸면 늦게 도착한 이전 응답이 현재 내용·기준선을 덮지 않도록
    let cancelled = false;
    setReportLoading(true);
    getMySiteReport(selected.team.id, selected.project.id, state.reportDate)
      .then((res) => {
        if (cancelled) return;
        const r = res.exists ? res.report : undefined;
        const loaded = r
          ? { reportDateText: r.report_date_text || '', thisWeek: r.this_week, nextWeek: r.next_week, notes: r.notes }
          : { reportDateText: '', thisWeek: [], nextWeek: [], notes: '' };
        setState((s) => ({ ...s, ...loaded }));
        setBaseline(contentKey({ ...emptyState(), ...loaded }));
        setLoadFailed(false);
        // 이전 조회 실패 안내가 남아 저장이 막힌 것처럼 보이지 않도록
        setError(null);
      })
      .catch(() => {
        if (cancelled) return;
        setLoadFailed(true);
        setError('보고서를 불러오지 못했습니다. 다른 보고서를 덮어쓰지 않도록 저장을 막았습니다. 다른 주·프로젝트를 고르거나 새로고침해주세요.');
        // 화면 내용은 선택한 보고서의 것이 아니므로 변경 상태로 붙잡아 두지 않음 (교체 전에 이미 확인받음)
        setBaseline(contentKey(stateRef.current));
      })
      .finally(() => { if (!cancelled) setReportLoading(false); });
    return () => { cancelled = true; };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selected?.project.id, state.reportDate]);

  const updateThisRow = (idx: number, patch: Partial<SiteTask>) => {
    setState((s) => ({ ...s, thisWeek: s.thisWeek.map((r, i) => i === idx ? { ...r, ...patch } : r) }));
  };
  const updateNextRow = (idx: number, patch: Partial<SiteNextTask>) => {
    setState((s) => ({ ...s, nextWeek: s.nextWeek.map((r, i) => i === idx ? { ...r, ...patch } : r) }));
  };
  const addThisRow = () => setState((s) => ({ ...s, thisWeek: [...s.thisWeek, { title: '', elapsed_days: '', start_date: '', due_date: '', progress: '' }] }));
  const addNextRow = () => setState((s) => ({ ...s, nextWeek: [...s.nextWeek, { title: '', start_date: '', due_date: '' }] }));
  const removeThisRow = (idx: number) => setState((s) => ({ ...s, thisWeek: s.thisWeek.filter((_, i) => i !== idx) }));
  const removeNextRow = (idx: number) => setState((s) => ({ ...s, nextWeek: s.nextWeek.filter((_, i) => i !== idx) }));

  const handleSave = useCallback(async () => {
    if (!selected || loadFailed) return;
    const savedKey = loadKeyRef.current;
    setSaving(true);
    setError(null);
    setSuccess(null);
    try {
      await saveSiteReport(selected.team.id, {
        site_project_id: selected.project.id,
        report_date: state.reportDate,
        report_date_text: state.reportDateText,
        this_week: state.thisWeek,
        next_week: state.nextWeek,
        notes: state.notes,
      });
      // 저장 요청에 보낸 내용 기준 — 저장 중에 더 입력한 내용은 계속 변경으로 남음.
      // 그사이 다른 프로젝트/주로 바뀌었으면 그쪽 기준선은 건드리지 않음
      if (loadKeyRef.current === savedKey) setBaseline(contentKey(state));
      setSuccess('저장되었습니다.');
      if (successTimerRef.current) clearTimeout(successTimerRef.current);
      successTimerRef.current = setTimeout(() => setSuccess(null), 2000);
    } catch (e: any) {
      setError(e?.message || '저장에 실패했습니다.');
    } finally {
      setSaving(false);
    }
  }, [selected, state, loadFailed]);

  if (loading) {
    return <div className="py-16"><Loading text="사이트 프로젝트 로딩 중..." size="lg" /></div>;
  }

  // 작성 중이던 내용이 있으면(숨김 중 배정이 모두 해제된 경우) 안내 화면 대신 내용과 오류를 보여줌
  if (teamProjects.length === 0 && stateKey === contentKey(emptyState())) {
    return (
      <div className="bg-white p-8 rounded-xl border border-neutral-200 text-center">
        <p className="text-sm text-neutral-500">
          작성자로 등록된 사이트 프로젝트가 없습니다.
        </p>
        <p className="text-xs text-neutral-400 mt-2">
          팀장에게 문의하여 사이트 프로젝트의 작성자로 등록해 달라고 요청하세요.
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-5">
      {error && <Alert type="error" onClose={() => setError(null)}>{error}</Alert>}
      {success && <Alert type="success" onClose={() => setSuccess(null)}>{success}</Alert>}

      {/* Team + project selector + date */}
      <div className="bg-white rounded-xl border border-neutral-200 p-4 space-y-3">
        <div className="flex flex-wrap gap-2 items-end">
          {teamProjects.length > 1 && (
            <div>
              <label className="block text-xs font-medium text-neutral-500 mb-1">팀</label>
              <select
                value={selectedTeamId ?? ''}
                onChange={async (e) => {
                  const tid = Number(e.target.value);
                  if (!(await confirmDiscard())) return;
                  setSelectedTeamId(tid);
                  const first = teamProjects.find((x) => x.team.id === tid)?.projects[0];
                  setSelectedProjectId(first?.id ?? null);
                }}
                className="px-2 py-1.5 text-sm border border-neutral-200 rounded focus:outline-none focus:border-neutral-400"
              >
                {teamProjects.map(({ team }) => (
                  <option key={team.id} value={team.id}>{team.name}</option>
                ))}
              </select>
            </div>
          )}
          <div>
            <label className="block text-xs font-medium text-neutral-500 mb-1">프로젝트</label>
            <select
              value={selectedProjectId ?? ''}
              onChange={async (e) => {
                const pid = Number(e.target.value);
                if (await confirmDiscard()) setSelectedProjectId(pid);
              }}
              className="px-2 py-1.5 text-sm border border-neutral-200 rounded focus:outline-none focus:border-neutral-400"
            >
              {(teamProjects.find((x) => x.team.id === selectedTeamId)?.projects || []).map((p) => (
                <option key={p.id} value={p.id}>{p.project_name}</option>
              ))}
            </select>
          </div>
          <div>
            <label className="block text-xs font-medium text-neutral-500 mb-1">대상 주(이 날짜가 속한 주)</label>
            <DateCommitInput
              value={state.reportDate}
              onCommit={async (reportDate) => {
                if (!(await confirmDiscard())) return false;
                setState((s) => ({ ...s, reportDate }));
                return true;
              }}
              className="px-2 py-1.5 text-sm border border-neutral-200 rounded focus:outline-none focus:border-neutral-400"
            />
          </div>
          <div className="flex-1 min-w-[160px]">
            <label className="block text-xs font-medium text-neutral-500 mb-1">보고일자(헤더 표시용, 자유 텍스트)</label>
            <input
              type="text"
              placeholder="예: 2026-04-23"
              value={state.reportDateText}
              onChange={(e) => setState((s) => ({ ...s, reportDateText: e.target.value }))}
              className="w-full px-2 py-1.5 text-sm border border-neutral-200 rounded focus:outline-none focus:border-neutral-400"
            />
          </div>
        </div>
        {selected && selected.project.authors && selected.project.authors.length > 0 && (
          <div className="text-xs text-neutral-500">
            <span className="font-medium text-neutral-600">작성자:</span> {selected.project.authors.map((a) => a.user_name).join(', ')}
          </div>
        )}
      </div>

      {reportLoading && <Loading text="보고서 로딩 중..." />}

      {/* 금주실적 */}
      <div className="bg-white rounded-xl border border-neutral-200 p-4 space-y-2">
        <div className="flex items-center justify-between">
          <h3 className="text-sm font-semibold text-neutral-900">금주실적 <span className="text-xs text-neutral-400 font-normal">(5컬럼)</span></h3>
          <button onClick={addThisRow} className="btn-subtle btn-sm">+ 행 추가</button>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full min-w-[640px] text-xs border border-neutral-200">
            <thead className="bg-neutral-50">
              <tr>
                <th className="border border-neutral-200 px-2 py-1 text-left w-[45%]">계획업무</th>
                <th className="border border-neutral-200 px-2 py-1 text-left w-[10%]">소요일</th>
                <th className="border border-neutral-200 px-2 py-1 text-left w-[12%]">시작일</th>
                <th className="border border-neutral-200 px-2 py-1 text-left w-[12%]">완료일</th>
                <th className="border border-neutral-200 px-2 py-1 text-left w-[10%]">실적</th>
                <th className="border border-neutral-200 px-2 py-1 w-[6%]"></th>
              </tr>
            </thead>
            <tbody>
              {state.thisWeek.length === 0 ? (
                <tr><td colSpan={6} className="text-center text-neutral-400 py-3">행이 없습니다. "+ 행 추가"를 눌러 시작하세요.</td></tr>
              ) : state.thisWeek.map((row, idx) => (
                <tr key={idx} className="align-top">
                  <td className="border border-neutral-200 p-1">
                    <textarea
                      value={row.title}
                      onChange={(e) => updateThisRow(idx, { title: e.target.value })}
                      rows={3}
                      placeholder="■ 한화손해보험&#10;<OpenAPI>&#10;1. DB 이관 배치 작업&#10;  - 배치 실행 환경에 맞게 재개발"
                      className={inputCls + ' font-mono'}
                    />
                  </td>
                  <td className="border border-neutral-200 p-1"><input value={row.elapsed_days} onChange={(e) => updateThisRow(idx, { elapsed_days: e.target.value })} className={inputCls} placeholder="2M / 1 / -" /></td>
                  <td className="border border-neutral-200 p-1"><input value={row.start_date} onChange={(e) => updateThisRow(idx, { start_date: e.target.value })} className={inputCls} placeholder="03/04" /></td>
                  <td className="border border-neutral-200 p-1"><input value={row.due_date} onChange={(e) => updateThisRow(idx, { due_date: e.target.value })} className={inputCls} placeholder="04/30" /></td>
                  <td className="border border-neutral-200 p-1"><input value={row.progress} onChange={(e) => updateThisRow(idx, { progress: e.target.value })} className={inputCls} placeholder="80%" /></td>
                  <td className="border border-neutral-200 p-1 text-center">
                    <button onClick={() => removeThisRow(idx)} className="text-neutral-400 hover:text-danger-500" title="행 삭제" aria-label="행 삭제">×</button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* 차주계획 */}
      <div className="bg-white rounded-xl border border-neutral-200 p-4 space-y-2">
        <div className="flex items-center justify-between">
          <h3 className="text-sm font-semibold text-neutral-900">차주계획 <span className="text-xs text-neutral-400 font-normal">(3컬럼)</span></h3>
          <button onClick={addNextRow} className="btn-subtle btn-sm">+ 행 추가</button>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full min-w-[640px] text-xs border border-neutral-200">
            <thead className="bg-neutral-50">
              <tr>
                <th className="border border-neutral-200 px-2 py-1 text-left w-[60%]">계획업무</th>
                <th className="border border-neutral-200 px-2 py-1 text-left w-[15%]">시작예정일</th>
                <th className="border border-neutral-200 px-2 py-1 text-left w-[15%]">완료예정일</th>
                <th className="border border-neutral-200 px-2 py-1 w-[10%]"></th>
              </tr>
            </thead>
            <tbody>
              {state.nextWeek.length === 0 ? (
                <tr><td colSpan={4} className="text-center text-neutral-400 py-3">행이 없습니다.</td></tr>
              ) : state.nextWeek.map((row, idx) => (
                <tr key={idx} className="align-top">
                  <td className="border border-neutral-200 p-1">
                    <textarea
                      value={row.title}
                      onChange={(e) => updateNextRow(idx, { title: e.target.value })}
                      rows={2}
                      className={inputCls + ' font-mono'}
                    />
                  </td>
                  <td className="border border-neutral-200 p-1"><input value={row.start_date} onChange={(e) => updateNextRow(idx, { start_date: e.target.value })} className={inputCls} placeholder="05/04" /></td>
                  <td className="border border-neutral-200 p-1"><input value={row.due_date} onChange={(e) => updateNextRow(idx, { due_date: e.target.value })} className={inputCls} placeholder="05/08" /></td>
                  <td className="border border-neutral-200 p-1 text-center">
                    <button onClick={() => removeNextRow(idx)} className="text-neutral-400 hover:text-danger-500" title="행 삭제" aria-label="행 삭제">×</button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* 특이사항 */}
      <div className="bg-white rounded-xl border border-neutral-200 p-4 space-y-2">
        <h3 className="text-sm font-semibold text-neutral-900">특이사항</h3>
        <textarea
          value={state.notes}
          onChange={(e) => setState((s) => ({ ...s, notes: e.target.value }))}
          rows={4}
          placeholder="- 배치 실행 환경을 운영서버가 아닌 이관서버에서 진행한다 하여..."
          className="w-full px-2 py-1.5 text-sm border border-neutral-200 rounded focus:outline-none focus:border-neutral-400 font-mono"
        />
      </div>

      <div className="flex flex-wrap items-center justify-end gap-3">
        <button
          onClick={handleSave}
          disabled={saving || !selected || reportLoading || loadFailed}
          className="btn-primary"
        >
          {saving ? '저장 중...' : '저장'}
        </button>
      </div>
    </div>
  );
}
