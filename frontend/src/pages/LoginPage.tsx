import { useState, FormEvent } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { login as loginApi } from '../services/api';
import { useAuth } from '../contexts/AuthContext';

export default function LoginPage() {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const { login } = useAuth();
  const navigate = useNavigate();

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    setIsLoading(true);

    try {
      const res = await loginApi({ email, password });
      login(res.user);
      navigate('/', { replace: true });
    } catch (err) {
      setError(err instanceof Error ? err.message : '로그인에 실패했습니다');
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
          <p className="text-sm text-neutral-500 mt-1">주간업무보고 자동화</p>
        </div>

        <form onSubmit={handleSubmit} className="bg-white rounded-xl border border-neutral-200 p-6 space-y-4">
          <h2 className="text-base font-semibold text-neutral-900">로그인</h2>

          {error && (
            <div className="text-sm text-red-600 bg-red-50 border border-red-200 rounded-lg px-3 py-2">
              {error}
            </div>
          )}

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
              className="input"
              placeholder="••••••"
            />
          </div>

          <button
            type="submit"
            disabled={isLoading}
            className="w-full py-2.5 bg-ink-800 text-white text-sm font-medium rounded-lg hover:bg-ink-900 disabled:opacity-50 transition-colors"
          >
            {isLoading ? '로그인 중...' : '로그인'}
          </button>

          <p className="text-center text-sm text-neutral-500">
            계정이 없으신가요?{' '}
            <Link to="/register" className="text-ink-700 font-medium hover:underline">
              회원가입
            </Link>
          </p>
        </form>
      </div>
    </div>
  );
}
