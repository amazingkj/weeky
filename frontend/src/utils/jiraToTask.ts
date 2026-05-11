import { SyncItem, Task } from '../types';

// "CruzAPIM 1.5", "CruzAPIM v2.0", "CruzAPIM 2.3.1" → "CruzAPIM"
function stripVersionSuffix(name: string): string {
  return name.replace(/\s+v?\d+(\.\d+)*\s*$/i, '').trim();
}

// status 명칭에서 진행률 추정. 매칭 안되면 30%.
function statusToProgress(status: string): number {
  const s = status.toLowerCase().trim();
  if (!s) return 0;

  const doneKeywords = ['done', '완료', 'closed', 'resolved', 'deployed', '배포', '종료'];
  if (doneKeywords.some(k => s.includes(k))) return 100;

  const reviewKeywords = ['review', 'qa', '검토', '리뷰', '테스트', 'verify', 'verification'];
  if (reviewKeywords.some(k => s.includes(k))) return 80;

  const progressKeywords = ['progress', 'doing', 'wip', '진행', '작업중'];
  if (progressKeywords.some(k => s.includes(k))) return 50;

  const todoKeywords = ['todo', 'to do', 'open', '신규', '대기', 'backlog', 'new'];
  if (todoKeywords.some(k => s.includes(k))) return 0;

  return 30;
}

// Jira URL .../browse/PROJ-123 → "PROJ-123"
function extractTicketKey(url: string): string {
  const match = url.match(/\/browse\/([^/?#]+)/);
  return match ? match[1] : '';
}

export function jiraItemsToTasks(items: SyncItem[]): Task[] {
  const tasks: Task[] = [];

  for (const item of items) {
    if (item.type !== 'issue' && item.type !== 'issue_done' && item.type !== 'issue_todo') {
      continue;
    }

    const title = item.solution ? stripVersionSuffix(item.solution) : item.title;
    const client = item.site || '';
    const key = extractTicketKey(item.url);
    const details = key ? `[${key}] ${item.title}` : item.title;
    const dueDate = item.due_date || '';
    const progress = statusToProgress(item.content || '');

    tasks.push({
      title: title || 'Jira',
      client,
      details,
      description: '',
      due_date: dueDate,
      progress,
    });
  }

  return tasks;
}
