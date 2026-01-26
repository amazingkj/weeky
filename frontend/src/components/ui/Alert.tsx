import { ReactNode, useCallback, useEffect, useRef, useState } from 'react';

type AlertType = 'success' | 'error' | 'warning' | 'info';

interface AlertProps {
  type: AlertType;
  children: ReactNode;
  onClose?: () => void;
  className?: string;
  autoClose?: number;
}

const alertStyles: Record<AlertType, { container: string; icon: ReactNode }> = {
  success: {
    container: 'bg-neutral-50 border-neutral-200 text-neutral-700',
    icon: (
      <svg className="w-4 h-4 text-neutral-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
      </svg>
    ),
  },
  error: {
    container: 'bg-red-50 border-red-200 text-red-700',
    icon: (
      <svg className="w-4 h-4 text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>
    ),
  },
  warning: {
    container: 'bg-amber-50 border-amber-200 text-amber-700',
    icon: (
      <svg className="w-4 h-4 text-amber-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
      </svg>
    ),
  },
  info: {
    container: 'bg-neutral-50 border-neutral-200 text-neutral-600',
    icon: (
      <svg className="w-4 h-4 text-neutral-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>
    ),
  },
};

export default function Alert({ type, children, onClose, className = '', autoClose }: AlertProps) {
  const styles = alertStyles[type];
  const [isVisible, setIsVisible] = useState(true);
  const [isLeaving, setIsLeaving] = useState(false);
  const closeTimerRef = useRef<ReturnType<typeof setTimeout>>();

  const handleClose = useCallback(() => {
    setIsLeaving(true);
    closeTimerRef.current = setTimeout(() => {
      setIsVisible(false);
      onClose?.();
    }, 150);
  }, [onClose]);

  useEffect(() => {
    if (autoClose && onClose) {
      const timer = setTimeout(() => handleClose(), autoClose);
      return () => clearTimeout(timer);
    }
  }, [autoClose, onClose, handleClose]);

  useEffect(() => {
    return () => {
      if (closeTimerRef.current) clearTimeout(closeTimerRef.current);
    };
  }, []);

  if (!isVisible) return null;

  return (
    <div
      className={`
        flex items-start gap-2.5 p-3 rounded-lg border text-sm
        ${styles.container}
        ${isLeaving ? 'animate-slideOut' : 'animate-slideIn'}
        ${className}
      `}
      role="alert"
    >
      <span className="flex-shrink-0 mt-0.5">{styles.icon}</span>
      <div className="flex-1 min-w-0 text-sm">{children}</div>
      {onClose ? (
        <button
          onClick={handleClose}
          className="flex-shrink-0 p-0.5 text-neutral-400 hover:text-neutral-600 transition-colors"
          aria-label="닫기"
        >
          <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      ) : null}
    </div>
  );
}
