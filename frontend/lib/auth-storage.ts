const TOKEN_KEY = "ats_access_token";
const USER_KEY = "ats_user";

export function getToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem(TOKEN_KEY);
}

export function setToken(token: string): void {
  localStorage.setItem(TOKEN_KEY, token);
}

export function removeToken(): void {
  localStorage.removeItem(TOKEN_KEY);
  localStorage.removeItem(USER_KEY);
}

export function isAuthenticated(): boolean {
  return !!getToken();
}

export function setStoredUser(user: { id: string; email: string; role: string; first_name: string; last_name: string }): void {
  localStorage.setItem(USER_KEY, JSON.stringify(user));
}

export function getStoredUser(): {
  id: string;
  email: string;
  role: string;
  first_name: string;
  last_name: string;
} | null {
  if (typeof window === "undefined") return null;
  try {
    const raw = localStorage.getItem(USER_KEY);
    if (!raw) return null;
    return JSON.parse(raw) as {
      id: string;
      email: string;
      role: string;
      first_name: string;
      last_name: string;
    };
  } catch {
    return null;
  }
}

export function isAdminRole(role?: string | null): boolean {
  return (role ?? "").toLowerCase() === "admin";
}
