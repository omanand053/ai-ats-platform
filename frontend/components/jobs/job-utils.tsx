import type { JobStatus } from "@/lib/job-types";

const styles: Record<JobStatus, string> = {
  draft: "bg-slate-100 text-slate-700 ring-slate-200",
  open: "bg-emerald-50 text-emerald-700 ring-emerald-200",
  closed: "bg-rose-50 text-rose-700 ring-rose-200",
};

const labels: Record<JobStatus, string> = {
  draft: "Draft",
  open: "Open",
  closed: "Closed",
};

export function StatusBadge({ status }: { status: JobStatus }) {
  return (
    <span
      className={`inline-flex items-center rounded-full px-2.5 py-1 text-[11px] font-semibold tracking-wide ring-1 ring-inset ${styles[status]}`}
    >
      <span className="mr-1.5 h-1.5 w-1.5 rounded-full bg-current opacity-70" aria-hidden />
      {labels[status]}
    </span>
  );
}

export function formatEmploymentType(type: string) {
  return type.replace(/_/g, " ").replace(/\b\w/g, (c) => c.toUpperCase());
}

export function formatDate(date: string) {
  return new Date(date).toLocaleDateString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
  });
}
