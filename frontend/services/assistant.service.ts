import { apiClient, ApiClientError } from "@/lib/api-client";
import { getToken, removeToken } from "@/lib/auth-storage";
import type { ApiResponse } from "@/lib/types";
import type { ResumeUploadResult } from "@/lib/resume-types";

const BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL ?? "/api/v1";

export type AssistantPayload = {
  answer?: string;
  reply?: string;
  confidence?: number;
  source?: string;
  intent?: string;
  suggested_actions?: string[];
  source_documents?: { id?: string; label: string; similarity?: number }[];
  data?: {
    jobs?: { id?: string; title?: string; location?: string; status?: string }[];
    candidates?: { id?: string; name?: string; email?: string; status?: string; first_name?: string; last_name?: string }[];
    summary?: Record<string, unknown>;
    overview?: Record<string, unknown>;
  };
};

export type AssistantUpload = {
  id: string;
  file_name: string;
  status: string;
  error?: string;
};

/** Synchronous chat — restored stable `/assistant/chat` only. */
export async function sendAssistantChat(prompt: string, _sessionId: string, resumeId?: string) {
  return apiClient<AssistantPayload>("/assistant/chat", {
    method: "POST",
    auth: true,
    body: {
      prompt,
      ...(resumeId ? { resume_id: resumeId } : {}),
    },
  });
}

/** Attach via existing resume upload (parse + embed). No new endpoints. */
export async function uploadAssistantAttachment(file: File): Promise<AssistantUpload> {
  const token = getToken();
  if (!token) {
    throw new ApiClientError("Please log in to upload a resume.", 401, "unauthorized");
  }

  const form = new FormData();
  form.append("file", file);

  let res: Response;
  try {
    res = await fetch(`${BASE_URL}/resumes/upload`, {
      method: "POST",
      headers: { Authorization: `Bearer ${token}` },
      body: form,
    });
  } catch {
    throw new ApiClientError("Network error. Check your connection and try again.", 0, "network_error");
  }

  const json = (await res.json().catch(() => null)) as ApiResponse<ResumeUploadResult> | null;

  if (res.status === 401) {
    removeToken();
    throw new ApiClientError("Session expired. Please log in again.", 401, "unauthorized");
  }

  if (!json?.success || !json.data?.resume_id) {
    throw new ApiClientError(
      json && "error" in json && json.error?.message ? json.error.message : "Upload failed",
      res.status,
      (json && "error" in json && json.error?.code) || "upload_error",
    );
  }

  return {
    id: String(json.data.resume_id),
    file_name: json.data.file_name || file.name,
    status: json.data.parsing_status === "completed" ? "attached" : "processing",
  };
}

export function formatAssistantAnswer(data: AssistantPayload): string {
  if (data.answer?.trim()) return data.answer.trim();
  if (data.reply?.trim()) return data.reply.trim();

  if (data.intent && data.data) {
    switch (data.intent) {
      case "search_jobs": {
        const jobs = data.data.jobs || [];
        return (
          jobs
            .slice(0, 8)
            .map((j) => `• ${j.title} — ${j.location || ""} (${j.status || ""})`)
            .join("\n") || "No jobs found."
        );
      }
      case "search_candidates": {
        const candidates = data.data.candidates || [];
        return (
          candidates
            .slice(0, 8)
            .map((c) => {
              const name = c.name || `${c.first_name || ""} ${c.last_name || ""}`.trim();
              return `• ${name} — ${c.email || ""} (${c.status || ""})`;
            })
            .join("\n") || "No candidates found."
        );
      }
      case "candidate_counts": {
        const s = data.data.summary || {};
        return `Candidates: ${s.candidates ?? 0}\nApplications: ${s.applications ?? 0}\nInterviews: ${s.interviews ?? 0}\nOffers: ${s.offers ?? 0}\nHired: ${s.hired ?? 0}`;
      }
      case "application_stats": {
        const summary = data.data.summary || data.data;
        const total = (summary as { applications?: number }).applications ?? 0;
        const byStatus = (summary as { by_status?: Record<string, number> }).by_status || {};
        let human = `Total applications: ${total}`;
        for (const [status, count] of Object.entries(byStatus)) {
          human += `\n• ${status}: ${count}`;
        }
        return human;
      }
      case "dashboard_summary": {
        const summary = data.data.summary || {};
        return `Jobs: ${summary.total_jobs ?? 0} (open: ${summary.open_jobs ?? 0})\nApplications: ${summary.applications ?? 0}\nInterviews: ${summary.interviews ?? 0}\nOffers: ${summary.offers ?? 0}\nHired: ${summary.hired ?? 0}`;
      }
      default:
        return JSON.stringify(data.data);
    }
  }

  return "No response.";
}

export function pageQuickActions(pathname: string): { label: string; prompt: string }[] {
  const base = [
    { label: "Find Candidates", prompt: "Find React candidates" },
    { label: "Review Resume", prompt: "Review the uploaded resume" },
    { label: "Generate Interview", prompt: "Generate interview questions" },
    { label: "Hiring Insights", prompt: "Hiring funnel" },
  ];
  if (pathname === "/dashboard" || pathname.endsWith("/dashboard")) {
    return [{ label: "Hiring Summary", prompt: "Dashboard overview" }, ...base];
  }
  if (pathname.includes("/candidates")) {
    return [{ label: "Compare Candidates", prompt: "Compare top candidates by AI score" }, ...base];
  }
  if (pathname.includes("/jobs")) {
    return [
      { label: "Generate JD", prompt: "Best practices for writing a software engineer job description" },
      ...base,
    ];
  }
  return base;
}

export function assistantSessionId(): string {
  if (typeof window === "undefined") return "";
  const key = "ats_assistant_session";
  let id = sessionStorage.getItem(key);
  if (!id) {
    id = `sess_${Date.now()}_${Math.random().toString(36).slice(2, 10)}`;
    sessionStorage.setItem(key, id);
  }
  return id;
}
