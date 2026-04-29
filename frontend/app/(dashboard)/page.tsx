"use client";

import Link from "next/link";
import { formatDistanceToNow } from "date-fns";
import {
  Server,
  Globe,
  AlertTriangle,
  CheckCircle2,
  Activity,
  ArrowRight,
} from "lucide-react";
import { useQueryClient } from "@tanstack/react-query";
import { useServers, SERVERS_REFETCH_INTERVAL } from "@/hooks/use-servers";
import { useSites, SITES_REFETCH_INTERVAL } from "@/hooks/use-sites";
import { useRecentAudit } from "@/hooks/use-recent-audit";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { RefreshIndicator } from "@/components/refresh-indicator";
import { cn } from "@/lib/utils";

const STATUS_COLORS: Record<string, string> = {
  deployed: "bg-green-100 text-green-800",
  ssl_pending: "bg-green-100 text-green-800",
  connected: "bg-green-100 text-green-800",
  ready_for_deploy: "bg-green-100 text-green-800",
  provisioned: "bg-blue-100 text-blue-800",
  pending: "bg-yellow-100 text-yellow-800",
  deploying: "bg-blue-100 text-blue-800",
  provisioning: "bg-blue-100 text-blue-800",
  deploy_failed: "bg-red-100 text-red-800",
  provision_failed: "bg-red-100 text-red-800",
  connection_failed: "bg-red-100 text-red-800",
  error: "bg-red-100 text-red-800",
  access_denied: "bg-red-100 text-red-800",
  failed: "bg-red-100 text-red-800",
  draft: "bg-gray-100 text-gray-700",
};

function statusColor(s: string) {
  return STATUS_COLORS[s] ?? "bg-gray-100 text-gray-700";
}

function isError(status: string) {
  return ["error", "failed", "access_denied"].includes(status);
}

function StatCard({
  title,
  value,
  icon: Icon,
  sub,
  subColor,
  href,
}: {
  title: string;
  value: number | string;
  icon: React.ElementType;
  sub?: string;
  subColor?: string;
  href?: string;
}) {
  const content = (
    <Card className={cn("transition-shadow", href && "hover:shadow-md cursor-pointer")}>
      <CardHeader className="flex flex-row items-center justify-between pb-2">
        <CardTitle className="text-sm font-medium text-muted-foreground">
          {title}
        </CardTitle>
        <Icon className="h-4 w-4 text-muted-foreground" />
      </CardHeader>
      <CardContent>
        <div className="text-3xl font-bold">{value}</div>
        {sub && (
          <p className={cn("mt-1 text-xs", subColor ?? "text-muted-foreground")}>
            {sub}
          </p>
        )}
      </CardContent>
    </Card>
  );
  return href ? <Link href={href}>{content}</Link> : content;
}

export default function DashboardPage() {
  const qc = useQueryClient();

  const { data: servers, isFetching: serversFetching, dataUpdatedAt: serversUpdatedAt } = useServers();
  const { data: sites, isFetching: sitesFetching, dataUpdatedAt: sitesUpdatedAt } = useSites();
  const { data: recentAudit, isFetching: auditFetching } = useRecentAudit(8);

  const totalServers = servers?.length ?? 0;
  // "active" servers = anything past initial connection (connected OR provisioned OR ready)
  const activeServers = servers?.filter((s) =>
    ["connected", "provisioning", "ready_for_deploy", "provisioned"].includes(s.Status)
  ).length ?? 0;
  const errorServers = servers?.filter((s) => isError(s.Status)).length ?? 0;

  const totalSites = sites?.length ?? 0;
  // "live" sites = deployed or ssl_pending (site is serving, just no cert yet)
  const liveSites = sites?.filter((s) =>
    ["deployed", "ssl_pending"].includes(s.Status)
  ).length ?? 0;
  const errorSites = sites?.filter((s) => isError(s.Status)).length ?? 0;

  const recentEvents = recentAudit?.items ?? [];
  const recentErrors = recentEvents.filter((e) => isError(e.ToStatus)).length;

  const anyFetching = serversFetching || sitesFetching || auditFetching;

  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="flex items-start justify-between">
        <div>
          <h2 className="text-2xl font-bold tracking-tight">Dashboard</h2>
          <p className="text-muted-foreground">Overview of your infrastructure</p>
        </div>
        <RefreshIndicator
          isFetching={anyFetching}
          dataUpdatedAt={serversUpdatedAt || sitesUpdatedAt}
          intervalSeconds={SERVERS_REFETCH_INTERVAL / 1000}
          onRefresh={() => {
            qc.invalidateQueries({ queryKey: ["servers"] });
            qc.invalidateQueries({ queryKey: ["sites"] });
            qc.invalidateQueries({ queryKey: ["audit-recent"] });
          }}
        />
      </div>

      {/* Stat cards */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard
          title="Servers"
          value={totalServers === 0 && !servers ? "—" : totalServers}
          icon={Server}
          sub={
            errorServers > 0
              ? `${errorServers} error${errorServers > 1 ? "s" : ""} · ${activeServers} active`
              : `${activeServers} active`
          }
          subColor={errorServers > 0 ? "text-red-600" : undefined}
          href="/servers"
        />
        <StatCard
          title="Sites"
          value={totalSites === 0 && !sites ? "—" : totalSites}
          icon={Globe}
          sub={
            errorSites > 0
              ? `${errorSites} error${errorSites > 1 ? "s" : ""} · ${liveSites} live`
              : `${liveSites} live`
          }
          subColor={errorSites > 0 ? "text-red-600" : undefined}
          href="/sites"
        />
        <StatCard
          title="Recent Errors"
          value={recentErrors}
          icon={recentErrors > 0 ? AlertTriangle : CheckCircle2}
          sub={recentErrors > 0 ? "in last 8 events" : "all clear"}
          subColor={recentErrors > 0 ? "text-red-600" : "text-green-600"}
          href="/audit"
        />
        <StatCard
          title="Auto-refresh"
          value={`${SERVERS_REFETCH_INTERVAL / 1000}s`}
          icon={Activity}
          sub="all tables refresh automatically"
        />
      </div>

      {/* Recent Activity */}
      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <h3 className="text-lg font-semibold">Recent Activity</h3>
          <Button variant="ghost" size="sm" asChild>
            <Link href="/audit" className="flex items-center gap-1 text-xs text-muted-foreground">
              View all <ArrowRight className="h-3 w-3" />
            </Link>
          </Button>
        </div>

        {recentEvents.length === 0 ? (
          <Card>
            <CardContent className="py-8 text-center text-sm text-muted-foreground">
              No events yet.
            </CardContent>
          </Card>
        ) : (
          <Card>
            <CardContent className="p-0">
              <ul className="divide-y">
                {recentEvents.map((event) => (
                  <li
                    key={event.ID}
                    className="flex items-center gap-4 px-4 py-3 text-sm"
                  >
                    <div className="w-14 shrink-0">
                      <Badge
                        variant="outline"
                        className={cn("text-xs capitalize", statusColor(event.ToStatus))}
                      >
                        {event.ResourceType}
                      </Badge>
                    </div>
                    <div className="min-w-0 flex-1">
                      <span className="font-mono text-xs text-muted-foreground">
                        {event.ResourceID.slice(0, 8)}…
                      </span>
                      <span className="mx-2 text-muted-foreground">→</span>
                      <Badge
                        variant="outline"
                        className={cn("text-xs", statusColor(event.ToStatus))}
                      >
                        {event.ToStatus}
                      </Badge>
                      {event.Reason && (
                        <span className="ml-2 text-muted-foreground">
                          {event.Reason}
                        </span>
                      )}
                    </div>
                    <div className="shrink-0 text-xs text-muted-foreground">
                      {formatDistanceToNow(new Date(event.CreatedAt), {
                        addSuffix: true,
                      })}
                    </div>
                  </li>
                ))}
              </ul>
            </CardContent>
          </Card>
        )}
      </div>

      {/* Quick links */}
      <div className="grid gap-4 sm:grid-cols-2">
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="flex items-center justify-between text-sm">
              <span>Servers</span>
              <Button variant="ghost" size="sm" asChild>
                <Link href="/servers" className="flex items-center gap-1 text-xs">
                  Manage <ArrowRight className="h-3 w-3" />
                </Link>
              </Button>
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-2">
            {!servers || servers.length === 0 ? (
              <p className="text-sm text-muted-foreground">No servers yet.</p>
            ) : (
              servers.slice(0, 5).map((s) => (
                <Link
                  key={s.ID}
                  href={`/servers/${s.ID}`}
                  className="flex items-center justify-between rounded-md px-2 py-1.5 text-sm hover:bg-accent"
                >
                  <span className="font-medium">{s.Name}</span>
                  <Badge
                    variant="outline"
                    className={cn("text-xs", statusColor(s.Status))}
                  >
                    {s.Status}
                  </Badge>
                </Link>
              ))
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="flex items-center justify-between text-sm">
              <span>Sites</span>
              <Button variant="ghost" size="sm" asChild>
                <Link href="/sites" className="flex items-center gap-1 text-xs">
                  Manage <ArrowRight className="h-3 w-3" />
                </Link>
              </Button>
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-2">
            {!sites || sites.length === 0 ? (
              <p className="text-sm text-muted-foreground">No sites yet.</p>
            ) : (
              sites.slice(0, 5).map((s) => (
                <Link
                  key={s.ID}
                  href={`/sites/${s.ID}`}
                  className="flex items-center justify-between rounded-md px-2 py-1.5 text-sm hover:bg-accent"
                >
                  <span className="font-medium">{s.Domain}</span>
                  <Badge
                    variant="outline"
                    className={cn("text-xs", statusColor(s.Status))}
                  >
                    {s.Status}
                  </Badge>
                </Link>
              ))
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
