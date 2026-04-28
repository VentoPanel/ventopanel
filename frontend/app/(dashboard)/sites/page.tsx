import { Globe } from "lucide-react";
import { SitesTable } from "@/components/sites-table";

export default function SitesPage() {
  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <Globe className="h-6 w-6 text-muted-foreground" />
        <div>
          <h2 className="text-2xl font-bold tracking-tight">Sites</h2>
          <p className="text-muted-foreground">
            Manage your deployed sites
          </p>
        </div>
      </div>
      <SitesTable />
    </div>
  );
}
