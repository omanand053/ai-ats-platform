"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/Button";

export function ErrorState({
  title = "We couldn't load this page",
  description = "An unexpected error occurred. You can retry the request, refresh the page, or go back.",
  onRetry,
  showBack = true,
  showRefresh = true,
  homeHref = "/dashboard",
}: {
  title?: string;
  description?: string;
  onRetry?: () => void;
  showBack?: boolean;
  showRefresh?: boolean;
  homeHref?: string;
}) {
  const router = useRouter();

  return (
    <div
      className="flex min-h-[50vh] flex-col items-center justify-center px-6 py-16 text-center"
      role="alert"
    >
      <div className="mb-5 flex h-14 w-14 items-center justify-center rounded-2xl bg-[var(--brand-soft)] text-[var(--brand)] ring-1 ring-blue-100">
        <svg viewBox="0 0 24 24" className="h-6 w-6" fill="none" stroke="currentColor" strokeWidth="1.8" aria-hidden>
          <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v4m0 4h.01M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z" />
        </svg>
      </div>
      <h1 className="text-xl font-semibold tracking-tight text-[var(--text-primary)]">{title}</h1>
      <p className="mt-2 max-w-md text-sm leading-6 text-[var(--text-muted)]">{description}</p>
      <div className="mt-7 flex flex-wrap items-center justify-center gap-2">
        {onRetry ? (
          <Button className="w-auto min-w-[7rem] px-5" onClick={onRetry}>
            Retry
          </Button>
        ) : null}
        {showRefresh ? (
          <Button
            variant="secondary"
            className="w-auto min-w-[7rem] px-5"
            onClick={() => window.location.reload()}
          >
            Refresh
          </Button>
        ) : null}
        {showBack ? (
          <Button
            variant="ghost"
            className="w-auto min-w-[7rem] px-5"
            onClick={() => router.back()}
          >
            Back
          </Button>
        ) : null}
        <Link
          href={homeHref}
          className="inline-flex h-10 items-center justify-center rounded-lg px-4 text-sm font-semibold text-[var(--brand)] transition hover:bg-[var(--brand-soft)]"
        >
          Dashboard
        </Link>
      </div>
    </div>
  );
}
