"use client";

import { useState, useEffect } from "react";
import { Lock, Loader2, ShieldCheck } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { toast } from "sonner";
import { fmSetPermissions, type FileItem } from "@/lib/api";
import { cn } from "@/lib/utils";

// ─── Types ────────────────────────────────────────────────────────────────────

interface TriBits { r: boolean; w: boolean; x: boolean }
interface PermState { owner: TriBits; group: TriBits; public: TriBits }

type Entity = keyof PermState;

// ─── Helpers ──────────────────────────────────────────────────────────────────

function bitsToDigit(b: TriBits): number {
  return (b.r ? 4 : 0) + (b.w ? 2 : 0) + (b.x ? 1 : 0);
}

function digitToBits(n: number): TriBits {
  return { r: !!(n & 4), w: !!(n & 2), x: !!(n & 1) };
}

function stateToDirToMode(state: PermState): string {
  return `${bitsToDigit(state.owner)}${bitsToDigit(state.group)}${bitsToDigit(state.public)}`;
}

function modeToState(mode: string): PermState | null {
  const digits = mode.replace(/[^0-7]/g, "");
  // Support 3 or 4-digit modes; ignore leading sticky/setuid digit.
  const trimmed = digits.slice(-3);
  if (trimmed.length !== 3) return null;
  const [o, g, p] = trimmed.split("").map(Number);
  return { owner: digitToBits(o), group: digitToBits(g), public: digitToBits(p) };
}

function modeDescription(mode: string): string {
  const presets: Record<string, string> = {
    "777": "Everyone can read, write, execute",
    "755": "Owner: full · Group & Public: read+execute",
    "644": "Owner: read+write · Group & Public: read-only",
    "600": "Owner only — private",
    "700": "Owner only — private + executable",
    "666": "All can read+write (no execute)",
    "444": "Read-only for everyone",
  };
  return presets[mode] ?? "";
}

// ─── Symbolic notation display (e.g. rwxr-xr-x) ──────────────────────────────

function toSymbolic(state: PermState): string {
  const f = (b: TriBits) =>
    `${b.r ? "r" : "-"}${b.w ? "w" : "-"}${b.x ? "x" : "-"}`;
  return f(state.owner) + f(state.group) + f(state.public);
}

// ─── Presets ──────────────────────────────────────────────────────────────────

const PRESETS = [
  { label: "644", hint: "file" },
  { label: "755", hint: "dir/exec" },
  { label: "600", hint: "private" },
  { label: "777", hint: "open" },
];

// ─── Props ────────────────────────────────────────────────────────────────────

interface Props {
  item: FileItem;
  onClose: () => void;
  onSaved?: () => void;
}

// ─── Component ────────────────────────────────────────────────────────────────

export function PermissionsModal({ item, onClose, onSaved }: Props) {
  const [state, setState] = useState<PermState>({
    owner:  { r: true,  w: true,  x: false },
    group:  { r: true,  w: false, x: false },
    public: { r: true,  w: false, x: false },
  });

  const [octal,  setOctal]  = useState("644");
  const [saving, setSaving] = useState(false);

  // Sync state → octal whenever checkboxes change.
  useEffect(() => {
    setOctal(stateToDirToMode(state));
  }, [state]);

  // Sync octal → state when user types in the text input.
  function handleOctalInput(val: string) {
    const clean = val.replace(/[^0-7]/g, "").slice(0, 4);
    setOctal(clean);
    const parsed = modeToState(clean);
    if (parsed) setState(parsed);
  }

  // Toggle one permission bit.
  function toggle(entity: Entity, bit: keyof TriBits) {
    setState((prev) => ({
      ...prev,
      [entity]: { ...prev[entity], [bit]: !prev[entity][bit] },
    }));
  }

  async function handleSave() {
    if (octal.length < 3) { toast.error("Enter a valid 3-digit octal mode"); return; }
    setSaving(true);
    try {
      await fmSetPermissions(item.path, octal);
      toast.success("Permissions updated");
      onSaved?.();
      onClose();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "Failed to update permissions");
    } finally {
      setSaving(false);
    }
  }

  const ROWS: { key: Entity; label: string }[] = [
    { key: "owner",  label: "Owner" },
    { key: "group",  label: "Group" },
    { key: "public", label: "Public" },
  ];

  const COLS: { key: keyof TriBits; label: string; value: number }[] = [
    { key: "r", label: "Read",    value: 4 },
    { key: "w", label: "Write",   value: 2 },
    { key: "x", label: "Execute", value: 1 },
  ];

  const desc = modeDescription(octal);

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4 backdrop-blur-sm">
      <div className="w-full max-w-md rounded-xl border bg-background shadow-2xl overflow-hidden">

        {/* Header */}
        <div className="flex items-center gap-3 border-b bg-muted/40 px-5 py-4">
          <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary/10">
            <Lock className="h-4 w-4 text-primary" />
          </div>
          <div className="flex-1 min-w-0">
            <p className="font-semibold text-sm">File Permissions</p>
            <p className="text-xs text-muted-foreground font-mono truncate">{item.path}</p>
          </div>
        </div>

        <div className="p-5 space-y-5">

          {/* Octal input + symbolic + presets */}
          <div className="flex items-start gap-3">
            <div className="space-y-1">
              <label className="text-xs text-muted-foreground font-medium">Octal mode</label>
              <Input
                value={octal}
                onChange={(e) => handleOctalInput(e.target.value)}
                maxLength={4}
                className="w-20 text-center font-mono text-lg font-bold tracking-widest"
              />
            </div>
            <div className="space-y-1 flex-1">
              <label className="text-xs text-muted-foreground font-medium">Symbolic</label>
              <div className="rounded-md border bg-muted/50 px-3 py-2 font-mono text-sm tracking-widest">
                {toSymbolic(state)}
              </div>
            </div>
          </div>

          {/* Description */}
          {desc && (
            <div className="flex items-center gap-2 rounded-md bg-blue-50 dark:bg-blue-950/40 px-3 py-2 text-xs text-blue-700 dark:text-blue-300">
              <ShieldCheck className="h-3.5 w-3.5 shrink-0" />
              {desc}
            </div>
          )}

          {/* Presets */}
          <div className="flex gap-2 flex-wrap">
            {PRESETS.map((p) => (
              <button
                key={p.label}
                onClick={() => handleOctalInput(p.label)}
                className={cn(
                  "rounded-full border px-3 py-1 text-xs font-mono font-medium transition-colors",
                  octal === p.label
                    ? "bg-primary text-primary-foreground border-primary"
                    : "hover:bg-muted border-border",
                )}
              >
                {p.label}
                <span className="ml-1 text-[10px] opacity-60">{p.hint}</span>
              </button>
            ))}
          </div>

          {/* 3×3 Checkbox Grid */}
          <div className="rounded-lg border overflow-hidden">
            <table className="w-full text-sm">
              <thead>
                <tr className="bg-muted/50 text-xs text-muted-foreground">
                  <th className="px-4 py-2 text-left font-medium w-24">Entity</th>
                  {COLS.map((col) => (
                    <th key={col.key} className="px-4 py-2 text-center font-medium">
                      <div>{col.label}</div>
                      <div className="text-[10px] opacity-60">(+{col.value})</div>
                    </th>
                  ))}
                  <th className="px-4 py-2 text-center font-medium w-12">
                    <div>Val</div>
                  </th>
                </tr>
              </thead>
              <tbody>
                {ROWS.map((row, ri) => (
                  <tr key={row.key} className={cn("border-t", ri % 2 === 0 ? "" : "bg-muted/20")}>
                    <td className="px-4 py-3 font-medium text-sm">{row.label}</td>
                    {COLS.map((col) => {
                      const checked = state[row.key][col.key];
                      return (
                        <td key={col.key} className="px-4 py-3 text-center">
                          <button
                            type="button"
                            onClick={() => toggle(row.key, col.key)}
                            className={cn(
                              "mx-auto flex h-6 w-6 items-center justify-center rounded-md border-2 transition-all",
                              checked
                                ? "border-primary bg-primary text-primary-foreground"
                                : "border-border hover:border-primary/50",
                            )}
                            aria-label={`${row.label} ${col.label}`}
                          >
                            {checked && (
                              <svg className="h-3 w-3" viewBox="0 0 12 12" fill="none">
                                <path d="M2 6l3 3 5-5" stroke="currentColor" strokeWidth="2"
                                  strokeLinecap="round" strokeLinejoin="round" />
                              </svg>
                            )}
                          </button>
                        </td>
                      );
                    })}
                    <td className="px-4 py-3 text-center">
                      <span className={cn(
                        "inline-block min-w-[1.5rem] rounded px-1.5 py-0.5 text-center font-mono text-sm font-bold",
                        "bg-muted",
                      )}>
                        {bitsToDigit(state[row.key])}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {/* Actions */}
          <div className="flex justify-end gap-2 pt-1">
            <Button variant="ghost" onClick={onClose} disabled={saving}>
              Cancel
            </Button>
            <Button onClick={handleSave} disabled={saving || octal.length < 3}>
              {saving
                ? <><Loader2 className="mr-2 h-4 w-4 animate-spin" /> Applying…</>
                : <><Lock className="mr-2 h-4 w-4" /> Apply {octal}</>
              }
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}
