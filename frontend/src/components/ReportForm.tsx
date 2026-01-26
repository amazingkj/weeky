import { useState, useCallback, useRef, useEffect } from 'react';
import { Report, Task, defaultTemplateStyle } from '../types';
import { generatePPT } from '../utils/pptGenerator';
import TaskList from './TaskList';
import SyncPanel from './SyncPanel';
import PptPreview from './PptPreview';
import Alert from './ui/Alert';

const STORAGE_KEYS = {
  teamName: 'weeky_team_name',
  authorName: 'weeky_author_name',
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

const initialReport: Report = {
  team_name: getCachedValue(STORAGE_KEYS.teamName),
  author_name: getCachedValue(STORAGE_KEYS.authorName),
  report_date: getDefaultDate(),
  this_week: [],
  next_week: [],
  issues: '',
  template_id: 0,
};

export default function ReportForm() {
  const [report, setReport] = useState<Report>(initialReport);
  const [isGenerating, setIsGenerating] = useState(false);
  const [showSyncPanel, setShowSyncPanel] = useState(false);
  const [showPreview, setShowPreview] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const successTimerRef = useRef<ReturnType<typeof setTimeout>>();

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
    setReport((prev) => ({ ...prev, [field]: value }));

    if (field === 'team_name' && typeof value === 'string') {
      setCachedValue(STORAGE_KEYS.teamName, value);
    }
    if (field === 'author_name' && typeof value === 'string') {
      setCachedValue(STORAGE_KEYS.authorName, value);
    }
  }, []);

  const handleAIGenerate = useCallback((thisWeek: Task[], nextWeek: Task[]) => {
    setReport((prev) => ({
      ...prev,
      this_week: [...prev.this_week, ...thisWeek],
      next_week: [...prev.next_week, ...nextWeek],
    }));
    setShowSyncPanel(false);
    const parts: string[] = [];
    if (thisWeek.length > 0) parts.push(`금주실적 ${thisWeek.length}건`);
    if (nextWeek.length > 0) parts.push(`차주계획 ${nextWeek.length}건`);
    showSuccess(`AI가 생성한 ${parts.join(', ')} 추가되었습니다.`);
  }, [showSuccess]);

  const validateReport = useCallback((): string | null => {
    if (!report.team_name.trim()) return '팀명을 입력해주세요.';
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

  const completedTasks = report.this_week.filter(t => t.progress === 100).length;
  const totalTasks = report.this_week.length;

  return (
    <div className="space-y-6">
      {/* Alerts */}
      {error ? (
        <Alert type="error" onClose={() => setError(null)}>{error}</Alert>
      ) : null}
      {success ? (
        <Alert type="success" onClose={() => setSuccess(null)}>{success}</Alert>
      ) : null}

      {/* Quick Stats */}
      {totalTasks > 0 ? (
        <div className="flex gap-6 text-sm">
          <Stat label="금주 업무" value={totalTasks} />
          <Stat label="완료" value={completedTasks} />
          <Stat label="진행중" value={totalTasks - completedTasks} />
          <Stat label="차주 계획" value={report.next_week.length} />
        </div>
      ) : null}

      {/* Action Buttons */}
      <div className="flex gap-2">
        <button
          type="button"
          onClick={() => setShowSyncPanel((p) => !p)}
          className={`flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium rounded-lg border transition-colors ${
            showSyncPanel
              ? 'bg-neutral-900 text-white border-neutral-900'
              : 'bg-white text-neutral-600 border-neutral-200 hover:border-neutral-300'
          }`}
        >
          {syncIcon}
          데이터 연동
        </button>
        <button
          type="button"
          onClick={() => setShowPreview((p) => !p)}
          className={`flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium rounded-lg border transition-colors ${
            showPreview
              ? 'bg-neutral-900 text-white border-neutral-900'
              : 'bg-white text-neutral-600 border-neutral-200 hover:border-neutral-300'
          }`}
        >
          {previewIcon}
          미리보기
        </button>
      </div>

      {/* Sync Panel */}
      {showSyncPanel ? (
        <div className="bg-white p-5 rounded-xl border border-neutral-200 animate-fadeIn">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-sm font-semibold text-neutral-900">외부 서비스 연동</h3>
            <button
              onClick={() => setShowSyncPanel(false)}
              className="p-1 text-neutral-400 hover:text-neutral-600 transition-colors"
            >
              {closeIcon}
            </button>
          </div>
          <SyncPanel onAIGenerate={handleAIGenerate} />
        </div>
      ) : null}

      {/* Form */}
      <form className="space-y-5" onSubmit={(e) => e.preventDefault()}>
        {/* Meta Info */}
        <section className="bg-white p-5 rounded-xl border border-neutral-200">
          <SectionHeader title="기본 정보" />
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div>
              <label className="block text-xs font-medium text-neutral-500 mb-1.5">팀명</label>
              <input
                type="text"
                value={report.team_name}
                onChange={(e) => updateField('team_name', e.target.value)}
                placeholder="개발팀"
                className="input"
              />
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

        {/* This Week */}
        <section className="bg-white p-5 rounded-xl border border-neutral-200">
          <TaskList
            title="금주실적"
            description="이번 주에 수행한 업무를 입력하세요"
            tasks={report.this_week}
            onChange={(tasks) => updateField('this_week', tasks)}
            showProgress={true}
            emptyIcon={emptyTaskIcon}
          />
        </section>

        {/* Next Week */}
        <section className="bg-white p-5 rounded-xl border border-neutral-200">
          <TaskList
            title="차주계획"
            description="다음 주에 예정된 업무를 입력하세요"
            tasks={report.next_week}
            onChange={(tasks) => updateField('next_week', tasks)}
            showProgress={false}
            emptyIcon={emptyPlanIcon}
          />
        </section>

        {/* Issues */}
        <section className="bg-white p-5 rounded-xl border border-neutral-200">
          <SectionHeader title="이슈/특이사항" optional />
          <textarea
            value={report.issues}
            onChange={(e) => updateField('issues', e.target.value)}
            placeholder="이슈나 특이사항이 있다면 입력하세요..."
            rows={3}
            className="input resize-none"
          />
        </section>

        {/* Download Button */}
        <div className="flex justify-end pt-2">
          <button
            type="button"
            onClick={handleDownload}
            disabled={isGenerating}
            className="px-5 py-2.5 bg-neutral-900 text-white text-sm font-medium rounded-lg
                       hover:bg-neutral-800 disabled:opacity-40 disabled:cursor-not-allowed
                       transition-colors flex items-center gap-2"
          >
            {isGenerating ? (
              <>
                <svg className="animate-spin w-4 h-4" viewBox="0 0 24 24">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" fill="none"/>
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"/>
                </svg>
                생성 중...
              </>
            ) : (
              <>
                {downloadIcon}
                PPT 다운로드
              </>
            )}
          </button>
        </div>
      </form>

      {/* PPT Preview */}
      {showPreview ? (
        <section className="bg-white p-5 rounded-xl border border-neutral-200 animate-fadeIn">
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
      ) : null}
    </div>
  );
}

// Sub-components

function SectionHeader({ title, optional }: { title: string; optional?: boolean }) {
  return (
    <div className="flex items-center gap-2 mb-3">
      <h3 className="text-sm font-semibold text-neutral-900">{title}</h3>
      {optional ? (
        <span className="text-xs text-neutral-400">(선택)</span>
      ) : null}
    </div>
  );
}

function Stat({ label, value }: { label: string; value: number }) {
  return (
    <div className="flex items-center gap-1.5">
      <span className="text-lg font-semibold text-neutral-900 tabular-nums">{value}</span>
      <span className="text-xs text-neutral-500">{label}</span>
    </div>
  );
}

// Hoisted static SVG icons
const syncIcon = (
  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
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
