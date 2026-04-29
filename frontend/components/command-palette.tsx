"use client";

import {
  useState, useEffect, useRef, useCallback, useMemo,
} from "react";
import { useRouter } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import {
  Search, Server, Globe, HardDrive, TerminalSquare, MonitorDot,
  LayoutDashboard, ShieldCheck, Users, ClipboardList, Settings,
  Activity, DatabaseBackup, BarChart2, Layers, ArrowRight, X, ScrollText,
} from "lucide-react";
import { fetchServers, fetchSites } from "@/lib/api";
import { cn } from "@/lib/utils";

// ─── Static pages ─────────────────────────────────────────────────────────────

const PAGES = [
  { id: "dashboard",    label: "Dashboard",     href: "/",              icon: LayoutDashboard, group: "Pages" },
  { id: "servers",      label: "Servers",        href: "/servers",       icon: Server,          group: "Pages" },
  { id: "sites",        label: "Sites",          href: "/sites",         icon: Globe,           group: "Pages" },
  { id: "files",        label: "File Manager",   href: "/files",         icon: HardDrive,       group: "Pages" },
  { id: "terminal",     label: "Web Terminal",   href: "/terminal",      icon: TerminalSquare,  group: "Pages" },
  { id: "monitor",      label: "Resource Monitor",href: "/monitor",      icon: MonitorDot,      group: "Pages" },
  { id: "logs",         label: "Log Viewer",     href: "/logs",          icon: ScrollText,      group: "Pages" },
  { id: "uptime",       label: "Uptime",         href: "/uptime",        icon: Activity,        group: "Pages" },
  { id: "observability",label: "Observability",  href: "/observability", icon: BarChart2,       group: "Pages" },
  { id: "templates",    label: "Templates",      href: "/templates",     icon: Layers,          group: "Pages" },
  { id: "backups",      label: "Backups",        href: "/backups",       icon: DatabaseBackup,  group: "Pages" },
  { id: "audit",        label: "Audit Log",      href: "/audit",         icon: ClipboardList,   group: "Pages" },
  { id: "users",        label: "Team",           href: "/users",         icon: Users,           group: "Pages" },
  { id: "security",     label: "Security",       href: "/security",      icon: ShieldCheck,     group: "Pages" },
  { id: "settings",     label: "Settings",       href: "/settings",      icon: Settings,        group: "Pages" },
];

// ─── Types ────────────────────────────────────────────────────────────────────

interface Item {
  id: string;
  label: string;
  sub?: string;
  href: string;
  icon: React.ElementType;
  group: string;
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

function match(text: string, query: string): boolean {
  if (!query) return true;
  const q = query.toLowerCase();
  return text.toLowerCase().includes(q);
}

function highlight(text: string, query: string) {
  if (!query) return <>{text}</>;
  const idx = text.toLowerCase().indexOf(query.toLowerCase());
  if (idx === -1) return <>{text}</>;
  return (
    <>
      {text.slice(0, idx)}
      <mark className="bg-primary/20 text-primary rounded-sm">{text.slice(idx, idx + query.length)}</mark>
      {text.slice(idx + query.length)}
    </>
  );
}

// ─── Context / provider hook ──────────────────────────────────────────────────

let _openFn: (() => void) | null = null;

export function openCommandPalette() { _openFn?.(); }

// ─── Component ────────────────────────────────────────────────────────────────

export function CommandPalette() {
  const router = useRouter();
  const [open, setOpen]   = useState(false);
  const [query, setQuery] = useState("");
  const [active, setActive] = useState(0);
  const inputRef  = useRef<HTMLInputElement>(null);
  const listRef   = useRef<HTMLDivElement>(null);

  // Register opener for external callers.
  useEffect(() => { _openFn = () => setOpen(true); return () => { _openFn = null; }; }, []);

  // Cmd+K / Ctrl+K global shortcut.
  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if ((e.metaKey || e.ctrlKey) && e.key === "k") {
        e.preventDefault();
        setOpen((v) => !v);
      }
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);

  // Focus input when opened.
  useEffect(() => {
    if (open) {
      setQuery("");
      setActive(0);
      setTimeout(() => inputRef.current?.focus(), 30);
    }
  }, [open]);

  // Fetch servers + sites (use cached data if available).
  const { data: servers } = useQuery({ queryKey: ["servers"], queryFn: fetchServers, staleTime: 30_000 });
  const { data: sites }   = useQuery({ queryKey: ["sites"],   queryFn: fetchSites,   staleTime: 30_000 });

  // Build full item list.
  const allItems: Item[] = useMemo(() => {
    const serverItems: Item[] = (servers ?? []).map((s) => ({
      id: `server-${s.ID}`,
      label: s.Name,
      sub: s.Host,
      href: `/servers/${s.ID}`,
      icon: Server,
      group: "Servers",
    }));

    const siteItems: Item[] = (sites ?? []).map((s) => ({
      id: `site-${s.ID}`,
      label: s.Name,
      sub: s.Domain,
      href: `/sites/${s.ID}`,
      icon: Globe,
      group: "Sites",
    }));

    return [...PAGES, ...serverItems, ...siteItems];
  }, [servers, sites]);

  // Filtered items.
  const filtered = useMemo(() => {
    if (!query.trim()) return allItems.slice(0, 12);
    return allItems.filter((i) => match(i.label + " " + (i.sub ?? ""), query));
  }, [allItems, query]);

  // Group filtered results.
  const groups = useMemo(() => {
    const map = new Map<string, Item[]>();
    filtered.forEach((item) => {
      const g = map.get(item.group) ?? [];
      g.push(item);
      map.set(item.group, g);
    });
    return map;
  }, [filtered]);

  // Flat list for keyboard navigation.
  const flatFiltered = useMemo(() => filtered, [filtered]);

  const navigate = useCallback((href: string) => {
    setOpen(false);
    router.push(href);
  }, [router]);

  // Keyboard navigation inside the palette.
  function onKeyDown(e: React.KeyboardEvent) {
    if (e.key === "Escape")     { setOpen(false); return; }
    if (e.key === "ArrowDown")  { e.preventDefault(); setActive((v) => Math.min(v + 1, flatFiltered.length - 1)); }
    if (e.key === "ArrowUp")    { e.preventDefault(); setActive((v) => Math.max(v - 1, 0)); }
    if (e.key === "Enter")      { flatFiltered[active] && navigate(flatFiltered[active].href); }
  }

  // Scroll active item into view.
  useEffect(() => {
    const el = listRef.current?.querySelector(`[data-idx="${active}"]`) as HTMLElement | null;
    el?.scrollIntoView({ block: "nearest" });
  }, [active]);

  if (!open) return null;

  return (
    <div
      className="fixed inset-0 z-50 flex items-start justify-center pt-[15vh] px-4"
      onMouseDown={(e) => { if (e.target === e.currentTarget) setOpen(false); }}
    >
      {/* Backdrop */}
      <div className="fixed inset-0 bg-black/50 backdrop-blur-sm" onClick={() => setOpen(false)} />

      {/* Palette */}
      <div className="relative w-full max-w-lg rounded-xl border bg-background shadow-2xl overflow-hidden">

        {/* Input */}
        <div className="flex items-center gap-3 border-b px-4 py-3">
          <Search className="h-4 w-4 text-muted-foreground shrink-0" />
          <input
            ref={inputRef}
            value={query}
            onChange={(e) => { setQuery(e.target.value); setActive(0); }}
            onKeyDown={onKeyDown}
            placeholder="Search pages, servers, sites…"
            className="flex-1 bg-transparent text-sm outline-none placeholder:text-muted-foreground"
          />
          {query && (
            <button onClick={() => setQuery("")} className="text-muted-foreground hover:text-foreground">
              <X className="h-4 w-4" />
            </button>
          )}
          <kbd className="hidden sm:inline-flex h-5 items-center gap-1 rounded border bg-muted px-1.5 text-[10px] font-medium text-muted-foreground">
            ESC
          </kbd>
        </div>

        {/* Results */}
        <div ref={listRef} className="max-h-[380px] overflow-y-auto py-2">
          {filtered.length === 0 ? (
            <p className="py-8 text-center text-sm text-muted-foreground">
              No results for &ldquo;{query}&rdquo;
            </p>
          ) : (
            (() => {
              let globalIdx = 0;
              return Array.from(groups.entries()).map(([group, items]) => (
                <div key={group}>
                  <p className="px-4 py-1.5 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
                    {group}
                  </p>
                  {items.map((item) => {
                    const idx = globalIdx++;
                    const isActive = idx === active;
                    return (
                      <button
                        key={item.id}
                        data-idx={idx}
                        onMouseEnter={() => setActive(idx)}
                        onClick={() => navigate(item.href)}
                        className={cn(
                          "flex w-full items-center gap-3 px-4 py-2.5 text-sm transition-colors",
                          isActive ? "bg-accent text-accent-foreground" : "hover:bg-accent/50",
                        )}
                      >
                        <div className={cn(
                          "flex h-7 w-7 shrink-0 items-center justify-center rounded-md",
                          isActive ? "bg-primary/10" : "bg-muted",
                        )}>
                          <item.icon className={cn("h-3.5 w-3.5", isActive ? "text-primary" : "text-muted-foreground")} />
                        </div>
                        <div className="min-w-0 flex-1 text-left">
                          <span className="block truncate font-medium">
                            {highlight(item.label, query)}
                          </span>
                          {item.sub && (
                            <span className="block truncate text-xs text-muted-foreground">
                              {highlight(item.sub, query)}
                            </span>
                          )}
                        </div>
                        {isActive && <ArrowRight className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />}
                      </button>
                    );
                  })}
                </div>
              ));
            })()
          )}
        </div>

        {/* Footer hint */}
        <div className="border-t px-4 py-2 flex items-center gap-4 text-[11px] text-muted-foreground">
          <span className="flex items-center gap-1">
            <kbd className="rounded border bg-muted px-1 py-0.5 font-mono">↑↓</kbd> navigate
          </span>
          <span className="flex items-center gap-1">
            <kbd className="rounded border bg-muted px-1 py-0.5 font-mono">↵</kbd> open
          </span>
          <span className="flex items-center gap-1">
            <kbd className="rounded border bg-muted px-1 py-0.5 font-mono">ESC</kbd> close
          </span>
        </div>
      </div>
    </div>
  );
}
