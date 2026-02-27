import { useState, useCallback, useEffect } from 'react';
import { useConfig } from '../hooks';
import Loading from './ui/Loading';
import Alert from './ui/Alert';
import { ConfigMap, GitLabProject } from '../types';
import { listGitLabProjects, syncJira, testHiworks } from '../services/api';

interface FormData {
  gitlab_token: string;
  jira_base_url: string;
  jira_email: string;
  jira_token: string;
  hiworks_office_id: string;
  hiworks_user_id: string;
  hiworks_password: string;
  claude_api_key: string;
}

const initialFormData: FormData = {
  gitlab_token: '',
  jira_base_url: '', jira_email: '', jira_token: '',
  hiworks_office_id: '', hiworks_user_id: '', hiworks_password: '',
  claude_api_key: '',
};

export default function ConfigPanel() {
  const { config, isLoading, error, refetch } = useConfig();
  const [formData, setFormData] = useState<FormData>(initialFormData);
  const [isSaving, setIsSaving] = useState(false);
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);
  const [expandedSections, setExpandedSections] = useState<Set<string>>(new Set(['gitlab']));

  // GitLab project discovery
  const [gitlabProjects, setGitlabProjects] = useState<GitLabProject[]>([]);
  const [selectedProjects, setSelectedProjects] = useState<Set<string>>(new Set());
  const [isLoadingProjects, setIsLoadingProjects] = useState(false);
  const [projectError, setProjectError] = useState<string | null>(null);

  // 연결 검증 상태: null=미검증, true=성공, false=실패
  const [verifiedGitlab, setVerifiedGitlab] = useState<boolean | null>(null);
  const [verifiedJira, setVerifiedJira] = useState<boolean | null>(null);
  const [verifiedHiworks, setVerifiedHiworks] = useState<boolean | null>(null);
  const [testingSection, setTestingSection] = useState<string | null>(null);

  // Pre-fill form with saved non-sensitive values (sensitive ones return '***configured***')
  useEffect(() => {
    if (!config || Object.keys(config).length === 0) return;
    setFormData((prev) => {
      const updated = { ...prev };
      for (const key of Object.keys(updated) as (keyof FormData)[]) {
        const val = config[key];
        if (val && val !== '***configured***') {
          updated[key] = val;
        }
      }
      return updated;
    });

    // Load saved project selections
    const savedProjects = config.gitlab_projects;
    if (savedProjects) {
      try {
        const parsed: { namespace: string; project: string }[] = JSON.parse(savedProjects);
        setSelectedProjects(new Set(parsed.map(p => `${p.namespace}/${p.project}`)));
      } catch { /* ignore */ }
    }
  }, [config]);

  const handleFieldChange = useCallback((field: keyof FormData, value: string) => {
    setFormData((prev) => ({ ...prev, [field]: value }));
  }, []);

  const toggleSection = useCallback((section: string) => {
    setExpandedSections((prev) => {
      const newSet = new Set(prev);
      if (newSet.has(section)) { newSet.delete(section); } else { newSet.add(section); }
      return newSet;
    });
  }, []);

  const handleLoadProjects = useCallback(async () => {
    setIsLoadingProjects(true);
    setProjectError(null);
    try {
      const projects = await listGitLabProjects();
      setGitlabProjects(projects || []);
      if (!projects || projects.length === 0) {
        setProjectError('접근 가능한 프로젝트가 없습니다.');
      }
    } catch (err) {
      setProjectError(err instanceof Error ? err.message : '프로젝트 목록 조회에 실패했습니다.');
    } finally {
      setIsLoadingProjects(false);
    }
  }, []);

  const toggleProject = useCallback((fullPath: string) => {
    setSelectedProjects(prev => {
      const newSet = new Set(prev);
      if (newSet.has(fullPath)) { newSet.delete(fullPath); } else { newSet.add(fullPath); }
      return newSet;
    });
  }, []);

  const handleTestConnection = useCallback(async (section: string) => {
    setTestingSection(section);
    setMessage(null);
    const setVerified = (v: boolean) => {
      switch (section) {
        case 'gitlab': setVerifiedGitlab(v); break;
        case 'jira': setVerifiedJira(v); break;
        case 'hiworks': setVerifiedHiworks(v); break;
      }
    };
    try {
      switch (section) {
        case 'gitlab': {
          await listGitLabProjects();
          break;
        }
        case 'jira': {
          // base_url은 저장된 값 사용 (config), email/token은 빈값 → 서버 저장값 fallback
          const baseUrl = config.jira_base_url || formData.jira_base_url;
          if (!baseUrl) throw new Error('Jira Base URL이 저장되지 않았습니다. 먼저 설정을 저장하세요.');
          const today = new Date().toISOString().split('T')[0];
          await syncJira({ base_url: baseUrl, email: '', token: '', start_date: today, end_date: today });
          break;
        }
        case 'hiworks': {
          await testHiworks();
          break;
        }
      }
      setVerified(true);
    } catch (err: any) {
      setVerified(false);
      setMessage({ type: 'error', text: `${section} 연결 실패: ${err.message}` });
    } finally {
      setTestingSection(null);
    }
  }, [config, formData.jira_base_url]);

  const handleSave = async () => {
    setIsSaving(true);
    setMessage(null);
    try {
      const configs: ConfigMap = {};
      Object.entries(formData).forEach(([key, value]) => {
        if (value.trim()) configs[key] = value.trim();
      });

      // Save selected GitLab projects as JSON
      if (selectedProjects.size > 0) {
        const projectList = Array.from(selectedProjects).map(fp => {
          const parts = fp.split('/');
          const project = parts.pop()!;
          const namespace = parts.join('/');
          return { namespace, project };
        });
        configs.gitlab_projects = JSON.stringify(projectList);
      }

      if (Object.keys(configs).length === 0) {
        setMessage({ type: 'error', text: '저장할 설정이 없습니다.' });
        return;
      }
      const { updateConfig } = await import('../services/api');
      await updateConfig(configs);
      setMessage({ type: 'success', text: '설정이 저장되었습니다.' });
      // 저장 후 검증 상태 초기화 (새 값으로 재검증 필요)
      setVerifiedGitlab(null);
      setVerifiedJira(null);
      setVerifiedHiworks(null);
      // Clear only sensitive fields (non-sensitive will be re-filled from config by useEffect)
      setFormData((prev) => ({
        ...prev,
        gitlab_token: '',
        jira_email: '',
        jira_token: '',
        hiworks_user_id: '',
        hiworks_password: '',
        claude_api_key: '',
      }));
      refetch();
    } catch {
      setMessage({ type: 'error', text: '저장에 실패했습니다.' });
    } finally {
      setIsSaving(false);
    }
  };

  const isConfigured = useCallback((key: string) => {
    const val = config[key];
    return val !== undefined && val !== '';
  }, [config]);
  const getConfiguredCount = useCallback((keys: string[]) => keys.filter(key => isConfigured(key)).length, [isConfigured]);

  if (isLoading) return <div className="py-16"><Loading text="설정을 불러오는 중..." size="lg" /></div>;
  if (error) return <Alert type="error">{error}</Alert>;

  const hasGitLabToken = isConfigured('gitlab_token');

  return (
    <div className="space-y-4">
      {message ? (
        <Alert type={message.type} onClose={() => setMessage(null)}>{message.text}</Alert>
      ) : null}

      <ConfigSection
        title="GitLab" description="커밋, MR 정보를 가져옵니다"
        expanded={expandedSections.has('gitlab')} onToggle={() => toggleSection('gitlab')}
        configuredCount={hasGitLabToken && selectedProjects.size > 0 ? 2 : hasGitLabToken ? 1 : 0}
        totalCount={2} verified={verifiedGitlab}
      >
        <ConfigInput label="Personal Access Token" type="password"
          value={formData.gitlab_token} onChange={(v) => handleFieldChange('gitlab_token', v)}
          placeholder={hasGitLabToken ? '새 토큰으로 변경하려면 입력' : 'glpat-xxxxx...'}
          configured={hasGitLabToken}
          helpText="User Settings > Access Tokens에서 발급 (read_api 권한)" />
        <div className="flex items-center gap-2 text-xs text-neutral-400">
          <span className="font-medium text-neutral-500">GitLab URL</span>
          <span>https://gitlab.direa.synology.me</span>
        </div>

        {/* Project Discovery */}
        <div>
          <div className="flex items-center justify-between mb-2">
            <label className="text-xs font-medium text-neutral-500">프로젝트 선택</label>
            <button
              type="button"
              onClick={handleLoadProjects}
              disabled={!hasGitLabToken || isLoadingProjects}
              className="flex items-center gap-1 px-2.5 py-1 text-[11px] font-medium rounded-md border
                         border-neutral-200 text-neutral-600 hover:border-neutral-300 hover:bg-neutral-50
                         disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
            >
              {isLoadingProjects ? spinner : refreshIcon}
              {isLoadingProjects ? '불러오는 중...' : '프로젝트 불러오기'}
            </button>
          </div>

          {!hasGitLabToken && (
            <p className="text-[10px] text-neutral-400">토큰을 먼저 저장하면 프로젝트 목록을 불러올 수 있습니다.</p>
          )}

          {projectError && (
            <p className="text-[11px] text-red-500 mb-2">{projectError}</p>
          )}

          {/* Selected projects summary */}
          {selectedProjects.size > 0 && gitlabProjects.length === 0 && (
            <div className="flex flex-wrap gap-1.5 mb-2">
              {Array.from(selectedProjects).map(fp => (
                <span key={fp} className="inline-flex items-center gap-1 px-2 py-1 bg-neutral-100 text-neutral-700 text-[11px] rounded-md">
                  {fp}
                  <button type="button" onClick={() => toggleProject(fp)}
                    className="text-neutral-400 hover:text-red-500 transition-colors">
                    {closeIconSmall}
                  </button>
                </span>
              ))}
            </div>
          )}

          {/* Project list */}
          {gitlabProjects.length > 0 && (
            <div className="max-h-48 overflow-y-auto border border-neutral-200 rounded-lg divide-y divide-neutral-100">
              {gitlabProjects.map(p => {
                const isSelected = selectedProjects.has(p.full_path);
                return (
                  <label key={p.id}
                    className={`flex items-center gap-2.5 px-3 py-2 cursor-pointer hover:bg-neutral-50 transition-colors ${
                      isSelected ? 'bg-neutral-50' : ''
                    }`}
                  >
                    <input type="checkbox" checked={isSelected}
                      onChange={() => toggleProject(p.full_path)}
                      className="rounded border-neutral-300 text-neutral-900 focus:ring-neutral-500 w-3.5 h-3.5" />
                    <div className="min-w-0 flex-1">
                      <div className="text-xs font-medium text-neutral-800 truncate">{p.name}</div>
                      <div className="text-[10px] text-neutral-400 truncate">{p.full_path}</div>
                    </div>
                  </label>
                );
              })}
            </div>
          )}

          {selectedProjects.size > 0 && (
            <p className="text-[10px] text-neutral-500 mt-1.5">{selectedProjects.size}개 프로젝트 선택됨</p>
          )}
        </div>
        {hasGitLabToken && <TestConnectionButton section="gitlab" testing={testingSection} verified={verifiedGitlab} onTest={handleTestConnection} />}
      </ConfigSection>

      <ConfigSection
        title="Jira" description="이슈 정보를 가져옵니다"
        expanded={expandedSections.has('jira')} onToggle={() => toggleSection('jira')}
        configuredCount={getConfiguredCount(['jira_base_url', 'jira_email', 'jira_token'])}
        totalCount={3} verified={verifiedJira}
      >
        <ConfigInput label="Base URL" type="url"
          value={formData.jira_base_url} onChange={(v) => handleFieldChange('jira_base_url', v)}
          placeholder="https://your-domain.atlassian.net" configured={isConfigured('jira_base_url')} />
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <ConfigInput label="Email" type="email" value={formData.jira_email}
            onChange={(v) => handleFieldChange('jira_email', v)}
            placeholder="your@email.com" configured={isConfigured('jira_email')} />
          <ConfigInput label="API Token" type="password" value={formData.jira_token}
            onChange={(v) => handleFieldChange('jira_token', v)}
            placeholder="Atlassian API Token" configured={isConfigured('jira_token')} />
        </div>
        {getConfiguredCount(['jira_base_url', 'jira_email', 'jira_token']) === 3 && <TestConnectionButton section="jira" testing={testingSection} verified={verifiedJira} onTest={handleTestConnection} />}
      </ConfigSection>

      <ConfigSection
        title="Hiworks" description="보낸 메일 정보를 가져옵니다"
        expanded={expandedSections.has('hiworks')} onToggle={() => toggleSection('hiworks')}
        configuredCount={getConfiguredCount(['hiworks_office_id', 'hiworks_user_id', 'hiworks_password'])}
        totalCount={3} verified={verifiedHiworks}
      >
        <ConfigInput label="회사 ID" value={formData.hiworks_office_id}
          onChange={(v) => handleFieldChange('hiworks_office_id', v)}
          placeholder={isConfigured('hiworks_office_id') ? '변경하려면 입력' : 'xxx.hiworks.com의 xxx'}
          configured={isConfigured('hiworks_office_id')} />
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <ConfigInput label="사용자 ID" value={formData.hiworks_user_id}
            onChange={(v) => handleFieldChange('hiworks_user_id', v)}
            placeholder="user_id" configured={isConfigured('hiworks_user_id')} />
          <ConfigInput label="비밀번호" type="password" value={formData.hiworks_password}
            onChange={(v) => handleFieldChange('hiworks_password', v)}
            placeholder="비밀번호" configured={isConfigured('hiworks_password')} />
        </div>
        <p className="text-[10px] text-neutral-400">비밀번호는 AES-256으로 암호화되어 저장됩니다.</p>
        {getConfiguredCount(['hiworks_office_id', 'hiworks_user_id', 'hiworks_password']) === 3 && <TestConnectionButton section="hiworks" testing={testingSection} verified={verifiedHiworks} onTest={handleTestConnection} />}
      </ConfigSection>

      <ConfigSection
        title="Claude AI" description="AI 기반 보고서 자동 생성"
        expanded={expandedSections.has('claude')} onToggle={() => toggleSection('claude')}
        configuredCount={getConfiguredCount(['claude_api_key'])}
        totalCount={1}
      >
        <ConfigInput label="API Key" type="password" value={formData.claude_api_key}
          onChange={(v) => handleFieldChange('claude_api_key', v)}
          placeholder={isConfigured('claude_api_key') ? '새 키로 변경하려면 입력' : 'sk-ant-api03-...'}
          configured={isConfigured('claude_api_key')}
          helpText="console.anthropic.com에서 발급" />
      </ConfigSection>

      <div className="flex justify-end pt-2">
        <button onClick={handleSave} disabled={isSaving}
          className="px-5 py-2.5 bg-neutral-900 text-white text-sm font-medium rounded-lg
                     hover:bg-neutral-800 disabled:opacity-40 transition-colors flex items-center gap-2"
        >
          {isSaving ? spinner : saveIcon}
          {isSaving ? '저장 중...' : '설정 저장'}
        </button>
      </div>
    </div>
  );
}

// Sub-components

interface ConfigSectionProps {
  title: string;
  description: string;
  expanded: boolean;
  onToggle: () => void;
  configuredCount: number;
  totalCount: number;
  verified?: boolean | null; // null = not tested, true = success, false = fail
  children: React.ReactNode;
}

function ConfigSection({ title, description, expanded, onToggle, configuredCount, totalCount, verified, children }: ConfigSectionProps) {
  const isFullySaved = configuredCount === totalCount;

  return (
    <section className="bg-white rounded-xl border border-neutral-200 overflow-hidden">
      <button
        onClick={onToggle}
        className="w-full px-5 py-4 flex items-center justify-between hover:bg-neutral-50 transition-colors"
      >
        <div className="text-left">
          <div className="flex items-center gap-2">
            <h3 className="text-sm font-semibold text-neutral-900">{title}</h3>
            {verified === true ? (
              <span className="flex items-center gap-1 px-1.5 py-0.5 bg-green-100 text-green-700 text-[10px] font-medium rounded">
                {checkIcon} 연결 확인
              </span>
            ) : verified === false ? (
              <span className="flex items-center gap-1 px-1.5 py-0.5 bg-red-100 text-red-600 text-[10px] font-medium rounded">
                {warnIcon} 연결 실패
              </span>
            ) : isFullySaved ? (
              <span className="flex items-center gap-1 px-1.5 py-0.5 bg-neutral-100 text-neutral-500 text-[10px] font-medium rounded">
                {savedIcon} 저장됨
              </span>
            ) : configuredCount > 0 ? (
              <span className="px-1.5 py-0.5 bg-neutral-100 text-neutral-500 text-[10px] font-medium rounded">
                {configuredCount}/{totalCount}
              </span>
            ) : (
              <span className="px-1.5 py-0.5 bg-neutral-50 text-neutral-400 text-[10px] font-medium rounded">
                미설정
              </span>
            )}
          </div>
          <p className="text-xs text-neutral-400 mt-0.5">{description}</p>
        </div>
        <span className={`text-neutral-400 transition-transform duration-150 ${expanded ? 'rotate-180' : ''}`}>
          {chevronIcon}
        </span>
      </button>

      {expanded ? (
        <div className="px-5 pb-5 pt-1 border-t border-neutral-100">
          <div className="space-y-3">{children}</div>
        </div>
      ) : null}
    </section>
  );
}

interface ConfigInputProps {
  label: string;
  type?: 'text' | 'password' | 'email' | 'url';
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  configured?: boolean;
  helpText?: string;
}

function ConfigInput({ label, type = 'text', value, onChange, placeholder, configured, helpText }: ConfigInputProps) {
  const [showPassword, setShowPassword] = useState(false);
  const inputType = type === 'password' && showPassword ? 'text' : type;

  return (
    <div>
      <label className="flex items-center gap-2 text-xs font-medium text-neutral-500 mb-1.5">
        {label}
        {configured ? (
          <span className="flex items-center gap-0.5 text-neutral-400 text-[10px]">{savedIcon} 저장됨</span>
        ) : null}
      </label>
      <div className="relative">
        <input
          type={inputType} value={value} onChange={(e) => onChange(e.target.value)} placeholder={placeholder}
          className="input pr-9"
        />
        {type === 'password' ? (
          <button type="button" onClick={() => setShowPassword(!showPassword)}
            className="absolute right-2.5 top-1/2 -translate-y-1/2 text-neutral-400 hover:text-neutral-600">
            {showPassword ? eyeOffIcon : eyeIcon}
          </button>
        ) : null}
      </div>
      {helpText ? <p className="text-[10px] text-neutral-400 mt-1">{helpText}</p> : null}
    </div>
  );
}

// Test connection button
interface TestConnectionButtonProps {
  section: string;
  testing: string | null;
  verified: boolean | null;
  onTest: (section: string) => void;
}

function TestConnectionButton({ section, testing, verified, onTest }: TestConnectionButtonProps) {
  const isTesting = testing === section;
  return (
    <div className="flex items-center gap-2 pt-1">
      <button type="button" onClick={() => onTest(section)} disabled={isTesting}
        className="flex items-center gap-1.5 px-2.5 py-1.5 text-[11px] font-medium rounded-md border
                   border-neutral-200 text-neutral-600 hover:border-neutral-300 hover:bg-neutral-50
                   disabled:opacity-40 disabled:cursor-not-allowed transition-colors">
        {isTesting ? spinner : plugIcon}
        {isTesting ? '테스트 중...' : '연결 테스트'}
      </button>
      {verified === true && <span className="text-[11px] text-green-600 font-medium flex items-center gap-1">{checkIcon} 연결 성공</span>}
      {verified === false && <span className="text-[11px] text-red-500 font-medium flex items-center gap-1">{warnIcon} 연결 실패 - 설정을 확인하세요</span>}
    </div>
  );
}

// Icons
const refreshIcon = (
  <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
  </svg>
);
const closeIconSmall = (
  <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
  </svg>
);
const checkIcon = (
  <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5} d="M5 13l4 4L19 7" />
  </svg>
);
const savedIcon = (
  <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 7H5a2 2 0 00-2 2v9a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-3m-1 4l-3 3m0 0l-3-3m3 3V4" />
  </svg>
);
const warnIcon = (
  <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
  </svg>
);
const plugIcon = (
  <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M13 10V3L4 14h7v7l9-11h-7z" />
  </svg>
);
const chevronIcon = (
  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
  </svg>
);
const saveIcon = (
  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M8 7H5a2 2 0 00-2 2v9a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-3m-1 4l-3 3m0 0l-3-3m3 3V4" />
  </svg>
);
const eyeIcon = (
  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
  </svg>
);
const eyeOffIcon = (
  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21" />
  </svg>
);
const spinner = (
  <svg className="animate-spin w-4 h-4" viewBox="0 0 24 24">
    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" fill="none"/>
    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"/>
  </svg>
);
