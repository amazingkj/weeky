import { ReactNode } from 'react';
import { ToastProvider } from '../components/ui/Toast';
import { ConfirmProvider } from '../components/ui/ConfirmDialog';

// render(ui, { wrapper: AppProviders }) — main.tsx와 같은 전역 UI Provider
export function AppProviders({ children }: { children: ReactNode }) {
  return (
    <ToastProvider>
      <ConfirmProvider>{children}</ConfirmProvider>
    </ToastProvider>
  );
}
