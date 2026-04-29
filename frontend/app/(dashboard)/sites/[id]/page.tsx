"use client";

import { use, useState } from "react";
import Link from "next/link";
import {
  ArrowLeft,
  Globe,
  Rocket,
  Pencil,
  Trash2,
  ExternalLink,
  Clock,
  ChevronDown,
  ChevronRight,
  ShieldCheck,
  ShieldAlert,
  ShieldX,
  RefreshCw,
  Container,
  RotateCcw,
  Terminal,
  KeyRound,
  Plus,
  X,
  Eye,
  EyeOff,
  Webhook,
  Copy,
  Check,
} from "lucide-react";
import { formatDistanceToNow, format } from "date-fns";
import { useQuery, useQueryClient, useMutation } from "@tanstack/react-query";
import {
  fetchSiteByID,
  fetchSiteLogs,
  fetchSiteSSL,
  renewSiteSSL,
  fetchContainerInfo,
  fetchContainerLogs,
  restartContainer,
  fetchEnvVars,
  upsertEnvVar,
  deleteEnvVar,
  regenerateWebhookToken,
  type TaskLog,
  type SSLCertInfo,
  type ContainerInfo,
  type EnvVarItem,
} from "@/lib/api";
import { useAuditEvents } from "@/hooks/use-audit";
import { useDeploySite, useDeleteSite } from "@/hooks/use-site-mutations";
import { useAuth } from "@/hooks/use-auth";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { SiteForm } from "@/components/site-form";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { RefreshIndicator } from "@/components/refresh-indicator";
import { toast } from "sonner";
import { cn } from "@/lib/utils";
import { useRouter } from "next/navigation";

const STATUS_COLORS: Record<string, string> = {
  access_denied: "bg-red-100 text-red-700",
  deploy_failed: "bg-red-100 text-red-700",
  deployed: "bg-green-100 text-green-700",
  ssl_active: "bg-green-100 text-green-700",
  deploying: "bg-blue-100 text-blue-700",
  ssl_pending: "bg-yellow-100 text-yellow-700",
};

function statusBadgeVariant(
  status: string,
): "success" | "destructive" | "warning" | "secondary" {
  switch (status.toLowerCase()) {
    case "deployed":
    case "ssl_active":
      return "success";
    case "deploy_failed":
    case "ssl_failed":
      return "destructive";
    case "deploying":
    case "ssl_pending":
      return "warning";
    default:
      return "secondary";
  }
}

function statusClass(s: string) {
  return STATUS_COLORS[s] ?? "bg-muted text-muted-foreground";
}

function formatDate(iso: string) {
  return new Date(iso).toLocaleString(undefined, {
    dateStyle: "short",
    timeStyle: "medium",
  });
}

export default function SiteDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  const router = useRouter();
  const qc = useQueryClient();

  const { data: site, isFetching: siteFetching, dataUpdatedAt } = useQuery({
    queryKey: ["site", id],
    queryFn: () => fetchSiteByID(id),
    refetchInterval: 15_000,
    refetchIntervalInBackground: false,
  });

  const {
    data: auditData,
    isLoading: auditLoading,
    hasNextPage,
    fetchNextPage,
    isFetchingNextPage,
  } = useAuditEvents({ resource_type: "site", resource_id: id });
  const events = auditData?.pages.flatMap((p) => p.items) ?? [];

  const { data: logs = [], isFetching: logsFetching } = useQuery({
    queryKey: ["site-logs", id],
    queryFn: () => fetchSiteLogs(id, 20),
    refetchInterval: 15_000,
    refetchIntervalInBackground: false,
  });

  const { isAdmin, canWrite } = useAuth();

  const { data: sslInfo } = useQuery({
    queryKey: ["site-ssl", id],
    queryFn: () => fetchSiteSSL(id),
    refetchInterval: 60_000,
    refetchIntervalInBackground: false,
    retry: false,
  });

  const renewSSL = useMutation({
    mutationFn: () => renewSiteSSL(id),
    onSuccess: () => {
      toast.success("SSL renewal queued");
      qc.invalidateQueries({ queryKey: ["site-ssl", id] });
    },
    onError: (err) => toast.error(err instanceof Error ? err.message : "Renew failed"),
  });

  const deploySite = useDeploySite();
  const deleteSite = useDeleteSite();

  const [editOpen, setEditOpen] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [expandedLog, setExpandedLog] = useState<string | null>(null);
  const [showContainerLogs, setShowContainerLogs] = useState(false);
  const [restarting, setRestarting] = useState(false);

  const hasRepo = Boolean(site?.RepositoryURL?.trim());

  const { data: containerInfo, refetch: refetchContainer } = useQuery<ContainerInfo>({
    queryKey: ["container", id],
    queryFn: () => fetchContainerInfo(id),
    enabled: hasRepo,
    refetchInterval: 20_000,
    refetchIntervalInBackground: false,
    retry: false,
  });

  const { data: containerLogs = "", isFetching: containerLogsFetching } = useQuery<string>({
    queryKey: ["container-logs", id],
    queryFn: () => fetchContainerLogs(id, 200),
    enabled: hasRepo && showContainerLogs,
    refetchInterval: showContainerLogs ? 10_000 : false,
    refetchIntervalInBackground: false,
    retry: false,
  });

  // ENV vars state
  const [newKey, setNewKey] = useState("");
  const [newValue, setNewValue] = useState("");
  const [addingEnv, setAddingEnv] = useState(false);
  const [visibleValues, setVisibleValues] = useState<Set<string>>(new Set());

  const { data: envVars = [], refetch: refetchEnv } = useQuery<EnvVarItem[]>({
    queryKey: ["env", id],
    queryFn: () => fetchEnvVars(id),
    enabled: hasRepo,
    retry: false,
  });

  async function handleUpsertEnv(e: React.FormEvent) {
    e.preventDefault();
    const key = newKey.trim().toUpperCase();
    if (!key) return;
    try {
      await upsertEnvVar(id, key, newValue);
      toast.success(`${key} saved`);
      setNewKey("");
      setNewValue("");
      setAddingEnv(false);
      refetchEnv();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Save failed");
    }
  }

  async function handleDeleteEnv(key: string) {
    try {
      await deleteEnvVar(id, key);
      toast.success(`${key} deleted`);
      refetchEnv();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Delete failed");
    }
  }

  function toggleVisible(key: string) {
    setVisibleValues((prev) => {
      const next = new Set(prev);
      if (next.has(key)) next.delete(key); else next.add(key);
      return next;
    });
  }

  // Webhook state
  const [webhookToken, setWebhookToken] = useState<string>("");
  const [copied, setCopied] = useState(false);
  const [regenerating, setRegenerating] = useState(false);

  // Sync webhook token from site data
  const currentToken = webhookToken || site?.WebhookToken || "";
  const apiBase = typeof window !== "undefined"
    ? `${window.location.protocol}//${window.location.host}`
    : "";
  const webhookURL = currentToken
    ? `${apiBase}/api/v1/webhook/${currentToken}`
    : "";

  async function handleCopyWebhook() {
    if (!webhookURL) return;
    await navigator.clipboard.writeText(webhookURL);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }

  async function handleRegenerate() {
    setRegenerating(true);
    try {
      const token = await regenerateWebhookToken(id);
      setWebhookToken(token);
      toast.success("Webhook token regenerated");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed");
    } finally {
      setRegenerating(false);
    }
  }

  async function handleRestart() {
    setRestarting(true);
    try {
      await restartContainer(id);
      toast.success("Container restarted");
      setTimeout(() => refetchContainer(), 2000);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Restart failed");
    } finally {
      setRestarting(false);
    }
  }

  async function handleDeploy() {
    try {
      await deploySite.mutateAsync(id);
      toast.success("Deploy queued");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Deploy failed");
    }
  }

  async function handleDelete() {
    try {
      await deleteSite.mutateAsync(id);
      toast.success("Site deleted");
      router.push("/sites");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Delete failed");
      setDeleteOpen(false);
    }
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Button variant="ghost" size="icon" asChild>
            <Link href="/sites">
              <ArrowLeft className="h-4 w-4" />
            </Link>
          </Button>
          <Globe className="h-6 w-6 text-muted-foreground" />
          <div>
            <h2 className="text-2xl font-bold tracking-tight">
              {site?.Name ?? id}
            </h2>
            <div className="flex items-center gap-1 text-sm text-muted-foreground">
              <span className="font-mono">{site?.Domain ?? "Loading…"}</span>
              {site?.Domain && (
                <a
                  href={`http://${site.Domain}`}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="hover:text-foreground"
                >
                  <ExternalLink className="h-3 w-3" />
                </a>
              )}
            </div>
          </div>
          {site && (
            <Badge variant={statusBadgeVariant(site.Status)}>
              {site.Status}
            </Badge>
          )}
        </div>

        <div className="flex items-center gap-2">
          <RefreshIndicator
            isFetching={siteFetching}
            dataUpdatedAt={dataUpdatedAt}
            intervalSeconds={15}
            onRefresh={() => {
              qc.invalidateQueries({ queryKey: ["site", id] });
              qc.invalidateQueries({ queryKey: ["audit"] });
              qc.invalidateQueries({ queryKey: ["site-logs", id] });
            }}
          />
          {canWrite && (
            <>
              <Button
                variant="outline"
                size="sm"
                disabled={deploySite.isPending}
                onClick={handleDeploy}
              >
                <Rocket className="mr-2 h-4 w-4" />
                Deploy
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setEditOpen(true)}
              >
                <Pencil className="mr-2 h-4 w-4" />
                Edit
              </Button>
            </>
          )}
          {isAdmin && (
            <Button
              variant="outline"
              size="sm"
              className="text-destructive hover:text-destructive"
              onClick={() => setDeleteOpen(true)}
            >
              <Trash2 className="mr-2 h-4 w-4" />
              Delete
            </Button>
          )}
        </div>
      </div>

      {/* Info cards */}
      {site && (
        <div className="grid gap-4 sm:grid-cols-4">
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm text-muted-foreground">
                Runtime
              </CardTitle>
            </CardHeader>
            <CardContent className="text-lg font-semibold capitalize">
              {site.Runtime || "—"}
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm text-muted-foreground">
                Domain
              </CardTitle>
            </CardHeader>
            <CardContent className="font-mono text-lg font-semibold">
              {site.Domain}
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm text-muted-foreground">
                Repository
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-1">
              <div className="truncate text-sm font-medium">
                {site.RepositoryURL || <span className="text-muted-foreground">—</span>}
              </div>
              {site.Branch && (
                <div className="flex items-center gap-1 text-xs text-muted-foreground">
                  <span>branch:</span>
                  <code className="font-mono bg-muted px-1 rounded">{site.Branch}</code>
                </div>
              )}
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm text-muted-foreground">
                Created
              </CardTitle>
            </CardHeader>
            <CardContent className="text-lg font-semibold">
              {site.CreatedAt
                ? new Date(site.CreatedAt).toLocaleDateString()
                : "—"}
            </CardContent>
          </Card>
        </div>
      )}

      {/* SSL Status */}
      {sslInfo && (
        <Card className={cn(
          "border-l-4",
          sslInfo.status === "valid" && "border-l-green-500",
          sslInfo.status === "expiring_soon" && "border-l-yellow-500",
          sslInfo.status === "expired" && "border-l-red-500",
          sslInfo.status === "no_cert" && "border-l-gray-300",
        )}>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="flex items-center gap-2 text-sm font-medium">
              {sslInfo.status === "valid" && <ShieldCheck className="h-4 w-4 text-green-600" />}
              {sslInfo.status === "expiring_soon" && <ShieldAlert className="h-4 w-4 text-yellow-600" />}
              {sslInfo.status === "expired" && <ShieldX className="h-4 w-4 text-red-600" />}
              {sslInfo.status === "no_cert" && <ShieldX className="h-4 w-4 text-muted-foreground" />}
              SSL Certificate
            </CardTitle>
            {canWrite && sslInfo.status !== "no_cert" && (
              <Button
                variant="outline"
                size="sm"
                disabled={renewSSL.isPending}
                onClick={() => renewSSL.mutate()}
              >
                <RefreshCw className={cn("mr-1.5 h-3.5 w-3.5", renewSSL.isPending && "animate-spin")} />
                Renew
              </Button>
            )}
          </CardHeader>
          <CardContent className="flex items-center gap-6 text-sm">
            <div>
              <p className="text-xs text-muted-foreground">Status</p>
              <p className={cn(
                "font-semibold capitalize",
                sslInfo.status === "valid" && "text-green-700",
                sslInfo.status === "expiring_soon" && "text-yellow-700",
                sslInfo.status === "expired" && "text-red-700",
              )}>
                {sslInfo.status.replace("_", " ")}
              </p>
            </div>
            {sslInfo.expires_at && sslInfo.status !== "no_cert" && (
              <>
                <div>
                  <p className="text-xs text-muted-foreground">Expires</p>
                  <p className="font-medium">
                    {format(new Date(sslInfo.expires_at), "dd MMM yyyy")}
                  </p>
                </div>
                <div>
                  <p className="text-xs text-muted-foreground">Days left</p>
                  <p className={cn(
                    "font-bold",
                    sslInfo.days_left <= 14 && "text-red-600",
                    sslInfo.days_left > 14 && sslInfo.days_left <= 30 && "text-yellow-600",
                    sslInfo.days_left > 30 && "text-green-700",
                  )}>
                    {sslInfo.days_left}
                  </p>
                </div>
              </>
            )}
            {sslInfo.status === "no_cert" && (
              <p className="text-muted-foreground">No certificate found on this server.</p>
            )}
          </CardContent>
        </Card>
      )}

      {/* ENV Variables — only for git-deployed sites */}
      {hasRepo && (
        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="flex items-center gap-2 text-sm font-medium">
              <KeyRound className="h-4 w-4 text-muted-foreground" />
              Environment Variables
            </CardTitle>
            {canWrite && (
              <Button variant="outline" size="sm" onClick={() => setAddingEnv((v) => !v)}>
                <Plus className="mr-1.5 h-3.5 w-3.5" />
                Add
              </Button>
            )}
          </CardHeader>
          <CardContent className="space-y-3">
            {canWrite && addingEnv && (
              <form onSubmit={handleUpsertEnv} className="flex items-end gap-2 pb-2 border-b">
                <div className="flex-1 space-y-1">
                  <label className="text-xs text-muted-foreground">Key</label>
                  <input
                    className="flex h-8 w-full rounded-md border border-input bg-background px-3 py-1 text-sm font-mono focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
                    placeholder="DATABASE_URL"
                    value={newKey}
                    onChange={(e) => setNewKey(e.target.value.toUpperCase())}
                    pattern="[A-Z_][A-Z0-9_]*"
                    required
                  />
                </div>
                <div className="flex-[2] space-y-1">
                  <label className="text-xs text-muted-foreground">Value</label>
                  <input
                    className="flex h-8 w-full rounded-md border border-input bg-background px-3 py-1 text-sm font-mono focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
                    placeholder="postgres://..."
                    value={newValue}
                    onChange={(e) => setNewValue(e.target.value)}
                  />
                </div>
                <Button type="submit" size="sm">Save</Button>
                <Button type="button" variant="ghost" size="sm" onClick={() => setAddingEnv(false)}>
                  <X className="h-4 w-4" />
                </Button>
              </form>
            )}

            {envVars.length === 0 ? (
              <p className="text-sm text-muted-foreground">No environment variables set.</p>
            ) : (
              <div className="divide-y text-sm">
                {envVars.map((v) => (
                  <div key={v.key} className="flex items-center gap-3 py-2">
                    <code className="font-mono font-semibold w-40 shrink-0 truncate">{v.key}</code>
                    <code className="font-mono text-muted-foreground flex-1 truncate">
                      {visibleValues.has(v.key) ? v.value : "••••••••"}
                    </code>
                    <button
                      type="button"
                      onClick={() => toggleVisible(v.key)}
                      className="text-muted-foreground hover:text-foreground"
                      title={visibleValues.has(v.key) ? "Hide" : "Reveal"}
                    >
                      {visibleValues.has(v.key)
                        ? <EyeOff className="h-3.5 w-3.5" />
                        : <Eye className="h-3.5 w-3.5" />}
                    </button>
                    {canWrite && (
                      <button
                        type="button"
                        onClick={() => handleDeleteEnv(v.key)}
                        className="text-muted-foreground hover:text-destructive"
                        title="Delete"
                      >
                        <X className="h-3.5 w-3.5" />
                      </button>
                    )}
                  </div>
                ))}
              </div>
            )}
            {envVars.length > 0 && (
              <p className="text-xs text-muted-foreground pt-1">
                Changes take effect on next Deploy or Restart.
              </p>
            )}
          </CardContent>
        </Card>
      )}

      {/* Webhook Deploy — only for git-deployed sites */}
      {hasRepo && (
        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="flex items-center gap-2 text-sm font-medium">
              <Webhook className="h-4 w-4 text-muted-foreground" />
              Webhook Deploy
            </CardTitle>
            {canWrite && (
              <Button
                variant="outline"
                size="sm"
                disabled={regenerating}
                onClick={handleRegenerate}
              >
                <RefreshCw className={cn("mr-1.5 h-3.5 w-3.5", regenerating && "animate-spin")} />
                Regenerate
              </Button>
            )}
          </CardHeader>
          <CardContent className="space-y-3">
            {currentToken ? (
              <>
                <div className="flex items-center gap-2">
                  <code className="flex-1 rounded-md bg-muted px-3 py-2 text-xs font-mono break-all">
                    {webhookURL}
                  </code>
                  <Button variant="outline" size="sm" onClick={handleCopyWebhook}>
                    {copied
                      ? <Check className="h-3.5 w-3.5 text-green-600" />
                      : <Copy className="h-3.5 w-3.5" />}
                  </Button>
                </div>
                <div className="rounded-md border bg-muted/40 p-3 text-xs text-muted-foreground space-y-1.5">
                  <p className="font-medium text-foreground">GitHub setup:</p>
                  <p>1. Repository → Settings → Webhooks → Add webhook</p>
                  <p>2. Payload URL: paste the URL above</p>
                  <p>3. Content type: <code className="bg-muted px-1 rounded">application/json</code></p>
                  <p>4. Events: <strong>Just the push event</strong></p>
                  <p>5. Save — every <code className="bg-muted px-1 rounded">git push</code> triggers auto-deploy</p>
                </div>
              </>
            ) : (
              <p className="text-sm text-muted-foreground">
                No webhook token yet.{" "}
                {canWrite && (
                  <button
                    className="underline hover:text-foreground"
                    onClick={handleRegenerate}
                  >
                    Generate one
                  </button>
                )}
              </p>
            )}
          </CardContent>
        </Card>
      )}

      {/* Container status — only for git-deployed sites */}
      {hasRepo && (
        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="flex items-center gap-2 text-sm font-medium">
              <Container className="h-4 w-4 text-muted-foreground" />
              Docker Container
            </CardTitle>
            <div className="flex items-center gap-2">
              <Button
                variant="ghost"
                size="sm"
                className="text-xs text-muted-foreground"
                onClick={() => setShowContainerLogs((v) => !v)}
              >
                <Terminal className="mr-1.5 h-3.5 w-3.5" />
                {showContainerLogs ? "Hide Logs" : "Show Logs"}
              </Button>
              {canWrite && (
                <Button
                  variant="outline"
                  size="sm"
                  disabled={restarting || containerInfo?.status !== "running"}
                  onClick={handleRestart}
                >
                  <RotateCcw className={cn("mr-1.5 h-3.5 w-3.5", restarting && "animate-spin")} />
                  Restart
                </Button>
              )}
            </div>
          </CardHeader>
          <CardContent className="space-y-3">
            {!containerInfo ? (
              <p className="text-sm text-muted-foreground">Loading…</p>
            ) : containerInfo.status === "no_container" ? (
              <p className="text-sm text-muted-foreground">Static site — no container.</p>
            ) : (
              <div className="flex flex-wrap items-center gap-6 text-sm">
                <div>
                  <p className="text-xs text-muted-foreground">Status</p>
                  <span className={cn(
                    "inline-block font-semibold capitalize",
                    containerInfo.status === "running" && "text-green-700",
                    containerInfo.status === "exited" && "text-red-600",
                    containerInfo.status === "not_found" && "text-muted-foreground",
                  )}>
                    {containerInfo.status === "not_found" ? "not found" : containerInfo.status}
                  </span>
                </div>
                {containerInfo.cpu_percent && (
                  <div>
                    <p className="text-xs text-muted-foreground">CPU</p>
                    <p className="font-medium">{containerInfo.cpu_percent}</p>
                  </div>
                )}
                {containerInfo.mem_usage && (
                  <div>
                    <p className="text-xs text-muted-foreground">Memory</p>
                    <p className="font-medium">{containerInfo.mem_usage}</p>
                  </div>
                )}
                {containerInfo.started_at && containerInfo.status === "running" && (
                  <div>
                    <p className="text-xs text-muted-foreground">Uptime</p>
                    <p className="font-medium">
                      {formatDistanceToNow(new Date(containerInfo.started_at), { addSuffix: false })}
                    </p>
                  </div>
                )}
              </div>
            )}

            {showContainerLogs && (
              <div className="mt-2">
                <div className="flex items-center justify-between mb-1">
                  <p className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
                    Container Logs
                  </p>
                  {containerLogsFetching && (
                    <RefreshCw className="h-3 w-3 animate-spin text-muted-foreground" />
                  )}
                </div>
                <pre className="rounded-md bg-muted/60 border p-3 text-xs font-mono overflow-x-auto max-h-80 overflow-y-auto whitespace-pre-wrap">
                  {containerLogs || "(no output)"}
                </pre>
              </div>
            )}
          </CardContent>
        </Card>
      )}

      {/* Deploy Logs */}
      <div>
        <div className="mb-3 flex items-center justify-between">
          <h3 className="text-lg font-semibold">Deploy Logs</h3>
          <span className="text-xs text-muted-foreground">
            {logsFetching ? "Refreshing…" : `${logs.length} runs`}
          </span>
        </div>

        {logs.length === 0 ? (
          <p className="text-sm text-muted-foreground">No deploy runs yet.</p>
        ) : (
          <div className="space-y-2">
            {logs.map((log: TaskLog) => {
              const isExpanded = expandedLog === log.ID;
              const statusColor =
                log.Status === "success"
                  ? "text-green-700 bg-green-50 border-green-200"
                  : log.Status === "failed"
                    ? "text-red-700 bg-red-50 border-red-200"
                    : "text-blue-700 bg-blue-50 border-blue-200";
              return (
                <div key={log.ID} className={cn("rounded-md border", statusColor)}>
                  <button
                    className="flex w-full items-center gap-3 px-3 py-2 text-left text-sm"
                    onClick={() => setExpandedLog(isExpanded ? null : log.ID)}
                  >
                    {isExpanded ? (
                      <ChevronDown className="h-3.5 w-3.5 shrink-0" />
                    ) : (
                      <ChevronRight className="h-3.5 w-3.5 shrink-0" />
                    )}
                    <span className="font-medium capitalize">{log.Status}</span>
                    <span className="font-mono text-xs opacity-60">
                      {log.ID.slice(0, 8)}
                    </span>
                    <span className="ml-auto text-xs opacity-70">
                      {formatDistanceToNow(new Date(log.StartedAt), {
                        addSuffix: true,
                      })}
                      {log.FinishedAt && (
                        <span className="ml-1 opacity-70">
                          ·{" "}
                          {Math.round(
                            (new Date(log.FinishedAt).getTime() -
                              new Date(log.StartedAt).getTime()) /
                              1000,
                          )}
                          s
                        </span>
                      )}
                    </span>
                  </button>
                  {isExpanded && log.Output && (
                    <pre className="overflow-x-auto border-t bg-black/5 px-4 py-3 font-mono text-xs leading-relaxed whitespace-pre-wrap">
                      {log.Output}
                    </pre>
                  )}
                  {isExpanded && !log.Output && (
                    <p className="border-t px-4 py-2 text-xs opacity-60">
                      No output captured.
                    </p>
                  )}
                </div>
              );
            })}
          </div>
        )}
      </div>

      {/* Audit history */}
      <div>
        <div className="mb-3 flex items-center justify-between">
          <h3 className="text-lg font-semibold">Event History</h3>
          <span className="text-xs text-muted-foreground">
            {events.length} events
          </span>
        </div>

        {auditLoading && (
          <p className="text-sm text-muted-foreground">Loading events…</p>
        )}

        {!auditLoading && events.length === 0 && (
          <p className="text-sm text-muted-foreground">No events yet.</p>
        )}

        {events.length > 0 && (
          <div className="relative ml-3 space-y-0">
            {events.map((e, i) => (
              <div key={e.ID} className="flex gap-4">
                {/* Timeline line */}
                <div className="flex flex-col items-center">
                  <div className="mt-1.5 h-2.5 w-2.5 rounded-full border-2 border-primary bg-background" />
                  {i < events.length - 1 && (
                    <div className="w-px flex-1 bg-border" />
                  )}
                </div>

                <div className="pb-5">
                  <div className="flex items-center gap-2">
                    <span
                      className={cn(
                        "rounded px-2 py-0.5 text-xs font-medium",
                        statusClass(e.ToStatus),
                      )}
                    >
                      {e.ToStatus}
                    </span>
                    {e.FromStatus && (
                      <span className="text-xs text-muted-foreground">
                        ← {e.FromStatus}
                      </span>
                    )}
                    <span className="text-xs text-muted-foreground">
                      · {e.Reason}
                    </span>
                  </div>
                  <div className="mt-1 flex items-center gap-1 text-xs text-muted-foreground">
                    <Clock className="h-3 w-3" />
                    {formatDate(e.CreatedAt)}
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}

        {hasNextPage && (
          <Button
            variant="outline"
            size="sm"
            disabled={isFetchingNextPage}
            onClick={() => fetchNextPage()}
          >
            {isFetchingNextPage ? "Loading…" : "Load older events"}
          </Button>
        )}
      </div>

      {/* Modals */}
      <SiteForm
        open={editOpen}
        onOpenChange={setEditOpen}
        site={site}
      />
      <ConfirmDialog
        open={deleteOpen}
        title={`Delete "${site?.Name}"?`}
        description="This site will be permanently removed. This cannot be undone."
        loading={deleteSite.isPending}
        onConfirm={handleDelete}
        onCancel={() => setDeleteOpen(false)}
      />
    </div>
  );
}
