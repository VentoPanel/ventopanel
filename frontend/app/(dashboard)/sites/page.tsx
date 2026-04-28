"use client";

import { useState } from "react";
import { Globe, Plus } from "lucide-react";
import { useQueryClient } from "@tanstack/react-query";
import { useSites, SITES_REFETCH_INTERVAL } from "@/hooks/use-sites";
import { SitesTable } from "@/components/sites-table";
import { SiteForm } from "@/components/site-form";
import { RefreshIndicator } from "@/components/refresh-indicator";
import { Button } from "@/components/ui/button";

export default function SitesPage() {
  const [open, setOpen] = useState(false);
  const qc = useQueryClient();
  const { isFetching, dataUpdatedAt } = useSites();

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Globe className="h-6 w-6 text-muted-foreground" />
          <div>
            <h2 className="text-2xl font-bold tracking-tight">Sites</h2>
            <p className="text-muted-foreground">Manage your deployed sites</p>
          </div>
        </div>
        <div className="flex items-center gap-3">
          <RefreshIndicator
            isFetching={isFetching}
            dataUpdatedAt={dataUpdatedAt}
            intervalSeconds={SITES_REFETCH_INTERVAL / 1000}
            onRefresh={() => qc.invalidateQueries({ queryKey: ["sites"] })}
          />
          <Button onClick={() => setOpen(true)}>
            <Plus className="mr-2 h-4 w-4" />
            New Site
          </Button>
        </div>
      </div>

      <SitesTable />
      <SiteForm open={open} onOpenChange={setOpen} />
    </div>
  );
}
