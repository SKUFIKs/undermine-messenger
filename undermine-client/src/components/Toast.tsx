import React from 'react';
import { CheckCircle, XCircle, Info, AlertTriangle, X } from 'lucide-react';

export type ToastType = 'success' | 'error' | 'info' | 'warning';

export interface ToastItem {
  id: number;
  type: ToastType;
  message: string;
}

interface ToastContainerProps {
  toasts: ToastItem[];
  onDismiss: (id: number) => void;
}

const TOAST_ICONS: Record<ToastType, React.ReactNode> = {
  success: <CheckCircle className="w-4 h-4 text-emerald-400 shrink-0" />,
  error: <XCircle className="w-4 h-4 text-red-400 shrink-0" />,
  info: <Info className="w-4 h-4 text-sky-400 shrink-0" />,
  warning: <AlertTriangle className="w-4 h-4 text-amber-400 shrink-0" />,
};

const TOAST_ACCENTS: Record<ToastType, string> = {
  success: 'border-l-emerald-400',
  error: 'border-l-red-400',
  info: 'border-l-sky-400',
  warning: 'border-l-amber-400',
};

/**
 * Всплывающие уведомления в правом верхнем углу. Используется для
 * информирования о событиях соединения, ошибках отправки и прочих
 * сиюминутных состояниях, не требующих модального окна.
 */
export const ToastContainer: React.FC<ToastContainerProps> = ({ toasts, onDismiss }) => {
  return (
    <div className="fixed top-4 right-4 z-[100] flex flex-col gap-2 w-80 max-w-[calc(100vw-2rem)] pointer-events-none">
      {toasts.map((toast) => (
        <div
          key={toast.id}
          className={`pointer-events-auto flex items-start gap-2.5 px-3.5 py-3 rounded-lg
            bg-[#1E1E1E]/95 backdrop-blur border border-[#2D2D2D] border-l-4 shadow-xl shadow-black/40
            ${TOAST_ACCENTS[toast.type]} animate-toast-in`}
          role="status"
        >
          {TOAST_ICONS[toast.type]}
          <span className="flex-1 text-[13px] text-[#E5E5E5] leading-snug break-words">
            {toast.message}
          </span>
          <button
            onClick={() => onDismiss(toast.id)}
            className="text-[#A0A0A0] hover:text-white transition-colors shrink-0 cursor-pointer"
            aria-label="Закрыть уведомление"
          >
            <X className="w-3.5 h-3.5" />
          </button>
        </div>
      ))}
    </div>
  );
};
