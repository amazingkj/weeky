import { useState, useCallback, useRef, useEffect } from 'react';
import { Report, Task, Team, TeamProject, defaultTemplateStyle } from '../types';
import { generatePPT } from '../utils/pptGenerator';
import { getConfig, saveReport, getMyTeams, getReports, getMySubmission, submitReport as apiSubmitReport, unsubmitReport as apiUnsubmitReport, getTeamProjects, autoCreateTeamProject } from '../services/api';
import { useAuth } from '../contexts/AuthContext';
import { isSameWeek } from '../utils/date';
import TaskList from './TaskList';
import SyncPanel from './SyncPanel';
import PptPreview from './PptPreview';
import Alert from './ui/Alert';

const STORAGE_KEYS = {
  authorName: 'jugan_author_name',
};

const getDefaultDate = (): string => {
  return new Date().toISOString().split('T')[0];
};

const getCachedValue = (key: string): string => {
  try {
    return localStorage.getItem(key) || '';
  } catch {
    return '';
  }
};

const setCachedValue = (key: string, value: string): void => {
  try {
    localStorage.setItem(key, value);
  } catch {
    // localStorage not available
  }
};

function findPreviousWeekReport(reports: Report[], currentDate: string): Report | null {
  const prevDate = new Date(currentDate);
  prevDate.setDate(prevDate.getDate() - 7);
  const prevDateStr = prevDate.toISOString().split('T')[0];
  return reports.find(r => isSameWeek(r.report_date, prevDateStr)) || null;
}

interface ReportFormProps {
  onNavigateToConfig?: () => void;
}

export default function ReportForm({ onNavigateToConfig }: ReportFormProps) {
  const { user } = useAuth();
  const [report, setReport] = useState<Report>(() => ({
    team_name: '',
    author_name: getCachedValue(STORAGE_KEYS.authorName) || user?.name || '',
    report_date: getDefaultDate(),
    this_week: [],
    next_week: [],
    issues: '',
    notes: '',
    next_issues: '',
    next_notes: '',
    template_id: 0,
  }));
  const [isGenerating, setIsGenerating] = useState(false);
  const [showAIPanel, setShowAIPanel] = useState(false);
  const [showPreview, setShowPreview] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [hasConfiguredServices, setHasConfiguredServices] = useState<boolean | null>(null);
  const [myTeams, setMyTeams] = useState<Team[]>([]);
  const [selectedTeamId, setSelectedTeamId] = useState<number | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submittedTeams, setSubmittedTeams] = useState<Map<number, number>>(new Map()); // teamId -> reportId
  const [isSaving, setIsSaving] = useState(false);
  const [existingReports, setExistingReports] = useState<Report[]>([]);
  const [carriedForward, setCarriedForward] = useState(false);
  const [teamProjects, setTeamProjects] = useState<TeamProject[]>([]);
  const successTimerRef = useRef<ReturnType<typeof setTimeout>>();

  // Load teams, config, existing reports on mount
  useEffect(() => {
    getConfig().then((config) => {
      const hasAny = ['gitlab_token', 'jira_token', 'hiworks_password'].some(
        (key) => config[key] === '***configured***'
      );
      setHasConfiguredServices(hasAny);
    }).catch(() => {});

    // Load teams and reports together to avoid race conditions
    Promise.all([
      getMyTeams().catch(() => [] as Team[]),
      getReports().catch(() => [] as Report[]),
    ]).then(([teams, reports]) => {
      setMyTeams(teams);
      setExistingReports(reports);

      const today = getDefaultDate();
      const existing = reports.find(r => r.report_date === today);

      if (existing) {
        setReport(existing);
        // Match team_name to a team
        const matched = teams.find(t => t.name === existing.team_name);
        if (matched) {
          setSelectedTeamId(matched.id);
        } else if (teams.length === 1) {
          setSelectedTeamId(teams[0].id);
        }
      } else {
        // 이번 주 보고서 없음 → 이전 주 차주계획 자동 채움
        const prevReport = findPreviousWeekReport(reports, today);
        if (prevReport && prevReport.next_week.length > 0) {
          const carried = prevReport.next_week.map(t => ({ ...t, _carriedForward: true }));
          setReport(prev => ({ ...prev, this_week: carried }));
          setCarriedForward(true);
        }
        if (teams.length === 1) {
          setSelectedTeamId(teams[0].id);
          setReport(prev => ({ ...prev, team_name: teams[0].name }));
        } else if (teams.length > 1) {
          // Auto-select first team
          setSelectedTeamId(teams[0].id);
          setReport(prev => prev.team_name ? prev : { ...prev, team_name: teams[0].name });
        }
      }
    });
  }, []);

  // Fetch team projects when team changes
  useEffect(() => {
    if (!selectedTeamId) { setTeamProjects([]); return; }
    getTeamProjects(selectedTeamId, true).then(setTeamProjects).catch(() => setTeamProjects([]));
  }, [selectedTeamId]);

  // Check submission status when team or date changes
  useEffect(() => {
    if (!selectedTeamId || !report.report_date) return;
    getMySubmission(selectedTeamId, report.report_date).then((result) => {
      if (result.submitted && result.submission) {
        setSubmittedTeams(prev => new Map(prev).set(selectedTeamId, result.submission!.report_id));
      } else {
        setSubmittedTeams(prev => {
          const next = new Map(prev);
          next.delete(selectedTeamId);
          return next;
        });
      }
    }).catch(() => {});
  }, [selectedTeamId, report.report_date]);

  useEffect(() => {
    return () => {
      if (successTimerRef.current) clearTimeout(successTimerRef.current);
    };
  }, []);

  const showSuccess = useCallback((message: string) => {
    if (successTimerRef.current) clearTimeout(successTimerRef.current);
    setSuccess(message);
    successTimerRef.current = setTimeout(() => setSuccess(null), 3000);
  }, []);

  const updateField = useCallback(<K extends keyof Report>(field: K, value: Report[K]) => {
    setReport((prev) => {
      const next = { ...prev, [field]: value };
      // When date changes, load existing report for that date
      if (field === 'report_date' && typeof value === 'string') {
        const existing = existingReports.find(r => r.report_date === value);
        if (existing) {
          setCarriedForward(false);
          return existing;
        } else {
          // Reset to blank for new date but keep team_name and author_name
          const blank = {
            ...next,
            id: undefined,
            this_week: [] as Task[],
            next_week: [],
            issues: '',
            notes: '',
            next_issues: '',
            next_notes: '',
          };
          // 이전 주 차주계획 자동 채움
          const prevReport = findPreviousWeekReport(existingReports, value);
          if (prevReport && prevReport.next_week.length > 0) {
            blank.this_week = prevReport.next_week.map(t => ({ ...t, _carriedForward: true }));
            setCarriedForward(true);
          } else {
            setCarriedForward(false);
          }
          return blank;
        }
      }
      return next;
    });

    if (field === 'author_name' && typeof value === 'string') {
      setCachedValue(STORAGE_KEYS.authorName, value);
    }
  }, [existingReports]);

  const handleAutoCreateProject = useCallback(async (name: string) => {
    if (!selectedTeamId || !name.trim()) return;
    try {
      const project = await autoCreateTeamProject(selectedTeamId, name.trim());
      setTeamProjects(prev => {
        if (prev.some(p => p.id === project.id)) return prev;
        return [...prev, project];
      });
    } catch { /* ignore */ }
  }, [selectedTeamId]);

  const handleAIGenerate = useCallback((thisWeek: Task[], nextWeek: Task[]) => {
    setReport((prev) => ({
      ...prev,
      this_week: [...prev.this_week, ...thisWeek],
      next_week: [...prev.next_week, ...nextWeek],
    }));
    setShowAIPanel(false);
    const parts: string[] = [];
    if (thisWeek.length > 0) parts.push(`금주실적 ${thisWeek.length}건`);
    if (nextWeek.length > 0) parts.push(`차주계획 ${nextWeek.length}건`);
    showSuccess(`AI가 생성한 ${parts.join(', ')} 추가되었습니다.`);
  }, [showSuccess]);

  const validateReport = useCallback((): string | null => {
    if (!report.team_name.trim() || report.team_name === '__custom__') return '팀명을 입력해주세요.';
    if (!report.author_name.trim()) return '이름을 입력해주세요.';
    if (report.this_week.length === 0) return '금주실적을 최소 1개 이상 입력해주세요.';
    return null;
  }, [report]);

  const handleDownload = useCallback(async () => {
    const validationError = validateReport();
    if (validationError) {
      setError(validationError);
      return;
    }

    setError(null);
    setIsGenerating(true);
    try {
      await generatePPT(report, defaultTemplateStyle);
      showSuccess('PPT가 다운로드되었습니다!');
    } catch (err) {
      console.error('Failed to generate PPT:', err);
      setError('PPT 생성에 실패했습니다.');
    } finally {
      setIsGenerating(false);
    }
  }, [report, validateReport, showSuccess]);

  const handleSave = useCallback(async () => {
    const validationError = validateReport();
    if (validationError) {
      setError(validationError);
      return;
    }
    setError(null);
    setIsSaving(true);
    try {
      // _carriedForward는 UI 전용 필드 → 백엔드 전송 전 제거
      const cleanReport = {
        ...report,
        this_week: report.this_week.map(({ _carriedForward, ...t }) => t),
        next_week: report.next_week.map(({ _carriedForward, ...t }) => t),
      };
      const saved = await saveReport(cleanReport);
      setReport(saved);
      setCarriedForward(false);
      // Update existing reports cache
      setExistingReports(prev => {
        const idx = prev.findIndex(r => r.id === saved.id);
        if (idx >= 0) {
          const next = [...prev];
          next[idx] = saved;
          return next;
        }
        return [...prev, saved];
      });
      showSuccess('저장되었습니다!');
    } catch (err: any) {
      setError(err.message || '저장에 실패했습니다.');
    } finally {
      setIsSaving(false);
    }
  }, [report, validateReport, showSuccess]);

  const handleSubmitToTeam = useCallback(async () => {
    const validationError = validateReport();
    if (validationError) {
      setError(validationError);
      return;
    }
    // 완료예정일 누락 차단 — 제출 시 팝업으로 명시 알림
    const missingDue = [
      ...report.this_week.filter(t => !t.due_date?.trim()),
      ...report.next_week.filter(t => !t.due_date?.trim()),
    ];
    if (missingDue.length > 0) {
      alert('완료예정일이 비어있는 항목이 있습니다.\n모두 작성한 후 제출해주세요.');
      return;
    }
    if (!selectedTeamId) {
      setError('제출할 팀을 선택해주세요.');
      return;
    }

    setError(null);
    setIsSubmitting(true);
    try {
      // Save (upsert) first, then submit
      const cleanReport = {
        ...report,
        this_week: report.this_week.map(({ _carriedForward, ...t }) => t),
        next_week: report.next_week.map(({ _carriedForward, ...t }) => t),
      };
      const saved = await saveReport(cleanReport);
      setReport(saved);
      // Update existing reports cache
      setExistingReports(prev => {
        const idx = prev.findIndex(r => r.id === saved.id);
        if (idx >= 0) {
          const next = [...prev];
          next[idx] = saved;
          return next;
        }
        return [...prev, saved];
      });
      await apiSubmitReport(selectedTeamId, saved.id!);
      setSubmittedTeams(prev => new Map(prev).set(selectedTeamId, saved.id!));
      showSuccess('보고서가 제출되었습니다!');
    } catch (err: any) {
      setError(err.message || '제출에 실패했습니다.');
    } finally {
      setIsSubmitting(false);
    }
  }, [report, selectedTeamId, validateReport, showSuccess]);

  const handleUnsubmit = useCallback(async () => {
    if (!selectedTeamId) return;
    const reportId = submittedTeams.get(selectedTeamId);
    if (!reportId) return;

    setError(null);
    setIsSubmitting(true);
    try {
      await apiUnsubmitReport(selectedTeamId, reportId);
      setSubmittedTeams(prev => {
        const next = new Map(prev);
        next.delete(selectedTeamId);
        return next;
      });
      showSuccess('제출이 취소되었습니다. 수정 후 다시 제출해주세요.');
    } catch (err: any) {
      setError(err.message || '제출 취소에 실패했습니다.');
    } finally {
      setIsSubmitting(false);
    }
  }, [selectedTeamId, submittedTeams, showSuccess]);

  return (
    <div className="space-y-6">
      {/* Alerts */}
      {error && (
        <Alert type="error" onClose={() => setError(null)}>{error}</Alert>
      )}
      {success && (
        <Alert type="success" onClose={() => setSuccess(null)}>{success}</Alert>
      )}

      {/* Setup Guide Banner */}
      {hasConfiguredServices === false && (
        <div className="flex items-center justify-between bg-blue-50 border border-blue-200 rounded-xl px-4 py-3">
          <div className="flex items-center gap-2">
            {infoIcon}
            <span className="text-sm text-blue-800">
              연동 서비스가 설정되지 않았습니다. 설정 탭에서 토큰을 먼저 등록해주세요.
            </span>
          </div>
          {onNavigateToConfig && (
            <button
              onClick={onNavigateToConfig}
              className="shrink-0 px-3 py-1.5 bg-blue-600 text-white text-xs font-medium rounded-lg hover:bg-blue-700 transition-colors"
            >
              설정으로 이동
            </button>
          )}
        </div>
      )}

      {/* AI / Stats bar */}
      <div className="flex items-center justify-between flex-wrap gap-2">
        <div className="flex items-center gap-3 text-xs text-neutral-500">
          <span className="font-medium text-neutral-700">금주 업무 <span className="text-neutral-900 font-semibold">{report.this_week.length}</span></span>
          <span>완료 <span className="text-green-600 font-semibold">{report.this_week.filter(t => t.progress === 100).length}</span></span>
          <span>진행중 <span className="text-blue-600 font-semibold">{report.this_week.filter(t => t.progress > 0 && t.progress < 100).length}</span></span>
          <span className="font-medium text-neutral-700">차주 계획 <span className="text-neutral-900 font-semibold">{report.next_week.length}</span></span>
        </div>
        <button
          type="button"
          onClick={() => setShowAIPanel((p) => !p)}
          className={`flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium rounded-lg border transition-colors ${
            showAIPanel
              ? 'bg-neutral-900 text-white border-neutral-900'
              : 'bg-white text-neutral-600 border-neutral-200 hover:border-neutral-300'
          }`}
        >
          {aiIcon}
          AI 자동 생성
        </button>
      </div>

      {/* AI Generate Panel */}
      {showAIPanel && (
        <div className="bg-white p-5 rounded-xl border border-neutral-200 shadow-sm animate-fadeIn">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-sm font-semibold text-neutral-900">AI 자동 생성</h3>
            <button
              onClick={() => setShowAIPanel(false)}
              className="p-1 text-neutral-400 hover:text-neutral-600 transition-colors"
            >
              {closeIcon}
            </button>
          </div>
          <SyncPanel onAIGenerate={handleAIGenerate} projectNames={teamProjects.map(p => p.client ? `${p.name} (고객사: ${p.client})` : p.name)} />
        </div>
      )}

      {/* Form */}
      <form className="space-y-5" onSubmit={(e) => e.preventDefault()}>
        {/* Meta Info */}
        <section className="bg-white p-5 rounded-xl border border-neutral-200 shadow-sm">
          <SectionHeader title="기본 정보" />
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div>
              <label className="block text-xs font-medium text-neutral-500 mb-1.5">팀명</label>
              {myTeams.length > 0 ? (
                <select
                  value={report.team_name}
                  onChange={(e) => {
                    updateField('team_name', e.target.value);
                    const team = myTeams.find(t => t.name === e.target.value);
                    if (team) setSelectedTeamId(team.id);
                  }}
                  className="input"
                >
                  <option value="">팀 선택</option>
                  {myTeams.map(t => (
                    <option key={t.id} value={t.name}>{t.name}</option>
                  ))}
                  <option value="__custom__">직접 입력...</option>
                </select>
              ) : (
                <input
                  type="text"
                  value={report.team_name}
                  onChange={(e) => updateField('team_name', e.target.value)}
                  placeholder="팀명 입력"
                  className="input"
                />
              )}
              {report.team_name === '__custom__' && (
                <input
                  type="text"
                  onChange={(e) => updateField('team_name', e.target.value)}
                  placeholder="팀명 직접 입력"
                  className="input mt-1"
                  autoFocus
                />
              )}
            </div>
            <div>
              <label className="block text-xs font-medium text-neutral-500 mb-1.5">이름</label>
              <input
                type="text"
                value={report.author_name}
                onChange={(e) => updateField('author_name', e.target.value)}
                placeholder="홍길동"
                className="input"
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-neutral-500 mb-1.5">보고일자</label>
              <input
                type="date"
                value={report.report_date}
                onChange={(e) => updateField('report_date', e.target.value)}
                className="input"
              />
            </div>
          </div>
        </section>

        {/* 금주(left) + 차주(right): 각 컬럼에 업무 + 이슈 + 특이사항 */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-5">
          {/* 금주 컬럼 */}
          <div className="space-y-5">
            <section className="bg-white p-5 rounded-xl border border-neutral-200 shadow-sm">
              {carriedForward && (
                <div className="flex items-center justify-between bg-blue-50 border border-blue-200 rounded-lg px-3 py-2 mb-4">
                  <div className="flex items-center gap-2">
                    {infoIcon}
                    <div className="text-xs text-blue-800">
                      <p>지난주 차주계획에서 {report.this_week.filter(t => t._carriedForward).length}건 불러왔습니다.</p>
                      <p className="text-blue-600 mt-0.5">같은 업무 제목을 사용하면 PPT에서 자동으로 합쳐집니다.</p>
                    </div>
                  </div>
                  <button
                    type="button"
                    onClick={() => {
                      setReport(prev => ({ ...prev, this_week: prev.this_week.filter(t => !t._carriedForward) }));
                      setCarriedForward(false);
                    }}
                    className="shrink-0 px-2 py-1 text-xs font-medium text-blue-600 hover:bg-blue-100 rounded transition-colors"
                  >
                    지우기
                  </button>
                </div>
              )}
              <TaskList
                title="금주실적"
                description="이번 주에 수행한 업무를 입력하세요"
                tasks={report.this_week}
                onChange={(tasks) => updateField('this_week', tasks)}
                showProgress={true}
                emptyIcon={emptyTaskIcon}
                projectSuggestions={teamProjects}
                onAutoCreateProject={handleAutoCreateProject}
              />
            </section>
            <TextSection
              title="금주 이슈"
              icon={issueIcon}
              value={report.issues}
              onChange={(v) => updateField('issues', v)}
              placeholder="이번 주 발생한 이슈가 있다면 입력하세요..."
            />
            <TextSection
              title="금주 특이사항"
              icon={noteIcon}
              value={report.notes}
              onChange={(v) => updateField('notes', v)}
              placeholder="이번 주 특이사항이 있다면 입력하세요..."
            />
          </div>

          {/* 차주 컬럼 */}
          <div className="space-y-5">
            <section className="bg-white p-5 rounded-xl border border-neutral-200 shadow-sm">
              {report.this_week.some(t => t.progress < 100) && (
                <div className="flex items-center justify-end mb-3">
                  <button
                    type="button"
                    onClick={() => {
                      const incomplete = report.this_week.filter(t => t.progress < 100);
                      const existing = report.next_week;
                      const newTasks = incomplete.filter(
                        inc => !existing.some(ex => ex.title.trim() === inc.title.trim() && ex.details?.trim() === inc.details?.trim())
                      ).map(t => ({ ...t }));
                      if (newTasks.length > 0) {
                        updateField('next_week', [...existing, ...newTasks]);
                      }
                    }}
                    className="px-2.5 py-1.5 text-xs font-medium text-blue-600 bg-blue-50 hover:bg-blue-100 border border-blue-200 rounded-lg transition-colors"
                  >
                    미완료 업무 복사 ({report.this_week.filter(t => t.progress < 100).length}건)
                  </button>
                </div>
              )}
              <TaskList
                title="차주계획"
                description="다음 주에 예정된 업무를 입력하세요"
                tasks={report.next_week}
                onChange={(tasks) => updateField('next_week', tasks)}
                showProgress={false}
                emptyIcon={emptyPlanIcon}
                projectSuggestions={teamProjects}
                onAutoCreateProject={handleAutoCreateProject}
              />
            </section>
          </div>
        </div>

        {/* Action Buttons */}
        <div className="flex items-center justify-end gap-3 pt-2">
          {/* Save */}
          <button
            type="button"
            onClick={handleSave}
            disabled={isSaving}
            className="px-4 py-2 text-sm font-medium rounded-lg border transition-colors flex items-center gap-2
                       bg-white text-neutral-700 border-neutral-200 hover:border-neutral-400
                       disabled:opacity-40 disabled:cursor-not-allowed"
          >
            {isSaving ? <>{spinnerIcon} 저장 중...</> : <>{saveIcon} 저장</>}
          </button>

          {/* Submit to team */}
          {myTeams.length > 0 && selectedTeamId && (
            submittedTeams.has(selectedTeamId) ? (
              <div className="flex items-center gap-2">
                <span className="px-4 py-2 text-sm font-medium rounded-lg bg-green-50 text-green-700 border border-green-200 flex items-center gap-1.5">
                  {checkIcon}
                  제출완료
                </span>
                <button
                  type="button"
                  onClick={handleUnsubmit}
                  disabled={isSubmitting}
                  className="px-4 py-2 text-sm font-medium rounded-lg border transition-colors flex items-center gap-1.5
                             text-red-500 border-red-200 hover:bg-red-50
                             disabled:opacity-40 disabled:cursor-not-allowed"
                >
                  {isSubmitting ? <>{spinnerIcon} 취소 중...</> : '제출 취소'}
                </button>
              </div>
            ) : (
              <button
                type="button"
                onClick={handleSubmitToTeam}
                disabled={isSubmitting}
                className="px-4 py-2 text-sm font-medium rounded-lg border transition-colors flex items-center gap-2
                           bg-white text-neutral-700 border-neutral-200 hover:border-neutral-400
                           disabled:opacity-40 disabled:cursor-not-allowed"
              >
                {isSubmitting ? <>{spinnerIcon} 제출 중...</> : <>{submitIcon} 제출</>}
              </button>
            )
          )}

          {/* Preview */}
          <button
            type="button"
            onClick={() => setShowPreview((p) => !p)}
            className={`px-4 py-2 text-sm font-medium rounded-lg border transition-colors flex items-center gap-2 ${
              showPreview
                ? 'bg-neutral-900 text-white border-neutral-900'
                : 'bg-white text-neutral-700 border-neutral-200 hover:border-neutral-400'
            }`}
          >
            {previewIcon}
            미리보기
          </button>

          {/* Download */}
          <button
            type="button"
            onClick={handleDownload}
            disabled={isGenerating}
            className="px-4 py-2 text-sm font-medium rounded-lg border transition-colors flex items-center gap-2
                       bg-white text-neutral-700 border-neutral-200 hover:border-neutral-400
                       disabled:opacity-40 disabled:cursor-not-allowed"
          >
            {isGenerating ? <>{spinnerIcon} 생성 중...</> : <>{downloadIcon} PPT 다운로드</>}
          </button>
        </div>
      </form>

      {/* PPT Preview */}
      {showPreview && (
        <section className="bg-white p-5 rounded-xl border border-neutral-200 shadow-sm animate-fadeIn">
          <div className="flex items-center justify-between mb-4">
            <SectionHeader title="PPT 미리보기" />
            <button
              onClick={() => setShowPreview(false)}
              className="p-1 text-neutral-400 hover:text-neutral-600 transition-colors"
            >
              {closeIcon}
            </button>
          </div>
          <PptPreview report={report} style={defaultTemplateStyle} />
        </section>
      )}
    </div>
  );
}

// Sub-components

function SectionHeader({ title, optional }: { title: string; optional?: boolean }) {
  return (
    <div className="flex items-center gap-2 mb-3">
      <h3 className="text-sm font-semibold text-neutral-900">{title}</h3>
      {optional && (
        <span className="text-xs text-neutral-400">(선택)</span>
      )}
    </div>
  );
}

function TextSection({
  title,
  icon,
  value,
  onChange,
  placeholder,
}: {
  title: string;
  icon: React.ReactNode;
  value: string;
  onChange: (v: string) => void;
  placeholder: string;
}) {
  return (
    <section className="bg-white p-5 rounded-xl border border-neutral-200 shadow-sm">
      <div className="flex items-center gap-2 mb-3">
        <span className="text-neutral-400">{icon}</span>
        <h3 className="text-sm font-semibold text-neutral-900">{title}</h3>
        <span className="text-xs text-neutral-400">(선택)</span>
      </div>
      <textarea
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        rows={3}
        className="input resize-none"
      />
    </section>
  );
}

// Hoisted static SVG icons
const infoIcon = (
  <svg className="w-4 h-4 text-blue-600 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
  </svg>
);

const aiIcon = (
  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z" />
  </svg>
);

const previewIcon = (
  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
  </svg>
);

const downloadIcon = (
  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
  </svg>
);

const closeIcon = (
  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M6 18L18 6M6 6l12 12" />
  </svg>
);

const emptyTaskIcon = (
  <svg className="w-12 h-12 text-neutral-200" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4" />
  </svg>
);

const emptyPlanIcon = (
  <svg className="w-12 h-12 text-neutral-200" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1} d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
  </svg>
);

const issueIcon = (
  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
  </svg>
);

const noteIcon = (
  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
  </svg>
);

const submitIcon = (
  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
  </svg>
);

const saveIcon = (
  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M8 7H5a2 2 0 00-2 2v9a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-3m-1 4l-3 3m0 0l-3-3m3 3V4" />
  </svg>
);

const checkIcon = (
  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
  </svg>
);

const spinnerIcon = (
  <svg className="animate-spin w-4 h-4" viewBox="0 0 24 24">
    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" fill="none"/>
    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"/>
  </svg>
);
