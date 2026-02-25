import { Template, Report, SyncResult, SyncItem, Task, GitHubSyncRequest, GitLabSyncRequest, JiraSyncRequest, HiworksSyncRequest, ConfigMap, AuthResponse, LoginRequest, RegisterRequest, User, InviteCode, GitLabProject, Team, TeamMember, TeamRole, RoleCode, ReportSubmission, TeamMemberWithSubmission, ConsolidatedReport } from '../types';

// AI Generate types
export interface GenerateReportRequest {
  items: SyncItem[];
  start_date: string;
  end_date: string;
  style?: 'concise' | 'detailed' | 'very_detailed';
}

export interface GenerateReportResponse {
  this_week: Task[];
  next_week?: Task[];
  summary: string;
}

const API_BASE = '/api/v1';

// Token management
function getToken(): string | null {
  return localStorage.getItem('token');
}

function setToken(token: string): void {
  localStorage.setItem('token', token);
}

function getRefreshToken(): string | null {
  return localStorage.getItem('refresh_token');
}

function setRefreshToken(token: string): void {
  localStorage.setItem('refresh_token', token);
}

function clearToken(): void {
  localStorage.removeItem('token');
  localStorage.removeItem('refresh_token');
  localStorage.removeItem('user');
}

// Event for 401 responses - AuthContext listens for this
export const AUTH_EXPIRED_EVENT = 'auth:expired';

function emitAuthExpired() {
  window.dispatchEvent(new CustomEvent(AUTH_EXPIRED_EVENT));
}

// Refresh the access token using the refresh token
let refreshPromise: Promise<boolean> | null = null;

async function tryRefreshToken(): Promise<boolean> {
  // Deduplicate concurrent refresh attempts
  if (refreshPromise) return refreshPromise;

  refreshPromise = (async () => {
    const rt = getRefreshToken();
    if (!rt) return false;

    try {
      const res = await fetch(`${API_BASE}/auth/refresh`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ refresh_token: rt }),
      });
      if (!res.ok) return false;

      const data = await res.json();
      setToken(data.token);
      return true;
    } catch {
      return false;
    } finally {
      refreshPromise = null;
    }
  })();

  return refreshPromise;
}

// Authenticated fetch wrapper
async function apiFetch(url: string, options: RequestInit = {}): Promise<Response> {
  const token = getToken();
  const headers: Record<string, string> = {
    ...(options.headers as Record<string, string> || {}),
  };

  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  if (options.body && !headers['Content-Type']) {
    headers['Content-Type'] = 'application/json';
  }

  const res = await fetch(url, { ...options, headers });

  if (res.status === 401) {
    // Try refreshing the token
    const refreshed = await tryRefreshToken();
    if (refreshed) {
      // Retry the original request with the new token
      headers['Authorization'] = `Bearer ${getToken()}`;
      return fetch(url, { ...options, headers });
    }

    // Refresh failed - fully logged out
    clearToken();
    emitAuthExpired();
    throw new Error('인증이 만료되었습니다. 다시 로그인해주세요.');
  }

  return res;
}

// ============ Auth API ============

export async function checkSetup(): Promise<{ initialized: boolean }> {
  const res = await fetch(`${API_BASE}/auth/setup`);
  if (!res.ok) throw new Error('Failed to check setup');
  return res.json();
}

export async function login(req: LoginRequest): Promise<AuthResponse> {
  const res = await fetch(`${API_BASE}/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(req),
  });
  if (!res.ok) {
    const error = await res.json();
    throw new Error(error.error || '로그인에 실패했습니다');
  }
  const data: AuthResponse = await res.json();
  setToken(data.token);
  setRefreshToken(data.refresh_token);
  localStorage.setItem('user', JSON.stringify(data.user));
  return data;
}

export async function register(req: RegisterRequest): Promise<AuthResponse> {
  const res = await fetch(`${API_BASE}/auth/register`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(req),
  });
  if (!res.ok) {
    const error = await res.json();
    throw new Error(error.error || '회원가입에 실패했습니다');
  }
  const data: AuthResponse = await res.json();
  setToken(data.token);
  setRefreshToken(data.refresh_token);
  localStorage.setItem('user', JSON.stringify(data.user));
  return data;
}

export async function getMe(): Promise<User> {
  const res = await apiFetch(`${API_BASE}/auth/me`);
  if (!res.ok) throw new Error('Failed to fetch user');
  return res.json();
}

export async function createInviteCode(): Promise<InviteCode> {
  const res = await apiFetch(`${API_BASE}/admin/invite-codes`, {
    method: 'POST',
  });
  if (!res.ok) throw new Error('Failed to create invite code');
  return res.json();
}

export async function getInviteCodes(): Promise<InviteCode[]> {
  const res = await apiFetch(`${API_BASE}/admin/invite-codes`);
  if (!res.ok) throw new Error('Failed to fetch invite codes');
  return res.json();
}

export { getToken, clearToken };

// ============ Template API ============

export async function getTemplates(): Promise<Template[]> {
  const res = await apiFetch(`${API_BASE}/templates`);
  if (!res.ok) throw new Error('Failed to fetch templates');
  return res.json();
}

export async function createTemplate(name: string, style: string = '{}'): Promise<Template> {
  const res = await apiFetch(`${API_BASE}/templates`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, style }),
  });
  if (!res.ok) throw new Error('Failed to create template');
  return res.json();
}

export async function updateTemplate(id: number, name: string, style: string): Promise<void> {
  const res = await apiFetch(`${API_BASE}/templates/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, style }),
  });
  if (!res.ok) throw new Error('Failed to update template');
}

export async function deleteTemplate(id: number): Promise<void> {
  const res = await apiFetch(`${API_BASE}/templates/${id}`, { method: 'DELETE' });
  if (!res.ok) throw new Error('Failed to delete template');
}

// ============ Report API ============

export async function getReport(id: number): Promise<Report> {
  const res = await apiFetch(`${API_BASE}/reports/${id}`);
  if (!res.ok) throw new Error('Failed to fetch report');
  return res.json();
}

export async function saveReport(report: Omit<Report, 'id'>): Promise<Report> {
  const res = await apiFetch(`${API_BASE}/reports/save`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(report),
  });
  if (!res.ok) throw new Error('Failed to save report');
  return res.json();
}

export async function getReports(): Promise<Report[]> {
  const res = await apiFetch(`${API_BASE}/reports`);
  if (!res.ok) throw new Error('Failed to fetch reports');
  return res.json();
}

export async function getUsers(): Promise<User[]> {
  const res = await apiFetch(`${API_BASE}/users`);
  if (!res.ok) throw new Error('Failed to fetch users');
  return res.json();
}

export async function getMySubmission(teamId: number, reportDate: string): Promise<{ submitted: boolean; submission?: ReportSubmission }> {
  const res = await apiFetch(`${API_BASE}/teams/${teamId}/my-submission?report_date=${reportDate}`);
  if (!res.ok) return { submitted: false };
  return res.json();
}

// ============ Config API ============

export async function getConfig(): Promise<ConfigMap> {
  const res = await apiFetch(`${API_BASE}/config`);
  if (!res.ok) throw new Error('Failed to fetch config');
  return res.json();
}

export async function updateConfig(configs: ConfigMap): Promise<void> {
  const res = await apiFetch(`${API_BASE}/config`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ configs }),
  });
  if (!res.ok) throw new Error('Failed to update config');
}

// ============ Sync API ============

export async function syncGitHub(request: GitHubSyncRequest): Promise<SyncResult> {
  const res = await apiFetch(`${API_BASE}/sync/github`, {
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
  const res = await apiFetch(`${API_BASE}/sync/gitlab`, {
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
  const res = await apiFetch(`${API_BASE}/sync/jira`, {
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
  const res = await apiFetch(`${API_BASE}/sync/hiworks`, {
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

// ============ GitLab Projects API ============

export async function listGitLabProjects(): Promise<GitLabProject[]> {
  const res = await apiFetch(`${API_BASE}/gitlab/projects`);
  if (!res.ok) {
    const error = await res.json();
    throw new Error(error.error || 'GitLab 프로젝트 목록 조회에 실패했습니다');
  }
  return res.json();
}

// ============ AI Generate API ============

export async function generateAIReport(request: GenerateReportRequest): Promise<GenerateReportResponse> {
  const res = await apiFetch(`${API_BASE}/ai/generate`, {
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

// ============ Team API ============

export async function createTeam(name: string, description: string = ''): Promise<Team> {
  const res = await apiFetch(`${API_BASE}/teams`, {
    method: 'POST',
    body: JSON.stringify({ name, description }),
  });
  if (!res.ok) {
    const error = await res.json();
    throw new Error(error.error || '팀 생성에 실패했습니다');
  }
  return res.json();
}

export async function getMyTeams(): Promise<Team[]> {
  const res = await apiFetch(`${API_BASE}/teams`);
  if (!res.ok) throw new Error('팀 목록 조회에 실패했습니다');
  return res.json();
}

export async function getTeam(id: number): Promise<Team> {
  const res = await apiFetch(`${API_BASE}/teams/${id}`);
  if (!res.ok) throw new Error('팀 조회에 실패했습니다');
  return res.json();
}

export async function updateTeam(id: number, name: string, description: string): Promise<void> {
  const res = await apiFetch(`${API_BASE}/teams/${id}`, {
    method: 'PUT',
    body: JSON.stringify({ name, description }),
  });
  if (!res.ok) throw new Error('팀 수정에 실패했습니다');
}

export async function deleteTeam(id: number): Promise<void> {
  const res = await apiFetch(`${API_BASE}/teams/${id}`, { method: 'DELETE' });
  if (!res.ok) throw new Error('팀 삭제에 실패했습니다');
}

// ============ Team Member API ============

export async function addTeamMember(teamId: number, email: string, role: TeamRole = 'member', roleCode: RoleCode = 'S'): Promise<TeamMember> {
  const res = await apiFetch(`${API_BASE}/teams/${teamId}/members`, {
    method: 'POST',
    body: JSON.stringify({ email, role, role_code: roleCode }),
  });
  if (!res.ok) {
    const error = await res.json();
    throw new Error(error.error || '멤버 추가에 실패했습니다');
  }
  return res.json();
}

export async function getTeamMembers(teamId: number): Promise<TeamMember[]> {
  const res = await apiFetch(`${API_BASE}/teams/${teamId}/members`);
  if (!res.ok) throw new Error('멤버 목록 조회에 실패했습니다');
  return res.json();
}

export async function updateTeamMember(teamId: number, memberId: number, role: TeamRole, roleCode: RoleCode): Promise<void> {
  const res = await apiFetch(`${API_BASE}/teams/${teamId}/members/${memberId}`, {
    method: 'PUT',
    body: JSON.stringify({ role, role_code: roleCode }),
  });
  if (!res.ok) throw new Error('멤버 수정에 실패했습니다');
}

export async function removeTeamMember(teamId: number, memberId: number): Promise<void> {
  const res = await apiFetch(`${API_BASE}/teams/${teamId}/members/${memberId}`, { method: 'DELETE' });
  if (!res.ok) throw new Error('멤버 제거에 실패했습니다');
}

// ============ Submission API ============

export async function submitReport(teamId: number, reportId: number): Promise<ReportSubmission> {
  const res = await apiFetch(`${API_BASE}/teams/${teamId}/submit`, {
    method: 'POST',
    body: JSON.stringify({ report_id: reportId }),
  });
  if (!res.ok) {
    const error = await res.json();
    throw new Error(error.error || '보고서 제출에 실패했습니다');
  }
  return res.json();
}

export async function unsubmitReport(teamId: number, reportId: number): Promise<void> {
  const res = await apiFetch(`${API_BASE}/teams/${teamId}/submit/${reportId}`, { method: 'DELETE' });
  if (!res.ok) throw new Error('제출 취소에 실패했습니다');
}

// ============ Team Submissions / Consolidated API ============

export async function getTeamSubmissions(teamId: number, reportDate: string): Promise<TeamMemberWithSubmission[]> {
  const res = await apiFetch(`${API_BASE}/teams/${teamId}/submissions?report_date=${reportDate}`);
  if (!res.ok) throw new Error('제출 현황 조회에 실패했습니다');
  return res.json();
}

export async function getTeamMemberReport(teamId: number, reportId: number): Promise<Report> {
  const res = await apiFetch(`${API_BASE}/teams/${teamId}/reports/${reportId}`);
  if (!res.ok) throw new Error('보고서 조회에 실패했습니다');
  return res.json();
}

export async function updateTeamMemberReport(teamId: number, reportId: number, report: Omit<Report, 'id'>): Promise<void> {
  const res = await apiFetch(`${API_BASE}/teams/${teamId}/reports/${reportId}`, {
    method: 'PUT',
    body: JSON.stringify(report),
  });
  if (!res.ok) throw new Error('보고서 수정에 실패했습니다');
}

export async function getConsolidatedReport(teamId: number, reportDate: string): Promise<ConsolidatedReport> {
  const res = await apiFetch(`${API_BASE}/teams/${teamId}/consolidated?report_date=${reportDate}`);
  if (!res.ok) throw new Error('취합 데이터 조회에 실패했습니다');
  return res.json();
}

export async function summarizeConsolidatedReport(teamId: number, reportDate: string): Promise<{ this_week: Task[]; next_week: Task[]; summary: string }> {
  const res = await apiFetch(`${API_BASE}/teams/${teamId}/ai/summarize`, {
    method: 'POST',
    body: JSON.stringify({ report_date: reportDate }),
  });
  if (!res.ok) {
    const error = await res.json();
    throw new Error(error.error || 'AI 요약 생성에 실패했습니다');
  }
  return res.json();
}
