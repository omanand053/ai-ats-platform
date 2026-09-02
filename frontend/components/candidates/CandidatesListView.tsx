"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { useCallback, useEffect, useState } from "react";
import { CandidateStatusBadge, formatDate } from "@/components/candidates/candidate-utils";
import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { EmptyState } from "@/components/ui/EmptyState";
import { Spinner } from "@/components/ui/Spinner";
import { PageHeader } from "@/components/ui/PageHeader";
import { TableSkeleton } from "@/components/ui/Skeleton";
import { ToastContainer } from "@/components/ui/Toast";
import { useRequireAuth } from "@/hooks/useRequireAuth";
import { useToast } from "@/hooks/useToast";
import { ApiClientError } from "@/lib/api-client";
import type { Candidate, CandidateStatus } from "@/lib/candidate-types";
import { listCandidates } from "@/services/candidate.service";
import { listJobs } from "@/services/job.service";

export function CandidatesListView() {
  const ready = useRequireAuth();
  const router = useRouter();
  const searchParams = useSearchParams();
  const { toasts, show, dismiss } = useToast();
  const [candidates, setCandidates] = useState<Candidate[]>([]);
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [total, setTotal] = useState(0);
  const [search, setSearch] = useState("");
  const [searchInput, setSearchInput] = useState("");
  const [statusFilter, setStatusFilter] = useState<"" | CandidateStatus>("");
  const [jobFilter, setJobFilter] = useState<string>("");
  const [jobs, setJobs] = useState<{ id: string; title: string }[]>([]);
  const [sort, setSort] = useState<"created_at" | "score">("score");

  useEffect(() => {
    const q = searchParams.get("q")?.trim() ?? "";
    if (q) {
      setSearchInput(q);
      setSearch(q);
      setPage(1);
    }
  }, [searchParams]);

  const fetchCandidates = useCallback(async () => {
    setLoading(true);
    try {
      const result = await listCandidates({
        page,
        limit: 10,
        status: statusFilter || undefined,
        search: search.trim() || undefined,
        job_id: jobFilter || undefined,
        sort,
      });
      setCandidates(result.candidates);
      setTotalPages(result.total_pages);
      setTotal(result.total);
    } catch (err) {
      const message = err instanceof ApiClientError ? err.message : "Failed to load candidates";
      show(message, "error");
    } finally {
      setLoading(false);
    }
  }, [page, search, statusFilter, sort, jobFilter, show]);

  useEffect(() => {
    void (async function () {
      try {
        const res = await listJobs({ limit: 200 });
        setJobs((res.jobs || []).map((j: any) => ({ id: j.id, title: j.title })));
      } catch (e) {
        // ignore
      }
    })();
  }, []);

  useEffect(() => {
    if (!ready) return;
    fetchCandidates();
  }, [ready, fetchCandidates]);

  function handleSearch(e: React.FormEvent) {
    e.preventDefault();
    setPage(1);
    setSearch(searchInput);
  }

  if (!ready) {
    return <Spinner />;
  }

  return (
    <>
      <>
        <PageHeader
          title="Candidates"
          subtitle={`${total} candidate${total !== 1 ? "s" : ""} across your pipeline`}
          actions={
            <Button className="w-auto px-5" onClick={() => router.push("/dashboard/candidates/new")}>
              + Add Candidate
            </Button>
          }
        />

        <Card className="mb-4" padding="sm">
          <form onSubmit={handleSearch} className="flex flex-col gap-3 lg:flex-row">
            <input
              type="search"
              value={searchInput}
              onChange={(e) => setSearchInput(e.target.value)}
              placeholder="Search by name or email..."
              className="ats-input flex-1"
              aria-label="Search candidates"
            />
            <select
              value={statusFilter}
              onChange={(e) => {
                setStatusFilter(e.target.value as "" | CandidateStatus);
                setPage(1);
              }}
              className="ats-input lg:w-44"
              aria-label="Filter by status"
            >
              <option value="">All statuses</option>
              <option value="applied">Applied</option>
              <option value="screening">Screening</option>
              <option value="shortlisted">Shortlisted</option>
              <option value="ai_shortlisted">AI Shortlisted</option>
              <option value="recruiter_shortlisted">Recruiter Shortlisted</option>
              <option value="interview">Interview</option>
              <option value="selected">Selected</option>
              <option value="offer">Offer</option>
              <option value="hired">Hired</option>
              <option value="rejected">Rejected</option>
            </select>
            <select
              value={jobFilter}
              onChange={(e) => {
                setJobFilter(e.target.value);
                setPage(1);
              }}
              className="ats-input lg:w-56"
              aria-label="Filter by job"
            >
              <option value="">All jobs</option>
              {jobs.map((j) => (
                <option key={j.id} value={j.id}>
                  {j.title}
                </option>
              ))}
            </select>
            <select
              value={sort}
              onChange={(e) => {
                setSort(e.target.value as "created_at" | "score");
                setPage(1);
              }}
              className="ats-input lg:w-44"
              aria-label="Sort candidates"
            >
              <option value="score">Highest score</option>
              <option value="created_at">Newest first</option>
            </select>
            <Button type="submit" variant="secondary" className="w-auto px-5">
              Search
            </Button>
          </form>
        </Card>

        <Card padding="none" className="overflow-hidden">
          {loading ? (
            <TableSkeleton rows={6} />
          ) : candidates.length === 0 ? (
            <EmptyState
              title="No candidates found"
              description="Add a candidate or upload a resume. Job fit and semantic scores live in each Job Workspace."
              actionLabel="Add your first candidate"
              onAction={() => router.push("/dashboard/candidates/new")}
            />
          ) : (
            <div className="ats-table-wrap max-h-[min(70vh,36rem)] border-0">
              <table className="ats-table min-w-full">
                <thead>
                  <tr>
                    <th>Name</th>
                    <th className="hidden sm:table-cell">Email</th>
                    <th className="hidden md:table-cell">Location</th>
                    <th>Status</th>
                    <th className="hidden lg:table-cell">Job</th>
                    <th className="hidden lg:table-cell">Added</th>
                    <th className="text-right">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {candidates.map((candidate) => (
                    <tr key={candidate.id}>
                      <td>
                        <Link
                          href={`/dashboard/candidates/${candidate.id}`}
                          className="ats-link"
                        >
                          {candidate.name}
                        </Link>
                        <p className="mt-0.5 text-xs text-slate-500 sm:hidden">{candidate.email}</p>
                      </td>
                      <td className="hidden sm:table-cell">{candidate.email}</td>
                      <td className="hidden md:table-cell">{candidate.location || "—"}</td>
                      <td>
                        <CandidateStatusBadge status={candidate.status} />
                      </td>
                      <td className="hidden lg:table-cell">
                        {candidate.job_id ? (
                          <Link href={`/dashboard/jobs/${candidate.job_id}`} className="ats-link text-sm">
                            Workspace
                          </Link>
                        ) : (
                          <span className="text-sm text-slate-400">—</span>
                        )}
                      </td>
                      <td className="hidden text-slate-500 lg:table-cell">
                        {formatDate(candidate.created_at)}
                      </td>
                      <td className="text-right">
                        <Link
                          href={`/dashboard/candidates/${candidate.id}/edit`}
                          className="text-sm font-medium text-slate-600 transition hover:text-slate-900"
                        >
                          Edit
                        </Link>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </Card>

        {totalPages > 1 && (
          <div className="mt-4 flex items-center justify-between gap-3 rounded-[var(--radius-lg)] border border-[var(--border)] bg-[var(--surface)] px-3 py-2 shadow-[var(--shadow-sm)]">
            <Button
              variant="ghost"
              size="sm"
              className="w-auto"
              disabled={page <= 1 || loading}
              onClick={() => setPage((p) => p - 1)}
            >
              Previous
            </Button>
            <span className="text-sm font-medium text-[var(--text-secondary)]">
              Page {page} of {totalPages}
            </span>
            <Button
              variant="ghost"
              size="sm"
              className="w-auto"
              disabled={page >= totalPages || loading}
              onClick={() => setPage((p) => p + 1)}
            >
              Next
            </Button>
          </div>
        )}
      </>
      <ToastContainer toasts={toasts} onDismiss={dismiss} />
    </>
  );
}
