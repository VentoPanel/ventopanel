"use client";

import { useState } from "react";
import { Globe, Plus } from "lucide-react";
import { SitesTable } from "@/components/sites-table";
import { SiteForm } from "@/components/site-form";
import { Button } from "@/components/ui/button";

export default function SitesPage() {
  const [open, setOpen] = useState(false);

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
        <Button onClick={() => setOpen(true)}>
          <Plus className="mr-2 h-4 w-4" />
          New Site
        </Button>
      </div>

      <SitesTable />

      <SiteForm open={open} onOpenChange={setOpen} />
    </div>
  );
}
