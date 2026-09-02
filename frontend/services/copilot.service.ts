import { getToken } from "@/lib/auth-storage";
import { apiClient } from "@/lib/api-client";

const BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL ?? "/api/v1";

export type CopilotIntent = "ATS_DATA" | "RAG" | "GENERAL" | string;

export type SourceDoc = {
  id?: string;
  label: string;
  type?: string;
  similarity?: number;
};

export type QuickAction = {
  id: string;
  label: string;
  prompt?: string;
  href?: string;
};

export type UICard = {
  type: "candidate" | "job" | "resume" | string;
  title: string;
  subtitle?: string;
  meta?: Record<string, unknown>;
  href?: string;
};

export type CopilotPayload = {
  answer?: string;
  reply?: string;
  confidence?: number;
  source?: string;
  intent?: CopilotIntent;
  suggested_actions?: string[];
  quick_actions?: QuickAction[];
  source_documents?: SourceDoc[];
  retrieved_context_count?: number;
  cards?: UICard[];
  data?: Record<string, unknown>;
  provider?: string;
  model?: string;
};

export type CopilotStatus = {
  connected: boolean;
  knowledge_ready: boolean;
  provider: string;
  model: string;
  resume_index_count: number;
  job_count: number;
  candidate_count: number;
  session_uploads: number;
};

export type CopilotUpload = {
  id: string;
  file_name: string;
  mime_type?: string;
  mode: string;
  status: string;
  error?: string;
  chunk_count?: number;
  char_count?: number;
  preview?: string;
  parsed_name?: string;
  parsed_email?: string;
  skills?: string[];
  ext?: string;
  imported_candidate_id?: string;
};

export function copilotSessionId(): string {
  if (typeof window === "undefined") return "";
  const key = "ats_copilot_session";
  let id = sessionStorage.getItem(key);
  if (!id) {
    id = `sess_${Date.now()}_${Math.random().toString(36).slice(2, 10)}`;
    sessionStorage.setItem(key, id);
  }
  return id;
}

export async function getCopilotStatus(sessionId: string) {
  return apiClient<CopilotStatus>(`/assistant/status?session_id=${encodeURIComponent(sessionId)}`, {
    auth: true,
  });
}

export async function listCopilotUploads(sessionId: string) {
  const res = await apiClient<{ uploads: CopilotUpload[] }>(
    `/assistant/uploads?session_id=${encodeURIComponent(sessionId)}`,
    { auth: true },
  );
  return res.uploads ?? [];
}

export async function uploadCopilotFile(file: File, sessionId: string, mode: "temporary" | "import" = "temporary") {
  const token = getToken();
  const form = new FormData();
  form.append("file", file);
  form.append("session_id", sessionId);
  form.append("mode", mode);
  const res = await fetch(`${BASE_URL}/assistant/uploads`, {
    method: "POST",
    headers: token ? { Authorization: `Bearer ${token}` } : {},
    body: form,
  });
  const json = await res.json();
  if (!res.ok) {
    throw new Error(json?.error?.message || "Upload failed");
  }
  return (json.data ?? json) as CopilotUpload;
}

export async function pasteCopilotText(text: string, sessionId: string, title = "pasted-text.txt") {
  return apiClient<CopilotUpload>("/assistant/uploads", {
    method: "POST",
    auth: true,
    body: { text, title, session_id: sessionId, mode: "temporary" },
  });
}

export async function deleteCopilotUpload(id: string) {
  return apiClient<{ deleted: boolean }>(`/assistant/uploads/${id}`, { method: "DELETE", auth: true });
}

export async function importCopilotUpload(id: string, jobId?: string) {
  return apiClient<{ upload: CopilotUpload; candidate: { id: string; name: string } }>(
    `/assistant/uploads/${id}/import`,
    { method: "POST", auth: true, body: jobId ? { job_id: jobId } : {} },
  );
}

export type StreamHandlers = {
  onMeta?: (meta: { intent?: string }) => void;
  onToken?: (text: string) => void;
  onDone?: (payload: CopilotPayload) => void;
  onError?: (message: string) => void;
};

export async function streamCopilotChat(
  prompt: string,
  sessionId: string,
  uploadId: string | undefined,
  handlers: StreamHandlers,
  signal?: AbortSignal,
) {
  const token = getToken();
  const res = await fetch(`${BASE_URL}/assistant/chat/stream`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: JSON.stringify({ prompt, session_id: sessionId, upload_id: uploadId }),
    signal,
  });
  if (!res.ok || !res.body) {
    const json = await res.json().catch(() => null);
    throw new Error(json?.error?.message || "Streaming failed");
  }

  const reader = res.body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";
  let eventName = "message";

  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    buffer += decoder.decode(value, { stream: true });
    const parts = buffer.split("\n");
    buffer = parts.pop() ?? "";
    for (const line of parts) {
      const trimmed = line.trimEnd();
      if (trimmed.startsWith("event:")) {
        eventName = trimmed.slice(6).trim();
        continue;
      }
      if (trimmed.startsWith("data:")) {
        const raw = trimmed.slice(5).trim();
        if (!raw) continue;
        try {
          const data = JSON.parse(raw);
          if (eventName === "meta") handlers.onMeta?.(data);
          else if (eventName === "token") handlers.onToken?.(data.text ?? "");
          else if (eventName === "error") handlers.onError?.(data.message ?? "Stream error");
          else if (eventName === "done") handlers.onDone?.(data as CopilotPayload);
        } catch {
          /* ignore partial JSON */
        }
        eventName = "message";
      }
    }
  }
}

export function pageQuickActions(pathname: string): QuickAction[] {
  const base: QuickAction[] = [
    { id: "find", label: "Find Candidates", prompt: "Find React candidates" },
    { id: "review", label: "Review Resume", prompt: "Review the uploaded resume" },
    { id: "interview", label: "Generate Interview", prompt: "Generate interview questions" },
    { id: "insights", label: "Hiring Insights", prompt: "Hiring funnel" },
  ];
  if (pathname === "/dashboard" || pathname.endsWith("/dashboard")) {
    return [{ id: "summary", label: "Hiring Summary", prompt: "Dashboard overview" }, ...base];
  }
  if (pathname.includes("/candidates")) {
    return [{ id: "compare", label: "Compare Candidates", prompt: "Compare top candidates by AI score" }, ...base];
  }
  if (pathname.includes("/jobs")) {
    return [{ id: "jd", label: "Generate JD", prompt: "Best practices for writing a software engineer job description" }, ...base];
  }
  return base;
}
