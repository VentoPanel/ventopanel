"use client";

import { useState, useEffect } from "react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useTheme } from "next-themes";
import {
  LayoutDashboard, Server, Globe, LogOut,
  ClipboardList, Settings, Users, Activity, DatabaseBackup,
  BarChart2, Layers, ShieldCheck, HardDrive, TerminalSquare,
  Sun, Moon, MonitorDot, Search, ScrollText, Container,
  Menu, X,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { clearToken } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { openCommandPalette } from "@/components/command-palette";

const links = [
  { href: "/",             label: "Dashboard",   icon: LayoutDashboard },
  { href: "/servers",      label: "Servers",      icon: Server },
  { href: "/sites",        label: "Sites",        icon: Globe },
  { href: "/uptime",       label: "Uptime",       icon: Activity },
  { href: "/observability",label: "Observability",icon: BarChart2 },
  { href: "/templates",    label: "Templates",    icon: Layers },
  { href: "/backups",      label: "Backups",      icon: DatabaseBackup },
  { href: "/audit",        label: "Audit Log",    icon: ClipboardList },
  { href: "/files",        label: "Files",        icon: HardDrive },
  { href: "/terminal",     label: "Terminal",     icon: TerminalSquare },
  { href: "/monitor",      label: "Monitor",      icon: MonitorDot },
  { href: "/logs",         label: "Logs",         icon: ScrollText },
  { href: "/nginx",        label: "Nginx",        icon: Container },
  { href: "/users",        label: "Team",         icon: Users },
  { href: "/security",     label: "Security",     icon: ShieldCheck },
  { href: "/settings",     label: "Settings",     icon: Settings },
];

// ─── Shared nav link list ─────────────────────────────────────────────────────

function NavLinks({
  pathname,
  onNavigate,
}: {
  pathname: string;
  onNavigate?: () => void;
}) {
  return (
    <nav className="flex flex-1 flex-col gap-1 overflow-y-auto">
      {links.map(({ href, label, icon: Icon }) => (
        <Link
          key={href}
          href={href}
          onClick={onNavigate}
          className={cn(
            "flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors",
            pathname === href
              ? "bg-primary text-primary-foreground"
              : "text-muted-foreground hover:bg-accent hover:text-accent-foreground",
          )}
        >
          <Icon className="h-4 w-4 shrink-0" />
          {label}
        </Link>
      ))}
    </nav>
  );
}

// ─── Bottom actions (theme + logout) ─────────────────────────────────────────

function BottomActions({ onNavigate }: { onNavigate?: () => void }) {
  const router = useRouter();
  const { setTheme, resolvedTheme } = useTheme();
  const isDark = resolvedTheme === "dark";

  function handleLogout() {
    clearToken();
    onNavigate?.();
    router.push("/login");
  }

  return (
    <div className="mt-auto flex flex-col gap-1 border-t pt-2">
      <button
        onClick={() => setTheme(isDark ? "light" : "dark")}
        className="flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium text-muted-foreground hover:bg-accent hover:text-accent-foreground transition-colors w-full"
      >
        {isDark ? <Sun className="h-4 w-4" /> : <Moon className="h-4 w-4" />}
        {isDark ? "Light mode" : "Dark mode"}
      </button>
      <Button
        variant="ghost"
        size="sm"
        className="justify-start gap-3 text-muted-foreground"
        onClick={handleLogout}
      >
        <LogOut className="h-4 w-4" />
        Logout
      </Button>
    </div>
  );
}

// ─── Desktop sidebar ──────────────────────────────────────────────────────────

function DesktopSidebar({ pathname }: { pathname: string }) {
  return (
    <aside className="hidden lg:flex h-screen w-56 shrink-0 flex-col border-r bg-background px-3 py-4">
      <div className="mb-6 px-3">
        <h1 className="text-lg font-bold tracking-tight text-foreground">VentoPanel</h1>
        <p className="text-xs text-muted-foreground">Control Panel</p>
      </div>

      <button
        onClick={openCommandPalette}
        className="mb-3 flex w-full items-center gap-2 rounded-md border border-border bg-muted/50 px-3 py-1.5 text-sm text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground"
      >
        <Search className="h-3.5 w-3.5 shrink-0" />
        <span className="flex-1 text-left">Search…</span>
        <kbd className="hidden rounded border bg-background px-1 py-0.5 text-[10px] font-mono sm:inline">⌘K</kbd>
      </button>

      <NavLinks pathname={pathname} />
      <BottomActions />
    </aside>
  );
}

// ─── Mobile header + drawer ───────────────────────────────────────────────────

function MobileNav({ pathname }: { pathname: string }) {
  const [open, setOpen] = useState(false);

  // Close on route change
  useEffect(() => { setOpen(false); }, [pathname]);

  // Lock body scroll when drawer is open
  useEffect(() => {
    if (open) document.body.style.overflow = "hidden";
    else document.body.style.overflow = "";
    return () => { document.body.style.overflow = ""; };
  }, [open]);

  const currentPage = links.find(l => l.href === pathname);

  return (
    <>
      {/* Top header bar */}
      <header className="lg:hidden flex h-14 shrink-0 items-center justify-between border-b bg-background px-4">
        <div className="flex items-center gap-2">
          <button
            onClick={() => setOpen(true)}
            className="rounded-md p-1.5 text-muted-foreground hover:bg-accent hover:text-foreground transition-colors"
            aria-label="Open menu"
          >
            <Menu className="h-5 w-5" />
          </button>
          <span className="font-bold tracking-tight">VentoPanel</span>
          {currentPage && (
            <span className="text-muted-foreground text-sm">/ {currentPage.label}</span>
          )}
        </div>
        <button
          onClick={openCommandPalette}
          className="rounded-md p-1.5 text-muted-foreground hover:bg-accent hover:text-foreground transition-colors"
          aria-label="Search"
        >
          <Search className="h-5 w-5" />
        </button>
      </header>

      {/* Backdrop */}
      {open && (
        <div
          className="lg:hidden fixed inset-0 z-40 bg-black/50 backdrop-blur-sm"
          onClick={() => setOpen(false)}
        />
      )}

      {/* Drawer */}
      <div
        className={cn(
          "lg:hidden fixed inset-y-0 left-0 z-50 flex w-72 flex-col border-r bg-background px-3 py-4 shadow-2xl transition-transform duration-300",
          open ? "translate-x-0" : "-translate-x-full",
        )}
      >
        {/* Drawer header */}
        <div className="mb-6 flex items-center justify-between px-3">
          <div>
            <h1 className="text-lg font-bold tracking-tight">VentoPanel</h1>
            <p className="text-xs text-muted-foreground">Control Panel</p>
          </div>
          <button
            onClick={() => setOpen(false)}
            className="rounded-md p-1.5 text-muted-foreground hover:bg-accent hover:text-foreground transition-colors"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Search */}
        <button
          onClick={() => { setOpen(false); openCommandPalette(); }}
          className="mb-3 flex w-full items-center gap-2 rounded-md border border-border bg-muted/50 px-3 py-2 text-sm text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground"
        >
          <Search className="h-3.5 w-3.5 shrink-0" />
          <span className="flex-1 text-left">Search…</span>
        </button>

        <NavLinks pathname={pathname} onNavigate={() => setOpen(false)} />
        <BottomActions onNavigate={() => setOpen(false)} />
      </div>
    </>
  );
}

// ─── Exported Nav ─────────────────────────────────────────────────────────────

export function Nav() {
  const pathname = usePathname();
  return (
    <>
      <DesktopSidebar pathname={pathname} />
      <MobileNav pathname={pathname} />
    </>
  );
}
