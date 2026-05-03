"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import {
  Eye, EyeOff, Loader2, Smartphone, ArrowLeft,
  Server, Shield, Zap, Globe,
} from "lucide-react";
import { login, isMFARequired, verifyMFA, registerUser, setToken } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { cn } from "@/lib/utils";

type Tab = "login" | "register";

// ─── Feature list shown on the left panel ────────────────────────────────────

const FEATURES = [
  { icon: Server,  text: "Manage unlimited servers via SSH" },
  { icon: Globe,   text: "Deploy & monitor sites in one click" },
  { icon: Shield,  text: "2FA, audit logs, team access control" },
  { icon: Zap,     text: "Real-time metrics, logs & terminal" },
];

export default function LoginPage() {
  const router = useRouter();
  const [tab, setTab] = useState<Tab>("login");

  const [email, setEmail]       = useState("");
  const [password, setPassword] = useState("");
  const [showPwd, setShowPwd]   = useState(false);

  const [mfaSession, setMfaSession] = useState<string | null>(null);
  const [mfaCode, setMfaCode]       = useState("");

  const [regEmail, setRegEmail]       = useState("");
  const [regPassword, setRegPassword] = useState("");
  const [regTeamID, setRegTeamID]     = useState("");
  const [showRegPwd, setShowRegPwd]   = useState(false);

  const [loading, setLoading]   = useState(false);
  const [error, setError]       = useState("");
  const [success, setSuccess]   = useState("");

  async function handleLogin(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      const res = await login(email.trim(), password);
      if (isMFARequired(res)) {
        setMfaSession(res.mfa_session);
      } else {
        setToken(res.token);
        router.push("/");
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Login failed");
    } finally {
      setLoading(false);
    }
  }

  async function handleMFA(e: React.FormEvent) {
    e.preventDefault();
    if (!mfaSession) return;
    setError("");
    setLoading(true);
    try {
      const res = await verifyMFA(mfaSession, mfaCode);
      setToken(res.token);
      router.push("/");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Invalid code");
    } finally {
      setLoading(false);
    }
  }

  async function handleRegister(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setSuccess("");
    setLoading(true);
    try {
      await registerUser({ email: regEmail.trim(), password: regPassword, team_id: regTeamID.trim() });
      setSuccess("Account created. You can now log in.");
      setTab("login");
      setEmail(regEmail.trim());
    } catch (err) {
      setError(err instanceof Error ? err.message : "Registration failed");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="flex min-h-screen">

      {/* ── Left panel: branding ── */}
      <div className="relative hidden lg:flex lg:w-1/2 flex-col justify-between overflow-hidden bg-gradient-to-br from-slate-900 via-blue-950 to-slate-900 px-12 py-10 text-white">
        {/* Grid pattern overlay */}
        <div
          className="pointer-events-none absolute inset-0 opacity-[0.04]"
          style={{
            backgroundImage: `linear-gradient(rgba(255,255,255,.5) 1px, transparent 1px),
                              linear-gradient(90deg, rgba(255,255,255,.5) 1px, transparent 1px)`,
            backgroundSize: "40px 40px",
          }}
        />
        {/* Glowing orb */}
        <div className="pointer-events-none absolute -top-32 -left-32 h-96 w-96 rounded-full bg-blue-600/20 blur-3xl" />
        <div className="pointer-events-none absolute bottom-0 right-0 h-64 w-64 rounded-full bg-indigo-500/15 blur-3xl" />

        {/* Logo */}
        <div className="relative flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-blue-500 font-bold text-lg shadow-lg shadow-blue-500/30">
            VP
          </div>
          <div>
            <div className="font-bold text-lg leading-none">VentoPanel</div>
            <div className="text-xs text-blue-300/80">Control Panel</div>
          </div>
        </div>

        {/* Tagline */}
        <div className="relative space-y-6">
          <h1 className="text-4xl font-bold leading-tight tracking-tight">
            Infrastructure<br />
            <span className="text-blue-400">under control</span>
          </h1>
          <p className="text-slate-400 text-sm leading-relaxed max-w-sm">
            A modern Go-powered panel to deploy sites, manage servers, stream logs and monitor resources — all from one place.
          </p>

          <div className="space-y-3 pt-2">
            {FEATURES.map(({ icon: Icon, text }) => (
              <div key={text} className="flex items-center gap-3 text-sm text-slate-300">
                <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg bg-blue-500/15 text-blue-400">
                  <Icon className="h-3.5 w-3.5" />
                </div>
                {text}
              </div>
            ))}
          </div>
        </div>

        {/* Footer */}
        <div className="relative text-xs text-slate-600">
          © {new Date().getFullYear()} VentoPanel
        </div>
      </div>

      {/* ── Right panel: form ── */}
      <div className="flex flex-1 flex-col items-center justify-center bg-background px-6 py-12">
        {/* Mobile logo */}
        <div className="mb-8 flex flex-col items-center gap-2 lg:hidden">
          <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-primary font-bold text-xl text-primary-foreground shadow-lg">
            VP
          </div>
          <div className="text-center">
            <div className="font-bold text-xl">VentoPanel</div>
            <div className="text-xs text-muted-foreground">Control Panel</div>
          </div>
        </div>

        <div className="w-full max-w-sm">

          {/* MFA step */}
          {mfaSession ? (
            <div className="space-y-5 animate-in fade-in slide-in-from-bottom-4 duration-300">
              <div className="space-y-1">
                <h2 className="text-2xl font-bold">Two-factor auth</h2>
                <p className="text-sm text-muted-foreground">Enter the code from your authenticator app</p>
              </div>

              {error && <ErrorBox>{error}</ErrorBox>}

              <form onSubmit={handleMFA} className="space-y-4">
                <div className="space-y-1.5">
                  <Label htmlFor="mfa-code">6-digit code</Label>
                  <Input
                    id="mfa-code"
                    type="text"
                    inputMode="numeric"
                    placeholder="000000"
                    maxLength={6}
                    autoComplete="one-time-code"
                    autoFocus
                    className="text-center text-2xl tracking-[0.5em] font-mono h-12"
                    value={mfaCode}
                    onChange={e => setMfaCode(e.target.value.replace(/\D/g, ""))}
                    required
                  />
                </div>
                <Button type="submit" className="w-full h-11" disabled={loading || mfaCode.length !== 6}>
                  {loading ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : <Smartphone className="mr-2 h-4 w-4" />}
                  Verify
                </Button>
              </form>

              <button
                type="button"
                className="flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground transition-colors"
                onClick={() => { setMfaSession(null); setMfaCode(""); setError(""); }}
              >
                <ArrowLeft className="h-3.5 w-3.5" /> Back to login
              </button>
            </div>
          ) : (
            <div className="space-y-5 animate-in fade-in slide-in-from-bottom-4 duration-300">
              <div className="space-y-1">
                <h2 className="text-2xl font-bold">
                  {tab === "login" ? "Welcome back" : "Create account"}
                </h2>
                <p className="text-sm text-muted-foreground">
                  {tab === "login"
                    ? "Sign in to your VentoPanel account"
                    : "Register with your team ID"}
                </p>
              </div>

              {/* Tab switcher */}
              <div className="flex rounded-lg border bg-muted/40 p-1 gap-1">
                {(["login", "register"] as Tab[]).map(t => (
                  <button
                    key={t}
                    onClick={() => { setTab(t); setError(""); setSuccess(""); }}
                    className={cn(
                      "flex-1 rounded-md py-1.5 text-sm font-medium transition-all duration-200",
                      tab === t
                        ? "bg-background text-foreground shadow-sm"
                        : "text-muted-foreground hover:text-foreground",
                    )}
                  >
                    {t === "login" ? "Sign In" : "Register"}
                  </button>
                ))}
              </div>

              {error   && <ErrorBox>{error}</ErrorBox>}
              {success && <SuccessBox>{success}</SuccessBox>}

              {/* Login form */}
              {tab === "login" && (
                <form onSubmit={handleLogin} className="space-y-4">
                  <div className="space-y-1.5">
                    <Label htmlFor="email">Email</Label>
                    <Input id="email" type="email" placeholder="admin@example.com"
                      autoComplete="email" value={email}
                      onChange={e => setEmail(e.target.value)} required />
                  </div>
                  <div className="space-y-1.5">
                    <Label htmlFor="password">Password</Label>
                    <div className="relative">
                      <Input id="password" type={showPwd ? "text" : "password"}
                        placeholder="••••••••" autoComplete="current-password"
                        value={password} onChange={e => setPassword(e.target.value)} required />
                      <PwdToggle show={showPwd} onToggle={() => setShowPwd(v => !v)} />
                    </div>
                  </div>
                  <Button type="submit" className="w-full h-11" disabled={loading}>
                    {loading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                    Sign In
                  </Button>
                </form>
              )}

              {/* Register form */}
              {tab === "register" && (
                <form onSubmit={handleRegister} className="space-y-4">
                  <div className="space-y-1.5">
                    <Label htmlFor="reg-email">Email</Label>
                    <Input id="reg-email" type="email" placeholder="you@example.com"
                      autoComplete="email" value={regEmail}
                      onChange={e => setRegEmail(e.target.value)} required />
                  </div>
                  <div className="space-y-1.5">
                    <Label htmlFor="reg-password">Password</Label>
                    <div className="relative">
                      <Input id="reg-password" type={showRegPwd ? "text" : "password"}
                        placeholder="Min 8 characters" autoComplete="new-password"
                        value={regPassword} onChange={e => setRegPassword(e.target.value)}
                        minLength={8} required />
                      <PwdToggle show={showRegPwd} onToggle={() => setShowRegPwd(v => !v)} />
                    </div>
                  </div>
                  <div className="space-y-1.5">
                    <Label htmlFor="team-id">Team ID</Label>
                    <Input id="team-id" type="text"
                      placeholder="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
                      value={regTeamID} onChange={e => setRegTeamID(e.target.value)} required />
                    <p className="text-xs text-muted-foreground">
                      UUID of your team. The first user becomes admin.
                    </p>
                  </div>
                  <Button type="submit" className="w-full h-11" disabled={loading}>
                    {loading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                    Create Account
                  </Button>
                </form>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

// ─── Small helpers ────────────────────────────────────────────────────────────

function PwdToggle({ show, onToggle }: { show: boolean; onToggle: () => void }) {
  return (
    <button
      type="button"
      className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors"
      onClick={onToggle}
    >
      {show ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
    </button>
  );
}

function ErrorBox({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex items-start gap-2 rounded-lg border border-destructive/20 bg-destructive/8 px-3 py-2.5 text-sm text-destructive">
      {children}
    </div>
  );
}

function SuccessBox({ children }: { children: React.ReactNode }) {
  return (
    <div className="rounded-lg border border-green-200 bg-green-50 px-3 py-2.5 text-sm text-green-700 dark:border-green-900/40 dark:bg-green-900/20 dark:text-green-400">
      {children}
    </div>
  );
}
