import { useMemo } from 'react';
import { Report, Task, TemplateStyle, defaultTemplateStyle } from '../types';
import { formatDateShort, getWeekRange } from '../utils/date';

interface PptPreviewProps {
  report: Report;
  style?: TemplateStyle;
}

// Template colors
const COLORS = {
  headerBg: '#F5F5F5',
  white: '#FFFFFF',
  border: '#E5E5E5',
};

const ALIGN_CLASS = { left: 'text-left', center: 'text-center', right: 'text-right' } as const;
const VALIGN_CLASS = { top: 'align-top', middle: 'align-middle', bottom: 'align-bottom' } as const;

function Cell({
  children,
  header = false,
  colSpan = 1,
  rowSpan = 1,
  align = 'left',
  valign = 'middle',
  compact = false,
  className = '',
}: {
  children?: React.ReactNode;
  header?: boolean;
  colSpan?: number;
  rowSpan?: number;
  align?: 'left' | 'center' | 'right';
  valign?: 'top' | 'middle' | 'bottom';
  compact?: boolean;
  className?: string;
}) {
  const alignClass = ALIGN_CLASS[align];
  const valignClass = VALIGN_CLASS[valign];
  const paddingClass = compact ? 'px-1 py-0' : 'px-2 py-1';
  const textSize = compact ? 'text-[10px]' : 'text-xs';

  return (
    <td
      className={`border border-black ${paddingClass} ${textSize} ${alignClass} ${valignClass} ${header ? 'font-bold' : ''} ${className}`}
      style={{ backgroundColor: header ? COLORS.headerBg : COLORS.white }}
      colSpan={colSpan}
      rowSpan={rowSpan}
    >
      {children}
    </td>
  );
}

// Shared sub-components for slides
function SlideHeader({ report }: { report: Report }) {
  return (
    <table className="w-full border-collapse text-xs">
      <tbody>
        <tr>
          <Cell header className="w-[13%]">프로젝트명</Cell>
          <Cell className="w-[27%]">{report.team_name || '팀명'}</Cell>
          <Cell header className="w-[16%]">보고일자</Cell>
          <Cell className="w-[13%]">{formatDateShort(report.report_date) || 'YYYY.MM.DD'}</Cell>
          <Cell header className="w-[16%]">작성자</Cell>
          <Cell className="w-[15%]">{report.author_name || '이름'}</Cell>
        </tr>
      </tbody>
    </table>
  );
}

function SectionTitle({ title }: { title: string }) {
  return (
    <table className="w-full border-collapse text-xs">
      <tbody>
        <tr>
          <Cell header align="center">{title}</Cell>
        </tr>
      </tbody>
    </table>
  );
}

function FooterTables({ issues, notes, showProgress }: { issues?: string; notes?: string; showProgress?: boolean }) {
  const headerW = showProgress ? 'w-[23%]' : 'w-[26%]';
  return (
    <>
      <table className="w-full border-collapse text-xs -mt-px">
        <tbody>
          <tr>
            <Cell header className={headerW}>이슈/위험 사항</Cell>
            <Cell colSpan={showProgress ? 3 : 2} className="whitespace-pre-line">{issues}</Cell>
          </tr>
        </tbody>
      </table>
      <table className="w-full border-collapse text-xs -mt-px">
        <tbody>
          <tr>
            <Cell header className={headerW}>특이 사항</Cell>
            <Cell colSpan={showProgress ? 3 : 2} className="whitespace-pre-line">{notes}</Cell>
          </tr>
        </tbody>
      </table>
    </>
  );
}

function getPreviewTextSize(taskCount: number): string {
  if (taskCount <= 5) return 'text-xs';
  if (taskCount <= 8) return 'text-[11px]';
  return 'text-[10px]';
}

function getFullDetails(t: { details?: string; description?: string }): string {
  let text = t.details || '-';
  if (t.description) text += '\n' + t.description;
  return text;
}

// Group tasks by title (preserving order of first appearance)
interface GroupedTask {
  title: string;
  items: Task[];
}

function groupTasksByTitle(tasks: Task[]): GroupedTask[] {
  const groups: GroupedTask[] = [];
  const indexMap = new Map<string, number>();
  for (const task of tasks) {
    const key = task.title.trim();
    if (indexMap.has(key)) {
      groups[indexMap.get(key)!].items.push(task);
    } else {
      indexMap.set(key, groups.length);
      groups.push({ title: key, items: [task] });
    }
  }
  return groups;
}

function TaskRows({ tasks, maxItems, dateRange, showProgress }: {
  tasks: Report['this_week'];
  maxItems: number;
  dateRange: string;
  showProgress?: boolean;
}) {
  const groups = groupTasksByTitle(tasks);
  const textSize = getPreviewTextSize(groups.length);
  return (
    <table className={`w-full border-collapse ${textSize} flex-1`}>
      <tbody>
        <tr className="h-7">
          <Cell header compact className={showProgress ? 'w-[23%]' : 'w-[26%]'}>계획업무 ({dateRange})</Cell>
          <Cell header compact align="center" className="w-[53%]">{showProgress ? '진행 사항' : ''}</Cell>
          <Cell header compact align="center" className={showProgress ? 'w-[11%]' : 'w-[21%]'}>{showProgress ? '완료일' : '완료예정일'}</Cell>
          {showProgress && (
            <Cell header compact align="center" className="w-[13%]">실적(%)</Cell>
          )}
        </tr>
        <tr>
          {/* Title column */}
          <Cell valign="top" className="whitespace-pre-line">
            {groups.slice(0, maxItems).map((g, gi) => (
              <div key={gi} className="mb-2">
                <div>{gi + 1}. {g.title}</div>
                {/* Spacer lines for all sub-items' detail lines (except first line of first item) */}
                {g.items.map((t, ii) => {
                  const lines = getFullDetails(t).split('\n');
                  const spacers = ii === 0 ? lines.length - 1 : lines.length;
                  return Array.from({ length: spacers }, (_, idx) => (
                    <div key={`${ii}-${idx}`}>&nbsp;</div>
                  ));
                })}
              </div>
            ))}
            {groups.length > maxItems && (
              <div className="text-gray-500">+{groups.length - maxItems}개 더</div>
            )}
          </Cell>
          {/* Details column */}
          <Cell valign="top" className="whitespace-pre-line">
            {groups.slice(0, maxItems).map((g, gi) => (
              <div key={gi} className="mb-2">
                {g.items.map((t, ii) => (
                  <div key={ii} className="whitespace-pre-line">{getFullDetails(t)}</div>
                ))}
              </div>
            ))}
          </Cell>
          {/* Date column */}
          <Cell valign="top" align="center">
            {groups.slice(0, maxItems).map((g, gi) => (
              <div key={gi} className="mb-2">
                {g.items.map((t, ii) => {
                  const lines = getFullDetails(t).split('\n');
                  return (
                    <div key={ii}>
                      <div>{t.due_date || '-'}</div>
                      {lines.slice(1).map((_, idx) => (
                        <div key={idx}>&nbsp;</div>
                      ))}
                    </div>
                  );
                })}
              </div>
            ))}
          </Cell>
          {/* Progress column */}
          {showProgress && (
            <Cell valign="top" align="center">
              {groups.slice(0, maxItems).map((g, gi) => (
                <div key={gi} className="mb-2">
                  {g.items.map((t, ii) => {
                    const lines = getFullDetails(t).split('\n');
                    return (
                      <div key={ii}>
                        <div>{t.progress}%</div>
                        {lines.slice(1).map((_, idx) => (
                          <div key={idx}>&nbsp;</div>
                        ))}
                      </div>
                    );
                  })}
                </div>
              ))}
            </Cell>
          )}
        </tr>
      </tbody>
    </table>
  );
}

export default function PptPreview({ report, style = defaultTemplateStyle }: PptPreviewProps) {
  const slides = useMemo(() => {
    const slideList: { title: string; content: React.ReactNode }[] = [];
    const dateRange = getWeekRange(report.report_date);
    const showProgress = style.showProgressBar;

    // Slide 1: 금주실적
    if (report.this_week.length > 0 || report.issues || report.notes) {
      slideList.push({
        title: '금주실적',
        content: (
          <div className="h-full p-1 flex flex-col gap-0">
            <SlideHeader report={report} />
            <SectionTitle title="금주실적" />
            <TaskRows tasks={report.this_week} maxItems={8} dateRange={dateRange} showProgress={showProgress} />
            <FooterTables issues={report.issues} notes={report.notes} showProgress={showProgress} />
          </div>
        ),
      });
    }

    // Slide 2: 차주계획
    if (report.next_week.length > 0 || report.next_issues || report.next_notes) {
      slideList.push({
        title: '차주계획',
        content: (
          <div className="h-full p-1 flex flex-col gap-0">
            <SlideHeader report={report} />
            <SectionTitle title="차주계획" />
            <TaskRows tasks={report.next_week} maxItems={8} dateRange={dateRange} />
            <FooterTables issues={report.next_issues} notes={report.next_notes} showProgress={false} />
          </div>
        ),
      });
    }

    return slideList;
  }, [report, style]);

  if (slides.length === 0) {
    return (
      <div className="text-center text-gray-500 py-8">
        금주실적을 입력하면 미리보기가 표시됩니다.
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <h3 className="text-lg font-semibold text-gray-800">PPT 미리보기</h3>
      <div className="space-y-6">
        {slides.map((slide, idx) => (
          <div
            key={idx}
            className="aspect-[4/3] bg-white rounded-lg shadow-lg border-2 border-gray-200 overflow-hidden max-w-4xl mx-auto"
          >
            <div className="h-full flex flex-col">
              <div className="text-sm text-gray-600 font-medium px-4 py-2 bg-gray-50 border-b">
                슬라이드 {idx + 1}: {slide.title}
              </div>
              <div className="flex-1 overflow-hidden">{slide.content}</div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
