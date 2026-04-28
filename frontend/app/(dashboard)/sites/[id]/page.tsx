"use client";

import { use, useState } from "react";
import Link from "next/link";
import {
  ArrowLeft,
  Globe,
  Rocket,
  Pencil,
  Trash2,
  ExternalLink,
  Clock,
} from "lucide-react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { fetchSiteByID } from "@/lib/api";
import { useAuditEvents } from "@/hooks/use-audit";
import { useDeploySite, useDeleteSite } from "@/hooks/use-site-mutations";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { SiteForm } from "@/components/site-form";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { RefreshIndicator } from "@/components/refresh-indicator";
import { toast } from "sonner";
import { cn } from "@/lib/utils";
import { useRouter } from "next/navigation";

const STATUS_COLORS: Record<string, string> = {
  access_denied: "bg-red-100 text-red-700",
  deploy_failed: "bg-red-100 text-red-700",
  deployed: "bg-green-100 text-green-700",
  ssl_active: "bg-green-100 text-green-700",
  deploying: "bg-blue-100 text-blue-700",
  ssl_pending: "bg-yellow-100 text-yellow-700",
};

function statusBadgeVariant(
  status: string,
): "success" | "destructive" | "warning" | "secondary" {
  switch (status.toLowerCase()) {
    case "deployed":
    case "ssl_active":
      return "success";
    case "deploy_failed":
    case "ssl_failed":
      return "destructive";
    case "deploying":
    case "ssl_pending":
      return "warning";
    default:
      return "secondary";
  }
}

function statusClass(s: string) {
  return STATUS_COLORS[s] ?? "bg-muted text-muted-foreground";
}

function formatDate(iso: string) {
  return new Date(iso).toLocaleString(undefined, {
    dateStyle: "short",
    timeStyle: "medium",
  });
}

export default function SiteDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  const router = useRouter();
  const qc = useQueryClient();

  const { data: site, isFetching: siteFetching, dataUpdatedAt } = useQuery({
    queryKey: ["site", id],
    queryFn: () => fetchSiteByID(id),
    refetchInterval: 15_000,
    refetchIntervalInBackground: false,
  });

  const {
    data: auditData,
    isLoading: auditLoading,
    hasNextPage,
    fetchNextPage,
    isFetchingNextPage,
  } = useAuditEvents({ resource_type: "site", resource_id: id });
  const events = auditData?.pages.flatMap((p) => p.items) ?? [];

  const deploySite = useDeploySite();
  const deleteSite = useDeleteSite();

  const [editOpen, setEditOpen] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);

  async function handleDeploy() {
    try {
      await deploySite.mutateAsync(id);
      toast.success("Deploy queued");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Deploy failed");
    }
  }

  async function handleDelete() {
    try {
      await deleteSite.mutateAsync(id);
      toast.success("Site deleted");
      router.push("/sites");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Delete failed");
      setDeleteOpen(false);
    }
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Button variant="ghost" size="icon" asChild>
            <Link href="/sites">
              <ArrowLeft className="h-4 w-4" />
            </Link>
          </Button>
          <Globe className="h-6 w-6 text-muted-foreground" />
          <div>
            <h2 className="text-2xl font-bold tracking-tight">
              {site?.Name ?? id}
            </h2>
            <div className="flex items-center gap-1 text-sm text-muted-foreground">
              <span className="font-mono">{site?.Domain ?? "Loading…"}</span>
              {site?.Domain && (
                <a
                  href={`http://${site.Domain}`}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="hover:text-foreground"
                >
                  <ExternalLink className="h-3 w-3" />
                </a>
              )}
            </div>
          </div>
          {site && (
            <Badge variant={statusBadgeVariant(site.Status)}>
              {site.Status}
            </Badge>
          )}
        </div>

        <div className="flex items-center gap-2">
          <RefreshIndicator
            isFetching={siteFetching}
            dataUpdatedAt={dataUpdatedAt}
            intervalSeconds={15}
            onRefresh={() => {
              qc.invalidateQueries({ queryKey: ["site", id] });
              qc.invalidateQueries({ queryKey: ["audit"] });
            }}
          />
          <Button
            variant="outline"
            size="sm"
            disabled={deploySite.isPending}
            onClick={handleDeploy}
          >
            <Rocket className="mr-2 h-4 w-4" />
            Deploy
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={() => setEditOpen(true)}
          >
            <Pencil className="mr-2 h-4 w-4" />
            Edit
          </Button>
          <Button
            variant="outline"
            size="sm"
            className="text-destructive hover:text-destructive"
            onClick={() => setDeleteOpen(true)}
          >
            <Trash2 className="mr-2 h-4 w-4" />
            Delete
          </Button>
        </div>
      </div>

      {/* Info cards */}
      {site && (
        <div className="grid gap-4 sm:grid-cols-4">
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm text-muted-foreground">
                Runtime
              </CardTitle>
            </CardHeader>
            <CardContent className="text-lg font-semibold capitalize">
              {site.Runtime || "—"}
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm text-muted-foreground">
                Domain
              </CardTitle>
            </CardHeader>
            <CardContent className="font-mono text-lg font-semibold">
              {site.Domain}
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm text-muted-foreground">
                Repository
              </CardTitle>
            </CardHeader>
            <CardContent className="truncate text-sm font-medium">
              {site.RepositoryURL || <span className="text-muted-foreground">—</span>}
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm text-muted-foreground">
                Created
              </CardTitle>
            </CardHeader>
            <CardContent className="text-lg font-semibold">
              {site.CreatedAt
                ? new Date(site.CreatedAt).toLocaleDateString()
                : "—"}
            </CardContent>
          </Card>
        </div>
      )}

      {/* Audit history */}
      <div>
        <div className="mb-3 flex items-center justify-between">
          <h3 className="text-lg font-semibold">Event History</h3>
          <span className="text-xs text-muted-foreground">
            {events.length} events
          </span>
        </div>

        {auditLoading && (
          <p className="text-sm text-muted-foreground">Loading events…</p>
        )}

        {!auditLoading && events.length === 0 && (
          <p className="text-sm text-muted-foreground">No events yet.</p>
        )}

        {events.length > 0 && (
          <div className="relative ml-3 space-y-0">
            {events.map((e, i) => (
              <div key={e.ID} className="flex gap-4">
                {/* Timeline line */}
                <div className="flex flex-col items-center">
                  <div className="mt-1.5 h-2.5 w-2.5 rounded-full border-2 border-primary bg-background" />
                  {i < events.length - 1 && (
                    <div className="w-px flex-1 bg-border" />
                  )}
                </div>

                <div className="pb-5">
                  <div className="flex items-center gap-2">
                    <span
                      className={cn(
                        "rounded px-2 py-0.5 text-xs font-medium",
                        statusClass(e.ToStatus),
                      )}
                    >
                      {e.ToStatus}
                    </span>
                    {e.FromStatus && (
                      <span className="text-xs text-muted-foreground">
                        ← {e.FromStatus}
                      </span>
                    )}
                    <span className="text-xs text-muted-foreground">
                      · {e.Reason}
                    </span>
                  </div>
                  <div className="mt-1 flex items-center gap-1 text-xs text-muted-foreground">
                    <Clock className="h-3 w-3" />
                    {formatDate(e.CreatedAt)}
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}

        {hasNextPage && (
          <Button
            variant="outline"
            size="sm"
            disabled={isFetchingNextPage}
            onClick={() => fetchNextPage()}
          >
            {isFetchingNextPage ? "Loading…" : "Load older events"}
          </Button>
        )}
      </div>

      {/* Modals */}
      <SiteForm
        open={editOpen}
        onOpenChange={setEditOpen}
        site={site}
      />
      <ConfirmDialog
        open={deleteOpen}
        title={`Delete "${site?.Name}"?`}
        description="This site will be permanently removed. This cannot be undone."
        loading={deleteSite.isPending}
        onConfirm={handleDelete}
        onCancel={() => setDeleteOpen(false)}
      />
    </div>
  );
}
