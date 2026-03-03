import PptxGenJS from 'pptxgenjs';
import { Report, TemplateStyle, Task, ConsolidatedReport, MemberReportData, defaultTemplateStyle } from '../types';
import { formatDateShort, getWeekRange, getNextWeekRange } from './date';

// Cross-browser file download (Safari compatible)
function downloadBlob(blob: Blob, filename: string): void {
  // Safari detection
  const isSafari = /^((?!chrome|android).)*safari/i.test(navigator.userAgent);
  const url = URL.createObjectURL(blob);

  if (isSafari) {
    // Safari: window.open works more reliably than anchor click for blob downloads
    const newWindow = window.open(url, '_blank');
    if (!newWindow) {
      // Popup blocked — fall back to location assign
      window.location.href = url;
    }
    setTimeout(() => URL.revokeObjectURL(url), 30000);
  } else {
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    a.style.display = 'none';
    document.body.appendChild(a);
    a.click();
    setTimeout(() => {
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    }, 200);
  }
}

// Template colors
const COLORS = {
  headerBg: 'E6E6E6',
  white: 'FFFFFF',
  black: '000000',
  border: '000000',
};

// Font settings
const FONT = {
  face: '맑은 고딕',
  size: 10,
};

// Layout constants (4:3 slide = 10 x 7.5 inch)
const LAYOUT = {
  x: 0.3,
  y: 0.3,
  w: 9.4,
  h: 6.9,
};

const ROW_H = {
  header: 0.35,
  section: 0.30,
  colHeader: 0.40,
  body: 5.00,
  issue: 0.40,
  note: 0.45,
};

const HEADER_COL_W = [1.3, 2.2, 1.1, 1.5, 1.1, 2.2];

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

// Get full detail text for a single task (details + description)
function getTaskDetailText(t: Task): string {
  let text = t.details || '-';
  if (t.description) text += '\n' + t.description;
  return text;
}

// Calculate body font size based on total content lines
function getBodyFontSize(tasks: Task[]): number {
  const groups = groupTasksByTitle(tasks);
  let totalLines = 0;
  for (const g of groups) {
    for (const t of g.items) {
      totalLines += getTaskDetailText(t).split('\n').length;
    }
    totalLines += 1; // spacing between groups
  }
  if (totalLines <= 18) return 10;
  if (totalLines <= 25) return 9;
  if (totalLines <= 35) return 8;
  return 7;
}

interface TaskSlideConfig {
  sectionTitle: string;
  dateRange: string;
  tasks: Task[];
  showProgress: boolean;
  issuesText: string;
  notesText: string;
}

export async function generatePPT(report: Report, style: TemplateStyle = defaultTemplateStyle): Promise<void> {
  const pptx = new PptxGenJS();

  pptx.author = report.author_name;
  pptx.title = `주간업무보고 - ${report.author_name}`;
  pptx.subject = '주간업무보고';
  pptx.layout = 'LAYOUT_4x3';

  createTaskSlide(pptx, report, {
    sectionTitle: '금주실적',
    dateRange: getWeekRange(report.report_date),
    tasks: report.this_week,
    showProgress: style.showProgressBar,
    issuesText: report.issues || '',
    notesText: report.notes || '',
  });

  // 금주실적 중 진척률 100% 미만 → 차주계획에 자동 복사
  const nextWeekTasks = [...report.next_week];
  for (const task of report.this_week) {
    if (task.progress < 100) {
      const alreadyExists = nextWeekTasks.some(t => t.title.trim() === task.title.trim());
      if (!alreadyExists) {
        nextWeekTasks.push({ ...task, progress: 0 });
      }
    }
  }

  createTaskSlide(pptx, report, {
    sectionTitle: '차주계획',
    dateRange: getNextWeekRange(report.report_date),
    tasks: nextWeekTasks,
    showProgress: false,
    issuesText: report.next_issues || '',
    notesText: report.next_notes || '',
  });

  const filename = generateFilename(report);
  const blob = await pptx.write({ outputType: 'blob' }) as Blob;
  downloadBlob(blob, filename);
}

function createTaskSlide(pptx: PptxGenJS, report: Report, config: TaskSlideConfig): void {
  const slide = pptx.addSlide();
  let currentY = LAYOUT.y;

  // Row 1: Header (프로젝트명, 보고일자, 작성자)
  slide.addTable(
    [[
      { text: '프로젝트명', options: { fill: { color: COLORS.headerBg }, bold: true, align: 'center' } },
      { text: report.team_name, options: { align: 'center' } },
      { text: '보고일자', options: { fill: { color: COLORS.headerBg }, bold: true, align: 'center' } },
      { text: formatDateShort(report.report_date), options: { align: 'center' } },
      { text: '작성자', options: { fill: { color: COLORS.headerBg }, bold: true, align: 'center' } },
      { text: report.author_name, options: { align: 'center' } },
    ]],
    {
      x: LAYOUT.x, y: currentY, w: LAYOUT.w, h: ROW_H.header,
      colW: HEADER_COL_W, rowH: [ROW_H.header],
      border: { type: 'solid', color: COLORS.border, pt: 0.5 },
      fontFace: FONT.face, fontSize: FONT.size, valign: 'middle',
    }
  );
  currentY += ROW_H.header;

  // Row 2: Section title
  slide.addTable(
    [[
      { text: config.sectionTitle, options: { fill: { color: COLORS.headerBg }, bold: true, align: 'center' } },
    ]],
    {
      x: LAYOUT.x, y: currentY, w: LAYOUT.w, h: ROW_H.section,
      colW: [LAYOUT.w], rowH: [ROW_H.section],
      border: { type: 'solid', color: COLORS.border, pt: 0.5 },
      fontFace: FONT.face, fontSize: FONT.size, valign: 'middle',
    }
  );
  currentY += ROW_H.section;

  // Row 3-4: Body (column headers + task content) — grouped by title
  const bodyFontSize = getBodyFontSize(config.tasks);
  const groups = groupTasksByTitle(config.tasks);

  const groupData = groups.map((g, gi) => {
    const titleParts: string[] = [];
    const detailParts: string[] = [];
    const dateParts: string[] = [];
    const progressParts: string[] = [];

    g.items.forEach((t, ii) => {
      const detailText = getTaskDetailText(t);
      const detailLines = detailText.split('\n');
      const lineCount = detailLines.length;

      // Title only on the first line of the first item in the group
      if (ii === 0) {
        titleParts.push(`${gi + 1}. ${g.title}`);
      } else {
        titleParts.push('');
      }
      // Pad remaining lines for this item
      for (let l = 1; l < lineCount; l++) titleParts.push('');

      detailParts.push(...detailLines);

      dateParts.push(t.due_date || '-');
      for (let l = 1; l < lineCount; l++) dateParts.push('');

      progressParts.push(`${t.progress}%`);
      for (let l = 1; l < lineCount; l++) progressParts.push('');
    });

    return {
      titleLines: titleParts,
      detailLines: detailParts,
      dateLines: dateParts,
      progressLines: progressParts,
    };
  });

  const taskTitles = groupData.map(d => d.titleLines.join('\n')).join('\n\n');
  const taskDetails = groupData.map(d => d.detailLines.join('\n')).join('\n\n');
  const taskDates = groupData.map(d => d.dateLines.join('\n')).join('\n\n');
  const taskProgress = groupData.map(d => d.progressLines.join('\n')).join('\n\n');

  let bodyColW: number[];
  let bodyHeaderRow: PptxGenJS.TableCell[];
  let bodyContentRow: PptxGenJS.TableCell[];

  if (config.showProgress) {
    bodyColW = [2.2, 5.0, 1.0, 1.2];
    bodyHeaderRow = [
      { text: `계획업무\n(${config.dateRange})`, options: { fill: { color: COLORS.headerBg }, bold: true, valign: 'middle' } },
      { text: '진행 사항', options: { fill: { color: COLORS.headerBg }, bold: true, align: 'center', valign: 'middle' } },
      { text: '완료일', options: { fill: { color: COLORS.headerBg }, bold: true, align: 'center', valign: 'middle' } },
      { text: '실적(%)', options: { fill: { color: COLORS.headerBg }, bold: true, align: 'center', valign: 'middle' } },
    ];
    bodyContentRow = [
      { text: taskTitles, options: { valign: 'top' } },
      { text: taskDetails, options: { valign: 'top' } },
      { text: taskDates, options: { valign: 'top', align: 'center' } },
      { text: taskProgress, options: { valign: 'top', align: 'center' } },
    ];
  } else {
    bodyColW = [2.4, 5.0, 2.0];
    bodyHeaderRow = [
      { text: `계획업무\n(${config.dateRange})`, options: { fill: { color: COLORS.headerBg }, bold: true, valign: 'middle' } },
      { text: config.sectionTitle === '차주계획' ? '' : '진행 사항', options: { fill: { color: COLORS.headerBg }, bold: true, align: 'center', valign: 'middle' } },
      { text: config.sectionTitle === '차주계획' ? '완료\n예정일' : '완료일', options: { fill: { color: COLORS.headerBg }, bold: true, align: 'center', valign: 'middle' } },
    ];
    bodyContentRow = [
      { text: taskTitles, options: { valign: 'top' } },
      { text: taskDetails, options: { valign: 'top' } },
      { text: taskDates, options: { valign: 'top', align: 'center' } },
    ];
  }

  slide.addTable(
    [bodyHeaderRow, bodyContentRow],
    {
      x: LAYOUT.x, y: currentY, w: LAYOUT.w, h: ROW_H.colHeader + ROW_H.body,
      colW: bodyColW, rowH: [ROW_H.colHeader, ROW_H.body],
      border: { type: 'solid', color: COLORS.border, pt: 0.5 },
      fontFace: FONT.face, fontSize: bodyFontSize, valign: 'middle',
    }
  );
  currentY += ROW_H.colHeader + ROW_H.body;

  // Row 5: Issues (use same colW as body for aligned borders)
  const footerCols = config.showProgress ? 4 : 3;
  const issueRow: PptxGenJS.TableCell[] = [
    { text: '이슈/위험 사항', options: { fill: { color: COLORS.headerBg }, bold: true } },
    { text: config.issuesText, options: { colspan: footerCols - 1 } },
  ];

  slide.addTable(
    [issueRow],
    {
      x: LAYOUT.x, y: currentY, w: LAYOUT.w, h: ROW_H.issue,
      colW: bodyColW, rowH: [ROW_H.issue],
      border: { type: 'solid', color: COLORS.border, pt: 0.5 },
      fontFace: FONT.face, fontSize: FONT.size, valign: 'middle',
    }
  );
  currentY += ROW_H.issue;

  // Row 6: Notes (use same colW as body for aligned borders)
  const noteRow: PptxGenJS.TableCell[] = [
    { text: '특이 사항', options: { fill: { color: COLORS.headerBg }, bold: true } },
    { text: config.notesText, options: { colspan: footerCols - 1 } },
  ];

  slide.addTable(
    [noteRow],
    {
      x: LAYOUT.x, y: currentY, w: LAYOUT.w, h: ROW_H.note,
      colW: bodyColW, rowH: [ROW_H.note],
      border: { type: 'solid', color: COLORS.border, pt: 0.5 },
      fontFace: FONT.face, fontSize: FONT.size, valign: 'middle',
    }
  );
}

function generateFilename(report: Report): string {
  const date = report.report_date.replace(/-/g, '');
  return `${report.team_name}_${report.author_name}_주간보고_${date}.pptx`;
}

// ============ Consolidated PPT (Team) ============

interface ConsolidatedTaskItem {
  task: Task;
  memberName: string;
  roleCode: string;
}

interface ConsolidatedGroup {
  title: string;
  items: ConsolidatedTaskItem[];
}

function groupConsolidatedTasks(
  members: MemberReportData[],
  section: 'this_week' | 'next_week'
): ConsolidatedGroup[] {
  const groups: ConsolidatedGroup[] = [];
  const indexMap = new Map<string, number>();

  for (const m of members) {
    if (!m.report) continue;
    const tasks = section === 'this_week' ? m.report.this_week : m.report.next_week;
    for (const t of tasks) {
      const key = t.title.trim();
      const item: ConsolidatedTaskItem = {
        task: t,
        memberName: m.user_name,
        roleCode: m.role_code,
      };
      if (indexMap.has(key)) {
        groups[indexMap.get(key)!].items.push(item);
      } else {
        indexMap.set(key, groups.length);
        groups.push({ title: key, items: [item] });
      }
    }
  }
  return groups;
}

function mergeIssuesNotes(
  members: MemberReportData[],
  field: 'issues' | 'notes' | 'next_issues' | 'next_notes'
): string {
  return members
    .filter(m => m.report && m.report[field])
    .map(m => m.user_name ? `[${m.user_name}] ${m.report![field]}` : m.report![field])
    .join('\n');
}

// Row data for row-aligned body tables
interface BodyRow {
  body: string;
  date: string;
  progress: string;
  bold: boolean;
}

function formatDateShortMMDD(dateStr: string): string {
  if (!dateStr) return '-';
  const parts = dateStr.split('-');
  if (parts.length >= 3) return `${parts[1]}/${parts[2]}`;
  return dateStr;
}

function subGroupByClient(items: ConsolidatedTaskItem[]): Map<string, ConsolidatedTaskItem[]> {
  const map = new Map<string, ConsolidatedTaskItem[]>();
  for (const item of items) {
    const key = (item.task.client || '').trim();
    if (!map.has(key)) map.set(key, []);
    map.get(key)!.push(item);
  }
  return map;
}

// Build detail rows for a list of items (without project/client headers)
function buildItemRows(items: ConsolidatedTaskItem[], indent: string): BodyRow[] {
  const rows: BodyRow[] = [];
  for (const item of items) {
    const memberTag = item.memberName ? ` ( ${item.memberName} )` : '';
    let detail = item.task.details || '-';
    if (item.task.description) detail += '\n' + item.task.description;
    const lines = detail.split('\n');
    lines.forEach((line, li) => {
      rows.push({
        body: li === 0 ? `${indent}- ${line}${memberTag}` : `${indent}  ${line}`,
        date: li === 0 ? formatDateShortMMDD(item.task.due_date) : '',
        progress: li === 0 ? `${item.task.progress}%` : '',
        bold: false,
      });
    });
  }
  return rows;
}

// Build left/right rows aligned by project+client
function buildAlignedRows(
  leftGroups: ConsolidatedGroup[],
  rightGroups: ConsolidatedGroup[]
): { leftRows: BodyRow[]; rightRows: BodyRow[] } {
  const emptyRow: BodyRow = { body: '', date: '', progress: '', bold: false };
  const leftRows: BodyRow[] = [];
  const rightRows: BodyRow[] = [];

  // Merge project titles preserving insertion order (left first, then right-only)
  const allTitles: string[] = [];
  const seen = new Set<string>();
  for (const g of leftGroups) { if (!seen.has(g.title)) { seen.add(g.title); allTitles.push(g.title); } }
  for (const g of rightGroups) { if (!seen.has(g.title)) { seen.add(g.title); allTitles.push(g.title); } }

  const leftMap = new Map(leftGroups.map(g => [g.title, g]));
  const rightMap = new Map(rightGroups.map(g => [g.title, g]));

  for (const title of allTitles) {
    const leftGroup = leftMap.get(title);
    const rightGroup = rightMap.get(title);

    // Project header row on both sides
    leftRows.push({ body: `[${title}]`, date: '', progress: '', bold: true });
    rightRows.push({ body: `[${title}]`, date: '', progress: '', bold: true });

    // Merge clients from both sides
    const leftClients = leftGroup ? subGroupByClient(leftGroup.items) : new Map<string, ConsolidatedTaskItem[]>();
    const rightClients = rightGroup ? subGroupByClient(rightGroup.items) : new Map<string, ConsolidatedTaskItem[]>();
    const allClients: string[] = [];
    const clientSeen = new Set<string>();
    for (const k of leftClients.keys()) { if (!clientSeen.has(k)) { clientSeen.add(k); allClients.push(k); } }
    for (const k of rightClients.keys()) { if (!clientSeen.has(k)) { clientSeen.add(k); allClients.push(k); } }

    for (const client of allClients) {
      const lItems = leftClients.get(client) || [];
      const rItems = rightClients.get(client) || [];
      const indent = client ? '    ' : '  ';

      // Client header on both sides
      if (client) {
        leftRows.push({ body: `  • ${client}`, date: '', progress: '', bold: false });
        rightRows.push({ body: `  • ${client}`, date: '', progress: '', bold: false });
      }

      // Build detail rows for each side
      const lDetailRows = buildItemRows(lItems, indent);
      const rDetailRows = buildItemRows(rItems, indent);
      const maxLen = Math.max(lDetailRows.length, rDetailRows.length);

      for (let i = 0; i < maxLen; i++) {
        leftRows.push(i < lDetailRows.length ? lDetailRows[i] : emptyRow);
        rightRows.push(i < rDetailRows.length ? rDetailRows[i] : emptyRow);
      }
    }
  }

  return { leftRows, rightRows };
}

function getConsolidatedRowH(fontSize: number): number {
  if (fontSize >= 9) return 0.21;
  if (fontSize >= 8) return 0.19;
  return 0.17;
}

function calcPagination(leftRows: BodyRow[], rightRows: BodyRow[], bodyH: number) {
  const maxRows = Math.max(leftRows.length, rightRows.length, 1);
  for (const fs of [9, 8, 7]) {
    const rh = getConsolidatedRowH(fs);
    const perPage = Math.floor(bodyH / rh);
    if (maxRows <= perPage) return { fontSize: fs, rowH: rh, pages: 1, rowsPerPage: perPage };
  }
  const rh = getConsolidatedRowH(7);
  const perPage = Math.floor(bodyH / rh);
  const pages = Math.ceil(maxRows / perPage);
  // Balance rows evenly across pages instead of packing page 1
  const balancedPerPage = Math.ceil(maxRows / pages);
  return { fontSize: 7, rowH: rh, pages, rowsPerPage: balancedPerPage };
}

export async function generateConsolidatedPPT(
  data: ConsolidatedReport,
  _style: TemplateStyle = defaultTemplateStyle,
  leaderName?: string
): Promise<void> {
  const pptx = new PptxGenJS();
  pptx.author = leaderName || data.team.name;
  pptx.title = `${data.team.name} 주간업무보고`;
  pptx.subject = '주간업무보고';
  pptx.layout = 'LAYOUT_4x3';

  const thisWeekGroups = groupConsolidatedTasks(data.members, 'this_week');
  const nextWeekGroups = groupConsolidatedTasks(data.members, 'next_week');

  const issues = mergeIssuesNotes(data.members, 'issues') || mergeIssuesNotes(data.members, 'next_issues');
  const notes = mergeIssuesNotes(data.members, 'notes') || mergeIssuesNotes(data.members, 'next_notes');

  const dateRange = getWeekRange(data.report_date);
  const nextDateRange = getNextWeekRange(data.report_date);
  const displayAuthor = leaderName || data.team.name;
  const halfW = LAYOUT.w / 2;
  const leftColW = [2.8, 1.0, 0.9];
  const rightColW = [3.2, 1.5];

  // Build row-aligned data (left/right aligned by project+client)
  const { leftRows, rightRows } = buildAlignedRows(thisWeekGroups, nextWeekGroups);

  // Issues/notes auto-fit: smaller font + dynamic height
  const issueLineCount = (issues || '-').split('\n').length;
  const noteLineCount = (notes || '-').split('\n').length;
  const issueFontSize = issueLineCount > 2 ? 7 : 8;
  const noteFontSize = noteLineCount > 2 ? 7 : 8;
  const issueH = Math.max(0.28, Math.min(0.55, issueLineCount * 0.14 + 0.05));
  const noteH = Math.max(0.28, Math.min(0.55, noteLineCount * 0.14 + 0.05));

  // Calculate available body height (reserve space for footer on all pages for simplicity)
  const fixedH = ROW_H.header + ROW_H.section + ROW_H.colHeader + issueH + noteH;
  const bodyH = LAYOUT.h - fixedH;

  const { fontSize, rowH, pages, rowsPerPage } = calcPagination(leftRows, rightRows, bodyH);

  for (let page = 0; page < pages; page++) {
    const slide = pptx.addSlide();
    let curY = LAYOUT.y;

    // Row 1: Header
    const pageLabel = pages > 1 ? ` (${page + 1}/${pages})` : '';
    slide.addTable(
      [[
        { text: '프로젝트명', options: { fill: { color: COLORS.headerBg }, bold: true, align: 'center' } },
        { text: data.team.name, options: { align: 'center' } },
        { text: '보고일자', options: { fill: { color: COLORS.headerBg }, bold: true, align: 'center' } },
        { text: formatDateShort(data.report_date), options: { align: 'center' } },
        { text: '작성자', options: { fill: { color: COLORS.headerBg }, bold: true, align: 'center' } },
        { text: displayAuthor + pageLabel, options: { align: 'center' } },
      ]],
      {
        x: LAYOUT.x, y: curY, w: LAYOUT.w, h: ROW_H.header,
        colW: HEADER_COL_W, rowH: [ROW_H.header],
        border: { type: 'solid', color: COLORS.border, pt: 0.5 },
        fontFace: FONT.face, fontSize: FONT.size, valign: 'middle',
      }
    );
    curY += ROW_H.header;

    // Row 2: Section headers
    slide.addTable(
      [[
        { text: '금주실적', options: { fill: { color: COLORS.headerBg }, bold: true, align: 'center' } },
        { text: '차주계획', options: { fill: { color: COLORS.headerBg }, bold: true, align: 'center' } },
      ]],
      {
        x: LAYOUT.x, y: curY, w: LAYOUT.w, h: ROW_H.section,
        colW: [halfW, halfW], rowH: [ROW_H.section],
        border: { type: 'solid', color: COLORS.border, pt: 0.5 },
        fontFace: FONT.face, fontSize: FONT.size, valign: 'middle',
      }
    );
    curY += ROW_H.section;

    // Row 3: Column headers (unified 5-column table)
    const unifiedColW = [...leftColW, ...rightColW]; // [2.8, 1.0, 0.9, 3.2, 1.5] = 9.4
    slide.addTable(
      [[
        { text: `계획업무\n(${dateRange})`, options: { fill: { color: COLORS.headerBg }, bold: true, valign: 'middle' } },
        { text: '완료일', options: { fill: { color: COLORS.headerBg }, bold: true, align: 'center', valign: 'middle' } },
        { text: '실적(%)', options: { fill: { color: COLORS.headerBg }, bold: true, align: 'center', valign: 'middle' } },
        { text: `계획업무\n(${nextDateRange})`, options: { fill: { color: COLORS.headerBg }, bold: true, valign: 'middle' } },
        { text: '완료\n예정일', options: { fill: { color: COLORS.headerBg }, bold: true, align: 'center', valign: 'middle' } },
      ]],
      {
        x: LAYOUT.x, y: curY, w: LAYOUT.w, h: ROW_H.colHeader,
        colW: unifiedColW, rowH: [ROW_H.colHeader],
        border: { type: 'solid', color: COLORS.border, pt: 0.5 },
        fontFace: FONT.face, fontSize, valign: 'middle',
      }
    );
    curY += ROW_H.colHeader;

    // Body rows for this page
    const isLastPage = page === pages - 1;
    const startIdx = page * rowsPerPage;
    const maxSide = Math.max(leftRows.length, rightRows.length, 1);
    const endIdx = Math.min(startIdx + rowsPerPage, maxSide);
    const contentRowCount = Math.max(endIdx - startIdx, 1);
    const emptyRow = { body: '', date: '', progress: '', bold: false };

    const pLeft = leftRows.slice(startIdx, endIdx);
    const pRight = rightRows.slice(startIdx, endIdx);
    while (pLeft.length < contentRowCount) pLeft.push(emptyRow);
    while (pRight.length < contentRowCount) pRight.push(emptyRow);

    // Non-last pages: filler row to fill full page (no issues/notes on these pages)
    // Last page: no filler, issues/notes go right after body
    const headerFixedH = ROW_H.header + ROW_H.section + ROW_H.colHeader;
    const fullPageBodyH = LAYOUT.h - headerFixedH;
    const contentH = contentRowCount * rowH;
    const rowHeights: number[] = Array(contentRowCount).fill(rowH);

    let hasFiller = false;
    if (!isLastPage) {
      const fillerH = fullPageBodyH - contentH;
      if (fillerH > 0.02) {
        hasFiller = true;
        rowHeights.push(fillerH);
      }
    }

    const totalRows = contentRowCount + (hasFiller ? 1 : 0);
    const bodyHeight = isLastPage ? contentH : fullPageBodyH;

    // Border helpers: outer border + vertical column separators only (no internal horizontal lines)
    const bSolid = { type: 'solid' as const, color: COLORS.border, pt: 0.5 };
    const bNone = { type: 'none' as const };
    const cellBrd = (ri: number) => [
      ri === 0 ? bSolid : bNone,                   // top: first row only
      bSolid,                                        // right: column separator
      ri === totalRows - 1 ? bSolid : bNone,        // bottom: last row only
      bSolid,                                        // left: column separator
    ] as [typeof bSolid, typeof bSolid, typeof bSolid, typeof bSolid];

    // Cell margin: minimize vertical padding to prevent row overflow
    const cellMargin: [number, number, number, number] = [1, 3, 1, 3]; // [top, right, bottom, left] in points

    // Unified body table (left 3 cols + right 2 cols in same rows)
    const unifiedBodyRows: PptxGenJS.TableRow[] = [];
    for (let i = 0; i < contentRowCount; i++) {
      const lr = pLeft[i];
      const rr = pRight[i];
      const brd = cellBrd(i);
      unifiedBodyRows.push([
        { text: lr.body, options: { valign: 'middle' as const, bold: lr.bold, border: brd, margin: cellMargin } },
        { text: lr.date, options: { valign: 'middle' as const, align: 'center' as const, border: brd, margin: cellMargin } },
        { text: lr.progress, options: { valign: 'middle' as const, align: 'center' as const, border: brd, margin: cellMargin } },
        { text: rr.body, options: { valign: 'middle' as const, bold: rr.bold, border: brd, margin: cellMargin } },
        { text: rr.date, options: { valign: 'middle' as const, align: 'center' as const, border: brd, margin: cellMargin } },
      ]);
    }
    if (hasFiller) {
      const brd = cellBrd(contentRowCount);
      unifiedBodyRows.push(
        [0, 1, 2, 3, 4].map(() => ({ text: '', options: { border: brd, margin: cellMargin } }))
      );
    }

    slide.addTable(unifiedBodyRows, {
      x: LAYOUT.x, y: curY, w: LAYOUT.w, h: bodyHeight,
      colW: unifiedColW, rowH: rowHeights,
      fontFace: FONT.face, fontSize,
    });
    curY += bodyHeight;

    // Issues/Notes — last page only, right after body
    if (isLastPage) {
      slide.addTable(
        [[
          { text: '이슈/위험사항', options: { fill: { color: COLORS.headerBg }, bold: true, fontSize: 9 } },
          { text: issues || '-', options: { fontSize: issueFontSize } },
        ]],
        {
          x: LAYOUT.x, y: curY, w: LAYOUT.w, h: issueH,
          colW: [1.5, LAYOUT.w - 1.5], rowH: [issueH],
          border: { type: 'solid', color: COLORS.border, pt: 0.5 },
          fontFace: FONT.face, fontSize: issueFontSize, valign: 'middle',
        }
      );
      curY += issueH;

      slide.addTable(
        [[
          { text: '특이사항', options: { fill: { color: COLORS.headerBg }, bold: true, fontSize: 9 } },
          { text: notes || '-', options: { fontSize: noteFontSize } },
        ]],
        {
          x: LAYOUT.x, y: curY, w: LAYOUT.w, h: noteH,
          colW: [1.5, LAYOUT.w - 1.5], rowH: [noteH],
          border: { type: 'solid', color: COLORS.border, pt: 0.5 },
          fontFace: FONT.face, fontSize: noteFontSize, valign: 'middle',
        }
      );
    }
  }

  const date = data.report_date.replace(/-/g, '');
  const filename = `${data.team.name}_주간보고_${displayAuthor}_${date}.pptx`;
  const blob = await pptx.write({ outputType: 'blob' }) as Blob;
  downloadBlob(blob, filename);
}
