"use client";

import Link from "next/link";
import { useCallback, useEffect, useState, type ReactNode } from "react";
import {
  BarChart,
  FunnelChart,
  TrendSparkline,
  DonutChart,
  PieChart,
  ChartEmpty,
} from "@/components/dashboard/charts";
import { Card } from "@/components/ui/Card";
import { Badge } from "@/components/ui/Badge";
import { ds, classNames } from "@/components/dashboard/designSystem";
import { PageHeader } from "@/components/ui/PageHeader";
import { DashboardSkeleton } from "@/components/ui/Skeleton";
import { ToastContainer } from "@/components/ui/Toast";
import { useRequireAuth } from "@/hooks/useRequireAuth";
import { useToast } from "@/hooks/useToast";
import { ApiClientError } from "@/lib/api-client";
import type { Candidate } from "@/lib/candidate-types";
import type { Job } from "@/lib/job-types";
import { listCandidates } from "@/services/candidate.service";
import { getAnalyticsOverview, type AnalyticsOverview } from "@/services/enterprise.service";
import { listJobs } from "@/services/job.service";

function emptyOverview(): AnalyticsOverview {
  return {
    total_jobs: 0,
    open_jobs: 0,
    closed_jobs: 0,
    applicants: 0,
    ai_shortlisted: 0,
    recruiter_shortlisted: 0,
    interviews: 0,
    offers: 0,
    selected: 0,
    rejected: 0,
    hired: 0,
    by_status: {},
    applications_per_job: [],
    top_skills: [],
    missing_skills: [],
    ai_match_distribution: [],
    hiring_trend: [],
    monthly_hiring: [],
    recruiter_productivity: [],
    funnel: [],
  };
}

function Kpi({
  label,
  value,
  hint,
  href,
  icon,
}: {
  label: string;
  value: string | number;
  hint?: string;
  href?: string;
  icon?: ReactNode;
}) {
  const content = (
    <>
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <p className="ats-label">{label}</p>
          <p className="mt-2 text-2xl font-semibold tabular-nums tracking-tight text-[var(--text-primary)]">
            {value}
          </p>
          {hint ? <p className="mt-1.5 text-xs text-[var(--text-muted)]">{hint}</p> : null}
        </div>
        {icon ? (
          <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-[var(--radius-md)] bg-[var(--surface-muted)] text-[var(--text-secondary)]">
            {icon}
          </div>
        ) : null}
      </div>
    </>
  );

  const classes = classNames(
    ds.kpiCard,
    "group block w-full transition hover:border-[var(--border-strong)] hover:shadow-[var(--shadow-md)]",
  );

  return href ? (
    <Link href={href} className={classes} aria-label={label}>
      {content}
    </Link>
  ) : (
    <div className={classes} role="group">
      {content}
    </div>
  );
}

function PanelEmpty({ title, message }: { title: string; message: string }) {
  return (
    <div
      className="flex min-h-[7rem] flex-col items-center justify-center rounded-[var(--radius-md)] border border-dashed border-[var(--border)] bg-[var(--surface-muted)]/40 px-4 py-6 text-center"
      role="status"
    >
      <p className="text-sm font-semibold text-[var(--text-primary)]">{title}</p>
      <p className="mt-1 max-w-xs text-sm text-[var(--text-muted)]">{message}</p>
    </div>
  );
}

function fmt(n?: number | null, digits = 0) {
  if (typeof n !== "number" || Number.isNaN(n)) return "—";
  return n.toFixed(digits);
}

function fmtDate(value?: string) {
  if (!value) return "—";
  try {
    return new Intl.DateTimeFormat("en", { month: "short", day: "numeric" }).format(new Date(value));
  } catch {
    return value;
  }
}

function pct(num: number, den: number) {
  if (!den || den <= 0) return "—";
  return `${((num / den) * 100).toFixed(1)}%`;
}

async function withTimeout<T>(promise: Promise<T>, ms = 3000): Promise<T | null> {
  const timeout = new Promise<null>((resolve) => {
    window.setTimeout(() => resolve(null), ms);
  });
  return Promise.race([promise, timeout]);
}

function IconBriefcase({ className = "h-4 w-4" }: { className?: string }) {
  return (
    <svg viewBox="0 0 24 24" className={className} fill="none" stroke="currentColor" strokeWidth="1.7" aria-hidden>
      <path strokeLinecap="round" strokeLinejoin="round" d="M9 6V5a2 2 0 012-2h2a2 2 0 012 2v1" />
      <rect x="3" y="7" width="18" height="13" rx="2" />
    </svg>
  );
}

function IconUsers({ className = "h-4 w-4" }: { className?: string }) {
  return (
    <svg viewBox="0 0 24 24" className={className} fill="none" stroke="currentColor" strokeWidth="1.7" aria-hidden>
      <circle cx="9" cy="7" r="3" />
      <path strokeLinecap="round" strokeLinejoin="round" d="M21 21v-2a4 4 0 00-4-4H6a4 4 0 00-4 4v2" />
    </svg>
  );
}

function IconTrend({ className = "h-4 w-4" }: { className?: string }) {
  return (
    <svg viewBox="0 0 24 24" className={className} fill="none" stroke="currentColor" strokeWidth="1.7" aria-hidden>
      <path strokeLinecap="round" strokeLinejoin="round" d="M3 17l6-6 4 4 7-7" />
      <path strokeLinecap="round" strokeLinejoin="round" d="M14 8h7v7" />
    </svg>
  );
}

function IconCalendar({ className = "h-4 w-4" }: { className?: string }) {
  return (
    <svg viewBox="0 0 24 24" className={className} fill="none" stroke="currentColor" strokeWidth="1.7" aria-hidden>
      <rect x="3" y="5" width="18" height="16" rx="2" />
      <path strokeLinecap="round" d="M16 3v4M8 3v4M3 11h18" />
    </svg>
  );
}

function IconSpark({ className = "h-4 w-4" }: { className?: string }) {
  return (
    <svg viewBox="0 0 24 24" className={className} fill="none" stroke="currentColor" strokeWidth="1.7" aria-hidden>
      <path strokeLinecap="round" strokeLinejoin="round" d="M12 3l1.5 5.5L19 10l-5.5 1.5L12 17l-1.5-5.5L5 10l5.5-1.5L12 3z" />
    </svg>
  );
}

export function DashboardView() {
  const ready = useRequireAuth();
  const { toasts, dismiss, show } = useToast();
  const [overview, setOverview] = useState<AnalyticsOverview | null>(null);
  const [recentJobs, setRecentJobs] = useState<Job[]>([]);
  const [recentCandidates, setRecentCandidates] = useState<Candidate[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [analytics, jobsResult, candidatesResult] = await Promise.all([
        withTimeout(getAnalyticsOverview()),
        withTimeout(listJobs({ limit: 5 })),
        withTimeout(listCandidates({ limit: 5 })),
      ]);

      setOverview(analytics ?? emptyOverview());
      setRecentJobs(jobsResult?.jobs ?? []);
      setRecentCandidates(candidatesResult?.candidates ?? []);
    } catch (err) {
      const message = err instanceof ApiClientError ? err.message : "Failed to load analytics";
      setError(message);
      setOverview(emptyOverview());
      setRecentJobs([]);
      setRecentCandidates([]);
      show(message, "error");
    } finally {
      setLoading(false);
    }
  }, [show]);

  useEffect(() => {
    if (!ready) return;
    void load();
  }, [load, ready]);

  if (!ready || loading) {
    return <DashboardSkeleton />;
  }

  const d = overview ?? emptyOverview();
  const interviewRate = pct(d.interviews, d.applicants);
  const offerRate = pct(d.offers, d.interviews > 0 ? d.interviews : d.applicants);
  const pipelineCount =
    (d.by_status?.applied ?? 0) +
    (d.by_status?.screening ?? 0) +
    (d.by_status?.shortlisted ?? 0) +
    (d.by_status?.ai_shortlisted ?? 0) +
    (d.by_status?.recruiter_shortlisted ?? 0) +
    (d.by_status?.interview ?? 0) +
    d.applicants;
  const velocity =
    typeof d.avg_time_to_hire_days === "number"
      ? `${fmt(d.avg_time_to_hire_days, 0)}d`
      : d.hired > 0
        ? "Active"
        : "—";
  const aiAvg =
    typeof d.avg_ai_match === "number" ? fmt(d.avg_ai_match, 1) : "—";

  const pipelineItems = (d.funnel ?? []).map((item) => ({ label: item.name, value: item.count }));
  const trendPoints = (d.hiring_trend ?? []).map((item) => ({ period: item.period, count: item.count }));
  const hasPipelineData = pipelineItems.length > 0;
  const hasTrendData = trendPoints.length > 0;

  return (
    <>
      <PageHeader
        title="Hiring overview"
        subtitle="Open roles, pipeline health, and what needs attention today."
        actions={
          <div className="flex flex-wrap gap-2">
            <Link
              href="/dashboard/jobs/new"
              className="inline-flex h-9 items-center rounded-[var(--radius-md)] border border-[var(--border-strong)] bg-[var(--surface)] px-3 text-xs font-semibold text-[var(--text-primary)] hover:bg-[var(--surface-muted)]"
            >
              New job
            </Link>
            <Link
              href="/dashboard/candidates/new"
              className="inline-flex h-9 items-center rounded-[var(--radius-md)] bg-[var(--accent)] px-3 text-xs font-semibold text-white hover:bg-[var(--accent-hover)]"
            >
              Add candidate
            </Link>
          </div>
        }
      />

      {error ? (
        <div className={classNames(ds.panel, "mb-5 px-4 py-3 text-sm")}>
          <p className="text-[var(--text-muted)]">
            Analytics is temporarily unavailable. Showing the latest available snapshot.
          </p>
          <button type="button" className="ats-link mt-1.5 text-sm" onClick={() => void load()}>
            Retry
          </button>
        </div>
      ) : null}

      <div className="space-y-6">
        <section aria-label="Key metrics">
          <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
            <Kpi
              href="/dashboard/jobs"
              icon={<IconBriefcase />}
              label="Open jobs"
              value={d.open_jobs}
              hint={`${d.total_jobs} total roles`}
            />
            <Kpi
              href="/dashboard/candidates"
              icon={<IconUsers />}
              label="Candidates in pipeline"
              value={d.applicants || pipelineCount}
              hint={`${d.ai_shortlisted} AI shortlisted`}
            />
            <Kpi
              href="/dashboard/reports"
              icon={<IconTrend />}
              label="Interview rate"
              value={interviewRate}
              hint={`${d.interviews} interviews scheduled`}
            />
            <Kpi
              href="/dashboard/reports"
              icon={<IconCalendar />}
              label="Offer rate"
              value={offerRate}
              hint={`${d.offers} offers · ${d.hired} hired`}
            />
          </div>
          <div className="mt-3 grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
            <Kpi
              icon={<IconTrend />}
              label="Hiring velocity"
              value={velocity}
              hint={
                typeof d.avg_time_to_hire_days === "number"
                  ? "Avg. days to hire"
                  : "Based on recent hires"
              }
            />
            <Kpi
              href="/dashboard/settings/ai"
              icon={<IconSpark />}
              label="AI match average"
              value={aiAvg}
              hint="Company-wide semantic fit"
            />
            <Kpi
              href="/dashboard/candidates"
              icon={<IconUsers />}
              label="Recruiter shortlisted"
              value={d.recruiter_shortlisted}
              hint="Ready for review"
            />
          </div>
        </section>

        <section className="grid gap-4 lg:grid-cols-[1.2fr_0.8fr]" aria-label="Funnel and trends">
          <Card
            padding="lg"
            header={
              <div>
                <h3 className="ats-section-title">Hiring funnel</h3>
                <p className="mt-0.5 text-sm text-[var(--text-muted)]">
                  Where candidates sit across stages
                </p>
              </div>
            }
          >
            {hasPipelineData ? (
              <FunnelChart
                items={pipelineItems}
                emptyMessage="Funnel data will appear as candidates progress."
              />
            ) : (
              <PanelEmpty
                title="No funnel activity yet"
                message="Open a job and add applicants to see stage progression."
              />
            )}
          </Card>

          <Card
            padding="lg"
            header={
              <div>
                <h3 className="ats-section-title">Hiring trend</h3>
                <p className="mt-0.5 text-sm text-[var(--text-muted)]">Recent activity over time</p>
              </div>
            }
          >
            <div className="space-y-4">
              {hasTrendData ? (
                <TrendSparkline
                  points={trendPoints}
                  emptyMessage="Trend data will appear when activity is reported."
                />
              ) : (
                <PanelEmpty
                  title="No trend data yet"
                  message="Charts populate as applications and hires are recorded."
                />
              )}

              <div>
                <h4 className="text-xs font-medium text-[var(--text-muted)]">Applications by job</h4>
                <div className="mt-2">
                  <BarChart
                    items={d.applications_per_job.map((x) => ({
                      label: (x as { job_title?: string; job_id?: string; name?: string }).job_title
                        ?? (x as { name?: string }).name
                        ?? (x as { job_id?: string }).job_id
                        ?? "Job",
                      value: x.count,
                    }))}
                    emptyMessage="No application volume yet"
                  />
                </div>
              </div>

              <div>
                <h4 className="text-xs font-medium text-[var(--text-muted)]">Status mix</h4>
                <div className="mt-2">
                  {d.by_status && Object.keys(d.by_status).length > 0 ? (
                    <DonutChart
                      items={Object.entries(d.by_status).map(([k, v], i) => ({
                        label: k.replace(/_/g, " "),
                        value: v,
                        color: i === 0 ? "var(--accent)" : undefined,
                      }))}
                    />
                  ) : d.ai_match_distribution && d.ai_match_distribution.length > 0 ? (
                    <PieChart items={d.ai_match_distribution.map((b) => ({ label: b.bucket, value: b.count }))} />
                  ) : (
                    <ChartEmpty message="No distribution data yet" />
                  )}
                </div>
              </div>
            </div>
          </Card>
        </section>

        <section className="grid gap-4 lg:grid-cols-2" aria-label="Recent activity">
          <Card
            padding="lg"
            header={
              <div className="flex items-center justify-between gap-3">
                <div>
                  <h3 className="ats-section-title">Recent candidates</h3>
                  <p className="mt-0.5 text-sm text-[var(--text-muted)]">Latest additions to the pipeline</p>
                </div>
                <Link href="/dashboard/candidates" className="ats-link text-sm">
                  View all
                </Link>
              </div>
            }
          >
            <div className="space-y-2">
              {recentCandidates.length > 0 ? (
                recentCandidates.slice(0, 5).map((candidate) => (
                  <Link key={candidate.id} href={`/dashboard/candidates/${candidate.id}`} className={ds.listRow}>
                    <div className="min-w-0">
                      <p className="truncate text-sm font-semibold text-[var(--text-primary)]">
                        {candidate.name}
                      </p>
                      <p className="truncate text-xs text-[var(--text-muted)]">
                        {candidate.current_designation ?? candidate.email}
                      </p>
                    </div>
                    <div className="shrink-0 text-right">
                      <Badge tone="accent">{candidate.status.replace(/_/g, " ")}</Badge>
                      <p className="mt-1 text-xs text-[var(--text-muted)]">{fmtDate(candidate.created_at)}</p>
                    </div>
                  </Link>
                ))
              ) : (
                <PanelEmpty
                  title="No candidates yet"
                  message="Add a candidate or upload a resume to start building your pipeline."
                />
              )}
            </div>
          </Card>

          <Card
            padding="lg"
            header={
              <div className="flex items-center justify-between gap-3">
                <div>
                  <h3 className="ats-section-title">Latest jobs</h3>
                  <p className="mt-0.5 text-sm text-[var(--text-muted)]">Recently created roles</p>
                </div>
                <Link href="/dashboard/jobs" className="ats-link text-sm">
                  View all
                </Link>
              </div>
            }
          >
            <div className="space-y-2">
              {recentJobs.length > 0 ? (
                recentJobs.slice(0, 5).map((job) => (
                  <Link key={job.id} href={`/dashboard/jobs/${job.id}`} className={ds.listRow}>
                    <div className="min-w-0">
                      <p className="truncate text-sm font-semibold text-[var(--text-primary)]">{job.title}</p>
                      <p className="truncate text-xs text-[var(--text-muted)]">
                        {job.location ?? "Remote"}
                      </p>
                    </div>
                    <div className="shrink-0 text-right">
                      <Badge tone={job.status === "open" ? "success" : "neutral"}>{job.status}</Badge>
                      <p className="mt-1 text-xs text-[var(--text-muted)]">{fmtDate(job.created_at)}</p>
                    </div>
                  </Link>
                ))
              ) : (
                <PanelEmpty
                  title="No jobs yet"
                  message="Create an open role to begin sourcing and AI matching."
                />
              )}
            </div>
          </Card>
        </section>
      </div>

      <ToastContainer toasts={toasts} onDismiss={dismiss} />
    </>
  );
}
