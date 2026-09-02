import type { ReactNode } from "react";
import { Button } from "@/components/ui/Button";

interface EmptyStateProps {
  title: string;
  description?: string;
  actionLabel?: string;
  onAction?: () => void;
  icon?: ReactNode;
}

export function EmptyState({
  title,
  description,
  actionLabel,
  onAction,
  icon,
}: EmptyStateProps) {
  return (
    <div className="flex flex-col items-center justify-center px-6 py-16 text-center sm:py-20">
      <div className="mb-4 flex h-12 w-12 items-center justify-center rounded-[var(--radius-lg)] bg-[var(--surface-muted)] text-[var(--text-muted)] ring-1 ring-[var(--border)]">
        {icon ?? (
          <svg viewBox="0 0 24 24" className="h-5 w-5" fill="none" stroke="currentColor" strokeWidth="1.8" aria-hidden>
            <path strokeLinecap="round" strokeLinejoin="round" d="M4 7h16M4 12h10M4 17h7" />
          </svg>
        )}
      </div>
      <p className="text-base font-semibold tracking-tight text-[var(--text-primary)]">{title}</p>
      {description ? (
        <p className="mt-1.5 max-w-md text-sm leading-6 text-[var(--text-muted)]">{description}</p>
      ) : null}
      {actionLabel && onAction ? (
        <Button className="mt-6 w-auto px-5" onClick={onAction}>
          {actionLabel}
        </Button>
      ) : null}
    </div>
  );
}
