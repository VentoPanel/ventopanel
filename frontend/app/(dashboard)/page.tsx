"use client";

import { Server, Globe, Activity } from "lucide-react";
import { useQueryClient } from "@tanstack/react-query";
import { useServers, SERVERS_REFETCH_INTERVAL } from "@/hooks/use-servers";
import { useSites, SITES_REFETCH_INTERVAL } from "@/hooks/use-sites";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { ServersTable } from "@/components/servers-table";
import { SitesTable } from "@/components/sites-table";
import { RefreshIndicator } from "@/components/refresh-indicator";

function StatCard({
  title,
  value,
  icon: Icon,
  description,
}: {
  title: string;
  value: number | string;
  icon: React.ElementType;
  description?: string;
}) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between pb-2">
        <CardTitle className="text-sm font-medium text-muted-foreground">
          {title}
        </CardTitle>
        <Icon className="h-4 w-4 text-muted-foreground" />
      </CardHeader>
      <CardContent>
        <div className="text-3xl font-bold">{value}</div>
        {description && (
          <p className="mt-1 text-xs text-muted-foreground">{description}</p>
        )}
      </CardContent>
    </Card>
  );
}

export default function DashboardPage() {
  const qc = useQueryClient();

  const {
    data: servers,
    isFetching: serversFetching,
    dataUpdatedAt: serversUpdatedAt,
  } = useServers();

  const {
    data: sites,
    isFetching: sitesFetching,
    dataUpdatedAt: sitesUpdatedAt,
  } = useSites();

  const connectedServers =
    servers?.filter((s) => s.Status === "connected").length ?? 0;
  const deployedSites =
    sites?.filter((s) => s.Status === "deployed").length ?? 0;

  return (
    <div className="space-y-8">
      <div className="flex items-start justify-between">
        <div>
          <h2 className="text-2xl font-bold tracking-tight">Dashboard</h2>
          <p className="text-muted-foreground">
            Overview of your infrastructure
          </p>
        </div>
        <div className="flex flex-col items-end gap-1">
          <RefreshIndicator
            isFetching={serversFetching}
            dataUpdatedAt={serversUpdatedAt}
            intervalSeconds={SERVERS_REFETCH_INTERVAL / 1000}
            onRefresh={() => {
              qc.invalidateQueries({ queryKey: ["servers"] });
              qc.invalidateQueries({ queryKey: ["sites"] });
            }}
          />
          {sitesFetching && !serversFetching && (
            <span className="text-xs text-muted-foreground">
              Refreshing sites…
            </span>
          )}
        </div>
      </div>

      <div className="grid gap-4 sm:grid-cols-3">
        <StatCard
          title="Total Servers"
          value={servers?.length ?? "—"}
          icon={Server}
          description={`${connectedServers} connected`}
        />
        <StatCard
          title="Total Sites"
          value={sites?.length ?? "—"}
          icon={Globe}
          description={`${deployedSites} deployed`}
        />
        <StatCard
          title="Status"
          value="OK"
          icon={Activity}
          description={`Auto-refresh every ${SERVERS_REFETCH_INTERVAL / 1000}s`}
        />
      </div>

      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h3 className="text-lg font-semibold">Servers</h3>
          <RefreshIndicator
            isFetching={serversFetching}
            dataUpdatedAt={serversUpdatedAt}
            intervalSeconds={SERVERS_REFETCH_INTERVAL / 1000}
            onRefresh={() =>
              qc.invalidateQueries({ queryKey: ["servers"] })
            }
          />
        </div>
        <ServersTable />
      </div>

      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h3 className="text-lg font-semibold">Sites</h3>
          <RefreshIndicator
            isFetching={sitesFetching}
            dataUpdatedAt={sitesUpdatedAt}
            intervalSeconds={SITES_REFETCH_INTERVAL / 1000}
            onRefresh={() => qc.invalidateQueries({ queryKey: ["sites"] })}
          />
        </div>
        <SitesTable />
      </div>
    </div>
  );
}
