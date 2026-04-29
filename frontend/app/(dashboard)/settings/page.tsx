"use client";

import { useEffect, useState } from "react";
import { Settings, Send, MessageSquare, Save, Eye, EyeOff, Activity } from "lucide-react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  fetchNotificationSettings,
  updateNotificationSettings,
  type NotificationSettings,
} from "@/lib/api";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { toast } from "sonner";

function MaskedInput({
  value,
  onChange,
  placeholder,
  id,
}: {
  value: string;
  onChange: (v: string) => void;
  placeholder?: string;
  id: string;
}) {
  const [show, setShow] = useState(false);
  return (
    <div className="relative">
      <Input
        id={id}
        type={show ? "text" : "password"}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        className="pr-10 font-mono text-sm"
      />
      <button
        type="button"
        className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
        onClick={() => setShow((s) => !s)}
      >
        {show ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
      </button>
    </div>
  );
}

export default function SettingsPage() {
  const qc = useQueryClient();

  const { data, isLoading } = useQuery({
    queryKey: ["settings-notifications"],
    queryFn: fetchNotificationSettings,
  });

  const [form, setForm] = useState<NotificationSettings>({
    telegram_bot_token: "",
    telegram_chat_id: "",
    whatsapp_webhook_url: "",
    uptime_notify_down: true,
    uptime_notify_recovery: true,
    uptime_fail_threshold: 1,
    uptime_recovery_threshold: 1,
  });

  useEffect(() => {
    if (!data) return;
    setForm({
      telegram_bot_token: data.telegram_bot_token ?? "",
      telegram_chat_id: data.telegram_chat_id ?? "",
      whatsapp_webhook_url: data.whatsapp_webhook_url ?? "",
      uptime_notify_down: data.uptime_notify_down ?? true,
      uptime_notify_recovery: data.uptime_notify_recovery ?? true,
      uptime_fail_threshold:
        typeof data.uptime_fail_threshold === "number" ? data.uptime_fail_threshold : 1,
      uptime_recovery_threshold:
        typeof data.uptime_recovery_threshold === "number"
          ? data.uptime_recovery_threshold
          : 1,
    });
  }, [data]);

  const { mutate: save, isPending: saving } = useMutation({
    mutationFn: updateNotificationSettings,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["settings-notifications"] });
      toast.success("Settings saved");
    },
    onError: (err) => {
      toast.error(err instanceof Error ? err.message : "Failed to save");
    },
  });

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    save(form);
  }

  return (
    <div className="max-w-2xl space-y-6">
      <div className="flex items-center gap-3">
        <Settings className="h-6 w-6 text-muted-foreground" />
        <div>
          <h2 className="text-2xl font-bold tracking-tight">Settings</h2>
          <p className="text-muted-foreground">Manage notification integrations</p>
        </div>
      </div>

      <form onSubmit={handleSubmit} className="space-y-6">
        {/* Telegram */}
        <Card>
          <CardHeader className="pb-4">
            <CardTitle className="flex items-center gap-2 text-base">
              <Send className="h-4 w-4" />
              Telegram
            </CardTitle>
            <CardDescription>
              Deploy alerts sent via Telegram Bot API. Create a bot with{" "}
              <a
                href="https://t.me/BotFather"
                target="_blank"
                rel="noopener noreferrer"
                className="underline"
              >
                @BotFather
              </a>{" "}
              to get a token.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-1.5">
              <Label htmlFor="tg-token">Bot Token</Label>
              <MaskedInput
                id="tg-token"
                value={form.telegram_bot_token}
                onChange={(v) => setForm((f) => ({ ...f, telegram_bot_token: v }))}
                placeholder="123456:ABC-DEF..."
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="tg-chat">Chat ID</Label>
              <Input
                id="tg-chat"
                value={form.telegram_chat_id}
                onChange={(e) => setForm((f) => ({ ...f, telegram_chat_id: e.target.value }))}
                placeholder="-100123456789"
                className="font-mono text-sm"
              />
              <p className="text-xs text-muted-foreground">
                Personal chat, group, or channel. Multiple IDs: comma or one per line.
              </p>
            </div>
          </CardContent>
        </Card>

        {/* WhatsApp */}
        <Card>
          <CardHeader className="pb-4">
            <CardTitle className="flex items-center gap-2 text-base">
              <MessageSquare className="h-4 w-4" />
              WhatsApp Webhook
            </CardTitle>
            <CardDescription>
              Compatible with Evolution API, Twilio, and other generic webhook providers.
              Leave empty to disable.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-1.5">
            <Label htmlFor="wa-url">Webhook URL</Label>
            <Input
              id="wa-url"
              value={form.whatsapp_webhook_url}
              onChange={(e) =>
                setForm((f) => ({ ...f, whatsapp_webhook_url: e.target.value }))
              }
              placeholder="https://api.example.com/webhook/send"
              className="font-mono text-sm"
            />
            <p className="text-xs text-muted-foreground mt-2">
              Multiple URLs: comma-separated or one per line (same message to each).
            </p>
          </CardContent>
        </Card>

        {/* Uptime alerts */}
        <Card>
          <CardHeader className="pb-4">
            <CardTitle className="flex items-center gap-2 text-base">
              <Activity className="h-4 w-4" />
              Uptime alerts
            </CardTitle>
            <CardDescription>
              Applies to HTTP checks every minute. Thresholds reduce noise when the network flaps.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <label className="flex cursor-pointer items-start gap-3">
              <input
                type="checkbox"
                className="mt-1 h-4 w-4 rounded border-input"
                checked={form.uptime_notify_down}
                onChange={(e) =>
                  setForm((f) => ({ ...f, uptime_notify_down: e.target.checked }))
                }
              />
              <span>
                <span className="font-medium">Notify when site goes down</span>
                <span className="block text-xs text-muted-foreground">
                  Disable if you only want recovery messages (e.g. fewer Telegram messages).
                </span>
              </span>
            </label>
            <label className="flex cursor-pointer items-start gap-3">
              <input
                type="checkbox"
                className="mt-1 h-4 w-4 rounded border-input"
                checked={form.uptime_notify_recovery}
                onChange={(e) =>
                  setForm((f) => ({ ...f, uptime_notify_recovery: e.target.checked }))
                }
              />
              <span>
                <span className="font-medium">Notify when site recovers</span>
                <span className="block text-xs text-muted-foreground">
                  Turn off if you only care about outages.
                </span>
              </span>
            </label>
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-1.5">
                <Label htmlFor="fail-th">Failed checks before DOWN alert</Label>
                <Input
                  id="fail-th"
                  type="number"
                  min={1}
                  max={60}
                  value={form.uptime_fail_threshold}
                  onChange={(e) =>
                    setForm((f) => ({
                      ...f,
                      uptime_fail_threshold: Math.min(
                        60,
                        Math.max(1, parseInt(e.target.value, 10) || 1),
                      ),
                    }))
                  }
                  className="font-mono text-sm"
                />
                <p className="text-xs text-muted-foreground">1 = notify on first failure (default).</p>
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="recv-th">OK checks before RECOVERY alert</Label>
                <Input
                  id="recv-th"
                  type="number"
                  min={1}
                  max={60}
                  value={form.uptime_recovery_threshold}
                  onChange={(e) =>
                    setForm((f) => ({
                      ...f,
                      uptime_recovery_threshold: Math.min(
                        60,
                        Math.max(1, parseInt(e.target.value, 10) || 1),
                      ),
                    }))
                  }
                  className="font-mono text-sm"
                />
                <p className="text-xs text-muted-foreground">
                  Require several green checks before “recovered” (reduces false positives).
                </p>
              </div>
            </div>
          </CardContent>
        </Card>

        <div className="flex justify-end">
          <Button type="submit" disabled={saving || isLoading}>
            <Save className="mr-2 h-4 w-4" />
            {saving ? "Saving…" : "Save Settings"}
          </Button>
        </div>
      </form>
    </div>
  );
}
