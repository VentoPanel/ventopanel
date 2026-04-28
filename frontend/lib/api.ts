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

export type ServerInput = {
  name: string;
  host: string;
  port: number;
  provider: string;
  ssh_user: string;
  ssh_password: string;
  status?: string;
};

export type SiteInput = {
  server_id: string;
  name: string;
  domain: string;
  runtime: string;
  repository_url: string;
  status?: string;
};

async function apiFetch<T>(
  path: string,
  options?: RequestInit,
): Promise<T> {
  const token = getToken();
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...(options?.headers as Record<string, string>),
  };
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  const res = await fetch(`/api/v1${path}`, { ...options, headers });

  if (res.status === 401 || res.status === 403) {
    if (typeof window !== "undefined") {
      window.location.href = "/login";
    }
    throw new ApiError(res.status, "Unauthorized");
  }

  if (res.status === 204) {
    return undefined as T;
  }

  if (!res.ok) {
    const body = await res.text();
    throw new ApiError(res.status, body);
  }

  return res.json() as Promise<T>;
}

// Servers
export async function fetchServers(): Promise<Server[]> {
  const data = await apiFetch<ListResponse<Server>>("/servers");
  return data.items ?? [];
}

export async function createServer(input: ServerInput): Promise<Server> {
  return apiFetch<Server>("/servers", {
    method: "POST",
    body: JSON.stringify({ ...input, status: input.status ?? "pending" }),
  });
}

export async function updateServer(
  id: string,
  input: ServerInput,
): Promise<Server> {
  return apiFetch<Server>(`/servers/${id}`, {
    method: "PUT",
    body: JSON.stringify(input),
  });
}

export async function deleteServer(id: string): Promise<void> {
  return apiFetch<void>(`/servers/${id}`, { method: "DELETE" });
}

export async function connectServer(id: string): Promise<Server> {
  return apiFetch<Server>(`/servers/${id}/connect`, { method: "POST" });
}

export async function provisionServer(id: string): Promise<{ status: string }> {
  return apiFetch<{ status: string }>(`/servers/${id}/provision`, {
    method: "POST",
  });
}

// Sites
export async function fetchSites(): Promise<Site[]> {
  const data = await apiFetch<ListResponse<Site>>("/sites");
  return data.items ?? [];
}

export async function createSite(input: SiteInput): Promise<Site> {
  return apiFetch<Site>("/sites", {
    method: "POST",
    body: JSON.stringify({ ...input, status: input.status ?? "draft" }),
  });
}

export async function updateSite(
  id: string,
  input: SiteInput,
): Promise<Site> {
  return apiFetch<Site>(`/sites/${id}`, {
    method: "PUT",
    body: JSON.stringify(input),
  });
}

export async function deleteSite(id: string): Promise<void> {
  return apiFetch<void>(`/sites/${id}`, { method: "DELETE" });
}

export async function deploySite(id: string): Promise<{ status: string }> {
  return apiFetch<{ status: string }>(`/sites/${id}/deploy`, {
    method: "POST",
  });
}
