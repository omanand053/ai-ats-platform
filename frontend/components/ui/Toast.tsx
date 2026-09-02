"use client";

import type { Toast } from "@/hooks/useToast";

interface ToastContainerProps {
  toasts: Toast[];
  onDismiss: (id: number) => void;
}

export function ToastContainer({ toasts, onDismiss }: ToastContainerProps) {
  if (!toasts.length) return null;

  return (
    <div
      className="fixed bottom-4 right-4 z-50 flex w-[calc(100%-2rem)] max-w-sm flex-col gap-2 sm:bottom-6 sm:right-6"
      aria-live="polite"
    >
      {toasts.map((toast) => (
        <div
          key={toast.id}
          role="status"
          className={`flex items-start gap-3 rounded-xl px-4 py-3 text-sm shadow-lg ring-1 backdrop-blur-md animate-[ats-toast-in_180ms_ease] ${
            toast.type === "success"
              ? "bg-white/95 text-emerald-900 ring-emerald-200"
              : "bg-white/95 text-red-900 ring-red-200"
          }`}
        >
          <span
            className={`mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-full text-[10px] font-bold text-white ${
              toast.type === "success" ? "bg-emerald-500" : "bg-red-500"
            }`}
            aria-hidden
          >
            {toast.type === "success" ? "✓" : "!"}
          </span>
          <span className="flex-1 leading-5 text-slate-800">{toast.message}</span>
          <button
            type="button"
            onClick={() => onDismiss(toast.id)}
            className="shrink-0 rounded-md px-1.5 py-0.5 text-slate-400 transition hover:bg-slate-100 hover:text-slate-700"
            aria-label="Dismiss notification"
          >
            ×
          </button>
        </div>
      ))}
    </div>
  );
}
