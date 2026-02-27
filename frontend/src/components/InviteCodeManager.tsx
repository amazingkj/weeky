import { useState, useEffect, useCallback } from 'react';
import { InviteCode } from '../types';
import { createInviteCode, getInviteCodes } from '../services/api';

export default function InviteCodeManager() {
  const [codes, setCodes] = useState<InviteCode[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState('');
  const [copiedCode, setCopiedCode] = useState<string | null>(null);

  const fetchCodes = useCallback(async () => {
    try {
      const data = await getInviteCodes();
      setCodes(data);
    } catch {
      setError('초대 코드 목록을 불러올 수 없습니다');
    }
  }, []);

  useEffect(() => {
    fetchCodes();
  }, [fetchCodes]);

  const handleCreate = async () => {
    setIsLoading(true);
    setError('');
    try {
      await createInviteCode();
      await fetchCodes();
    } catch {
      setError('초대 코드 생성에 실패했습니다');
    } finally {
      setIsLoading(false);
    }
  };

  const handleCopy = async (code: string) => {
    try {
      await navigator.clipboard.writeText(code);
    } catch {
      // HTTPS가 아닌 환경에서는 execCommand fallback
      const textarea = document.createElement('textarea');
      textarea.value = code;
      textarea.style.position = 'fixed';
      textarea.style.opacity = '0';
      document.body.appendChild(textarea);
      textarea.select();
      document.execCommand('copy');
      document.body.removeChild(textarea);
    }
    setCopiedCode(code);
    setTimeout(() => setCopiedCode(null), 2000);
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold text-neutral-900">초대 코드 관리</h3>
        <button
          onClick={handleCreate}
          disabled={isLoading}
          className="px-3 py-1.5 bg-white text-neutral-700 text-xs font-medium rounded-lg border border-neutral-300 hover:bg-neutral-50 hover:border-neutral-400 disabled:opacity-50 transition-colors"
        >
          {isLoading ? '생성 중...' : '새 초대 코드'}
        </button>
      </div>

      {error && (
        <div className="text-sm text-red-600 bg-red-50 border border-red-200 rounded-lg px-3 py-2">
          {error}
        </div>
      )}

      {codes.length === 0 ? (
        <p className="text-sm text-neutral-500">아직 생성된 초대 코드가 없습니다.</p>
      ) : (
        <div className="space-y-2">
          {codes.map(ic => (
            <div
              key={ic.id}
              className={`flex items-center justify-between px-3 py-2 rounded-lg border ${
                ic.used_by ? 'bg-neutral-50 border-neutral-200' : 'bg-white border-neutral-200'
              }`}
            >
              <div className="flex items-center gap-3">
                <code className={`text-sm font-mono ${ic.used_by ? 'text-neutral-400 line-through' : 'text-neutral-900'}`}>
                  {ic.code}
                </code>
                {ic.used_by ? (
                  <span className="text-xs text-neutral-400">사용됨</span>
                ) : (
                  <span className="text-xs text-green-600">사용 가능</span>
                )}
              </div>
              {!ic.used_by && (
                <button
                  onClick={() => handleCopy(ic.code)}
                  className="text-xs text-neutral-500 hover:text-neutral-900 transition-colors"
                >
                  {copiedCode === ic.code ? '복사됨!' : '복사'}
                </button>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
