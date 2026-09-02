"use client";

import { useEffect, useMemo, useState } from "react";
import { formatPct, type RankedApplicant } from "@/components/jobs/recruiter/filters";
import { Button } from "@/components/ui/Button";
import { Skeleton } from "@/components/ui/Skeleton";
import { ApiClientError } from "@/lib/api-client";
import {
  runJobCopilot,
  type AIAssistantResponse,
  type CopilotType,
} from "@/services/job.service";

const TABS: { id: CopilotTab; label: string }[] = [
  { id: "summary", label: "Summary" },
  { id: "compare", label: "Comparison" },
  { id: "interview", label: "Interview" },
  { id: "emails", label: "Email" },
];

type CopilotTab = "summary" | "compare" | "interview" | "emails";

export function AICopilotPanel({
  jobId,
  candidateId,
  comparePool,
  onToast,
}: {
  jobId: string;
  candidateId: string;
  comparePool: RankedApplicant[];
  onToast: (message: string, tone?: "success" | "error") => void;
}) {
  const [tab, setTab] = useState<CopilotTab>("summary");
  const [loading, setLoading] = useState(false);
  const [results, setResults] = useState<Record<CopilotTab, AIAssistantResponse | null>>({
    summary: null,
    compare: null,
    interview: null,
    emails: null,
  });
  const [difficulty, setDifficulty] = useState<"easy" | "medium" | "hard">("medium");
  const [emailKind, setEmailKind] = useState("interview_invite");
  const [selectedCompare, setSelectedCompare] = useState<string[]>([]);

  const cache = useMemo(() => new Map<string, AIAssistantResponse>(), [candidateId]);

  useEffect(() => {
    setResults({ summary: null, compare: null, interview: null, emails: null });
    setSelectedCompare([candidateId]);
  }, [candidateId]);

  async function run(type: CopilotType, extra?: Record<string, unknown>) {
    const key = `${type}:${JSON.stringify(extra ?? {})}`;
    const cached = cache.get(key);
    if (cached) {
      setResults((prev) => ({ ...prev, [type]: cached }));
      return;
    }
    setLoading(true);
    try {
      const resp = await runJobCopilot(jobId, {
        type,
        candidate_id: candidateId,
        ...(extra as object),
      });
      cache.set(key, resp);
      setResults((prev) => ({ ...prev, [type]: resp }));
      if (!resp.ai_available) {
        onToast(resp.message || "AI unavailable — showing deterministic output", "error");
      }
    } catch (err) {
      onToast(err instanceof ApiClientError ? err.message : "Copilot request failed", "error");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    if (tab === "summary") void run("summary");
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tab, candidateId]);

  function toggleCompare(id: string) {
    setSelectedCompare((prev) => {
      if (prev.includes(id)) return prev.filter((x) => x !== id);
      if (prev.length >= 5) return prev;
      return [...prev, id];
    });
  }

  return (
    <section className="rounded-2xl border border-slate-200 bg-gradient-to-b from-slate-50 to-white p-3 sm:p-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div>
          <h3 className="text-sm font-semibold text-slate-900">AI Assistant</h3>
          <p className="text-xs text-slate-500">Lazy-loaded panels · cached 10m on server</p>
        </div>
      </div>

      <div className="mt-3 flex flex-wrap gap-1.5" role="tablist" aria-label="Copilot tabs">
        {TABS.map((t) => (
          <button
            key={t.id}
            type="button"
            role="tab"
            aria-selected={tab === t.id}
            className={`rounded-lg px-2.5 py-1.5 text-xs font-semibold transition ${
              tab === t.id
                ? "bg-blue-600 text-white shadow-sm"
                : "bg-white text-slate-600 ring-1 ring-slate-200 hover:bg-slate-50"
            }`}
            onClick={() => setTab(t.id)}
          >
            {t.label}
          </button>
        ))}
      </div>

      <div className="mt-4 min-h-[160px]">
        {tab === "interview" && (
          <div className="mb-3 flex flex-wrap items-center gap-2">
            <label className="text-xs font-medium text-slate-600">
              Difficulty
              <select
                className="ats-input ml-2 inline-block w-auto"
                value={difficulty}
                onChange={(e) => setDifficulty(e.target.value as typeof difficulty)}
              >
                <option value="easy">Easy</option>
                <option value="medium">Medium</option>
                <option value="hard">Hard</option>
              </select>
            </label>
            <Button
              size="sm"
              className="w-auto px-3"
              loading={loading}
              onClick={() => void run("interview", { difficulty })}
            >
              Generate questions
            </Button>
          </div>
        )}

        {tab === "emails" && (
          <div className="mb-3 flex flex-wrap items-center gap-2">
            <select
              className="ats-input w-auto"
              value={emailKind}
              onChange={(e) => setEmailKind(e.target.value)}
            >
              <option value="interview_invite">Interview Invite</option>
              <option value="offer">Offer</option>
              <option value="rejection">Rejection</option>
              <option value="follow_up">Follow-up</option>
            </select>
            <Button
              size="sm"
              className="w-auto px-3"
              loading={loading}
              onClick={() => void run("email", { email_kind: emailKind })}
            >
              Generate email
            </Button>
          </div>
        )}

        {tab === "compare" && (
          <div className="mb-3 space-y-2">
            <p className="text-xs text-slate-500">Select up to 5 applicants to compare</p>
            <div className="flex max-h-28 flex-wrap gap-1.5 overflow-auto">
              {comparePool.slice(0, 30).map((row) => (
                <label
                  key={row.candidate.id}
                  className={`inline-flex cursor-pointer items-center gap-1 rounded-lg px-2 py-1 text-xs ring-1 ${
                    selectedCompare.includes(row.candidate.id)
                      ? "bg-blue-50 text-blue-800 ring-blue-200"
                      : "bg-white text-slate-600 ring-slate-200"
                  }`}
                >
                  <input
                    type="checkbox"
                    className="sr-only"
                    checked={selectedCompare.includes(row.candidate.id)}
                    onChange={() => toggleCompare(row.candidate.id)}
                  />
                  {row.candidate.name}
                </label>
              ))}
            </div>
            <Button
              size="sm"
              className="w-auto px-3"
              loading={loading}
              disabled={selectedCompare.length < 2}
              onClick={() => void run("compare", { candidate_ids: selectedCompare })}
            >
              Compare
            </Button>
          </div>
        )}

        {loading ? (
          <div className="space-y-2">
            <Skeleton className="h-4 w-2/3" />
            <Skeleton className="h-24 w-full rounded-xl" />
          </div>
        ) : results[tab] ? (
          <CopilotResultView result={results[tab] as AIAssistantResponse} />
        ) : (
          <p className="text-sm text-slate-500">
            {tab === "interview" || tab === "emails" || tab === "compare"
              ? "Choose options and generate."
              : "Loading copilot…"}
          </p>
        )}
      </div>
    </section>
  );
}

function CopilotResultView({ result }: { result: AIAssistantResponse }) {
  const compareRows = Array.isArray(result.structured?.candidates)
    ? (result.structured?.candidates as Array<Record<string, unknown>>)
    : null;

  return (
    <div className="space-y-3">
      {result.cached && (
        <p className="text-[11px] font-medium uppercase tracking-wide text-emerald-700">Cached</p>
      )}
      {!result.ai_available && (
        <div className="rounded-xl bg-amber-50 px-3 py-2 text-xs text-amber-900 ring-1 ring-amber-100">
          {result.message || "AI unavailable"} {result.fallback_reason ? `· ${result.fallback_reason}` : ""}
        </div>
      )}

      {compareRows ? (
        <div className="overflow-auto rounded-xl ring-1 ring-slate-200">
          <table className="min-w-full text-left text-xs">
            <thead className="bg-slate-50 text-slate-500">
              <tr>
                <th className="px-3 py-2">Candidate</th>
                <th className="px-3 py-2">AI Match</th>
                <th className="px-3 py-2">Semantic</th>
                <th className="px-3 py-2">Skills</th>
                <th className="px-3 py-2">Exp</th>
                <th className="px-3 py-2">Rec</th>
              </tr>
            </thead>
            <tbody>
              {compareRows.map((row) => (
                <tr key={String(row.candidate_id)} className="border-t border-slate-100 bg-white">
                  <td className="px-3 py-2 font-medium text-slate-900">
                    {String(row.candidate_name)}
                    {result.structured?.winner_id === row.candidate_id ? (
                      <span className="ml-1 text-emerald-700">· Winner</span>
                    ) : null}
                  </td>
                  <td className="px-3 py-2 tabular-nums">{formatPct(Number(row.ai_match))}</td>
                  <td className="px-3 py-2 tabular-nums">{formatPct(Number(row.semantic))}</td>
                  <td className="px-3 py-2 tabular-nums">{formatPct(Number(row.skills))}</td>
                  <td className="px-3 py-2 tabular-nums">{formatPct(Number(row.experience))}</td>
                  <td className="px-3 py-2">{String(row.recommendation)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <pre className="max-h-80 overflow-auto whitespace-pre-wrap rounded-xl bg-white p-3 text-sm leading-6 text-slate-800 ring-1 ring-slate-100">
          {result.answer || "No content"}
        </pre>
      )}

      {result.explainability && (
        <div className="rounded-xl bg-slate-50 px-3 py-2 text-xs text-slate-600 ring-1 ring-slate-100">
          <p className="font-semibold text-slate-800">Explainability</p>
          <p className="mt-1">{result.explainability.reason}</p>
          <p className="mt-1">Confidence: {result.explainability.confidence}</p>
          {result.explainability.evidence?.length ? (
            <ul className="mt-1 list-disc pl-4">
              {result.explainability.evidence.map((e) => (
                <li key={e}>{e}</li>
              ))}
            </ul>
          ) : null}
        </div>
      )}
    </div>
  );
}
