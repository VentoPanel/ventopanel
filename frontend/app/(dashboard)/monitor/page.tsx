"use client";

import { useState, useEffect, useRef, useCallback } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  AreaChart, Area, XAxis, YAxis, Tooltip, ResponsiveContainer, CartesianGrid,
} from "recharts";
import {
  Cpu, MemoryStick, HardDrive, Activity, ServerIcon,
  Wifi, ArrowDown, ArrowUp, Loader2, WifiOff,
} from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { fetchServers, getToken, type Server } from "@/lib/api";

// Derives HTTP base URL from env vars, falling back to the same host on port 8080
// (the Go API port). This mirrors what the terminal does for WebSocket connections,
// and bypasses the Next.js proxy which can buffer SSE responses.
function getApiBaseUrl(): string {
  const ws = process.env.NEXT_PUBLIC_API_WS_URL?.trim();
  if (ws) return ws.replace(/^ws(s?):\/\//, "http$1://").replace(/\/$/, "");
  const base = process.env.NEXT_PUBLIC_API_BASE_URL?.trim();
  if (base) return base.replace(/\/$/, "");
  // Use same host as the browser but port 8080 (direct to Go API, no proxy).
  if (typeof window !== "undefined") {
    const proto = window.location.protocol; // "http:" or "https:"
    return `${proto}//${window.location.hostname}:8080`;
  }
  return "";
}
import { cn } from "@/lib/utils";

// ─── Types ────────────────────────────────────────────────────────────────────

interface Snapshot {
  ts: number;
  cpu_pct: number;
  ram_total_mb: number;
  ram_used_mb: number;
  disk_total: string;
  disk_used: string;
  disk_pct: string;
  load1: number;
  load5: number;
  net_rx_kb: number;
  net_tx_kb: number;
}

interface ChartPoint extends Snapshot {
  time: string;
  ram_pct: number;
}

const MAX_POINTS = 40;

function toPoint(s: Snapshot): ChartPoint {
  return {
    ...s,
    time: new Date(s.ts).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit" }),
    ram_pct: s.ram_total_mb > 0 ? Math.round((s.ram_used_mb / s.ram_total_mb) * 100) : 0,
  };
}

// ─── Stat card ────────────────────────────────────────────────────────────────

function StatCard({
  icon: Icon,
  label,
  value,
  sub,
  color,
  loading,
}: {
  icon: React.ElementType;
  label: string;
  value: string;
  sub?: string;
  color?: string;
  loading?: boolean;
}) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between pb-2">
        <CardTitle className="text-sm font-medium text-muted-foreground">{label}</CardTitle>
        <div className={cn("flex h-8 w-8 items-center justify-center rounded-lg", color ?? "bg-muted")}>
          <Icon className="h-4 w-4" />
        </div>
      </CardHeader>
      <CardContent>
        {loading ? (
          <div className="h-7 w-20 animate-pulse rounded bg-muted" />
        ) : (
          <>
            <div className="text-2xl font-bold">{value}</div>
            {sub && <p className="text-xs text-muted-foreground mt-0.5">{sub}</p>}
          </>
        )}
      </CardContent>
    </Card>
  );
}

// ─── Chart card ───────────────────────────────────────────────────────────────

function ChartCard({
  title,
  dataKey,
  color,
  data,
  unit,
  domain,
}: {
  title: string;
  dataKey: string;
  color: string;
  data: ChartPoint[];
  unit: string;
  domain?: [number, number];
}) {
  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="text-sm font-medium">{title}</CardTitle>
      </CardHeader>
      <CardContent className="pb-2">
        <ResponsiveContainer width="100%" height={140}>
          <AreaChart data={data} margin={{ top: 4, right: 4, bottom: 0, left: -20 }}>
            <defs>
              <linearGradient id={`grad-${dataKey}`} x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%"  stopColor={color} stopOpacity={0.25} />
                <stop offset="95%" stopColor={color} stopOpacity={0} />
              </linearGradient>
            </defs>
            <CartesianGrid strokeDasharray="3 3" stroke="hsl(var(--border))" strokeOpacity={0.5} />
            <XAxis dataKey="time" tick={{ fontSize: 10, fill: "hsl(var(--muted-foreground))" }} tickLine={false} interval="preserveStartEnd" />
            <YAxis domain={domain ?? [0, 100]} tick={{ fontSize: 10, fill: "hsl(var(--muted-foreground))" }} tickLine={false} unit={unit} />
            <Tooltip
              contentStyle={{ background: "hsl(var(--card))", border: "1px solid hsl(var(--border))", borderRadius: 8, fontSize: 12 }}
              labelStyle={{ color: "hsl(var(--foreground))" }}
              formatter={(v: number) => [`${v}${unit}`, dataKey]}
            />
            <Area type="monotone" dataKey={dataKey} stroke={color} strokeWidth={2} fill={`url(#grad-${dataKey})`} dot={false} isAnimationActive={false} />
          </AreaChart>
        </ResponsiveContainer>
      </CardContent>
    </Card>
  );
}

// ─── Main page ────────────────────────────────────────────────────────────────

export default function MonitorPage() {
  const [serverId, setServerId] = useState("");
  const [history, setHistory]   = useState<ChartPoint[]>([]);
  const [latest,  setLatest]    = useState<ChartPoint | null>(null);
  const [status,  setStatus]    = useState<"idle" | "connecting" | "live" | "error">("idle");
  const [errMsg,  setErrMsg]    = useState("");
  const esRef        = useRef<EventSource | null>(null);
  const retryRef     = useRef(0);
  const retryTimer   = useRef<ReturnType<typeof setTimeout> | null>(null);
  const serverIdRef  = useRef("");

  const { data: servers, isLoading: serversLoading } = useQuery({
    queryKey: ["servers"],
    queryFn: fetchServers,
    staleTime: 30_000,
  });

  const MAX_RETRIES = 5;

  const stop = useCallback(() => {
    if (retryTimer.current) { clearTimeout(retryTimer.current); retryTimer.current = null; }
    esRef.current?.close();
    esRef.current = null;
    retryRef.current = 0;
    serverIdRef.current = "";
    setStatus("idle");
  }, []);

  const openStream = useCallback((sid: string) => {
    esRef.current?.close();
    esRef.current = null;

    const token = getToken() ?? "";
    const apiBase = getApiBaseUrl();
    const sseUrl = `${apiBase}/api/v1/servers/${sid}/metrics/stream?token=${encodeURIComponent(token)}`;

    const es = new EventSource(sseUrl);
    esRef.current = es;

    es.addEventListener("metrics", (e) => {
      try {
        const snap: Snapshot = JSON.parse((e as MessageEvent).data);
        const pt = toPoint(snap);
        setLatest(pt);
        setHistory((prev) => {
          const next = [...prev, pt];
          return next.length > MAX_POINTS ? next.slice(-MAX_POINTS) : next;
        });
        setStatus("live");
        retryRef.current = 0; // successful data → reset retry counter
      } catch { /* ignore parse errors */ }
    });

    // Only handle *server-sent* named "error" events (they carry a data field).
    // Transport-level errors are handled exclusively by es.onerror below.
    es.addEventListener("error", (e) => {
      if (!(e instanceof MessageEvent) || !e.data) return;
      setErrMsg(e.data as string);
      setStatus("error");
      es.close();
    });

    // Transport-level error (connection drop, proxy timeout, etc.).
    es.onerror = () => {
      es.close();
      esRef.current = null;

      if (retryRef.current < MAX_RETRIES) {
        retryRef.current += 1;
        // Brief reconnect delay: 2 s → 4 s → 8 s … capped at 30 s.
        const delay = Math.min(2 ** retryRef.current * 1000, 30_000);
        setStatus("connecting");
        retryTimer.current = setTimeout(() => {
          if (serverIdRef.current) openStream(serverIdRef.current);
        }, delay);
      } else {
        setErrMsg("Stream disconnected after " + MAX_RETRIES + " reconnect attempts.");
        setStatus("error");
      }
    };
  // openStream is recreated via useCallback — intentionally not listed to avoid circular dep.
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const start = useCallback(async (sid: string) => {
    if (retryTimer.current) { clearTimeout(retryTimer.current); retryTimer.current = null; }
    esRef.current?.close();
    esRef.current = null;
    retryRef.current = 0;
    serverIdRef.current = sid;

    setHistory([]);
    setLatest(null);
    setErrMsg("");
    setStatus("connecting");

    // Preflight: quick HTTP check to get a readable error if SSH fails immediately.
    const token = getToken() ?? "";
    const apiBase = getApiBaseUrl();
    const sseUrl = `${apiBase}/api/v1/servers/${sid}/metrics/stream?token=${encodeURIComponent(token)}`;
    try {
      const preflight = await fetch(sseUrl, { headers: token ? { Authorization: `Bearer ${token}` } : {} });
      if (!preflight.ok) {
        const body = await preflight.json().catch(() => ({ error: `HTTP ${preflight.status}` }));
        setErrMsg(body?.error ?? `HTTP ${preflight.status}`);
        setStatus("error");
        return;
      }
    } catch { /* network error — let EventSource handle it */ }

    openStream(sid);
  }, [openStream]);

  // Start stream when server changes.
  useEffect(() => {
    if (serverId) start(serverId);
    else stop();
    return stop;
  }, [serverId, start, stop]);

  const loading = status === "connecting" || !latest;
  const selectedServer = servers?.find((s: Server) => s.ID === serverId);

  return (
    <div className="flex flex-col gap-6">

      {/* ── Header ── */}
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-center gap-3">
          <Activity className="h-6 w-6 text-muted-foreground" />
          <div>
            <h2 className="text-2xl font-bold tracking-tight">Resource Monitor</h2>
            <p className="text-xs text-muted-foreground">
              {status === "live" && selectedServer
                ? `Live · ${selectedServer.Name} (${selectedServer.Host}) · updated every 3 s`
                : "Select a server to start monitoring"}
            </p>
          </div>
        </div>

        {/* Server selector */}
        <div className="relative flex items-center">
          <ServerIcon className="pointer-events-none absolute left-2.5 h-3.5 w-3.5 text-muted-foreground" />
          <select
            value={serverId}
            onChange={(e) => setServerId(e.target.value)}
            className="h-8 rounded-md border border-input bg-background pl-8 pr-8 text-xs font-medium shadow-sm focus:outline-none focus:ring-1 focus:ring-ring appearance-none cursor-pointer hover:bg-accent transition-colors min-w-[200px]"
            disabled={serversLoading}
          >
            <option value="">— Select a server —</option>
            {servers?.map((srv: Server) => (
              <option key={srv.ID} value={srv.ID}>{srv.Name} ({srv.Host})</option>
            ))}
          </select>
          {(serversLoading || status === "connecting") && (
            <Loader2 className="pointer-events-none absolute right-2.5 h-3.5 w-3.5 animate-spin text-muted-foreground" />
          )}
        </div>
      </div>

      {/* ── No server selected ── */}
      {!serverId && (
        <div className="flex flex-col items-center justify-center py-24 gap-4 text-center">
          <div className="flex h-16 w-16 items-center justify-center rounded-2xl bg-muted/50">
            <Activity className="h-8 w-8 opacity-30" />
          </div>
          <div>
            <p className="font-medium">No server selected</p>
            <p className="text-sm text-muted-foreground mt-1">Choose a server above to see live resource usage</p>
          </div>
        </div>
      )}

      {/* ── Error ── */}
      {serverId && status === "error" && (
        <div className="flex flex-col gap-1 rounded-lg border border-destructive/30 bg-destructive/10 px-4 py-3 text-sm text-destructive">
          <div className="flex items-center gap-2">
            <WifiOff className="h-4 w-4 shrink-0" />
            <span className="font-medium">Could not connect to metrics stream</span>
          </div>
          {errMsg ? (
            <p className="ml-6 text-xs opacity-80">{errMsg}</p>
          ) : (
            <p className="ml-6 text-xs opacity-80">Check SSH credentials and server availability. Make sure port 22 is open.</p>
          )}
          <button
            onClick={() => start(serverId)}
            className="ml-6 mt-1 w-fit text-xs underline underline-offset-2 hover:opacity-70"
          >
            Retry
          </button>
        </div>
      )}

      {/* ── Stat cards ── */}
      {serverId && status !== "error" && (
        <>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <StatCard icon={Cpu}        label="CPU Usage"     loading={loading} value={latest ? `${latest.cpu_pct.toFixed(1)}%` : "—"}   sub={latest ? `Load ${latest.load1.toFixed(2)} / ${latest.load5.toFixed(2)}` : undefined} color="bg-blue-100 dark:bg-blue-950 text-blue-500" />
            <StatCard icon={MemoryStick} label="RAM Usage"    loading={loading} value={latest ? `${latest.ram_pct}%` : "—"}               sub={latest ? `${latest.ram_used_mb} MB / ${latest.ram_total_mb} MB` : undefined}           color="bg-violet-100 dark:bg-violet-950 text-violet-500" />
            <StatCard icon={HardDrive}  label="Disk (/)"      loading={loading} value={latest?.disk_pct ?? "—"}                           sub={latest ? `${latest.disk_used} used of ${latest.disk_total}` : undefined}               color="bg-amber-100 dark:bg-amber-950 text-amber-500" />
            <StatCard icon={Wifi}       label="Network"       loading={loading}
              value={latest ? `↓${latest.net_rx_kb} ↑${latest.net_tx_kb}` : "—"}
              sub="KB/s (last 3 s)"
              color="bg-green-100 dark:bg-green-950 text-green-500"
            />
          </div>

          {/* ── Charts ── */}
          <div className="grid gap-4 lg:grid-cols-2">
            <ChartCard title="CPU %" dataKey="cpu_pct" color="#3b82f6" data={history} unit="%" domain={[0, 100]} />
            <ChartCard title="RAM %" dataKey="ram_pct" color="#8b5cf6" data={history} unit="%" domain={[0, 100]} />
          </div>

          <div className="grid gap-4 lg:grid-cols-2">
            <ChartCard title="Network RX  KB/s" dataKey="net_rx_kb" color="#10b981" data={history} unit="" />
            <ChartCard title="Network TX  KB/s" dataKey="net_tx_kb" color="#f59e0b" data={history} unit="" />
          </div>

          {/* Status indicator */}
          <div className="flex items-center gap-2 text-xs text-muted-foreground">
            {status === "live"
              ? <><span className="h-1.5 w-1.5 rounded-full bg-green-500 animate-pulse" /> Live — {history.length} data points</>
              : <><Loader2 className="h-3 w-3 animate-spin" /> {retryRef.current > 0 ? `Reconnecting… (attempt ${retryRef.current}/${MAX_RETRIES})` : "Connecting…"}</>
            }
          </div>
        </>
      )}
    </div>
  );
}
