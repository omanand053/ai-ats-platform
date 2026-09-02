import type { CandidateStatus, ProcessingStatus } from "@/lib/candidate-types";

const statusStyles: Record<CandidateStatus, string> = {
  applied: "bg-sky-50 text-sky-700 ring-sky-200",
  screening: "bg-amber-50 text-amber-700 ring-amber-200",
  shortlisted: "bg-violet-50 text-violet-700 ring-violet-200",
  recruiter_shortlisted: "bg-fuchsia-50 text-fuchsia-700 ring-fuchsia-200",
  ai_shortlisted: "bg-indigo-50 text-indigo-700 ring-indigo-200",
  interview: "bg-blue-50 text-blue-700 ring-blue-200",
  selected: "bg-emerald-50 text-emerald-700 ring-emerald-200",
  offer: "bg-emerald-50 text-emerald-700 ring-emerald-200",
  hired: "bg-green-50 text-green-700 ring-green-200",
  rejected: "bg-rose-50 text-rose-700 ring-rose-200",
};

const statusLabels: Record<CandidateStatus, string> = {
  applied: "Applied",
  screening: "Screening",
  shortlisted: "Shortlisted",
  recruiter_shortlisted: "Recruiter Shortlisted",
  ai_shortlisted: "AI Shortlisted",
  interview: "Interview",
  selected: "Selected",
  offer: "Offer",
  hired: "Hired",
  rejected: "Rejected",
};

export function CandidateStatusBadge({ status }: { status: CandidateStatus }) {
  return (
    <span
      className={`inline-flex items-center rounded-full px-2.5 py-1 text-[11px] font-semibold tracking-wide ring-1 ring-inset ${statusStyles[status]}`}
    >
      <span className="mr-1.5 h-1.5 w-1.5 rounded-full bg-current opacity-70" aria-hidden />
      {statusLabels[status]}
    </span>
  );
}

export function ProcessingStatusBadge({ status }: { status: ProcessingStatus }) {
  const styles: Record<ProcessingStatus, string> = {
    pending: "bg-slate-100 text-slate-600 ring-slate-200",
    processing: "bg-amber-50 text-amber-700 ring-amber-200",
    completed: "bg-emerald-50 text-emerald-700 ring-emerald-200",
    failed: "bg-rose-50 text-rose-700 ring-rose-200",
  };
  return (
    <span
      className={`inline-flex items-center rounded-full px-2.5 py-1 text-[11px] font-semibold capitalize tracking-wide ring-1 ring-inset ${styles[status]}`}
    >
      {status}
    </span>
  );
}

export function formatDate(date: string) {
  return new Date(date).toLocaleDateString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
  });
}
