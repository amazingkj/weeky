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

  createTaskSlide(pptx, report, {
    sectionTitle: '차주계획',
    dateRange: getNextWeekRange(report.report_date),
    tasks: report.next_week,
    showProgress: false,
    issuesText: '',
    notesText: '',
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
        autoPage: false,
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
        autoPage: false,
      }
  );
  currentY += ROW_H.section;

  // Row 3-4: Body (column headers + task content) — grouped by title, sub-grouped by client
  const bodyFontSize = getBodyFontSize(config.tasks);
  const groups = groupTasksByTitle(config.tasks);

  const groupData = groups.map((g, gi) => {
    const titleParts: string[] = [];
    const detailParts: string[] = [];
    const dateParts: string[] = [];
    const progressParts: string[] = [];

    // Project title (own line)
    titleParts.push(`${gi + 1}. ${g.title}`);
    detailParts.push('');
    dateParts.push('');
    progressParts.push('');

    // Sub-group by client
    const clientMap = new Map<string, Task[]>();
    for (const t of g.items) {
      const key = (t.client || '').trim();
      if (!clientMap.has(key)) clientMap.set(key, []);
      clientMap.get(key)!.push(t);
    }

    for (const [client, tasks] of clientMap) {
      if (client) {
        titleParts.push(`  • ${client}`);
        detailParts.push('');
        dateParts.push('');
        progressParts.push('');
      }

      for (const t of tasks) {
        const detailText = getTaskDetailText(t);
        const detailLines = detailText.split('\n');

        titleParts.push('');
        for (let l = 1; l < detailLines.length; l++) titleParts.push('');

        detailParts.push(...detailLines);

        dateParts.push(t.due_date || '-');
        for (let l = 1; l < detailLines.length; l++) dateParts.push('');

        progressParts.push(`${t.progress}%`);
        for (let l = 1; l < detailLines.length; l++) progressParts.push('');
      }
    }

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

  // Row 3: Column header — 별도 테이블 (rowH 최소높이 확장이 body에 영향 안 줌)
  slide.addTable(
      [bodyHeaderRow],
      {
        x: LAYOUT.x, y: currentY, w: LAYOUT.w, h: ROW_H.colHeader,
        colW: bodyColW, rowH: [ROW_H.colHeader],
        border: { type: 'solid', color: COLORS.border, pt: 0.5 },
        fontFace: FONT.face, fontSize: bodyFontSize, valign: 'middle',
        autoPage: false,
      }
  );
  currentY += ROW_H.colHeader;

  // Row 4: Body — 단일 행, footerY까지 정확히 채움
  const footerY = LAYOUT.y + LAYOUT.h - ROW_H.issue - ROW_H.note;
  const bodyH = footerY - currentY;

  slide.addTable(
      [bodyContentRow],
      {
        x: LAYOUT.x, y: currentY, w: LAYOUT.w, h: bodyH,
        colW: bodyColW, rowH: [bodyH],
        border: { type: 'solid', color: COLORS.border, pt: 0.5 },
        fontFace: FONT.face, fontSize: bodyFontSize, valign: 'top',
        autoPage: false,
      }
  );

  // Row 5+6: Issues + Notes — 슬라이드 하단 절대 고정
  const footerCols = config.showProgress ? 4 : 3;
  slide.addTable(
      [
        [
          { text: '이슈/위험 사항', options: { fill: { color: COLORS.headerBg }, bold: true } },
          { text: config.issuesText, options: { colspan: footerCols - 1 } },
        ],
        [
          { text: '특이 사항', options: { fill: { color: COLORS.headerBg }, bold: true } },
          { text: config.notesText, options: { colspan: footerCols - 1 } },
        ],
      ],
      {
        x: LAYOUT.x, y: footerY, w: LAYOUT.w, h: ROW_H.issue + ROW_H.note,
        colW: bodyColW, rowH: [ROW_H.issue, ROW_H.note],
        border: { type: 'solid', color: COLORS.border, pt: 0.5 },
        fontFace: FONT.face, fontSize: FONT.size, valign: 'middle',
        autoPage: false,
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
      .filter(m => m.report && m.report[field]?.trim())
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
    const detail = item.task.details || '-';
    rows.push({
      body: `${indent}- ${detail}${memberTag}`,
      date: formatDateShortMMDD(item.task.due_date),
      progress: `${item.task.progress}%`,
      bold: false,
    });
  }
  return rows;
}

// NBSP( )를 사용하여 PPT 렌더 시 leading 공백 보존 보장
// (일반 space는 splitTextToFit의 split(' ')에 의해 토큰화돼 leading 공백이 유실되고,
//  일부 PPT 렌더러도 leading 일반공백을 시각적으로 무시할 수 있음)
const NBSP = ' ';
const INDENT_CLIENT_HEADER = NBSP.repeat(2);   // "  • 고객사"
const INDENT_DETAIL_W_CLIENT = NBSP.repeat(6); // "      - 진행사항"
const INDENT_DETAIL_NO_CLIENT = NBSP.repeat(2); // "  - 진행사항"

// Build rows for one side independently (project → client → detail)
function buildSideRows(groups: ConsolidatedGroup[]): BodyRow[] {
  const rows: BodyRow[] = [];
  for (const g of groups) {
    rows.push({ body: `[${g.title}]`, date: '', progress: '', bold: true });
    const clientMap = subGroupByClient(g.items);
    for (const [client, items] of clientMap) {
      const indent = client ? INDENT_DETAIL_W_CLIENT : INDENT_DETAIL_NO_CLIENT;
      if (client) {
        rows.push({ body: `${INDENT_CLIENT_HEADER}• ${client}`, date: '', progress: '', bold: false });
      }
      rows.push(...buildItemRows(items, indent));
    }
  }
  return rows;
}

// Build left/right rows independently, pad shorter side at the end
function buildAlignedRows(
    leftGroups: ConsolidatedGroup[],
    rightGroups: ConsolidatedGroup[]
): { leftRows: BodyRow[]; rightRows: BodyRow[] } {
  const emptyRow: BodyRow = { body: '', date: '', progress: '', bold: false };
  const leftRows = buildSideRows(leftGroups);
  const rightRows = buildSideRows(rightGroups);

  // Pad shorter side to match total length
  while (leftRows.length < rightRows.length) leftRows.push(emptyRow);
  while (rightRows.length < leftRows.length) rightRows.push(emptyRow);

  return { leftRows, rightRows };
}

// lineSpacing = fontSize + 2 (pt) 기준 줄 높이 (inch = pt/72)
function getConsolidatedRowH(fontSize: number): number {
  return (fontSize + 2) / 72;
  // 9pt → 11pt → 0.153", 8pt → 10pt → 0.139", 7pt → 9pt → 0.125", 6pt → 8pt → 0.111"
}

// 한글/CJK 문자는 폭이 2배 → 2글자로 카운트
function measureTextWidth(text: string): number {
  let w = 0;
  for (const ch of text) {
    w += /[\u3000-\u9fff\uac00-\ud7af\uff01-\uff60]/.test(ch) ? 2 : 1;
  }
  return w;
}

// 단어 단위로 텍스트를 셀 폭에 맞게 분할 (word-wrap 대체)
// leading whitespace(NBSP/space)는 추출해두고 wrap된 모든 줄에 다시 prepend하여
// 미리보기의 CSS pl-* 동작(둘째 줄도 들여쓰기)과 일치시킨다.
function splitTextToFit(text: string, charsPerLine: number): string[] {
  if (!text) return [''];
  if (measureTextWidth(text) <= charsPerLine) return [text];

  let i = 0;
  while (i < text.length && (text[i] === ' ' || text[i] === ' ')) i++;
  const indent = text.slice(0, i);
  const body = text.slice(i);
  const indentW = measureTextWidth(indent);
  const innerCPL = Math.max(charsPerLine - indentW, 1);

  const segments = body.split(' ');
  const lines: string[] = [];
  let current = '';
  for (const seg of segments) {
    const test = current ? `${current} ${seg}` : seg;
    if (measureTextWidth(test) <= innerCPL) {
      current = test;
    } else {
      if (current) lines.push(current);
      current = seg;
    }
  }
  if (current) lines.push(current);
  return lines.length > 0 ? lines.map(l => indent + l) : [indent];
}

function getCharsPerInch(fs: number): number {
  if (fs >= 9) return 14;
  if (fs >= 8) return 16;
  if (fs >= 7) return 18;
  return 21;
}

// 한 쪽의 BodyRow를 visual line 단위로 확장 (word-wrap 분할)
function expandSideRows(rows: BodyRow[], cpl: number): BodyRow[] {
  const expanded: BodyRow[] = [];
  for (const r of rows) {
    const lines = splitTextToFit(r.body, cpl);
    for (let j = 0; j < lines.length; j++) {
      expanded.push({
        body: lines[j],
        date: j === 0 ? r.date : '',
        progress: j === 0 ? r.progress : '',
        bold: j === 0 && r.bold,
      });
    }
  }
  return expanded;
}

// 좌우 각각 독립 확장 후 짧은 쪽만 끝에서 패딩
function expandAlignedRows(
    leftRows: BodyRow[], rightRows: BodyRow[],
    leftCPL: number, rightCPL: number
): { leftRows: BodyRow[]; rightRows: BodyRow[] } {
  const empty: BodyRow = { body: '', date: '', progress: '', bold: false };
  const eL = expandSideRows(leftRows, leftCPL);
  const eR = expandSideRows(rightRows, rightCPL);

  while (eL.length < eR.length) eL.push(empty);
  while (eR.length < eL.length) eR.push(empty);

  return { leftRows: eL, rightRows: eR };
}

// 각 폰트 크기별로 텍스트 확장 후 페이지 수 결정
function calcPagination(
    leftRows: BodyRow[], rightRows: BodyRow[], bodyH: number,
    leftColW0: number, rightColW0: number, marginW: number
) {
  const fontSizes = [9, 8, 7, 6] as const;

  function tryFont(fs: number) {
    const cpi = getCharsPerInch(fs);
    const leftCPL = Math.floor((leftColW0 - marginW) * cpi);
    const rightCPL = Math.floor((rightColW0 - marginW) * cpi);
    const { leftRows: eL, rightRows: eR } = expandAlignedRows(leftRows, rightRows, leftCPL, rightCPL);
    const maxRows = Math.max(eL.length, eR.length, 1);
    const rh = getConsolidatedRowH(fs);
    const perPage = Math.floor(bodyH / rh);
    return { eL, eR, maxRows, rh, perPage };
  }

  // 1페이지에 들어가면 가장 큰 폰트 사용
  for (const fs of fontSizes) {
    const { eL, eR, maxRows, rh, perPage } = tryFont(fs);
    if (maxRows <= perPage) {
      return { fontSize: fs, rowH: rh, pages: 1, rowsPerPage: perPage, expandedLeft: eL, expandedRight: eR };
    }
  }

  // 여러 페이지: 최소 폰트로 최소 페이지 수 → 같은 페이지 수에서 가장 큰 폰트
  const res6 = tryFont(6);
  const minPages = Math.ceil(res6.maxRows / res6.perPage);

  for (const fs of fontSizes) {
    const { eL, eR, maxRows, rh, perPage } = tryFont(fs);
    if (Math.ceil(maxRows / perPage) <= minPages) {
      return { fontSize: fs, rowH: rh, pages: minPages, rowsPerPage: perPage, expandedLeft: eL, expandedRight: eR };
    }
  }

  return { fontSize: 6, rowH: res6.rh, pages: minPages, rowsPerPage: res6.perPage, expandedLeft: res6.eL, expandedRight: res6.eR };
}

export async function generateConsolidatedPPT(
    data: ConsolidatedReport,
    leaderName?: string
): Promise<void> {
  const pptx = new PptxGenJS();
  pptx.author = leaderName || data.team.name;
  pptx.title = `${data.team.name} 주간업무보고`;
  pptx.subject = '주간업무보고';
  pptx.layout = 'LAYOUT_4x3';

  const thisWeekGroups = groupConsolidatedTasks(data.members, 'this_week');
  const nextWeekGroups = groupConsolidatedTasks(data.members, 'next_week');

  const issuesLeft = mergeIssuesNotes(data.members, 'issues');
  const issuesRight = mergeIssuesNotes(data.members, 'next_issues');
  const notesLeft = mergeIssuesNotes(data.members, 'notes');
  const notesRight = mergeIssuesNotes(data.members, 'next_notes');

  const dateRange = getWeekRange(data.report_date);
  const nextDateRange = getNextWeekRange(data.report_date);
  const displayAuthor = leaderName || data.team.name;
  const leftW = 4.6;                    // 헤더 "보고일자 | 날짜" 경계에 맞춤
  const rightW = LAYOUT.w - leftW;      // 4.8
  const leftColW = [3.0, 0.8, 0.8];    // 계획업무 | 완료일 | 실적%  (합 = 4.6)
  const rightColW = [3.8, 1.0];         // 계획업무 | 완료예정일      (합 = 4.8)

  // Build row-aligned data (left/right aligned by project+client)
  const { leftRows, rightRows } = buildAlignedRows(thisWeekGroups, nextWeekGroups);

  // Footer: 3-column layout [header | 금주 | 차주] matching left/right body split
  // footer는 unifiedColW + colspan으로 처리하므로 별도 footerColW 불필요

  // FIX: 이슈/특이사항 높이 — 텍스트 길이 기반으로 wrap 줄 수 추정
  // 셀 너비 = LAYOUT.w - 헤더열(1.5) = 7.9inch, 폰트 8pt 기준 한 줄 약 110자
  const ISSUE_CELL_W = LAYOUT.w - 1.5;
  const CHARS_PER_LINE = Math.floor(ISSUE_CELL_W * 14); // 1inch당 약 14자 (8pt 기준)
  const LINE_H = 0.22;

  function calcWrappedLineCount(text: string, baseFontSize: number): number {
    const lines = (text || '-').split('\n');
    const charsPerLine = Math.floor(CHARS_PER_LINE * (8 / baseFontSize));
    return lines.reduce((sum, line) => sum + Math.max(1, Math.ceil(line.length / charsPerLine)), 0);
  }

  const issueLineCount = Math.max(
      calcWrappedLineCount(issuesLeft, 8),
      calcWrappedLineCount(issuesRight, 8)
  );
  const issueFontSize = issueLineCount > 3 ? 7 : 8;
  const issueH = Math.max(0.50, Math.min(1.5, issueLineCount * LINE_H + 0.15));

  const noteLineCount = Math.max(
      calcWrappedLineCount(notesLeft, 8),
      calcWrappedLineCount(notesRight, 8)
  );
  const noteFontSize = noteLineCount > 3 ? 7 : 8;
  const noteH = Math.max(0.50, Math.min(1.5, noteLineCount * LINE_H + 0.15));

  // FIX: footer 높이를 먼저 확보하고 body 높이 계산
  // (LAYOUT.y + LAYOUT.h) - LAYOUT.y - fixedH = LAYOUT.h - fixedH
  const fixedH = ROW_H.header + ROW_H.section + ROW_H.colHeader + issueH + noteH;
  const bodyH = LAYOUT.h - fixedH;

  const cellMargin: [number, number, number, number] = [1, 3, 1, 3];
  const marginW = (cellMargin[1] + cellMargin[3]) / 72;

  const { fontSize, pages, rowsPerPage, expandedLeft, expandedRight } =
      calcPagination(leftRows, rightRows, bodyH, leftColW[0], rightColW[0], marginW);

  const unifiedColW = [...leftColW, ...rightColW];

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
          autoPage: false,
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
          colW: [leftW, rightW], rowH: [ROW_H.section],
          border: { type: 'solid', color: COLORS.border, pt: 0.5 },
          fontFace: FONT.face, fontSize: FONT.size, valign: 'middle',
          autoPage: false,
        }
    );
    curY += ROW_H.section;

    // Row 3: Column headers (unified 5-column table)
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
          autoPage: false,
        }
    );
    curY += ROW_H.colHeader;

    // Body — 확장된 visual line 사용 (텍스트 미리 분할 → word-wrap 불일치 제거)
    const isLastPage = page === pages - 1;
    const startIdx = page * rowsPerPage;
    const maxSide = Math.max(expandedLeft.length, expandedRight.length, 1);
    const endIdx = Math.min(startIdx + rowsPerPage, maxSide);
    const emptyRow: BodyRow = { body: '', date: '', progress: '', bold: false };

    const pLeft = expandedLeft.slice(startIdx, endIdx);
    const pRight = expandedRight.slice(startIdx, endIdx);
    while (pLeft.length < pRight.length) pLeft.push(emptyRow);
    while (pRight.length < pLeft.length) pRight.push(emptyRow);

    const footerY = LAYOUT.y + LAYOUT.h - issueH - noteH;
    const headerFixedH = ROW_H.header + ROW_H.section + ROW_H.colHeader;
    const fullPageBodyH = LAYOUT.h - headerFixedH;
    const bodyHeight = isLastPage ? (footerY - curY) : fullPageBodyH;

    // 1:1 TextProps — lineSpacing 고정으로 셀 간 줄 높이 통일
    const lineH = fontSize + 2; // 고정 줄 높이 (pt)
    const baseOpts = { breakLine: true as const, fontSize, paraSpaceBefore: 0, paraSpaceAfter: 0, lineSpacing: lineH };
    const leftBody: PptxGenJS.TextProps[] = [];
    const leftDate: PptxGenJS.TextProps[] = [];
    const leftProgress: PptxGenJS.TextProps[] = [];
    const rightBody: PptxGenJS.TextProps[] = [];
    const rightDate: PptxGenJS.TextProps[] = [];

    for (const r of pLeft) {
      leftBody.push({ text: r.body, options: { ...baseOpts, ...(r.bold ? { bold: true } : {}) } });
      leftDate.push({ text: r.date || ' ', options: baseOpts });
      leftProgress.push({ text: r.progress || ' ', options: baseOpts });
    }
    for (const r of pRight) {
      rightBody.push({ text: r.body || ' ', options: { ...baseOpts, ...(r.bold ? { bold: true } : {}) } });
      rightDate.push({ text: r.date || ' ', options: baseOpts });
    }

    slide.addTable(
        [[
          { text: leftBody, options: { valign: 'top' as const, margin: cellMargin } },
          { text: leftDate, options: { valign: 'top' as const, align: 'center' as const, margin: cellMargin } },
          { text: leftProgress, options: { valign: 'top' as const, align: 'center' as const, margin: cellMargin } },
          { text: rightBody, options: { valign: 'top' as const, margin: cellMargin } },
          { text: rightDate, options: { valign: 'top' as const, align: 'center' as const, margin: cellMargin } },
        ]],
        {
          x: LAYOUT.x, y: curY, w: LAYOUT.w, h: bodyHeight,
          colW: unifiedColW, rowH: [bodyHeight],
          border: { type: 'solid', color: COLORS.border, pt: 0.5 },
          fontFace: FONT.face, fontSize,
          autoPage: false,
        }
    );
    curY += bodyHeight;

    // Issues/Notes — 마지막 페이지에만, body 바로 아래
    // FIX: body와 동일한 unifiedColW 사용 → 경계선 정렬
    // footer: 이슈/특이사항을 하나의 테이블로 합쳐서 curY 누적 오차 제거
    if (isLastPage) {
      const footerFontSize = Math.min(issueFontSize, noteFontSize);
      const issuesMerged = [issuesLeft, issuesRight].filter(Boolean).join('  /  ') || '-';
      const notesMerged = [notesLeft, notesRight].filter(Boolean).join('  /  ') || '-';
      const footerColW = [HEADER_COL_W[0], LAYOUT.w - HEADER_COL_W[0]]; // [1.3, 8.1] 프로젝트명 너비에 맞춤
      slide.addTable(
          [
            [
              { text: '이슈/위험사항', options: { fill: { color: COLORS.headerBg }, bold: true, fontSize: 9, valign: 'middle' } },
              { text: issuesMerged, options: { fontSize: issueFontSize, valign: 'middle' } },
            ],
            [
              { text: '특이사항', options: { fill: { color: COLORS.headerBg }, bold: true, fontSize: 9, valign: 'middle' } },
              { text: notesMerged, options: { fontSize: noteFontSize, valign: 'middle' } },
            ],
          ],
          {
            x: LAYOUT.x, y: footerY, w: LAYOUT.w, h: issueH + noteH,
            colW: footerColW, rowH: [issueH, noteH],
            border: { type: 'solid', color: COLORS.border, pt: 0.5 },
            fontFace: FONT.face, fontSize: footerFontSize, valign: 'middle',
            autoPage: false,
          }
      );
    }
  }

  const date = data.report_date.replace(/-/g, '');
  const filename = `${data.team.name}_주간보고_${displayAuthor}_${date}.pptx`;
  const blob = await pptx.write({ outputType: 'blob' }) as Blob;
  downloadBlob(blob, filename);
}