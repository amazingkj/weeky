import { useState, useEffect } from 'react';
import { TeamHistoryResponse, ConsolidatedReport, defaultTemplateStyle } from '../types';
import { getTeamHistory, getConsolidatedReport } from '../services/api';
import { generateConsolidatedPPT } from '../utils/pptGenerator';
import { useAuth } from '../contexts/AuthContext';
import Loading from './ui/Loading';

interface WeeklyHistoryPanelProps {
  teamId: number;
}

function formatFriday(dateStr: string): string {
  const d = new Date(dateStr);
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  return `${y}-${m}-${day}`;
}

export default function WeeklyHistoryPanel({ teamId }: WeeklyHistoryPanelProps) {
  const { user } = useAuth();
  const [data, setData] = useState<TeamHistoryResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [weeks, setWeeks] = useState(12);
  const [downloadingWeek, setDownloadingWeek] = useState<string | null>(null);

  useEffect(() => {
    setLoading(true);
    getTeamHistory(teamId, weeks)
      .then(setData)
      .catch(() => setData(null))
      .finally(() => setLoading(false));
  }, [teamId, weeks]);

  const handleDownload = async (weekDate: string) => {
    setDownloadingWeek(weekDate);
    try {
      const consolidated: ConsolidatedReport = await getConsolidatedReport(teamId, weekDate);
      await generateConsolidatedPPT(consolidated, defaultTemplateStyle, user?.name);
    } catch {
      alert('PPT 생성에 실패했습니다.');
    } finally {
      setDownloadingWeek(null);
    }
  };

  if (loading) return <Loading text="히스토리 로딩 중..." />;
  if (!data) {
    return (
      <div className="text-center py-8">
        <p className="text-sm text-neutral-400">히스토리 데이터가 없습니다.</p>
      </div>
    );
  }

  const hasAnySubmissions = data.weeks.some(w => w.submitted_count > 0);

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <h4 className="text-sm font-semibold text-neutral-900">주차별 취합 PPT</h4>
        <select
          value={weeks}
          onChange={e => setWeeks(Number(e.target.value))}
          className="text-xs border border-neutral-200 rounded-lg px-2 py-1 focus:outline-none focus:border-neutral-400"
        >
          <option value={8}>최근 8주</option>
          <option value={12}>최근 12주</option>
          <option value={24}>최근 24주</option>
        </select>
      </div>

      {!hasAnySubmissions ? (
        <div className="text-center py-8">
          <p className="text-sm text-neutral-400">제출된 보고서가 없습니다.</p>
          <p className="text-xs text-neutral-300 mt-1">팀원들이 보고서를 제출하면 주차별로 표시됩니다.</p>
        </div>
      ) : (
        <div className="border border-neutral-200 rounded-lg overflow-hidden">
          <table className="w-full text-xs">
            <thead>
              <tr className="bg-neutral-50 border-b border-neutral-200">
                <th className="text-left px-3 py-2 font-medium text-neutral-600">주차 (금요일)</th>
                <th className="text-center px-3 py-2 font-medium text-neutral-600">제출현황</th>
                <th className="text-left px-3 py-2 font-medium text-neutral-600">제출자</th>
                <th className="text-center px-3 py-2 font-medium text-neutral-600">다운로드</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-neutral-100">
              {data.weeks.map((w) => {
                const hasSubmissions = w.submitted_count > 0;
                const isDownloading = downloadingWeek === w.week_date;
                return (
                  <tr key={w.week_date} className={hasSubmissions ? '' : 'opacity-40'}>
                    <td className="px-3 py-2.5 font-medium text-neutral-900">
                      {formatFriday(w.friday_date)}
                    </td>
                    <td className="px-3 py-2.5 text-center">
                      <span className={`px-1.5 py-0.5 rounded text-[10px] font-medium ${
                        w.submitted_count === w.total_members
                          ? 'bg-green-100 text-green-700'
                          : w.submitted_count > 0
                            ? 'bg-amber-100 text-amber-700'
                            : 'bg-neutral-100 text-neutral-400'
                      }`}>
                        {w.submitted_count}/{w.total_members}
                      </span>
                    </td>
                    <td className="px-3 py-2.5 text-neutral-500 truncate max-w-[200px]" title={(w.submitted_names || []).join(', ')}>
                      {(w.submitted_names || []).join(', ') || '-'}
                    </td>
                    <td className="px-3 py-2.5 text-center">
                      {hasSubmissions ? (
                        <button
                          onClick={() => handleDownload(w.week_date)}
                          disabled={isDownloading}
                          className="inline-flex items-center gap-1 px-2 py-1 text-[10px] font-medium text-neutral-600 bg-white border border-neutral-200 rounded-md hover:border-neutral-300 hover:bg-neutral-50 disabled:opacity-40 transition-colors"
                        >
                          {isDownloading ? (
                            <svg className="animate-spin w-3 h-3" viewBox="0 0 24 24">
                              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" fill="none"/>
                              <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
                            </svg>
                          ) : (
                            <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
                            </svg>
                          )}
                          PPT
                        </button>
                      ) : (
                        <span className="text-neutral-300">-</span>
                      )}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
