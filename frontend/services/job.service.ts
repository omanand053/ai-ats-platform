import { apiClient } from "@/lib/api-client";
import type { Job, JobListResult, JobPayload } from "@/lib/job-types";

export async function listJobs(params?: {
  page?: number;
  limit?: number;
  status?: string;
}): Promise<JobListResult> {
  const search = new URLSearchParams();
  if (params?.page) search.set("page", String(params.page));
  if (params?.limit) search.set("limit", String(params.limit));
  if (params?.status) search.set("status", params.status);

  const query = search.toString();
  return apiClient<JobListResult>(`/jobs${query ? `?${query}` : ""}`, { auth: true });
}

export async function getJob(id: string): Promise<Job> {
  const data = await apiClient<{ job: Job }>(`/jobs/${id}`, { auth: true });
  return data.job;
}

export async function createJob(payload: JobPayload): Promise<Job> {
  const data = await apiClient<{ job: Job }>("/jobs", {
    method: "POST",
    body: payload,
    auth: true,
  });
  return data.job;
}

export async function updateJob(id: string, payload: JobPayload): Promise<Job> {
  const data = await apiClient<{ job: Job }>(`/jobs/${id}`, {
    method: "PUT",
    body: payload,
    auth: true,
  });
  return data.job;
}

export async function deleteJob(id: string): Promise<void> {
  await apiClient<{ message: string }>(`/jobs/${id}`, {
    method: "DELETE",
    auth: true,
  });
}

export type SemanticMatchStatus =
  | "ok"
  | "job_embedding_missing"
  | "no_resume_embeddings"
  | "no_matches";

export interface AIMatchBreakdown {
  semantic: number;
  skills: number;
  experience: number;
  education: number;
  projects: number;
}

export interface FitScoreBreakdown {
  skills: number;
  experience: number;
  education: number;
  seniority: number;
  location: number;
  certifications: number;
}

export interface SemanticMatch {
  candidate_id: string;
  candidate_name: string;
  resume_id: string;
  /** Semantic similarity percent 0–100 (compatibility). */
  similarity_score: number;
  /** Legacy rule-based eligibility score (compatibility). Prefer eligibility_score. */
  overall_score?: number;
  ai_match_score: number;
  confidence: "high" | "medium" | "low" | string;
  strengths: string[];
  missing_skills: string[];
  matched_skills?: string[];
  why_shortlisted: string;
  why_not_shortlisted?: string;
  eligibility_score?: number;
  eligibility_breakdown?: FitScoreBreakdown;
  ai_match_breakdown?: AIMatchBreakdown;
  /** Soft warning only — never blocks ranking. */
  low_eligibility?: boolean;
}

export interface NotShortlistedCandidate {
  candidate_id: string;
  candidate_name: string;
  reason: string;
  eligibility_score?: number;
  why_not_shortlisted: string;
}

export interface SemanticMatchResult {
  status: SemanticMatchStatus;
  message: string;
  job_id: string;
  top_k: number;
  matches: SemanticMatch[];
  not_shortlisted?: NotShortlistedCandidate[];
  weights?: {
    semantic: number;
    skills: number;
    experience: number;
    education: number;
    projects: number;
  };
}

export async function getJobSemanticMatches(
  jobId: string,
  params?: { top_k?: number },
): Promise<SemanticMatchResult> {
  const search = new URLSearchParams();
  if (params?.top_k) search.set("top_k", String(params.top_k));
  const query = search.toString();
  return apiClient<SemanticMatchResult>(
    `/jobs/${jobId}/semantic-matches${query ? `?${query}` : ""}`,
    { auth: true },
  );
}

export interface AIAssistantReferencedCandidate {
  candidate_id: string;
  candidate_name: string;
  resume_id: string;
  resume_file_name?: string;
  resume_label: string;
  /** Recruiter-facing similarity percent in [0, 100]. */
  similarity_score: number;
  overall_score?: number;
}

export interface AIExplainability {
  reason: string;
  evidence: string[];
  confidence: string;
  relevant_resume_sections: string[];
}

export interface AIAssistantResponse {
  answer: string;
  ai_available: boolean;
  provider_used: string;
  message?: string;
  fallback_reason?: string;
  confidence_score?: number;
  referenced_candidates: AIAssistantReferencedCandidate[];
  semantic_matches_used: SemanticMatch[];
  provider?: string;
  model?: string;
  retrieval_status?: string;
  retrieval_message?: string;
  type?: string;
  cached?: boolean;
  structured?: Record<string, unknown>;
  explainability?: AIExplainability;
}

export type CopilotType =
  | "qa"
  | "summary"
  | "interview"
  | "recommendation"
  | "insights"
  | "jd_optimizer"
  | "email"
  | "compare";

export async function askJobAIAssistant(
  jobId: string,
  payload: { question: string; top_k?: number },
): Promise<AIAssistantResponse> {
  return apiClient<AIAssistantResponse>(`/jobs/${jobId}/ai-assistant`, {
    method: "POST",
    body: payload,
    auth: true,
  });
}

export async function runJobCopilot(
  jobId: string,
  payload: {
    type: CopilotType;
    question?: string;
    top_k?: number;
    candidate_id?: string;
    candidate_ids?: string[];
    email_kind?: string;
    difficulty?: string;
  },
): Promise<AIAssistantResponse> {
  return apiClient<AIAssistantResponse>(`/jobs/${jobId}/ai-assistant`, {
    method: "POST",
    body: payload,
    auth: true,
  });
}
