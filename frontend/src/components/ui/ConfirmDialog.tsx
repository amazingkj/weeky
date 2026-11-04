import { createContext, ReactNode, useCallback, useContext, useEffect, useId, useRef, useState } from 'react';

export interface ConfirmOptions {
  title: string;
  message?: ReactNode;
  confirmLabel?: string;
  cancelLabel?: string;
  // 삭제처럼 되돌릴 수 없는 동작 — 빨간 확인 버튼, 기본 포커스는 취소
  danger?: boolean;
}

type ConfirmFn = (options: ConfirmOptions) => Promise<boolean>;

const ConfirmContext = createContext<ConfirmFn | null>(null);

// window.confirm 대체 — `if (!(await confirm({...}))) return;`
export function useConfirm(): ConfirmFn {
  const confirm = useContext(ConfirmContext);
  if (!confirm) throw new Error('useConfirm은 ConfirmProvider 안에서 사용해야 합니다');
  return confirm;
}

interface PendingConfirm extends ConfirmOptions {
  id: number;
  resolve: (ok: boolean) => void;
}

export function ConfirmProvider({ children }: { children: ReactNode }) {
  const [pending, setPending] = useState<PendingConfirm | null>(null);
  const pendingRef = useRef<PendingConfirm | null>(null);
  const nextId = useRef(0);

  const confirm = useCallback<ConfirmFn>((options) => new Promise<boolean>((resolve) => {
    // 이미 열린 확인창이 있으면 취소로 처리하고 새 요청으로 교체
    pendingRef.current?.resolve(false);
    pendingRef.current = { ...options, id: nextId.current++, resolve };
    setPending(pendingRef.current);
  }), []);

  const close = useCallback((ok: boolean) => {
    pendingRef.current?.resolve(ok);
    pendingRef.current = null;
    setPending(null);
  }, []);

  return (
    <ConfirmContext.Provider value={confirm}>
      {children}
      {/* 요청마다 새로 마운트 — 교체돼도 포커스 이동이 다시 실행됨 */}
      {pending && <ConfirmDialog key={pending.id} {...pending} onClose={close} />}
    </ConfirmContext.Provider>
  );
}

function ConfirmDialog({
  title,
  message,
  confirmLabel = '확인',
  cancelLabel = '취소',
  danger = false,
  onClose,
}: ConfirmOptions & { onClose: (ok: boolean) => void }) {
  const titleId = useId();
  const messageId = useId();
  const cancelRef = useRef<HTMLButtonElement>(null);
  const confirmRef = useRef<HTMLButtonElement>(null);

  // 열릴 때 포커스 이동, 닫힐 때 원래 위치로 복귀
  useEffect(() => {
    const previous = document.activeElement as HTMLElement | null;
    (danger ? cancelRef : confirmRef).current?.focus();
    return () => previous?.focus?.();
  }, [danger]);

  const handleKeyDown = (e: React.KeyboardEvent) => {
    // 모달이 떠 있는 동안 페이지의 전역 단축키(Ctrl+S 저장 등)가 동작하지 않도록
    e.stopPropagation();
    if ((e.ctrlKey || e.metaKey) && (e.code === 'KeyS' || e.key.toLowerCase() === 's')) {
      e.preventDefault();
      return;
    }
    if (e.key === 'Escape') {
      e.stopPropagation();
      onClose(false);
      return;
    }
    // 포커스를 두 버튼 안에 가둠
    if (e.key === 'Tab') {
      e.preventDefault();
      const next = document.activeElement === cancelRef.current ? confirmRef : cancelRef;
      next.current?.focus();
    }
  };

  return (
    <div
      className="fixed inset-0 z-[60] flex items-end sm:items-center justify-center p-4 bg-neutral-950/40 animate-fadeIn"
      onMouseDown={(e) => { if (e.target === e.currentTarget) onClose(false); }}
    >
      <div
        role="alertdialog"
        aria-modal="true"
        aria-labelledby={titleId}
        aria-describedby={message ? messageId : undefined}
        // 본문을 클릭해도 포커스가 대화상자 안에 남아 Esc·Tab 처리가 유지되도록
        tabIndex={-1}
        onKeyDown={handleKeyDown}
        className="w-full max-w-sm bg-white rounded-xl border border-neutral-200 shadow-lg p-5 animate-slideUp"
      >
        <h2 id={titleId} className="text-base font-semibold text-neutral-900">{title}</h2>
        {message && (
          <div id={messageId} className="mt-2 text-sm text-neutral-600 whitespace-pre-line">{message}</div>
        )}
        <div className="mt-5 flex justify-end gap-2">
          <button
            ref={cancelRef}
            type="button"
            onClick={() => onClose(false)}
            className="btn-secondary"
          >
            {cancelLabel}
          </button>
          <button
            ref={confirmRef}
            type="button"
            onClick={() => onClose(true)}
            className={danger ? 'btn-danger' : 'btn-primary'}
          >
            {confirmLabel}
          </button>
        </div>
      </div>
    </div>
  );
}
