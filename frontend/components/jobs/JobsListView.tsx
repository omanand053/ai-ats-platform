"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useCallback, useEffect, useMemo, useState } from "react";
import { formatDate, formatEmploymentType, StatusBadge } from "@/components/jobs/job-utils";
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
import type { Job, JobStatus } from "@/lib/job-types";
import { listJobs } from "@/services/job.service";

export function JobsListView() {
  const ready = useRequireAuth();
  const router = useRouter();
  const { toasts, show, dismiss } = useToast();
  const [jobs, setJobs] = useState<Job[]>([]);
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [total, setTotal] = useState(0);
  const [search, setSearch] = useState("");
  const [statusFilter, setStatusFilter] = useState<"" | JobStatus>("");

  const fetchJobs = useCallback(async () => {
    setLoading(true);
    try {
      const searching = search.trim().length > 0;
      const result = await listJobs({
        page: searching ? 1 : page,
        limit: searching ? 100 : 10,
        status: statusFilter || undefined,
      });
      setJobs(result.jobs);
      setTotalPages(result.total_pages);
      setTotal(result.total);
    } catch (err) {
      const message = err instanceof ApiClientError ? err.message : "Failed to load jobs";
      show(message, "error");
    } finally {
      setLoading(false);
    }
  }, [page, search, statusFilter, show]);

  useEffect(() => {
    if (!ready) return;
    fetchJobs();
  }, [ready, fetchJobs]);

  const filteredJobs = useMemo(() => {
    const q = search.trim().toLowerCase();
    if (!q) return jobs;
    return jobs.filter((job) => job.title.toLowerCase().includes(q));
  }, [jobs, search]);

  if (!ready) {
    return <Spinner />;
  }

  return (
    <>
      <>
        <PageHeader
          title="Jobs"
          subtitle={`${total} role${total !== 1 ? "s" : ""} in your pipeline`}
          actions={
            <Button className="w-auto px-5" onClick={() => router.push("/dashboard/jobs/new")}>
              + New Job
            </Button>
          }
        />

        <Card className="mb-4" padding="sm">
          <div className="flex flex-col gap-3 sm:flex-row">
            <input
              type="search"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Search by title..."
              className="ats-input flex-1"
              aria-label="Search jobs by title"
            />
            <select
              value={statusFilter}
              onChange={(e) => {
                setStatusFilter(e.target.value as "" | JobStatus);
                setPage(1);
              }}
              className="ats-input sm:w-44"
              aria-label="Filter by status"
            >
              <option value="">All statuses</option>
              <option value="draft">Draft</option>
              <option value="open">Open</option>
              <option value="closed">Closed</option>
            </select>
          </div>
        </Card>

        <Card padding="none" className="overflow-hidden">
          {loading ? (
            <TableSkeleton rows={6} />
          ) : filteredJobs.length === 0 ? (
            <EmptyState
              title="No jobs found"
              description="Create a role to start matching candidates and running AI screening."
              actionLabel="Create your first job"
              onAction={() => router.push("/dashboard/jobs/new")}
            />
          ) : (
            <div className="max-h-[min(70vh,60vh)] overflow-auto">
              <div className="grid gap-3 md:grid-cols-2 lg:grid-cols-3">
                {filteredJobs.map((job) => (
                  <Card key={job.id} padding="sm" hover>
                    <div className="flex items-start justify-between gap-3">
                      <div className="min-w-0">
                        <Link href={`/dashboard/jobs/${job.id}`} className="ats-link text-sm font-semibold">
                          {job.title}
                        </Link>
                        <p className="mt-1 text-xs text-[var(--text-muted)]">{job.department || "—"} · {job.location || "Remote"}</p>
                      </div>
                      <div className="text-right">
                        <StatusBadge status={job.status} />
                        <p className="mt-2 text-xs text-[var(--text-muted)]">{formatDate(job.created_at)}</p>
                      </div>
                    </div>
                    <div className="mt-3 flex items-center justify-between">
                      <div className="text-xs text-[var(--text-muted)]">{formatEmploymentType(job.employment_type)}</div>
                      <Link href={`/dashboard/jobs/${job.id}/edit`} className="ats-link text-sm">Edit</Link>
                    </div>
                  </Card>
                ))}
              </div>
            </div>
          )}
        </Card>

        {!search.trim() && totalPages > 1 && (
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
