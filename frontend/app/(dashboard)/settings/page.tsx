"use client";

import { useEffect, useState } from "react";
import { Settings, Send, MessageSquare, Save, Eye, EyeOff } from "lucide-react";
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
  });

  useEffect(() => {
    if (data) setForm(data);
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
                Your personal chat ID or a group/channel ID (starts with -100).
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
