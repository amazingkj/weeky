export function formatDateShort(dateStr: string): string {
  if (!dateStr) return '';
  const parts = dateStr.split('-');
  if (parts.length !== 3) return dateStr;
  return `${parts[0]}.${parts[1]}.${parts[2]}`;
}

export function getWeekRange(dateStr: string): string {
  if (!dateStr) return '';
  const date = new Date(dateStr);
  const dayOfWeek = date.getDay();
  const monday = new Date(date);
  monday.setDate(date.getDate() - (dayOfWeek === 0 ? 6 : dayOfWeek - 1));
  const friday = new Date(monday);
  friday.setDate(monday.getDate() + 4);

  const fmt = (d: Date) => `${d.getFullYear()}.${String(d.getMonth() + 1).padStart(2, '0')}.${String(d.getDate()).padStart(2, '0')}`;
  return `${fmt(monday)}~${fmt(friday)}`;
}

export function getMonday(dateStr: string): string {
  const date = new Date(dateStr);
  const dayOfWeek = date.getDay();
  const monday = new Date(date);
  monday.setDate(date.getDate() - (dayOfWeek === 0 ? 6 : dayOfWeek - 1));
  return `${monday.getFullYear()}-${String(monday.getMonth() + 1).padStart(2, '0')}-${String(monday.getDate()).padStart(2, '0')}`;
}

export function isSameWeek(dateA: string, dateB: string): boolean {
  return getMonday(dateA) === getMonday(dateB);
}

export function getNextWeekRange(dateStr: string): string {
  if (!dateStr) return '';
  const date = new Date(dateStr);
  const dayOfWeek = date.getDay();
  const nextMonday = new Date(date);
  nextMonday.setDate(date.getDate() + (7 - (dayOfWeek === 0 ? 6 : dayOfWeek - 1)));
  const nextFriday = new Date(nextMonday);
  nextFriday.setDate(nextMonday.getDate() + 4);

  const fmt = (d: Date) => `${d.getFullYear()}.${String(d.getMonth() + 1).padStart(2, '0')}.${String(d.getDate()).padStart(2, '0')}`;
  return `${fmt(nextMonday)}~${fmt(nextFriday)}`;
}
