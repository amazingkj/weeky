import { useState, useCallback, Suspense, lazy } from 'react';
import ErrorBoundary from './components/ErrorBoundary';
import Loading from './components/ui/Loading';

const ReportForm = lazy(() => import('./components/ReportForm'));
const ConfigPanel = lazy(() => import('./components/ConfigPanel'));

type Tab = 'report' | 'config';

interface TabConfig {
  id: Tab;
  label: string;
  icon: React.ReactNode;
}

const reportIcon = (
  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
  </svg>
);

const configIcon = (
  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
  </svg>
);

const TABS: TabConfig[] = [
  { id: 'report', label: '보고서 작성', icon: reportIcon },
  { id: 'config', label: '설정', icon: configIcon },
];

function App() {
  const [activeTab, setActiveTab] = useState<Tab>('report');

  const handleTabChange = useCallback((tab: Tab) => {
    setActiveTab(tab);
  }, []);

  return (
    <div className="min-h-screen bg-neutral-50">
      <Header />
      <Navigation activeTab={activeTab} onTabChange={handleTabChange} />
      <main className="max-w-3xl mx-auto px-4 sm:px-6 py-8">
        <ErrorBoundary>
          <Suspense fallback={<LoadingFallback />}>
            {activeTab === 'report' ? <ReportForm /> : <ConfigPanel />}
          </Suspense>
        </ErrorBoundary>
      </main>
    </div>
  );
}

function Header() {
  return (
    <header className="border-b border-neutral-200 bg-white">
      <div className="max-w-3xl mx-auto px-4 sm:px-6 py-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2.5">
            <div className="w-7 h-7 rounded-lg bg-neutral-900 flex items-center justify-center">
              <svg className="w-4 h-4 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4" />
              </svg>
            </div>
            <h1 className="text-base font-semibold text-neutral-900 tracking-tight">weeky</h1>
          </div>
          <span className="text-xs text-neutral-400 font-mono">v0.2</span>
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
    <nav className="border-b border-neutral-200 bg-white">
      <div className="max-w-3xl mx-auto px-4 sm:px-6">
        <div className="flex gap-0" role="tablist">
          {TABS.map((tab) => {
            const isActive = activeTab === tab.id;
            return (
              <button
                key={tab.id}
                onClick={() => onTabChange(tab.id)}
                role="tab"
                aria-selected={isActive}
                aria-controls={`${tab.id}-panel`}
                className={`
                  relative flex items-center gap-1.5 px-3 py-2.5 text-sm font-medium
                  transition-colors border-b-2 -mb-px
                  ${isActive
                    ? 'border-neutral-900 text-neutral-900'
                    : 'border-transparent text-neutral-500 hover:text-neutral-700'
                  }
                `}
              >
                <span className={isActive ? 'text-neutral-900' : 'text-neutral-400'}>
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

const loadingFallback = (
  <div className="py-16">
    <Loading text="로딩 중..." size="lg" />
  </div>
);

function LoadingFallback() {
  return loadingFallback;
}

export default App;
