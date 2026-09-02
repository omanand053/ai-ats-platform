"use client";

import { Button } from "@/components/ui/Button";

interface DeleteCandidateModalProps {
  candidateName: string;
  loading?: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}

export function DeleteCandidateModal({
  candidateName,
  loading,
  onConfirm,
  onCancel,
}: DeleteCandidateModalProps) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 p-4 backdrop-blur-[2px]">
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="delete-candidate-title"
        className="w-full max-w-md rounded-2xl bg-white p-6 shadow-2xl ring-1 ring-slate-200"
      >
        <div className="mb-4 flex h-10 w-10 items-center justify-center rounded-xl bg-rose-50 text-rose-600 ring-1 ring-rose-100">
          <svg viewBox="0 0 24 24" className="h-5 w-5" fill="none" stroke="currentColor" strokeWidth="1.8" aria-hidden>
            <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v4m0 4h.01M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z" />
          </svg>
        </div>
        <h3 id="delete-candidate-title" className="text-lg font-semibold text-slate-900">
          Delete candidate?
        </h3>
        <p className="mt-2 text-sm leading-6 text-slate-600">
          Are you sure you want to delete <strong className="text-slate-900">{candidateName}</strong>? This
          action cannot be undone.
        </p>
        <div className="mt-6 flex flex-col gap-3 sm:flex-row-reverse">
          <Button variant="danger" className="w-auto px-4" loading={loading} onClick={onConfirm}>
            Delete
          </Button>
          <Button variant="secondary" className="w-auto px-4" disabled={loading} onClick={onCancel}>
            Cancel
          </Button>
        </div>
      </div>
    </div>
  );
}
