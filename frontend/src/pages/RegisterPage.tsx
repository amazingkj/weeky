import { useState, useEffect, FormEvent } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { register as registerApi, checkSetup } from '../services/api';
import { useAuth } from '../contexts/AuthContext';

export default function RegisterPage() {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [name, setName] = useState('');
  const [inviteCode, setInviteCode] = useState('');
  const [error, setError] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [isFirstUser, setIsFirstUser] = useState(false);
  const { login } = useAuth();
  const navigate = useNavigate();

  useEffect(() => {
    checkSetup().then(res => {
      setIsFirstUser(!res.initialized);
    }).catch(() => {});
  }, []);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    setIsLoading(true);

    try {
      const res = await registerApi({ email, password, name, invite_code: inviteCode });
      login(res.user);
      navigate('/', { replace: true });
    } catch (err) {
      setError(err instanceof Error ? err.message : '회원가입에 실패했습니다');
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-neutral-100 flex items-center justify-center px-4">
      <div className="w-full max-w-sm">
        <div className="text-center mb-8">
          <div className="inline-flex w-11 h-11 rounded-xl bg-ink-900 items-center justify-center mb-3">
            <span className="font-serif font-bold text-white text-lg">주</span>
          </div>
          <h1 className="font-serif text-2xl font-bold text-neutral-900 tracking-tight">jugan</h1>
          <p className="text-sm text-neutral-500 mt-1">
            {isFirstUser ? '첫 번째 관리자 계정을 만들어주세요' : '주간업무보고 자동화'}
          </p>
        </div>

        <form onSubmit={handleSubmit} className="bg-white rounded-xl border border-neutral-200 p-6 space-y-4">
          <h2 className="text-base font-semibold text-neutral-900">
            {isFirstUser ? '관리자 계정 생성' : '회원가입'}
          </h2>

          {isFirstUser && (
            <div className="text-sm text-ink-800 bg-ink-50 border border-ink-100 rounded-lg px-3 py-2">
              첫 번째 사용자는 자동으로 관리자 권한을 부여받습니다. 초대 코드가 필요하지 않습니다.
            </div>
          )}

          {error && (
            <div className="text-sm text-red-600 bg-red-50 border border-red-200 rounded-lg px-3 py-2">
              {error}
            </div>
          )}

          <div>
            <label className="block text-sm font-medium text-neutral-700 mb-1">이름</label>
            <input
              type="text"
              value={name}
              onChange={e => setName(e.target.value)}
              required
              className="input"
              placeholder="홍길동"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-neutral-700 mb-1">이메일</label>
            <input
              type="email"
              value={email}
              onChange={e => setEmail(e.target.value)}
              required
              className="input"
              placeholder="you@example.com"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-neutral-700 mb-1">비밀번호</label>
            <input
              type="password"
              value={password}
              onChange={e => setPassword(e.target.value)}
              required
              minLength={6}
              className="input"
              placeholder="6자 이상"
            />
          </div>

          {!isFirstUser && (
            <div>
              <label className="block text-sm font-medium text-neutral-700 mb-1">초대 코드</label>
              <input
                type="text"
                value={inviteCode}
                onChange={e => setInviteCode(e.target.value)}
                required
                className="input font-mono"
                placeholder="관리자에게 받은 초대 코드"
              />
            </div>
          )}

          <button
            type="submit"
            disabled={isLoading}
            className="w-full py-2.5 bg-ink-800 text-white text-sm font-medium rounded-lg hover:bg-ink-900 disabled:opacity-50 transition-colors"
          >
            {isLoading ? '가입 중...' : isFirstUser ? '관리자 계정 생성' : '회원가입'}
          </button>

          <p className="text-center text-sm text-neutral-500">
            이미 계정이 있으신가요?{' '}
            <Link to="/login" className="text-ink-700 font-medium hover:underline">
              로그인
            </Link>
          </p>
        </form>
      </div>
    </div>
  );
}
