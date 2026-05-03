"use client";

import { useState, useEffect, useCallback } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import dynamic from "next/dynamic";
import {
  Globe, RefreshCw, CheckCircle, XCircle, Plus,
  Trash2, Power, PowerOff, RotateCcw, ShieldCheck,
  AlertCircle, ChevronDown, Loader2, Save, FlaskConical,
} from "lucide-react";
import { toast } from "sonner";

import { fetchServers, fetchNginxVhosts, fetchNginxStatus, fetchNginxVhost,
  saveNginxVhost, createNginxVhost, deleteNginxVhost,
  enableNginxVhost, disableNginxVhost, testNginxConfig,
  reloadNginx, issueNginxCert,
  type NginxVhost, type NginxStatus } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

const MonacoEditor = dynamic(() => import("@monaco-editor/react"), { ssr: false });

// ─── Helpers ──────────────────────────────────────────────────────────────────

function StatusBadge({ active }: { active: boolean }) {
  return (
    <span className={cn(
      "inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium",
      active ? "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400"
             : "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
    )}>
      {active ? <CheckCircle className="h-3 w-3" /> : <XCircle className="h-3 w-3" />}
      {active ? "active" : "inactive"}
    </span>
  );
}

// ─── Create Vhost Modal ───────────────────────────────────────────────────────

function CreateModal({
  serverId,
  onClose,
}: {
  serverId: string;
  onClose: () => void;
}) {
  const qc = useQueryClient();
  const [name, setName] = useState("");
  const mut = useMutation({
    mutationFn: () => createNginxVhost(serverId, name),
    onSuccess: () => {
      toast.success(`Vhost "${name}" created`);
      qc.invalidateQueries({ queryKey: ["nginx-vhosts", serverId] });
      onClose();
    },
    onError: (e: Error) => toast.error(e.message),
  });
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className="w-full max-w-sm rounded-lg border bg-background p-6 shadow-xl">
        <h2 className="mb-4 text-lg font-semibold">New Virtual Host</h2>
        <label className="mb-1 block text-sm font-medium text-muted-foreground">Server name / domain</label>
        <input
          autoFocus
          className="mb-4 w-full rounded-md border bg-muted/30 px-3 py-2 text-sm outline-none focus:ring-2 focus:ring-primary"
          placeholder="example.com"
          value={name}
          onChange={e => setName(e.target.value)}
          onKeyDown={e => e.key === "Enter" && name && mut.mutate()}
        />
        <div className="flex justify-end gap-2">
          <Button variant="ghost" size="sm" onClick={onClose}>Cancel</Button>
          <Button size="sm" disabled={!name || mut.isPending} onClick={() => mut.mutate()}>
            {mut.isPending && <Loader2 className="mr-1.5 h-3.5 w-3.5 animate-spin" />}
            Create
          </Button>
        </div>
      </div>
    </div>
  );
}

// ─── SSL Modal ────────────────────────────────────────────────────────────────

function SSLModal({
  serverId,
  vhostName,
  onClose,
}: {
  serverId: string;
  vhostName: string;
  onClose: () => void;
}) {
  const [domain, setDomain] = useState(vhostName);
  const [email, setEmail]   = useState("");
  const [log, setLog]       = useState("");
  const mut = useMutation({
    mutationFn: () => issueNginxCert(serverId, vhostName, domain, email || undefined),
    onSuccess: (data) => {
      setLog(data.output);
      if (data.success) toast.success("SSL certificate issued successfully");
      else toast.error("Certbot finished with errors — check output");
    },
    onError: (e: Error) => { setLog(e.message); toast.error(e.message); },
  });
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className="w-full max-w-lg rounded-lg border bg-background p-6 shadow-xl">
        <h2 className="mb-4 text-lg font-semibold">Issue SSL Certificate</h2>
        <label className="mb-1 block text-sm font-medium text-muted-foreground">Domain</label>
        <input
          className="mb-3 w-full rounded-md border bg-muted/30 px-3 py-2 text-sm outline-none focus:ring-2 focus:ring-primary"
          value={domain}
          onChange={e => setDomain(e.target.value)}
        />
        <label className="mb-1 block text-sm font-medium text-muted-foreground">Email (optional)</label>
        <input
          className="mb-4 w-full rounded-md border bg-muted/30 px-3 py-2 text-sm outline-none focus:ring-2 focus:ring-primary"
          placeholder="admin@example.com"
          value={email}
          onChange={e => setEmail(e.target.value)}
        />
        {log && (
          <pre className="mb-4 max-h-48 overflow-auto rounded-md bg-muted p-3 text-xs text-muted-foreground whitespace-pre-wrap">
            {log}
          </pre>
        )}
        <div className="flex justify-end gap-2">
          <Button variant="ghost" size="sm" onClick={onClose}>Close</Button>
          <Button size="sm" disabled={!domain || mut.isPending} onClick={() => mut.mutate()}>
            {mut.isPending ? <Loader2 className="mr-1.5 h-3.5 w-3.5 animate-spin" /> : <ShieldCheck className="mr-1.5 h-3.5 w-3.5" />}
            Issue Certificate
          </Button>
        </div>
      </div>
    </div>
  );
}

// ─── Config Editor Modal ──────────────────────────────────────────────────────

function EditorModal({
  serverId,
  vhost,
  onClose,
}: {
  serverId: string;
  vhost: NginxVhost;
  onClose: () => void;
}) {
  const qc = useQueryClient();
  const [content, setContent] = useState<string | undefined>(undefined);
  const [testOutput, setTestOutput] = useState<string | null>(null);
  const [testOk, setTestOk] = useState<boolean | null>(null);

  // Load config
  const { isLoading, data: vhostData } = useQuery({
    queryKey: ["nginx-vhost-content", serverId, vhost.name],
    queryFn: () => fetchNginxVhost(serverId, vhost.name),
  });

  useEffect(() => {
    if (vhostData) setContent(vhostData.content);
  }, [vhostData]);

  const saveMut = useMutation({
    mutationFn: () => saveNginxVhost(serverId, vhost.name, content ?? ""),
    onSuccess: (data) => {
      setTestOk(data.test_ok);
      setTestOutput(data.test_output);
      if (data.test_ok) {
        toast.success("Config saved — nginx test passed");
        qc.invalidateQueries({ queryKey: ["nginx-vhosts", serverId] });
      } else {
        toast.error("Config saved but nginx test failed");
      }
    },
    onError: (e: Error) => toast.error(e.message),
  });

  return (
    <div className="fixed inset-0 z-50 flex flex-col bg-background">
      {/* Header */}
      <div className="flex items-center justify-between border-b px-4 py-3">
        <div className="flex items-center gap-2">
          <Globe className="h-4 w-4 text-muted-foreground" />
          <span className="font-mono text-sm font-medium">{vhost.path}</span>
          {testOk === true  && <span className="text-xs text-green-600">✓ nginx test ok</span>}
          {testOk === false && <span className="text-xs text-red-500">✗ nginx test failed</span>}
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={onClose}>Close</Button>
          <Button size="sm" disabled={saveMut.isPending || isLoading} onClick={() => saveMut.mutate()}>
            {saveMut.isPending ? <Loader2 className="mr-1.5 h-3.5 w-3.5 animate-spin" /> : <Save className="mr-1.5 h-3.5 w-3.5" />}
            Save & Test
          </Button>
        </div>
      </div>

      {/* Editor */}
      <div className="flex flex-1 overflow-hidden">
        <div className="flex-1">
          {isLoading ? (
            <div className="flex h-full items-center justify-center text-muted-foreground">
              <Loader2 className="h-5 w-5 animate-spin" />
            </div>
          ) : (
            <MonacoEditor
              height="100%"
              language="nginx"
              theme="vs-dark"
              value={content}
              onChange={v => setContent(v ?? "")}
              options={{
                minimap: { enabled: false },
                fontSize: 13,
                lineNumbers: "on",
                wordWrap: "on",
                scrollBeyondLastLine: false,
              }}
            />
          )}
        </div>

        {/* Test output sidebar */}
        {testOutput && (
          <div className="w-80 border-l bg-muted/30 p-4 overflow-auto">
            <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-muted-foreground">nginx -t output</p>
            <pre className="whitespace-pre-wrap text-xs">{testOutput}</pre>
          </div>
        )}
      </div>
    </div>
  );
}

// ─── Main Page ────────────────────────────────────────────────────────────────

export default function NginxPage() {
  const qc = useQueryClient();
  const [serverId, setServerId] = useState<string>("");
  const [editing, setEditing]   = useState<NginxVhost | null>(null);
  const [sslTarget, setSslTarget] = useState<NginxVhost | null>(null);
  const [showCreate, setShowCreate] = useState(false);

  const { data: servers = [] } = useQuery({
    queryKey: ["servers"],
    queryFn: fetchServers,
  });

  // Auto-select first server
  useEffect(() => {
    if (!serverId && servers.length > 0) setServerId(servers[0].ID);
  }, [servers, serverId]);

  const { data: status, refetch: refetchStatus } = useQuery<NginxStatus>({
    queryKey: ["nginx-status", serverId],
    queryFn: () => fetchNginxStatus(serverId),
    enabled: !!serverId,
    refetchInterval: 30_000,
  });

  const { data: vhosts = [], isFetching: loadingVhosts, refetch: refetchVhosts } = useQuery<NginxVhost[]>({
    queryKey: ["nginx-vhosts", serverId],
    queryFn: () => fetchNginxVhosts(serverId),
    enabled: !!serverId,
  });

  const refresh = useCallback(() => {
    refetchStatus();
    refetchVhosts();
  }, [refetchStatus, refetchVhosts]);

  // Mutations
  const enableMut = useMutation({
    mutationFn: (name: string) => enableNginxVhost(serverId, name),
    onSuccess: (data, name) => {
      toast.success(data.test_ok ? `"${name}" enabled` : `"${name}" enabled (config warning)`);
      qc.invalidateQueries({ queryKey: ["nginx-vhosts", serverId] });
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const disableMut = useMutation({
    mutationFn: (name: string) => disableNginxVhost(serverId, name),
    onSuccess: (_data, name) => {
      toast.success(`"${name}" disabled`);
      qc.invalidateQueries({ queryKey: ["nginx-vhosts", serverId] });
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const deleteMut = useMutation({
    mutationFn: (name: string) => deleteNginxVhost(serverId, name),
    onSuccess: (_data, name) => {
      toast.success(`"${name}" deleted`);
      qc.invalidateQueries({ queryKey: ["nginx-vhosts", serverId] });
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const testMut = useMutation({
    mutationFn: () => testNginxConfig(serverId),
    onSuccess: (data) => {
      if (data.ok) toast.success("nginx -t: OK");
      else toast.error("nginx -t failed:\n" + data.output);
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const reloadMut = useMutation({
    mutationFn: () => reloadNginx(serverId),
    onSuccess: () => { toast.success("nginx reloaded"); refetchStatus(); },
    onError: (e: Error) => toast.error(e.message),
  });

  return (
    <>
      <div className="flex h-full flex-col gap-6 p-6">
        {/* Header */}
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div className="flex items-center gap-2">
            <Globe className="h-5 w-5 text-primary" />
            <h1 className="text-xl font-bold">Nginx Manager</h1>
          </div>

          <div className="flex flex-wrap items-center gap-2">
            {/* Server selector */}
            <div className="relative">
              <select
                className="appearance-none rounded-md border bg-background px-3 py-1.5 pr-8 text-sm outline-none focus:ring-2 focus:ring-primary"
                value={serverId}
                onChange={e => setServerId(e.target.value)}
              >
                {servers.map((s: Server) => (
                  <option key={s.ID} value={s.ID}>{s.Name} ({s.Host})</option>
                ))}
              </select>
              <ChevronDown className="pointer-events-none absolute right-2 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
            </div>

            <Button variant="outline" size="sm" onClick={refresh} disabled={loadingVhosts}>
              <RefreshCw className={cn("h-3.5 w-3.5", loadingVhosts && "animate-spin")} />
            </Button>
            <Button variant="outline" size="sm" disabled={!serverId || testMut.isPending} onClick={() => testMut.mutate()}>
              {testMut.isPending ? <Loader2 className="mr-1.5 h-3.5 w-3.5 animate-spin" /> : <FlaskConical className="mr-1.5 h-3.5 w-3.5" />}
              Test Config
            </Button>
            <Button variant="outline" size="sm" disabled={!serverId || reloadMut.isPending} onClick={() => reloadMut.mutate()}>
              {reloadMut.isPending ? <Loader2 className="mr-1.5 h-3.5 w-3.5 animate-spin" /> : <RotateCcw className="mr-1.5 h-3.5 w-3.5" />}
              Reload
            </Button>
            <Button size="sm" disabled={!serverId} onClick={() => setShowCreate(true)}>
              <Plus className="mr-1.5 h-3.5 w-3.5" />
              New Vhost
            </Button>
          </div>
        </div>

        {/* Status bar */}
        {status && (
          <div className="flex flex-wrap items-center gap-4 rounded-lg border bg-muted/30 px-4 py-3 text-sm">
            <span className="flex items-center gap-2 font-medium">
              nginx status: <StatusBadge active={status.active} />
            </span>
            <span className="text-muted-foreground">{status.version}</span>
            {status.config_test === "ok" ? (
              <span className="flex items-center gap-1 text-green-600">
                <CheckCircle className="h-3.5 w-3.5" /> config OK
              </span>
            ) : (
              <span className="flex items-center gap-1 text-red-500">
                <AlertCircle className="h-3.5 w-3.5" /> config error
              </span>
            )}
          </div>
        )}

        {/* Vhosts table */}
        {!serverId ? (
          <div className="flex flex-1 items-center justify-center text-muted-foreground">
            Select a server to manage nginx
          </div>
        ) : vhosts.length === 0 && !loadingVhosts ? (
          <div className="flex flex-1 flex-col items-center justify-center gap-3 text-muted-foreground">
            <Globe className="h-10 w-10 opacity-20" />
            <p>No virtual hosts found in /etc/nginx/sites-available</p>
            <Button size="sm" onClick={() => setShowCreate(true)}>
              <Plus className="mr-1.5 h-3.5 w-3.5" /> Create first vhost
            </Button>
          </div>
        ) : (
          <div className="overflow-x-auto rounded-lg border">
            <table className="w-full min-w-[560px] text-sm">
              <thead className="bg-muted/50">
                <tr>
                  <th className="px-4 py-2.5 text-left font-medium text-muted-foreground">Name</th>
                  <th className="px-4 py-2.5 text-left font-medium text-muted-foreground">Status</th>
                  <th className="px-4 py-2.5 text-left font-medium text-muted-foreground">Path</th>
                  <th className="px-4 py-2.5 text-right font-medium text-muted-foreground">Actions</th>
                </tr>
              </thead>
              <tbody>
                {vhosts.map((v: NginxVhost) => (
                  <tr key={v.name} className="border-t transition-colors hover:bg-muted/30">
                    <td className="px-4 py-3 font-mono font-medium">{v.name}</td>
                    <td className="px-4 py-3">
                      <StatusBadge active={v.enabled} />
                    </td>
                    <td className="px-4 py-3 font-mono text-xs text-muted-foreground">{v.path}</td>
                    <td className="px-4 py-3">
                      <div className="flex items-center justify-end gap-1">
                        {/* Edit */}
                        <Button
                          variant="ghost" size="sm"
                          className="h-7 px-2 text-xs"
                          onClick={() => setEditing(v)}
                        >
                          Edit
                        </Button>
                        {/* Enable / Disable */}
                        {v.enabled ? (
                          <Button
                            variant="ghost" size="sm"
                            className="h-7 px-2 text-xs text-amber-600 hover:text-amber-700"
                            disabled={disableMut.isPending}
                            onClick={() => disableMut.mutate(v.name)}
                          >
                            <PowerOff className="mr-1 h-3 w-3" /> Disable
                          </Button>
                        ) : (
                          <Button
                            variant="ghost" size="sm"
                            className="h-7 px-2 text-xs text-green-600 hover:text-green-700"
                            disabled={enableMut.isPending}
                            onClick={() => enableMut.mutate(v.name)}
                          >
                            <Power className="mr-1 h-3 w-3" /> Enable
                          </Button>
                        )}
                        {/* SSL */}
                        <Button
                          variant="ghost" size="sm"
                          className="h-7 px-2 text-xs text-blue-600 hover:text-blue-700"
                          onClick={() => setSslTarget(v)}
                        >
                          <ShieldCheck className="mr-1 h-3 w-3" /> SSL
                        </Button>
                        {/* Delete */}
                        <Button
                          variant="ghost" size="sm"
                          className="h-7 px-2 text-xs text-destructive hover:text-destructive"
                          disabled={deleteMut.isPending}
                          onClick={() => {
                            if (confirm(`Delete vhost "${v.name}"?`)) deleteMut.mutate(v.name);
                          }}
                        >
                          <Trash2 className="h-3 w-3" />
                        </Button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Modals */}
      {showCreate && <CreateModal serverId={serverId} onClose={() => setShowCreate(false)} />}
      {sslTarget   && <SSLModal serverId={serverId} vhostName={sslTarget.name} onClose={() => setSslTarget(null)} />}
      {editing     && <EditorModal serverId={serverId} vhost={editing} onClose={() => setEditing(null)} />}
    </>
  );
}
