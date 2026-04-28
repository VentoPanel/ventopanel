"use client";

import { use } from "react";
import Link from "next/link";
import {
  ArrowLeft,
  Server,
  Cpu,
  MemoryStick,
  HardDrive,
  Clock,
  Plug,
  Wrench,
} from "lucide-react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { fetchServers } from "@/lib/api";
import { useServerStats } from "@/hooks/use-server-stats";
import { useConnectServer, useProvisionServer } from "@/hooks/use-server-mutations";
import { useAuth } from "@/hooks/use-auth";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { RefreshIndicator } from "@/components/refresh-indicator";
import { toast } from "sonner";
import { cn } from "@/lib/utils";

function StatBar({
  label,
  used,
  total,
  unit = "",
}: {
  label: string;
  used: number;
  total: number;
  unit?: string;
}) {
  const pct = total > 0 ? Math.min(100, Math.round((used / total) * 100)) : 0;
  const color =
    pct >= 90
      ? "bg-red-500"
      : pct >= 70
        ? "bg-yellow-500"
        : "bg-green-500";

  return (
    <div className="space-y-1">
      <div className="flex justify-between text-xs text-muted-foreground">
        <span>{label}</span>
        <span>
          {used}
          {unit} / {total}
          {unit} ({pct}%)
        </span>
      </div>
      <div className="h-2 w-full rounded-full bg-muted">
        <div
          className={cn("h-2 rounded-full transition-all", color)}
          style={{ width: `${pct}%` }}
        />
      </div>
    </div>
  );
}

function statusVariant(status: string) {
  switch (status.toLowerCase()) {
    case "connected":
    case "ready_for_deploy":
      return "success";
    case "connection_failed":
    case "provision_failed":
      return "destructive";
    default:
      return "secondary";
  }
}

export default function ServerDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  const qc = useQueryClient();

  const { data: servers } = useQuery({
    queryKey: ["servers"],
    queryFn: fetchServers,
  });
  const server = servers?.find((s) => s.ID === id);

  const {
    data: stats,
    isLoading: statsLoading,
    isError: statsError,
    isFetching: statsFetching,
    dataUpdatedAt,
  } = useServerStats(id);

  const { canWrite } = useAuth();
  const connectServer = useConnectServer();
  const provisionServer = useProvisionServer();

  async function handleConnect() {
    try {
      await connectServer.mutateAsync(id);
      toast.success("Connection successful");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Connect failed");
    }
  }

  async function handleProvision() {
    try {
      await provisionServer.mutateAsync(id);
      toast.success("Provisioning queued");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Provision failed");
    }
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Button variant="ghost" size="icon" asChild>
            <Link href="/servers">
              <ArrowLeft className="h-4 w-4" />
            </Link>
          </Button>
          <Server className="h-6 w-6 text-muted-foreground" />
          <div>
            <h2 className="text-2xl font-bold tracking-tight">
              {server?.Name ?? id}
            </h2>
            <p className="font-mono text-sm text-muted-foreground">
              {server ? `${server.Host}:${server.Port}` : "Loading…"}
            </p>
          </div>
          {server && (
            <Badge variant={statusVariant(server.Status)}>
              {server.Status}
            </Badge>
          )}
        </div>

        <div className="flex items-center gap-2">
          <RefreshIndicator
            isFetching={statsFetching}
            dataUpdatedAt={dataUpdatedAt}
            intervalSeconds={30}
            onRefresh={() =>
              qc.invalidateQueries({ queryKey: ["server-stats", id] })
            }
          />
          {canWrite && (
            <>
              <Button
                variant="outline"
                size="sm"
                disabled={connectServer.isPending}
                onClick={handleConnect}
              >
                <Plug className="mr-2 h-4 w-4" />
                Connect
              </Button>
              <Button
                variant="outline"
                size="sm"
                disabled={provisionServer.isPending}
                onClick={handleProvision}
              >
                <Wrench className="mr-2 h-4 w-4" />
                Provision
              </Button>
            </>
          )}
        </div>
      </div>

      {/* Server info */}
      {server && (
        <div className="grid gap-4 sm:grid-cols-3">
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm text-muted-foreground">
                Provider
              </CardTitle>
            </CardHeader>
            <CardContent className="text-lg font-semibold capitalize">
              {server.Provider || "—"}
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm text-muted-foreground">
                SSH User
              </CardTitle>
            </CardHeader>
            <CardContent className="font-mono text-lg font-semibold">
              {server.SSHUser}
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm text-muted-foreground">
                Added
              </CardTitle>
            </CardHeader>
            <CardContent className="text-lg font-semibold">
              {server.CreatedAt
                ? new Date(server.CreatedAt).toLocaleDateString()
                : "—"}
            </CardContent>
          </Card>
        </div>
      )}

      {/* Live stats */}
      <div>
        <h3 className="mb-3 text-lg font-semibold">Live Monitoring</h3>

        {statsLoading && (
          <p className="text-sm text-muted-foreground">
            Fetching server stats…
          </p>
        )}
        {statsError && (
          <p className="text-sm text-destructive">
            Could not fetch stats — server may be offline or not yet connected.
          </p>
        )}

        {stats && (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            {/* CPU */}
            <Card>
              <CardHeader className="flex flex-row items-center justify-between pb-2">
                <CardTitle className="text-sm font-medium text-muted-foreground">
                  CPU
                </CardTitle>
                <Cpu className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent className="space-y-2">
                <p className="text-2xl font-bold">{stats.cpu_cores} cores</p>
                <p className="text-xs text-muted-foreground">
                  Load avg (1 min):{" "}
                  <span className="font-medium text-foreground">
                    {stats.load_avg_1.toFixed(2)}
                  </span>
                </p>
                <StatBar
                  label="Load"
                  used={Math.round(stats.load_avg_1 * 100)}
                  total={stats.cpu_cores * 100}
                />
              </CardContent>
            </Card>

            {/* RAM */}
            <Card>
              <CardHeader className="flex flex-row items-center justify-between pb-2">
                <CardTitle className="text-sm font-medium text-muted-foreground">
                  Memory
                </CardTitle>
                <MemoryStick className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent className="space-y-2">
                <p className="text-2xl font-bold">
                  {stats.ram_used_mb.toLocaleString()} MB
                  <span className="ml-1 text-sm font-normal text-muted-foreground">
                    used
                  </span>
                </p>
                <StatBar
                  label="RAM"
                  used={stats.ram_used_mb}
                  total={stats.ram_total_mb}
                  unit=" MB"
                />
              </CardContent>
            </Card>

            {/* Disk */}
            <Card>
              <CardHeader className="flex flex-row items-center justify-between pb-2">
                <CardTitle className="text-sm font-medium text-muted-foreground">
                  Disk
                </CardTitle>
                <HardDrive className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent className="space-y-2">
                <p className="text-2xl font-bold">
                  {stats.disk_used}
                  <span className="ml-1 text-sm font-normal text-muted-foreground">
                    / {stats.disk_total}
                  </span>
                </p>
                <div className="h-2 w-full rounded-full bg-muted">
                  <div
                    className={cn(
                      "h-2 rounded-full transition-all",
                      parseInt(stats.disk_pct) >= 90
                        ? "bg-red-500"
                        : parseInt(stats.disk_pct) >= 70
                          ? "bg-yellow-500"
                          : "bg-green-500",
                    )}
                    style={{ width: stats.disk_pct }}
                  />
                </div>
                <p className="text-xs text-muted-foreground">
                  {stats.disk_free} free · {stats.disk_pct} used
                </p>
              </CardContent>
            </Card>

            {/* Uptime */}
            <Card>
              <CardHeader className="flex flex-row items-center justify-between pb-2">
                <CardTitle className="text-sm font-medium text-muted-foreground">
                  Uptime
                </CardTitle>
                <Clock className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <p className="text-lg font-bold leading-snug">{stats.uptime}</p>
              </CardContent>
            </Card>
          </div>
        )}
      </div>
    </div>
  );
}
