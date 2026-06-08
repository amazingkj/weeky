import { useState, useEffect } from 'react';
import { ReportSubmission, SiteReport, Report } from '../types';
import { getMySubmissions, getMySiteReports, getReport } from '../services/api';
import Loading from './ui/Loading';

interface MyHistoryPanelProps {
  teamId: number;
}

type Filter = 'all' | 'report' | 'site';

// 본사 제출 이력 + 사이트 보고서를 하나의 타임라인으로 합친 항목
type HistoryItem =
  | { kind: 'report'; key: string; date: string; submittedAt?: string; status: string; reportId: number }
  | { kind: 'site'; key: string; date: string; dateText?: string; projectName: string; report: SiteReport };

function itemDate(it: HistoryItem): string {
  return it.kind === 'site' ? it.report.report_date : it.date;
}

export default function MyHistoryPanel({ teamId }: MyHistoryPanelProps) {
  const [loading, setLoading] = useState(true);
  const [submissions, setSubmissions] = useState<ReportSubmission[]>([]);
  const [siteReports, setSiteReports] = useState<SiteReport[]>([]);
  const [filter, setFilter] = useState<Filter>('all');

  // 펼친 항목 key + 본사 보고서 본문 캐시(report_id 기준)
  const [expanded, setExpanded] = useState<string | null>(null);
  const [reportCache, setReportCache] = useState<Record<number, Report | null>>({});
  // 현재 fetch 중인 report_id (전역 1개 플래그 대신 항목별로 추적해 동시 전환 시 깜빡임 방지)
  const [loadingReportId, setLoadingReportId] = useState<number | null>(null);

  useEffect(() => {
    setLoading(true);
    setExpanded(null);
    Promise.all([
      getMySubmissions(teamId).catch(() => [] as ReportSubmission[]),
      getMySiteReports(teamId).catch(() => [] as SiteReport[]),
    ])
      .then(([subs, sites]) => {
        setSubmissions(subs);
        setSiteReports(sites);
      })
      .finally(() => setLoading(false));
  }, [teamId]);

  const items: HistoryItem[] = [
    ...submissions.map<HistoryItem>((s) => ({
      kind: 'report',
      key: `r-${s.id}`,
      date: s.report_date || '',
      submittedAt: s.submitted_at,
      status: s.status,
      reportId: s.report_id,
    })),
    ...siteReports.map<HistoryItem>((sr) => ({
      kind: 'site',
      key: `s-${sr.id}`,
      date: sr.report_date,
      dateText: sr.report_date_text,
      projectName: sr.project_name,
      report: sr,
    })),
  ]
    .filter((it) => filter === 'all' || it.kind === filter)
    .sort((a, b) => itemDate(b).localeCompare(itemDate(a)));

  const toggle = async (it: HistoryItem) => {
    if (expanded === it.key) {
      setExpanded(null);
      return;
    }
    setExpanded(it.key);
    if (it.kind === 'report' && reportCache[it.reportId] === undefined) {
      const reportId = it.reportId;
      setLoadingReportId(reportId);
      try {
        const report = await getReport(reportId);
        setReportCache((prev) => ({ ...prev, [reportId]: report || null }));
      } catch {
        setReportCache((prev) => ({ ...prev, [reportId]: null }));
      } finally {
        // 그 사이 다른 항목으로 전환됐다면 그 항목의 로딩 상태는 유지
        setLoadingReportId((prev) => (prev === reportId ? null : prev));
      }
    }
  };

  if (loading) return <Loading text="내 히스토리 로딩 중..." />;

  const counts = {
    all: submissions.length + siteReports.length,
    report: submissions.length,
    site: siteReports.length,
  };

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <h4 className="text-sm font-semibold text-neutral-900">내 보고서 히스토리</h4>
        <div className="flex gap-1">
          {([
            ['all', '전체'],
            ['report', '본사'],
            ['site', '사이트'],
          ] as [Filter, string][]).map(([key, label]) => (
            <button
              key={key}
              onClick={() => { setFilter(key); setExpanded(null); }}
              className={`px-2 py-1 text-[11px] font-medium rounded-md border transition-colors ${
                filter === key
                  ? 'bg-neutral-900 text-white border-neutral-900'
                  : 'bg-white text-neutral-500 border-neutral-200 hover:border-neutral-300'
              }`}
            >
              {label} {counts[key]}
            </button>
          ))}
        </div>
      </div>

      {items.length === 0 ? (
        <p className="text-xs text-neutral-400 py-6 text-center">
          {filter === 'all' ? '작성한 보고서가 없습니다.' : '해당 유형의 보고서가 없습니다.'}
        </p>
      ) : (
        <div className="space-y-1">
          {items.map((it) => {
            const isOpen = expanded === it.key;
            return (
              <div key={it.key}>
                <div
                  className={`flex items-center justify-between px-3 py-2 rounded-lg border transition-colors ${
                    isOpen ? 'bg-blue-50 border-blue-200' : 'bg-white border-neutral-100 hover:border-neutral-200'
                  }`}
                >
                  <div className="flex items-center gap-3 min-w-0">
                    <span
                      className={`px-1.5 py-0.5 rounded text-[10px] font-medium shrink-0 ${
                        it.kind === 'site' ? 'bg-purple-100 text-purple-700' : 'bg-sky-100 text-sky-700'
                      }`}
                    >
                      {it.kind === 'site' ? '사이트' : '본사'}
                    </span>
                    <span className="text-sm font-medium text-neutral-900 shrink-0">
                      {it.kind === 'site' ? it.dateText || it.date : it.date}
                    </span>
                    {it.kind === 'site' ? (
                      <span className="text-xs text-neutral-500 truncate">{it.projectName}</span>
                    ) : (
                      it.submittedAt && (
                        <span className="text-xs text-neutral-400">
                          {new Date(it.submittedAt).toLocaleString('ko-KR', {
                            month: 'short',
                            day: 'numeric',
                            hour: '2-digit',
                            minute: '2-digit',
                          })}
                        </span>
                      )
                    )}
                    {it.kind === 'report' && (
                      <span className="px-1.5 py-0.5 bg-green-100 text-green-700 rounded text-[10px] font-medium shrink-0">
                        {it.status === 'submitted' ? '제출' : it.status}
                      </span>
                    )}
                  </div>
                  <button
                    onClick={() => toggle(it)}
                    className="text-xs text-neutral-600 hover:text-neutral-900 font-medium transition-colors shrink-0 ml-2"
                  >
                    {isOpen ? '접기' : '보기'}
                  </button>
                </div>

                {isOpen && (
                  <div className="mt-1 mb-2 bg-neutral-50 p-4 rounded-xl border border-neutral-200">
                    {it.kind === 'report' ? (
                      <ReportBody report={reportCache[it.reportId]} loading={loadingReportId === it.reportId} />
                    ) : (
                      <SiteBody report={it.report} />
                    )}
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}

function ReportBody({ report, loading }: { report: Report | null | undefined; loading: boolean }) {
  if (loading) return <Loading text="보고서 로딩 중..." />;
  if (!report) return <p className="text-xs text-neutral-400">보고서를 찾을 수 없습니다.</p>;
  return (
    <div className="grid grid-cols-1 lg:grid-cols-2 gap-4 text-sm">
      <div className="space-y-2">
        <div className="font-semibold text-neutral-900 text-sm border-b border-neutral-300 pb-1">금주실적</div>
        {report.this_week.length === 0 ? (
          <p className="text-neutral-400 text-xs">없음</p>
        ) : (
          <div className="space-y-2">
            {report.this_week.map((t, i) => (
              <div key={i} className="bg-white rounded-md px-3 py-2 border border-neutral-200 text-xs">
                <div className="flex items-center justify-between">
                  <span className="font-semibold text-neutral-900">{t.title}</span>
                  <span className="text-neutral-500 font-medium">{t.progress}%</span>
                </div>
                {t.details && <div className="text-neutral-700 mt-1 whitespace-pre-line">{t.details}</div>}
              </div>
            ))}
          </div>
        )}
      </div>
      <div className="space-y-2">
        <div className="font-semibold text-neutral-900 text-sm border-b border-neutral-300 pb-1">차주계획</div>
        {report.next_week.length === 0 ? (
          <p className="text-neutral-400 text-xs">없음</p>
        ) : (
          <div className="space-y-2">
            {report.next_week.map((t, i) => (
              <div key={i} className="bg-white rounded-md px-3 py-2 border border-neutral-200 text-xs">
                <div className="font-semibold text-neutral-900">{t.title}</div>
                {t.details && <div className="text-neutral-700 mt-1 whitespace-pre-line">{t.details}</div>}
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

function SiteBody({ report }: { report: SiteReport }) {
  return (
    <div className="space-y-3 text-sm">
      <div className="text-xs text-neutral-500">
        {report.project_name}
        {report.author_names?.length > 0 && <span className="ml-2">· {report.author_names.join(', ')}</span>}
      </div>
      <div className="space-y-2">
        <div className="font-semibold text-neutral-900 text-sm border-b border-neutral-300 pb-1">금주실적</div>
        {report.this_week.length === 0 ? (
          <p className="text-neutral-400 text-xs">없음</p>
        ) : (
          <div className="space-y-2">
            {report.this_week.map((t, i) => (
              <div key={i} className="bg-white rounded-md px-3 py-2 border border-neutral-200 text-xs">
                <div className="flex items-center justify-between">
                  <span className="font-semibold text-neutral-900">{t.title}</span>
                  <span className="text-neutral-500 font-medium">{t.progress}%</span>
                </div>
                <div className="text-neutral-500 mt-1">
                  {t.start_date && <span>시작 {t.start_date}</span>}
                  {t.due_date && <span className="ml-2">완료예정 {t.due_date}</span>}
                  {t.elapsed_days && <span className="ml-2">경과 {t.elapsed_days}</span>}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
      <div className="space-y-2">
        <div className="font-semibold text-neutral-900 text-sm border-b border-neutral-300 pb-1">차주계획</div>
        {report.next_week.length === 0 ? (
          <p className="text-neutral-400 text-xs">없음</p>
        ) : (
          <div className="space-y-2">
            {report.next_week.map((t, i) => (
              <div key={i} className="bg-white rounded-md px-3 py-2 border border-neutral-200 text-xs">
                <div className="font-semibold text-neutral-900">{t.title}</div>
                <div className="text-neutral-500 mt-1">
                  {t.start_date && <span>시작 {t.start_date}</span>}
                  {t.due_date && <span className="ml-2">완료예정 {t.due_date}</span>}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
      {report.notes && (
        <div className="space-y-1">
          <div className="font-semibold text-neutral-900 text-sm border-b border-neutral-300 pb-1">특이사항</div>
          <p className="text-neutral-700 text-xs whitespace-pre-line">{report.notes}</p>
        </div>
      )}
    </div>
  );
}
