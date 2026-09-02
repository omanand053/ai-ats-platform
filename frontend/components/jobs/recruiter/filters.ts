import type { Candidate, CandidateStatus } from "@/lib/candidate-types";
import type { SemanticMatch } from "@/services/job.service";

export type WorkspaceFilters = {
  search: string;
  stage: CandidateStatus | "all";
  minAiMatch: number;
  minExperience: number;
  skill: string;
  location: string;
  confidence: "all" | "high" | "medium" | "low";
  resumeUploaded: "all" | "yes" | "no";
};

export const defaultFilters = (): WorkspaceFilters => ({
  search: "",
  stage: "all",
  minAiMatch: 0,
  minExperience: 0,
  skill: "",
  location: "",
  confidence: "all",
  resumeUploaded: "all",
});

export function filtersStorageKey(jobId: string) {
  return `ats-recruiter-filters-v2:${jobId}`;
}

export function loadSavedFilters(jobId: string): WorkspaceFilters {
  if (typeof window === "undefined") return defaultFilters();
  try {
    const raw = localStorage.getItem(filtersStorageKey(jobId));
    if (!raw) return defaultFilters();
    const parsed = JSON.parse(raw) as Partial<WorkspaceFilters>;
    const base = defaultFilters();
    return {
      ...base,
      ...parsed,
      // Drop legacy education field if present in old saves.
      confidence: parsed.confidence ?? "all",
      resumeUploaded: parsed.resumeUploaded ?? "all",
    };
  } catch {
    return defaultFilters();
  }
}

export function saveFilters(jobId: string, filters: WorkspaceFilters) {
  if (typeof window === "undefined") return;
  localStorage.setItem(filtersStorageKey(jobId), JSON.stringify(filters));
}

export type RankedApplicant = {
  candidate: Candidate;
  match?: SemanticMatch;
};

export function applyFilters(rows: RankedApplicant[], filters: WorkspaceFilters): RankedApplicant[] {
  const q = filters.search.trim().toLowerCase();
  const skill = filters.skill.trim().toLowerCase();
  const loc = filters.location.trim().toLowerCase();

  return rows.filter(({ candidate, match }) => {
    if (filters.stage !== "all" && candidate.status !== filters.stage) return false;
    const ai = match?.ai_match_score ?? -1;
    if (ai < filters.minAiMatch) return false;
    if ((candidate.experience_years ?? 0) < filters.minExperience) return false;
    if (skill) {
      const hay = [...(candidate.skills ?? []), ...(match?.matched_skills ?? [])]
        .join(" ")
        .toLowerCase();
      if (!hay.includes(skill)) return false;
    }
    if (loc && !(candidate.location || "").toLowerCase().includes(loc)) return false;
    if (filters.confidence !== "all") {
      const conf = (match?.confidence || "").toLowerCase();
      if (conf !== filters.confidence) return false;
    }
    if (filters.resumeUploaded === "yes") {
      if (!(candidate.resume_url || candidate.resume_text)) return false;
    }
    if (filters.resumeUploaded === "no") {
      if (candidate.resume_url || candidate.resume_text) return false;
    }
    if (q) {
      const blob = [
        candidate.name,
        candidate.email,
        candidate.current_designation,
        candidate.current_company,
        candidate.location,
        ...(candidate.skills ?? []),
      ]
        .filter(Boolean)
        .join(" ")
        .toLowerCase();
      if (!blob.includes(q)) return false;
    }
    return true;
  });
}

export function formatPct(value?: number | null) {
  if (typeof value !== "number" || Number.isNaN(value)) return "—";
  if (value > 0 && value < 1) return `${value.toFixed(1)}%`;
  return `${Math.round(value)}%`;
}

export function confidenceTone(confidence?: string) {
  switch ((confidence || "").toLowerCase()) {
    case "high":
      return "bg-emerald-50 text-emerald-800 ring-emerald-100";
    case "medium":
      return "bg-amber-50 text-amber-900 ring-amber-100";
    default:
      return "bg-slate-50 text-slate-700 ring-slate-200";
  }
}

export function resumeStatusLabel(candidate: Candidate) {
  if (candidate.parsing_status === "completed") return "Parsed";
  if (candidate.parsing_status === "processing") return "Parsing…";
  if (candidate.parsing_status === "failed") return "Parse failed";
  if (candidate.resume_url || candidate.resume_text) return "Attached";
  return "No resume";
}

export function exportCandidatesCsv(rows: RankedApplicant[]) {
  const header = [
    "name",
    "email",
    "designation",
    "experience_years",
    "status",
    "ai_match",
    "semantic",
    "confidence",
    "location",
  ];
  const lines = [header.join(",")];
  for (const { candidate, match } of rows) {
    const vals = [
      candidate.name,
      candidate.email,
      candidate.current_designation || "",
      candidate.experience_years ?? "",
      candidate.status,
      match?.ai_match_score ?? "",
      match?.similarity_score ?? "",
      match?.confidence || "",
      candidate.location || "",
    ].map((v) => `"${String(v).replace(/"/g, '""')}"`);
    lines.push(vals.join(","));
  }
  const blob = new Blob([lines.join("\n")], { type: "text/csv;charset=utf-8" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = `applicants-${Date.now()}.csv`;
  a.click();
  URL.revokeObjectURL(url);
}
