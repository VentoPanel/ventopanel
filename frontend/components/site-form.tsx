"use client";

import { useState, useEffect } from "react";
import { toast } from "sonner";
import { useQuery } from "@tanstack/react-query";
import { type Site, type SiteInput, type SiteTemplate, fetchTemplates } from "@/lib/api";
import { useServers } from "@/hooks/use-servers";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { useCreateSite, useUpdateSite } from "@/hooks/use-site-mutations";
import { cn } from "@/lib/utils";

const RUNTIMES = ["php", "node", "python", "go", "static"];

const defaultForm: SiteInput = {
  server_id: "",
  name: "",
  domain: "",
  runtime: "node",
  repository_url: "",
  branch: "main",
  healthcheck_path: "/",
  template_id: "",
  status: "draft",
};

interface SiteFormProps {
  open: boolean;
  onOpenChange: (v: boolean) => void;
  site?: Site;
}

const RUNTIME_COLOR: Record<string, string> = {
  node: "bg-green-100 text-green-700",
  python: "bg-blue-100 text-blue-700",
  php: "bg-purple-100 text-purple-700",
  go: "bg-cyan-100 text-cyan-700",
  static: "bg-gray-100 text-gray-700",
};

export function SiteForm({ open, onOpenChange, site }: SiteFormProps) {
  const isEdit = Boolean(site);
  const [form, setForm] = useState<SiteInput>(defaultForm);
  const [showTemplates, setShowTemplates] = useState(false);
  const { data: servers } = useServers();

  const { data: templates = [] } = useQuery<SiteTemplate[]>({
    queryKey: ["templates"],
    queryFn: fetchTemplates,
    staleTime: Infinity,
  });

  useEffect(() => {
    if (site) {
      setForm({
        server_id: site.ServerID,
        name: site.Name,
        domain: site.Domain,
        runtime: site.Runtime,
        repository_url: site.RepositoryURL,
        branch: site.Branch || "main",
        healthcheck_path: site.HealthcheckPath || "/",
        template_id: site.TemplateID || "",
        status: site.Status,
      });
    } else {
      setForm({
        ...defaultForm,
        server_id: servers?.[0]?.ID ?? "",
      });
    }
    setShowTemplates(false);
  }, [site, open, servers]);

  const create = useCreateSite();
  const update = useUpdateSite();
  const isPending = create.isPending || update.isPending;

  function set(field: keyof SiteInput, value: string) {
    setForm((f) => ({ ...f, [field]: value }));
  }

  function applyTemplate(t: SiteTemplate) {
    setForm((f) => ({
      ...f,
      template_id: t.id,
      runtime: t.runtime,
      healthcheck_path: t.healthcheck_path || "/",
    }));
    setShowTemplates(false);
  }

  function clearTemplate() {
    setForm((f) => ({ ...f, template_id: "" }));
  }

  const selectedTemplate = templates.find((t) => t.id === form.template_id);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    try {
      if (isEdit && site) {
        await update.mutateAsync({ id: site.ID, input: form });
        toast.success("Site updated");
      } else {
        await create.mutateAsync(form);
        toast.success("Site created");
      }
      onOpenChange(false);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Request failed");
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-xl">
        <DialogHeader>
          <DialogTitle>{isEdit ? "Edit Site" : "New Site"}</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4 pt-2">
          <div className="space-y-1.5">
            <Label htmlFor="si-server">Server</Label>
            <select
              id="si-server"
              required
              value={form.server_id}
              onChange={(e) => set("server_id", e.target.value)}
              className="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
            >
              <option value="">— select server —</option>
              {servers?.map((s) => (
                <option key={s.ID} value={s.ID}>
                  {s.Name} ({s.Host})
                </option>
              ))}
            </select>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label htmlFor="si-name">Name</Label>
              <Input
                id="si-name"
                required
                value={form.name}
                onChange={(e) => set("name", e.target.value)}
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="si-domain">Domain</Label>
              <Input
                id="si-domain"
                required
                placeholder="app.example.com"
                value={form.domain}
                onChange={(e) => set("domain", e.target.value)}
              />
            </div>
          </div>

          {/* Framework Template selector */}
          <div className="space-y-1.5">
            <div className="flex items-center justify-between">
              <Label>Framework Template</Label>
              {selectedTemplate ? (
                <button
                  type="button"
                  onClick={clearTemplate}
                  className="text-xs text-muted-foreground underline-offset-2 hover:underline"
                >
                  Clear
                </button>
              ) : (
                <button
                  type="button"
                  onClick={() => setShowTemplates((v) => !v)}
                  className="text-xs text-primary underline-offset-2 hover:underline"
                >
                  {showTemplates ? "Hide" : "Browse templates"}
                </button>
              )}
            </div>

            {/* Selected template badge */}
            {selectedTemplate && (
              <div className="flex items-center gap-2 rounded-md border border-primary/30 bg-primary/5 px-3 py-2 text-sm">
                <span
                  className={cn(
                    "rounded-full px-2 py-0.5 text-xs font-medium",
                    RUNTIME_COLOR[selectedTemplate.runtime] ?? "bg-muted text-muted-foreground",
                  )}
                >
                  {selectedTemplate.runtime}
                </span>
                <span className="font-medium">{selectedTemplate.name}</span>
                <span className="ml-auto text-xs text-muted-foreground">
                  Dockerfile will be managed by VentoPanel
                </span>
              </div>
            )}

            {/* Template grid */}
            {showTemplates && !selectedTemplate && (
              <div className="grid grid-cols-2 gap-2 rounded-md border bg-muted/30 p-2 max-h-64 overflow-y-auto">
                {templates.map((t) => (
                  <button
                    key={t.id}
                    type="button"
                    onClick={() => applyTemplate(t)}
                    className={cn(
                      "flex flex-col items-start gap-1 rounded-md border bg-background p-3 text-left text-sm transition-colors hover:border-primary hover:bg-primary/5",
                      form.template_id === t.id && "border-primary bg-primary/5",
                    )}
                  >
                    <div className="flex items-center gap-2 w-full">
                      <span
                        className={cn(
                          "rounded-full px-2 py-0.5 text-xs font-medium",
                          RUNTIME_COLOR[t.runtime] ?? "bg-muted text-muted-foreground",
                        )}
                      >
                        {t.runtime}
                      </span>
                      <span className="font-medium truncate">{t.name}</span>
                    </div>
                    <p className="text-xs text-muted-foreground line-clamp-2">{t.description}</p>
                  </button>
                ))}
                <button
                  type="button"
                  onClick={() => setShowTemplates(false)}
                  className="col-span-2 flex flex-col items-start gap-1 rounded-md border border-dashed bg-background p-3 text-left text-sm transition-colors hover:border-primary"
                >
                  <span className="font-medium text-muted-foreground">Auto-detect</span>
                  <p className="text-xs text-muted-foreground">
                    VentoPanel will detect your runtime from repo files (package.json, go.mod, etc.)
                  </p>
                </button>
              </div>
            )}
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label htmlFor="si-runtime">
                Runtime{" "}
                <span className="text-xs text-muted-foreground font-normal">(display only)</span>
              </Label>
              <select
                id="si-runtime"
                value={form.runtime}
                onChange={(e) => set("runtime", e.target.value)}
                className="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
              >
                {RUNTIMES.map((r) => (
                  <option key={r} value={r}>
                    {r}
                  </option>
                ))}
              </select>
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="si-repo">Repository URL</Label>
              <Input
                id="si-repo"
                placeholder="https://github.com/…"
                value={form.repository_url}
                onChange={(e) => set("repository_url", e.target.value)}
              />
              <p className="text-xs text-muted-foreground">
                Runtime auto-detected from repo files
              </p>
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label htmlFor="si-branch">Branch</Label>
              <Input
                id="si-branch"
                placeholder="main"
                value={form.branch}
                onChange={(e) => set("branch", e.target.value)}
              />
              <p className="text-xs text-muted-foreground">
                Webhook deploys only this branch.
              </p>
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="si-hcpath">Healthcheck Path</Label>
              <Input
                id="si-hcpath"
                placeholder="/"
                value={form.healthcheck_path ?? "/"}
                onChange={(e) => set("healthcheck_path", e.target.value)}
              />
              <p className="text-xs text-muted-foreground">
                URL path for uptime checks, e.g. <code>/health</code>
              </p>
            </div>
          </div>

          <div className="flex justify-end gap-2 pt-2">
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={isPending}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={isPending}>
              {isPending ? "Saving…" : isEdit ? "Save Changes" : "Create"}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  );
}
