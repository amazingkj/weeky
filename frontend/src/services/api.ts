import { Template, Report, SyncResult, SyncItem, Task, GitHubSyncRequest, GitLabSyncRequest, JiraSyncRequest, HiworksSyncRequest, ConfigMap } from '../types';

// AI Generate types
export interface GenerateReportRequest {
  items: SyncItem[];
  start_date: string;
  end_date: string;
}

export interface GenerateReportResponse {
  this_week: Task[];
  next_week?: Task[];
  summary: string;
}

const API_BASE = '/api/v1';

export async function getTemplates(): Promise<Template[]> {
  const res = await fetch(`${API_BASE}/templates`);
  if (!res.ok) throw new Error('Failed to fetch templates');
  return res.json();
}

export async function createTemplate(name: string, style: string = '{}'): Promise<Template> {
  const res = await fetch(`${API_BASE}/templates`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, style }),
  });
  if (!res.ok) throw new Error('Failed to create template');
  return res.json();
}

export async function updateTemplate(id: number, name: string, style: string): Promise<void> {
  const res = await fetch(`${API_BASE}/templates/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, style }),
  });
  if (!res.ok) throw new Error('Failed to update template');
}

export async function deleteTemplate(id: number): Promise<void> {
  const res = await fetch(`${API_BASE}/templates/${id}`, { method: 'DELETE' });
  if (!res.ok) throw new Error('Failed to delete template');
}

export async function getReport(id: number): Promise<Report> {
  const res = await fetch(`${API_BASE}/reports/${id}`);
  if (!res.ok) throw new Error('Failed to fetch report');
  return res.json();
}

export async function saveReport(report: Omit<Report, 'id'>): Promise<Report> {
  const res = await fetch(`${API_BASE}/reports`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(report),
  });
  if (!res.ok) throw new Error('Failed to save report');
  return res.json();
}

// Config API
export async function getConfig(): Promise<ConfigMap> {
  const res = await fetch(`${API_BASE}/config`);
  if (!res.ok) throw new Error('Failed to fetch config');
  return res.json();
}

export async function updateConfig(configs: ConfigMap): Promise<void> {
  const res = await fetch(`${API_BASE}/config`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ configs }),
  });
  if (!res.ok) throw new Error('Failed to update config');
}

// Sync API
export async function syncGitHub(request: GitHubSyncRequest): Promise<SyncResult> {
  const res = await fetch(`${API_BASE}/sync/github`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(request),
  });
  if (!res.ok) {
    const error = await res.json();
    throw new Error(error.error || 'Failed to sync GitHub');
  }
  return res.json();
}

export async function syncGitLab(request: GitLabSyncRequest): Promise<SyncResult> {
  const res = await fetch(`${API_BASE}/sync/gitlab`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(request),
  });
  if (!res.ok) {
    const error = await res.json();
    throw new Error(error.error || 'Failed to sync GitLab');
  }
  return res.json();
}

export async function syncJira(request: JiraSyncRequest): Promise<SyncResult> {
  const res = await fetch(`${API_BASE}/sync/jira`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(request),
  });
  if (!res.ok) {
    const error = await res.json();
    throw new Error(error.error || 'Failed to sync Jira');
  }
  return res.json();
}

export async function syncHiworks(request: HiworksSyncRequest): Promise<SyncResult> {
  const res = await fetch(`${API_BASE}/sync/hiworks`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(request),
  });
  if (!res.ok) {
    const error = await res.json();
    throw new Error(error.error || 'Failed to sync Hiworks');
  }
  return res.json();
}

// AI Generate API
export async function generateAIReport(request: GenerateReportRequest): Promise<GenerateReportResponse> {
  const res = await fetch(`${API_BASE}/ai/generate`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(request),
  });
  if (!res.ok) {
    const error = await res.json();
    throw new Error(error.error || 'AI 보고서 생성에 실패했습니다');
  }
  return res.json();
}
