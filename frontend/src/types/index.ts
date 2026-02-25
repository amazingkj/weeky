export interface Task {
  title: string;
  details?: string; // 진행 사항
  description?: string; // 진행사항 상세내용
  due_date: string;
  progress: number; // 0-100
}

export interface TemplateStyle {
  primaryColor: string;
  secondaryColor: string;
  titleFontSize: number;
  bodyFontSize: number;
  showProgressBar: boolean;
  headerLayout: 'left' | 'center';
}

export const defaultTemplateStyle: TemplateStyle = {
  primaryColor: '#2563EB',
  secondaryColor: '#64748B',
  titleFontSize: 36,
  bodyFontSize: 11,
  showProgressBar: true,
  headerLayout: 'center',
};

export interface Template {
  id: number;
  name: string;
  style: string; // JSON string of TemplateStyle
  created_at: string;
}

export function parseTemplateStyle(styleJson: string): TemplateStyle {
  try {
    const parsed = JSON.parse(styleJson);
    return { ...defaultTemplateStyle, ...parsed };
  } catch {
    return defaultTemplateStyle;
  }
}

export interface Report {
  id?: number;
  team_name: string;
  author_name: string;
  report_date: string;
  this_week: Task[];
  next_week: Task[];
  issues: string;
  notes: string;
  next_issues: string;
  next_notes: string;
  template_id: number;
}

export interface SyncItem {
  title: string;
  content?: string; // 메일 본문 등 상세 내용
  date: string;
  url: string;
  type: 'commit' | 'pr' | 'mr' | 'issue' | 'issue_done' | 'issue_todo' | 'email';
  source?: string; // 출처 프로젝트명 (e.g., "group/project")
}

export interface SyncResult {
  source: 'github' | 'gitlab' | 'jira' | 'hiworks';
  items: SyncItem[];
  synced_at: string;
}

export interface GitHubSyncRequest {
  token?: string;
  owner: string;
  repo: string;
  start_date: string;
  end_date: string;
}

export interface GitLabSyncRequest {
  token?: string;
  base_url?: string;  // defaults to https://gitlab.com
  namespace: string;  // group or username
  project: string;    // project name
  start_date: string;
  end_date: string;
}

export interface JiraSyncRequest {
  base_url: string;
  email?: string;
  token?: string;
  start_date: string;
  end_date: string;
}

export interface HiworksSyncRequest {
  office_id?: string;   // 회사 ID (xxx.hiworks.com의 xxx)
  user_id?: string;     // 사용자 ID
  password?: string;    // 비밀번호
  start_date: string;
  end_date: string;
}

export interface ConfigMap {
  [key: string]: string;
}

export interface GitLabProject {
  id: number;
  name: string;
  full_path: string;
  namespace: string;
  project: string;
  web_url: string;
}

// Auth types
export interface User {
  id: number;
  email: string;
  name: string;
  is_admin: boolean;
  created_at: string;
}

export interface AuthResponse {
  token: string;
  refresh_token: string;
  user: User;
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface RegisterRequest {
  email: string;
  password: string;
  name: string;
  invite_code: string;
}

export interface InviteCode {
  id: number;
  code: string;
  created_by: number;
  used_by?: number;
  created_at: string;
  used_at?: string;
}

// ============ Team types ============

export type TeamRole = 'leader' | 'group_leader' | 'member';
export type RoleCode = 'S' | 'D' | 'G' | 'C' | 'B';

export const TEAM_ROLE_LABELS: Record<TeamRole, string> = {
  leader: '팀장',
  group_leader: '그룹장',
  member: '팀원',
};

export const ROLE_CODE_LABELS: Record<RoleCode, string> = {
  S: '사원',
  D: '대리',
  G: '과장',
  C: '차장',
  B: '부장',
};

export interface Team {
  id: number;
  name: string;
  description: string;
  created_by: number;
  created_at: string;
}

export interface TeamMember {
  id: number;
  team_id: number;
  user_id: number;
  role: TeamRole;
  role_code: RoleCode;
  joined_at: string;
  user_name?: string;
  user_email?: string;
}

export interface ReportSubmission {
  id: number;
  report_id: number;
  team_id: number;
  user_id: number;
  status: string;
  submitted_at?: string;
  created_at: string;
  user_name?: string;
  user_email?: string;
}

export interface TeamMemberWithSubmission extends TeamMember {
  submission?: ReportSubmission;
}

export interface MemberReportData {
  user_id: number;
  user_name: string;
  role_code: RoleCode;
  report?: Report;
}

export interface ConsolidatedReport {
  team: Team;
  report_date: string;
  members: MemberReportData[];
}
