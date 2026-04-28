"use client";

import { useSites } from "@/hooks/use-sites";
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

  if (isLoading)
    return <p className="text-sm text-muted-foreground">Loading sites…</p>;
  if (isError)
    return <p className="text-sm text-destructive">Failed to load sites.</p>;
  if (!sites?.length)
    return <p className="text-sm text-muted-foreground">No sites found.</p>;

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Name</TableHead>
          <TableHead>Domain</TableHead>
          <TableHead>Runtime</TableHead>
          <TableHead>Status</TableHead>
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
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
