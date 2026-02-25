import { useState } from 'react';
import { Team } from '../types';
import { createTeam } from '../services/api';

interface TeamCreateModalProps {
  isOpen: boolean;
  onClose: () => void;
  onCreated: (team: Team) => void;
}

export default function TeamCreateModal({ isOpen, onClose, onCreated }: TeamCreateModalProps) {
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  if (!isOpen) return null;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) {
      setError('팀 이름을 입력해주세요.');
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const team = await createTeam(name.trim(), description.trim());
      onCreated(team);
      setName('');
      setDescription('');
      onClose();
    } catch (err: any) {
      setError(err.message || '팀 생성에 실패했습니다.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/30" onClick={onClose} />
      <div className="relative bg-white rounded-xl border border-neutral-200 shadow-lg w-full max-w-md mx-4 p-6">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-sm font-semibold text-neutral-900">팀 생성</h3>
          <button onClick={onClose} className="p-1 text-neutral-400 hover:text-neutral-600 transition-colors">
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        {error && (
          <div className="mb-3 p-2 bg-red-50 border border-red-200 rounded-lg text-xs text-red-700">{error}</div>
        )}

        <form onSubmit={handleSubmit} className="space-y-3">
          <div>
            <label className="block text-xs font-medium text-neutral-500 mb-1">팀 이름 *</label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="예: CruzAPIM팀"
              className="input"
              autoFocus
            />
          </div>
          <div>
            <label className="block text-xs font-medium text-neutral-500 mb-1">설명</label>
            <input
              type="text"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="팀 설명 (선택)"
              className="input"
            />
          </div>
          <div className="flex justify-end gap-2 pt-2">
            <button type="button" onClick={onClose}
              className="px-3 py-1.5 text-xs font-medium text-neutral-500 bg-neutral-100 rounded-lg hover:bg-neutral-200 transition-colors">
              취소
            </button>
            <button type="submit" disabled={loading}
              className="px-3 py-1.5 text-xs font-medium text-white bg-neutral-900 rounded-lg hover:bg-neutral-800 disabled:opacity-40 transition-colors">
              {loading ? '생성 중...' : '팀 생성'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
