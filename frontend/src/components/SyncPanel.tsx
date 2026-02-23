import { useState, useEffect, useCallback } from 'react';
import { Task } from '../types';
import { syncGitLab, syncJira, syncHiworks, getConfig, generateAIReport } from '../services/api';
import { SyncResult } from '../types';

interface SyncPanelProps {
  onAddItems?: (items: never[]) => void;
  onAIGenerate?: (thisWeek: Task[], nextWeek: Task[]) => void;
}

const getWeekRange = () => {
  const today = new Date();
  const dayOfWeek = today.getDay();
  const monday = new Date(today);
  monday.setDate(today.getDate() - (dayOfWeek === 0 ? 6 : dayOfWeek - 1));
  const friday = new Date(monday);
  friday.setDate(monday.getDate() + 4);

  return {
    start: monday.toISOString().split('T')[0],
    end: friday.toISOString().split('T')[0],
  };
};

type ReportStyle = 'concise' | 'detailed' | 'very_detailed';

export default function SyncPanel({ onAIGenerate }: SyncPanelProps) {
  const [dateRange, setDateRange] = useState(() => getWeekRange());
  const [reportStyle, setReportStyle] = useState<ReportStyle>(() => {
    const saved = localStorage.getItem('reportStyle');
    if (saved === 'concise' || saved === 'detailed' || saved === 'very_detailed') return saved;
    return 'concise';
  });
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [config, setConfig] = useState<Record<string, string>>({});
  const [configLoaded, setConfigLoaded] = useState(false);

  useEffect(() => {
    const loadConfig = async () => {
      try {
        const data = await getConfig();
        setConfig(data);
      } catch (err) {
        console.error('Failed to load config:', err);
      } finally {
        setConfigLoaded(true);
      }
    };
    loadConfig();
  }, []);

  const configuredServices = useCallback(() => {
    const services: string[] = [];
    if (config.gitlab_token === '***configured***') services.push('GitLab');
    if (config.jira_token === '***configured***') services.push('Jira');
    if (config.hiworks_password === '***configured***') services.push('Hiworks');
    return services;
  }, [config]);

  const handleGenerate = useCallback(async () => {
    setError(null);
    setIsLoading(true);

    const promises: Promise<SyncResult | null>[] = [];

    if (config.gitlab_token === '***configured***') {
      const baseUrl = config.gitlab_base_url || 'https://gitlab.direa.synology.me';

      // Multi-project support: check gitlab_projects first, fallback to single project
      let projectList: { namespace: string; project: string }[] = [];
      try {
        const stored = config.gitlab_projects;
        if (stored) {
          projectList = JSON.parse(stored);
        }
      } catch { /* ignore parse errors */ }

      // Fallback to legacy single project config
      if (projectList.length === 0) {
        const namespace = config.gitlab_namespace || '';
        const project = config.gitlab_project || '';
        if (namespace && project) {
          projectList = [{ namespace, project }];
        }
      }

      for (const p of projectList) {
        promises.push(
          syncGitLab({
            base_url: baseUrl, namespace: p.namespace, project: p.project,
            start_date: dateRange.start, end_date: dateRange.end,
          }).catch((err) => {
            console.warn(`GitLab sync failed for ${p.namespace}/${p.project}:`, err);
            return null;
          })
        );
      }
    }

    if (config.jira_token === '***configured***') {
      const baseUrl = config.jira_base_url || '';
      if (baseUrl) {
        promises.push(
          syncJira({
            base_url: baseUrl,
            start_date: dateRange.start, end_date: dateRange.end,
          }).catch(() => null)
        );
      }
    }

    if (config.hiworks_password === '***configured***') {
      promises.push(
        syncHiworks({
          start_date: dateRange.start, end_date: dateRange.end,
        }).catch(() => null)
      );
    }

    if (promises.length === 0) {
      setError('설정된 연동 서비스가 없습니다. 설정 탭에서 서비스를 먼저 등록해주세요.');
      setIsLoading(false);
      return;
    }

    try {
      const syncResults = (await Promise.all(promises)).filter((r): r is SyncResult => r !== null);
      const allItems = syncResults.flatMap((r) => r.items);

      if (allItems.length === 0) {
        setError('해당 기간에 동기화된 데이터가 없습니다.');
        setIsLoading(false);
        return;
      }

      const response = await generateAIReport({
        items: allItems,
        start_date: dateRange.start,
        end_date: dateRange.end,
        style: reportStyle,
      });
      onAIGenerate?.(response.this_week, response.next_week || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'AI 생성에 실패했습니다.');
    } finally {
      setIsLoading(false);
    }
  }, [config, dateRange, reportStyle, onAIGenerate]);

  const services = configuredServices();

  return (
    <div className="space-y-4">
      {/* Date Range */}
      <div className="flex flex-wrap items-center gap-3 pb-4 border-b border-neutral-100">
        <span className="text-xs font-medium text-neutral-500">조회 기간</span>
        <div className="flex items-center gap-2">
          <input
            type="date"
            value={dateRange.start}
            onChange={(e) => setDateRange({ ...dateRange, start: e.target.value })}
            className="input !w-auto !py-1.5"
          />
          <span className="text-neutral-300">~</span>
          <input
            type="date"
            value={dateRange.end}
            onChange={(e) => setDateRange({ ...dateRange, end: e.target.value })}
            className="input !w-auto !py-1.5"
          />
        </div>
      </div>

      {/* Report Style */}
      <div className="flex items-center gap-3 pb-4 border-b border-neutral-100">
        <span className="text-xs font-medium text-neutral-500">보고서 스타일</span>
        <div className="flex gap-1.5">
          <button
            type="button"
            onClick={() => { setReportStyle('concise'); localStorage.setItem('reportStyle', 'concise'); }}
            className={`px-3 py-1.5 text-xs font-medium rounded-lg border transition-colors ${
              reportStyle === 'concise'
                ? 'bg-neutral-900 text-white border-neutral-900'
                : 'bg-white text-neutral-500 border-neutral-200 hover:border-neutral-300'
            }`}
          >
            간단하게
          </button>
          <button
            type="button"
            onClick={() => { setReportStyle('detailed'); localStorage.setItem('reportStyle', 'detailed'); }}
            className={`px-3 py-1.5 text-xs font-medium rounded-lg border transition-colors ${
              reportStyle === 'detailed'
                ? 'bg-neutral-900 text-white border-neutral-900'
                : 'bg-white text-neutral-500 border-neutral-200 hover:border-neutral-300'
            }`}
          >
            상세하게
          </button>
          <button
            type="button"
            onClick={() => { setReportStyle('very_detailed'); localStorage.setItem('reportStyle', 'very_detailed'); }}
            className={`px-3 py-1.5 text-xs font-medium rounded-lg border transition-colors ${
              reportStyle === 'very_detailed'
                ? 'bg-neutral-900 text-white border-neutral-900'
                : 'bg-white text-neutral-500 border-neutral-200 hover:border-neutral-300'
            }`}
          >
            완전상세
          </button>
        </div>
        <span className="text-[10px] text-neutral-400">
          {reportStyle === 'concise' ? '한 줄 요약 스타일' : reportStyle === 'detailed' ? '세부 항목 포함 상세 스타일' : '모든 작업을 빠짐없이 기술하는 완전 상세 스타일'}
        </span>
      </div>

      {/* Configured Services Info */}
      {configLoaded ? (
        <div className="flex items-center gap-2 flex-wrap">
          {services.length > 0 ? (
            services.map((name) => (
              <span
                key={name}
                className="text-[10px] px-2 py-0.5 rounded-full bg-neutral-100 text-neutral-600 font-medium"
              >
                {name}
              </span>
            ))
          ) : (
            <span className="text-xs text-neutral-400">설정 탭에서 서비스를 등록해주세요.</span>
          )}
        </div>
      ) : null}

      {/* AI Generate Button */}
      <button
        onClick={handleGenerate}
        disabled={isLoading || services.length === 0}
        className="w-full px-4 py-3 bg-neutral-900 text-white text-sm font-medium rounded-lg
                   hover:bg-neutral-800 disabled:opacity-40 disabled:cursor-not-allowed
                   transition-colors flex items-center justify-center gap-2"
      >
        {isLoading ? spinner : aiIcon}
        {isLoading ? '데이터 수집 + AI 생성 중...' : 'AI 자동 생성'}
      </button>

      {/* Error */}
      {error ? (
        <p className="text-sm text-red-600 p-3 bg-red-50 rounded-lg border border-red-200">{error}</p>
      ) : null}
    </div>
  );
}

// Hoisted icons
const aiIcon = (
  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z" />
  </svg>
);
const spinner = (
  <svg className="animate-spin w-4 h-4" viewBox="0 0 24 24">
    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" fill="none"/>
    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
  </svg>
);
