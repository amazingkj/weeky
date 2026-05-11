function pad2(n: number): string {
  return String(n).padStart(2, '0');
}

function formatDate(d: Date, sep: string): string {
  return `${d.getFullYear()}${sep}${pad2(d.getMonth() + 1)}${sep}${pad2(d.getDate())}`;
}

function getMondayDate(dateStr: string): Date {
  const date = new Date(dateStr);
  const dayOfWeek = date.getDay();
  const monday = new Date(date);
  monday.setDate(date.getDate() - (dayOfWeek === 0 ? 6 : dayOfWeek - 1));
  return monday;
}

function getFridayDate(monday: Date): Date {
  const friday = new Date(monday);
  friday.setDate(monday.getDate() + 4);
  return friday;
}

// YYYY-MM-DD → "MM/DD" — 짧은 표기 (취합 PPT 본문, 미리보기 공통)
export function formatDateMMDD(dateStr: string): string {
  if (!dateStr) return '-';
  const parts = dateStr.split('-');
  if (parts.length >= 3) return `${parts[1]}/${parts[2]}`;
  return dateStr;
}

// 주차의 월/금 범위 — separator(`.` or `/`), 양옆 공백 여부를 지정 가능
export function formatWeekRange(dateStr: string, sep: '.' | '/' = '.', spacePadded = false): string {
  if (!dateStr) return '';
  const monday = getMondayDate(dateStr);
  const friday = getFridayDate(monday);
  const tilde = spacePadded ? ' ~ ' : '~';
  return `${formatDate(monday, sep)}${tilde}${formatDate(friday, sep)}`;
}

// 다음 주차의 월/금 범위
export function formatNextWeekRange(dateStr: string, sep: '.' | '/' = '.', spacePadded = false): string {
  if (!dateStr) return '';
  const monday = getMondayDate(dateStr);
  const nextMonday = new Date(monday);
  nextMonday.setDate(monday.getDate() + 7);
  const nextFriday = getFridayDate(nextMonday);
  const tilde = spacePadded ? ' ~ ' : '~';
  return `${formatDate(nextMonday, sep)}${tilde}${formatDate(nextFriday, sep)}`;
}

// 주차의 금요일 YYYY-MM-DD
export function getFridayOfWeek(dateStr: string): string {
  if (!dateStr) return '';
  const friday = getFridayDate(getMondayDate(dateStr));
  return `${friday.getFullYear()}-${pad2(friday.getMonth() + 1)}-${pad2(friday.getDate())}`;
}

export function formatDateShort(dateStr: string): string {
  if (!dateStr) return '';
  const parts = dateStr.split('-');
  if (parts.length !== 3) return dateStr;
  return `${parts[0]}.${parts[1]}.${parts[2]}`;
}

export function getWeekRange(dateStr: string): string {
  return formatWeekRange(dateStr, '.', false);
}

export function getMonday(dateStr: string): string {
  const monday = getMondayDate(dateStr);
  return `${monday.getFullYear()}-${pad2(monday.getMonth() + 1)}-${pad2(monday.getDate())}`;
}

export function isSameWeek(dateA: string, dateB: string): boolean {
  return getMonday(dateA) === getMonday(dateB);
}

export function getNextWeekRange(dateStr: string): string {
  return formatNextWeekRange(dateStr, '.', false);
}
