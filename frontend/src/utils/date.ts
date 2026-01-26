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

  const fmt = (d: Date) => `${d.getFullYear()}.${d.getMonth() + 1}.${d.getDate()}`;
  return `${fmt(monday)}~${fmt(friday)}`;
}

export function getNextWeekRange(dateStr: string): string {
  if (!dateStr) return '';
  const date = new Date(dateStr);
  const dayOfWeek = date.getDay();
  const nextMonday = new Date(date);
  nextMonday.setDate(date.getDate() + (7 - (dayOfWeek === 0 ? 6 : dayOfWeek - 1)));
  const nextFriday = new Date(nextMonday);
  nextFriday.setDate(nextMonday.getDate() + 4);

  const fmt = (d: Date) => `${d.getFullYear()}.${d.getMonth() + 1}.${d.getDate()}`;
  return `${fmt(nextMonday)}~${fmt(nextFriday)}`;
}
