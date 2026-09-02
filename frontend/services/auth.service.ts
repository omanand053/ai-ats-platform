import { apiClient } from "@/lib/api-client";
import { removeToken, setToken, setStoredUser } from "@/lib/auth-storage";
import type { AuthData, LoginPayload, SignupPayload, User } from "@/lib/types";

export async function signup(payload: SignupPayload): Promise<AuthData> {
  const data = await apiClient<AuthData>("/auth/signup", {
    method: "POST",
    body: payload,
  });
  setToken(data.access_token);
  setStoredUser({
    id: data.user.id,
    email: data.user.email,
    role: data.user.role,
    first_name: data.user.first_name,
    last_name: data.user.last_name,
  });
  return data;
}

export async function login(payload: LoginPayload): Promise<AuthData> {
  const data = await apiClient<AuthData>("/auth/login", {
    method: "POST",
    body: payload,
  });
  setToken(data.access_token);
  setStoredUser({
    id: data.user.id,
    email: data.user.email,
    role: data.user.role,
    first_name: data.user.first_name,
    last_name: data.user.last_name,
  });
  return data;
}

export async function getMe(): Promise<User> {
  const data = await apiClient<{ user: User }>("/auth/me", { auth: true });
  setStoredUser({
    id: data.user.id,
    email: data.user.email,
    role: data.user.role,
    first_name: data.user.first_name,
    last_name: data.user.last_name,
  });
  return data.user;
}

export function logout(): void {
  removeToken();
}
