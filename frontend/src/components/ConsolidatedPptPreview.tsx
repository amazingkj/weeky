import { useMemo } from 'react';
import { ConsolidatedReport, MemberReportData, Task } from '../types';
import { formatDateShort, getWeekRange, getNextWeekRange } from '../utils/date';

interface ConsolidatedPptPreviewProps {
  data: ConsolidatedReport;
  leaderName?: string;
}

interface ConsolidatedTask {
  title: string;
  items: { task: Task; memberName: string; roleCode: string }[];
}

interface PreviewRow {
  body: string;
  date: string;
  progress: string;
  bold: boolean;
  isProjectHeader?: boolean;
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
        groups[indexMap.get(key)!].items.push({ task: t, memberName: m.user_name, roleCode: m.role_code });
      } else {
        indexMap.set(key, groups.length);
        groups.push({ title: key, items: [{ task: t, memberName: m.user_name, roleCode: m.role_code }] });
      }
    }
  }
  return groups;
}

function mergeText(members: MemberReportData[], field: 'issues' | 'notes' | 'next_issues' | 'next_notes'): string {
  return members
    .filter(m => m.report && m.report[field]?.trim())
    .map(m => m.user_name ? `[${m.user_name}] ${m.report![field]}` : m.report![field])
    .join('\n');
}

function formatDateMMDD(dateStr: string): string {
  if (!dateStr) return '-';
  const parts = dateStr.split('-');
  if (parts.length >= 3) return `${parts[1]}/${parts[2]}`;
  return dateStr;
}

function subGroupByClient(items: ConsolidatedTask['items']): Map<string, ConsolidatedTask['items']> {
  const map = new Map<string, ConsolidatedTask['items']>();
  for (const item of items) {
    const key = (item.task.client || '').trim();
    if (!map.has(key)) map.set(key, []);
    map.get(key)!.push(item);
  }
  return map;
}

// Build detail rows for a list of items (without project/client headers)
function buildItemPreviewRows(items: ConsolidatedTask['items']): PreviewRow[] {
  const rows: PreviewRow[] = [];
  for (const item of items) {
    const memberTag = item.memberName ? ` (${item.memberName})` : '';
    const detail = item.task.details || '-';
    rows.push({
      body: `- ${detail}${memberTag}`,
      date: formatDateMMDD(item.task.due_date),
      progress: `${item.task.progress}%`,
      bold: false,
    });
  }
  return rows;
}

// Build rows for one side independently (project → client → detail)
function buildSidePreviewRows(groups: ConsolidatedTask[]): PreviewRow[] {
  const rows: PreviewRow[] = [];
  for (const g of groups) {
    rows.push({ body: `[${g.title}]`, date: '', progress: '', bold: true, isProjectHeader: true });
    const clientMap = subGroupByClient(g.items);
    for (const [client, items] of clientMap) {
      if (client) {
        rows.push({ body: `• ${client}`, date: '', progress: '', bold: false, isProjectHeader: false });
      }
      rows.push(...buildItemPreviewRows(items));
    }
  }
  return rows;
}

// Build left/right rows independently, pad shorter side at the end
function buildAlignedPreviewRows(
  leftGroups: ConsolidatedTask[],
  rightGroups: ConsolidatedTask[]
): { leftRows: PreviewRow[]; rightRows: PreviewRow[] } {
  const empty: PreviewRow = { body: '', date: '', progress: '', bold: false };
  const leftRows = buildSidePreviewRows(leftGroups);
  const rightRows = buildSidePreviewRows(rightGroups);

  while (leftRows.length < rightRows.length) leftRows.push(empty);
  while (rightRows.length < leftRows.length) rightRows.push(empty);

  return { leftRows, rightRows };
}

// Same pagination as pptGenerator — breakLine 실측 줄 높이 기반
function calcPagination(leftCount: number, rightCount: number, bodyH: number) {
  const maxLines = Math.max(leftCount, rightCount, 1);
  const getRowH = (fs: number) => (fs + 2) / 72;

  for (const fs of [9, 8, 7, 6]) {
    const rh = getRowH(fs);
    const perPage = Math.floor(bodyH / rh);
    if (maxLines <= perPage) return { fontSize: fs, pages: 1, linesPerPage: perPage };
  }

  const minRh = getRowH(6);
  const minPages = Math.ceil(maxLines / Math.floor(bodyH / minRh));

  for (const fs of [9, 8, 7, 6]) {
    const rh = getRowH(fs);
    const perPage = Math.floor(bodyH / rh);
    if (Math.ceil(maxLines / perPage) <= minPages) {
      return { fontSize: fs, pages: minPages, linesPerPage: perPage };
    }
  }

  const perPage = Math.floor(bodyH / minRh);
  return { fontSize: 6, pages: minPages, linesPerPage: perPage };
}

const emptyRow: PreviewRow = { body: '', date: '', progress: '', bold: false };

// Header cell
function HCell({ children, className = '', colSpan = 1, align = 'left' }: {
  children?: React.ReactNode; className?: string; colSpan?: number; align?: string;
}) {
  const alignCls = align === 'center' ? 'text-center' : 'text-left';
  return (
    <td className={`border border-black px-1.5 py-0.5 text-[10px] font-bold bg-[#F5F5F5] ${alignCls} ${className}`} colSpan={colSpan}>
      {children}
    </td>
  );
}

function DCell({ children, className = '', colSpan = 1 }: {
  children?: React.ReactNode; className?: string; colSpan?: number;
}) {
  return (
    <td className={`border border-black px-1.5 py-0.5 text-[10px] ${className}`} colSpan={colSpan}>
      {children}
    </td>
  );
}

export default function ConsolidatedPptPreview({ data, leaderName }: ConsolidatedPptPreviewProps) {
  const slides = useMemo<{ title: string; content: React.ReactNode }[]>(() => {
    const thisWeekGroups = groupTasksByTitle(data.members, 'this_week');
    const nextWeekGroups = groupTasksByTitle(data.members, 'next_week');
    const { leftRows, rightRows } = buildAlignedPreviewRows(thisWeekGroups, nextWeekGroups);

    const issuesLeft = mergeText(data.members, 'issues');
    const issuesRight = mergeText(data.members, 'next_issues');
    const notesLeft = mergeText(data.members, 'notes');
    const notesRight = mergeText(data.members, 'next_notes');
    const dateRange = getWeekRange(data.report_date);
    const nextDateRange = getNextWeekRange(data.report_date);

    // footer 높이 계산 — pptGenerator와 동일 로직
    const ISSUE_CELL_W = 9.4 - 1.5;
    const CHARS_PER_LINE = Math.floor(ISSUE_CELL_W * 14);
    const LINE_H = 0.22;
    function calcWrappedLineCount(text: string): number {
      const lines = (text || '-').split('\n');
      const charsPerLine = Math.floor(CHARS_PER_LINE * (8 / 8));
      return lines.reduce((sum, line) => sum + Math.max(1, Math.ceil(line.length / charsPerLine)), 0);
    }
    const issueLineCount = Math.max(calcWrappedLineCount(issuesLeft), calcWrappedLineCount(issuesRight));
    const noteLineCount = Math.max(calcWrappedLineCount(notesLeft), calcWrappedLineCount(notesRight));
    const issueH = Math.max(0.50, Math.min(1.5, issueLineCount * LINE_H + 0.15));
    const noteH = Math.max(0.50, Math.min(1.5, noteLineCount * LINE_H + 0.15));
    const bodyH = 6.9 - (0.35 + 0.30 + 0.40 + issueH + noteH);

    const { pages, linesPerPage } = calcPagination(leftRows.length, rightRows.length, bodyH);

    const slideList: { title: string; content: React.ReactNode }[] = [];

    for (let p = 0; p < pages; p++) {
      const startIdx = p * linesPerPage;
      const endIdx = startIdx + linesPerPage;
      const pageLeft = leftRows.slice(startIdx, endIdx);
      const pageRight = rightRows.slice(startIdx, endIdx);
      const rowCount = Math.max(pageLeft.length, pageRight.length, 1);
      while (pageLeft.length < rowCount) pageLeft.push(emptyRow);
      while (pageRight.length < rowCount) pageRight.push(emptyRow);

      const pageLabel = pages > 1 ? ` (${p + 1}/${pages})` : '';
      const isLastPage = p === pages - 1;

      slideList.push({
        title: `취합 보고서${pageLabel}`,
        content: (
          <div className="h-full p-1 flex flex-col gap-0 text-[10px]">
            {/* Header */}
            <table className="w-full border-collapse">
              <tbody>
                <tr>
                  <HCell className="w-[13%]" align="center">프로젝트명</HCell>
                  <DCell className="w-[20%] text-center">{data.team.name}</DCell>
                  <HCell className="w-[10%]" align="center">보고일자</HCell>
                  <DCell className="w-[12%] text-center">{formatDateShort(data.report_date)}</DCell>
                  <HCell className="w-[10%]" align="center">작성자</HCell>
                  <DCell className="w-[35%] text-center">
                    {leaderName || data.team.name}
                    {pageLabel}
                  </DCell>
                </tr>
              </tbody>
            </table>

            {/* Unified body table: left (3 cols) + right (2 cols) */}
            <table className="w-full border-collapse -mt-px flex-1">
              <colgroup>
                <col style={{ width: '32%' }} />
                <col style={{ width: '8.5%' }} />
                <col style={{ width: '8.5%' }} />
                <col style={{ width: '40.5%' }} />
                <col style={{ width: '10.5%' }} />
              </colgroup>
              <tbody>
                {/* Section headers */}
                <tr>
                  <HCell colSpan={3} align="center">금주실적</HCell>
                  <HCell colSpan={2} align="center">차주계획</HCell>
                </tr>
                {/* Column headers */}
                <tr>
                  <HCell>계획업무 ({dateRange})</HCell>
                  <HCell align="center">완료일</HCell>
                  <HCell align="center">실적(%)</HCell>
                  <HCell>계획업무 ({nextDateRange})</HCell>
                  <HCell align="center">완료예정일</HCell>
                </tr>
                {/* Body rows - no horizontal borders between rows */}
                {Array.from({ length: rowCount }).map((_, i) => {
                  const lr = pageLeft[i] || emptyRow;
                  const rr = pageRight[i] || emptyRow;
                  const isFirst = i === 0;
                  const isLast = i === rowCount - 1;
                  const brdCls = `border-l border-r border-black ${isFirst ? 'border-t ' : ''}${isLast ? 'border-b ' : ''}`;
                  return (
                    <tr key={i} className="leading-tight">
                      <td className={`${brdCls} px-1 py-0 align-top`}>
                        {lr.bold
                          ? <span className="font-medium">{lr.body}</span>
                          : <span className="pl-2">{lr.body}</span>
                        }
                      </td>
                      <td className={`${brdCls} px-1 py-0 text-center align-top`}>{lr.date}</td>
                      <td className={`${brdCls} px-1 py-0 text-center align-top`}>{lr.progress}</td>
                      <td className={`${brdCls} px-1 py-0 align-top`}>
                        {rr.bold
                          ? <span className="font-medium">{rr.body}</span>
                          : <span className="pl-2">{rr.body}</span>
                        }
                      </td>
                      <td className={`${brdCls} px-1 py-0 text-center align-top`}>{rr.date}</td>
                    </tr>
                  );
                })}
              </tbody>
            </table>

            {/* Footer: issues + notes (last page only, 3-column: header | 금주 | 차주) */}
            {isLastPage && (
              <table className="w-full border-collapse -mt-px">
                <colgroup>
                  <col style={{ width: '14%' }} />
                  <col style={{ width: '43%' }} />
                  <col style={{ width: '43%' }} />
                </colgroup>
                <tbody>
                  <tr>
                    <HCell>이슈/위험사항</HCell>
                    <DCell className="whitespace-pre-line">{issuesLeft || '-'}</DCell>
                    <DCell className="whitespace-pre-line">{issuesRight || '-'}</DCell>
                  </tr>
                  <tr>
                    <HCell>특이사항</HCell>
                    <DCell className="whitespace-pre-line">{notesLeft || '-'}</DCell>
                    <DCell className="whitespace-pre-line">{notesRight || '-'}</DCell>
                  </tr>
                </tbody>
              </table>
            )}
          </div>
        ),
      });
    }

    return slideList;
  }, [data]);

  return (
    <div className="space-y-4">
      <h3 className="text-sm font-semibold text-neutral-800">
        취합 PPT 미리보기
        {slides.length > 1 && (
          <span className="ml-2 text-xs font-normal text-neutral-500">{slides.length}페이지</span>
        )}
      </h3>
      <div className="space-y-6">
        {slides.map((slide, idx) => (
          <div key={idx}
            className="aspect-[4/3] bg-white rounded-lg shadow-lg border-2 border-gray-200 overflow-hidden max-w-5xl mx-auto">
            <div className="h-full flex flex-col">
              <div className="text-xs text-gray-600 font-medium px-4 py-2 bg-gray-50 border-b flex items-center justify-between">
                <span>슬라이드 {idx + 1}: {slide.title}</span>
                {slides.length > 1 && (
                  <span className="text-[10px] text-neutral-400">{idx + 1} / {slides.length}</span>
                )}
              </div>
              <div className="flex-1 overflow-auto">{slide.content}</div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
