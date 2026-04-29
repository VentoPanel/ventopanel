"use client";

import { useState, useEffect } from "react";
import {
  ShieldCheck,
  Key,
  Plus,
  Trash2,
  Copy,
  Check,
  Loader2,
  Smartphone,
  Eye,
  EyeOff,
  AlertTriangle,
} from "lucide-react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  fetchAPITokens,
  createAPIToken,
  revokeAPIToken,
  setupTOTP,
  enableTOTP,
  disableTOTP,
  getTokenPayload,
  type APIToken,
  type TOTPSetup,
} from "@/lib/api";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
} from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { toast } from "sonner";
import { formatDistanceToNow } from "date-fns";
// ─────────────────────────── QR code canvas ────────────────────────────────

function QRCanvas({ url }: { url: string }) {
  const [dataUrl, setDataUrl] = useState<string>("");

  useEffect(() => {
    if (!url) return;
    // Dynamically import qrcode so it only runs in the browser.
    import("qrcode").then((QRCode) => {
      QRCode.default.toDataURL(url, { width: 200, margin: 1 }).then(setDataUrl);
    });
  }, [url]);

  if (!dataUrl) {
    return (
      <div className="flex h-[200px] w-[200px] items-center justify-center rounded border bg-muted text-xs text-muted-foreground">
        Generating…
      </div>
    );
  }

  return (
    // eslint-disable-next-line @next/next/no-img-element
    <img src={dataUrl} alt="QR code" width={200} height={200} className="rounded border" />
  );
}

// ─────────────────────────── API Tokens section ────────────────────────────

function APITokensSection() {
  const qc = useQueryClient();
  const [newName, setNewName] = useState("");
  const [createdToken, setCreatedToken] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const [revokeTarget, setRevokeTarget] = useState<APIToken | null>(null);

  const { data: tokens = [], isLoading } = useQuery({
    queryKey: ["api-tokens"],
    queryFn: fetchAPITokens,
  });

  const createMutation = useMutation({
    mutationFn: () => createAPIToken(newName.trim()),
    onSuccess: (data) => {
      qc.invalidateQueries({ queryKey: ["api-tokens"] });
      setCreatedToken(data.token);
      setNewName("");
      toast.success("Token created — copy it now!");
    },
    onError: (err) => toast.error(err instanceof Error ? err.message : "Failed"),
  });

  const revokeMutation = useMutation({
    mutationFn: (id: string) => revokeAPIToken(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["api-tokens"] });
      toast.success("Token revoked");
      setRevokeTarget(null);
    },
    onError: (err) => {
      toast.error(err instanceof Error ? err.message : "Failed");
      setRevokeTarget(null);
    },
  });

  function copyToken() {
    if (!createdToken) return;
    navigator.clipboard.writeText(createdToken).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    });
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center gap-2">
          <Key className="h-4 w-4 text-muted-foreground" />
          <CardTitle className="text-base">API Tokens</CardTitle>
        </div>
        <CardDescription>
          Use personal tokens for CI/CD pipelines and scripts. A token has
          the same access level as your account.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">

        {/* Created token banner */}
        {createdToken && (
          <div className="rounded-md border border-amber-300 bg-amber-50 p-4 space-y-2">
            <div className="flex items-center gap-2 text-amber-800 text-sm font-medium">
              <AlertTriangle className="h-4 w-4" />
              Copy your token now — it won&apos;t be shown again.
            </div>
            <div className="flex items-center gap-2">
              <code className="flex-1 rounded bg-white border px-3 py-2 text-xs font-mono truncate text-amber-900">
                {createdToken}
              </code>
              <Button variant="outline" size="icon" onClick={copyToken} className="shrink-0">
                {copied ? <Check className="h-4 w-4 text-green-600" /> : <Copy className="h-4 w-4" />}
              </Button>
            </div>
            <Button
              variant="ghost"
              size="sm"
              className="text-muted-foreground"
              onClick={() => setCreatedToken(null)}
            >
              Dismiss
            </Button>
          </div>
        )}

        {/* Create form */}
        <div className="flex gap-2">
          <Input
            placeholder="Token name (e.g. GitHub Actions)"
            value={newName}
            onChange={(e) => setNewName(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && newName.trim() && createMutation.mutate()}
          />
          <Button
            disabled={!newName.trim() || createMutation.isPending}
            onClick={() => createMutation.mutate()}
          >
            {createMutation.isPending ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Plus className="h-4 w-4" />
            )}
          </Button>
        </div>

        {/* Token list */}
        {isLoading && (
          <p className="text-sm text-muted-foreground">Loading…</p>
        )}
        {!isLoading && tokens.length === 0 && (
          <p className="text-sm text-muted-foreground">No tokens yet.</p>
        )}
        {tokens.length > 0 && (
          <ul className="divide-y rounded border">
            {tokens.map((t) => (
              <li key={t.id} className="flex items-center gap-3 px-4 py-3">
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium truncate">{t.name}</p>
                  <p className="text-xs text-muted-foreground">
                    Created{" "}
                    {formatDistanceToNow(new Date(t.created_at), { addSuffix: true })}
                    {t.last_used_at && (
                      <> · Last used{" "}
                        {formatDistanceToNow(new Date(t.last_used_at), { addSuffix: true })}
                      </>
                    )}
                  </p>
                </div>
                <Badge variant="outline" className="font-mono text-xs shrink-0">
                  vp_••••••••
                </Badge>
                <Button
                  variant="ghost"
                  size="icon"
                  className="text-destructive hover:text-destructive h-7 w-7 shrink-0"
                  onClick={() => setRevokeTarget(t)}
                >
                  <Trash2 className="h-3.5 w-3.5" />
                </Button>
              </li>
            ))}
          </ul>
        )}
      </CardContent>

      <ConfirmDialog
        open={!!revokeTarget}
        title={`Revoke "${revokeTarget?.name}"?`}
        description="Any scripts using this token will stop working immediately."
        loading={revokeMutation.isPending}
        onConfirm={() => revokeTarget && revokeMutation.mutate(revokeTarget.id)}
        onCancel={() => setRevokeTarget(null)}
      />
    </Card>
  );
}

// ─────────────────────────── 2FA / TOTP section ────────────────────────────

type TOTPStep = "idle" | "setup" | "verify" | "disable";

function TwoFactorSection() {
  const payload = getTokenPayload();
  const [step, setStep] = useState<TOTPStep>("idle");
  const [setup, setSetup] = useState<TOTPSetup | null>(null);
  const [code, setCode] = useState("");
  const [showSecret, setShowSecret] = useState(false);
  // Read current 2FA status from JWT token payload.
  const [enabled, setEnabled] = useState<boolean>(payload?.totp_enabled ?? false);

  const setupMutation = useMutation({
    mutationFn: setupTOTP,
    onSuccess: (data) => {
      setSetup(data);
      setStep("verify");
    },
    onError: (err) => toast.error(err instanceof Error ? err.message : "Failed"),
  });

  const enableMutation = useMutation({
    mutationFn: () => enableTOTP(code),
    onSuccess: () => {
      toast.success("Two-factor authentication enabled");
      setEnabled(true);
      setStep("idle");
      setCode("");
      setSetup(null);
    },
    onError: (err) => toast.error(err instanceof Error ? err.message : "Invalid code"),
  });

  const disableMutation = useMutation({
    mutationFn: () => disableTOTP(code),
    onSuccess: () => {
      toast.success("Two-factor authentication disabled");
      setEnabled(false);
      setStep("idle");
      setCode("");
    },
    onError: (err) => toast.error(err instanceof Error ? err.message : "Invalid code"),
  });

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center gap-2">
          <Smartphone className="h-4 w-4 text-muted-foreground" />
          <CardTitle className="text-base">Two-Factor Authentication</CardTitle>
        </div>
        <CardDescription>
          Protect your account with a time-based one-time password (TOTP) app
          such as Google Authenticator or Authy.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">

        {/* Status badge */}
        <div className="flex items-center gap-3">
          {enabled ? (
            <Badge className="bg-green-100 text-green-800 border-green-300">
              <ShieldCheck className="mr-1 h-3 w-3" /> Enabled
            </Badge>
          ) : (
            <Badge variant="outline" className="text-muted-foreground">
              Disabled
            </Badge>
          )}
        </div>

        {/* Idle state */}
        {step === "idle" && !enabled && (
          <Button
            onClick={() => {
              setStep("setup");
              setupMutation.mutate();
            }}
            disabled={setupMutation.isPending}
          >
            {setupMutation.isPending ? (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            ) : (
              <Smartphone className="mr-2 h-4 w-4" />
            )}
            Set up 2FA
          </Button>
        )}

        {step === "idle" && enabled && (
          <Button
            variant="destructive"
            onClick={() => setStep("disable")}
          >
            Disable 2FA
          </Button>
        )}

        {/* Setup: QR code */}
        {step === "verify" && setup && (
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground">
              Scan the QR code with your authenticator app, then enter the
              6-digit code to confirm.
            </p>
            <QRCanvas url={setup.url} />
            <div className="space-y-1">
              <Label className="text-xs text-muted-foreground">
                Or enter the secret manually:
              </Label>
              <div className="flex items-center gap-2">
                <code className="rounded bg-muted px-3 py-1 text-xs font-mono">
                  {showSecret ? setup.secret : "•".repeat(setup.secret.length)}
                </code>
                <button
                  type="button"
                  onClick={() => setShowSecret((v) => !v)}
                  className="text-muted-foreground hover:text-foreground"
                >
                  {showSecret ? (
                    <EyeOff className="h-3.5 w-3.5" />
                  ) : (
                    <Eye className="h-3.5 w-3.5" />
                  )}
                </button>
              </div>
            </div>
            <div className="flex gap-2 items-end">
              <div className="space-y-1 flex-1">
                <Label htmlFor="totp-code">Verification code</Label>
                <Input
                  id="totp-code"
                  placeholder="123456"
                  maxLength={6}
                  value={code}
                  onChange={(e) => setCode(e.target.value.replace(/\D/g, ""))}
                  onKeyDown={(e) => e.key === "Enter" && code.length === 6 && enableMutation.mutate()}
                />
              </div>
              <Button
                disabled={code.length !== 6 || enableMutation.isPending}
                onClick={() => enableMutation.mutate()}
              >
                {enableMutation.isPending ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  "Enable"
                )}
              </Button>
              <Button variant="ghost" onClick={() => { setStep("idle"); setCode(""); setSetup(null); }}>
                Cancel
              </Button>
            </div>
          </div>
        )}

        {/* Disable: confirm with code */}
        {step === "disable" && (
          <div className="space-y-3">
            <p className="text-sm text-muted-foreground">
              Enter your current authenticator code to disable 2FA.
            </p>
            <div className="flex gap-2 items-end">
              <div className="space-y-1 flex-1">
                <Label htmlFor="disable-code">Authenticator code</Label>
                <Input
                  id="disable-code"
                  placeholder="123456"
                  maxLength={6}
                  value={code}
                  onChange={(e) => setCode(e.target.value.replace(/\D/g, ""))}
                  onKeyDown={(e) => e.key === "Enter" && code.length === 6 && disableMutation.mutate()}
                />
              </div>
              <Button
                variant="destructive"
                disabled={code.length !== 6 || disableMutation.isPending}
                onClick={() => disableMutation.mutate()}
              >
                {disableMutation.isPending ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  "Disable"
                )}
              </Button>
              <Button variant="ghost" onClick={() => { setStep("idle"); setCode(""); }}>
                Cancel
              </Button>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

// ─────────────────────────── Page ────────────────────────────────────────────

export default function SecurityPage() {
  return (
    <div className="max-w-3xl space-y-6">
      <div className="flex items-center gap-3">
        <ShieldCheck className="h-6 w-6 text-muted-foreground" />
        <div>
          <h2 className="text-2xl font-bold tracking-tight">Security</h2>
          <p className="text-muted-foreground">
            API tokens and two-factor authentication
          </p>
        </div>
      </div>

      <APITokensSection />
      <TwoFactorSection />
    </div>
  );
}
