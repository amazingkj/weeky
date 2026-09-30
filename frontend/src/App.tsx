import { useState, useCallback, Suspense, lazy } from 'react';
import { Routes, Route, Navigate } from 'react-router-dom';
import { useAuth } from './contexts/AuthContext';
import ErrorBoundary from './components/ErrorBoundary';
import Loading from './components/ui/Loading';
import { getWeekOfMonthLabel, getWeekRangeShort, toLocalYMD } from './utils/date';

const ReportForm = lazy(() => import('./components/ReportForm'));
const SiteReportForm = lazy(() => import('./components/SiteReportForm'));
const TeamPanel = lazy(() => import('./components/TeamPanel'));
const ConfigPanel = lazy(() => import('./components/ConfigPanel'));
const InviteCodeManager = lazy(() => import('./components/InviteCodeManager'));
const AdminUserManager = lazy(() => import('./components/AdminUserManager'));
const LoginPage = lazy(() => import('./pages/LoginPage'));
const RegisterPage = lazy(() => import('./pages/RegisterPage'));

type Tab = 'report' | 'site' | 'team' | 'config';

interface TabConfig {
  id: Tab;
  label: string;
  icon: React.ReactNode;
}

const reportIcon = (
  <svg fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
  </svg>
);

const siteIcon = (
  <svg fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0H5m14 0h2M5 21H3m4-7h.01M11 14h.01M7 10h.01M11 10h.01M7 6h.01M11 6h.01" />
  </svg>
);

const teamIcon = (
  <svg fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
  </svg>
);

const configIcon = (
  <svg fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
  </svg>
);

const TABS: TabConfig[] = [
  { id: 'report', label: '보고서 작성', icon: reportIcon },
  { id: 'site', label: '사이트 보고서', icon: siteIcon },
  { id: 'team', label: '팀', icon: teamIcon },
  { id: 'config', label: '설정', icon: configIcon },
];

function App() {
  const { isAuthenticated, isLoading } = useAuth();

  if (isLoading) {
    return (
      <div className="min-h-screen bg-neutral-100 flex items-center justify-center">
        <Loading text="로딩 중..." size="lg" />
      </div>
    );
  }

  return (
    <Suspense fallback={<div className="min-h-screen bg-neutral-100 flex items-center justify-center"><Loading text="로딩 중..." size="lg" /></div>}>
      <Routes>
        <Route path="/login" element={isAuthenticated ? <Navigate to="/" replace /> : <LoginPage />} />
        <Route path="/register" element={isAuthenticated ? <Navigate to="/" replace /> : <RegisterPage />} />
        <Route path="/*" element={isAuthenticated ? <AuthenticatedApp /> : <Navigate to="/login" replace />} />
      </Routes>
    </Suspense>
  );
}

function AuthenticatedApp() {
  const [activeTab, setActiveTab] = useState<Tab>('report');

  const handleTabChange = useCallback((tab: Tab) => {
    setActiveTab(tab);
  }, []);

  return (
    <div className="min-h-screen bg-neutral-100">
      <div className="sticky top-0 z-30">
        <Header />
        <Navigation activeTab={activeTab} onTabChange={handleTabChange} />
      </div>
      {/* 모바일 하단 탭바 높이만큼 pb 확보 */}
      <main className="max-w-6xl mx-auto px-4 sm:px-6 py-6 sm:py-8 pb-28 sm:pb-8">
        <ErrorBoundary>
          <Suspense fallback={<LoadingFallback />}>
            <div role="tabpanel" id={`${activeTab}-panel`} aria-labelledby={`${activeTab}-tab`}>
              {activeTab === 'report' && <ReportForm onNavigateToConfig={() => setActiveTab('config')} />}
              {activeTab === 'site' && <SiteReportForm />}
              {activeTab === 'team' && <TeamPanel />}
              {activeTab === 'config' && <ConfigWithInvite />}
            </div>
          </Suspense>
        </ErrorBoundary>
      </main>
      <MobileTabBar activeTab={activeTab} onTabChange={handleTabChange} />
    </div>
  );
}

function ConfigWithInvite() {
  const { user } = useAuth();
  return (
    <div className="space-y-8">
      <ConfigPanel />
      {user?.is_admin && (
        <>
          <div className="bg-white rounded-xl border border-neutral-200 p-5">
            <AdminUserManager />
          </div>
          <div className="bg-white rounded-xl border border-neutral-200 p-5">
            <InviteCodeManager />
          </div>
        </>
      )}
    </div>
  );
}

function Header() {
  const { user, logout } = useAuth();
  const today = toLocalYMD();

  return (
    <header className="border-b border-neutral-200 bg-white/95 backdrop-blur">
      <div className="max-w-6xl mx-auto px-4 sm:px-6 py-3">
        <div className="flex items-center justify-between gap-3">
          <div className="flex items-center gap-2.5 min-w-0">
            <div className="w-7 h-7 shrink-0 rounded-lg bg-ink-900 flex items-center justify-center">
              <span className="font-serif font-bold text-white text-[13px] leading-none">주</span>
            </div>
            <h1 className="font-serif text-lg font-bold text-neutral-900 tracking-tight leading-none">jugan</h1>
            {/* 주차 칩 — 이번 주가 이 앱의 단위 */}
            <span className="ml-1 inline-flex items-center gap-1.5 px-2 py-1 rounded-md bg-ink-50 border border-ink-100 text-[11px] font-medium text-ink-700 whitespace-nowrap">
              {getWeekOfMonthLabel(today)}
              <span className="hidden sm:inline text-ink-400 font-normal">{getWeekRangeShort(today)}</span>
            </span>
          </div>
          <div className="flex items-center gap-1 shrink-0">
            {user && (
              <span className="text-xs text-neutral-500 truncate max-w-[9rem]">
                {user.name}
                {user.is_admin && (
                  <span className="ml-1 px-1.5 py-0.5 bg-neutral-100 text-neutral-600 rounded text-[10px] font-medium">
                    관리자
                  </span>
                )}
              </span>
            )}
            <button
              onClick={logout}
              className="p-2 text-xs text-neutral-400 hover:text-neutral-700 transition-colors"
            >
              로그아웃
            </button>
          </div>
        </div>
      </div>
    </header>
  );
}

interface NavigationProps {
  activeTab: Tab;
  onTabChange: (tab: Tab) => void;
}

function Navigation({ activeTab, onTabChange }: NavigationProps) {
  return (
    <nav className="hidden sm:block border-b border-neutral-200 bg-neutral-50/90 backdrop-blur">
      <div className="max-w-6xl mx-auto px-4 sm:px-6">
        <div className="flex gap-0" role="tablist">
          {TABS.map((tab) => {
            const isActive = activeTab === tab.id;
            return (
              <button
                key={tab.id}
                id={`${tab.id}-tab`}
                onClick={() => onTabChange(tab.id)}
                role="tab"
                aria-selected={isActive}
                aria-controls={`${tab.id}-panel`}
                className={`
                  relative flex items-center gap-1.5 px-3 py-2.5 text-sm font-medium
                  transition-colors border-b-2 -mb-px
                  ${isActive
                    ? 'border-ink-700 text-ink-800'
                    : 'border-transparent text-neutral-500 hover:text-neutral-700'
                  }
                `}
              >
                <span className={`[&_svg]:w-4 [&_svg]:h-4 ${isActive ? 'text-ink-700' : 'text-neutral-400'}`}>
                  {tab.icon}
                </span>
                {tab.label}
              </button>
            );
          })}
        </div>
      </div>
    </nav>
  );
}

// 모바일 전용 하단 탭바 — 엄지 도달 범위, 44px+ 터치 타깃
function MobileTabBar({ activeTab, onTabChange }: NavigationProps) {
  return (
    <nav className="sm:hidden fixed bottom-0 inset-x-0 z-40 bg-white/95 backdrop-blur border-t border-neutral-200 pb-safe">
      <div className="grid grid-cols-4" role="tablist">
        {TABS.map((tab) => {
          const isActive = activeTab === tab.id;
          return (
            <button
              key={tab.id}
              onClick={() => onTabChange(tab.id)}
              role="tab"
              aria-selected={isActive}
              className={`relative flex flex-col items-center justify-center gap-0.5 py-2 min-h-[52px] transition-colors ${
                isActive ? 'text-ink-700' : 'text-neutral-400'
              }`}
            >
              {isActive && (
                <span className="absolute top-0 left-1/2 -translate-x-1/2 w-8 h-0.5 rounded-full bg-ink-700" />
              )}
              <span className="[&_svg]:w-5 [&_svg]:h-5">{tab.icon}</span>
              <span className="text-[10px] font-medium leading-none">{tab.label}</span>
            </button>
          );
        })}
      </div>
    </nav>
  );
}

const loadingFallback = (
  <div className="py-16">
    <Loading text="로딩 중..." size="lg" />
  </div>
);

function LoadingFallback() {
  return loadingFallback;
}

export default App;
