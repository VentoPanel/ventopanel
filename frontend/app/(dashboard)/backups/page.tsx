"use client";

import { useState } from "react";
import { DatabaseBackup, Download, RefreshCw, Play } from "lucide-react";
import { formatDistanceToNow } from "date-fns";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { fetchBackups, triggerBackup, downloadBackup, type BackupMeta } from "@/lib/api";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { toast } from "sonner";

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`;
}

export default function BackupsPage() {
  const qc = useQueryClient();
  const [downloading, setDownloading] = useState<string | null>(null);

  const { data: backups = [], isFetching } = useQuery<BackupMeta[]>({
    queryKey: ["backups"],
    queryFn: fetchBackups,
    refetchInterval: 30_000,
  });

  const { mutate: trigger, isPending: triggering } = useMutation({
    mutationFn: triggerBackup,
    onSuccess: () => {
      toast.success("Backup created successfully");
      qc.invalidateQueries({ queryKey: ["backups"] });
    },
    onError: (err) => {
      toast.error(err instanceof Error ? err.message : "Backup failed");
    },
  });

  async function handleDownload(name: string) {
    setDownloading(name);
    try {
      await downloadBackup(name);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Download failed");
    } finally {
      setDownloading(null);
    }
  }

  return (
    <div className="space-y-6 max-w-3xl">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <DatabaseBackup className="h-6 w-6 text-muted-foreground" />
          <div>
            <h2 className="text-2xl font-bold tracking-tight">Backups</h2>
            <p className="text-muted-foreground text-sm">
              PostgreSQL dumps · auto-runs daily · keeps last {backups.length > 0 ? "7" : "7"} files
            </p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          {isFetching && <RefreshCw className="h-4 w-4 animate-spin text-muted-foreground" />}
          <Button
            size="sm"
            onClick={() => trigger()}
            disabled={triggering}
          >
            <Play className="mr-1.5 h-3.5 w-3.5" />
            {triggering ? "Running…" : "Run now"}
          </Button>
        </div>
      </div>

      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-base">Backup archives</CardTitle>
          <CardDescription>
            Each archive is a <code>.tar.gz</code> containing one CSV per table.
            Stored in the <code>/data/backups</code> volume on the server.
          </CardDescription>
        </CardHeader>
        <CardContent className="p-0">
          {backups.length === 0 ? (
            <p className="px-6 py-4 text-sm text-muted-foreground">
              No backups yet. Click "Run now" to create the first one, or wait for the daily scheduler
              (runs 5 minutes after API startup, then every 24 hours).
            </p>
          ) : (
            <div className="divide-y">
              {backups.map((b) => (
                <div
                  key={b.name}
                  className="flex items-center gap-4 px-6 py-3 hover:bg-muted/40 transition-colors"
                >
                  <div className="min-w-0 flex-1">
                    <p className="truncate font-mono text-sm">{b.name}</p>
                    <p className="text-xs text-muted-foreground mt-0.5">
                      {formatBytes(b.size_bytes)} ·{" "}
                      {formatDistanceToNow(new Date(b.created_at), { addSuffix: true })}
                    </p>
                  </div>
                  <Button
                    variant="ghost"
                    size="sm"
                    disabled={downloading === b.name}
                    onClick={() => handleDownload(b.name)}
                    title="Download"
                  >
                    {downloading === b.name ? (
                      <RefreshCw className="h-4 w-4 animate-spin" />
                    ) : (
                      <Download className="h-4 w-4" />
                    )}
                  </Button>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
