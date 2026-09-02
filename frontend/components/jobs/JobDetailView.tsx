"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { DeleteJobModal } from "@/components/jobs/DeleteJobModal";
import { formatDate, formatEmploymentType, StatusBadge } from "@/components/jobs/job-utils";
import { RecruiterWorkspace } from "@/components/jobs/recruiter/RecruiterWorkspace";
import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { Spinner } from "@/components/ui/Spinner";
import { ToastContainer } from "@/components/ui/Toast";
import { useRequireAuth } from "@/hooks/useRequireAuth";
import { useToast } from "@/hooks/useToast";
import { ApiClientError } from "@/lib/api-client";
import type { Candidate } from "@/lib/candidate-types";
import type { Job } from "@/lib/job-types";
import { listCandidatesByJob } from "@/services/candidate.service";
import {
  askJobAIAssistant,
  deleteJob,
  getJob,
  getJobSemanticMatches,
  type AIAssistantResponse,
  type SemanticMatchResult,
} from "@/services/job.service";

function DetailRow({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="border-b border-slate-100 py-3 last:border-0">
      <dt className="ats-label">{label}</dt>
      <dd className="mt-1 text-sm font-semibold text-slate-900">{value}</dd>
    </div>
  );
}

function SkillChip({ children }: { children: React.ReactNode }) {
  return (
    <span className="inline-flex rounded-lg bg-blue-50 px-2 py-0.5 text-xs font-semibold text-blue-700 ring-1 ring-blue-100">
      {children}
    </span>
  );
}

function formatPct(value?: number | null) {
  if (typeof value !== "number" || Number.isNaN(value)) return "—";
  if (value > 0 && value < 1) return `${value.toFixed(1)}%`;
  return `${Math.round(value)}%`;
}

export function JobDetailView({ jobId }: { jobId: string }) {
  const ready = useRequireAuth();
  const router = useRouter();
  const { toasts, show, dismiss } = useToast();
  const [job, setJob] = useState<Job | null>(null);
  const [semantic, setSemantic] = useState<SemanticMatchResult | null>(null);
  const [candidates, setCandidates] = useState<Candidate[]>([]);
  const [loading, setLoading] = useState(true);
  const [workspaceLoading, setWorkspaceLoading] = useState(true);
  const [deleting, setDeleting] = useState(false);
  const [showDelete, setShowDelete] = useState(false);
  const [question, setQuestion] = useState("");
  const [asking, setAsking] = useState(false);
  const [assistant, setAssistant] = useState<AIAssistantResponse | null>(null);

  useEffect(() => {
    if (!ready) return;

    const loadPage = async () => {
      setLoading(true);
      setWorkspaceLoading(true);

      try {
        const jobData = await getJob(jobId);
        setJob(jobData);
      } catch (err) {
        const message = err instanceof ApiClientError ? err.message : "Failed to load job";
        show(message, "error");
        setJob(null);
        setSemantic(null);
        setCandidates([]);
        setLoading(false);
        setWorkspaceLoading(false);
        return;
      } finally {
        setLoading(false);
      }

      const [semanticSettled, candidatesSettled] = await Promise.allSettled([
        getJobSemanticMatches(jobId, { top_k: 200 }),
        listCandidatesByJob(jobId, { limit: 500, page: 1, sort: "score" }),
      ]);

      if (semanticSettled.status === "fulfilled") {
        setSemantic(semanticSettled.value);
      } else {
        setSemantic(null);
        const err = semanticSettled.reason;
        show(
          err instanceof ApiClientError ? err.message : "Failed to load semantic matches",
          "error",
        );
      }

      if (candidatesSettled.status === "fulfilled") {
        setCandidates(candidatesSettled.value.candidates);
      } else {
        setCandidates([]);
        const err = candidatesSettled.reason;
        show(err instanceof ApiClientError ? err.message : "Failed to load candidates", "error");
      }
      setWorkspaceLoading(false);
    };

    loadPage();
  }, [jobId, ready, show]);

  async function handleDelete() {
    if (!job) return;
    setDeleting(true);
    try {
      await deleteJob(job.id);
      show("Job deleted successfully");
      setShowDelete(false);
      router.push("/dashboard/jobs");
    } catch (err) {
      show(err instanceof ApiClientError ? err.message : "Failed to delete job", "error");
    } finally {
      setDeleting(false);
    }
  }

  async function handleAskAssistant(e: React.FormEvent) {
    e.preventDefault();
    if (!job || !question.trim() || asking) return;
    setAsking(true);
    try {
      const result = await askJobAIAssistant(job.id, { question: question.trim(), top_k: 5 });
      setAssistant(result);
    } catch (err) {
      show(
        err instanceof ApiClientError ? err.message : "Failed to get AI assistant answer",
        "error",
      );
    } finally {
      setAsking(false);
    }
  }

  if (!ready || loading) {
    return <Spinner label="Loading job..." />;
  }

  if (!job) {
    return (
      <>
        <>
          <Card className="text-center">
            <p className="text-sm text-red-600">Job not found</p>
            <Button
              variant="secondary"
              className="mx-auto mt-4 w-auto px-5"
              onClick={() => router.push("/dashboard/jobs")}
            >
              Back to jobs
            </Button>
          </Card>
        </>
        <ToastContainer toasts={toasts} onDismiss={dismiss} />
      </>
    );
  }

  return (
    <>
      <>
        <div className="mb-6 flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
          <div>
            <Link href="/dashboard/jobs" className="ats-link text-sm">
              ← Back to jobs
            </Link>
            <div className="mt-2 flex flex-wrap items-center gap-3">
              <h2 className="ats-page-title">{job.title}</h2>
              <StatusBadge status={job.status} />
            </div>
          </div>
          <div className="flex flex-wrap gap-2">
            <Button
              variant="secondary"
              className="w-auto px-4"
              onClick={() => router.push(`/dashboard/jobs/${job.id}/edit`)}
            >
              Edit
            </Button>
            <Button variant="danger" className="w-auto px-4" onClick={() => setShowDelete(true)}>
              Delete
            </Button>
          </div>
        </div>

        <Card padding="lg">
          <dl>
            <DetailRow label="Department" value={job.department || "—"} />
            <DetailRow label="Location" value={job.location || "—"} />
            <DetailRow label="Employment Type" value={formatEmploymentType(job.employment_type)} />
            <DetailRow label="Experience Required" value={job.experience_required || "—"} />
            <DetailRow
              label="Required Skills"
              value={
                job.required_skills?.length ? (
                  <div className="mt-1 flex flex-wrap gap-2">
                    {job.required_skills.map((skill) => (
                      <SkillChip key={skill}>{skill}</SkillChip>
                    ))}
                  </div>
                ) : (
                  "—"
                )
              }
            />
            <DetailRow
              label="Description"
              value={
                job.description ? (
                  <span className="font-normal leading-6 text-slate-700 whitespace-pre-wrap">
                    {job.description}
                  </span>
                ) : (
                  "—"
                )
              }
            />
            <DetailRow label="Created" value={formatDate(job.created_at)} />
            <DetailRow label="Last Updated" value={formatDate(job.updated_at)} />
          </dl>
        </Card>

        <Card className="mt-6" padding="lg">
          <RecruiterWorkspace
            job={job}
            candidates={candidates}
            setCandidates={setCandidates}
            semantic={semantic}
            loading={workspaceLoading}
            onToast={(message, tone) => show(message, tone === "error" ? "error" : "success")}
          />
        </Card>

        <Card className="mt-6" padding="lg">
          <h3 className="ats-section-title">AI Assistant</h3>
          <p className="mt-1 text-sm text-slate-500">
            Ask natural-language questions about candidates for this job. Answers use Top-K
            semantic resume retrieval (RAG) only — not the full candidate database.
          </p>

          <form onSubmit={handleAskAssistant} className="mt-4 space-y-3">
            <textarea
              value={question}
              onChange={(e) => setQuestion(e.target.value)}
              rows={3}
              placeholder="e.g. Who has the strongest Go and PostgreSQL background?"
                className="ats-input"
                aria-label="Ask the AI assistant"
            />
            <Button type="submit" className="w-auto px-5" loading={asking} disabled={!question.trim()}>
              Ask assistant
            </Button>
          </form>

          {assistant && (
            <div className="mt-5 space-y-4">
              {!assistant.ai_available && (
                <div
                  role="status"
                  className="rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-950"
                >
                  <p className="font-medium">
                    AI response unavailable. Displaying retrieved candidate data.
                  </p>
                  {assistant.message && <p className="mt-1 text-amber-900/80">{assistant.message}</p>}
                </div>
              )}

              {assistant.ai_available && assistant.answer && (
                <div className="rounded-xl bg-slate-50 px-4 py-4 ring-1 ring-slate-200">
                  <p className="ats-label">Answer</p>
                  <p className="mt-2 whitespace-pre-wrap text-sm leading-6 text-slate-900">
                    {assistant.answer}
                  </p>
                </div>
              )}

              {assistant.referenced_candidates?.length ? (
                <div>
                  <p className="text-sm font-semibold text-slate-800">Retrieved candidates</p>
                  <ul className="mt-2 divide-y divide-slate-100 overflow-hidden rounded-xl ring-1 ring-slate-200">
                    {assistant.referenced_candidates.map((c) => (
                      <li
                        key={`${c.candidate_id}-${c.resume_id}`}
                        className="flex items-center justify-between gap-3 bg-white px-4 py-3"
                      >
                        <span className="text-sm font-medium text-slate-900">{c.candidate_name}</span>
                        <span className="text-xs text-slate-500">
                          Similarity {formatPct(c.similarity_score)}
                        </span>
                      </li>
                    ))}
                  </ul>
                </div>
              ) : null}
            </div>
          )}
        </Card>
      </>

      {showDelete && (
        <DeleteJobModal
          jobTitle={job.title}
          loading={deleting}
          onConfirm={handleDelete}
          onCancel={() => setShowDelete(false)}
        />
      )}
      <ToastContainer toasts={toasts} onDismiss={dismiss} />
    </>
  );
}
