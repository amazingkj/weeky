import { useState, useEffect, useCallback } from 'react';
import { Team, TeamMember, ReportSubmission, Report, TEAM_ROLE_LABELS, ROLE_CODE_LABELS } from '../types';
import { getMyTeams, getTeamMembers, getMySubmission, getMySubmissions, getReports, deleteTeam, updateTeam } from '../services/api';
import { useAuth } from '../contexts/AuthContext';
import TeamCreateModal from './TeamCreateModal';
import TeamMemberManager from './TeamMemberManager';
import TeamSubmissionPanel from './TeamSubmissionPanel';
import Loading from './ui/Loading';

export default function TeamPanel() {
  const { user } = useAuth();
  const [teams, setTeams] = useState<Team[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [selectedTeam, setSelectedTeam] = useState<Team | null>(null);
  const [myRole, setMyRole] = useState<string | null>(null);
  const [activeView, setActiveView] = useState<'members' | 'submissions'>('submissions');
  const [teamMembers, setTeamMembers] = useState<TeamMember[]>([]);
  const [mySubmissionStatus, setMySubmissionStatus] = useState<boolean>(false);
  const [submissionHistory, setSubmissionHistory] = useState<ReportSubmission[]>([]);
  const [historyLoading, setHistoryLoading] = useState(false);
  const [viewingReport, setViewingReport] = useState<Report | null>(null);
  const [viewingReportLoading, setViewingReportLoading] = useState(false);
  const [editingTeamName, setEditingTeamName] = useState(false);
  const [teamNameInput, setTeamNameInput] = useState('');
  const [teamDescInput, setTeamDescInput] = useState('');
  const [teamNameSaving, setTeamNameSaving] = useState(false);

  const fetchTeams = useCallback(async () => {
    try {
      const data = await getMyTeams();
      setTeams(data);
      if (data.length > 0 && !selectedTeam) {
        setSelectedTeam(data[0]);
      }
    } catch {
      // ignore
    } finally {
      setLoading(false);
    }
  }, [selectedTeam]);

  useEffect(() => { fetchTeams(); }, [fetchTeams]);

  // Fetch my role and team members when team is selected
  useEffect(() => {
    if (!selectedTeam || !user) return;
    getTeamMembers(selectedTeam.id).then((members: TeamMember[]) => {
      setTeamMembers(members);
      const me = members.find(m => m.user_id === user.id);
      setMyRole(me?.role || null);
    }).catch(() => { setMyRole(null); setTeamMembers([]); });

    // Check my submission status for current week
    const today = new Date().toISOString().split('T')[0];
    getMySubmission(selectedTeam.id, today).then((result) => {
      setMySubmissionStatus(result.submitted);
    }).catch(() => setMySubmissionStatus(false));

    // Fetch submission history
    setHistoryLoading(true);
    setViewingReport(null);
    getMySubmissions(selectedTeam.id).then((data) => {
      setSubmissionHistory(data);
    }).catch(() => setSubmissionHistory([])).finally(() => setHistoryLoading(false));
  }, [selectedTeam, user]);

  const handleTeamCreated = (team: Team) => {
    setTeams(prev => [team, ...prev]);
    setSelectedTeam(team);
    setMyRole('leader');
  };

  const handleTeamNameSave = async () => {
    if (!selectedTeam || !teamNameInput.trim()) return;
    setTeamNameSaving(true);
    try {
      await updateTeam(selectedTeam.id, teamNameInput.trim(), teamDescInput.trim());
      const updated = { ...selectedTeam, name: teamNameInput.trim(), description: teamDescInput.trim() };
      setSelectedTeam(updated);
      setTeams(prev => prev.map(t => t.id === updated.id ? updated : t));
      setEditingTeamName(false);
    } catch (err: any) {
      alert(err.message || '팀 이름 수정에 실패했습니다.');
    } finally {
      setTeamNameSaving(false);
    }
  };

  const isLeaderOrGroupLeader = myRole === 'leader' || myRole === 'group_leader' || user?.is_admin;

  if (loading) return <div className="py-16"><Loading text="팀 정보 로딩 중..." size="lg" /></div>;

  // No teams state
  if (teams.length === 0) {
    return (
      <div className="space-y-6">
        <div className="bg-white p-8 rounded-xl border border-neutral-200 text-center">
          <svg className="w-12 h-12 text-neutral-200 mx-auto mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1} d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
          </svg>
          <p className="text-neutral-500 text-sm mb-4">소속된 팀이 없습니다.</p>
          <button onClick={() => setShowCreateModal(true)}
            className="px-4 py-2 text-sm font-medium text-white bg-neutral-900 rounded-lg hover:bg-neutral-800 transition-colors">
            팀 생성하기
          </button>
        </div>
        <TeamCreateModal isOpen={showCreateModal} onClose={() => setShowCreateModal(false)} onCreated={handleTeamCreated} />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Team selector */}
      <div className="flex items-center gap-3">
        <div className="flex gap-2 flex-wrap flex-1">
          {teams.map(t => (
            <button key={t.id}
              onClick={() => { setSelectedTeam(t); setActiveView('submissions'); }}
              className={`px-3 py-1.5 text-xs font-medium rounded-lg border transition-colors ${
                selectedTeam?.id === t.id
                  ? 'bg-neutral-900 text-white border-neutral-900'
                  : 'bg-white text-neutral-600 border-neutral-200 hover:border-neutral-300'
              }`}>
              {t.name}
            </button>
          ))}
        </div>
        <button onClick={() => setShowCreateModal(true)}
          className="flex items-center gap-1 px-2.5 py-1.5 text-xs font-medium text-neutral-600 bg-neutral-100 hover:bg-neutral-200 rounded-lg transition-colors">
          <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
          </svg>
          새 팀
        </button>
      </div>

      {/* Team content */}
      {selectedTeam && (
        <div className="bg-white rounded-xl border border-neutral-200 overflow-hidden">
          {/* Team info header */}
          <div className="px-5 py-3 border-b border-neutral-100 flex items-center justify-between">
            <div className="flex-1 min-w-0">
              {editingTeamName ? (
                <div className="flex items-center gap-2">
                  <div className="flex-1 space-y-1">
                    <input
                      type="text"
                      value={teamNameInput}
                      onChange={e => setTeamNameInput(e.target.value)}
                      className="w-full text-sm font-semibold text-neutral-900 border border-neutral-300 rounded-lg px-2 py-1 focus:outline-none focus:border-neutral-500"
                      placeholder="팀 이름"
                      autoFocus
                      onKeyDown={e => {
                        if (e.key === 'Escape') setEditingTeamName(false);
                        if (e.key === 'Enter' && teamNameInput.trim()) {
                          e.preventDefault();
                          handleTeamNameSave();
                        }
                      }}
                    />
                    <input
                      type="text"
                      value={teamDescInput}
                      onChange={e => setTeamDescInput(e.target.value)}
                      className="w-full text-xs text-neutral-500 border border-neutral-200 rounded-lg px-2 py-1 focus:outline-none focus:border-neutral-400"
                      placeholder="설명 (선택)"
                      onKeyDown={e => {
                        if (e.key === 'Escape') setEditingTeamName(false);
                        if (e.key === 'Enter' && teamNameInput.trim()) {
                          e.preventDefault();
                          handleTeamNameSave();
                        }
                      }}
                    />
                  </div>
                  <button
                    onClick={handleTeamNameSave}
                    disabled={!teamNameInput.trim() || teamNameSaving}
                    className="p-1 text-green-600 hover:text-green-700 disabled:opacity-40 transition-colors" title="저장">
                    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                    </svg>
                  </button>
                  <button
                    onClick={() => setEditingTeamName(false)}
                    className="p-1 text-neutral-400 hover:text-neutral-600 transition-colors" title="취소">
                    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                    </svg>
                  </button>
                </div>
              ) : (
                <>
                  <h2 className="text-sm font-semibold text-neutral-900">{selectedTeam.name}</h2>
                  {selectedTeam.description && (
                    <p className="text-xs text-neutral-400 mt-0.5">{selectedTeam.description}</p>
                  )}
                </>
              )}
            </div>
            <div className="flex items-center gap-2">
              {myRole && (
                <span className="px-2 py-0.5 bg-neutral-100 text-neutral-600 rounded text-[10px] font-medium">
                  {myRole === 'leader' ? '팀장' : myRole === 'group_leader' ? '그룹장' : '팀원'}
                </span>
              )}
              {(myRole === 'leader' || user?.is_admin) && !editingTeamName && (
                <button
                  onClick={() => {
                    setTeamNameInput(selectedTeam.name);
                    setTeamDescInput(selectedTeam.description || '');
                    setEditingTeamName(true);
                  }}
                  className="p-1 text-neutral-400 hover:text-neutral-600 transition-colors" title="팀 이름 수정">
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                  </svg>
                </button>
              )}
              {(myRole === 'leader' || user?.is_admin) && (
                <button
                  onClick={async () => {
                    if (!confirm(`"${selectedTeam.name}" 팀을 삭제하시겠습니까?\n모든 멤버와 제출 데이터가 삭제됩니다.`)) return;
                    try {
                      await deleteTeam(selectedTeam.id);
                      setTeams(prev => prev.filter(t => t.id !== selectedTeam.id));
                      setSelectedTeam(teams.length > 1 ? teams.find(t => t.id !== selectedTeam.id) || null : null);
                    } catch (err: any) {
                      alert(err.message || '팀 삭제에 실패했습니다.');
                    }
                  }}
                  className="p-1 text-neutral-400 hover:text-red-500 transition-colors" title="팀 삭제">
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                  </svg>
                </button>
              )}
            </div>
          </div>

          {/* Sub-tabs for leader/group_leader */}
          {isLeaderOrGroupLeader && (
            <div className="px-5 pt-3 flex gap-0 border-b border-neutral-100">
              <button
                onClick={() => setActiveView('submissions')}
                className={`px-3 py-2 text-xs font-medium border-b-2 -mb-px transition-colors ${
                  activeView === 'submissions'
                    ? 'border-neutral-900 text-neutral-900'
                    : 'border-transparent text-neutral-500 hover:text-neutral-700'
                }`}>
                취합 현황
              </button>
              {(myRole === 'leader' || user?.is_admin) && (
                <button
                  onClick={() => setActiveView('members')}
                  className={`px-3 py-2 text-xs font-medium border-b-2 -mb-px transition-colors ${
                    activeView === 'members'
                      ? 'border-neutral-900 text-neutral-900'
                      : 'border-transparent text-neutral-500 hover:text-neutral-700'
                  }`}>
                  멤버 관리
                </button>
              )}
            </div>
          )}

          {/* Content */}
          <div className="p-5">
            {isLeaderOrGroupLeader && activeView === 'submissions' && (
              <TeamSubmissionPanel team={selectedTeam} />
            )}
            {(myRole === 'leader' || user?.is_admin) && activeView === 'members' && (
              <TeamMemberManager team={selectedTeam} />
            )}
            {myRole === 'member' && !user?.is_admin && (
              <div className="space-y-5">
                {/* My submission status */}
                <div className="flex items-center gap-3 p-4 rounded-lg border border-neutral-100 bg-neutral-50">
                  <div className={`w-10 h-10 rounded-full flex items-center justify-center ${
                    mySubmissionStatus ? 'bg-green-100' : 'bg-amber-100'
                  }`}>
                    {mySubmissionStatus ? (
                      <svg className="w-5 h-5 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                      </svg>
                    ) : (
                      <svg className="w-5 h-5 text-amber-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                      </svg>
                    )}
                  </div>
                  <div>
                    <div className="text-sm font-medium text-neutral-900">
                      이번 주 보고서: {mySubmissionStatus ? '제출완료' : '미제출'}
                    </div>
                    <div className="text-xs text-neutral-500">
                      {mySubmissionStatus
                        ? '보고서가 정상적으로 제출되었습니다.'
                        : '보고서 작성 탭에서 보고서를 작성 후 제출해주세요.'}
                    </div>
                  </div>
                </div>

                {/* Team members list */}
                <div>
                  <div className="flex items-center gap-2 mb-3">
                    <h4 className="text-sm font-semibold text-neutral-900">팀원 목록</h4>
                    <span className="px-1.5 py-0.5 bg-neutral-100 text-neutral-600 text-xs font-medium rounded">
                      {teamMembers.length}
                    </span>
                  </div>
                  <div className="divide-y divide-neutral-100">
                    {teamMembers.map((m) => (
                      <div key={m.id} className="flex items-center gap-3 py-2">
                        <div className="w-7 h-7 rounded-full bg-neutral-200 flex items-center justify-center text-xs font-medium text-neutral-600">
                          {(m.user_name || '?')[0]}
                        </div>
                        <div className="flex-1 min-w-0">
                          <div className="text-sm font-medium text-neutral-900 truncate">{m.user_name}</div>
                          <div className="text-xs text-neutral-400 truncate">{m.user_email}</div>
                        </div>
                        <span className="px-1.5 py-0.5 bg-neutral-100 text-neutral-500 rounded text-[10px] font-medium">
                          {TEAM_ROLE_LABELS[m.role]}
                        </span>
                        <span className="px-1.5 py-0.5 bg-neutral-100 text-neutral-500 rounded text-[10px] font-medium">
                          {m.role_code} ({ROLE_CODE_LABELS[m.role_code]})
                        </span>
                      </div>
                    ))}
                  </div>
                </div>

                {/* Submission history */}
                <div>
                  <div className="flex items-center gap-2 mb-3">
                    <h4 className="text-sm font-semibold text-neutral-900">제출 이력</h4>
                    <span className="px-1.5 py-0.5 bg-neutral-100 text-neutral-600 text-xs font-medium rounded">
                      {submissionHistory.length}
                    </span>
                  </div>
                  {historyLoading ? (
                    <Loading text="이력 로딩 중..." />
                  ) : submissionHistory.length === 0 ? (
                    <p className="text-xs text-neutral-400">제출 이력이 없습니다.</p>
                  ) : (
                    <div className="space-y-1">
                      {submissionHistory.map((sub) => (
                        <div key={sub.id}
                          className={`flex items-center justify-between px-3 py-2 rounded-lg border transition-colors ${
                            viewingReport && viewingReport.report_date === sub.report_date
                              ? 'bg-blue-50 border-blue-200'
                              : 'bg-white border-neutral-100 hover:border-neutral-200'
                          }`}>
                          <div className="flex items-center gap-3">
                            <span className="text-sm font-medium text-neutral-900">{sub.report_date}</span>
                            {sub.submitted_at && (
                              <span className="text-xs text-neutral-400">
                                {new Date(sub.submitted_at).toLocaleString('ko-KR', { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })}
                              </span>
                            )}
                            <span className="px-1.5 py-0.5 bg-green-100 text-green-700 rounded text-[10px] font-medium">
                              {sub.status === 'submitted' ? '제출' : sub.status}
                            </span>
                          </div>
                          <button
                            onClick={async () => {
                              if (viewingReport && viewingReport.report_date === sub.report_date) {
                                setViewingReport(null);
                                return;
                              }
                              setViewingReportLoading(true);
                              try {
                                const reports = await getReports();
                                const report = reports.find(r => r.report_date === sub.report_date);
                                setViewingReport(report || null);
                              } catch {
                                setViewingReport(null);
                              } finally {
                                setViewingReportLoading(false);
                              }
                            }}
                            disabled={viewingReportLoading}
                            className="text-xs text-neutral-600 hover:text-neutral-900 font-medium transition-colors disabled:opacity-40">
                            {viewingReport && viewingReport.report_date === sub.report_date ? '접기' : '보기'}
                          </button>
                        </div>
                      ))}
                    </div>
                  )}

                  {/* Viewing a past report */}
                  {viewingReportLoading && <div className="py-3"><Loading text="보고서 로딩 중..." /></div>}
                  {viewingReport && !viewingReportLoading && (
                    <div className="mt-3 bg-neutral-50 p-4 rounded-xl border border-neutral-200">
                      <div className="flex items-center justify-between mb-3">
                        <h5 className="text-sm font-semibold text-neutral-900">{viewingReport.report_date} 보고서</h5>
                        <button onClick={() => setViewingReport(null)}
                          className="p-1 text-neutral-400 hover:text-neutral-600 transition-colors">
                          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M6 18L18 6M6 6l12 12" />
                          </svg>
                        </button>
                      </div>
                      <div className="grid grid-cols-3 gap-2 mb-3 text-xs">
                        <div><span className="text-neutral-500">팀명:</span> {viewingReport.team_name}</div>
                        <div><span className="text-neutral-500">작성자:</span> {viewingReport.author_name}</div>
                        <div><span className="text-neutral-500">일자:</span> {viewingReport.report_date}</div>
                      </div>
                      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4 text-sm">
                        <div className="space-y-2">
                          <div className="font-semibold text-neutral-900 text-sm border-b border-neutral-300 pb-1">금주실적</div>
                          {viewingReport.this_week.length === 0 ? (
                            <p className="text-neutral-400 text-xs">없음</p>
                          ) : (
                            <div className="space-y-2">
                              {viewingReport.this_week.map((t, i) => (
                                <div key={i} className="bg-white rounded-md px-3 py-2 border border-neutral-200 text-xs">
                                  <div className="flex items-center justify-between">
                                    <span className="font-semibold text-neutral-900">{t.title}</span>
                                    <span className="text-neutral-500 font-medium">{t.progress}%</span>
                                  </div>
                                  {t.details && <div className="text-neutral-700 mt-1 whitespace-pre-line">{t.details}</div>}
                                </div>
                              ))}
                            </div>
                          )}
                        </div>
                        <div className="space-y-2">
                          <div className="font-semibold text-neutral-900 text-sm border-b border-neutral-300 pb-1">차주계획</div>
                          {viewingReport.next_week.length === 0 ? (
                            <p className="text-neutral-400 text-xs">없음</p>
                          ) : (
                            <div className="space-y-2">
                              {viewingReport.next_week.map((t, i) => (
                                <div key={i} className="bg-white rounded-md px-3 py-2 border border-neutral-200 text-xs">
                                  <div className="font-semibold text-neutral-900">{t.title}</div>
                                  {t.details && <div className="text-neutral-700 mt-1 whitespace-pre-line">{t.details}</div>}
                                </div>
                              ))}
                            </div>
                          )}
                        </div>
                      </div>
                    </div>
                  )}
                </div>
              </div>
            )}
          </div>
        </div>
      )}

      <TeamCreateModal isOpen={showCreateModal} onClose={() => setShowCreateModal(false)} onCreated={handleTeamCreated} />
    </div>
  );
}
