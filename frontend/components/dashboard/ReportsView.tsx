"use client";

import { useEffect, useState } from "react";
import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { ErrorState } from "@/components/ui/ErrorState";
import { PageHeader } from "@/components/ui/PageHeader";
import { DashboardSkeleton } from "@/components/ui/Skeleton";
import { ToastContainer } from "@/components/ui/Toast";
import { useRequireAuth } from "@/hooks/useRequireAuth";
import { useToast } from "@/hooks/useToast";
import { ApiClientError } from "@/lib/api-client";
import { getAnalyticsOverview, listAuditLogs, type AnalyticsOverview } from "@/services/enterprise.service";
import { BarChart, DonutChart, PieChart, TrendSparkline, ChartEmpty } from "@/components/dashboard/charts";
import { listCandidates } from "@/services/candidate.service";
import { listJobs } from "@/services/job.service";

function downloadText(filename: string, content: string, mime: string) {
  const blob = new Blob([content], { type: mime });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}

function toCsv(rows: string[][]) {
  return rows.map((r) => r.map((c) => `"${String(c).replace(/"/g, '""')}"`).join(",")).join("\n");
}

export function ReportsView() {
  const ready = useRequireAuth();
  const { toasts, dismiss, show } = useToast();
  const [overview, setOverview] = useState<AnalyticsOverview | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);

  function load() {
    setLoading(true);
    setLoadError(null);
    getAnalyticsOverview()
      .then(setOverview)
      .catch((err) => {
        const message = err instanceof ApiClientError ? err.message : "Failed to load reports";
        setLoadError(message);
        show(message, "error");
      })
      .finally(() => setLoading(false));
  }

  useEffect(() => {
    if (!ready) return;
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps -- load on auth ready only
  }, [ready, show]);

  async function exportHiringCsv() {
    if (!overview) return;
    const rows = [
      ["metric", "value"],
      ["total_jobs", String(overview.total_jobs)],
      ["open_jobs", String(overview.open_jobs)],
      ["applicants", String(overview.applicants)],
      ["interviews", String(overview.interviews)],
      ["offers", String(overview.offers)],
      ["selected", String(overview.selected)],
      ["rejected", String(overview.rejected)],
      ["avg_ai_match", String(overview.avg_ai_match ?? "")],
      ["avg_time_to_hire_days", String(overview.avg_time_to_hire_days ?? "")],
    ];
    downloadText(`hiring-report-${Date.now()}.csv`, toCsv(rows), "text/csv;charset=utf-8");
    show("Hiring CSV exported");
  }

  async function exportCandidatesCsv() {
    const result = await listCandidates({ page: 1, limit: 500, sort: "score" });
    const rows = [
      ["name", "email", "status", "experience_years", "overall_score", "job_id"],
      ...result.candidates.map((c) => [
        c.name,
        c.email,
        c.status,
        String(c.experience_years ?? ""),
        String(c.overall_score ?? ""),
        c.job_id ?? "",
      ]),
    ];
    downloadText(`candidate-report-${Date.now()}.csv`, toCsv(rows), "text/csv;charset=utf-8");
    // Excel-compatible UTF-8 CSV
    downloadText(`candidate-report-${Date.now()}.xlsx.csv`, toCsv(rows), "application/vnd.ms-excel");
    show("Candidate report exported (CSV + Excel-compatible)");
  }

  async function exportJobsCsv() {
    const result = await listJobs({ page: 1, limit: 200 });
    const rows = [
      ["title", "status", "location", "employment_type", "created_at"],
      ...result.jobs.map((j) => [
        j.title,
        j.status,
        j.location ?? "",
        j.employment_type,
        j.created_at,
      ]),
    ];
    downloadText(`job-report-${Date.now()}.csv`, toCsv(rows), "text/csv;charset=utf-8");
    show("Job report exported");
  }

  async function exportInterviewCsv() {
    if (!overview) return;
    const rows = [
      ["stage", "count"],
      ...overview.funnel.map((f) => [f.name, String(f.count)]),
    ];
    downloadText(`interview-report-${Date.now()}.csv`, toCsv(rows), "text/csv;charset=utf-8");
    show("Interview funnel exported");
  }

  function exportPdf() {
    if (!overview) return;
    const w = window.open("", "_blank", "noopener,noreferrer");
    if (!w) {
      show("Pop-up blocked — allow pop-ups for PDF print", "error");
      return;
    }
    w.document.write(`<!doctype html><html><head><title>Hiring Report</title>
      <style>body{font-family:Segoe UI,sans-serif;padding:32px;color:#0f172a}
      h1{font-size:22px}table{border-collapse:collapse;width:100%;margin-top:16px}
      td,th{border:1px solid #e2e8f0;padding:8px;text-align:left;font-size:13px}</style></head><body>
      <h1>Hiring Report</h1>
      <p>Generated ${new Date().toLocaleString()}</p>
      <table><tr><th>Metric</th><th>Value</th></tr>
      <tr><td>Applicants</td><td>${overview.applicants}</td></tr>
      <tr><td>Open jobs</td><td>${overview.open_jobs}</td></tr>
      <tr><td>Interviews</td><td>${overview.interviews}</td></tr>
      <tr><td>Offers</td><td>${overview.offers}</td></tr>
      <tr><td>Selected</td><td>${overview.selected}</td></tr>
      <tr><td>Rejected</td><td>${overview.rejected}</td></tr>
      </table>
      <script>window.onload=()=>window.print()</script>
      </body></html>`);
    w.document.close();
  }

  async function exportAudit() {
    try {
      const logs = await listAuditLogs({ limit: 200 });
      const rows = [
        ["created_at", "action", "resource_type", "resource_id", "actor_user_id"],
        ...logs.map((l) => [
          l.created_at,
          l.action,
          l.resource_type,
          l.resource_id ?? "",
          l.actor_user_id ?? "",
        ]),
      ];
      downloadText(`audit-logs-${Date.now()}.csv`, toCsv(rows), "text/csv;charset=utf-8");
      show("Audit logs exported");
    } catch (err) {
      show(err instanceof ApiClientError ? err.message : "Audit export failed (admin/HM only)", "error");
    }
  }

  if (!ready || loading) {
    return (
      <>
        <DashboardSkeleton />
      </>
    );
  }

  if (loadError && !overview) {
    return (
      <>
        <>
          <ErrorState title="Reports unavailable" description={loadError} onRetry={load} />
        </>
        <ToastContainer toasts={toasts} onDismiss={dismiss} />
      </>
    );
  }

  return (
    <>
      <>
        <PageHeader
          title="Recruiter Reports"
          subtitle="Export CSV, Excel-compatible CSV, and print-ready PDF hiring reports."
        />
        <div className="mt-4 grid gap-3 md:grid-cols-2">
          <Card padding="lg" className="space-y-3 md:col-span-2">
            <h3 className="font-semibold text-[var(--text-primary)]">Hiring Overview</h3>
            <p className="text-sm text-[var(--text-muted)]">Visual summary of recent hiring activity.</p>
            <div className="mt-3 grid gap-3 lg:grid-cols-3">
              <div className="col-span-1">
                {overview?.hiring_trend && overview.hiring_trend.length > 0 ? (
                  <TrendSparkline points={overview.hiring_trend.map((p) => ({ period: p.period, count: p.count }))} />
                ) : (
                  <ChartEmpty />
                )}
              </div>

              <div className="col-span-1">
                {overview && overview.by_status && Object.keys(overview.by_status).length > 0 ? (
                  <DonutChart items={Object.entries(overview.by_status).map(([k, v], i) => ({ label: k.replace(/_/g, " "), value: v, color: i === 0 ? 'var(--brand)' : undefined }))} />
                ) : (
                  <ChartEmpty />
                )}
              </div>

              <div className="col-span-1">
                {overview && overview.ai_match_distribution && overview.ai_match_distribution.length > 0 ? (
                  <PieChart items={overview.ai_match_distribution.map((b) => ({ label: b.bucket, value: b.count }))} />
                ) : (
                  <ChartEmpty />
                )}
              </div>
            </div>
          </Card>
          <Card padding="lg" className="space-y-3">
            <h3 className="font-semibold text-[#0b1220]">Candidate Report</h3>
            <p className="text-sm text-[#6b7280]">Up to 500 candidates with scores and stages.</p>
            <Button className="w-auto px-4" onClick={() => void exportCandidatesCsv()}>
              Export CSV / Excel
            </Button>
          </Card>
          <Card padding="lg" className="space-y-3">
            <h3 className="font-semibold text-[#0b1220]">Job Report</h3>
            <Button className="w-auto px-4" onClick={() => void exportJobsCsv()}>
              Export CSV
            </Button>
          </Card>
          <Card padding="lg" className="space-y-3">
            <h3 className="font-semibold text-[#0b1220]">Hiring Report</h3>
            <div className="flex flex-wrap gap-2">
              <Button className="w-auto px-4" onClick={() => void exportHiringCsv()}>
                Export CSV
              </Button>
              <Button variant="secondary" className="w-auto px-4" onClick={exportPdf}>
                Export PDF
              </Button>
            </div>
          </Card>
          <Card padding="lg" className="space-y-3">
            <h3 className="font-semibold text-[#0b1220]">Interview Report</h3>
            <Button className="w-auto px-4" onClick={() => void exportInterviewCsv()}>
              Export funnel CSV
            </Button>
          </Card>
          <Card padding="lg" className="space-y-3 md:col-span-2">
            <h3 className="font-semibold text-[#0b1220]">Audit Logs</h3>
            <p className="text-sm text-[#6b7280]">Admin / hiring manager only.</p>
            <Button variant="secondary" className="w-auto px-4" onClick={() => void exportAudit()}>
              Export audit CSV
            </Button>
          </Card>
        </div>
      </>
      <ToastContainer toasts={toasts} onDismiss={dismiss} />
    </>
  );
}
