"use client";

import { useState, Suspense } from "react";
import dynamic from "next/dynamic";
import { useQuery } from "@tanstack/react-query";
import { TerminalSquare, ServerIcon, RefreshCw, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { fetchServers, type Server } from "@/lib/api";
import { cn } from "@/lib/utils";

// xterm.js uses DOM APIs — must be loaded client-side only.
const TerminalClient = dynamic(() => import("@/components/terminal-client"), {
  ssr: false,
  loading: () => (
    <div className="flex h-full items-center justify-center bg-[#0d1117]">
      <Loader2 className="h-6 w-6 animate-spin text-white/40" />
    </div>
  ),
});

export default function TerminalPage() {
  const [serverId, setServerId] = useState<string>("");
  const [key, setKey]           = useState(0); // bump to force reconnect

  const { data: servers, isLoading } = useQuery({
    queryKey: ["servers"],
    queryFn: fetchServers,
    staleTime: 30_000,
  });

  const selectedServer = servers?.find((s: Server) => s.ID === serverId);
  const connected = !!serverId;

  return (
    <div className="flex h-[calc(100vh-4rem)] flex-col gap-0">

      {/* ── Toolbar ── */}
      <div className="flex items-center gap-3 border-b bg-background px-4 py-2 shrink-0">
        <TerminalSquare className="h-5 w-5 text-muted-foreground" />
        <h2 className="font-semibold text-sm">Web Terminal</h2>

        <div className="ml-2 h-4 w-px bg-border" />

        {/* Server selector */}
        <div className="relative flex items-center">
          <ServerIcon className="pointer-events-none absolute left-2.5 h-3.5 w-3.5 text-muted-foreground" />
          <select
            value={serverId}
            onChange={(e) => { setServerId(e.target.value); setKey((k) => k + 1); }}
            className="h-8 rounded-md border border-input bg-background pl-8 pr-8 text-xs font-medium shadow-sm focus:outline-none focus:ring-1 focus:ring-ring appearance-none cursor-pointer hover:bg-accent transition-colors min-w-[200px]"
            disabled={isLoading}
          >
            <option value="">— Select a server —</option>
            {servers?.map((srv: Server) => (
              <option key={srv.ID} value={srv.ID}>
                {srv.Name}  ({srv.Host})
              </option>
            ))}
          </select>
          {isLoading && (
            <Loader2 className="pointer-events-none absolute right-2.5 h-3.5 w-3.5 animate-spin text-muted-foreground" />
          )}
        </div>

        {/* Status badge */}
        {connected && (
          <span className="flex items-center gap-1.5 text-xs text-green-500">
            <span className="h-1.5 w-1.5 rounded-full bg-green-500 animate-pulse" />
            {selectedServer?.Host}
          </span>
        )}

        <div className="flex-1" />

        {/* Reconnect */}
        {connected && (
          <Button
            size="sm"
            variant="ghost"
            onClick={() => setKey((k) => k + 1)}
            title="Reconnect"
          >
            <RefreshCw className="h-3.5 w-3.5" />
          </Button>
        )}
      </div>

      {/* ── Terminal area ── */}
      <div className={cn("flex-1 bg-[#0d1117] overflow-hidden", !connected && "flex items-center justify-center")}>
        {!connected ? (
          <div className="flex flex-col items-center gap-4 text-center">
            <div className="flex h-16 w-16 items-center justify-center rounded-2xl bg-white/5">
              <TerminalSquare className="h-8 w-8 text-white/30" />
            </div>
            <div>
              <p className="text-white/60 font-medium">Select a server to open a terminal</p>
              <p className="text-white/30 text-sm mt-1">SSH connection is established in the browser</p>
            </div>
          </div>
        ) : (
          <Suspense fallback={null}>
            <TerminalClient
              key={`${serverId}-${key}`}
              serverId={serverId}
              serverName={selectedServer?.Name}
            />
          </Suspense>
        )}
      </div>
    </div>
  );
}
