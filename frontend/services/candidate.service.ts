import { apiClient } from "@/lib/api-client";
import type { Candidate, CandidateListResult, CandidatePayload } from "@/lib/candidate-types";

export async function listCandidates(params?: {
  page?: number;
  limit?: number;
  status?: string;
  search?: string;
  job_id?: string;
  sort?: string;
}): Promise<CandidateListResult> {
  const searchParams = new URLSearchParams();
  if (params?.page) searchParams.set("page", String(params.page));
  if (params?.limit) searchParams.set("limit", String(params.limit));
  if (params?.status) searchParams.set("status", params.status);
  if (params?.search) searchParams.set("search", params.search);
  if (params?.job_id) searchParams.set("job_id", params.job_id);
  if (params?.sort) searchParams.set("sort", params.sort);

  const query = searchParams.toString();
  return apiClient<CandidateListResult>(`/candidates${query ? `?${query}` : ""}`, { auth: true });
}

export async function getCandidate(id: string): Promise<Candidate> {
  const data = await apiClient<{ candidate: Candidate }>(`/candidates/${id}`, { auth: true });
  return data.candidate;
}

export async function createCandidate(payload: CandidatePayload): Promise<Candidate> {
  const data = await apiClient<{ candidate: Candidate }>("/candidates", {
    method: "POST",
    body: payload,
    auth: true,
  });
  return data.candidate;
}

export async function updateCandidate(id: string, payload: CandidatePayload): Promise<Candidate> {
  const data = await apiClient<{ candidate: Candidate }>(`/candidates/${id}`, {
    method: "PUT",
    body: payload,
    auth: true,
  });
  return data.candidate;
}

export async function deleteCandidate(id: string): Promise<void> {
  await apiClient<{ message: string }>(`/candidates/${id}`, {
    method: "DELETE",
    auth: true,
  });
}

export async function listCandidatesByJob(
  jobId: string,
  params?: { page?: number; limit?: number; status?: string; search?: string; sort?: string },
): Promise<CandidateListResult> {
  const searchParams = new URLSearchParams();
  if (params?.page) searchParams.set("page", String(params.page));
  if (params?.limit) searchParams.set("limit", String(params.limit));
  if (params?.status) searchParams.set("status", params.status);
  if (params?.search) searchParams.set("search", params.search);
  if (params?.sort) searchParams.set("sort", params.sort);

  const query = searchParams.toString();
  return apiClient<CandidateListResult>(
    `/jobs/${jobId}/candidates${query ? `?${query}` : ""}`,
    { auth: true },
  );
}

export interface CandidateNote {
  id: string;
  company_id: string;
  candidate_id: string;
  author_user_id?: string;
  body: string;
  created_at: string;
  updated_at: string;
}

export interface CandidateTimelineItem {
  id: string;
  event_type: string;
  label: string;
  timestamp: string;
  source: "recorded" | "inferred" | string;
}

export async function listCandidateNotes(candidateId: string): Promise<CandidateNote[]> {
  const data = await apiClient<{ notes: CandidateNote[] }>(`/candidates/${candidateId}/notes`, {
    auth: true,
  });
  return data.notes ?? [];
}

export async function createCandidateNote(candidateId: string, body: string): Promise<CandidateNote> {
  const data = await apiClient<{ note: CandidateNote }>(`/candidates/${candidateId}/notes`, {
    method: "POST",
    body: { body },
    auth: true,
  });
  return data.note;
}

export async function deleteCandidateNote(candidateId: string, noteId: string): Promise<void> {
  await apiClient<{ message: string }>(`/candidates/${candidateId}/notes/${noteId}`, {
    method: "DELETE",
    auth: true,
  });
}

export async function getCandidateTimeline(candidateId: string): Promise<CandidateTimelineItem[]> {
  const data = await apiClient<{ timeline: CandidateTimelineItem[] }>(
    `/candidates/${candidateId}/timeline`,
    { auth: true },
  );
  return data.timeline ?? [];
}
