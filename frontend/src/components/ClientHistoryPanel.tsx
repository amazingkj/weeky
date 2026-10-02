import { useState, useEffect } from 'react';
import { ClientHistoryEntry, ClientHistoryResponse } from '../types';
import { getClientHistory } from '../services/api';
import { getMonday, getWeekOfMonthLabel, getWeekRangeShort } from '../utils/date';
import Loading from './ui/Loading';

interface ClientHistoryPanelProps {
  teamId: number;
}

interface ClientSummary {
  client: string;
  count: number;
  lastDate: string;
  authors: Set<string>;
}

// 팀장/그룹장용 — 팀원이 제출한 원본 보고서 + 사이트 보고서를 고객사별 주차 타임라인으로 표시
export default function ClientHistoryPanel({ teamId }: ClientHistoryPanelProps) {
  const [weeks, setWeeks] = useState(12);
  const [data, setData] = useState<ClientHistoryResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selected, setSelected] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError(null);
    getClientHistory(teamId, weeks)
      .then((res) => { if (!cancelled) setData(res); })
      .catch((err) => { if (!cancelled) { setData(null); setError(err.message); } })
      .finally(() => { if (!cancelled) setLoading(false); });
    return () => { cancelled = true; };
  }, [teamId, weeks]);

  // 고객사별 요약 — 최근 활동순
  const summaries: ClientSummary[] = [];
  const byClient = new Map<string, ClientSummary>();
  for (const e of data?.entries ?? []) {
    let s = byClient.get(e.client);
    if (!s) {
      s = { client: e.client, count: 0, lastDate: e.report_date, authors: new Set() };
      byClient.set(e.client, s);
      summaries.push(s);
    }
    s.count++;
    if (e.report_date > s.lastDate) s.lastDate = e.report_date;
    for (const a of e.authors) s.authors.add(a);
  }
  summaries.sort((a, b) => b.lastDate.localeCompare(a.lastDate) || a.client.localeCompare(b.client, 'ko'));

  // 선택한 고객사가 목록에 없으면(기간 변경 등) 가장 최근 고객사로
  const activeClient = selected && byClient.has(selected) ? selected : summaries[0]?.client ?? null;

  // 선택 고객사의 항목을 주차(월요일)별로 묶음 — entries는 서버에서 보고일 내림차순
  const weekGroups: { monday: string; entries: ClientHistoryEntry[] }[] = [];
  for (const e of data?.entries ?? []) {
    if (e.client !== activeClient) continue;
    const monday = getMonday(e.report_date);
    const last = weekGroups[weekGroups.length - 1];
    if (last && last.monday === monday) last.entries.push(e);
    else weekGroups.push({ monday, entries: [e] });
  }

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <h4 className="text-sm font-semibold text-neutral-900">고객사 히스토리</h4>
        <select
          value={weeks}
          onChange={(e) => setWeeks(Number(e.target.value))}
          aria-label="조회 기간"
          className="text-xs border border-neutral-200 rounded-lg px-2 py-1 focus:outline-none focus:border-neutral-400"
        >
          <option value={8}>최근 8주</option>
          <option value={12}>최근 12주</option>
          <option value={24}>최근 24주</option>
        </select>
      </div>
      <p className="text-[11px] text-neutral-400">
        팀원이 제출한 보고서(미제출 제외)와 사이트 보고서 중 고객사가 지정된 업무만 표시됩니다.
      </p>

      {loading ? (
        <Loading text="고객사 히스토리 로딩 중..." />
      ) : error ? (
        <p className="text-xs text-red-500 py-6 text-center">{error}</p>
      ) : summaries.length === 0 ? (
        <p className="text-xs text-neutral-400 py-6 text-center">기간 내 고객사가 지정된 업무가 없습니다.</p>
      ) : (
        <>
          <div className="flex flex-wrap gap-1.5">
            {summaries.map((s) => (
              <button
                key={s.client}
                onClick={() => setSelected(s.client)}
                className={`px-2.5 py-1.5 text-xs font-medium rounded-lg border transition-colors ${
                  activeClient === s.client
                    ? 'bg-ink-800 text-white border-ink-800'
                    : 'bg-white text-neutral-600 border-neutral-200 hover:border-neutral-300'
                }`}
              >
                {s.client}
                <span className={`ml-1.5 ${activeClient === s.client ? 'text-white/70' : 'text-neutral-400'}`}>
                  {s.count}건 · {s.authors.size}명
                </span>
              </button>
            ))}
          </div>

          <div className="space-y-3">
            {weekGroups.map((g) => (
              <div key={g.monday} className="border border-neutral-200 rounded-lg overflow-hidden">
                <div className="px-3 py-2 bg-neutral-50 border-b border-neutral-200 text-xs font-medium text-neutral-700">
                  {getWeekOfMonthLabel(g.monday)}
                  <span className="ml-2 text-neutral-400 font-normal">{getWeekRangeShort(g.monday)}</span>
                </div>
                <WeekSection title="금주실적" entries={g.entries.filter((e) => e.section === 'this_week')} showProgress />
                <WeekSection title="차주계획" entries={g.entries.filter((e) => e.section === 'next_week')} />
              </div>
            ))}
          </div>
        </>
      )}
    </div>
  );
}

function WeekSection({ title, entries, showProgress = false }: { title: string; entries: ClientHistoryEntry[]; showProgress?: boolean }) {
  if (entries.length === 0) return null;
  return (
    <div className="px-3 py-2 space-y-1.5">
      <div className="text-[11px] font-semibold text-neutral-500">{title}</div>
      {entries.map((e, i) => (
        <div key={i} className="flex items-start justify-between gap-3 text-xs">
          <div className="min-w-0">
            <span
              className={`mr-1.5 px-1.5 py-0.5 rounded text-[10px] font-medium ${
                e.kind === 'site' ? 'bg-purple-100 text-purple-700' : 'bg-sky-100 text-sky-700'
              }`}
            >
              {e.kind === 'site' ? '사이트' : '본사'}
            </span>
            <span className="font-medium text-neutral-900">{e.project}</span>
            {e.work && <span className="ml-1.5 text-neutral-700 whitespace-pre-line">{e.work}</span>}
            <div className="mt-0.5 text-[11px] text-neutral-400">
              {e.authors.join(', ')}
              {e.due_date && <span className="ml-2">완료예정 {e.due_date}</span>}
            </div>
          </div>
          {showProgress && e.progress && (
            <span className="shrink-0 text-neutral-500 font-medium">{e.progress}%</span>
          )}
        </div>
      ))}
    </div>
  );
}
