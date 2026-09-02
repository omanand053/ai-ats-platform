"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { DashboardShell } from "@/components/dashboard/DashboardShell";
import { DeleteCandidateModal } from "@/components/candidates/DeleteCandidateModal";
import {
  CandidateStatusBadge,
  formatDate,
  ProcessingStatusBadge,
} from "@/components/candidates/candidate-utils";
import { Avatar } from "@/components/ui/Avatar";
import { Button } from "@/components/ui/Button";
import { ToastContainer } from "@/components/ui/Toast";
import { useRequireAuth } from "@/hooks/useRequireAuth";
import { useToast } from "@/hooks/useToast";
import { ApiClientError } from "@/lib/api-client";
import type { Candidate } from "@/lib/candidate-types";
import { deleteCandidate, getCandidate } from "@/services/candidate.service";

function DetailRow({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="py-2">
      <dt className="text-xs font-medium text-[#6b7280]">{label}</dt>
      <dd className="mt-1 text-sm font-semibold text-[#0b1220]">{value}</dd>
    </div>
  );
}

export function CandidateDetailView({ candidateId }: { candidateId: string }) {
  const ready = useRequireAuth();
  const router = useRouter();
  const { toasts, show, dismiss } = useToast();
  const [candidate, setCandidate] = useState<Candidate | null>(null);
  const [loading, setLoading] = useState(true);
  const [deleting, setDeleting] = useState(false);
  const [showDelete, setShowDelete] = useState(false);

  useEffect(() => {
    if (!ready) return;
    getCandidate(candidateId)
      .then(setCandidate)
      .catch((err) => {
        const message = err instanceof ApiClientError ? err.message : "Failed to load candidate";
        show(message, "error");
      })
      .finally(() => setLoading(false));
  }, [ready, candidateId, show]);

  async function handleDelete() {
    if (!candidate) return;
    setDeleting(true);
    try {
      await deleteCandidate(candidate.id);
      show("Candidate deleted successfully");
      router.push("/dashboard/candidates");
    } catch (err) {
      const message = err instanceof ApiClientError ? err.message : "Failed to delete candidate";
      show(message, "error");
    } finally {
      setDeleting(false);
      setShowDelete(false);
    }
  }

  if (!ready || loading) {
    return (
      <div className="flex min-h-full items-center justify-center">
        <p className="text-sm text-zinc-500">Loading candidate...</p>
      </div>
    );
  }

  if (!candidate) {
    return (
      <DashboardShell>
        <div className="rounded-2xl bg-white p-8 text-center shadow-sm ring-1 ring-zinc-200">
          <p className="text-sm text-red-600">Candidate not found</p>
          <Button
            variant="secondary"
            className="mx-auto mt-4 w-auto px-5"
            onClick={() => router.push("/dashboard/candidates")}
          >
            Back to candidates
          </Button>
        </div>
      </DashboardShell>
    );
  }

  return (
    <>
      <DashboardShell>
        <div className="mb-4 flex items-center justify-between gap-4">
          <div className="flex items-center gap-4">
            <Avatar name={candidate.name} size="lg" />
            <div>
              <Link href="/dashboard/candidates" className="text-sm font-medium text-[#4f46e5] hover:underline">
                ← Back to candidates
              </Link>
              <div className="mt-1 flex items-center gap-3">
                <h2 className="text-xl font-bold text-[#0b1220]">{candidate.name}</h2>
                <CandidateStatusBadge status={candidate.status} />
              </div>
              <p className="mt-1 text-sm text-[#6b7280]">{candidate.email}</p>
            </div>
          </div>
          <div className="flex gap-2">
            <Button
              variant="secondary"
              className="w-auto px-3 py-1"
              onClick={() => router.push(`/dashboard/candidates/${candidate.id}/edit`)}
            >
              Edit
            </Button>
            <Button variant="danger" className="w-auto px-3 py-1" onClick={() => setShowDelete(true)}>
              Delete
            </Button>
          </div>
        </div>

        <div className="rounded-2xl bg-white p-4 shadow-sm ring-1 ring-zinc-200 sm:p-6">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <DetailRow label="Phone" value={candidate.phone || "—"} />
              <DetailRow
                label="Experience"
                value={
                  candidate.experience_years !== undefined
                    ? `${candidate.experience_years} years`
                    : "—"
                }
              />
              <DetailRow label="Current Company" value={candidate.current_company || "—"} />
              <DetailRow label="Designation" value={candidate.current_designation || "—"} />
              <DetailRow label="Location" value={candidate.location || "—"} />
              <DetailRow label="Source" value={candidate.source || "—"} />
            </div>
            <div>
              <DetailRow
                label="Job"
                value={
                  candidate.job_id ? (
                    <Link
                      href={`/dashboard/jobs/${candidate.job_id}`}
                      className="font-normal text-[#4f46e5] hover:underline"
                    >
                      {candidate.job_id}
                    </Link>
                  ) : (
                    "—"
                  )
                }
              />
              <DetailRow
                label="Skills"
                value={
                  candidate.skills?.length ? (
                    <div className="mt-1 flex flex-wrap gap-2">
                      {candidate.skills.map((skill) => (
                        <span
                          key={skill}
                          className="rounded-full bg-indigo-50 px-2.5 py-0.5 text-xs font-medium text-indigo-700"
                        >
                          {skill}
                        </span>
                      ))}
                    </div>
                  ) : (
                    "—"
                  )
                }
              />
              <DetailRow
                label="Parsing Status"
                value={<ProcessingStatusBadge status={candidate.parsing_status} />}
              />
              <DetailRow
                label="Embedding Status"
                value={<ProcessingStatusBadge status={candidate.embedding_status} />}
              />
            </div>
          </div>

          <div className="mt-4">
            <DetailRow label="Resume URL" value={candidate.resume_url || "—"} />
            <DetailRow
              label="Resume Summary"
              value={
                candidate.resume_summary ? (
                  <span className="font-normal">{candidate.resume_summary}</span>
                ) : (
                  "—"
                )
              }
            />
            <div className="mt-2 flex gap-4 text-sm text-[#6b7280]">
              <div>Created: {formatDate(candidate.created_at)}</div>
              <div>Updated: {formatDate(candidate.updated_at)}</div>
            </div>
          </div>
        </div>
      </DashboardShell>

      {showDelete && (
        <DeleteCandidateModal
          candidateName={candidate.name}
          loading={deleting}
          onConfirm={handleDelete}
          onCancel={() => setShowDelete(false)}
        />
      )}
      <ToastContainer toasts={toasts} onDismiss={dismiss} />
    </>
  );
}
