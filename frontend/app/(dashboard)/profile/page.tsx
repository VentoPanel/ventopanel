"use client";

import { useState } from "react";
import { useQuery, useMutation } from "@tanstack/react-query";
import { User, KeyRound, Mail, ShieldCheck, Eye, EyeOff, Loader2 } from "lucide-react";
import { fetchMe, changePassword, changeEmail } from "@/lib/api";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { toast } from "sonner";

function PwdInput({
  id, value, onChange, placeholder,
}: { id: string; value: string; onChange: (v: string) => void; placeholder?: string }) {
  const [show, setShow] = useState(false);
  return (
    <div className="relative">
      <Input
        id={id}
        type={show ? "text" : "password"}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        className="pr-10"
        autoComplete="new-password"
      />
      <button
        type="button"
        onClick={() => setShow((s) => !s)}
        className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
      >
        {show ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
      </button>
    </div>
  );
}

export default function ProfilePage() {
  const { data: me, isLoading } = useQuery({ queryKey: ["me"], queryFn: fetchMe });

  // Password form
  const [pwdForm, setPwdForm] = useState({ current: "", next: "", confirm: "" });

  const { mutate: savePwd, isPending: savingPwd } = useMutation({
    mutationFn: () => changePassword(pwdForm.current, pwdForm.next),
    onSuccess: () => {
      toast.success("Password updated");
      setPwdForm({ current: "", next: "", confirm: "" });
    },
    onError: (e) => toast.error(e instanceof Error ? e.message : "Failed to update password"),
  });

  function handlePwdSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (pwdForm.next.length < 8) { toast.error("New password must be at least 8 characters"); return; }
    if (pwdForm.next !== pwdForm.confirm) { toast.error("Passwords do not match"); return; }
    savePwd();
  }

  // Email form
  const [emailForm, setEmailForm] = useState({ email: "", password: "" });

  const { mutate: saveEmail, isPending: savingEmail } = useMutation({
    mutationFn: () => changeEmail(emailForm.email, emailForm.password),
    onSuccess: () => {
      toast.success("Email updated — please log in again");
      setEmailForm({ email: "", password: "" });
    },
    onError: (e) => toast.error(e instanceof Error ? e.message : "Failed to update email"),
  });

  function handleEmailSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!emailForm.email) { toast.error("Enter a new email address"); return; }
    saveEmail();
  }

  if (isLoading) {
    return (
      <div className="flex items-center gap-2 text-muted-foreground">
        <Loader2 className="h-4 w-4 animate-spin" /> Loading profile…
      </div>
    );
  }

  return (
    <div className="max-w-2xl space-y-6">
      {/* Header */}
      <div className="flex items-center gap-3">
        <User className="h-6 w-6 text-muted-foreground" />
        <div>
          <h2 className="text-2xl font-bold tracking-tight">Profile</h2>
          <p className="text-muted-foreground">Manage your account settings</p>
        </div>
      </div>

      {/* Info card */}
      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="flex items-center gap-2 text-base">
            <ShieldCheck className="h-4 w-4" />
            Account info
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-3 text-sm">
          <div className="flex items-center justify-between">
            <span className="text-muted-foreground">Email</span>
            <span className="font-medium">{me?.email}</span>
          </div>
          <div className="flex items-center justify-between">
            <span className="text-muted-foreground">Role</span>
            <Badge variant="secondary" className="capitalize">{me?.role}</Badge>
          </div>
          <div className="flex items-center justify-between">
            <span className="text-muted-foreground">2FA (TOTP)</span>
            {me?.totp_enabled
              ? <Badge variant="success">Enabled</Badge>
              : <Badge variant="secondary">Disabled</Badge>}
          </div>
          <div className="flex items-center justify-between">
            <span className="text-muted-foreground">Member since</span>
            <span className="font-mono text-xs">{me?.created_at?.slice(0, 10)}</span>
          </div>
        </CardContent>
      </Card>

      {/* Change password */}
      <Card>
        <CardHeader className="pb-4">
          <CardTitle className="flex items-center gap-2 text-base">
            <KeyRound className="h-4 w-4" />
            Change password
          </CardTitle>
          <CardDescription>Minimum 8 characters. You will stay logged in.</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handlePwdSubmit} className="space-y-4">
            <div className="space-y-1.5">
              <Label htmlFor="cur-pwd">Current password</Label>
              <PwdInput id="cur-pwd" value={pwdForm.current} onChange={(v) => setPwdForm((f) => ({ ...f, current: v }))} placeholder="••••••••" />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="new-pwd">New password</Label>
              <PwdInput id="new-pwd" value={pwdForm.next} onChange={(v) => setPwdForm((f) => ({ ...f, next: v }))} placeholder="Min. 8 characters" />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="conf-pwd">Confirm new password</Label>
              <PwdInput id="conf-pwd" value={pwdForm.confirm} onChange={(v) => setPwdForm((f) => ({ ...f, confirm: v }))} placeholder="Repeat new password" />
            </div>
            <div className="flex justify-end">
              <Button type="submit" disabled={savingPwd || !pwdForm.current || !pwdForm.next}>
                {savingPwd ? <><Loader2 className="mr-2 h-4 w-4 animate-spin" />Saving…</> : "Update password"}
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>

      {/* Change email */}
      <Card>
        <CardHeader className="pb-4">
          <CardTitle className="flex items-center gap-2 text-base">
            <Mail className="h-4 w-4" />
            Change email
          </CardTitle>
          <CardDescription>Enter your current password to confirm the change.</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleEmailSubmit} className="space-y-4">
            <div className="space-y-1.5">
              <Label htmlFor="new-email">New email address</Label>
              <Input
                id="new-email"
                type="email"
                value={emailForm.email}
                onChange={(e) => setEmailForm((f) => ({ ...f, email: e.target.value }))}
                placeholder="you@example.com"
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="email-pwd">Current password</Label>
              <PwdInput id="email-pwd" value={emailForm.password} onChange={(v) => setEmailForm((f) => ({ ...f, password: v }))} placeholder="••••••••" />
            </div>
            <div className="flex justify-end">
              <Button type="submit" disabled={savingEmail || !emailForm.email || !emailForm.password}>
                {savingEmail ? <><Loader2 className="mr-2 h-4 w-4 animate-spin" />Saving…</> : "Update email"}
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
