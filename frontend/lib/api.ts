export const TOKEN_KEY = "vp_token";

export function getToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem(TOKEN_KEY);
}

export function setToken(token: string): void {
  localStorage.setItem(TOKEN_KEY, token);
}

export function clearToken(): void {
  localStorage.removeItem(TOKEN_KEY);
}

export interface Server {
  ID: string;
  Name: string;
  Host: string;
  Port: number;
  Provider: string;
  Status: string;
  SSHUser: string;
  CreatedAt: string;
  UpdatedAt: string;
}

export interface Site {
  ID: string;
  ServerID: string;
  Name: string;
  Domain: string;
  Runtime: string;
  RepositoryURL: string;
  Status: string;
  CreatedAt: string;
  UpdatedAt: string;
}

interface ListResponse<T> {
  items: T[];
}

class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

async function apiFetch<T>(path: string): Promise<T> {
  const token = getToken();
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
  };
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  const res = await fetch(`/api/v1${path}`, { headers });

  if (res.status === 401 || res.status === 403) {
    if (typeof window !== "undefined") {
      window.location.href = "/login";
    }
    throw new ApiError(res.status, "Unauthorized");
  }

  if (!res.ok) {
    const body = await res.text();
    throw new ApiError(res.status, body);
  }

  return res.json() as Promise<T>;
}

export async function fetchServers(): Promise<Server[]> {
  const data = await apiFetch<ListResponse<Server>>("/servers");
  return data.items ?? [];
}

export async function fetchSites(): Promise<Site[]> {
  const data = await apiFetch<ListResponse<Site>>("/sites");
  return data.items ?? [];
}
