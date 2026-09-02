import Link from "next/link";
import type { ReactNode } from "react";

interface AuthLayoutProps {
  title: string;
  subtitle: string;
  children: ReactNode;
  footer: ReactNode;
}

export function AuthLayout({ title, subtitle, children, footer }: AuthLayoutProps) {
  return (
    <div className="relative flex min-h-full flex-1 items-center justify-center overflow-hidden bg-[#f7f8fa] px-4 py-12">
      <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(900px_420px_at_15%_-10%,#dbeafe_0%,transparent_55%),radial-gradient(700px_360px_at_90%_10%,#e2e8f0_0%,transparent_50%)]" />
      <div className="relative w-full max-w-md">
        <div className="mb-8 text-center">
          <Link
            href="/"
            className="inline-flex items-center gap-2 rounded-full bg-white px-3 py-1.5 text-xs font-semibold tracking-wide text-slate-700 ring-1 ring-slate-200 shadow-sm"
          >
            <span className="flex h-5 w-5 items-center justify-center rounded-md bg-slate-900 text-[10px] text-white">
              AI
            </span>
            AI ATS Platform
          </Link>
          <h1 className="mt-6 text-3xl font-semibold tracking-tight text-slate-900">{title}</h1>
          <p className="mt-2 text-sm leading-6 text-slate-500">{subtitle}</p>
        </div>

        <div className="rounded-2xl bg-white p-6 shadow-[var(--shadow-lg)] ring-1 ring-slate-200/80 sm:p-8">
          {children}
        </div>

        <p className="mt-6 text-center text-sm text-slate-500">{footer}</p>
      </div>
    </div>
  );
}
