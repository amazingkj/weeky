function formatDotDate(d: Date): string {
  return `${d.getFullYear()}.${String(d.getMonth() + 1).padStart(2, '0')}.${String(d.getDate()).padStart(2, '0')}`;
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

export function formatDateShort(dateStr: string): string {
  if (!dateStr) return '';
  const parts = dateStr.split('-');
  if (parts.length !== 3) return dateStr;
  return `${parts[0]}.${parts[1]}.${parts[2]}`;
}

export function getWeekRange(dateStr: string): string {
  if (!dateStr) return '';
  const monday = getMondayDate(dateStr);
  const friday = getFridayDate(monday);
  return `${formatDotDate(monday)}~${formatDotDate(friday)}`;
}

export function getMonday(dateStr: string): string {
  const monday = getMondayDate(dateStr);
  return `${monday.getFullYear()}-${String(monday.getMonth() + 1).padStart(2, '0')}-${String(monday.getDate()).padStart(2, '0')}`;
}

export function isSameWeek(dateA: string, dateB: string): boolean {
  return getMonday(dateA) === getMonday(dateB);
}

export function getNextWeekRange(dateStr: string): string {
  if (!dateStr) return '';
  const monday = getMondayDate(dateStr);
  const nextMonday = new Date(monday);
  nextMonday.setDate(monday.getDate() + 7);
  const nextFriday = getFridayDate(nextMonday);
  return `${formatDotDate(nextMonday)}~${formatDotDate(nextFriday)}`;
}
