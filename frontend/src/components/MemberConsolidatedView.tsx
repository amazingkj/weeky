import { useState, useEffect, lazy, Suspense } from 'react';
import { ConsolidatedReport, Task } from '../types';
import { getConsolidatedReport, getConsolidatedEdit } from '../services/api';
import { useAuth } from '../contexts/AuthContext';
import Loading from './ui/Loading';

const ConsolidatedPptPreview = lazy(() => import('./ConsolidatedPptPreview'));

const DAY_NAMES = ['일', '월', '화', '수', '목', '금', '토'];

function getRecentFridays(count = 8): string[] {
  const fridays: string[] = [];
  const now = new Date();
  const day = now.getDay();
  const diff = day <= 5 ? 5 - day : 5 - day + 7;
  const thisFriday = new Date(now);
  thisFriday.setDate(now.getDate() + diff);
  for (let i = 0; i < count; i++) {
    const d = new Date(thisFriday);
    d.setDate(thisFriday.getDate() - i * 7);
    fridays.push(d.toISOString().split('T')[0]);
  }
  return fridays;
}

interface MemberConsolidatedViewProps {
  teamId: number;
  myName?: string;
}

// 팀원이 본인 글이 어떻게 취합되었는지 read-only로 확인하는 뷰.
// 리더가 저장한 편집본이 있으면 그것을 우선 표시 (실제 PPT와 동일).
export default function MemberConsolidatedView({ teamId, myName }: MemberConsolidatedViewProps) {
  const { user } = useAuth();
  const [reportDate, setReportDate] = useState<string>(() => getRecentFridays(1)[0]);
  const [consolidated, setConsolidated] = useState<ConsolidatedReport | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [loaded, setLoaded] = useState(false);

  useEffect(() => {
    // 날짜 바뀌면 이전 결과 초기화
    setConsolidated(null);
    setLoaded(false);
    setError(null);
  }, [reportDate]);

  const fetchConsolidated = async () => {
    setLoading(true);
    setError(null);
    try {
      // 리더의 편집본 우선 적용 — 없으면 원본 취합
      const base = await getConsolidatedReport(teamId, reportDate);
      const edit = await getConsolidatedEdit(teamId, reportDate).catch(() => null);

      if (edit && edit.exists && edit.data) {
        setConsolidated(buildEditedConsolidated(base, edit.data, myName));
      } else {
        setConsolidated(base);
      }
      setLoaded(true);
    } catch (err: any) {
      setError(err.message || '취합 결과 조회에 실패했습니다.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="space-y-4">
      <div>
        <div className="flex items-center gap-2 mb-3">
          <h4 className="text-sm font-semibold text-neutral-900">취합 결과 보기</h4>
          <span className="text-[10px] text-neutral-400">팀 보고서가 어떻게 합쳐졌는지 확인</span>
        </div>

        <div className="flex items-center gap-2 flex-wrap">
          {getRecentFridays(8).map(friday => {
            const d = new Date(friday + 'T00:00:00');
            const label = `${d.getMonth() + 1}/${d.getDate()}(${DAY_NAMES[d.getDay()]})`;
            const isSelected = reportDate === friday;
            return (
              <button key={friday}
                onClick={() => setReportDate(friday)}
                className={`px-2.5 py-1 text-xs font-medium rounded-lg border transition-colors ${
                  isSelected
                    ? 'bg-neutral-900 text-white border-neutral-900'
                    : 'bg-white text-neutral-500 border-neutral-200 hover:border-neutral-300'
                }`}>
                {label}
              </button>
            );
          })}
        </div>
        <div className="flex items-center gap-2 mt-2">
          <span className="text-xs text-neutral-400">{reportDate} (금)</span>
          <button onClick={fetchConsolidated} disabled={loading}
            className="px-3 py-1.5 text-xs font-medium text-white bg-neutral-900 rounded-lg hover:bg-neutral-800 disabled:opacity-40 transition-colors">
            {loading ? '조회 중...' : '취합 결과 보기'}
          </button>
        </div>
      </div>

      {error && (
        <div className="p-2 bg-red-50 border border-red-200 rounded-lg text-xs text-red-700">
          {error}
          <button onClick={() => setError(null)} className="ml-2 underline">닫기</button>
        </div>
      )}

      {loaded && consolidated && (
        <>
          {/* 미제출 안내 — 본인이 미제출 상태면 표시 */}
          {(() => {
            const me = consolidated.members.find(m => m.user_id === user?.id);
            if (!me?.report) {
              return (
                <div className="p-3 bg-amber-50 border border-amber-200 rounded-lg text-xs text-amber-700">
                  이번 주 본인 보고서가 제출되지 않아 취합에 포함되지 않았습니다.
                </div>
              );
            }
            return null;
          })()}

          <Suspense fallback={<Loading text="미리보기 로딩 중..." />}>
            <ConsolidatedPptPreview data={consolidated} />
          </Suspense>
        </>
      )}

      {loaded && !consolidated && !loading && (
        <p className="text-xs text-neutral-400 text-center py-4">취합 데이터가 없습니다.</p>
      )}
    </div>
  );
}

// 리더가 저장한 편집본(flat 데이터)을 ConsolidatedReport 구조로 재조립.
// _memberName이 있는 task는 해당 멤버의 보고서로 복원하고, 없는 task는 익명 멤버로 폴백.
function buildEditedConsolidated(
  base: ConsolidatedReport,
  edit: { this_week: Task[]; next_week: Task[]; issues: string; notes: string; next_issues: string; next_notes: string },
  _myName?: string,
): ConsolidatedReport {
  // 멤버별 task 그룹화 (TeamSubmissionPanel.buildEditedConsolidated 와 동일 로직)
  const memberMap = new Map<string, { name: string; roleCode: string; thisWeek: Task[]; nextWeek: Task[] }>();
  const addToMember = (tasks: Task[], section: 'thisWeek' | 'nextWeek') => {
    for (const t of tasks) {
      const key = `${t._memberName || ''}|${t._roleCode || 'S'}`;
      if (!memberMap.has(key)) {
        memberMap.set(key, { name: t._memberName || '', roleCode: t._roleCode || 'S', thisWeek: [], nextWeek: [] });
      }
      memberMap.get(key)![section].push(t);
    }
  };
  addToMember(edit.this_week, 'thisWeek');
  addToMember(edit.next_week, 'nextWeek');

  if (memberMap.size === 0) {
    memberMap.set('|S', { name: '', roleCode: 'S', thisWeek: edit.this_week, nextWeek: edit.next_week });
  }

  // base.members에서 user_id 매핑 시도 (이름으로)
  const nameToUserId = new Map<string, number>();
  for (const m of base.members) {
    if (m.user_name) nameToUserId.set(m.user_name, m.user_id);
  }

  const members: ConsolidatedReport['members'] = [];
  let isFirst = true;
  for (const [, m] of memberMap) {
    members.push({
      user_id: nameToUserId.get(m.name) || 0,
      user_name: m.name,
      role_code: m.roleCode as any,
      report: {
        team_name: base.team.name,
        author_name: m.name,
        report_date: base.report_date,
        this_week: m.thisWeek,
        next_week: m.nextWeek,
        issues: isFirst ? edit.issues : '',
        notes: isFirst ? edit.notes : '',
        next_issues: isFirst ? edit.next_issues : '',
        next_notes: isFirst ? edit.next_notes : '',
        template_id: 0,
      },
    });
    isFirst = false;
  }

  return { ...base, members };
}
