"use client";

import { useState } from "react";
import { Server, Plus } from "lucide-react";
import { useQueryClient } from "@tanstack/react-query";
import { useServers, SERVERS_REFETCH_INTERVAL } from "@/hooks/use-servers";
import { ServersTable } from "@/components/servers-table";
import { ServerForm } from "@/components/server-form";
import { RefreshIndicator } from "@/components/refresh-indicator";
import { Button } from "@/components/ui/button";
import { useAuth } from "@/hooks/use-auth";

export default function ServersPage() {
  const [open, setOpen] = useState(false);
  const qc = useQueryClient();
  const { isFetching, dataUpdatedAt } = useServers();
  const { canWrite } = useAuth();

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-3">
          <Server className="h-6 w-6 text-muted-foreground" />
          <div>
            <h2 className="text-2xl font-bold tracking-tight">Servers</h2>
            <p className="text-muted-foreground">
              Manage your connected servers
            </p>
          </div>
        </div>
        <div className="flex items-center gap-3">
          <RefreshIndicator
            isFetching={isFetching}
            dataUpdatedAt={dataUpdatedAt}
            intervalSeconds={SERVERS_REFETCH_INTERVAL / 1000}
            onRefresh={() => qc.invalidateQueries({ queryKey: ["servers"] })}
          />
          {canWrite && (
            <Button onClick={() => setOpen(true)}>
              <Plus className="mr-2 h-4 w-4" />
              New Server
            </Button>
          )}
        </div>
      </div>

      <ServersTable />
      <ServerForm open={open} onOpenChange={setOpen} />
    </div>
  );
}
