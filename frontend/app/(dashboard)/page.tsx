"use client";

import { Server, Globe, Activity } from "lucide-react";
import { useServers } from "@/hooks/use-servers";
import { useSites } from "@/hooks/use-sites";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { ServersTable } from "@/components/servers-table";
import { SitesTable } from "@/components/sites-table";

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
  const { data: servers } = useServers();
  const { data: sites } = useSites();

  const connectedServers =
    servers?.filter((s) => s.Status === "connected").length ?? 0;
  const deployedSites =
    sites?.filter((s) => s.Status === "deployed").length ?? 0;

  return (
    <div className="space-y-8">
      <div>
        <h2 className="text-2xl font-bold tracking-tight">Dashboard</h2>
        <p className="text-muted-foreground">
          Overview of your infrastructure
        </p>
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
          description="API is healthy"
        />
      </div>

      <div className="space-y-4">
        <h3 className="text-lg font-semibold">Recent Servers</h3>
        <ServersTable />
      </div>

      <div className="space-y-4">
        <h3 className="text-lg font-semibold">Recent Sites</h3>
        <SitesTable />
      </div>
    </div>
  );
}
