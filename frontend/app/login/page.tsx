"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Eye, EyeOff, Loader2, Smartphone, ArrowLeft } from "lucide-react";
import {
  login,
  isMFARequired,
  verifyMFA,
  registerUser,
  setToken,
} from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { cn } from "@/lib/utils";

type Tab = "login" | "register";

export default function LoginPage() {
  const router = useRouter();
  const [tab, setTab] = useState<Tab>("login");

  // Login form
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [showPwd, setShowPwd] = useState(false);

  // MFA step
  const [mfaSession, setMfaSession] = useState<string | null>(null);
  const [mfaCode, setMfaCode] = useState("");

  // Register form
  const [regEmail, setRegEmail] = useState("");
  const [regPassword, setRegPassword] = useState("");
  const [regTeamID, setRegTeamID] = useState("");
  const [showRegPwd, setShowRegPwd] = useState(false);

  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");

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
      await registerUser({
        email: regEmail.trim(),
        password: regPassword,
        team_id: regTeamID.trim(),
      });
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
    <div className="flex min-h-screen items-center justify-center bg-muted/40 px-4">
      <Card className="w-full max-w-md shadow-lg">
        <CardHeader className="space-y-1 text-center">
          <div className="mx-auto mb-2 flex h-12 w-12 items-center justify-center rounded-xl bg-primary text-primary-foreground font-bold text-xl">
            VP
          </div>
          <CardTitle className="text-2xl font-bold">VentoPanel</CardTitle>
          <CardDescription>
            The Go-powered Control Panel for Scalable Apps
          </CardDescription>
        </CardHeader>

        <CardContent className="space-y-4">

          {/* ── MFA step ── */}
          {mfaSession ? (
            <>
              <div className="flex items-center gap-2 text-sm text-muted-foreground">
                <Smartphone className="h-4 w-4" />
                Two-factor authentication required
              </div>

              {error && (
                <p className="rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
                  {error}
                </p>
              )}

              <form onSubmit={handleMFA} className="space-y-4">
                <div className="space-y-1.5">
                  <Label htmlFor="mfa-code">Authenticator code</Label>
                  <Input
                    id="mfa-code"
                    type="text"
                    inputMode="numeric"
                    placeholder="123456"
                    maxLength={6}
                    autoComplete="one-time-code"
                    autoFocus
                    value={mfaCode}
                    onChange={(e) => setMfaCode(e.target.value.replace(/\D/g, ""))}
                    required
                  />
                </div>
                <Button type="submit" className="w-full" disabled={loading || mfaCode.length !== 6}>
                  {loading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                  Verify
                </Button>
              </form>

              <button
                type="button"
                className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground"
                onClick={() => { setMfaSession(null); setMfaCode(""); setError(""); }}
              >
                <ArrowLeft className="h-3 w-3" /> Back to login
              </button>
            </>
          ) : (
            <>
              {/* Tab switcher */}
              <div className="flex rounded-lg border p-1">
                {(["login", "register"] as Tab[]).map((t) => (
                  <button
                    key={t}
                    onClick={() => {
                      setTab(t);
                      setError("");
                      setSuccess("");
                    }}
                    className={cn(
                      "flex-1 rounded-md py-1.5 text-sm font-medium transition-colors",
                      tab === t
                        ? "bg-primary text-primary-foreground shadow-sm"
                        : "text-muted-foreground hover:text-foreground",
                    )}
                  >
                    {t === "login" ? "Sign In" : "Register"}
                  </button>
                ))}
              </div>

              {error && (
                <p className="rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
                  {error}
                </p>
              )}
              {success && (
                <p className="rounded-md bg-green-50 px-3 py-2 text-sm text-green-700">
                  {success}
                </p>
              )}

              {/* Login form */}
              {tab === "login" && (
                <form onSubmit={handleLogin} className="space-y-4">
                  <div className="space-y-1.5">
                    <Label htmlFor="email">Email</Label>
                    <Input
                      id="email"
                      type="email"
                      placeholder="admin@example.com"
                      autoComplete="email"
                      value={email}
                      onChange={(e) => setEmail(e.target.value)}
                      required
                    />
                  </div>
                  <div className="space-y-1.5">
                    <Label htmlFor="password">Password</Label>
                    <div className="relative">
                      <Input
                        id="password"
                        type={showPwd ? "text" : "password"}
                        placeholder="••••••••"
                        autoComplete="current-password"
                        value={password}
                        onChange={(e) => setPassword(e.target.value)}
                        required
                      />
                      <button
                        type="button"
                        className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                        onClick={() => setShowPwd((v) => !v)}
                      >
                        {showPwd ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                      </button>
                    </div>
                  </div>
                  <Button type="submit" className="w-full" disabled={loading}>
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
                    <Input
                      id="reg-email"
                      type="email"
                      placeholder="you@example.com"
                      autoComplete="email"
                      value={regEmail}
                      onChange={(e) => setRegEmail(e.target.value)}
                      required
                    />
                  </div>
                  <div className="space-y-1.5">
                    <Label htmlFor="reg-password">Password</Label>
                    <div className="relative">
                      <Input
                        id="reg-password"
                        type={showRegPwd ? "text" : "password"}
                        placeholder="Min 8 characters"
                        autoComplete="new-password"
                        value={regPassword}
                        onChange={(e) => setRegPassword(e.target.value)}
                        minLength={8}
                        required
                      />
                      <button
                        type="button"
                        className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                        onClick={() => setShowRegPwd((v) => !v)}
                      >
                        {showRegPwd ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                      </button>
                    </div>
                  </div>
                  <div className="space-y-1.5">
                    <Label htmlFor="team-id">Team ID</Label>
                    <Input
                      id="team-id"
                      type="text"
                      placeholder="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
                      value={regTeamID}
                      onChange={(e) => setRegTeamID(e.target.value)}
                      required
                    />
                    <p className="text-xs text-muted-foreground">
                      UUID of the team this account belongs to. The first registered
                      user becomes admin.
                    </p>
                  </div>
                  <Button type="submit" className="w-full" disabled={loading}>
                    {loading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                    Create Account
                  </Button>
                </form>
              )}
            </>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
