export function Spinner({ label = "Loading..." }: { label?: string }) {
  return (
    <div className="flex min-h-[40vh] flex-col items-center justify-center gap-3" role="status" aria-live="polite">
      <div className="h-8 w-8 animate-spin rounded-full border-2 border-slate-200 border-t-blue-600" />
      <p className="text-sm text-slate-500">{label}</p>
    </div>
  );
}

export function InlineLoading({ label = "Loading..." }: { label?: string }) {
  return (
    <div className="flex items-center justify-center gap-3 px-6 py-12" role="status" aria-live="polite">
      <div className="h-5 w-5 animate-spin rounded-full border-2 border-slate-200 border-t-blue-600" />
      <p className="text-sm text-slate-500">{label}</p>
    </div>
  );
}
