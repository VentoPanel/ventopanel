"use client";

import { useState, useEffect } from "react";
import { toast } from "sonner";
import { type Site, type SiteInput } from "@/lib/api";
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
import { useCreateSite, useUpdateSite } from "@/hooks/use-site-mutations";

const RUNTIMES = ["php", "node"];

const defaultForm: SiteInput = {
  server_id: "",
  name: "",
  domain: "",
  runtime: "node",
  repository_url: "",
  branch: "main",
  status: "draft",
};

interface SiteFormProps {
  open: boolean;
  onOpenChange: (v: boolean) => void;
  site?: Site;
}

export function SiteForm({ open, onOpenChange, site }: SiteFormProps) {
  const isEdit = Boolean(site);
  const [form, setForm] = useState<SiteInput>(defaultForm);
  const { data: servers } = useServers();

  useEffect(() => {
    if (site) {
      setForm({
        server_id: site.ServerID,
        name: site.Name,
        domain: site.Domain,
        runtime: site.Runtime,
        repository_url: site.RepositoryURL,
        branch: site.Branch || "main",
        status: site.Status,
      });
    } else {
      setForm({
        ...defaultForm,
        server_id: servers?.[0]?.ID ?? "",
      });
    }
  }, [site, open, servers]);

  const create = useCreateSite();
  const update = useUpdateSite();
  const isPending = create.isPending || update.isPending;

  function set(field: keyof SiteInput, value: string) {
    setForm((f) => ({ ...f, [field]: value }));
  }

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
      <DialogContent>
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

          <div className="space-y-1.5">
            <Label htmlFor="si-branch">Branch</Label>
            <Input
              id="si-branch"
              placeholder="main"
              value={form.branch}
              onChange={(e) => set("branch", e.target.value)}
            />
            <p className="text-xs text-muted-foreground">
              Branch to deploy. Webhook auto-deploys only when this branch is pushed.
            </p>
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
