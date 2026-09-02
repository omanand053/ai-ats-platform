"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { EmptyState } from "@/components/ui/EmptyState";
import { ErrorState } from "@/components/ui/ErrorState";
import { PageHeader } from "@/components/ui/PageHeader";
import { DashboardSkeleton } from "@/components/ui/Skeleton";
import { ToastContainer } from "@/components/ui/Toast";
import { useRequireAuth } from "@/hooks/useRequireAuth";
import { useToast } from "@/hooks/useToast";
import { ApiClientError } from "@/lib/api-client";
import { createInterview, listInterviews, type Interview } from "@/services/enterprise.service";
import { listCandidatesByJob } from "@/services/candidate.service";
import { listJobs } from "@/services/job.service";

const TIMEZONES = [
  "UTC",
  "America/New_York",
  "America/Chicago",
  "America/Denver",
  "America/Los_Angeles",
  "Europe/London",
  "Europe/Paris",
  "Europe/Berlin",
  "Asia/Kolkata",
  "Asia/Singapore",
  "Asia/Tokyo",
  "Australia/Sydney",
];

function detectTimezone() {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC";
  } catch {
    return "UTC";
  }
}

export function CalendarView() {
  const ready = useRequireAuth();
  const { toasts, dismiss, show } = useToast();
  const [items, setItems] = useState<Interview[]>([]);
  const [jobs, setJobs] = useState<{ id: string; title: string }[]>([]);
  const [candidates, setCandidates] = useState<{ id: string; name: string }[]>([]);
  const [jobId, setJobId] = useState("");
  const [candidateId, setCandidateId] = useState("");
  const [when, setWhen] = useState("");
  const [tz, setTz] = useState(detectTimezone);
  const [loading, setLoading] = useState(true);
  const [loadingCandidates, setLoadingCandidates] = useState(false);
  const [saving, setSaving] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);

  const timezoneOptions = useMemo(() => {
    const base = detectTimezone();
    return Array.from(new Set([base, ...TIMEZONES]));
  }, []);

  const loadBase = useCallback(async () => {
    setLoading(true);
    setLoadError(null);
    try {
      const [iv, jobResult] = await Promise.all([
        listInterviews(),
        listJobs({ page: 1, limit: 100, status: "open" }),
      ]);
      setItems(iv);
      const mapped = jobResult.jobs.map((j) => ({ id: j.id, title: j.title }));
      // If no open jobs, fall back to all jobs so schedulers aren't blocked.
      if (!mapped.length) {
        const all = await listJobs({ page: 1, limit: 100 });
        setJobs(all.jobs.map((j) => ({ id: j.id, title: j.title })));
      } else {
        setJobs(mapped);
      }
    } catch (err) {
      setLoadError(err instanceof ApiClientError ? err.message : "Failed to load calendar");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (!ready) return;
    void loadBase();
  }, [ready, loadBase]);

  useEffect(() => {
    if (!jobId) {
      setCandidates([]);
      setCandidateId("");
      return;
    }
    let cancelled = false;
    setLoadingCandidates(true);
    setCandidateId("");
    listCandidatesByJob(jobId, { page: 1, limit: 200, sort: "score" })
      .then((res) => {
        if (cancelled) return;
        const list = res.candidates.map((c) => ({ id: c.id, name: c.name }));
        setCandidates(list);
        if (list[0]) setCandidateId(list[0].id);
      })
      .catch((err) => {
        if (!cancelled) {
          setCandidates([]);
          show(err instanceof ApiClientError ? err.message : "Failed to load candidates", "error");
        }
      })
      .finally(() => {
        if (!cancelled) setLoadingCandidates(false);
      });
    return () => {
      cancelled = true;
    };
  }, [jobId, show]);

  const byDay = useMemo(() => {
    const map = new Map<string, Interview[]>();
    for (const iv of items) {
      const key = new Date(iv.scheduled_at).toLocaleDateString(undefined, {
        weekday: "short",
        month: "short",
        day: "numeric",
        year: "numeric",
      });
      const arr = map.get(key) ?? [];
      arr.push(iv);
      map.set(key, arr);
    }
    return [...map.entries()];
  }, [items]);

  async function schedule() {
    if (!jobId || !candidateId || !when) {
      show("Select a job, candidate, and date before scheduling", "error");
      return;
    }
    setSaving(true);
    try {
      const scheduled_at = new Date(when).toISOString();
      await createInterview({
        candidate_id: candidateId,
        job_id: jobId,
        title: "Interview",
        scheduled_at,
        duration_minutes: 45,
        timezone: tz,
      });
      show("Interview scheduled — team notified");
      setWhen("");
      const list = await listInterviews();
      setItems(list);
    } catch (err) {
      show(err instanceof ApiClientError ? err.message : "Failed to schedule interview", "error");
    } finally {
      setSaving(false);
    }
  }

  if (!ready || loading) {
    return (
      <>
        <DashboardSkeleton />
      </>
    );
  }

  if (loadError) {
    return (
      <>
        <>
          <ErrorState
            title="Calendar unavailable"
            description={loadError}
            onRetry={() => void loadBase()}
          />
        </>
        <ToastContainer toasts={toasts} onDismiss={dismiss} />
      </>
    );
  }

  const canSchedule = Boolean(jobId && candidateId && when && !loadingCandidates);

  return (
    <>
      <>
        <PageHeader
          title="Interview Calendar"
          subtitle="Schedule against a job pipeline — only that job’s candidates appear."
        />

        <div className="mt-4 grid gap-4 lg:grid-cols-[minmax(18rem,24rem)_1fr]">
          <Card padding="lg" className="h-fit space-y-3">
            <div>
              <p className="text-xs font-semibold uppercase tracking-[0.08em] text-[#6b7280]">Schedule flow</p>
                <h3 className="mt-1 text-base font-semibold text-[#0b1220]">New interview</h3>
            </div>

            <ol className="space-y-2.5" aria-label="Interview scheduling steps">
              <li>
                <label className="block text-sm font-medium text-[#0b1220]" htmlFor="schedule-job">
                  1. Select job
                </label>
                <select
                  id="schedule-job"
                  className="ats-input mt-1.5"
                  value={jobId}
                  onChange={(e) => setJobId(e.target.value)}
                >
                  <option value="">Choose a job…</option>
                  {jobs.map((j) => (
                    <option key={j.id} value={j.id}>
                      {j.title}
                    </option>
                  ))}
                </select>
                {!jobs.length ? (
                  <p className="mt-1.5 text-xs text-amber-700">Create a job before scheduling.</p>
                ) : null}
              </li>

              <li>
                <label className="block text-sm font-medium text-[#0b1220]" htmlFor="schedule-candidate">
                  2. Select candidate
                </label>
                <select
                  id="schedule-candidate"
                  className="ats-input mt-1.5"
                  value={candidateId}
                  disabled={!jobId || loadingCandidates}
                  onChange={(e) => setCandidateId(e.target.value)}
                >
                  {!jobId ? (
                    <option value="">Select a job first</option>
                  ) : loadingCandidates ? (
                    <option value="">Loading candidates…</option>
                  ) : candidates.length === 0 ? (
                    <option value="">No candidates on this job</option>
                  ) : (
                    candidates.map((c) => (
                      <option key={c.id} value={c.id}>
                        {c.name}
                      </option>
                    ))
                  )}
                </select>
                {jobId && !loadingCandidates && candidates.length === 0 ? (
                  <p className="mt-1.5 text-xs text-slate-500">
                    Add applicants to this job to schedule interviews.
                  </p>
                ) : null}
              </li>

              <li>
                <label className="block text-sm font-medium text-[#0b1220]" htmlFor="schedule-when">
                  3. Date & time
                </label>
                <input
                  id="schedule-when"
                  type="datetime-local"
                  className="ats-input mt-1.5"
                  value={when}
                  disabled={!candidateId}
                  onChange={(e) => setWhen(e.target.value)}
                />
              </li>

              <li>
                <label className="block text-sm font-medium text-[#0b1220]" htmlFor="schedule-tz">
                  4. Timezone
                </label>
                <select
                  id="schedule-tz"
                  className="ats-input mt-1.5"
                  value={tz}
                  disabled={!candidateId}
                  onChange={(e) => setTz(e.target.value)}
                >
                  {timezoneOptions.map((z) => (
                    <option key={z} value={z}>
                      {z}
                    </option>
                  ))}
                </select>
              </li>
            </ol>

            <Button
              className="w-full"
              loading={saving}
              disabled={!canSchedule}
              onClick={() => void schedule()}
            >
              Schedule interview
            </Button>
          </Card>

          <Card padding="lg">
            <div className="flex items-end justify-between gap-3">
              <div>
                <h3 className="text-base font-semibold text-[#0b1220]">Upcoming</h3>
                <p className="mt-0.5 text-sm text-[#6b7280]">
                  {items.length} scheduled interview{items.length === 1 ? "" : "s"}
                </p>
              </div>
            </div>

            <div className="mt-4 space-y-4">
              {byDay.length === 0 ? (
                <EmptyState
                  title="No interviews yet"
                  description="Pick a job, choose a candidate from that pipeline, then schedule a time."
                />
              ) : (
                byDay.map(([day, list]) => (
                  <section key={day}>
                    <h4 className="text-xs font-semibold uppercase tracking-[0.08em] text-slate-400">
                      {day}
                    </h4>
                    <ul className="mt-2 space-y-2">
                      {list.map((iv) => (
                        <li key={iv.id} className="rounded-xl border border-gray-100 bg-white px-3.5 py-3 transition hover:shadow-sm">
                          <p className="text-sm font-semibold text-[#0b1220]">{iv.candidate_name || iv.title}</p>
                          <p className="mt-1 text-xs text-[#6b7280]">{new Date(iv.scheduled_at).toLocaleString()} · {iv.duration_minutes}m · {iv.timezone} · {iv.status}</p>
                        </li>
                      ))}
                    </ul>
                  </section>
                ))
              )}
            </div>
          </Card>
        </div>
      </>
      <ToastContainer toasts={toasts} onDismiss={dismiss} />
    </>
  );
}
