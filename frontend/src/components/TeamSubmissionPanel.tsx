import { useState, lazy, Suspense } from 'react';
import { Team, TeamMemberWithSubmission, Report, ConsolidatedReport, Task, ROLE_CODE_LABELS, defaultTemplateStyle } from '../types';
import { getTeamSubmissions, getTeamMemberReport, getConsolidatedReport, summarizeConsolidatedReport } from '../services/api';
import { generateConsolidatedPPT } from '../utils/pptGenerator';
import { useAuth } from '../contexts/AuthContext';
import Loading from './ui/Loading';

const ConsolidatedPptPreview = lazy(() => import('./ConsolidatedPptPreview'));

interface TeamSubmissionPanelProps {
  team: Team;
}

const getDefaultDate = (): string => new Date().toISOString().split('T')[0];

export default function TeamSubmissionPanel({ team }: TeamSubmissionPanelProps) {
  const { user } = useAuth();
  const [reportDate, setReportDate] = useState(getDefaultDate);
  const [submissions, setSubmissions] = useState<TeamMemberWithSubmission[]>([]);
  const [loaded, setLoaded] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [selectedReport, setSelectedReport] = useState<Report | null>(null);
  const [selectedMemberName, setSelectedMemberName] = useState('');
  const [reportLoading, setReportLoading] = useState(false);

  const [showPreview, setShowPreview] = useState(false);
  const [consolidated, setConsolidated] = useState<ConsolidatedReport | null>(null);
  const [pptLoading, setPptLoading] = useState(false);
  const [aiLoading, setAiLoading] = useState(false);
  const [aiResult, setAiResult] = useState<{ this_week: Task[]; next_week: Task[]; summary: string } | null>(null);

  const fetchSubmissions = async () => {
    setLoading(true);
    setError(null);
    setSelectedReport(null);
    try {
      const data = await getTeamSubmissions(team.id, reportDate);
      setSubmissions(data);
      setLoaded(true);
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  const handleMemberClick = async (member: TeamMemberWithSubmission) => {
    if (!member.submission) return;
    setReportLoading(true);
    setSelectedMemberName(member.user_name || '');
    try {
      const report = await getTeamMemberReport(team.id, member.submission.report_id);
      setSelectedReport(report);
    } catch (err: any) {
      setError(err.message);
    } finally {
      setReportLoading(false);
    }
  };

  const handleDownloadPPT = async () => {
    setPptLoading(true);
    setError(null);
    try {
      const data = await getConsolidatedReport(team.id, reportDate);
      setConsolidated(data);
      await generateConsolidatedPPT(data, defaultTemplateStyle, user?.name);
    } catch (err: any) {
      setError(err.message);
    } finally {
      setPptLoading(false);
    }
  };

  const handleAISummarize = async () => {
    setAiLoading(true);
    setError(null);
    try {
      const result = await summarizeConsolidatedReport(team.id, reportDate);
      setAiResult(result);
    } catch (err: any) {
      setError(err.message);
    } finally {
      setAiLoading(false);
    }
  };

  const handleTogglePreview = async () => {
    if (!showPreview && !consolidated) {
      setPptLoading(true);
      try {
        const data = await getConsolidatedReport(team.id, reportDate);
        setConsolidated(data);
      } catch (err: any) {
        setError(err.message);
        setPptLoading(false);
        return;
      } finally {
        setPptLoading(false);
      }
    }
    setShowPreview(!showPreview);
  };

  const submittedCount = submissions.filter(s => s.submission).length;

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2 mb-3">
        <h3 className="text-sm font-semibold text-neutral-900">제출 현황</h3>
      </div>

      {error && (
        <div className="p-2 bg-red-50 border border-red-200 rounded-lg text-xs text-red-700">{error}
          <button onClick={() => setError(null)} className="ml-2 underline">닫기</button>
        </div>
      )}

      {/* Date picker + fetch */}
      <div className="flex items-end gap-2">
        <div>
          <label className="block text-xs text-neutral-500 mb-1">보고 일자</label>
          <input type="date" value={reportDate} onChange={(e) => setReportDate(e.target.value)}
            className="input text-xs" />
        </div>
        <button onClick={fetchSubmissions} disabled={loading}
          className="px-3 py-1.5 text-xs font-medium text-white bg-neutral-900 rounded-lg hover:bg-neutral-800 disabled:opacity-40 transition-colors">
          {loading ? '조회 중...' : '조회'}
        </button>
      </div>

      {/* Submissions table */}
      {loaded && (
        <>
          <div className="text-xs text-neutral-500">
            제출: {submittedCount}/{submissions.length}명
          </div>
          <div className="border border-neutral-200 rounded-lg overflow-hidden">
            <table className="w-full text-xs">
              <thead>
                <tr className="bg-neutral-50 border-b border-neutral-200">
                  <th className="text-left px-3 py-2 font-medium text-neutral-600">이름</th>
                  <th className="text-left px-3 py-2 font-medium text-neutral-600">이메일</th>
                  <th className="text-center px-3 py-2 font-medium text-neutral-600">직급</th>
                  <th className="text-center px-3 py-2 font-medium text-neutral-600">상태</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-neutral-100">
                {submissions.map((m) => (
                  <tr key={m.id}
                    onClick={() => handleMemberClick(m)}
                    className={`transition-colors ${m.submission ? 'cursor-pointer hover:bg-neutral-50' : ''}`}>
                    <td className="px-3 py-2 font-medium text-neutral-900">{m.user_name}</td>
                    <td className="px-3 py-2 text-neutral-500">{m.user_email}</td>
                    <td className="px-3 py-2 text-center">
                      <span className="px-1.5 py-0.5 bg-neutral-100 text-neutral-600 rounded text-[10px] font-medium">
                        {m.role_code} ({ROLE_CODE_LABELS[m.role_code]})
                      </span>
                    </td>
                    <td className="px-3 py-2 text-center">
                      {m.submission ? (
                        <span className="px-1.5 py-0.5 bg-green-100 text-green-700 rounded text-[10px] font-medium">제출완료</span>
                      ) : (
                        <span className="px-1.5 py-0.5 bg-neutral-100 text-neutral-400 rounded text-[10px] font-medium">미제출</span>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {/* Selected member report detail */}
          {reportLoading && <div className="py-4"><Loading text="보고서 로딩 중..." /></div>}
          {selectedReport && !reportLoading && (
            <div className="bg-neutral-50 p-4 rounded-xl border border-neutral-200">
              <div className="flex items-center justify-between mb-3">
                <h4 className="text-sm font-semibold text-neutral-900">{selectedMemberName}의 보고서</h4>
                <button onClick={() => setSelectedReport(null)}
                  className="p-1 text-neutral-400 hover:text-neutral-600 transition-colors">
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>
              <div className="space-y-3 text-xs">
                <div className="grid grid-cols-3 gap-2">
                  <div><span className="text-neutral-500">팀명:</span> {selectedReport.team_name}</div>
                  <div><span className="text-neutral-500">작성자:</span> {selectedReport.author_name}</div>
                  <div><span className="text-neutral-500">일자:</span> {selectedReport.report_date}</div>
                </div>
                {selectedReport.this_week.length > 0 && (
                  <div>
                    <div className="font-medium text-neutral-700 mb-1">금주실적 ({selectedReport.this_week.length}건)</div>
                    {selectedReport.this_week.map((t, i) => (
                      <div key={i} className="ml-2 py-1 border-b border-neutral-100 last:border-0">
                        <div className="font-medium">{t.title} <span className="text-neutral-400">{t.progress}%</span></div>
                        {t.details && <div className="text-neutral-500">{t.details}</div>}
                      </div>
                    ))}
                  </div>
                )}
                {selectedReport.next_week.length > 0 && (
                  <div>
                    <div className="font-medium text-neutral-700 mb-1">차주계획 ({selectedReport.next_week.length}건)</div>
                    {selectedReport.next_week.map((t, i) => (
                      <div key={i} className="ml-2 py-1 border-b border-neutral-100 last:border-0">
                        <div className="font-medium">{t.title}</div>
                        {t.details && <div className="text-neutral-500">{t.details}</div>}
                      </div>
                    ))}
                  </div>
                )}
                {selectedReport.issues && <div><span className="font-medium text-neutral-700">이슈:</span> {selectedReport.issues}</div>}
                {selectedReport.notes && <div><span className="font-medium text-neutral-700">특이사항:</span> {selectedReport.notes}</div>}
              </div>
            </div>
          )}

          {/* Action buttons */}
          {submittedCount > 0 && (
            <div className="flex gap-2 flex-wrap">
              <button onClick={handleDownloadPPT} disabled={pptLoading}
                className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-white bg-neutral-900 rounded-lg hover:bg-neutral-800 disabled:opacity-40 transition-colors">
                <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
                </svg>
                {pptLoading ? '생성 중...' : '취합 PPT 다운로드'}
              </button>
              <button onClick={handleTogglePreview}
                className={`flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg border transition-colors ${
                  showPreview
                    ? 'bg-neutral-900 text-white border-neutral-900'
                    : 'bg-white text-neutral-600 border-neutral-200 hover:border-neutral-300'
                }`}>
                <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                </svg>
                미리보기
              </button>
              <button onClick={handleAISummarize} disabled={aiLoading}
                className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg border transition-colors bg-white text-neutral-600 border-neutral-200 hover:border-neutral-300 disabled:opacity-40">
                <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z" />
                </svg>
                {aiLoading ? 'AI 요약 중...' : 'AI 요약'}
              </button>
            </div>
          )}

          {/* AI Summary Result */}
          {aiResult && (
            <div className="bg-blue-50 p-4 rounded-xl border border-blue-200">
              <div className="flex items-center justify-between mb-3">
                <h4 className="text-sm font-semibold text-blue-900">AI 요약 결과</h4>
                <button onClick={() => setAiResult(null)}
                  className="p-1 text-blue-400 hover:text-blue-600 transition-colors">
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>
              <div className="space-y-3 text-xs">
                {aiResult.summary && (
                  <div className="text-sm text-blue-800 mb-2">{aiResult.summary}</div>
                )}
                {aiResult.this_week.length > 0 && (
                  <div>
                    <div className="font-medium text-blue-700 mb-1">금주실적 ({aiResult.this_week.length}건)</div>
                    {aiResult.this_week.map((t, i) => (
                      <div key={i} className="ml-2 py-1 border-b border-blue-100 last:border-0">
                        <div className="font-medium text-blue-900">{t.title}</div>
                        {t.details && <div className="text-blue-700">{t.details}</div>}
                      </div>
                    ))}
                  </div>
                )}
                {aiResult.next_week.length > 0 && (
                  <div>
                    <div className="font-medium text-blue-700 mb-1">차주계획 ({aiResult.next_week.length}건)</div>
                    {aiResult.next_week.map((t, i) => (
                      <div key={i} className="ml-2 py-1 border-b border-blue-100 last:border-0">
                        <div className="font-medium text-blue-900">{t.title}</div>
                        {t.details && <div className="text-blue-700">{t.details}</div>}
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>
          )}

          {/* Consolidated preview */}
          {showPreview && consolidated && (
            <Suspense fallback={<Loading text="미리보기 로딩 중..." />}>
              <ConsolidatedPptPreview data={consolidated} />
            </Suspense>
          )}
        </>
      )}
    </div>
  );
}
