"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useTheme } from "next-themes";
import {
  LayoutDashboard, Server, Globe, LogOut,
  ClipboardList, Settings, Users, Activity, DatabaseBackup, BarChart2, Layers, ShieldCheck, HardDrive, TerminalSquare,
  Sun, Moon,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { clearToken } from "@/lib/api";
import { Button } from "@/components/ui/button";

const links = [
  { href: "/", label: "Dashboard", icon: LayoutDashboard },
  { href: "/servers", label: "Servers", icon: Server },
  { href: "/sites", label: "Sites", icon: Globe },
  { href: "/uptime", label: "Uptime", icon: Activity },
  { href: "/observability", label: "Observability", icon: BarChart2 },
  { href: "/templates", label: "Templates", icon: Layers },
  { href: "/backups", label: "Backups", icon: DatabaseBackup },
  { href: "/audit", label: "Audit Log", icon: ClipboardList },
  { href: "/files", label: "Files", icon: HardDrive },
  { href: "/terminal", label: "Terminal", icon: TerminalSquare },
  { href: "/users", label: "Team", icon: Users },
  { href: "/security", label: "Security", icon: ShieldCheck },
  { href: "/settings", label: "Settings", icon: Settings },
];

export function Nav() {
  const pathname  = usePathname();
  const router    = useRouter();
  const { theme, setTheme, resolvedTheme } = useTheme();

  function handleLogout() {
    clearToken();
    router.push("/login");
  }

  const isDark = resolvedTheme === "dark";

  return (
    <aside className="flex h-screen w-56 flex-col border-r bg-background px-3 py-4">
      <div className="mb-6 px-3">
        <h1 className="text-lg font-bold tracking-tight text-foreground">
          VentoPanel
        </h1>
        <p className="text-xs text-muted-foreground">Control Panel</p>
      </div>

      <nav className="flex flex-1 flex-col gap-1 overflow-y-auto">
        {links.map(({ href, label, icon: Icon }) => (
          <Link
            key={href}
            href={href}
            className={cn(
              "flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors",
              pathname === href
                ? "bg-primary text-primary-foreground"
                : "text-muted-foreground hover:bg-accent hover:text-accent-foreground",
            )}
          >
            <Icon className="h-4 w-4" />
            {label}
          </Link>
        ))}
      </nav>

      {/* Bottom actions */}
      <div className="mt-auto flex flex-col gap-1 pt-2 border-t">
        {/* Theme toggle */}
        <button
          onClick={() => setTheme(isDark ? "light" : "dark")}
          className="flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium text-muted-foreground hover:bg-accent hover:text-accent-foreground transition-colors w-full"
          title={isDark ? "Switch to light mode" : "Switch to dark mode"}
        >
          {isDark
            ? <Sun  className="h-4 w-4" />
            : <Moon className="h-4 w-4" />
          }
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
    </aside>
  );
}
