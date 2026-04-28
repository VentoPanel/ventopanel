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

// Auth
export interface LoginResponse {
  token: string;
  email: string;
  role: string;
}

export async function login(email: string, password: string): Promise<LoginResponse> {
  const res = await fetch("/api/v1/auth/login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password }),
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new ApiError(res.status, body.error ?? "Login failed");
  }
  return res.json() as Promise<LoginResponse>;
}

export async function registerUser(params: {
  email: string;
  password: string;
  team_id: string;
}): Promise<{ id: string; email: string; role: string }> {
  const res = await fetch("/api/v1/auth/register", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(params),
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new ApiError(res.status, body.error ?? "Registration failed");
  }
  return res.json();
}

// Audit
export interface AuditEvent {
  ID: string;
  ResourceType: string;
  ResourceID: string;
  FromStatus: string;
  ToStatus: string;
  Reason: string;
  TaskID: string;
  CreatedAt: string;
}

export interface AuditPage {
  items: AuditEvent[];
  next_cursor: string;
}

export async function fetchAuditEvents(params: {
  resource_type?: string;
  resource_id?: string;
  limit?: number;
  before?: string;
}): Promise<AuditPage> {
  const qs = new URLSearchParams();
  if (params.resource_type) qs.set("resource_type", params.resource_type);
  if (params.resource_id) qs.set("resource_id", params.resource_id);
  if (params.limit) qs.set("limit", String(params.limit));
  if (params.before) qs.set("before", params.before);

  return apiFetch<AuditPage>(`/audit?${qs.toString()}`);
}

export interface ServerStats {
  cpu_cores: number;
  load_avg_1: number;
  ram_total_mb: number;
  ram_used_mb: number;
  disk_total: string;
  disk_used: string;
  disk_free: string;
  disk_pct: string;
  uptime: string;
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

export async function fetchServerStats(id: string): Promise<ServerStats> {
  return apiFetch<ServerStats>(`/servers/${id}/stats`);
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

export async function fetchSiteByID(id: string): Promise<Site> {
  return apiFetch<Site>(`/sites/${id}`);
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
