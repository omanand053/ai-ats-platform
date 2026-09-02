"use client";

import { CandidateStatusBadge } from "@/components/candidates/candidate-utils";
import { ActionIcon } from "@/components/jobs/recruiter/ActionIcon";
import {
  confidenceTone,
  formatPct,
  resumeStatusLabel,
  type RankedApplicant,
} from "@/components/jobs/recruiter/filters";
import { Avatar } from "@/components/ui/Avatar";
import type { CandidateStatus } from "@/lib/candidate-types";

export function CandidateCard({
  row,
  rank,
  selected,
  busy,
  onToggleSelect,
  onOpen,
  onStage,
  onResume,
  onConfirmReject,
}: {
  row: RankedApplicant;
  rank: number;
  selected: boolean;
  busy: boolean;
  onToggleSelect: () => void;
  onOpen: () => void;
  onStage: (status: CandidateStatus) => void;
  onResume: () => void;
  onConfirmReject: () => void;
}) {
  const { candidate, match } = row;
  const lowEligibility =
    match?.low_eligibility ||
    (typeof match?.eligibility_score === "number" && match.eligibility_score < 40);

  return (
    <article
      role="button"
      tabIndex={0}
      onClick={onOpen}
      onKeyDown={(e) => {
        if (e.key === "Enter" || e.key === " ") {
          e.preventDefault();
          onOpen();
        }
      }}
      className={`group mb-2 cursor-pointer rounded-[var(--radius-lg)] border bg-white p-1.5 shadow-[0_6px_18px_rgba(13,18,30,0.04)] transition duration-150 ease-out hover:-translate-y-0.5 hover:border-gray-200 hover:shadow-[0_8px_16px_rgba(13,18,30,0.06)] focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[#7c3aed] ${
        selected ? "ring-2 ring-blue-100 border-blue-300" : "border-[var(--border)]"
      }`}
    >
      <div className="flex gap-3">
        <div
          className="flex flex-col items-center gap-2 pt-0.5"
          onClick={(e) => e.stopPropagation()}
          onKeyDown={(e) => e.stopPropagation()}
        >
          <input
            type="checkbox"
            checked={selected}
            onChange={onToggleSelect}
            onClick={(e) => e.stopPropagation()}
            aria-label={`Select ${candidate.name}`}
            className="h-3.5 w-3.5 rounded border-slate-300 text-blue-600 focus:ring-blue-500"
          />
          <Avatar name={candidate.name} size="sm" />
        </div>

        <div className="min-w-0 flex-1">
          <div className="flex flex-col gap-0.5 sm:flex-row sm:items-start sm:justify-between">
            <div className="min-w-0">
              <div className="flex flex-wrap items-center gap-2">
                <span className="inline-flex h-5 min-w-5 items-center justify-center rounded-full bg-slate-100 px-1.5 text-[10px] font-semibold tabular-nums text-slate-600">
                  #{rank}
                </span>
                <span className="truncate text-sm font-semibold tracking-tight text-[#0b1220] group-hover:text-[#4f46e5]">
                  {candidate.name}
                </span>
                {lowEligibility && (
                  <span
                    title="Soft warning only — candidate remains ranked"
                    className="inline-flex rounded-md bg-amber-50 px-1.5 py-0.5 text-[10px] font-semibold text-amber-900 ring-1 ring-amber-100"
                  >
                    Low eligibility
                  </span>
                )}
              </div>
              <p className="mt-0.5 truncate text-xs leading-5 text-[#6b7280]">
                {candidate.current_designation || "No designation"}
                {typeof candidate.experience_years === "number"
                  ? ` · ${candidate.experience_years} yrs`
                  : ""}
                {candidate.location ? ` · ${candidate.location}` : ""}
              </p>
            </div>

            <div className="flex flex-wrap items-center gap-1 sm:justify-end">
              <span className="inline-flex rounded-lg bg-blue-50 px-2.5 py-1 text-xs font-bold tabular-nums text-blue-800 ring-1 ring-blue-100">
                {formatPct(match?.ai_match_score)}
              </span>
              <span
                className={`inline-flex rounded-lg px-2 py-1 text-[10px] font-semibold capitalize ring-1 ${confidenceTone(match?.confidence)}`}
              >
                {match?.confidence || "—"}
              </span>
              <CandidateStatusBadge status={candidate.status} />
              <span className="inline-flex rounded-lg bg-slate-50 px-2 py-1 text-[10px] font-medium text-slate-600 ring-1 ring-slate-200">
                {resumeStatusLabel(candidate)}
              </span>
            </div>
          </div>

          <div
            className="mt-1 flex flex-wrap items-center gap-0.5 border-t border-gray-100 pt-1"
            onClick={(e) => e.stopPropagation()}
            onMouseDown={(e) => e.stopPropagation()}
          >
            {(candidate.status === "applied" ||
              candidate.status === "screening" ||
              candidate.status === "shortlisted") && (
              <ActionIcon
                label="AI Shortlist"
                disabled={busy}
                tone="primary"
                onClick={() => onStage("ai_shortlisted")}
              >
                ⭐
              </ActionIcon>
            )}
            <ActionIcon
              label="Recruiter Shortlist"
              disabled={busy || candidate.status === "rejected" || candidate.status === "selected"}
              tone="primary"
              onClick={() => onStage("recruiter_shortlisted")}
            >
              👍
            </ActionIcon>
            <ActionIcon
              label="Schedule Interview"
              disabled={
                busy ||
                candidate.status === "interview" ||
                candidate.status === "selected" ||
                candidate.status === "rejected"
              }
              onClick={() => onStage("interview")}
            >
              📅
            </ActionIcon>
            <ActionIcon
              label="Mark Selected"
              disabled={busy || candidate.status === "selected" || candidate.status === "rejected"}
              tone="success"
              onClick={() => onStage("selected")}
            >
              🏆
            </ActionIcon>
            <ActionIcon
              label="Reject Candidate"
              disabled={busy || candidate.status === "rejected"}
              tone="danger"
              onClick={onConfirmReject}
            >
              ❌
            </ActionIcon>
            <ActionIcon
              label="Preview Resume"
              disabled={busy || !candidate.resume_url}
              onClick={onResume}
            >
              📄
            </ActionIcon>
            <ActionIcon label="Candidate Profile" disabled={busy} onClick={onOpen}>
              👤
            </ActionIcon>
          </div>
        </div>
      </div>
    </article>
  );
}
