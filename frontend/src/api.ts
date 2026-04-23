import type { ApiErrorPayload, TokenPair } from "./types";

export const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || "/api";

export class ApiError extends Error {
  status: number;
  payload: ApiErrorPayload | null;

  constructor(status: number, payload: ApiErrorPayload | null) {
    super(payload?.message || payload?.error || `Request failed with status ${status}`);
    this.name = "ApiError";
    this.status = status;
    this.payload = payload;
  }
}

type ApiOptions = RequestInit & {
  token?: string | null;
};

export async function apiRequest<T>(path: string, options: ApiOptions = {}): Promise<T> {
  const headers = new Headers(options.headers);
  const hasBody = options.body != null;

  if (hasBody && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }
  if (options.token) {
    headers.set("Authorization", `Bearer ${options.token}`);
  }

  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...options,
    headers
  });

  if (response.status === 204) {
    return undefined as T;
  }

  const contentType = response.headers.get("Content-Type") || "";
  const data = contentType.includes("application/json") ? await response.json() : null;

  if (!response.ok) {
    throw new ApiError(response.status, data);
  }

  return data as T;
}

export function saveTokens(tokens: TokenPair) {
  localStorage.setItem("pingme.accessToken", tokens.access_token);
  localStorage.setItem("pingme.refreshToken", tokens.refresh_token);
}

export function loadTokens(): TokenPair | null {
  const accessToken = localStorage.getItem("pingme.accessToken");
  const refreshToken = localStorage.getItem("pingme.refreshToken");
  if (!accessToken || !refreshToken) {
    return null;
  }
  return {
    access_token: accessToken,
    refresh_token: refreshToken
  };
}

export function clearTokens() {
  localStorage.removeItem("pingme.accessToken");
  localStorage.removeItem("pingme.refreshToken");
}
