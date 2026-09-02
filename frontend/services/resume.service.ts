import { apiClient, ApiClientError } from "@/lib/api-client";
import { getToken, removeToken } from "@/lib/auth-storage";
import type { ApiResponse } from "@/lib/types";
import type { ResumeUploadResult } from "@/lib/resume-types";

const BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL ?? "/api/v1";

export async function uploadResume(file: File): Promise<ResumeUploadResult> {
  const token = getToken();
  if (!token) {
    if (typeof window !== "undefined" && !window.location.pathname.startsWith("/login")) {
      window.location.href = "/login";
    }
    throw new ApiClientError("Please log in to upload a resume.", 401, "unauthorized");
  }

  const form = new FormData();
  form.append("file", file);

  const headers: HeadersInit = {
    Authorization: `Bearer ${token}`,
  };

  let res: Response;
  try {
    res = await fetch(`${BASE_URL}/resumes/upload`, {
      method: "POST",
      headers,
      body: form,
    });
  } catch {
    throw new ApiClientError("Network error. Check your connection and try again.", 0, "network_error");
  }

  let json: ApiResponse<ResumeUploadResult> | null = null;
  try {
    json = await res.json();
  } catch {
    /* non-JSON */
  }

  if (res.status === 401) {
    removeToken();
    if (typeof window !== "undefined" && !window.location.pathname.startsWith("/login")) {
      window.location.href = "/login";
    }
    throw new ApiClientError("Session expired. Please log in again.", 401, "unauthorized");
  }

  if (!json || !json.success) {
    const message = json && !json.success ? json.error.message : "Resume upload failed";
    const code = json && !json.success ? json.error.code : "unknown";
    throw new ApiClientError(message, res.status, code);
  }

  return json.data;
}

export async function attachResume(resumeId: string, candidateId: string): Promise<void> {
  await apiClient<{ message: string }>(`/resumes/${resumeId}/attach`, {
    method: "POST",
    body: { candidate_id: candidateId },
    auth: true,
  });
}
