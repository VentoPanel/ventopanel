import { Skeleton } from "@/components/ui/skeleton";
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table";

interface Props {
  /** Number of columns to render in each skeleton row. */
  cols: number;
  /** Number of skeleton rows to render. Defaults to 5. */
  rows?: number;
  /** Optional column header labels (just visual hints). */
  headers?: string[];
}

/**
 * Drop-in skeleton replacement for any data table while its query is loading.
 * Usage:
 *   if (isLoading) return <TableSkeleton cols={5} headers={["Name","Host","Status","…"]} />;
 */
export function TableSkeleton({ cols, rows = 5, headers }: Props) {
  const colWidths = [
    "w-1/3", "w-1/4", "w-1/5", "w-1/6", "w-1/6",
    "w-1/5", "w-1/4", "w-1/3",
  ];

  return (
    <div className="rounded-md border overflow-hidden">
      <Table>
        {headers && (
          <TableHeader>
            <TableRow className="bg-muted/40">
              {headers.map((h) => (
                <TableHead key={h} className="text-xs font-medium text-muted-foreground">
                  {h}
                </TableHead>
              ))}
              {/* fill remaining cols with empty headers */}
              {Array.from({ length: Math.max(0, cols - headers.length) }).map((_, i) => (
                <TableHead key={`empty-${i}`} />
              ))}
            </TableRow>
          </TableHeader>
        )}
        <TableBody>
          {Array.from({ length: rows }).map((_, ri) => (
            <TableRow key={ri} className="animate-pulse">
              {Array.from({ length: cols }).map((_, ci) => (
                <TableCell key={ci} className="py-3">
                  <Skeleton className={`h-4 ${colWidths[ci % colWidths.length]}`} />
                </TableCell>
              ))}
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}
