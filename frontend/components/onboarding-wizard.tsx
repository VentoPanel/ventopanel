"use client";

import { useState, useCallback } from "react";
import { useRouter } from "next/navigation";
import {
  X, Server, Globe, HardDrive, TerminalSquare, ShieldCheck,
  CheckCircle2, Loader2, ArrowRight, ChevronRight, Wifi, WifiOff,
  KeyRound, Eye, EyeOff,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { cn } from "@/lib/utils";
import { createServer, connectServer, type ServerInput } from "@/lib/api";
import { toast } from "sonner";

// ─── Persistence ──────────────────────────────────────────────────────────────

const STORAGE_KEY = "vp_onboarding_done";
export function isOnboardingDone(): boolean {
  if (typeof window === "undefined") return true;
  return !!localStorage.getItem(STORAGE_KEY);
}
export function markOnboardingDone() {
  localStorage.setItem(STORAGE_KEY, "1");
}

// ─── Steps config ─────────────────────────────────────────────────────────────

const STEPS = ["welcome", "server", "connect", "done"] as const;
type Step = (typeof STEPS)[number];

// ─── Feature cards for welcome screen ────────────────────────────────────────

const FEATURES = [
  { icon: Server,        label: "Server Management",  desc: "Provision & monitor your VPS" },
  { icon: Globe,         label: "Site Deployment",    desc: "Deploy from Git in one click" },
  { icon: HardDrive,     label: "File Manager",       desc: "Browse & edit remote files" },
  { icon: TerminalSquare,label: "Web Terminal",       desc: "SSH in your browser" },
  { icon: ShieldCheck,   label: "Security",           desc: "2FA, API tokens, team access" },
];

// ─── Progress indicator ───────────────────────────────────────────────────────

function StepDots({ current }: { current: Step }) {
  const labels = ["Welcome", "Add Server", "Connect", "Done"];
  return (
    <div className="flex items-center justify-center gap-2">
      {STEPS.map((s, i) => (
        <div key={s} className="flex items-center gap-2">
          <div className={cn(
            "h-2 w-2 rounded-full transition-all",
            s === current ? "bg-primary w-6" : STEPS.indexOf(current) > i ? "bg-primary/60" : "bg-muted-foreground/30",
          )} />
        </div>
      ))}
    </div>
  );
}

// ─── Main component ───────────────────────────────────────────────────────────

interface Props {
  onDone: () => void;
}

export function OnboardingWizard({ onDone }: Props) {
  const router = useRouter();
  const [step, setStep]         = useState<Step>("welcome");
  const [busy, setBusy]         = useState(false);
  const [showPass, setShowPass] = useState(false);
  const [serverId, setServerId] = useState<string | null>(null);

  const [form, setForm] = useState<ServerInput>({
    name:     "",
    host:     "",
    port:     22,
    provider: "custom",
    ssh_user: "root",
    ssh_password: "",
  });

  function dismiss() { markOnboardingDone(); onDone(); }

  // ── Step: Add server ────────────────────────────────────────────────────────
  const handleAddServer = useCallback(async () => {
    if (!form.name.trim() || !form.host.trim()) {
      toast.error("Name and host are required");
      return;
    }
    setBusy(true);
    try {
      const srv = await createServer(form);
      setServerId(srv.ID);
      setStep("connect");
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "Failed to add server");
    } finally {
      setBusy(false);
    }
  }, [form]);

  // ── Step: Test connection ───────────────────────────────────────────────────
  const [connResult, setConnResult] = useState<"ok" | "fail" | null>(null);

  const handleConnect = useCallback(async () => {
    if (!serverId) return;
    setBusy(true);
    setConnResult(null);
    try {
      await connectServer(serverId);
      setConnResult("ok");
      setStep("done");
    } catch {
      setConnResult("fail");
    } finally {
      setBusy(false);
    }
  }, [serverId]);

  // ── Render ─────────────────────────────────────────────────────────────────
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-background/80 backdrop-blur-sm p-4">
      <div className="relative w-full max-w-lg rounded-2xl border bg-background shadow-2xl overflow-hidden">

        {/* Close */}
        <button
          onClick={dismiss}
          className="absolute right-4 top-4 z-10 rounded-md p-1 text-muted-foreground hover:bg-muted hover:text-foreground transition-colors"
          aria-label="Skip onboarding"
        >
          <X className="h-4 w-4" />
        </button>

        {/* Progress */}
        <div className="border-b px-6 py-3">
          <StepDots current={step} />
        </div>

        {/* ── WELCOME ── */}
        {step === "welcome" && (
          <div className="p-8 text-center">
            <div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-2xl bg-primary/10">
              <Server className="h-8 w-8 text-primary" />
            </div>
            <h2 className="text-2xl font-bold tracking-tight">Welcome to VentoPanel</h2>
            <p className="mt-2 text-sm text-muted-foreground">
              Your self-hosted server management panel. Let&apos;s connect your first server — it takes about 2 minutes.
            </p>

            <div className="mt-6 grid grid-cols-1 gap-2 text-left sm:grid-cols-2">
              {FEATURES.map(({ icon: Icon, label, desc }) => (
                <div key={label} className="flex items-start gap-3 rounded-lg border p-3">
                  <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-md bg-muted">
                    <Icon className="h-3.5 w-3.5 text-muted-foreground" />
                  </div>
                  <div>
                    <p className="text-xs font-semibold">{label}</p>
                    <p className="text-xs text-muted-foreground">{desc}</p>
                  </div>
                </div>
              ))}
            </div>

            <div className="mt-8 flex flex-col gap-2 sm:flex-row sm:justify-center">
              <Button onClick={() => setStep("server")} className="gap-2">
                Get Started <ArrowRight className="h-4 w-4" />
              </Button>
              <Button variant="ghost" onClick={dismiss} className="text-muted-foreground">
                I&apos;ll set up later
              </Button>
            </div>
          </div>
        )}

        {/* ── ADD SERVER ── */}
        {step === "server" && (
          <div className="p-8">
            <div className="mb-6">
              <h2 className="text-xl font-bold">Add Your First Server</h2>
              <p className="mt-1 text-sm text-muted-foreground">
                Enter your VPS credentials. VentoPanel connects over SSH — no agent required.
              </p>
            </div>

            <div className="space-y-4">
              <div className="grid grid-cols-2 gap-3">
                <div className="col-span-2 space-y-1.5">
                  <Label>Server name</Label>
                  <Input
                    placeholder="e.g. Production VPS"
                    value={form.name}
                    onChange={(e) => setForm({ ...form, name: e.target.value })}
                    autoFocus
                  />
                </div>
                <div className="col-span-2 space-y-1.5 sm:col-span-1">
                  <Label>Host / IP</Label>
                  <Input
                    placeholder="1.2.3.4 or server.com"
                    value={form.host}
                    onChange={(e) => setForm({ ...form, host: e.target.value })}
                  />
                </div>
                <div className="space-y-1.5">
                  <Label>SSH Port</Label>
                  <Input
                    type="number"
                    value={form.port}
                    onChange={(e) => setForm({ ...form, port: Number(e.target.value) })}
                  />
                </div>
                <div className="space-y-1.5">
                  <Label>SSH User</Label>
                  <Input
                    value={form.ssh_user}
                    onChange={(e) => setForm({ ...form, ssh_user: e.target.value })}
                  />
                </div>
                <div className="col-span-2 space-y-1.5 sm:col-span-1">
                  <Label>SSH Password</Label>
                  <div className="relative">
                    <Input
                      type={showPass ? "text" : "password"}
                      placeholder="••••••••"
                      value={form.ssh_password ?? ""}
                      onChange={(e) => setForm({ ...form, ssh_password: e.target.value })}
                      className="pr-9"
                    />
                    <button
                      type="button"
                      onClick={() => setShowPass((v) => !v)}
                      className="absolute right-2.5 top-2.5 text-muted-foreground hover:text-foreground"
                    >
                      {showPass ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                    </button>
                  </div>
                </div>
              </div>

              <p className="flex items-center gap-1.5 text-xs text-muted-foreground">
                <KeyRound className="h-3 w-3" />
                Password is encrypted with AES-GCM and never stored in plaintext.
              </p>
            </div>

            <div className="mt-8 flex justify-between">
              <Button variant="ghost" onClick={() => setStep("welcome")}>Back</Button>
              <Button onClick={handleAddServer} disabled={busy} className="gap-2">
                {busy ? <Loader2 className="h-4 w-4 animate-spin" /> : <ChevronRight className="h-4 w-4" />}
                {busy ? "Saving…" : "Save & Test Connection"}
              </Button>
            </div>
          </div>
        )}

        {/* ── CONNECT ── */}
        {step === "connect" && (
          <div className="p-8 text-center">
            <div className={cn(
              "mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-2xl transition-colors",
              busy ? "bg-blue-100 dark:bg-blue-950" : connResult === "fail" ? "bg-red-100 dark:bg-red-950" : "bg-muted",
            )}>
              {busy
                ? <Loader2 className="h-8 w-8 animate-spin text-blue-500" />
                : connResult === "fail"
                  ? <WifiOff className="h-8 w-8 text-red-500" />
                  : <Wifi className="h-8 w-8 text-muted-foreground" />
              }
            </div>

            <h2 className="text-xl font-bold">
              {busy ? "Connecting…" : connResult === "fail" ? "Connection Failed" : "Test Connection"}
            </h2>
            <p className="mt-2 text-sm text-muted-foreground">
              {busy
                ? "Opening SSH connection and verifying credentials…"
                : connResult === "fail"
                  ? "Could not connect. Check your IP, port, and credentials, then try again."
                  : "Click below to test the SSH connection to your server."}
            </p>

            <div className="mt-8 flex flex-col gap-2 sm:flex-row sm:justify-center">
              {!busy && (
                <>
                  <Button onClick={handleConnect} disabled={busy} className="gap-2">
                    <Wifi className="h-4 w-4" />
                    {connResult === "fail" ? "Retry" : "Test Connection"}
                  </Button>
                  <Button variant="ghost" onClick={() => setStep("done")} className="text-muted-foreground">
                    Skip for now
                  </Button>
                </>
              )}
            </div>
          </div>
        )}

        {/* ── DONE ── */}
        {step === "done" && (
          <div className="p-8 text-center">
            <div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-2xl bg-green-100 dark:bg-green-950">
              <CheckCircle2 className="h-8 w-8 text-green-500" />
            </div>
            <h2 className="text-xl font-bold">You&apos;re all set!</h2>
            <p className="mt-2 text-sm text-muted-foreground">
              {connResult === "ok"
                ? "Server connected successfully. What would you like to do next?"
                : "Server saved. You can connect and provision it from the Servers page."}
            </p>

            <div className="mt-8 grid gap-3">
              <Button
                variant="outline"
                className="justify-between h-14 px-5"
                onClick={() => { dismiss(); router.push("/sites"); }}
              >
                <div className="flex items-center gap-3">
                  <Globe className="h-5 w-5 text-muted-foreground" />
                  <div className="text-left">
                    <p className="font-medium text-sm">Deploy a Site</p>
                    <p className="text-xs text-muted-foreground">Connect a Git repo and go live</p>
                  </div>
                </div>
                <ChevronRight className="h-4 w-4 text-muted-foreground" />
              </Button>

              <Button
                variant="outline"
                className="justify-between h-14 px-5"
                onClick={() => { dismiss(); router.push("/files"); }}
              >
                <div className="flex items-center gap-3">
                  <HardDrive className="h-5 w-5 text-muted-foreground" />
                  <div className="text-left">
                    <p className="font-medium text-sm">Browse Files</p>
                    <p className="text-xs text-muted-foreground">Open the file manager for this server</p>
                  </div>
                </div>
                <ChevronRight className="h-4 w-4 text-muted-foreground" />
              </Button>

              <Button
                variant="outline"
                className="justify-between h-14 px-5"
                onClick={() => { dismiss(); router.push("/terminal"); }}
              >
                <div className="flex items-center gap-3">
                  <TerminalSquare className="h-5 w-5 text-muted-foreground" />
                  <div className="text-left">
                    <p className="font-medium text-sm">Open Terminal</p>
                    <p className="text-xs text-muted-foreground">SSH session in your browser</p>
                  </div>
                </div>
                <ChevronRight className="h-4 w-4 text-muted-foreground" />
              </Button>
            </div>

            <Button variant="ghost" className="mt-4 text-muted-foreground" onClick={dismiss}>
              Explore the panel on my own
            </Button>
          </div>
        )}
      </div>
    </div>
  );
}
