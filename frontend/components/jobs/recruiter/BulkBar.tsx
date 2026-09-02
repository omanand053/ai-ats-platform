"use client";

import { Button } from "@/components/ui/Button";
import type { CandidateStatus } from "@/lib/candidate-types";

export function BulkBar({
  count,
  busy,
  onClear,
  onExport,
  onBulkStage,
}: {
  count: number;
  busy: boolean;
  onClear: () => void;
  onExport: () => void;
  onBulkStage: (status: CandidateStatus) => void;
}) {
  if (count === 0) return null;

  return (
    <div className="sticky top-0 z-10 mb-3 flex flex-wrap items-center justify-between gap-2 rounded-[12px] border border-blue-200 bg-blue-50/90 px-2 py-1.5 shadow-sm backdrop-blur">
      <p className="text-sm font-medium text-blue-900">{count} selected</p>
      <div className="flex flex-wrap gap-2">
        <Button
          size="sm"
          variant="secondary"
          className="w-auto px-3"
          disabled={busy}
          onClick={() => onBulkStage("recruiter_shortlisted")}
        >
          Shortlist
        </Button>
        <Button
          size="sm"
          variant="secondary"
          className="w-auto px-3"
          disabled={busy}
          onClick={() => onBulkStage("interview")}
        >
          Interview
        </Button>
        <Button
          size="sm"
          variant="danger"
          className="w-auto px-3"
          disabled={busy}
          onClick={() => onBulkStage("rejected")}
        >
          Reject
        </Button>
        <Button size="sm" variant="secondary" className="w-auto px-3" disabled={busy} onClick={onExport}>
          Export
        </Button>
        <Button size="sm" variant="secondary" className="w-auto px-3" disabled={busy} onClick={onClear}>
          Clear
        </Button>
      </div>
    </div>
  );
}
