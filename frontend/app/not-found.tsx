import Link from "next/link";

export default function NotFound() {
  return (
    <div className="flex min-h-[70vh] flex-col items-center justify-center px-6 text-center">
      <p className="text-xs font-semibold uppercase tracking-[0.14em] text-slate-400">404</p>
      <h1 className="mt-2 text-2xl font-semibold tracking-tight text-slate-900">Page not found</h1>
      <p className="mt-2 max-w-md text-sm leading-6 text-slate-500">
        This route doesn&apos;t exist or may have moved. Head back to your hiring workspace.
      </p>
      <div className="mt-6 flex flex-wrap justify-center gap-3">
        <Link
          href="/dashboard"
          className="inline-flex h-10 items-center rounded-lg bg-blue-600 px-5 text-sm font-semibold text-white transition hover:bg-blue-500"
        >
          Go to dashboard
        </Link>
        <Link
          href="/dashboard/jobs"
          className="inline-flex h-10 items-center rounded-lg bg-white px-5 text-sm font-semibold text-slate-700 ring-1 ring-slate-200 transition hover:bg-slate-50"
        >
          Browse jobs
        </Link>
      </div>
    </div>
  );
}
