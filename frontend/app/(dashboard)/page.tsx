"use client";

import { useState, useEffect } from "react";
import Link from "next/link";
import { formatDistanceToNow, format } from "date-fns";
import {
  Server, Globe, AlertTriangle, CheckCircle2, Activity,
  ArrowRight, Rocket, TrendingUp, Wifi, WifiOff, RefreshCw,
  TerminalSquare, HardDrive, ScrollText, Container,
} from "lucide-react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  AreaChart, Area, BarChart, Bar,
  XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend,
} from "recharts";
import {
  fetchDashboardSummary, fetchUptimeTrend, fetchDeployTrend,
  type DashboardSummary, type UptimeTrendPoint, type DeployTrendPoint,
} from "@/lib/api";
import { useServers, SERVERS_REFETCH_INTERVAL } from "@/hooks/use-servers";
import { useSites } from "@/hooks/use-sites";
import { useRecentAudit } from "@/hooks/use-recent-audit";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { OnboardingWizard, isOnboardingDone, markOnboardingDone } from "@/components/onboarding-wizard";
import { cn } from "@/lib/utils";

// ─── Status helpers ───────────────────────────────────────────────────────────

const STATUS_COLORS: Record<string, string> = {
  deployed: "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400",
  ssl_pending: "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400",
  connected: "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400",
  ready_for_deploy: "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400",
  provisioned: "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400",
  pending: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400",
  deploying: "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400",
  provisioning: "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400",
  deploy_failed: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
  provision_failed: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
  connection_failed: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
  error: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
  access_denied: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
  failed: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
  draft: "bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-400",
};

function statusColor(s: string) {
  return STATUS_COLORS[s] ?? "bg-gray-100 text-gray-700";
}

function isErrorStatus(status: string) {
  return ["error", "failed", "deploy_failed", "provision_failed",
          "connection_failed", "access_denied"].includes(status);
}

// ─── Stat Card ────────────────────────────────────────────────────────────────

interface StatCardProps {
  title: string;
  value: number | string;
  icon: React.ElementType;
  iconColor?: string;
  sub?: string;
  subColor?: string;
  href?: string;
  loading?: boolean;
  badge?: React.ReactNode;
}

function StatCard({ title, value, icon: Icon, iconColor, sub, subColor, href, loading, badge }: StatCardProps) {
  const content = (
    <Card className={cn(
      "relative overflow-hidden transition-all duration-200",
      href && "hover:shadow-md hover:-translate-y-0.5 cursor-pointer",
    )}>
      <CardContent className="p-5">
        {loading ? (
          <div className="space-y-2">
            <Skeleton className="h-4 w-24" />
            <Skeleton className="h-9 w-16" />
            <Skeleton className="h-3 w-32" />
          </div>
        ) : (
          <div className="flex items-start justify-between gap-2">
            <div className="min-w-0">
              <p className="text-sm font-medium text-muted-foreground">{title}</p>
              <p className="mt-1 text-3xl font-bold tracking-tight">{value}</p>
              {sub && (
                <p className={cn("mt-1 text-xs", subColor ?? "text-muted-foreground")}>{sub}</p>
              )}
            </div>
            <div className={cn(
              "flex h-10 w-10 shrink-0 items-center justify-center rounded-xl",
              iconColor ?? "bg-primary/10 text-primary",
            )}>
              <Icon className="h-5 w-5" />
            </div>
          </div>
        )}
        {badge && <div className="mt-3">{badge}</div>}
      </CardContent>
    </Card>
  );
  return href ? <Link href={href}>{content}</Link> : content;
}

// ─── Mini progress bar ────────────────────────────────────────────────────────

function MiniBar({ value, max, color }: { value: number; max: number; color: string }) {
  const pct = max > 0 ? Math.round((value / max) * 100) : 0;
  return (
    <div className="mt-2 h-1.5 w-full rounded-full bg-muted">
      <div className={cn("h-full rounded-full transition-all", color)} style={{ width: `${pct}%` }} />
    </div>
  );
}

// ─── Quick action ─────────────────────────────────────────────────────────────

function QuickAction({ href, icon: Icon, label }: { href: string; icon: React.ElementType; label: string }) {
  return (
    <Link
      href={href}
      className="flex flex-col items-center gap-1.5 rounded-xl border bg-muted/30 p-3 text-center text-xs font-medium text-muted-foreground transition-all hover:bg-accent hover:text-foreground hover:shadow-sm"
    >
      <Icon className="h-5 w-5" />
      {label}
    </Link>
  );
}

// ─── Main Page ────────────────────────────────────────────────────────────────

export default function DashboardPage() {
  const qc = useQueryClient();

  const { data: servers, isFetching: serversFetching } = useServers();
  const { data: sites,   isFetching: sitesFetching   } = useSites();
  const { data: recentAudit } = useRecentAudit(8);

  const { data: summary, isFetching: summaryFetching } = useQuery<DashboardSummary>({
    queryKey: ["dashboard-summary"],
    queryFn: fetchDashboardSummary,
    refetchInterval: 30_000,
  });

  const { data: uptimeTrend = [] } = useQuery<UptimeTrendPoint[]>({
    queryKey: ["uptime-trend"],
    queryFn: fetchUptimeTrend,
    refetchInterval: 60_000,
  });

  const { data: deployTrend = [] } = useQuery<DeployTrendPoint[]>({
    queryKey: ["deploy-trend"],
    queryFn: fetchDeployTrend,
    refetchInterval: 60_000,
  });

  // Onboarding
  const [showWizard, setShowWizard] = useState(false);
  useEffect(() => {
    if (!isOnboardingDone() && servers !== undefined && servers.length === 0) {
      setShowWizard(true);
    }
  }, [servers]);

  const handleWizardDone = () => {
    markOnboardingDone();
    setShowWizard(false);
    qc.invalidateQueries({ queryKey: ["servers"] });
  };

  const recentEvents = recentAudit?.items ?? [];
  const recentErrors = recentEvents.filter(e => isErrorStatus(e.ToStatus)).length;
  const anyFetching = serversFetching || sitesFetching || summaryFetching;

  // Chart data prep
  const uptimeChartData = uptimeTrend.map(p => ({
    name: format(new Date(p.hour), "HH:mm"),
    up: p.up_count,
    down: p.down_count,
    latency: Math.round(p.avg_latency_ms),
  }));

  const deployChartData = deployTrend.map(p => ({
    name: format(new Date(p.day), "MMM d"),
    success: p.success,
    failed: p.failed,
  }));

  const avgUptime = summary?.uptime.avg_pct ?? 100;

  return (
    <div className="space-y-6">
      {showWizard && <OnboardingWizard onDone={handleWizardDone} />}

      {/* ── Header ── */}
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 className="text-2xl font-bold tracking-tight">Dashboard</h2>
          <p className="text-sm text-muted-foreground">Infrastructure overview</p>
        </div>
        <Button
          variant="ghost" size="sm"
          disabled={anyFetching}
          onClick={() => {
            qc.invalidateQueries({ queryKey: ["servers"] });
            qc.invalidateQueries({ queryKey: ["sites"] });
            qc.invalidateQueries({ queryKey: ["dashboard-summary"] });
            qc.invalidateQueries({ queryKey: ["uptime-trend"] });
            qc.invalidateQueries({ queryKey: ["deploy-trend"] });
            qc.invalidateQueries({ queryKey: ["audit-recent"] });
          }}
        >
          <RefreshCw className={cn("h-4 w-4", anyFetching && "animate-spin")} />
          <span className="ml-1.5 hidden sm:inline">Refresh</span>
        </Button>
      </div>

      {/* ── Stat Cards ── */}
      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <StatCard
          loading={!summary}
          title="Servers"
          value={summary?.servers.total ?? 0}
          icon={Server}
          iconColor={
            (summary?.servers.failed ?? 0) > 0
              ? "bg-red-100 text-red-600 dark:bg-red-900/30 dark:text-red-400"
              : "bg-blue-100 text-blue-600 dark:bg-blue-900/30 dark:text-blue-400"
          }
          sub={
            (summary?.servers.failed ?? 0) > 0
              ? `${summary!.servers.failed} failed · ${summary!.servers.connected} connected`
              : `${summary?.servers.connected ?? 0} connected`
          }
          subColor={(summary?.servers.failed ?? 0) > 0 ? "text-red-600" : undefined}
          href="/servers"
        />
        <StatCard
          loading={!summary}
          title="Sites"
          value={summary?.sites.total ?? 0}
          icon={Globe}
          iconColor={
            (summary?.sites.failed ?? 0) > 0
              ? "bg-red-100 text-red-600 dark:bg-red-900/30 dark:text-red-400"
              : "bg-green-100 text-green-600 dark:bg-green-900/30 dark:text-green-400"
          }
          sub={
            (summary?.sites.failed ?? 0) > 0
              ? `${summary!.sites.failed} failed · ${summary!.sites.deployed} deployed`
              : `${summary?.sites.deployed ?? 0} deployed`
          }
          subColor={(summary?.sites.failed ?? 0) > 0 ? "text-red-600" : undefined}
          href="/sites"
        />
        <StatCard
          loading={!summary}
          title="Uptime"
          value={`${avgUptime.toFixed(1)}%`}
          icon={avgUptime >= 99 ? Wifi : WifiOff}
          iconColor={
            avgUptime >= 99
              ? "bg-green-100 text-green-600 dark:bg-green-900/30 dark:text-green-400"
              : avgUptime >= 95
              ? "bg-yellow-100 text-yellow-600 dark:bg-yellow-900/30 dark:text-yellow-400"
              : "bg-red-100 text-red-600 dark:bg-red-900/30 dark:text-red-400"
          }
          sub={`${summary?.uptime.sites_up ?? 0} up · ${summary?.uptime.sites_down ?? 0} down`}
          subColor={(summary?.uptime.sites_down ?? 0) > 0 ? "text-red-600" : "text-green-600"}
          href="/uptime"
          badge={
            summary && (
              <MiniBar
                value={summary.uptime.sites_up}
                max={summary.uptime.sites_up + summary.uptime.sites_down}
                color="bg-green-500"
              />
            )
          }
        />
        <StatCard
          loading={!summary}
          title="Deploys today"
          value={summary ? (summary.deploys.today_24h_success + summary.deploys.today_24h_failed) : 0}
          icon={Rocket}
          iconColor={
            (summary?.deploys.today_24h_failed ?? 0) > 0
              ? "bg-orange-100 text-orange-600 dark:bg-orange-900/30 dark:text-orange-400"
              : "bg-violet-100 text-violet-600 dark:bg-violet-900/30 dark:text-violet-400"
          }
          sub={
            summary
              ? `${summary.deploys.today_24h_success} ok · ${summary.deploys.today_24h_failed} failed`
              : "last 24h"
          }
          subColor={(summary?.deploys.today_24h_failed ?? 0) > 0 ? "text-orange-600" : undefined}
        />
      </div>

      {/* ── Charts ── */}
      <div className="grid gap-4 lg:grid-cols-2">
        {/* Uptime trend */}
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="flex items-center gap-2 text-sm font-semibold">
              <TrendingUp className="h-4 w-4 text-muted-foreground" />
              Uptime — last 24h
            </CardTitle>
          </CardHeader>
          <CardContent>
            {uptimeTrend.length === 0 ? (
              <div className="flex h-48 items-center justify-center text-sm text-muted-foreground">
                No uptime data yet
              </div>
            ) : (
              <ResponsiveContainer width="100%" height={180}>
                <AreaChart data={uptimeChartData} margin={{ top: 4, right: 4, left: -20, bottom: 0 }}>
                  <defs>
                    <linearGradient id="gradUp" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%"  stopColor="#22c55e" stopOpacity={0.3} />
                      <stop offset="95%" stopColor="#22c55e" stopOpacity={0} />
                    </linearGradient>
                    <linearGradient id="gradDown" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%"  stopColor="#ef4444" stopOpacity={0.3} />
                      <stop offset="95%" stopColor="#ef4444" stopOpacity={0} />
                    </linearGradient>
                  </defs>
                  <CartesianGrid strokeDasharray="3 3" stroke="hsl(var(--border))" />
                  <XAxis dataKey="name" tick={{ fontSize: 11 }} stroke="hsl(var(--muted-foreground))" />
                  <YAxis tick={{ fontSize: 11 }} stroke="hsl(var(--muted-foreground))" />
                  <Tooltip
                    contentStyle={{
                      background: "hsl(var(--popover))",
                      border: "1px solid hsl(var(--border))",
                      borderRadius: 8,
                      fontSize: 12,
                    }}
                  />
                  <Legend wrapperStyle={{ fontSize: 12 }} />
                  <Area type="monotone" dataKey="up"   name="Up"   stroke="#22c55e" fill="url(#gradUp)"   strokeWidth={2} />
                  <Area type="monotone" dataKey="down" name="Down" stroke="#ef4444" fill="url(#gradDown)" strokeWidth={2} />
                </AreaChart>
              </ResponsiveContainer>
            )}
          </CardContent>
        </Card>

        {/* Deploy trend */}
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="flex items-center gap-2 text-sm font-semibold">
              <Rocket className="h-4 w-4 text-muted-foreground" />
              Deploy activity — last 7 days
            </CardTitle>
          </CardHeader>
          <CardContent>
            {deployTrend.length === 0 ? (
              <div className="flex h-48 items-center justify-center text-sm text-muted-foreground">
                No deploy data yet
              </div>
            ) : (
              <ResponsiveContainer width="100%" height={180}>
                <BarChart data={deployChartData} margin={{ top: 4, right: 4, left: -20, bottom: 0 }}>
                  <CartesianGrid strokeDasharray="3 3" stroke="hsl(var(--border))" />
                  <XAxis dataKey="name" tick={{ fontSize: 11 }} stroke="hsl(var(--muted-foreground))" />
                  <YAxis tick={{ fontSize: 11 }} stroke="hsl(var(--muted-foreground))" allowDecimals={false} />
                  <Tooltip
                    contentStyle={{
                      background: "hsl(var(--popover))",
                      border: "1px solid hsl(var(--border))",
                      borderRadius: 8,
                      fontSize: 12,
                    }}
                  />
                  <Legend wrapperStyle={{ fontSize: 12 }} />
                  <Bar dataKey="success" name="Success" fill="#22c55e" radius={[4, 4, 0, 0]} />
                  <Bar dataKey="failed"  name="Failed"  fill="#ef4444" radius={[4, 4, 0, 0]} />
                </BarChart>
              </ResponsiveContainer>
            )}
          </CardContent>
        </Card>
      </div>

      {/* ── Bottom row: Activity + Servers/Sites + Quick access ── */}
      <div className="grid gap-4 lg:grid-cols-3">

        {/* Recent Activity */}
        <Card className="lg:col-span-1">
          <CardHeader className="pb-2">
            <CardTitle className="flex items-center justify-between text-sm font-semibold">
              <span className="flex items-center gap-2">
                <Activity className="h-4 w-4 text-muted-foreground" />
                Recent Activity
              </span>
              <Button variant="ghost" size="sm" asChild>
                <Link href="/audit" className="flex items-center gap-1 text-xs text-muted-foreground">
                  All <ArrowRight className="h-3 w-3" />
                </Link>
              </Button>
            </CardTitle>
          </CardHeader>
          <CardContent className="p-0">
            {recentEvents.length === 0 ? (
              <p className="px-4 py-6 text-center text-sm text-muted-foreground">No events yet.</p>
            ) : (
              <ul className="divide-y">
                {recentEvents.slice(0, 6).map(event => (
                  <li key={event.ID} className="flex items-start gap-3 px-4 py-2.5 text-xs">
                    <Badge
                      variant="outline"
                      className={cn("mt-0.5 shrink-0 capitalize", statusColor(event.ToStatus))}
                    >
                      {event.ToStatus}
                    </Badge>
                    <div className="min-w-0 flex-1">
                      <span className="font-mono text-muted-foreground">{event.ResourceType}</span>
                      {event.Reason && (
                        <p className="truncate text-muted-foreground">{event.Reason}</p>
                      )}
                    </div>
                    <span className="shrink-0 whitespace-nowrap text-muted-foreground">
                      {formatDistanceToNow(new Date(event.CreatedAt), { addSuffix: true })}
                    </span>
                  </li>
                ))}
              </ul>
            )}
          </CardContent>
        </Card>

        {/* Servers + Sites quick lists */}
        <div className="flex flex-col gap-4 lg:col-span-1">
          <Card className="flex-1">
            <CardHeader className="pb-2">
              <CardTitle className="flex items-center justify-between text-sm font-semibold">
                <span className="flex items-center gap-2">
                  <Server className="h-4 w-4 text-muted-foreground" />
                  Servers
                </span>
                <Button variant="ghost" size="sm" asChild>
                  <Link href="/servers" className="flex items-center gap-1 text-xs text-muted-foreground">
                    Manage <ArrowRight className="h-3 w-3" />
                  </Link>
                </Button>
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-1">
              {!servers || servers.length === 0 ? (
                <p className="text-sm text-muted-foreground">No servers yet.</p>
              ) : (
                servers.slice(0, 4).map(s => (
                  <Link
                    key={s.ID}
                    href={`/servers/${s.ID}`}
                    className="flex items-center justify-between rounded-md px-2 py-1.5 text-sm hover:bg-accent"
                  >
                    <span className="truncate font-medium">{s.Name}</span>
                    <Badge variant="outline" className={cn("ml-2 shrink-0 text-xs", statusColor(s.Status))}>
                      {s.Status}
                    </Badge>
                  </Link>
                ))
              )}
            </CardContent>
          </Card>

          <Card className="flex-1">
            <CardHeader className="pb-2">
              <CardTitle className="flex items-center justify-between text-sm font-semibold">
                <span className="flex items-center gap-2">
                  <Globe className="h-4 w-4 text-muted-foreground" />
                  Sites
                </span>
                <Button variant="ghost" size="sm" asChild>
                  <Link href="/sites" className="flex items-center gap-1 text-xs text-muted-foreground">
                    Manage <ArrowRight className="h-3 w-3" />
                  </Link>
                </Button>
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-1">
              {!sites || sites.length === 0 ? (
                <p className="text-sm text-muted-foreground">No sites yet.</p>
              ) : (
                sites.slice(0, 4).map(s => (
                  <Link
                    key={s.ID}
                    href={`/sites/${s.ID}`}
                    className="flex items-center justify-between rounded-md px-2 py-1.5 text-sm hover:bg-accent"
                  >
                    <span className="truncate font-medium">{s.Domain}</span>
                    <Badge variant="outline" className={cn("ml-2 shrink-0 text-xs", statusColor(s.Status))}>
                      {s.Status}
                    </Badge>
                  </Link>
                ))
              )}
            </CardContent>
          </Card>
        </div>

        {/* Quick access tools */}
        <Card className="lg:col-span-1">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-semibold">Quick Access</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-3 gap-2">
              <QuickAction href="/terminal"     icon={TerminalSquare} label="Terminal" />
              <QuickAction href="/monitor"      icon={Activity}       label="Monitor"  />
              <QuickAction href="/logs"         icon={ScrollText}     label="Logs"     />
              <QuickAction href="/files"        icon={HardDrive}      label="Files"    />
              <QuickAction href="/nginx"        icon={Container}      label="Nginx"    />
              <QuickAction href="/audit"        icon={recentErrors > 0 ? AlertTriangle : CheckCircle2} label="Audit" />
            </div>

            {/* Error alert */}
            {recentErrors > 0 && (
              <div className="mt-4 flex items-center gap-2 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-xs text-red-700 dark:border-red-900/50 dark:bg-red-900/20 dark:text-red-400">
                <AlertTriangle className="h-3.5 w-3.5 shrink-0" />
                {recentErrors} recent error{recentErrors > 1 ? "s" : ""} — check Audit Log
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
