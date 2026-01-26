export interface Task {
  title: string;
  details?: string; // 진행 사항 상세 내용
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
  template_id: number;
}

export interface SyncItem {
  title: string;
  content?: string; // 메일 본문 등 상세 내용
  date: string;
  url: string;
  type: 'commit' | 'pr' | 'mr' | 'issue' | 'issue_done' | 'issue_todo' | 'email';
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
