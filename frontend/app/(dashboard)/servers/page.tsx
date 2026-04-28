"use client";

import { useState } from "react";
import { Server, Plus } from "lucide-react";
import { ServersTable } from "@/components/servers-table";
import { ServerForm } from "@/components/server-form";
import { Button } from "@/components/ui/button";

export default function ServersPage() {
  const [open, setOpen] = useState(false);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Server className="h-6 w-6 text-muted-foreground" />
          <div>
            <h2 className="text-2xl font-bold tracking-tight">Servers</h2>
            <p className="text-muted-foreground">Manage your connected servers</p>
          </div>
        </div>
        <Button onClick={() => setOpen(true)}>
          <Plus className="mr-2 h-4 w-4" />
          New Server
        </Button>
      </div>

      <ServersTable />

      <ServerForm open={open} onOpenChange={setOpen} />
    </div>
  );
}
