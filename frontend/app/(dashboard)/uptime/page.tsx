"use client";

import Link from "next/link";
import { Activity, ExternalLink, RefreshCw } from "lucide-react";
import { formatDistanceToNow } from "date-fns";
import { useQuery } from "@tanstack/react-query";
import { fetchUptimeOverview, type UptimeSiteOverview } from "@/lib/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { cn } from "@/lib/utils";

function StatusDot({ status }: { status: string }) {
  if (!status) return <span className="h-2.5 w-2.5 rounded-full bg-muted inline-block" />;
  return (
    <span
      className={cn(
        "inline-block h-2.5 w-2.5 rounded-full",
        status === "up" ? "bg-green-500" : "bg-red-500",
      )}
    />
  );
}

function PctBadge({ pct }: { pct: number }) {
  const color =
    pct >= 99 ? "text-green-600" : pct >= 90 ? "text-yellow-600" : "text-red-600";
  return <span className={cn("font-semibold tabular-nums", color)}>{pct.toFixed(1)}%</span>;
}

function formatBytes(bytes: number) {
  if (bytes === 0) return "—";
  return bytes < 1024 ? `${bytes} B` : `${(bytes / 1024).toFixed(0)} KB`;
}

export default function UptimePage() {
  const { data: sites = [], isFetching } = useQuery<UptimeSiteOverview[]>({
    queryKey: ["uptime-overview"],
    queryFn: fetchUptimeOverview,
    refetchInterval: 60_000,
  });

  const upCount = sites.filter((s) => s.last_status === "up").length;
  const downCount = sites.filter((s) => s.last_status === "down").length;
  const unchecked = sites.filter((s) => !s.last_status).length;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Activity className="h-6 w-6 text-muted-foreground" />
          <div>
            <h2 className="text-2xl font-bold tracking-tight">Uptime Monitor</h2>
            <p className="text-muted-foreground text-sm">All sites · checks every 60 seconds</p>
          </div>
        </div>
        {isFetching && <RefreshCw className="h-4 w-4 animate-spin text-muted-foreground" />}
      </div>

      {/* Summary cards */}
      <div className="grid grid-cols-3 gap-4">
        <Card>
          <CardContent className="pt-5 pb-4 text-center">
            <p className="text-3xl font-bold text-green-600">{upCount}</p>
            <p className="text-xs text-muted-foreground mt-1">Operational</p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-5 pb-4 text-center">
            <p className="text-3xl font-bold text-red-600">{downCount}</p>
            <p className="text-xs text-muted-foreground mt-1">Down</p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-5 pb-4 text-center">
            <p className="text-3xl font-bold text-muted-foreground">{unchecked}</p>
            <p className="text-xs text-muted-foreground mt-1">Not yet checked</p>
          </CardContent>
        </Card>
      </div>

      {/* Sites table */}
      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-base">All sites</CardTitle>
        </CardHeader>
        <CardContent className="p-0">
          {sites.length === 0 ? (
            <p className="px-6 py-4 text-sm text-muted-foreground">
              No sites found or no checks recorded yet.
            </p>
          ) : (
            <div className="divide-y">
              {sites.map((site) => (
                <div
                  key={site.site_id}
                  className="flex items-center gap-4 px-6 py-3 hover:bg-muted/40 transition-colors"
                >
                  <StatusDot status={site.last_status} />

                  <div className="min-w-0 flex-1">
                    <p className="truncate font-medium text-sm">{site.site_name}</p>
                    <p className="truncate text-xs text-muted-foreground">{site.domain}</p>
                  </div>

                  {/* Uptime % */}
                  <div className="text-right hidden sm:block w-16">
                    {site.last_status ? (
                      <PctBadge pct={site.uptime_pct_90} />
                    ) : (
                      <span className="text-xs text-muted-foreground">—</span>
                    )}
                    <p className="text-xs text-muted-foreground">90 checks</p>
                  </div>

                  {/* Latency */}
                  <div className="text-right hidden md:block w-20">
                    {site.latency_ms > 0 ? (
                      <>
                        <p className="text-sm font-medium">{site.latency_ms} ms</p>
                        <p className="text-xs text-muted-foreground">latency</p>
                      </>
                    ) : (
                      <span className="text-xs text-muted-foreground">—</span>
                    )}
                  </div>

                  {/* Last checked */}
                  <div className="text-right hidden lg:block w-28">
                    {site.last_status ? (
                      <p className="text-xs text-muted-foreground">
                        {formatDistanceToNow(new Date(site.last_checked_at), { addSuffix: true })}
                      </p>
                    ) : (
                      <p className="text-xs text-muted-foreground">Never</p>
                    )}
                  </div>

                  <Link
                    href={`/sites/${site.site_id}`}
                    className="ml-2 text-muted-foreground hover:text-foreground"
                    title="View site"
                  >
                    <ExternalLink className="h-4 w-4" />
                  </Link>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
