"use client";

import { useState } from "react";
import { toast } from "sonner";
import { Pencil, Trash2, Rocket } from "lucide-react";
import { type Site } from "@/lib/api";
import { useSites } from "@/hooks/use-sites";
import {
  useDeleteSite,
  useDeploySite,
} from "@/hooks/use-site-mutations";
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
import { SiteForm } from "@/components/site-form";
import { ConfirmDialog } from "@/components/confirm-dialog";

function statusVariant(
  status: string,
): "success" | "destructive" | "warning" | "secondary" {
  switch (status.toLowerCase()) {
    case "deployed":
    case "active":
      return "success";
    case "error":
    case "failed":
      return "destructive";
    case "deploying":
    case "pending":
      return "warning";
    default:
      return "secondary";
  }
}

export function SitesTable() {
  const { data: sites, isLoading, isError } = useSites();
  const deleteSite = useDeleteSite();
  const deploySite = useDeploySite();

  const [editTarget, setEditTarget] = useState<Site | undefined>();
  const [editOpen, setEditOpen] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<Site | undefined>();

  if (isLoading)
    return <p className="text-sm text-muted-foreground">Loading sites…</p>;
  if (isError)
    return <p className="text-sm text-destructive">Failed to load sites.</p>;
  if (!sites?.length)
    return <p className="text-sm text-muted-foreground">No sites yet.</p>;

  async function handleDeploy(s: Site) {
    try {
      await deploySite.mutateAsync(s.ID);
      toast.success(`${s.Name}: deploy queued`);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Deploy failed");
    }
  }

  async function handleDelete() {
    if (!deleteTarget) return;
    try {
      await deleteSite.mutateAsync(deleteTarget.ID);
      toast.success(`${deleteTarget.Name} deleted`);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Delete failed");
    } finally {
      setDeleteTarget(undefined);
    }
  }

  return (
    <>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Name</TableHead>
            <TableHead>Domain</TableHead>
            <TableHead>Runtime</TableHead>
            <TableHead>Status</TableHead>
            <TableHead className="text-right">Actions</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {sites.map((s) => (
            <TableRow key={s.ID}>
              <TableCell className="font-medium">{s.Name}</TableCell>
              <TableCell className="font-mono text-xs">{s.Domain}</TableCell>
              <TableCell className="capitalize">{s.Runtime}</TableCell>
              <TableCell>
                <Badge variant={statusVariant(s.Status)}>{s.Status}</Badge>
              </TableCell>
              <TableCell>
                <div className="flex justify-end gap-1">
                  <Button
                    size="sm"
                    variant="ghost"
                    title="Deploy"
                    disabled={deploySite.isPending}
                    onClick={() => handleDeploy(s)}
                  >
                    <Rocket className="h-4 w-4" />
                  </Button>
                  <Button
                    size="sm"
                    variant="ghost"
                    title="Edit"
                    onClick={() => {
                      setEditTarget(s);
                      setEditOpen(true);
                    }}
                  >
                    <Pencil className="h-4 w-4" />
                  </Button>
                  <Button
                    size="sm"
                    variant="ghost"
                    title="Delete"
                    className="text-destructive hover:text-destructive"
                    onClick={() => setDeleteTarget(s)}
                  >
                    <Trash2 className="h-4 w-4" />
                  </Button>
                </div>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>

      <SiteForm
        open={editOpen}
        onOpenChange={(v) => {
          setEditOpen(v);
          if (!v) setEditTarget(undefined);
        }}
        site={editTarget}
      />

      <ConfirmDialog
        open={Boolean(deleteTarget)}
        title={`Delete "${deleteTarget?.Name}"?`}
        description="This site will be permanently removed. This cannot be undone."
        loading={deleteSite.isPending}
        onConfirm={handleDelete}
        onCancel={() => setDeleteTarget(undefined)}
      />
    </>
  );
}
