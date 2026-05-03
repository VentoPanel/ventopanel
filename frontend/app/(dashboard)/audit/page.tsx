"use client";

import { useState, useMemo, useEffect, useCallback } from "react";
import {
  ClipboardList, ChevronDown, Search, RefreshCw,
  Server, Globe, CheckCircle2, XCircle, AlertCircle,
  ArrowRight, Shield, Zap, Activity,
} from "lucide-react";
import { useAuditEvents } from "@/hooks/use-audit";
import { type AuditEvent } from "@/lib/api";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";

// ─── Types ────────────────────────────────────────────────────────────────────
type ResourceFilter = "" | "server" | "site";

// ─── Status helpers ───────────────────────────────────────────────────────────
const SUCCESS_STATUSES = new Set(["deployed", "connected", "provisioned", "ready_for_deploy", "ssl_active", "ssl_renewed"]);
const ERROR_STATUSES   = new Set(["failed", "error", "access_denied", "deploy_failed", "provision_failed", "ssl_failed", "connection_failed"]);

function isSuccess(s: string) { return SUCCESS_STATUSES.has(s.toLowerCase()); }
function isError(s: string)   { return ERROR_STATUSES.has(s.toLowerCase()) || s.toLowerCase().includes("fail") || s.toLowerCase().includes("error"); }

const STATUS_CHIP: Record<string, string> = {
  deployed:           "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400",
  connected:          "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400",
  provisioned:        "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400",
  ready_for_deploy:   "bg-blue-100  text-blue-700  dark:bg-blue-900/30  dark:text-blue-400",
  ssl_active:         "bg-blue-100  text-blue-700  dark:bg-blue-900/30  dark:text-blue-400",
  ssl_renewed:        "bg-blue-100  text-blue-700  dark:bg-blue-900/30  dark:text-blue-400",
  deploying:          "bg-sky-100   text-sky-700   dark:bg-sky-900/30   dark:text-sky-400",
  provisioning:       "bg-sky-100   text-sky-700   dark:bg-sky-900/30   dark:text-sky-400",
  pending:            "bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400",
  access_denied:      "bg-red-100   text-red-700   dark:bg-red-900/30   dark:text-red-400",
  deploy_failed:      "bg-red-100   text-red-700   dark:bg-red-900/30   dark:text-red-400",
  provision_failed:   "bg-red-100   text-red-700   dark:bg-red-900/30   dark:text-red-400",
  ssl_failed:         "bg-red-100   text-red-700   dark:bg-red-900/30   dark:text-red-400",
  connection_failed:  "bg-red-100   text-red-700   dark:bg-red-900/30   dark:text-red-400",
};

function chipClass(s: string) {
  return STATUS_CHIP[s.toLowerCase()] ?? "bg-muted text-muted-foreground";
}

// ─── Time helpers ─────────────────────────────────────────────────────────────
function relativeTime(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime();
  const s = Math.floor(diff / 1000);
  if (s < 60) return `${s}s ago`;
  const m = Math.floor(s / 60);
  if (m < 60) return `${m}m ago`;
  const h = Math.floor(m / 60);
  if (h < 24) return `${h}h ago`;
  return new Date(iso).toLocaleDateString(undefined, { month: "short", day: "numeric" });
}

function fullDate(iso: string): string {
  return new Date(iso).toLocaleString(undefined, { dateStyle: "medium", timeStyle: "medium" });
}

function shortID(id: string): string { return id.slice(0, 8); }

// ─── Event icon ───────────────────────────────────────────────────────────────
function EventIcon({ event }: { event: AuditEvent }) {
  const to = event.ToStatus.toLowerCase();
  if (isError(to))   return <XCircle    className="h-4 w-4 text-destructive" />;
  if (isSuccess(to)) return <CheckCircle2 className="h-4 w-4 text-green-500" />;
  if (to.includes("deploy") || to.includes("provision"))
    return <Zap className="h-4 w-4 text-blue-400" />;
  return <Activity className="h-4 w-4 text-muted-foreground" />;
}

// ─── Resource icon ────────────────────────────────────────────────────────────
function ResourceIcon({ type }: { type: string }) {
  if (type === "server") return <Server className="h-3 w-3" />;
  if (type === "site")   return <Globe  className="h-3 w-3" />;
  return <Shield className="h-3 w-3" />;
}

// ─── Timeline event row ───────────────────────────────────────────────────────
function EventRow({ event }: { event: AuditEvent }) {
  const [expanded, setExpanded] = useState(false);
  const to = event.ToStatus.toLowerCase();
  const lineColor = isError(to)
    ? "border-destructive/30"
    : isSuccess(to)
    ? "border-green-500/30"
    : "border-border";

  return (
    <div
      className={cn(
        "group relative flex gap-4 border-l-2 pl-4 pb-5 last:pb-0 cursor-pointer select-none",
        lineColor,
      )}
      onClick={() => setExpanded(v => !v)}
    >
      {/* dot on the timeline line */}
      <div className="absolute -left-[9px] top-0.5 flex h-4 w-4 items-center justify-center rounded-full border bg-background">
        <EventIcon event={event} />
      </div>

      <div className="flex-1 min-w-0 space-y-1">
        {/* top row */}
        <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-sm">
          {/* resource badge */}
          <span className="flex items-center gap-1 rounded border px-1.5 py-0.5 text-xs text-muted-foreground">
            <ResourceIcon type={event.ResourceType} />
            {event.ResourceType}
          </span>
          {/* resource ID */}
          <code
            className="font-mono text-xs text-muted-foreground"
            title={event.ResourceID}
          >
            {shortID(event.ResourceID)}
          </code>
          {/* transition */}
          <div className="flex items-center gap-1.5">
            {event.FromStatus && (
              <>
                <span className={cn("rounded px-2 py-0.5 text-xs font-medium", chipClass(event.FromStatus))}>
                  {event.FromStatus}
                </span>
                <ArrowRight className="h-3 w-3 text-muted-foreground shrink-0" />
              </>
            )}
            <span className={cn("rounded px-2 py-0.5 text-xs font-medium", chipClass(event.ToStatus))}>
              {event.ToStatus}
            </span>
          </div>
          {/* time */}
          <span className="ml-auto text-xs text-muted-foreground shrink-0" title={fullDate(event.CreatedAt)}>
            {relativeTime(event.CreatedAt)}
          </span>
        </div>

        {/* reason */}
        {event.Reason && (
          <p className="text-xs text-muted-foreground truncate group-hover:text-foreground transition-colors">
            {event.Reason}
          </p>
        )}

        {/* expanded details */}
        {expanded && (
          <div className="mt-2 rounded-md border bg-muted/30 px-3 py-2 text-xs space-y-1 font-mono">
            <div><span className="text-muted-foreground">Event ID: </span>{event.ID}</div>
            <div><span className="text-muted-foreground">Resource: </span>{event.ResourceID}</div>
            {event.TaskID && <div><span className="text-muted-foreground">Task ID:  </span>{event.TaskID}</div>}
            <div><span className="text-muted-foreground">Time:     </span>{fullDate(event.CreatedAt)}</div>
          </div>
        )}
      </div>
    </div>
  );
}

// ─── Stats card ───────────────────────────────────────────────────────────────
function StatCard({
  label, value, icon: Icon, color,
}: {
  label: string;
  value: number;
  icon: React.ElementType;
  color: string;
}) {
  return (
    <div className="flex items-center gap-3 rounded-lg border bg-card px-4 py-3">
      <div className={cn("flex h-8 w-8 items-center justify-center rounded-lg", color)}>
        <Icon className="h-4 w-4" />
      </div>
      <div>
        <div className="text-lg font-bold leading-none">{value}</div>
        <div className="text-xs text-muted-foreground mt-0.5">{label}</div>
      </div>
    </div>
  );
}

// ─── Main page ────────────────────────────────────────────────────────────────
export default function AuditPage() {
  const [resourceType, setResourceType] = useState<ResourceFilter>("");
  const [search, setSearch] = useState("");
  const [tick, setTick] = useState(0); // used to force relative-time re-render

  const {
    data, isFetchingNextPage, hasNextPage, fetchNextPage, isLoading, refetch, isFetching,
  } = useAuditEvents(resourceType ? { resource_type: resourceType } : undefined);

  // auto-refresh every 30s
  useEffect(() => {
    const id = setInterval(() => { refetch(); setTick(t => t + 1); }, 30_000);
    return () => clearInterval(id);
  }, [refetch]);

  const allEvents = data?.pages.flatMap(p => p.items) ?? [];

  // client-side search filter
  const events = useMemo(() => {
    if (!search.trim()) return allEvents;
    const q = search.toLowerCase();
    return allEvents.filter(e =>
      e.ResourceID.toLowerCase().includes(q) ||
      e.Reason?.toLowerCase().includes(q) ||
      e.ToStatus.toLowerCase().includes(q) ||
      e.FromStatus?.toLowerCase().includes(q),
    );
  }, [allEvents, search]);

  // stats from loaded data
  const stats = useMemo(() => {
    const total    = allEvents.length;
    const errors   = allEvents.filter(e => isError(e.ToStatus)).length;
    const successes= allEvents.filter(e => isSuccess(e.ToStatus)).length;
    return { total, errors, successes };
  }, [allEvents]);

  const handleRefresh = useCallback(() => { refetch(); setTick(t => t + 1); }, [refetch]);

  return (
    <div className="space-y-5">
      {/* Header */}
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-primary/10 text-primary">
            <ClipboardList className="h-5 w-5" />
          </div>
          <div>
            <h2 className="text-2xl font-bold tracking-tight">Audit Log</h2>
            <p className="text-sm text-muted-foreground">Status-change events for servers and sites</p>
          </div>
        </div>

        <Button
          variant="outline"
          size="sm"
          onClick={handleRefresh}
          disabled={isFetching}
          className="gap-2"
        >
          <RefreshCw className={cn("h-3.5 w-3.5", isFetching && "animate-spin")} />
          Refresh
        </Button>
      </div>

      {/* Stats */}
      {!isLoading && (
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
          <StatCard label="Events loaded"   value={stats.total}    icon={Activity}      color="bg-muted text-muted-foreground" />
          <StatCard label="Successful"      value={stats.successes} icon={CheckCircle2} color="bg-green-100 text-green-600 dark:bg-green-900/30 dark:text-green-400" />
          <StatCard label="Errors / Denied" value={stats.errors}    icon={XCircle}      color="bg-red-100 text-red-600 dark:bg-red-900/30 dark:text-red-400" />
        </div>
      )}

      {/* Filters */}
      <div className="flex flex-wrap items-center gap-3">
        {/* Resource type pills */}
        <div className="flex rounded-lg border bg-muted/40 p-1 gap-1">
          {([
            { value: "",       label: "All" },
            { value: "server", label: "Servers" },
            { value: "site",   label: "Sites" },
          ] as { value: ResourceFilter; label: string }[]).map(opt => (
            <button
              key={opt.value}
              onClick={() => setResourceType(opt.value)}
              className={cn(
                "rounded-md px-3 py-1 text-sm font-medium transition-all duration-150",
                resourceType === opt.value
                  ? "bg-background text-foreground shadow-sm"
                  : "text-muted-foreground hover:text-foreground",
              )}
            >
              {opt.label}
            </button>
          ))}
        </div>

        {/* Search */}
        <div className="relative flex-1 min-w-[200px] max-w-sm">
          <Search className="absolute left-3 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="Search by status, reason or ID…"
            value={search}
            onChange={e => setSearch(e.target.value)}
            className="pl-9 h-9 text-sm"
          />
        </div>

        {search && (
          <span className="text-xs text-muted-foreground">
            {events.length} / {allEvents.length} events
          </span>
        )}
      </div>

      {/* Timeline */}
      {isLoading ? (
        <div className="space-y-4 border-l-2 border-border pl-4">
          {Array.from({ length: 8 }).map((_, i) => (
            <div key={i} className="h-10 animate-pulse rounded-md bg-muted" />
          ))}
        </div>
      ) : events.length === 0 ? (
        <div className="flex flex-col items-center justify-center gap-3 rounded-xl border border-dashed py-16 text-center">
          <AlertCircle className="h-8 w-8 text-muted-foreground/50" />
          <div>
            <p className="font-medium text-muted-foreground">No events found</p>
            {search && <p className="text-xs text-muted-foreground mt-1">Try clearing the search filter</p>}
          </div>
        </div>
      ) : (
        <div className="rounded-xl border bg-card p-5">
          <div className="space-y-0">
            {events.map(e => (
              <EventRow key={e.ID} event={e} />
            ))}
          </div>
        </div>
      )}

      {/* Load more */}
      {hasNextPage && !search && (
        <div className="flex justify-center">
          <Button
            variant="outline"
            size="sm"
            disabled={isFetchingNextPage}
            onClick={() => fetchNextPage()}
            className="gap-2"
          >
            <ChevronDown className={cn("h-4 w-4", isFetchingNextPage && "animate-bounce")} />
            {isFetchingNextPage ? "Loading…" : "Load more events"}
          </Button>
        </div>
      )}

      {/* Footer note */}
      {!isLoading && (
        <p className="text-center text-xs text-muted-foreground">
          Auto-refreshes every 30 seconds · Click any event to expand details
        </p>
      )}
    </div>
  );
}
