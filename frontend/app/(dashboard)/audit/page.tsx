"use client";

import { useState } from "react";
import { ClipboardList, ChevronDown } from "lucide-react";
import { useAuditEvents } from "@/hooks/use-audit";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { TableSkeleton } from "@/components/ui/table-skeleton";
import { cn } from "@/lib/utils";

type ResourceType = "" | "server" | "site";

const STATUS_COLORS: Record<string, string> = {
  access_denied: "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400",
  deploy_failed: "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400",
  provision_failed: "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400",
  ssl_failed: "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400",
  deployed: "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400",
  connected: "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400",
  ready_for_deploy: "bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400",
  ssl_active: "bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400",
};

function statusClass(status: string): string {
  return (
    STATUS_COLORS[status] ??
    "bg-muted text-muted-foreground"
  );
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleString(undefined, {
    dateStyle: "short",
    timeStyle: "medium",
  });
}

function shortID(id: string): string {
  return id.slice(0, 8) + "…";
}

export default function AuditPage() {
  const [resourceType, setResourceType] = useState<ResourceType>("");

  const {
    data,
    isFetchingNextPage,
    hasNextPage,
    fetchNextPage,
    isLoading,
  } = useAuditEvents(resourceType ? { resource_type: resourceType } : undefined);

  const events = data?.pages.flatMap((p) => p.items) ?? [];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <ClipboardList className="h-6 w-6 text-muted-foreground" />
          <div>
            <h2 className="text-2xl font-bold tracking-tight">Audit Log</h2>
            <p className="text-muted-foreground">
              Status change events for servers and sites
            </p>
          </div>
        </div>

        {/* Filter by resource type */}
        <div className="flex items-center gap-2">
          <span className="text-sm text-muted-foreground">Filter:</span>
          {(["", "server", "site"] as ResourceType[]).map((type) => (
            <button
              key={type}
              onClick={() => setResourceType(type)}
              className={cn(
                "rounded-md px-3 py-1 text-sm font-medium transition-colors",
                resourceType === type
                  ? "bg-primary text-primary-foreground"
                  : "bg-muted text-muted-foreground hover:bg-accent hover:text-accent-foreground",
              )}
            >
              {type === "" ? "All" : type.charAt(0).toUpperCase() + type.slice(1)}
            </button>
          ))}
        </div>
      </div>

      {isLoading ? (
        <TableSkeleton cols={6} rows={8} headers={["Time", "Resource", "ID", "From", "To", "Reason"]} />
      ) : (
      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Time</TableHead>
              <TableHead>Resource</TableHead>
              <TableHead>ID</TableHead>
              <TableHead>From</TableHead>
              <TableHead>To</TableHead>
              <TableHead>Reason</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {events.length === 0 && (
              <TableRow>
                <TableCell colSpan={6} className="py-10 text-center text-muted-foreground">
                  No events found
                </TableCell>
              </TableRow>
            )}
            {events.map((e) => (
              <TableRow key={e.ID}>
                <TableCell className="whitespace-nowrap text-xs text-muted-foreground">
                  {formatDate(e.CreatedAt)}
                </TableCell>
                <TableCell>
                  <Badge variant="outline" className="capitalize">
                    {e.ResourceType}
                  </Badge>
                </TableCell>
                <TableCell className="font-mono text-xs" title={e.ResourceID}>
                  {shortID(e.ResourceID)}
                </TableCell>
                <TableCell>
                  <span
                    className={cn(
                      "rounded px-2 py-0.5 text-xs font-medium",
                      statusClass(e.FromStatus),
                    )}
                  >
                    {e.FromStatus || "—"}
                  </span>
                </TableCell>
                <TableCell>
                  <span
                    className={cn(
                      "rounded px-2 py-0.5 text-xs font-medium",
                      statusClass(e.ToStatus),
                    )}
                  >
                    {e.ToStatus}
                  </span>
                </TableCell>
                <TableCell className="text-xs text-muted-foreground">
                  {e.Reason}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
      )} {/* end isLoading ternary */}

      {hasNextPage && (
        <div className="flex justify-center">
          <Button
            variant="outline"
            size="sm"
            disabled={isFetchingNextPage}
            onClick={() => fetchNextPage()}
          >
            {isFetchingNextPage ? (
              "Loading…"
            ) : (
              <>
                <ChevronDown className="mr-2 h-4 w-4" />
                Load more
              </>
            )}
          </Button>
        </div>
      )}
    </div>
  );
}
