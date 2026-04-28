"use client";

import { useServers } from "@/hooks/use-servers";
import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

function statusVariant(status: string) {
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

  if (isLoading)
    return <p className="text-sm text-muted-foreground">Loading servers…</p>;
  if (isError)
    return (
      <p className="text-sm text-destructive">Failed to load servers.</p>
    );
  if (!servers?.length)
    return <p className="text-sm text-muted-foreground">No servers found.</p>;

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Name</TableHead>
          <TableHead>Host</TableHead>
          <TableHead>Provider</TableHead>
          <TableHead>Status</TableHead>
          <TableHead>SSH User</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {servers.map((s) => (
          <TableRow key={s.ID}>
            <TableCell className="font-medium">{s.Name}</TableCell>
            <TableCell className="font-mono text-xs">
              {s.Host}:{s.Port}
            </TableCell>
            <TableCell className="capitalize">{s.Provider}</TableCell>
            <TableCell>
              <Badge variant={statusVariant(s.Status)}>{s.Status}</Badge>
            </TableCell>
            <TableCell>{s.SSHUser}</TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
