"use client";

import {
  useState, useEffect, useRef, useCallback,
} from "react";
import { useQuery } from "@tanstack/react-query";
import {
  ScrollText, Play, Square, Trash2, Download,
  RefreshCw, AlertCircle, ChevronDown,
} from "lucide-react";
import { fetchServers, fetchLogUnits, fetchLogContainers, getToken } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

// ─── Types ────────────────────────────────────────────────────────────────────

type Source = "journal" | "docker" | "file";

interface LogLine {
  id: number;
  text: string;
  ts: number;
}

// ─── ANSI stripping (simple approach — keeps text readable) ───────────────────

function stripAnsi(str: string): string {
  // eslint-disable-next-line no-control-regex
  return str.replace(/\x1B\[[0-9;]*[A-Za-z]/g, "");
}

// ─── Component ────────────────────────────────────────────────────────────────

export default function LogsPage() {
  const [serverId, setServerId]     = useState("");
  const [source, setSource]         = useState<Source>("file");
  const [unit, setUnit]             = useState("_all");
  const [container, setContainer]   = useState("");
  const [filePath, setFilePath]     = useState("/var/log/syslog");
  const [lines, setLines]           = useState("200");
  const [streaming, setStreaming]   = useState(false);
  const [logLines, setLogLines]     = useState<LogLine[]>([]);
  const [error, setError]           = useState("");
  const [autoScroll, setAutoScroll] = useState(true);
  const [filter, setFilter]         = useState("");

  const esRef      = useRef<EventSource | null>(null);
  const bottomRef  = useRef<HTMLDivElement>(null);
  const lineIdRef  = useRef(0);
  const maxLines   = 2000; // keep last N lines in memory

  // ─── Server & resource lists ────────────────────────────────────────────────

  const { data: servers } = useQuery({ queryKey: ["servers"], queryFn: fetchServers, staleTime: 30_000 });

  const { data: units = [] } = useQuery({
    queryKey: ["log-units", serverId],
    queryFn: () => fetchLogUnits(serverId),
    enabled: !!serverId && source === "journal",
  });

  const { data: containers = [] } = useQuery({
    queryKey: ["log-containers", serverId],
    queryFn: () => fetchLogContainers(serverId),
    enabled: !!serverId && source === "docker",
  });

  // Don't auto-select a unit — leave "_all" (all services) as default.
  useEffect(() => { if (containers.length && !container) setContainer(containers[0]); }, [containers, container]);

  // ─── Auto-scroll ────────────────────────────────────────────────────────────

  useEffect(() => {
    if (autoScroll) bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [logLines, autoScroll]);

  // ─── Streaming ──────────────────────────────────────────────────────────────

  const stopStream = useCallback(() => {
    if (retryTimer.current) { clearTimeout(retryTimer.current); retryTimer.current = null; }
    esRef.current?.close();
    esRef.current = null;
    retryRef.current = 0;
    setStreaming(false);
  }, []);

  const retryRef   = useRef(0);
  const retryTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const MAX_RETRIES = 5;

  const openEs = useCallback((sid: string) => {
    esRef.current?.close();
    esRef.current = null;

    const token = getToken() ?? "";
    const params = new URLSearchParams({ source, lines });
    if (source === "journal") params.set("unit", unit || "_all");
    if (source === "docker")  params.set("container", container);
    if (source === "file")    params.set("path", filePath);

    // SSE goes via Nginx → api:8080 with proxy_buffering off.
    // Use current window origin so port 8080 does not need to be public.
    function getApiBase(): string {
      const ws = process.env.NEXT_PUBLIC_API_WS_URL?.trim();
      if (ws) return ws.replace(/^ws(s?):\/\//, "http$1://").replace(/\/$/, "");
      const base = process.env.NEXT_PUBLIC_API_BASE_URL?.trim();
      if (base) return base.replace(/\/$/, "");
      return window.location.origin;
    }
    const url = `${getApiBase()}/api/v1/servers/${sid}/logs/stream?${params}&token=${encodeURIComponent(token)}`;

    const es = new EventSource(url);
    esRef.current = es;

    es.addEventListener("log", (e) => {
      const text = stripAnsi((e as MessageEvent).data);
      setLogLines((prev) => {
        const next = [...prev, { id: lineIdRef.current++, text, ts: Date.now() }];
        return next.length > maxLines ? next.slice(next.length - maxLines) : next;
      });
      retryRef.current = 0; // reset on successful data
    });

    // Only handle *server-sent* named "error" events (they carry a data field).
    // Transport-level errors (connection drop, proxy timeout) have no data and
    // are handled exclusively by es.onerror below.
    es.addEventListener("error", (e) => {
      if (!(e instanceof MessageEvent) || !e.data) return;
      setError(e.data as string);
      es.close();
      setStreaming(false);
    });

    es.onerror = () => {
      es.close();
      esRef.current = null;
      if (retryRef.current < MAX_RETRIES) {
        retryRef.current += 1;
        const delay = Math.min(2 ** retryRef.current * 1000, 30_000);
        retryTimer.current = setTimeout(() => openEs(sid), delay);
      } else {
        setError(`Connection lost after ${MAX_RETRIES} reconnect attempts.`);
        setStreaming(false);
      }
    };
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [source, lines, unit, container, filePath]);

  const startStream = useCallback(async () => {
    if (!serverId) return;
    if (retryTimer.current) { clearTimeout(retryTimer.current); retryTimer.current = null; }
    esRef.current?.close();
    retryRef.current = 0;
    setLogLines([]);
    setError("");
    setStreaming(true);

    // Preflight: verify endpoint is reachable and get a clear error if not.
    const token = getToken() ?? "";
    const params = new URLSearchParams({ source, lines });
    if (source === "journal") params.set("unit", unit || "_all");
    if (source === "docker")  params.set("container", container);
    if (source === "file")    params.set("path", filePath);
    const ws = process.env.NEXT_PUBLIC_API_WS_URL?.trim();
    const base = process.env.NEXT_PUBLIC_API_BASE_URL?.trim();
    const apiBase = ws
      ? ws.replace(/^ws(s?):\/\//, "http$1://").replace(/\/$/, "")
      : base
        ? base.replace(/\/$/, "")
        : `${window.location.protocol}//${window.location.hostname}:8080`;
    const preflightUrl = `${apiBase}/api/v1/servers/${serverId}/logs/stream?${params}&token=${encodeURIComponent(token)}`;
    try {
      const r = await fetch(preflightUrl, {
        headers: token ? { Authorization: `Bearer ${token}` } : {},
        signal: AbortSignal.timeout(8000),
      });
      if (!r.ok) {
        const body = await r.json().catch(() => ({ error: `HTTP ${r.status}` }));
        setError(body?.error ?? `HTTP ${r.status}`);
        setStreaming(false);
        return;
      }
    } catch { /* network error — let EventSource handle & retry */ }

    openEs(serverId);
  }, [serverId, source, lines, unit, container, filePath, openEs]);

  // Stop stream on unmount.
  useEffect(() => () => stopStream(), [stopStream]);

  // ─── Download ───────────────────────────────────────────────────────────────

  function downloadLogs() {
    const text = logLines.map((l) => l.text).join("\n");
    const blob = new Blob([text], { type: "text/plain" });
    const a    = document.createElement("a");
    a.href = URL.createObjectURL(blob);
    a.download = `logs-${serverId}-${source}-${Date.now()}.txt`;
    a.click();
    URL.revokeObjectURL(a.href);
  }

  // ─── Filtered view ──────────────────────────────────────────────────────────

  const visible = filter
    ? logLines.filter((l) => l.text.toLowerCase().includes(filter.toLowerCase()))
    : logLines;

  // ─── Render ─────────────────────────────────────────────────────────────────

  return (
    <div className="flex h-[calc(100vh-4rem)] flex-col gap-4">
      <div className="flex items-center gap-2">
        <ScrollText className="h-5 w-5 text-muted-foreground" />
        <h1 className="text-xl font-semibold">Log Viewer</h1>
        {streaming && (
          <span className="flex items-center gap-1.5 rounded-full bg-emerald-500/10 px-2 py-0.5 text-xs font-medium text-emerald-500">
            <span className="h-1.5 w-1.5 rounded-full bg-emerald-500 animate-pulse" />
            Live
          </span>
        )}
      </div>

      {/* ─── Controls ─────────────────────────────────────────────────────── */}
      <div className="flex flex-wrap items-end gap-3 rounded-xl border bg-card p-4">

        {/* Server */}
        <div className="flex flex-col gap-1.5">
          <label className="text-xs font-medium text-muted-foreground">Server</label>
          <select
            value={serverId}
            onChange={(e) => { setServerId(e.target.value); stopStream(); }}
            className="h-9 rounded-md border bg-background px-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring min-w-[160px]"
          >
            <option value="">Select server…</option>
            {(servers ?? []).map((s) => (
              <option key={s.ID} value={s.ID}>{s.Name}</option>
            ))}
          </select>
        </div>

        {/* Source */}
        <div className="flex flex-col gap-1.5">
          <label className="text-xs font-medium text-muted-foreground">Source</label>
          <div className="flex rounded-md border overflow-hidden h-9">
            {(["journal", "docker", "file"] as Source[]).map((s) => (
              <button
                key={s}
                onClick={() => { setSource(s); stopStream(); }}
                className={cn(
                  "px-3 text-sm font-medium transition-colors",
                  source === s
                    ? "bg-primary text-primary-foreground"
                    : "bg-background text-muted-foreground hover:bg-accent",
                )}
              >
                {s === "journal" ? "Journal" : s === "docker" ? "Docker" : "File"}
              </button>
            ))}
          </div>
        </div>

        {/* Unit / container / path */}
        {source === "journal" && (
          <div className="flex flex-col gap-1.5">
            <label className="text-xs font-medium text-muted-foreground">Unit</label>
            <select
              value={unit}
              onChange={(e) => setUnit(e.target.value)}
              className="h-9 rounded-md border bg-background px-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring min-w-[160px]"
            >
              <option value="_all">All services</option>
              {units.map((u) => <option key={u} value={u}>{u}</option>)}
            </select>
          </div>
        )}
        {source === "docker" && (
          <div className="flex flex-col gap-1.5">
            <label className="text-xs font-medium text-muted-foreground">Container</label>
            <select
              value={container}
              onChange={(e) => setContainer(e.target.value)}
              className="h-9 rounded-md border bg-background px-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring min-w-[160px]"
            >
              {containers.length === 0 && <option value="">No containers</option>}
              {containers.map((c) => <option key={c} value={c}>{c}</option>)}
            </select>
          </div>
        )}
        {source === "file" && (
          <div className="flex flex-col gap-1.5">
            <label className="text-xs font-medium text-muted-foreground">File path</label>
            <input
              value={filePath}
              onChange={(e) => setFilePath(e.target.value)}
              placeholder="/var/log/syslog"
              className="h-9 rounded-md border bg-background px-3 text-sm focus:outline-none focus:ring-2 focus:ring-ring w-52"
            />
          </div>
        )}

        {/* Tail lines */}
        <div className="flex flex-col gap-1.5">
          <label className="text-xs font-medium text-muted-foreground">Tail lines</label>
          <select
            value={lines}
            onChange={(e) => setLines(e.target.value)}
            className="h-9 rounded-md border bg-background px-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
          >
            {["50", "100", "200", "500", "1000"].map((n) => (
              <option key={n} value={n}>{n}</option>
            ))}
          </select>
        </div>

        {/* Actions */}
        <div className="flex items-end gap-2 ml-auto">
          {streaming ? (
            <Button variant="destructive" size="sm" onClick={stopStream} className="gap-1.5">
              <Square className="h-3.5 w-3.5" /> Stop
            </Button>
          ) : (
            <Button size="sm" onClick={startStream} disabled={!serverId} className="gap-1.5">
              <Play className="h-3.5 w-3.5" /> Start
            </Button>
          )}
          <Button variant="outline" size="sm" onClick={startStream} disabled={!serverId} title="Restart stream">
            <RefreshCw className="h-3.5 w-3.5" />
          </Button>
          <Button variant="outline" size="sm" onClick={() => setLogLines([])} disabled={logLines.length === 0} title="Clear">
            <Trash2 className="h-3.5 w-3.5" />
          </Button>
          <Button variant="outline" size="sm" onClick={downloadLogs} disabled={logLines.length === 0} title="Download">
            <Download className="h-3.5 w-3.5" />
          </Button>
        </div>
      </div>

      {/* ─── Filter ───────────────────────────────────────────────────────── */}
      <div className="flex items-center gap-3">
        <input
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
          placeholder="Filter logs…"
          className="h-8 flex-1 max-w-sm rounded-md border bg-background px-3 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
        />
        <span className="text-xs text-muted-foreground">
          {filter ? `${visible.length} / ${logLines.length} lines` : `${logLines.length} lines`}
        </span>
        <button
          onClick={() => setAutoScroll((v) => !v)}
          className={cn(
            "flex items-center gap-1.5 rounded-md border px-2.5 py-1 text-xs font-medium transition-colors",
            autoScroll ? "border-primary/30 bg-primary/10 text-primary" : "border-border text-muted-foreground hover:bg-accent",
          )}
        >
          <ChevronDown className="h-3.5 w-3.5" />
          Auto-scroll
        </button>
      </div>

      {/* ─── Log terminal ─────────────────────────────────────────────────── */}
      {!serverId ? (
        <div className="flex flex-1 items-center justify-center rounded-xl border border-dashed text-muted-foreground text-sm">
          Select a server to start viewing logs
        </div>
      ) : (
        <div className="flex-1 overflow-y-auto rounded-xl border bg-[#0d1117] font-mono text-xs text-[#c9d1d9] p-4 relative">
          {error && (
            <div className="mb-3 flex items-center gap-2 rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-red-400 text-xs">
              <AlertCircle className="h-3.5 w-3.5 shrink-0" />
              {error}
            </div>
          )}

          {visible.length === 0 && !streaming && !error && (
            <span className="text-[#484f58]">Press Start to begin streaming…</span>
          )}
          {visible.length === 0 && streaming && !error && (
            <span className="text-[#484f58] animate-pulse">Waiting for log entries…</span>
          )}

          {visible.map((line) => (
            <LogRow key={line.id} text={line.text} filter={filter} />
          ))}

          <div ref={bottomRef} />
        </div>
      )}
    </div>
  );
}

// ─── LogRow: highlights filter match ─────────────────────────────────────────

function LogRow({ text, filter }: { text: string; filter: string }) {
  if (!filter) {
    return <div className="leading-5 whitespace-pre-wrap break-all">{text}</div>;
  }
  const idx = text.toLowerCase().indexOf(filter.toLowerCase());
  if (idx === -1) {
    return <div className="leading-5 whitespace-pre-wrap break-all opacity-40">{text}</div>;
  }
  return (
    <div className="leading-5 whitespace-pre-wrap break-all">
      {text.slice(0, idx)}
      <mark className="bg-yellow-400/30 text-yellow-300 rounded-sm">{text.slice(idx, idx + filter.length)}</mark>
      {text.slice(idx + filter.length)}
    </div>
  );
}
