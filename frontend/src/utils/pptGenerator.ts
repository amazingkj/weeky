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

  createTaskSlide(pptx, report, {
    sectionTitle: '차주계획',
    dateRange: getNextWeekRange(report.report_date),
    tasks: report.next_week,
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

function getConsolidatedFontSize(groups: ConsolidatedGroup[]): number {
  let totalLines = 0;
  for (const g of groups) {
    totalLines += 1; // group title line
    for (const item of g.items) {
      let detail = item.task.details || '-';
      if (item.task.description) detail += '\n' + item.task.description;
      totalLines += detail.split('\n').length;
    }
    totalLines += 1; // spacing between groups
  }
  if (totalLines <= 15) return 9;
  if (totalLines <= 22) return 8;
  if (totalLines <= 30) return 7;
  return 6;
}

function buildConsolidatedColumnText(groups: ConsolidatedGroup[]) {
  const bodyLines: string[] = [];
  const dateLines: string[] = [];
  const progressLines: string[] = [];

  groups.forEach((g, gi) => {
    // Group title line
    bodyLines.push(`${gi + 1}. ${g.title}`);
    dateLines.push('');
    progressLines.push('');

    // Each member's task under this group
    g.items.forEach(item => {
      const memberTag = item.memberName ? `(${item.memberName} ${item.roleCode}) ` : '';
      let detail = `${memberTag}${item.task.details || '-'}`;
      if (item.task.description) detail += '\n' + item.task.description;
      const lines = detail.split('\n');

      lines.forEach((line, li) => {
        bodyLines.push(line);
        dateLines.push(li === 0 ? (item.task.due_date || '-') : '');
        progressLines.push(li === 0 ? `${item.task.progress}%` : '');
      });
    });

    // Spacing between groups
    if (gi < groups.length - 1) {
      bodyLines.push('');
      dateLines.push('');
      progressLines.push('');
    }
  });

  return {
    body: bodyLines.join('\n'),
    dates: dateLines.join('\n'),
    progress: progressLines.join('\n'),
  };
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

  // Determine body font size based on total content
  const allGroups = [...thisWeekGroups, ...nextWeekGroups];
  const bodyFontSize = getConsolidatedFontSize(allGroups);

  const slide = pptx.addSlide();
  let currentY = LAYOUT.y;

  // Row 1: Header
  slide.addTable(
    [[
      { text: '프로젝트명', options: { fill: { color: COLORS.headerBg }, bold: true, align: 'center' } },
      { text: data.team.name, options: { align: 'center' } },
      { text: '보고일자', options: { fill: { color: COLORS.headerBg }, bold: true, align: 'center' } },
      { text: formatDateShort(data.report_date), options: { align: 'center' } },
      { text: '작성자', options: { fill: { color: COLORS.headerBg }, bold: true, align: 'center' } },
      { text: displayAuthor, options: { align: 'center' } },
    ]],
    {
      x: LAYOUT.x, y: currentY, w: LAYOUT.w, h: ROW_H.header,
      colW: HEADER_COL_W, rowH: [ROW_H.header],
      border: { type: 'solid', color: COLORS.border, pt: 0.5 },
      fontFace: FONT.face, fontSize: FONT.size, valign: 'middle',
    }
  );
  currentY += ROW_H.header;

  // Row 2: Two-column section headers (금주실적 | 차주계획)
  const halfW = LAYOUT.w / 2;
  slide.addTable(
    [[
      { text: '금주실적', options: { fill: { color: COLORS.headerBg }, bold: true, align: 'center' } },
      { text: '차주계획', options: { fill: { color: COLORS.headerBg }, bold: true, align: 'center' } },
    ]],
    {
      x: LAYOUT.x, y: currentY, w: LAYOUT.w, h: ROW_H.section,
      colW: [halfW, halfW], rowH: [ROW_H.section],
      border: { type: 'solid', color: COLORS.border, pt: 0.5 },
      fontFace: FONT.face, fontSize: FONT.size, valign: 'middle',
    }
  );
  currentY += ROW_H.section;

  // Row 3: Column headers for both sides
  const leftColHeader: PptxGenJS.TableCell[] = [
    { text: `계획업무\n(${dateRange})`, options: { fill: { color: COLORS.headerBg }, bold: true, valign: 'middle' } },
    { text: '완료일', options: { fill: { color: COLORS.headerBg }, bold: true, align: 'center', valign: 'middle' } },
    { text: '실적(%)', options: { fill: { color: COLORS.headerBg }, bold: true, align: 'center', valign: 'middle' } },
  ];
  const rightColHeader: PptxGenJS.TableCell[] = [
    { text: `계획업무\n(${nextDateRange})`, options: { fill: { color: COLORS.headerBg }, bold: true, valign: 'middle' } },
    { text: '완료\n예정일', options: { fill: { color: COLORS.headerBg }, bold: true, align: 'center', valign: 'middle' } },
  ];

  // Left column headers
  const leftHeaderW = [2.8, 1.0, 0.9];
  slide.addTable(
    [leftColHeader],
    {
      x: LAYOUT.x, y: currentY, w: halfW, h: ROW_H.colHeader,
      colW: leftHeaderW, rowH: [ROW_H.colHeader],
      border: { type: 'solid', color: COLORS.border, pt: 0.5 },
      fontFace: FONT.face, fontSize: bodyFontSize, valign: 'middle',
    }
  );

  // Right column headers
  const rightHeaderW = [3.2, 1.5];
  slide.addTable(
    [rightColHeader],
    {
      x: LAYOUT.x + halfW, y: currentY, w: halfW, h: ROW_H.colHeader,
      colW: rightHeaderW, rowH: [ROW_H.colHeader],
      border: { type: 'solid', color: COLORS.border, pt: 0.5 },
      fontFace: FONT.face, fontSize: bodyFontSize, valign: 'middle',
    }
  );
  currentY += ROW_H.colHeader;

  // Row 4: Body content (two-column)
  const leftData = buildConsolidatedColumnText(thisWeekGroups);
  const rightData = buildConsolidatedColumnText(nextWeekGroups);

  // Left body
  slide.addTable(
    [[
      { text: leftData.body, options: { valign: 'top' } },
      { text: leftData.dates, options: { valign: 'top', align: 'center' } },
      { text: leftData.progress, options: { valign: 'top', align: 'center' } },
    ]],
    {
      x: LAYOUT.x, y: currentY, w: halfW, h: ROW_H.body,
      colW: leftHeaderW, rowH: [ROW_H.body],
      border: { type: 'solid', color: COLORS.border, pt: 0.5 },
      fontFace: FONT.face, fontSize: bodyFontSize, valign: 'top',
    }
  );

  // Right body
  slide.addTable(
    [[
      { text: rightData.body, options: { valign: 'top' } },
      { text: rightData.dates, options: { valign: 'top', align: 'center' } },
    ]],
    {
      x: LAYOUT.x + halfW, y: currentY, w: halfW, h: ROW_H.body,
      colW: rightHeaderW, rowH: [ROW_H.body],
      border: { type: 'solid', color: COLORS.border, pt: 0.5 },
      fontFace: FONT.face, fontSize: bodyFontSize, valign: 'top',
    }
  );
  currentY += ROW_H.body;

  // Row 5: Issues (full width)
  slide.addTable(
    [[
      { text: '이슈/위험사항', options: { fill: { color: COLORS.headerBg }, bold: true } },
      { text: issues || '-', options: {} },
    ]],
    {
      x: LAYOUT.x, y: currentY, w: LAYOUT.w, h: ROW_H.issue,
      colW: [1.5, LAYOUT.w - 1.5], rowH: [ROW_H.issue],
      border: { type: 'solid', color: COLORS.border, pt: 0.5 },
      fontFace: FONT.face, fontSize: FONT.size, valign: 'middle',
    }
  );
  currentY += ROW_H.issue;

  // Row 6: Notes (full width)
  slide.addTable(
    [[
      { text: '특이사항', options: { fill: { color: COLORS.headerBg }, bold: true } },
      { text: notes || '-', options: {} },
    ]],
    {
      x: LAYOUT.x, y: currentY, w: LAYOUT.w, h: ROW_H.note,
      colW: [1.5, LAYOUT.w - 1.5], rowH: [ROW_H.note],
      border: { type: 'solid', color: COLORS.border, pt: 0.5 },
      fontFace: FONT.face, fontSize: FONT.size, valign: 'middle',
    }
  );

  // Generate filename: {팀이름}_주간보고_{작성자}_{날짜}.pptx
  const date = data.report_date.replace(/-/g, '');
  const filename = `${data.team.name}_주간보고_${displayAuthor}_${date}.pptx`;
  const blob = await pptx.write({ outputType: 'blob' }) as Blob;
  downloadBlob(blob, filename);
}
