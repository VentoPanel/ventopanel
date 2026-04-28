import { Server } from "lucide-react";
import { ServersTable } from "@/components/servers-table";

export default function ServersPage() {
  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <Server className="h-6 w-6 text-muted-foreground" />
        <div>
          <h2 className="text-2xl font-bold tracking-tight">Servers</h2>
          <p className="text-muted-foreground">
            Manage your connected servers
          </p>
        </div>
      </div>
      <ServersTable />
    </div>
  );
}
