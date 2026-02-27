import { useState, useEffect, lazy, Suspense } from 'react';
import { Team, TeamMemberWithSubmission, Report, ConsolidatedReport, Task, ROLE_CODE_LABELS, defaultTemplateStyle } from '../types';
import { getTeamSubmissions, getTeamMemberReport, getConsolidatedReport, summarizeConsolidatedReport, updateTeamMemberReport } from '../services/api';
import TaskList from './TaskList';
import { generatePPT, generateConsolidatedPPT } from '../utils/pptGenerator';
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

  const [selectedMemberId, setSelectedMemberId] = useState<number | null>(null);
  const [selectedMemberName, setSelectedMemberName] = useState('');
  const [reportLoading, setReportLoading] = useState(false);

  const [editedReport, setEditedReport] = useState<Report | null>(null);
  const [editingMember, setEditingMember] = useState(false);
  const [fullscreen, setFullscreen] = useState(false);
  const [saving, setSaving] = useState(false);
  const [saveSuccess, setSaveSuccess] = useState(false);

  const [showPreview, setShowPreview] = useState(false);
  const [consolidated, setConsolidated] = useState<ConsolidatedReport | null>(null);
  const [pptLoading, setPptLoading] = useState(false);
  const [aiLoading, setAiLoading] = useState(false);
  const [aiResult, setAiResult] = useState<{ this_week: Task[]; next_week: Task[]; summary: string } | null>(null);

  // 취합 편집 (플랫 뷰)
  const [editingConsolidated, setEditingConsolidated] = useState(false);
  // 플랫 편집용 state
  const [flatThisWeek, setFlatThisWeek] = useState<Task[]>([]);
  const [flatNextWeek, setFlatNextWeek] = useState<Task[]>([]);
  const [flatIssues, setFlatIssues] = useState('');
  const [flatNotes, setFlatNotes] = useState('');
  const [flatNextIssues, setFlatNextIssues] = useState('');
  const [flatNextNotes, setFlatNextNotes] = useState('');

  // ESC로 전체화면 닫기
  useEffect(() => {
    if (!fullscreen) return;
    const handleKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setFullscreen(false);
    };
    window.addEventListener('keydown', handleKey);
    return () => window.removeEventListener('keydown', handleKey);
  }, [fullscreen]);

  const fetchSubmissions = async () => {
    setLoading(true);
    setError(null);
    setEditedReport(null);
    setSelectedMemberId(null);
    setConsolidated(null);
    setEditingConsolidated(false);
    setAiResult(null);
    setShowPreview(false);
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
    // 같은 멤버 클릭 → 토글 (접기)
    if (selectedMemberId === member.id) {
      setSelectedMemberId(null);
      setEditedReport(null);
      setEditingMember(false);
      setFullscreen(false);
      setSaveSuccess(false);
      return;
    }
    setReportLoading(true);
    setSelectedMemberId(member.id);
    setEditingMember(false);
    setFullscreen(false);
    setSelectedMemberName(member.user_name || '');
    try {
      const report = await getTeamMemberReport(team.id, member.submission.report_id);
      setEditedReport({ ...report, this_week: report.this_week.map(t => ({ ...t })), next_week: report.next_week.map(t => ({ ...t })) });
    } catch (err: any) {
      setError(err.message);
    } finally {
      setReportLoading(false);
    }
  };

  // ConsolidatedReport → 플랫 데이터로 변환
  const flattenConsolidated = (data: ConsolidatedReport) => {
    const thisWeek: Task[] = [];
    const nextWeek: Task[] = [];
    const mergeField = (field: 'issues' | 'notes' | 'next_issues' | 'next_notes') =>
      data.members
        .filter(m => m.report && m.report[field])
        .map(m => `[${m.user_name}] ${m.report![field]}`)
        .join('\n');

    for (const m of data.members) {
      if (!m.report) continue;
      const tag = `(${m.user_name} ${m.role_code})`;
      for (const t of m.report.this_week) {
        thisWeek.push({ ...t, details: `${tag} ${t.details || ''}`.trim() });
      }
      for (const t of m.report.next_week) {
        nextWeek.push({ ...t, details: `${tag} ${t.details || ''}`.trim() });
      }
    }

    setFlatThisWeek(thisWeek);
    setFlatNextWeek(nextWeek);
    setFlatIssues(mergeField('issues'));
    setFlatNotes(mergeField('notes'));
    setFlatNextIssues(mergeField('next_issues'));
    setFlatNextNotes(mergeField('next_notes'));
  };

  // 플랫 데이터 → ConsolidatedReport (단일 멤버) 변환
  const buildEditedConsolidated = (baseData: ConsolidatedReport): ConsolidatedReport => ({
    ...baseData,
    members: [{
      user_id: 0,
      user_name: '',
      role_code: 'S',
      report: {
        team_name: baseData.team.name,
        author_name: '',
        report_date: baseData.report_date,
        this_week: flatThisWeek,
        next_week: flatNextWeek,
        issues: flatIssues,
        notes: flatNotes,
        next_issues: flatNextIssues,
        next_notes: flatNextNotes,
        template_id: 0,
      },
    }],
  });

  const handleStartEditConsolidated = async () => {
    let data = consolidated;
    if (!data) {
      setPptLoading(true);
      try {
        data = await getConsolidatedReport(team.id, reportDate);
        setConsolidated(data);
      } catch (err: any) {
        setError(err.message);
        setPptLoading(false);
        return;
      } finally {
        setPptLoading(false);
      }
    }
    flattenConsolidated(data);
    setEditingConsolidated(true);
  };

  const handleDownloadPPT = async () => {
    setPptLoading(true);
    setError(null);
    try {
      let data: ConsolidatedReport;
      if (editingConsolidated && consolidated) {
        // 편집 모드: 플랫 데이터로 구성된 단일 멤버 ConsolidatedReport 사용
        data = buildEditedConsolidated(consolidated);
      } else {
        data = consolidated || await getConsolidatedReport(team.id, reportDate);
        if (!consolidated) setConsolidated(data);
      }
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

  // AI 결과를 취합 편집 폼에 적용
  const applyAiToConsolidated = async (result: { this_week: Task[]; next_week: Task[]; summary: string }) => {
    // 이슈/특이사항은 원본 취합 데이터에서 가져옴
    if (!consolidated) {
      setPptLoading(true);
      try {
        const data = await getConsolidatedReport(team.id, reportDate);
        setConsolidated(data);
        const mergeField = (field: 'issues' | 'notes' | 'next_issues' | 'next_notes') =>
          data.members.filter(m => m.report && m.report[field]).map(m => `[${m.user_name}] ${m.report![field]}`).join('\n');
        setFlatIssues(mergeField('issues'));
        setFlatNotes(mergeField('notes'));
        setFlatNextIssues(mergeField('next_issues'));
        setFlatNextNotes(mergeField('next_notes'));
      } catch (err: any) {
        setError(err.message);
        setPptLoading(false);
        return;
      } finally {
        setPptLoading(false);
      }
    }
    // AI 결과로 업무 목록 교체
    setFlatThisWeek(result.this_week.map(t => ({ ...t, progress: t.progress || 0, due_date: t.due_date || '' })));
    setFlatNextWeek(result.next_week.map(t => ({ ...t, progress: t.progress || 0, due_date: t.due_date || '' })));
    setEditingConsolidated(true);
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
                {submissions.map((m, idx) => (
                  <tr key={m.id}
                    onClick={() => handleMemberClick(m)}
                    className={`transition-colors ${m.submission ? 'cursor-pointer hover:bg-neutral-50' : ''} ${selectedMemberId === m.id ? 'bg-blue-50' : idx % 2 === 1 ? 'bg-neutral-50' : ''}`}>
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
          {editedReport && !reportLoading && (
            <div className={fullscreen
              ? 'fixed inset-0 z-50 bg-white overflow-auto p-6 lg:p-10'
              : 'bg-neutral-50 p-4 rounded-xl border border-neutral-200 shadow-sm'
            }>
              <div className="flex items-center justify-between mb-3">
                <h4 className={`font-semibold text-neutral-900 ${fullscreen ? 'text-lg' : 'text-sm'}`}>{selectedMemberName}의 보고서</h4>
                <div className="flex items-center gap-2">
                  {!fullscreen && (
                    <>
                      <button
                        onClick={() => generatePPT(editedReport, defaultTemplateStyle)}
                        className="flex items-center gap-1 px-2.5 py-1 text-xs font-medium rounded-lg border transition-colors bg-white text-neutral-600 border-neutral-200 hover:border-neutral-300">
                        <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
                        </svg>
                        PPT
                      </button>
                      <button
                        onClick={() => setEditingMember(!editingMember)}
                        className={`flex items-center gap-1 px-2.5 py-1 text-xs font-medium rounded-lg border transition-colors ${
                          editingMember
                            ? 'bg-neutral-900 text-white border-neutral-900'
                            : 'bg-white text-neutral-600 border-neutral-200 hover:border-neutral-300'
                        }`}>
                        <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                        </svg>
                        편집
                      </button>
                    </>
                  )}
                  <button onClick={() => setFullscreen(!fullscreen)}
                    className="flex items-center gap-1 px-2.5 py-1 text-xs font-medium rounded-lg border transition-colors bg-white text-neutral-600 border-neutral-200 hover:border-neutral-300">
                    {fullscreen ? (
                      <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 9V4.5M9 9H4.5M9 9L3.75 3.75M9 15v4.5M9 15H4.5M9 15l-5.25 5.25M15 9h4.5M15 9V4.5M15 9l5.25-5.25M15 15h4.5M15 15v4.5m0-4.5l5.25 5.25" />
                      </svg>
                    ) : (
                      <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M3.75 3.75v4.5m0-4.5h4.5m-4.5 0L9 9M3.75 20.25v-4.5m0 4.5h4.5m-4.5 0L9 15M20.25 3.75h-4.5m4.5 0v4.5m0-4.5L15 9m5.25 11.25h-4.5m4.5 0v-4.5m0 4.5L15 15" />
                      </svg>
                    )}
                    {fullscreen ? '축소' : '전체화면'}
                  </button>
                  <button onClick={() => { setEditedReport(null); setSelectedMemberId(null); setEditingMember(false); setFullscreen(false); setSaveSuccess(false); }}
                    className="p-1 text-neutral-400 hover:text-neutral-600 transition-colors">
                    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M6 18L18 6M6 6l12 12" />
                    </svg>
                  </button>
                </div>
              </div>
              <div className={fullscreen ? 'text-base' : 'text-xs'}>
                {/* 기본정보 (읽기전용) */}
                <div className={`grid grid-cols-3 gap-2 mb-3 ${fullscreen ? 'text-sm' : ''}`}>
                  <div><span className="text-neutral-500">팀명:</span> {editedReport.team_name}</div>
                  <div><span className="text-neutral-500">작성자:</span> {editedReport.author_name}</div>
                  <div><span className="text-neutral-500">일자:</span> {editedReport.report_date}</div>
                </div>

                {editingMember && !fullscreen ? (
                  /* 편집 모드: 2단 레이아웃 */
                  <div className="space-y-4">
                    <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
                      <div className="space-y-3">
                        <TaskList
                          title="금주실적"
                          tasks={editedReport.this_week}
                          onChange={(tasks) => setEditedReport(prev => prev ? { ...prev, this_week: tasks } : prev)}
                          showProgress={true}
                        />
                        <div>
                          <label className="block text-xs font-medium text-neutral-700 mb-1">이슈/위험사항</label>
                          <textarea value={editedReport.issues}
                            onChange={(e) => setEditedReport(prev => prev ? { ...prev, issues: e.target.value } : prev)}
                            rows={2} className="w-full px-2.5 py-1.5 bg-white border border-neutral-200 rounded-md focus:outline-none focus:ring-1 focus:ring-neutral-400 text-xs text-neutral-700 resize-y"
                            placeholder="이슈 사항" />
                        </div>
                        <div>
                          <label className="block text-xs font-medium text-neutral-700 mb-1">특이사항</label>
                          <textarea value={editedReport.notes}
                            onChange={(e) => setEditedReport(prev => prev ? { ...prev, notes: e.target.value } : prev)}
                            rows={2} className="w-full px-2.5 py-1.5 bg-white border border-neutral-200 rounded-md focus:outline-none focus:ring-1 focus:ring-neutral-400 text-xs text-neutral-700 resize-y"
                            placeholder="특이사항" />
                        </div>
                      </div>
                      <div className="space-y-3">
                        <TaskList
                          title="차주계획"
                          tasks={editedReport.next_week}
                          onChange={(tasks) => setEditedReport(prev => prev ? { ...prev, next_week: tasks } : prev)}
                          showProgress={false}
                        />
                        <div>
                          <label className="block text-xs font-medium text-neutral-700 mb-1">차주 이슈</label>
                          <textarea value={editedReport.next_issues}
                            onChange={(e) => setEditedReport(prev => prev ? { ...prev, next_issues: e.target.value } : prev)}
                            rows={2} className="w-full px-2.5 py-1.5 bg-white border border-neutral-200 rounded-md focus:outline-none focus:ring-1 focus:ring-neutral-400 text-xs text-neutral-700 resize-y"
                            placeholder="차주 이슈" />
                        </div>
                        <div>
                          <label className="block text-xs font-medium text-neutral-700 mb-1">차주 특이사항</label>
                          <textarea value={editedReport.next_notes}
                            onChange={(e) => setEditedReport(prev => prev ? { ...prev, next_notes: e.target.value } : prev)}
                            rows={2} className="w-full px-2.5 py-1.5 bg-white border border-neutral-200 rounded-md focus:outline-none focus:ring-1 focus:ring-neutral-400 text-xs text-neutral-700 resize-y"
                            placeholder="차주 특이사항" />
                        </div>
                      </div>
                    </div>
                    <div className="flex items-center gap-2 pt-2">
                      <button
                        onClick={async () => {
                          if (!editedReport?.id) return;
                          setSaving(true);
                          try {
                            await updateTeamMemberReport(team.id, editedReport.id, editedReport);
                            setConsolidated(null);
                            setEditingConsolidated(false);
                            setSaveSuccess(true);
                            setTimeout(() => setSaveSuccess(false), 2000);
                          } catch (err: any) {
                            setError(err.message);
                          } finally {
                            setSaving(false);
                          }
                        }}
                        disabled={saving}
                        className="px-4 py-1.5 text-xs font-medium text-white bg-neutral-900 rounded-lg hover:bg-neutral-800 disabled:opacity-40 transition-colors"
                      >
                        {saving ? '저장 중...' : '저장'}
                      </button>
                      {saveSuccess && <span className="text-xs text-green-600 font-medium">저장되었습니다</span>}
                    </div>
                  </div>
                ) : (
                  /* 읽기 모드: 2단 레이아웃 */
                  <div className={`grid grid-cols-1 lg:grid-cols-2 gap-4 ${fullscreen ? 'text-base gap-8' : 'text-sm'}`}>
                    {/* 좌: 금주실적 */}
                    <div className="space-y-2">
                      <div className="font-semibold text-neutral-900 text-sm border-b border-neutral-300 pb-1">금주실적</div>
                      {editedReport.this_week.length === 0 ? (
                        <p className="text-neutral-400">없음</p>
                      ) : (
                        <div className="space-y-2">
                          {editedReport.this_week.map((t, i) => (
                            <div key={i} className="bg-white rounded-md px-3 py-2 border border-neutral-200">
                              <div className="flex items-center justify-between">
                                <span className="font-semibold text-neutral-900">{t.title}</span>
                                <span className="text-neutral-500 text-xs font-medium">{t.progress}%</span>
                              </div>
                              {t.details && <div className="text-neutral-700 mt-1 whitespace-pre-line">{t.details}</div>}
                              {t.description && <div className="text-neutral-500 mt-1 whitespace-pre-line">{t.description}</div>}
                              {t.due_date && <div className="text-neutral-500 mt-1">완료일: {t.due_date}</div>}
                            </div>
                          ))}
                        </div>
                      )}
                      {editedReport.issues && (
                        <div className="mt-3">
                          <div className="text-neutral-600 text-xs font-semibold mb-1">이슈/위험사항</div>
                          <p className="text-neutral-800 whitespace-pre-line">{editedReport.issues}</p>
                        </div>
                      )}
                      {editedReport.notes && (
                        <div>
                          <div className="text-neutral-600 text-xs font-semibold mb-1">특이사항</div>
                          <p className="text-neutral-800 whitespace-pre-line">{editedReport.notes}</p>
                        </div>
                      )}
                    </div>
                    {/* 우: 차주계획 */}
                    <div className="space-y-2">
                      <div className="font-semibold text-neutral-900 text-sm border-b border-neutral-300 pb-1">차주계획</div>
                      {editedReport.next_week.length === 0 ? (
                        <p className="text-neutral-400">없음</p>
                      ) : (
                        <div className="space-y-2">
                          {editedReport.next_week.map((t, i) => (
                            <div key={i} className="bg-white rounded-md px-3 py-2 border border-neutral-200">
                              <div className="font-semibold text-neutral-900">{t.title}</div>
                              {t.details && <div className="text-neutral-700 mt-1 whitespace-pre-line">{t.details}</div>}
                              {t.description && <div className="text-neutral-500 mt-1 whitespace-pre-line">{t.description}</div>}
                              {t.due_date && <div className="text-neutral-500 mt-1">완료예정일: {t.due_date}</div>}
                            </div>
                          ))}
                        </div>
                      )}
                      {editedReport.next_issues && (
                        <div className="mt-3">
                          <div className="text-neutral-600 text-xs font-semibold mb-1">차주 이슈</div>
                          <p className="text-neutral-800 whitespace-pre-line">{editedReport.next_issues}</p>
                        </div>
                      )}
                      {editedReport.next_notes && (
                        <div>
                          <div className="text-neutral-600 text-xs font-semibold mb-1">차주 특이사항</div>
                          <p className="text-neutral-800 whitespace-pre-line">{editedReport.next_notes}</p>
                        </div>
                      )}
                    </div>
                  </div>
                )}
              </div>
            </div>
          )}

          {/* Action buttons */}
          {submittedCount > 0 && (
            <div className="flex gap-2 flex-wrap items-center">
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
              <div className="flex-1" />
              <button onClick={handleDownloadPPT} disabled={pptLoading}
                className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg border transition-colors bg-white text-neutral-600 border-neutral-200 hover:border-neutral-300 disabled:opacity-40">
                <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
                </svg>
                {pptLoading ? '생성 중...' : '취합 PPT 다운로드'}
              </button>
              <button onClick={() => editingConsolidated ? setEditingConsolidated(false) : handleStartEditConsolidated()} disabled={pptLoading}
                className={`flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg border transition-colors ${
                  editingConsolidated
                    ? 'bg-neutral-900 text-white border-neutral-900'
                    : 'bg-white text-neutral-600 border-neutral-200 hover:border-neutral-300'
                } disabled:opacity-40`}>
                <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                </svg>
                취합 편집
              </button>
            </div>
          )}

          {/* Consolidated edit mode (flat view) */}
          {editingConsolidated && (
            <div className="bg-amber-50 p-4 rounded-xl border border-amber-200 space-y-4">
              <div className="flex items-center justify-between">
                <h4 className="text-sm font-semibold text-amber-900">취합 보고서 편집</h4>
                <button onClick={() => setEditingConsolidated(false)}
                  className="p-1 text-amber-400 hover:text-amber-600 transition-colors">
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>
              <div className="flex items-center gap-2">
                <p className="text-xs text-amber-700 flex-1">취합된 전체 내용을 직접 수정하세요. 수정 후 PPT 다운로드/미리보기에 바로 반영됩니다.</p>
                <button
                  onClick={async () => {
                    setAiLoading(true);
                    setError(null);
                    try {
                      const result = await summarizeConsolidatedReport(team.id, reportDate);
                      setAiResult(result);
                      setFlatThisWeek(result.this_week.map(t => ({ ...t, progress: t.progress || 0, due_date: t.due_date || '' })));
                      setFlatNextWeek(result.next_week.map(t => ({ ...t, progress: t.progress || 0, due_date: t.due_date || '' })));
                    } catch (err: any) {
                      setError(err.message);
                    } finally {
                      setAiLoading(false);
                    }
                  }}
                  disabled={aiLoading}
                  className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg border transition-colors bg-white text-amber-700 border-amber-300 hover:bg-amber-100 disabled:opacity-40 whitespace-nowrap"
                >
                  <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z" />
                  </svg>
                  {aiLoading ? 'AI 정리 중...' : 'AI 정리'}
                </button>
              </div>

              {/* 2단 레이아웃: 금주(좌) / 차주(우) */}
              <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
                {/* 좌: 금주실적 */}
                <div className="space-y-3">
                  <TaskList
                    title="금주실적"
                    tasks={flatThisWeek}
                    onChange={setFlatThisWeek}
                    showProgress={true}
                  />
                  <div>
                    <label className="block text-xs font-medium text-neutral-700 mb-1">이슈/위험사항</label>
                    <textarea value={flatIssues} onChange={(e) => setFlatIssues(e.target.value)}
                      rows={3} className="w-full px-2.5 py-1.5 bg-white border border-neutral-200 rounded-md focus:outline-none focus:ring-1 focus:ring-neutral-400 text-xs text-neutral-700 resize-y"
                      placeholder="이슈/위험사항" />
                  </div>
                  <div>
                    <label className="block text-xs font-medium text-neutral-700 mb-1">특이사항</label>
                    <textarea value={flatNotes} onChange={(e) => setFlatNotes(e.target.value)}
                      rows={3} className="w-full px-2.5 py-1.5 bg-white border border-neutral-200 rounded-md focus:outline-none focus:ring-1 focus:ring-neutral-400 text-xs text-neutral-700 resize-y"
                      placeholder="특이사항" />
                  </div>
                </div>

                {/* 우: 차주계획 */}
                <div className="space-y-3">
                  <TaskList
                    title="차주계획"
                    tasks={flatNextWeek}
                    onChange={setFlatNextWeek}
                    showProgress={false}
                  />
                  <div>
                    <label className="block text-xs font-medium text-neutral-700 mb-1">차주 이슈</label>
                    <textarea value={flatNextIssues} onChange={(e) => setFlatNextIssues(e.target.value)}
                      rows={3} className="w-full px-2.5 py-1.5 bg-white border border-neutral-200 rounded-md focus:outline-none focus:ring-1 focus:ring-neutral-400 text-xs text-neutral-700 resize-y"
                      placeholder="차주 이슈" />
                  </div>
                  <div>
                    <label className="block text-xs font-medium text-neutral-700 mb-1">차주 특이사항</label>
                    <textarea value={flatNextNotes} onChange={(e) => setFlatNextNotes(e.target.value)}
                      rows={3} className="w-full px-2.5 py-1.5 bg-white border border-neutral-200 rounded-md focus:outline-none focus:ring-1 focus:ring-neutral-400 text-xs text-neutral-700 resize-y"
                      placeholder="차주 특이사항" />
                  </div>
                </div>
              </div>

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
                <button
                  onClick={() => applyAiToConsolidated(aiResult)}
                  className="mt-2 flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-white bg-blue-700 rounded-lg hover:bg-blue-800 transition-colors"
                >
                  <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                  </svg>
                  취합 편집에 적용
                </button>
              </div>
            </div>
          )}

          {/* Consolidated preview */}
          {showPreview && consolidated && (
            <Suspense fallback={<Loading text="미리보기 로딩 중..." />}>
              <ConsolidatedPptPreview data={editingConsolidated ? buildEditedConsolidated(consolidated) : consolidated} />
            </Suspense>
          )}
        </>
      )}
    </div>
  );
}
