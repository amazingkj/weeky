import { useMemo } from 'react';
import { ConsolidatedReport, MemberReportData, Task } from '../types';
import { formatDateShort, getWeekRange, getNextWeekRange } from '../utils/date';

interface ConsolidatedPptPreviewProps {
  data: ConsolidatedReport;
}

// Group tasks by title across all members, with member attribution
interface ConsolidatedTask {
  title: string;
  items: { task: Task; memberName: string; roleCode: string }[];
}

function groupTasksByTitle(
  members: MemberReportData[],
  section: 'this_week' | 'next_week'
): ConsolidatedTask[] {
  const groups: ConsolidatedTask[] = [];
  const indexMap = new Map<string, number>();

  for (const m of members) {
    if (!m.report) continue;
    const tasks = section === 'this_week' ? m.report.this_week : m.report.next_week;
    for (const t of tasks) {
      const key = t.title.trim();
      if (indexMap.has(key)) {
        groups[indexMap.get(key)!].items.push({
          task: t,
          memberName: m.user_name,
          roleCode: m.role_code,
        });
      } else {
        indexMap.set(key, groups.length);
        groups.push({
          title: key,
          items: [{ task: t, memberName: m.user_name, roleCode: m.role_code }],
        });
      }
    }
  }
  return groups;
}

function mergeText(members: MemberReportData[], field: 'issues' | 'notes' | 'next_issues' | 'next_notes'): string {
  return members
    .filter(m => m.report && m.report[field])
    .map(m => `[${m.user_name}] ${m.report![field]}`)
    .join('\n');
}

// Reusable cell
function Cell({
  children, header = false, colSpan = 1, align = 'left', valign = 'middle', className = '',
}: {
  children?: React.ReactNode; header?: boolean; colSpan?: number; align?: string; valign?: string; className?: string;
}) {
  const alignClass = align === 'center' ? 'text-center' : align === 'right' ? 'text-right' : 'text-left';
  const valignClass = valign === 'top' ? 'align-top' : 'align-middle';
  return (
    <td
      className={`border border-black px-1.5 py-0.5 text-[10px] ${alignClass} ${valignClass} ${header ? 'font-bold' : ''} ${className}`}
      style={{ backgroundColor: header ? '#F5F5F5' : '#FFFFFF' }}
      colSpan={colSpan}
    >
      {children}
    </td>
  );
}

function TaskGroupRows({ groups, showProgress }: { groups: ConsolidatedTask[]; showProgress: boolean }) {
  return (
    <>
      {groups.map((g, gi) => (
        <tr key={gi}>
          <Cell valign="top" className="font-medium whitespace-pre-line">
            {gi + 1}. {g.title}
          </Cell>
          <Cell valign="top" className="whitespace-pre-line">
            {g.items.map((item, ii) => (
              <div key={ii} className="py-0.5">
                <span className="text-neutral-500">({item.memberName} {item.roleCode})</span>{' '}
                {item.task.details || '-'}
                {item.task.description && <div className="text-neutral-400 ml-2">{item.task.description}</div>}
              </div>
            ))}
          </Cell>
          <Cell valign="top" align="center" className="whitespace-pre-line">
            {g.items.map((item, ii) => (
              <div key={ii} className="py-0.5">{item.task.due_date || '-'}</div>
            ))}
          </Cell>
          {showProgress && (
            <Cell valign="top" align="center" className="whitespace-pre-line">
              {g.items.map((item, ii) => (
                <div key={ii} className="py-0.5">{item.task.progress}%</div>
              ))}
            </Cell>
          )}
        </tr>
      ))}
    </>
  );
}

export default function ConsolidatedPptPreview({ data }: ConsolidatedPptPreviewProps) {
  const slides = useMemo(() => {
    const thisWeekGroups = groupTasksByTitle(data.members, 'this_week');
    const nextWeekGroups = groupTasksByTitle(data.members, 'next_week');
    const issues = mergeText(data.members, 'issues');
    const notes = mergeText(data.members, 'notes');
    const nextIssues = mergeText(data.members, 'next_issues');
    const nextNotes = mergeText(data.members, 'next_notes');
    const dateRange = getWeekRange(data.report_date);
    const nextDateRange = getNextWeekRange(data.report_date);

    const slideList: { title: string; content: React.ReactNode }[] = [];

    // Build 2-column slide layout
    slideList.push({
      title: '취합 보고서',
      content: (
        <div className="h-full p-1 flex flex-col gap-0 text-[10px]">
          {/* Header row */}
          <table className="w-full border-collapse">
            <tbody>
              <tr>
                <Cell header className="w-[13%]">프로젝트명</Cell>
                <Cell className="w-[20%]">{data.team.name}</Cell>
                <Cell header className="w-[10%]">보고일자</Cell>
                <Cell className="w-[12%]">{formatDateShort(data.report_date)}</Cell>
                <Cell header className="w-[10%]">작성자</Cell>
                <Cell className="w-[35%]">
                  {data.members.filter(m => m.report).map(m => `${m.user_name}(${m.role_code})`).join(', ')}
                </Cell>
              </tr>
            </tbody>
          </table>

          {/* 2-column body: 금주실적 | 차주계획 */}
          <div className="flex flex-1 gap-0 -mt-px">
            {/* Left: 금주실적 */}
            <div className="flex-1 flex flex-col border-r border-black -mr-px">
              <table className="w-full border-collapse">
                <tbody>
                  <tr><Cell header align="center" colSpan={4}>금주실적</Cell></tr>
                  <tr>
                    <Cell header className="w-[25%]">계획업무 ({dateRange})</Cell>
                    <Cell header align="center" className="w-[45%]">진행 사항</Cell>
                    <Cell header align="center" className="w-[15%]">완료일</Cell>
                    <Cell header align="center" className="w-[15%]">실적(%)</Cell>
                  </tr>
                  <TaskGroupRows groups={thisWeekGroups} showProgress={true} />
                </tbody>
              </table>
            </div>

            {/* Right: 차주계획 */}
            <div className="flex-1 flex flex-col">
              <table className="w-full border-collapse">
                <tbody>
                  <tr><Cell header align="center" colSpan={3}>차주계획</Cell></tr>
                  <tr>
                    <Cell header className="w-[30%]">계획업무 ({nextDateRange})</Cell>
                    <Cell header align="center" className="w-[45%]"></Cell>
                    <Cell header align="center" className="w-[25%]">완료예정일</Cell>
                  </tr>
                  <TaskGroupRows groups={nextWeekGroups} showProgress={false} />
                </tbody>
              </table>
            </div>
          </div>

          {/* Footer: issues + notes */}
          <table className="w-full border-collapse -mt-px">
            <tbody>
              <tr>
                <Cell header className="w-[15%]">이슈/위험사항</Cell>
                <Cell className="whitespace-pre-line">{issues || nextIssues || '-'}</Cell>
              </tr>
              <tr>
                <Cell header className="w-[15%]">특이사항</Cell>
                <Cell className="whitespace-pre-line">{notes || nextNotes || '-'}</Cell>
              </tr>
            </tbody>
          </table>
        </div>
      ),
    });

    return slideList;
  }, [data]);

  return (
    <div className="space-y-4">
      <h3 className="text-sm font-semibold text-neutral-800">취합 PPT 미리보기</h3>
      <div className="space-y-6">
        {slides.map((slide, idx) => (
          <div key={idx}
            className="aspect-[4/3] bg-white rounded-lg shadow-lg border-2 border-gray-200 overflow-hidden max-w-5xl mx-auto">
            <div className="h-full flex flex-col">
              <div className="text-xs text-gray-600 font-medium px-4 py-2 bg-gray-50 border-b">
                슬라이드 {idx + 1}: {slide.title}
              </div>
              <div className="flex-1 overflow-auto">{slide.content}</div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
