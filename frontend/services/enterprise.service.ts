import { apiClient } from "@/lib/api-client";

export interface NamedCount {
  name: string;
  count: number;
}

export interface BucketCount {
  bucket: string;
  count: number;
}

export interface TrendPoint {
  period: string;
  count: number;
}

export interface AnalyticsOverview {
  total_jobs: number;
  open_jobs: number;
  closed_jobs: number;
  applicants: number;
  ai_shortlisted: number;
  recruiter_shortlisted: number;
  interviews: number;
  offers: number;
  selected: number;
  rejected: number;
  hired: number;
  avg_ai_match?: number;
  avg_time_to_hire_days?: number;
  offer_acceptance_rate?: number;
  by_status: Record<string, number>;
  applications_per_job: NamedCount[];
  top_skills: NamedCount[];
  missing_skills: NamedCount[];
  ai_match_distribution: BucketCount[];
  hiring_trend: TrendPoint[];
  monthly_hiring: TrendPoint[];
  recruiter_productivity: NamedCount[];
  funnel: NamedCount[];
}

export interface CompanyAISettings {
  company_id: string;
  weight_semantic: number;
  weight_skills: number;
  weight_experience: number;
  weight_education: number;
  weight_projects: number;
  confidence_threshold: number;
  eligibility_threshold: number;
  updated_at: string;
}

export interface AppNotification {
  id: string;
  type: string;
  title: string;
  body: string;
  entity_type?: string;
  entity_id?: string;
  read_at?: string;
  created_at: string;
}

export interface AuditLog {
  id: string;
  actor_user_id?: string;
  action: string;
  resource_type: string;
  resource_id?: string;
  meta?: Record<string, unknown>;
  created_at: string;
}

export interface Interview {
  id: string;
  candidate_id: string;
  job_id?: string;
  title: string;
  scheduled_at: string;
  duration_minutes: number;
  timezone: string;
  location?: string;
  meeting_url?: string;
  status: string;
  candidate_name?: string;
}

export async function getAnalyticsOverview(): Promise<AnalyticsOverview> {
  return apiClient<AnalyticsOverview>("/analytics/overview", { auth: true });
}

export async function getAISettings(): Promise<CompanyAISettings> {
  const data = await apiClient<{ settings: CompanyAISettings }>("/settings/ai", { auth: true });
  return data.settings;
}

export async function updateAISettings(payload: Partial<CompanyAISettings>): Promise<CompanyAISettings> {
  const data = await apiClient<{ settings: CompanyAISettings }>("/settings/ai", {
    method: "PUT",
    body: payload,
    auth: true,
  });
  return data.settings;
}

export async function listNotifications(): Promise<{ notifications: AppNotification[]; unread: number }> {
  return apiClient("/notifications", { auth: true });
}

export async function markNotificationRead(id: string): Promise<void> {
  await apiClient(`/notifications/${id}/read`, { method: "POST", auth: true });
}

export async function markAllNotificationsRead(): Promise<void> {
  await apiClient("/notifications/read-all", { method: "POST", auth: true });
}

export async function listAuditLogs(params?: { limit?: number; offset?: number }): Promise<AuditLog[]> {
  const q = new URLSearchParams();
  if (params?.limit) q.set("limit", String(params.limit));
  if (params?.offset) q.set("offset", String(params.offset));
  const query = q.toString();
  const data = await apiClient<{ logs: AuditLog[] }>(`/audit-logs${query ? `?${query}` : ""}`, {
    auth: true,
  });
  return data.logs ?? [];
}

export async function listInterviews(params?: { from?: string; to?: string }): Promise<Interview[]> {
  const q = new URLSearchParams();
  if (params?.from) q.set("from", params.from);
  if (params?.to) q.set("to", params.to);
  const query = q.toString();
  const data = await apiClient<{ interviews: Interview[] }>(
    `/interviews${query ? `?${query}` : ""}`,
    { auth: true },
  );
  return data.interviews ?? [];
}

export async function createInterview(payload: {
  candidate_id: string;
  job_id?: string;
  title?: string;
  scheduled_at: string;
  duration_minutes?: number;
  timezone?: string;
  location?: string;
  meeting_url?: string;
  notes?: string;
}): Promise<Interview> {
  const data = await apiClient<{ interview: Interview }>("/interviews", {
    method: "POST",
    body: payload,
    auth: true,
  });
  return data.interview;
}

export interface CollaborationComment {
  id: string;
  candidate_id: string;
  author_user_id?: string;
  body: string;
  mentions?: string[];
  created_at: string;
}

export async function listCandidateComments(candidateId: string): Promise<CollaborationComment[]> {
  const data = await apiClient<{ comments: CollaborationComment[] }>(
    `/candidates/${candidateId}/comments`,
    { auth: true },
  );
  return data.comments ?? [];
}

export async function createCandidateComment(
  candidateId: string,
  body: string,
  mentions?: string[],
): Promise<CollaborationComment> {
  const data = await apiClient<{ comment: CollaborationComment }>(
    `/candidates/${candidateId}/comments`,
    {
      method: "POST",
      body: { body, mentions },
      auth: true,
    },
  );
  return data.comment;
}

export async function assignCandidate(candidateId: string, assignedTo: string | null): Promise<void> {
  await apiClient(`/candidates/${candidateId}/assign`, {
    method: "POST",
    body: { assigned_to: assignedTo },
    auth: true,
  });
}
