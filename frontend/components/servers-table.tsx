"use client";

import { useState } from "react";
import Link from "next/link";
import { toast } from "sonner";
import { Pencil, Trash2, Plug, Wrench, ExternalLink } from "lucide-react";
import { type Server } from "@/lib/api";
import { useServers } from "@/hooks/use-servers";
import { useAuth } from "@/hooks/use-auth";
import {
  useDeleteServer,
  useConnectServer,
  useProvisionServer,
} from "@/hooks/use-server-mutations";
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
import { ServerForm } from "@/components/server-form";
import { ConfirmDialog } from "@/components/confirm-dialog";

function statusVariant(
  status: string,
): "success" | "destructive" | "warning" | "secondary" {
  switch (status.toLowerCase()) {
    case "connected":
    case "provisioned":
      return "success";
    case "error":
    case "failed":
      return "destructive";
    case "pending":
    case "provisioning":
      return "warning";
    default:
      return "secondary";
  }
}

export function ServersTable() {
  const { data: servers, isLoading, isError } = useServers();
  const { isAdmin, canWrite } = useAuth();
  const deleteServer = useDeleteServer();
  const connectServer = useConnectServer();
  const provisionServer = useProvisionServer();

  const [editTarget, setEditTarget] = useState<Server | undefined>();
  const [editOpen, setEditOpen] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<Server | undefined>();

  if (isLoading)
    return <TableSkeleton cols={5} headers={["Name", "Host", "Provider", "Status", ""]} />;
  if (isError)
    return <p className="text-sm text-destructive">Failed to load servers.</p>;
  if (!servers?.length)
    return <p className="text-sm text-muted-foreground">No servers yet.</p>;

  async function handleConnect(s: Server) {
    try {
      await connectServer.mutateAsync(s.ID);
      toast.success(`${s.Name}: connection successful`);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Connect failed");
    }
  }

  async function handleProvision(s: Server) {
    try {
      await provisionServer.mutateAsync(s.ID);
      toast.success(`${s.Name}: provision queued`);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Provision failed");
    }
  }

  async function handleDelete() {
    if (!deleteTarget) return;
    try {
      await deleteServer.mutateAsync(deleteTarget.ID);
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
            <TableHead>Host</TableHead>
            <TableHead>Provider</TableHead>
            <TableHead>Status</TableHead>
            <TableHead>SSH User</TableHead>
            <TableHead className="text-right">Actions</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {servers.map((s) => (
            <TableRow key={s.ID}>
              <TableCell className="font-medium">
                <Link
                  href={`/servers/${s.ID}`}
                  className="flex items-center gap-1 hover:underline"
                >
                  {s.Name}
                  <ExternalLink className="h-3 w-3 text-muted-foreground" />
                </Link>
              </TableCell>
              <TableCell className="font-mono text-xs">
                {s.Host}:{s.Port}
              </TableCell>
              <TableCell className="capitalize">{s.Provider}</TableCell>
              <TableCell>
                <Badge variant={statusVariant(s.Status)}>{s.Status}</Badge>
              </TableCell>
              <TableCell>{s.SSHUser}</TableCell>
              <TableCell>
                <div className="flex justify-end gap-1">
                  {canWrite && (
                    <>
                      <Button
                        size="sm"
                        variant="ghost"
                        title="Connect"
                        disabled={connectServer.isPending}
                        onClick={() => handleConnect(s)}
                      >
                        <Plug className="h-4 w-4" />
                      </Button>
                      <Button
                        size="sm"
                        variant="ghost"
                        title="Provision"
                        disabled={provisionServer.isPending}
                        onClick={() => handleProvision(s)}
                      >
                        <Wrench className="h-4 w-4" />
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
                    </>
                  )}
                  {isAdmin && (
                    <Button
                      size="sm"
                      variant="ghost"
                      title="Delete"
                      className="text-destructive hover:text-destructive"
                      onClick={() => setDeleteTarget(s)}
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  )}
                </div>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>

      <ServerForm
        open={editOpen}
        onOpenChange={(v) => {
          setEditOpen(v);
          if (!v) setEditTarget(undefined);
        }}
        server={editTarget}
      />

      <ConfirmDialog
        open={Boolean(deleteTarget)}
        title={`Delete "${deleteTarget?.Name}"?`}
        description="All associated sites will also be deleted. This cannot be undone."
        loading={deleteServer.isPending}
        onConfirm={handleDelete}
        onCancel={() => setDeleteTarget(undefined)}
      />
    </>
  );
}
