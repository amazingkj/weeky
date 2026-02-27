import { useState, useEffect, useCallback } from 'react';
import { Team, TeamMember, TeamRole, RoleCode, User, TEAM_ROLE_LABELS, ROLE_CODE_LABELS } from '../types';
import { getTeamMembers, addTeamMember, updateTeamMember, removeTeamMember, getUsers } from '../services/api';

interface TeamMemberManagerProps {
  team: Team;
}

const ROLES: TeamRole[] = ['leader', 'group_leader', 'member'];
const ROLE_CODES: RoleCode[] = ['S', 'D', 'G', 'C', 'B'];

export default function TeamMemberManager({ team }: TeamMemberManagerProps) {
  const [members, setMembers] = useState<TeamMember[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [allUsers, setAllUsers] = useState<User[]>([]);
  const [selectedUserId, setSelectedUserId] = useState<string>('');
  const [newRole, setNewRole] = useState<TeamRole>('member');
  const [newRoleCode, setNewRoleCode] = useState<RoleCode>('S');
  const [adding, setAdding] = useState(false);

  const fetchMembers = useCallback(async () => {
    try {
      const data = await getTeamMembers(team.id);
      setMembers(data);
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }, [team.id]);

  useEffect(() => { fetchMembers(); }, [fetchMembers]);

  // Load all registered users
  useEffect(() => {
    getUsers().then(setAllUsers).catch(() => {});
  }, []);

  // Filter out users already in the team
  const availableUsers = allUsers.filter(
    u => !members.some(m => m.user_id === u.id)
  );

  const handleAdd = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedUserId) return;
    const user = allUsers.find(u => u.id === Number(selectedUserId));
    if (!user) return;
    setAdding(true);
    setError(null);
    try {
      await addTeamMember(team.id, user.email, newRole, newRoleCode);
      setSelectedUserId('');
      setNewRole('member');
      setNewRoleCode('S');
      await fetchMembers();
    } catch (err: any) {
      setError(err.message);
    } finally {
      setAdding(false);
    }
  };

  const handleUpdateRole = async (member: TeamMember, role: TeamRole) => {
    try {
      await updateTeamMember(team.id, member.id, role, member.role_code);
      await fetchMembers();
    } catch (err: any) {
      setError(err.message);
    }
  };

  const handleUpdateRoleCode = async (member: TeamMember, roleCode: RoleCode) => {
    try {
      await updateTeamMember(team.id, member.id, member.role, roleCode);
      await fetchMembers();
    } catch (err: any) {
      setError(err.message);
    }
  };

  const handleUpdateName = async (member: TeamMember, name: string) => {
    const trimmed = name.trim();
    if (!trimmed || trimmed === member.user_name) return;
    try {
      await updateTeamMember(team.id, member.id, member.role, member.role_code, trimmed);
      await fetchMembers();
    } catch (err: any) {
      setError(err.message);
    }
  };

  const handleRemove = async (member: TeamMember) => {
    if (!confirm(`${member.user_name}님을 팀에서 제거하시겠습니까?`)) return;
    try {
      await removeTeamMember(team.id, member.id);
      await fetchMembers();
    } catch (err: any) {
      setError(err.message);
    }
  };

  if (loading) return <div className="text-xs text-neutral-400 py-4 text-center">로딩 중...</div>;

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2 mb-3">
        <h3 className="text-sm font-semibold text-neutral-900">멤버 관리</h3>
        <span className="px-1.5 py-0.5 bg-neutral-100 text-neutral-600 text-xs font-medium rounded">{members.length}</span>
      </div>

      {error && (
        <div className="p-2 bg-red-50 border border-red-200 rounded-lg text-xs text-red-700">{error}
          <button onClick={() => setError(null)} className="ml-2 underline">닫기</button>
        </div>
      )}

      {/* Add member form */}
      <form onSubmit={handleAdd} className="flex flex-wrap gap-2 items-end">
        <div className="flex-1 min-w-[180px]">
          <label className="block text-xs text-neutral-500 mb-1">사용자</label>
          <select value={selectedUserId} onChange={(e) => setSelectedUserId(e.target.value)}
            className="input text-xs py-[7px]">
            <option value="">사용자 선택</option>
            {availableUsers.map(u => (
              <option key={u.id} value={u.id}>{u.name} ({u.email})</option>
            ))}
          </select>
        </div>
        <div>
          <label className="block text-xs text-neutral-500 mb-1">역할</label>
          <select value={newRole} onChange={(e) => setNewRole(e.target.value as TeamRole)}
            className="input text-xs py-[7px]">
            {ROLES.map(r => <option key={r} value={r}>{TEAM_ROLE_LABELS[r]}</option>)}
          </select>
        </div>
        <div>
          <label className="block text-xs text-neutral-500 mb-1">직급</label>
          <select value={newRoleCode} onChange={(e) => setNewRoleCode(e.target.value as RoleCode)}
            className="input text-xs py-[7px]">
            {ROLE_CODES.map(rc => <option key={rc} value={rc}>{rc} ({ROLE_CODE_LABELS[rc]})</option>)}
          </select>
        </div>
        <button type="submit" disabled={adding}
          className="px-3 py-1.5 text-xs font-medium text-white bg-neutral-900 rounded-lg hover:bg-neutral-800 disabled:opacity-40 transition-colors">
          {adding ? '추가 중...' : '추가'}
        </button>
      </form>

      {/* Member list */}
      <div className="divide-y divide-neutral-100">
        {members.map((m) => (
          <div key={m.id} className="group flex items-center gap-3 py-2.5">
            <div className="flex-1 min-w-0">
              <input
                type="text"
                defaultValue={m.user_name}
                onBlur={(e) => handleUpdateName(m, e.target.value)}
                onKeyDown={(e) => { if (e.key === 'Enter') (e.target as HTMLInputElement).blur(); }}
                className="text-sm font-medium text-neutral-900 bg-transparent border-b border-transparent hover:border-neutral-300 focus:border-neutral-400 focus:outline-none w-full truncate transition-colors"
              />
              <div className="text-xs text-neutral-400 truncate">{m.user_email}</div>
            </div>
            <select value={m.role} onChange={(e) => handleUpdateRole(m, e.target.value as TeamRole)}
              className="text-xs border border-neutral-200 rounded-md px-2 py-1 bg-white">
              {ROLES.map(r => <option key={r} value={r}>{TEAM_ROLE_LABELS[r]}</option>)}
            </select>
            <select value={m.role_code} onChange={(e) => handleUpdateRoleCode(m, e.target.value as RoleCode)}
              className="text-xs border border-neutral-200 rounded-md px-2 py-1 bg-white">
              {ROLE_CODES.map(rc => <option key={rc} value={rc}>{rc}</option>)}
            </select>
            <button onClick={() => handleRemove(m)}
              className="opacity-0 group-hover:opacity-100 p-1 text-neutral-400 hover:text-red-500 transition-all" title="제거">
              <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
              </svg>
            </button>
          </div>
        ))}
      </div>
    </div>
  );
}
