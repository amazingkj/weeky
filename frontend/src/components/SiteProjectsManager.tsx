import { useState, useEffect, useCallback } from 'react';
import { SiteProject, TeamMember } from '../types';
import { getSiteProjects, createSiteProject, updateSiteProject, deleteSiteProject, getTeamMembers } from '../services/api';
import Loading from './ui/Loading';

interface Props {
  teamId: number;
}

interface ProjectDraft {
  project_name: string;
  client_name: string;
  author_ids: number[];
}

const emptyDraft = (): ProjectDraft => ({ project_name: '', client_name: '', author_ids: [] });

export default function SiteProjectsManager({ teamId }: Props) {
  const [projects, setProjects] = useState<SiteProject[]>([]);
  const [members, setMembers] = useState<TeamMember[]>([]);
  const [loading, setLoading] = useState(true);
  const [draft, setDraft] = useState<ProjectDraft>(emptyDraft);
  const [editing, setEditing] = useState<SiteProject | null>(null);

  const reload = useCallback(async () => {
    setLoading(true);
    try {
      const [ps, ms] = await Promise.all([getSiteProjects(teamId), getTeamMembers(teamId)]);
      setProjects(ps);
      setMembers(ms);
    } finally {
      setLoading(false);
    }
  }, [teamId]);

  useEffect(() => { reload(); }, [reload]);

  const handleCreate = async () => {
    if (!draft.project_name.trim()) return;
    try {
      await createSiteProject(teamId, {
        project_name: draft.project_name.trim(),
        client_name: draft.client_name.trim(),
        author_ids: draft.author_ids,
      });
      setDraft(emptyDraft());
      reload();
    } catch (e: any) {
      alert(e?.message || '생성 실패');
    }
  };

  const handleSaveEdit = async () => {
    if (!editing || !editing.project_name.trim()) return;
    try {
      await updateSiteProject(teamId, editing.id, {
        project_name: editing.project_name.trim(),
        client_name: editing.client_name.trim(),
        is_active: editing.is_active,
        author_ids: (editing.authors || []).map((a) => a.user_id),
      });
      setEditing(null);
      reload();
    } catch (e: any) {
      alert(e?.message || '수정 실패');
    }
  };

  const handleDelete = async (pid: number) => {
    if (!confirm('이 사이트 프로젝트를 삭제하시겠습니까? 관련 보고서도 함께 삭제됩니다.')) return;
    try {
      await deleteSiteProject(teamId, pid);
      reload();
    } catch (e: any) {
      alert(e?.message || '삭제 실패');
    }
  };

  const toggleDraftAuthor = (userId: number) => {
    setDraft((d) => {
      const exists = d.author_ids.includes(userId);
      return { ...d, author_ids: exists ? d.author_ids.filter((x) => x !== userId) : [...d.author_ids, userId] };
    });
  };

  const toggleEditAuthor = (userId: number) => {
    setEditing((cur) => {
      if (!cur) return cur;
      const authors = cur.authors || [];
      const exists = authors.some((a) => a.user_id === userId);
      const member = members.find((m) => m.user_id === userId);
      const next = exists
        ? authors.filter((a) => a.user_id !== userId)
        : [...authors, { site_project_id: cur.id, user_id: userId, user_name: member?.user_name, user_email: member?.user_email, sort_order: authors.length }];
      return { ...cur, authors: next };
    });
  };

  if (loading) return <Loading text="사이트 프로젝트 로딩 중..." />;

  return (
    <div className="space-y-5">
      <p className="text-xs text-neutral-500">
        파견 사이트 보고서를 작성할 프로젝트와 작성자를 등록합니다. 등록된 작성자만 해당 사이트 보고서를 작성할 수 있습니다. 사이트 보고서는 취합 PPT의 본사 슬라이드 뒤에 편집 없이 그대로 추가됩니다.
      </p>

      {/* Create form */}
      <div className="bg-neutral-50 rounded-lg border border-neutral-200 p-4 space-y-3">
        <div className="text-xs font-semibold text-neutral-700">새 사이트 프로젝트 추가</div>
        <div className="grid grid-cols-2 gap-2">
          <input
            type="text"
            value={draft.project_name}
            onChange={(e) => setDraft({ ...draft, project_name: e.target.value })}
            placeholder="프로젝트명 (예: 한화손해보험 마이데이터, 유지보수)"
            className="px-2 py-1.5 text-sm border border-neutral-200 rounded focus:outline-none focus:border-neutral-400"
          />
          <input
            type="text"
            value={draft.client_name}
            onChange={(e) => setDraft({ ...draft, client_name: e.target.value })}
            placeholder="고객사명 (예: 한화손해보험)"
            className="px-2 py-1.5 text-sm border border-neutral-200 rounded focus:outline-none focus:border-neutral-400"
          />
        </div>
        <div>
          <div className="text-xs text-neutral-500 mb-1">작성자 선택 (다중)</div>
          <div className="flex flex-wrap gap-1.5">
            {members.length === 0 ? (
              <span className="text-xs text-neutral-400">팀원이 없습니다.</span>
            ) : members.map((m) => {
              const selected = draft.author_ids.includes(m.user_id);
              return (
                <button
                  key={m.user_id}
                  onClick={() => toggleDraftAuthor(m.user_id)}
                  className={`px-2 py-1 text-xs rounded border ${selected ? 'bg-neutral-900 text-white border-neutral-900' : 'bg-white text-neutral-600 border-neutral-200 hover:border-neutral-300'}`}
                >
                  {m.user_name}
                </button>
              );
            })}
          </div>
        </div>
        <button
          onClick={handleCreate}
          disabled={!draft.project_name.trim()}
          className="px-3 py-1.5 text-xs font-medium text-white bg-neutral-900 rounded hover:bg-neutral-800 disabled:opacity-40"
        >
          추가
        </button>
      </div>

      {/* List */}
      {projects.length === 0 ? (
        <p className="text-xs text-neutral-400 text-center py-4">등록된 사이트 프로젝트가 없습니다.</p>
      ) : (
        <div className="divide-y divide-neutral-100">
          {projects.map((p) => (
            <div key={p.id} className="py-3">
              {editing?.id === p.id ? (
                <div className="space-y-2 bg-neutral-50 p-3 rounded">
                  <div className="grid grid-cols-2 gap-2">
                    <input
                      type="text"
                      value={editing.project_name}
                      onChange={(e) => setEditing({ ...editing, project_name: e.target.value })}
                      className="px-2 py-1.5 text-sm border border-neutral-300 rounded focus:outline-none focus:border-neutral-400"
                    />
                    <input
                      type="text"
                      value={editing.client_name}
                      onChange={(e) => setEditing({ ...editing, client_name: e.target.value })}
                      className="px-2 py-1.5 text-sm border border-neutral-300 rounded focus:outline-none focus:border-neutral-400"
                    />
                  </div>
                  <div>
                    <div className="text-xs text-neutral-500 mb-1">작성자</div>
                    <div className="flex flex-wrap gap-1.5">
                      {members.map((m) => {
                        const selected = (editing.authors || []).some((a) => a.user_id === m.user_id);
                        return (
                          <button
                            key={m.user_id}
                            onClick={() => toggleEditAuthor(m.user_id)}
                            className={`px-2 py-1 text-xs rounded border ${selected ? 'bg-neutral-900 text-white border-neutral-900' : 'bg-white text-neutral-600 border-neutral-200 hover:border-neutral-300'}`}
                          >
                            {m.user_name}
                          </button>
                        );
                      })}
                    </div>
                  </div>
                  <label className="flex items-center gap-1.5 text-xs text-neutral-600">
                    <input
                      type="checkbox"
                      checked={editing.is_active}
                      onChange={(e) => setEditing({ ...editing, is_active: e.target.checked })}
                    />
                    활성
                  </label>
                  <div className="flex gap-2">
                    <button onClick={handleSaveEdit} className="px-3 py-1 text-xs font-medium text-white bg-neutral-900 rounded">저장</button>
                    <button onClick={() => setEditing(null)} className="px-3 py-1 text-xs text-neutral-600 hover:text-neutral-800">취소</button>
                  </div>
                </div>
              ) : (
                <div className="flex items-start justify-between gap-3">
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      <span className={`text-sm font-medium ${p.is_active ? 'text-neutral-900' : 'text-neutral-400 line-through'}`}>
                        {p.project_name}
                      </span>
                      {p.client_name && <span className="text-xs text-neutral-400">{p.client_name}</span>}
                      {!p.is_active && <span className="px-1.5 py-0.5 bg-neutral-100 text-neutral-400 text-[10px] rounded">비활성</span>}
                    </div>
                    <div className="text-xs text-neutral-500 mt-1">
                      작성자: {p.authors && p.authors.length > 0 ? p.authors.map((a) => a.user_name).join(', ') : <span className="text-neutral-400">미지정</span>}
                    </div>
                  </div>
                  <div className="flex gap-1">
                    <button onClick={() => setEditing(p)} className="p-1 text-neutral-400 hover:text-neutral-700" title="수정">
                      <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                      </svg>
                    </button>
                    <button onClick={() => handleDelete(p.id)} className="p-1 text-neutral-400 hover:text-red-500" title="삭제">
                      <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                      </svg>
                    </button>
                  </div>
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
