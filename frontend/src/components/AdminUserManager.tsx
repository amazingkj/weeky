import { useState, useEffect, useCallback } from 'react';
import { User } from '../types';
import { adminGetUsers, adminResetPassword } from '../services/api';

export default function AdminUserManager() {
  const [users, setUsers] = useState<User[]>([]);
  const [error, setError] = useState('');
  const [resetTarget, setResetTarget] = useState<User | null>(null);
  const [newPassword, setNewPassword] = useState('');
  const [isResetting, setIsResetting] = useState(false);
  const [successMsg, setSuccessMsg] = useState('');

  const fetchUsers = useCallback(async () => {
    try {
      const data = await adminGetUsers();
      setUsers(data);
    } catch {
      setError('사용자 목록을 불러올 수 없습니다');
    }
  }, []);

  useEffect(() => {
    fetchUsers();
  }, [fetchUsers]);

  const handleReset = async () => {
    if (!resetTarget || newPassword.length < 6) return;
    setIsResetting(true);
    setError('');
    try {
      await adminResetPassword(resetTarget.id, newPassword);
      setSuccessMsg(`${resetTarget.name}님의 비밀번호가 초기화되었습니다`);
      setResetTarget(null);
      setNewPassword('');
      setTimeout(() => setSuccessMsg(''), 3000);
    } catch {
      setError('비밀번호 초기화에 실패했습니다');
    } finally {
      setIsResetting(false);
    }
  };

  return (
    <div className="space-y-4">
      <h3 className="text-sm font-semibold text-neutral-900">사용자 관리</h3>

      {error && (
        <div className="text-sm text-red-600 bg-red-50 border border-red-200 rounded-lg px-3 py-2">
          {error}
        </div>
      )}
      {successMsg && (
        <div className="text-sm text-green-700 bg-green-50 border border-green-200 rounded-lg px-3 py-2">
          {successMsg}
        </div>
      )}

      {users.length === 0 ? (
        <p className="text-sm text-neutral-500">등록된 사용자가 없습니다.</p>
      ) : (
        <div className="space-y-2">
          {users.map(u => (
            <div
              key={u.id}
              className="flex items-center justify-between px-3 py-2.5 rounded-lg border border-neutral-200 bg-white"
            >
              <div className="flex items-center gap-3 min-w-0">
                <div className="w-8 h-8 rounded-full bg-neutral-100 flex items-center justify-center text-xs font-medium text-neutral-600 flex-shrink-0">
                  {u.name.charAt(0)}
                </div>
                <div className="min-w-0">
                  <div className="text-sm font-medium text-neutral-900 flex items-center gap-1.5">
                    {u.name}
                    {u.is_admin && (
                      <span className="text-[10px] px-1.5 py-0.5 bg-blue-50 text-blue-600 rounded font-medium">관리자</span>
                    )}
                  </div>
                  <div className="text-xs text-neutral-500 truncate">{u.email}</div>
                </div>
              </div>
              <button
                onClick={() => { setResetTarget(u); setNewPassword(''); setError(''); }}
                className="px-2.5 py-1.5 text-xs text-neutral-600 hover:text-neutral-900 border border-neutral-300 rounded-lg hover:bg-neutral-50 transition-colors flex-shrink-0"
              >
                비밀번호 초기화
              </button>
            </div>
          ))}
        </div>
      )}

      {resetTarget && (
        <div className="fixed inset-0 bg-black/30 flex items-center justify-center z-50" onClick={() => setResetTarget(null)}>
          <div className="bg-white rounded-xl shadow-xl border border-neutral-200 p-5 w-full max-w-sm mx-4" onClick={e => e.stopPropagation()}>
            <h4 className="text-sm font-semibold text-neutral-900 mb-1">비밀번호 초기화</h4>
            <p className="text-xs text-neutral-500 mb-4">
              <span className="font-medium text-neutral-700">{resetTarget.name}</span> ({resetTarget.email})
            </p>
            <input
              type="password"
              value={newPassword}
              onChange={e => setNewPassword(e.target.value)}
              placeholder="새 비밀번호 (6자 이상)"
              className="w-full px-3 py-2 text-sm border border-neutral-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent mb-3"
              autoFocus
            />
            <div className="flex justify-end gap-2">
              <button
                onClick={() => setResetTarget(null)}
                className="px-3 py-1.5 text-xs text-neutral-600 border border-neutral-300 rounded-lg hover:bg-neutral-50"
              >
                취소
              </button>
              <button
                onClick={handleReset}
                disabled={newPassword.length < 6 || isResetting}
                className="px-3 py-1.5 text-xs text-white bg-blue-600 rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {isResetting ? '처리 중...' : '초기화'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
