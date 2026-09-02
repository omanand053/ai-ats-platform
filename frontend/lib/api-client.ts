import { getToken, removeToken } from "./auth-storage";
import type { ApiResponse } from "./types";

const BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL ?? "/api/v1";

export class ApiClientError extends Error {
  status: number;
  code: string;

  constructor(message: string, status: number, code = "unknown") {
    super(message);
    this.status = status;
    this.code = code;
  }
}

type RequestOptions = Omit<RequestInit, "body"> & {
  body?: unknown;
  auth?: boolean;
};

export async function apiClient<T>(
  path: string,
  options: RequestOptions = {},
): Promise<T> {
  const { body, auth = false, headers, ...rest } = options;

  const reqHeaders: HeadersInit = {
    "Content-Type": "application/json",
    ...headers,
  };

  if (auth) {
    const token = getToken();
    if (!token) {
      if (typeof window !== "undefined" && !window.location.pathname.startsWith("/login")) {
        window.location.href = "/login";
      }
      throw new ApiClientError("Please log in to continue.", 401, "unauthorized");
    }
    (reqHeaders as Record<string, string>)["Authorization"] = `Bearer ${token}`;
  }

  let res: Response;
  try {
    res = await fetch(`${BASE_URL}${path}`, {
      ...rest,
      headers: reqHeaders,
      body: body !== undefined ? JSON.stringify(body) : undefined,
    });
  } catch {
    throw new ApiClientError("Network error. Check your connection and try again.", 0, "network_error");
  }

  let json: ApiResponse<T> | null = null;
  try {
    json = await res.json();
  } catch {
    /* non-JSON response */
  }

  // Only treat 401 as an expired session for authenticated requests.
  // Login/signup also return 401 for bad credentials — surface that message instead.
  if (res.status === 401 && auth) {
    removeToken();
    if (typeof window !== "undefined" && !window.location.pathname.startsWith("/login")) {
      window.location.href = "/login";
    }
    throw new ApiClientError("Session expired. Please log in again.", 401, "unauthorized");
  }

  if (!json || !json.success) {
    const message =
      json && !json.success
        ? json.error.message
        : res.status >= 500
          ? "The server ran into a problem. Please retry in a moment."
          : res.status === 404
            ? "The requested resource was not found."
            : res.status === 403
              ? "You don’t have permission to perform this action."
              : `Request failed (${res.status || "unknown"}). Please try again.`;
    const code = json && !json.success ? json.error.code : "unknown";
    throw new ApiClientError(message, res.status, code);
  }

  return json.data;
}
