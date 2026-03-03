import { useMemo } from 'react';
import { ConsolidatedReport, MemberReportData, Task } from '../types';
import { formatDateShort, getWeekRange, getNextWeekRange } from '../utils/date';

interface ConsolidatedPptPreviewProps {
  data: ConsolidatedReport;
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
    .filter(m => m.report && m.report[field])
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
    const memberTag = item.memberName ? ` ( ${item.memberName} )` : '';
    let detail = item.task.details || '-';
    if (item.task.description) detail += '\n' + item.task.description;
    const lines = detail.split('\n');
    lines.forEach((line, li) => {
      rows.push({
        body: li === 0 ? `- ${line}${memberTag}` : line,
        date: li === 0 ? formatDateMMDD(item.task.due_date) : '',
        progress: li === 0 ? `${item.task.progress}%` : '',
        bold: false,
      });
    });
  }
  return rows;
}

// Build left/right preview rows aligned by project+client
function buildAlignedPreviewRows(
  leftGroups: ConsolidatedTask[],
  rightGroups: ConsolidatedTask[]
): { leftRows: PreviewRow[]; rightRows: PreviewRow[] } {
  const empty: PreviewRow = { body: '', date: '', progress: '', bold: false };
  const leftRows: PreviewRow[] = [];
  const rightRows: PreviewRow[] = [];

  // Merge project titles preserving insertion order
  const allTitles: string[] = [];
  const seen = new Set<string>();
  for (const g of leftGroups) { if (!seen.has(g.title)) { seen.add(g.title); allTitles.push(g.title); } }
  for (const g of rightGroups) { if (!seen.has(g.title)) { seen.add(g.title); allTitles.push(g.title); } }

  const leftMap = new Map(leftGroups.map(g => [g.title, g]));
  const rightMap = new Map(rightGroups.map(g => [g.title, g]));

  for (const title of allTitles) {
    const leftGroup = leftMap.get(title);
    const rightGroup = rightMap.get(title);

    leftRows.push({ body: `[${title}]`, date: '', progress: '', bold: true, isProjectHeader: true });
    rightRows.push({ body: `[${title}]`, date: '', progress: '', bold: true, isProjectHeader: true });

    const leftClients = leftGroup ? subGroupByClient(leftGroup.items) : new Map<string, ConsolidatedTask['items']>();
    const rightClients = rightGroup ? subGroupByClient(rightGroup.items) : new Map<string, ConsolidatedTask['items']>();
    const allClients: string[] = [];
    const clientSeen = new Set<string>();
    for (const k of leftClients.keys()) { if (!clientSeen.has(k)) { clientSeen.add(k); allClients.push(k); } }
    for (const k of rightClients.keys()) { if (!clientSeen.has(k)) { clientSeen.add(k); allClients.push(k); } }

    for (const client of allClients) {
      const lItems = leftClients.get(client) || [];
      const rItems = rightClients.get(client) || [];

      if (client) {
        leftRows.push({ body: `• ${client}`, date: '', progress: '', bold: false, isProjectHeader: false });
        rightRows.push({ body: `• ${client}`, date: '', progress: '', bold: false, isProjectHeader: false });
      }

      const lDetailRows = buildItemPreviewRows(lItems);
      const rDetailRows = buildItemPreviewRows(rItems);
      const maxLen = Math.max(lDetailRows.length, rDetailRows.length);

      for (let i = 0; i < maxLen; i++) {
        leftRows.push(i < lDetailRows.length ? lDetailRows[i] : empty);
        rightRows.push(i < rDetailRows.length ? rDetailRows[i] : empty);
      }
    }
  }

  return { leftRows, rightRows };
}

// Same pagination as pptGenerator with balanced distribution
function calcPagination(leftCount: number, rightCount: number, bodyH: number) {
  const maxLines = Math.max(leftCount, rightCount, 1);
  const getRowH = (fs: number) => fs >= 9 ? 0.21 : fs >= 8 ? 0.19 : 0.17;

  for (const fs of [9, 8, 7]) {
    const rh = getRowH(fs);
    const perPage = Math.floor(bodyH / rh);
    if (maxLines <= perPage) return { fontSize: fs, pages: 1, linesPerPage: perPage };
  }
  const rh = getRowH(7);
  const perPage = Math.floor(bodyH / rh);
  const pages = Math.ceil(maxLines / perPage);
  const balancedPerPage = Math.ceil(maxLines / pages);
  return { fontSize: 7, pages, linesPerPage: balancedPerPage };
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

export default function ConsolidatedPptPreview({ data }: ConsolidatedPptPreviewProps) {
  const slides = useMemo(() => {
    const thisWeekGroups = groupTasksByTitle(data.members, 'this_week');
    const nextWeekGroups = groupTasksByTitle(data.members, 'next_week');
    const { leftRows, rightRows } = buildAlignedPreviewRows(thisWeekGroups, nextWeekGroups);

    const issues = mergeText(data.members, 'issues');
    const notes = mergeText(data.members, 'notes');
    const nextIssues = mergeText(data.members, 'next_issues');
    const nextNotes = mergeText(data.members, 'next_notes');
    const dateRange = getWeekRange(data.report_date);
    const nextDateRange = getNextWeekRange(data.report_date);

    const issueLineCount = (issues || nextIssues || '-').split('\n').length;
    const noteLineCount = (notes || nextNotes || '-').split('\n').length;
    const issueH = Math.max(0.28, Math.min(0.55, issueLineCount * 0.14 + 0.05));
    const noteH = Math.max(0.28, Math.min(0.55, noteLineCount * 0.14 + 0.05));
    const bodyH = 6.9 - (0.35 + 0.30 + 0.40 + issueH + noteH);

    const { pages, linesPerPage } = calcPagination(leftRows.length, leftRows.length, bodyH);

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
                    {data.members.filter(m => m.report).map(m => `${m.user_name}(${m.role_code})`).join(', ')}
                    {pageLabel}
                  </DCell>
                </tr>
              </tbody>
            </table>

            {/* Unified body table: left (3 cols) + right (2 cols) */}
            <table className="w-full border-collapse -mt-px flex-1">
              <colgroup>
                <col style={{ width: '30%' }} />
                <col style={{ width: '10%' }} />
                <col style={{ width: '10%' }} />
                <col style={{ width: '35%' }} />
                <col style={{ width: '15%' }} />
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

            {/* Footer: issues + notes (last page only) */}
            {isLastPage && (
              <table className="w-full border-collapse -mt-px">
                <tbody>
                  <tr>
                    <HCell className="w-[15%]">이슈/위험사항</HCell>
                    <DCell className="whitespace-pre-line">{issues || nextIssues || '-'}</DCell>
                  </tr>
                  <tr>
                    <HCell className="w-[15%]">특이사항</HCell>
                    <DCell className="whitespace-pre-line">{notes || nextNotes || '-'}</DCell>
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
