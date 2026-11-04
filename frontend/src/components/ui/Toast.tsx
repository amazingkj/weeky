import { createContext, ReactNode, useCallback, useContext, useMemo, useRef, useState } from 'react';
import Alert from './Alert';

type ToastType = 'success' | 'error' | 'warning' | 'info';

interface ToastApi {
  success: (message: string) => void;
  error: (message: string) => void;
  warning: (message: string) => void;
  info: (message: string) => void;
}

interface ToastItem {
  id: number;
  type: ToastType;
  message: string;
}

const ToastContext = createContext<ToastApi | null>(null);

// window.alert 대체 — 화면 위에 잠깐 떴다 사라지는 알림
export function useToast(): ToastApi {
  const toast = useContext(ToastContext);
  if (!toast) throw new Error('useToast는 ToastProvider 안에서 사용해야 합니다');
  return toast;
}

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<ToastItem[]>([]);
  const nextId = useRef(0);

  const remove = useCallback((id: number) => {
    setToasts((prev) => prev.filter((t) => t.id !== id));
  }, []);

  const api = useMemo<ToastApi>(() => {
    const push = (type: ToastType) => (message: string) => {
      const id = nextId.current++;
      setToasts((prev) => [...prev, { id, type, message }]);
    };
    return { success: push('success'), error: push('error'), warning: push('warning'), info: push('info') };
  }, []);

  return (
    <ToastContext.Provider value={api}>
      {children}
      <div className="fixed top-3 inset-x-4 sm:inset-x-auto sm:top-20 sm:right-6 sm:w-96 z-[70] flex flex-col gap-2 pointer-events-none">
        {toasts.map((t) => (
          <ToastEntry key={t.id} toast={t} onRemove={remove} />
        ))}
      </div>
    </ToastContext.Provider>
  );
}

function ToastEntry({ toast, onRemove }: { toast: ToastItem; onRemove: (id: number) => void }) {
  const onClose = useCallback(() => onRemove(toast.id), [onRemove, toast.id]);
  return (
    <Alert
      type={toast.type}
      onClose={onClose}
      // 오류는 읽을 시간을 더 줌
      autoClose={toast.type === 'error' ? 6000 : 3500}
      className="pointer-events-auto shadow-lg"
    >
      {toast.message}
    </Alert>
  );
}
