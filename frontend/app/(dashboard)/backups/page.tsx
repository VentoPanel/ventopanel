"use client";

import { useState, useEffect } from "react";
import { DatabaseBackup, Download, RefreshCw, Play, Settings2, Save } from "lucide-react";
import { formatDistanceToNow } from "date-fns";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  fetchBackups,
  triggerBackup,
  downloadBackup,
  fetchBackupSettings,
  updateBackupSettings,
  type BackupMeta,
  type BackupSettings,
} from "@/lib/api";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { toast } from "sonner";
import { cn } from "@/lib/utils";

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
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [form, setForm] = useState<BackupSettings>({
    auto_enabled: true,
    retention_count: 7,
    notify_success: false,
  });
  const [savingSettings, setSavingSettings] = useState(false);

  const { data: backups = [], isFetching } = useQuery<BackupMeta[]>({
    queryKey: ["backups"],
    queryFn: fetchBackups,
    refetchInterval: 30_000,
  });

  const { data: backupSettings } = useQuery<BackupSettings>({
    queryKey: ["backup-settings"],
    queryFn: fetchBackupSettings,
    staleTime: 60_000,
  });

  useEffect(() => {
    if (backupSettings) setForm(backupSettings);
  }, [backupSettings]);

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

  async function handleSaveSettings(e: React.FormEvent) {
    e.preventDefault();
    setSavingSettings(true);
    try {
      await updateBackupSettings(form);
      toast.success("Backup settings saved");
      qc.invalidateQueries({ queryKey: ["backup-settings"] });
      setSettingsOpen(false);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to save settings");
    } finally {
      setSavingSettings(false);
    }
  }

  const retentionCount = backupSettings?.retention_count ?? 7;

  return (
    <div className="space-y-6 max-w-3xl">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <DatabaseBackup className="h-6 w-6 text-muted-foreground" />
          <div>
            <h2 className="text-2xl font-bold tracking-tight">Backups</h2>
            <p className="text-muted-foreground text-sm">
              PostgreSQL dumps · {backupSettings?.auto_enabled ? "auto daily" : "manual only"} · keeps last {retentionCount} files
            </p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          {isFetching && <RefreshCw className="h-4 w-4 animate-spin text-muted-foreground" />}
          <Button
            variant="outline"
            size="sm"
            onClick={() => setSettingsOpen((v) => !v)}
          >
            <Settings2 className="mr-1.5 h-3.5 w-3.5" />
            Schedule
          </Button>
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

      {/* Schedule settings panel */}
      {settingsOpen && (
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="flex items-center gap-2 text-base">
              <Settings2 className="h-4 w-4" />
              Backup Schedule
            </CardTitle>
            <CardDescription>
              Configure automatic daily backups. Changes take effect immediately.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleSaveSettings} className="space-y-4">
              <label className="flex items-center gap-3 cursor-pointer">
                <input
                  type="checkbox"
                  className="h-4 w-4 rounded"
                  checked={form.auto_enabled}
                  onChange={(e) => setForm((f) => ({ ...f, auto_enabled: e.target.checked }))}
                />
                <div>
                  <p className="text-sm font-medium">Enable automatic daily backups</p>
                  <p className="text-xs text-muted-foreground">Runs every 24 hours. Disable to run backups manually only.</p>
                </div>
              </label>

              <div className="space-y-1.5">
                <label className="text-sm font-medium" htmlFor="retention">
                  Keep last <strong>{form.retention_count}</strong> backups
                </label>
                <input
                  id="retention"
                  type="range"
                  min={1}
                  max={30}
                  value={form.retention_count}
                  onChange={(e) => setForm((f) => ({ ...f, retention_count: Number(e.target.value) }))}
                  className="w-full"
                />
                <div className="flex justify-between text-xs text-muted-foreground">
                  <span>1</span>
                  <span>30</span>
                </div>
              </div>

              <label className="flex items-center gap-3 cursor-pointer">
                <input
                  type="checkbox"
                  className="h-4 w-4 rounded"
                  checked={form.notify_success}
                  onChange={(e) => setForm((f) => ({ ...f, notify_success: e.target.checked }))}
                />
                <div>
                  <p className="text-sm font-medium">Notify on successful backup</p>
                  <p className="text-xs text-muted-foreground">Failed backups always send a notification.</p>
                </div>
              </label>

              <div className="flex gap-2 pt-2">
                <Button type="submit" size="sm" disabled={savingSettings}>
                  <Save className="mr-1.5 h-3.5 w-3.5" />
                  {savingSettings ? "Saving…" : "Save"}
                </Button>
                <Button type="button" variant="outline" size="sm" onClick={() => setSettingsOpen(false)}>
                  Cancel
                </Button>
              </div>
            </form>
          </CardContent>
        </Card>
      )}

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
              No backups yet. Click "Run now" to create the first one
              {backupSettings?.auto_enabled
                ? ", or wait for the daily scheduler."
                : ". Auto-scheduling is disabled."}
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
