"use client";

import { useState, useEffect } from "react";
import { toast } from "sonner";
import { type Server, type ServerInput } from "@/lib/api";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useCreateServer, useUpdateServer } from "@/hooks/use-server-mutations";

const PROVIDERS = ["hetzner", "digitalocean", "aws", "linode", "custom"];

const defaultForm: ServerInput = {
  name: "",
  host: "",
  port: 22,
  provider: "hetzner",
  ssh_user: "root",
  ssh_password: "",
  status: "pending",
};

interface ServerFormProps {
  open: boolean;
  onOpenChange: (v: boolean) => void;
  server?: Server;
}

export function ServerForm({ open, onOpenChange, server }: ServerFormProps) {
  const isEdit = Boolean(server);
  const [form, setForm] = useState<ServerInput>(defaultForm);

  useEffect(() => {
    if (server) {
      setForm({
        name: server.Name,
        host: server.Host,
        port: server.Port,
        provider: server.Provider,
        ssh_user: server.SSHUser,
        ssh_password: "",
        status: server.Status,
      });
    } else {
      setForm(defaultForm);
    }
  }, [server, open]);

  const create = useCreateServer();
  const update = useUpdateServer();
  const isPending = create.isPending || update.isPending;

  function set(field: keyof ServerInput, value: string | number) {
    setForm((f) => ({ ...f, [field]: value }));
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    try {
      if (isEdit && server) {
        await update.mutateAsync({ id: server.ID, input: form });
        toast.success("Server updated");
      } else {
        await create.mutateAsync(form);
        toast.success("Server created");
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
          <DialogTitle>{isEdit ? "Edit Server" : "New Server"}</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4 pt-2">
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label htmlFor="s-name">Name</Label>
              <Input
                id="s-name"
                required
                value={form.name}
                onChange={(e) => set("name", e.target.value)}
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="s-host">Host / IP</Label>
              <Input
                id="s-host"
                required
                value={form.host}
                onChange={(e) => set("host", e.target.value)}
              />
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label htmlFor="s-port">SSH Port</Label>
              <Input
                id="s-port"
                type="number"
                min={1}
                max={65535}
                required
                value={form.port}
                onChange={(e) => set("port", Number(e.target.value))}
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="s-provider">Provider</Label>
              <select
                id="s-provider"
                value={form.provider}
                onChange={(e) => set("provider", e.target.value)}
                className="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
              >
                {PROVIDERS.map((p) => (
                  <option key={p} value={p}>
                    {p}
                  </option>
                ))}
              </select>
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label htmlFor="s-sshuser">SSH User</Label>
              <Input
                id="s-sshuser"
                required
                value={form.ssh_user}
                onChange={(e) => set("ssh_user", e.target.value)}
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="s-sshpass">
                SSH Password{isEdit && " (leave blank to keep)"}
              </Label>
              <Input
                id="s-sshpass"
                type="password"
                required={!isEdit}
                value={form.ssh_password}
                onChange={(e) => set("ssh_password", e.target.value)}
              />
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
