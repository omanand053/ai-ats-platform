"use client";

import { Tooltip } from "@/components/ui/Tooltip";

export function ActionIcon({
  label,
  onClick,
  disabled,
  loading,
  tone = "neutral",
  children,
}: {
  label: string;
  onClick: () => void;
  disabled?: boolean;
  loading?: boolean;
  tone?: "neutral" | "danger" | "primary" | "success";
  children: React.ReactNode;
}) {
  const tones = {
    neutral: "bg-white text-slate-700 ring-slate-200 hover:bg-slate-50 hover:ring-slate-300",
    danger: "bg-white text-rose-600 ring-rose-100 hover:bg-rose-50 hover:ring-rose-200",
    primary: "bg-white text-blue-700 ring-blue-100 hover:bg-blue-50 hover:ring-blue-200",
    success: "bg-white text-emerald-700 ring-emerald-100 hover:bg-emerald-50 hover:ring-emerald-200",
  };

  return (
    <Tooltip label={label} delayMs={220}>
      <button
        type="button"
        aria-label={label}
        disabled={disabled || loading}
        onClick={(e) => {
          e.preventDefault();
          e.stopPropagation();
          onClick();
        }}
        onMouseDown={(e) => e.stopPropagation()}
        className={`inline-flex h-7 w-7 shrink-0 items-center justify-center rounded-lg text-[12px] ring-1 transition duration-150 ease-out hover:-translate-y-px hover:shadow-sm focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-500 disabled:pointer-events-none disabled:opacity-40 ${tones[tone]}`}
      >
        {loading ? (
          <span className="h-3.5 w-3.5 animate-spin rounded-full border-2 border-slate-300 border-t-slate-700" />
        ) : (
          children
        )}
      </button>
    </Tooltip>
  );
}
