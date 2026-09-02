"use client";

import { useDeferredValue, useMemo, useState, useTransition } from "react";
import { BulkBar } from "@/components/jobs/recruiter/BulkBar";
import { CandidateCard } from "@/components/jobs/recruiter/CandidateCard";
import { CandidateDrawer } from "@/components/jobs/recruiter/CandidateDrawer";
import { RecruiterFilters } from "@/components/jobs/recruiter/RecruiterFilters";
import { VirtualList } from "@/components/jobs/recruiter/VirtualList";
import {
  applyFilters,
  exportCandidatesCsv,
  loadSavedFilters,
  saveFilters,
  type RankedApplicant,
  type WorkspaceFilters,
} from "@/components/jobs/recruiter/filters";
import { ConfirmDialog } from "@/components/ui/ConfirmDialog";
import { EmptyState } from "@/components/ui/EmptyState";
import { Skeleton } from "@/components/ui/Skeleton";
import type { Candidate, CandidatePayload, CandidateStatus } from "@/lib/candidate-types";
import type { Job } from "@/lib/job-types";
import { viewResumeFile } from "@/lib/resume-file";
import { updateCandidate } from "@/services/candidate.service";
import type { SemanticMatch, SemanticMatchResult } from "@/services/job.service";
import { ApiClientError } from "@/lib/api-client";

function buildPayload(candidate: Candidate, status: CandidateStatus): CandidatePayload {
  return {
    job_id: candidate.job_id,
    name: candidate.name,
    email: candidate.email,
    phone: candidate.phone ?? "",
    experience_years: candidate.experience_years,
    current_company: candidate.current_company ?? "",
    current_designation: candidate.current_designation ?? "",
    location: candidate.location ?? "",
    skills: candidate.skills ?? [],
    status,
    resume_url: candidate.resume_url ?? "",
    resume_text: candidate.resume_text ?? "",
    resume_summary: candidate.resume_summary ?? "",
    source: candidate.source ?? "",
    parsing_status: candidate.parsing_status,
    embedding_status: candidate.embedding_status,
  };
}

function isRecruiterShortlisted(status: CandidateStatus) {
  return status === "shortlisted" || status === "recruiter_shortlisted";
}

export function RecruiterWorkspace({
  job,
  candidates,
  setCandidates,
  semantic,
  loading,
  onToast,
}: {
  job: Job;
  candidates: Candidate[];
  setCandidates: React.Dispatch<React.SetStateAction<Candidate[]>>;
  semantic: SemanticMatchResult | null;
  loading: boolean;
  onToast: (message: string, tone?: "success" | "error") => void;
}) {
  const [filters, setFilters] = useState<WorkspaceFilters>(() => loadSavedFilters(job.id));
  const deferredFilters = useDeferredValue(filters);
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [drawerId, setDrawerId] = useState<string | null>(null);
  const [busyId, setBusyId] = useState<string | null>(null);
  const [bulkBusy, setBulkBusy] = useState(false);
  const [confirmReject, setConfirmReject] = useState<Candidate | null>(null);
  const [confirmBulk, setConfirmBulk] = useState<CandidateStatus | null>(null);
  const [, startTransition] = useTransition();
  const [page, setPage] = useState(1);
  const pageSize = 100;

  const semanticLookup = useMemo(() => {
    const map = new Map<string, SemanticMatch>();
    (semantic?.matches ?? []).forEach((m) => map.set(m.candidate_id, m));
    return map;
  }, [semantic]);

  const ranked = useMemo<RankedApplicant[]>(() => {
    return [...candidates]
      .map((candidate) => ({ candidate, match: semanticLookup.get(candidate.id) }))
      .sort((a, b) => {
        const sa = a.match?.ai_match_score ?? -1;
        const sb = b.match?.ai_match_score ?? -1;
        if (sb !== sa) return sb - sa;
        return (b.match?.similarity_score ?? -1) - (a.match?.similarity_score ?? -1);
      });
  }, [candidates, semanticLookup]);

  const filtered = useMemo(
    () => applyFilters(ranked, deferredFilters),
    [ranked, deferredFilters],
  );

  const totalPages = Math.max(1, Math.ceil(filtered.length / pageSize));
  const pageSafe = Math.min(page, totalPages);
  const pageRows = useMemo(() => {
    const start = (pageSafe - 1) * pageSize;
    return filtered.slice(start, start + pageSize);
  }, [filtered, pageSafe]);

  const summaryCards = useMemo(
    () => [
      { label: "Applicants", value: candidates.length },
      { label: "AI Shortlisted", value: candidates.filter((c) => c.status === "ai_shortlisted").length },
      {
        label: "Recruiter Shortlisted",
        value: candidates.filter((c) => isRecruiterShortlisted(c.status)).length,
      },
      { label: "Interview", value: candidates.filter((c) => c.status === "interview").length },
      { label: "Selected", value: candidates.filter((c) => c.status === "selected").length },
      { label: "Rejected", value: candidates.filter((c) => c.status === "rejected").length },
    ],
    [candidates],
  );

  const drawerRow = useMemo(
    () => ranked.find((r) => r.candidate.id === drawerId) ?? null,
    [ranked, drawerId],
  );

  async function handleStage(candidate: Candidate, status: CandidateStatus) {
    setBusyId(candidate.id);
    try {
      const updated = await updateCandidate(candidate.id, buildPayload(candidate, status));
      setCandidates((curr) => curr.map((c) => (c.id === updated.id ? updated : c)));
      onToast(`${candidate.name} → ${status.replace(/_/g, " ")}`);
    } catch (err) {
      const message = err instanceof ApiClientError ? err.message : "Failed to update stage";
      onToast(message, "error");
    } finally {
      setBusyId(null);
    }
  }

  async function runBulk(status: CandidateStatus) {
    const ids = [...selectedIds];
    if (!ids.length) return;
    setBulkBusy(true);
    let ok = 0;
    let fail = 0;
    for (const id of ids) {
      const candidate = candidates.find((c) => c.id === id);
      if (!candidate) continue;
      try {
        const updated = await updateCandidate(candidate.id, buildPayload(candidate, status));
        setCandidates((curr) => curr.map((c) => (c.id === updated.id ? updated : c)));
        ok += 1;
      } catch {
        fail += 1;
      }
    }
    setBulkBusy(false);
    setSelectedIds(new Set());
    setConfirmBulk(null);
    onToast(`Bulk ${status.replace(/_/g, " ")}: ${ok} updated${fail ? `, ${fail} failed` : ""}`);
  }

  return (
    <div className="space-y-2">
      <div className="flex flex-wrap items-end justify-between gap-2">
        <div>
          <h3 className="ats-section-title">Recruiter Workspace</h3>
          <p className="mt-1 text-sm text-slate-500">
            Enterprise review for this job — ranked by Overall AI Match. Click a card to open the
            drawer.
          </p>
        </div>
        {semantic?.weights && (
          <p className="text-xs text-slate-500">
            Weights · semantic {Math.round(semantic.weights.semantic * 100)}% · skills{" "}
            {Math.round(semantic.weights.skills * 100)}% · exp{" "}
            {Math.round(semantic.weights.experience * 100)}%
          </p>
        )}
      </div>

      <div className="grid gap-2 sm:grid-cols-2 xl:grid-cols-6">
        {summaryCards.map((card) => (
          <div
            key={card.label}
            className="rounded-[12px] border border-slate-200/80 bg-gradient-to-b from-slate-50 to-white px-2 py-1.5"
          >
            <p className="text-[11px] font-medium uppercase tracking-wide text-slate-500">
              {card.label}
            </p>
            <p className="mt-1 text-xl font-semibold tabular-nums text-slate-900">{card.value}</p>
          </div>
        ))}
      </div>

      <div className="mt-2">
        <RecruiterFilters
        filters={filters}
        resultCount={filtered.length}
        onChange={(next) => {
          startTransition(() => {
            setFilters(next);
            setPage(1);
          });
        }}
        onSave={() => {
          saveFilters(job.id, filters);
          onToast("Filters saved for this job");
        }}
        onReset={() => setPage(1)}
        />
      </div>

      <div className="mt-2">
        <BulkBar
        count={selectedIds.size}
        busy={bulkBusy}
        onClear={() => setSelectedIds(new Set())}
        onExport={() => {
          const rows = filtered.filter((r) => selectedIds.has(r.candidate.id));
          exportCandidatesCsv(rows.length ? rows : filtered);
          onToast("CSV exported");
        }}
        onBulkStage={(status) => {
          if (status === "rejected") setConfirmBulk(status);
          else void runBulk(status);
        }}
      />
      </div>

      {loading ? (
        <div className="space-y-2">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-20 w-full rounded-[12px]" />
          ))}
        </div>
      ) : filtered.length === 0 ? (
        <EmptyState
          title={candidates.length === 0 ? "No applicants yet" : "No matches for these filters"}
          description={
            candidates.length === 0
              ? "Link candidates to this job to review AI Match, stages, and notes."
              : "Try clearing filters or lowering the AI Match threshold."
          }
        />
      ) : (
        <>
          <VirtualList
            items={pageRows}
            height={Math.min(560, Math.max(200, pageRows.length * 92))}
            estimateSize={92}
            renderItem={(row, index) => (
              <CandidateCard
                row={row}
                rank={(pageSafe - 1) * pageSize + index + 1}
                selected={selectedIds.has(row.candidate.id)}
                busy={busyId === row.candidate.id || bulkBusy}
                onToggleSelect={() => {
                  setSelectedIds((prev) => {
                    const next = new Set(prev);
                    if (next.has(row.candidate.id)) next.delete(row.candidate.id);
                    else next.add(row.candidate.id);
                    return next;
                  });
                }}
                onOpen={() => setDrawerId(row.candidate.id)}
                onStage={(status) => void handleStage(row.candidate, status)}
                onResume={() => {
                  if (!row.candidate.resume_url) return;
                  viewResumeFile(row.candidate.resume_url).catch(() =>
                    onToast("Unable to open resume", "error"),
                  );
                }}
                onConfirmReject={() => setConfirmReject(row.candidate)}
              />
            )}
          />

          {totalPages > 1 && (
            <div className="flex flex-wrap items-center justify-between gap-2 pt-0.5">
              <p className="text-xs text-slate-500">
                Page {pageSafe} of {totalPages} · showing {pageRows.length} of {filtered.length}
              </p>
              <div className="flex gap-2">
                <button
                  type="button"
                  className="rounded-lg px-3 py-1.5 text-sm ring-1 ring-slate-200 disabled:opacity-40"
                  disabled={pageSafe <= 1}
                  onClick={() => setPage((p) => Math.max(1, p - 1))}
                >
                  Previous
                </button>
                <button
                  type="button"
                  className="rounded-lg px-3 py-1.5 text-sm ring-1 ring-slate-200 disabled:opacity-40"
                  disabled={pageSafe >= totalPages}
                  onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                >
                  Next
                </button>
              </div>
            </div>
          )}
        </>
      )}

      <CandidateDrawer
        open={Boolean(drawerId)}
        row={drawerRow}
        job={job}
        comparePool={ranked}
        onClose={() => setDrawerId(null)}
        onToast={onToast}
      />

      <ConfirmDialog
        open={Boolean(confirmReject)}
        title="Reject candidate?"
        description={
          confirmReject
            ? `Reject ${confirmReject.name}? This moves them to the Rejected stage.`
            : ""
        }
        confirmLabel="Reject"
        danger
        loading={busyId === confirmReject?.id}
        onCancel={() => setConfirmReject(null)}
        onConfirm={() => {
          if (!confirmReject) return;
          void handleStage(confirmReject, "rejected").then(() => setConfirmReject(null));
        }}
      />

      <ConfirmDialog
        open={Boolean(confirmBulk)}
        title="Bulk reject?"
        description={`Reject ${selectedIds.size} selected candidates?`}
        confirmLabel="Reject all"
        danger
        loading={bulkBusy}
        onCancel={() => setConfirmBulk(null)}
        onConfirm={() => void runBulk("rejected")}
      />
    </div>
  );
}
