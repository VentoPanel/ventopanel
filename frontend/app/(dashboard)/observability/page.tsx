"use client";

import {
  BarChart,
  Bar,
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Legend,
} from "recharts";
import { format, parseISO } from "date-fns";
import { useQuery } from "@tanstack/react-query";
import {
  fetchDashboardSummary,
  fetchUptimeTrend,
  fetchDeployTrend,
  type DashboardSummary,
  type UptimeTrendPoint,
  type DeployTrendPoint,
} from "@/lib/api";
import {
  BarChart2,
  Server,
  Globe,
  Activity,
  Rocket,
  CheckCircle2,
  XCircle,
  RefreshCw,
} from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";

// ── Small stat card ──────────────────────────────────────────────────────────

function StatCard({
  icon: Icon,
  label,
  value,
  sub,
  color = "text-foreground",
}: {
  icon: React.ElementType;
  label: string;
  value: string | number;
  sub?: string;
  color?: string;
}) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between pb-2">
        <CardTitle className="text-sm font-medium text-muted-foreground">
          {label}
        </CardTitle>
        <Icon className="h-4 w-4 text-muted-foreground" />
      </CardHeader>
      <CardContent>
        <p className={cn("text-3xl font-bold", color)}>{value}</p>
        {sub && <p className="mt-1 text-xs text-muted-foreground">{sub}</p>}
      </CardContent>
    </Card>
  );
}

function StatCardSkeleton() {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between pb-2">
        <Skeleton className="h-4 w-24" />
        <Skeleton className="h-4 w-4 rounded" />
      </CardHeader>
      <CardContent>
        <Skeleton className="h-9 w-20" />
        <Skeleton className="mt-1 h-3 w-32" />
      </CardContent>
    </Card>
  );
}

// ── Page ─────────────────────────────────────────────────────────────────────

export default function ObservabilityPage() {
  const { data: summary, isLoading: summaryLoading } =
    useQuery<DashboardSummary>({
      queryKey: ["dashboard-summary"],
      queryFn: fetchDashboardSummary,
      refetchInterval: 60_000,
      staleTime: 55_000,
    });

  const { data: uptimeTrend = [], isLoading: uptimeLoading } = useQuery<
    UptimeTrendPoint[]
  >({
    queryKey: ["dashboard-uptime-trend"],
    queryFn: fetchUptimeTrend,
    refetchInterval: 120_000,
    staleTime: 110_000,
  });

  const { data: deployTrend = [], isLoading: deployLoading } = useQuery<
    DeployTrendPoint[]
  >({
    queryKey: ["dashboard-deploy-trend"],
    queryFn: fetchDeployTrend,
    refetchInterval: 120_000,
    staleTime: 110_000,
  });

  // Format uptime trend data for chart
  const uptimeChartData = uptimeTrend.map((p) => ({
    time: format(parseISO(p.hour), "HH:mm"),
    Up: p.up_count,
    Down: p.down_count,
    "Avg ms": Math.round(p.avg_latency_ms),
  }));

  // Format deploy trend data for chart
  const deployChartData = deployTrend.map((p) => ({
    day: format(parseISO(p.day), "dd MMM"),
    Success: p.success,
    Failed: p.failed,
  }));

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-3">
        <BarChart2 className="h-6 w-6 text-muted-foreground" />
        <div>
          <h2 className="text-2xl font-bold tracking-tight">Observability</h2>
          <p className="text-sm text-muted-foreground">
            Platform metrics · auto-refreshes every minute
          </p>
        </div>
      </div>

      {/* Summary cards */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {summaryLoading ? (
          <>
            {[0, 1, 2, 3].map((i) => <StatCardSkeleton key={i} />)}
          </>
        ) : summary ? (
          <>
            <StatCard
              icon={Globe}
              label="Sites"
              value={summary.sites.total}
              sub={`${summary.sites.deployed} deployed · ${summary.sites.failed} failed`}
              color={summary.sites.failed > 0 ? "text-red-600" : "text-green-700"}
            />
            <StatCard
              icon={Server}
              label="Servers"
              value={summary.servers.total}
              sub={`${summary.servers.connected} connected · ${summary.servers.failed} failed`}
              color={summary.servers.failed > 0 ? "text-red-600" : "text-green-700"}
            />
            <StatCard
              icon={Activity}
              label="Uptime (avg)"
              value={`${summary.uptime.avg_pct.toFixed(1)}%`}
              sub={`${summary.uptime.sites_up} up · ${summary.uptime.sites_down} down`}
              color={
                summary.uptime.sites_down > 0
                  ? "text-red-600"
                  : summary.uptime.avg_pct < 99
                  ? "text-yellow-600"
                  : "text-green-700"
              }
            />
            <StatCard
              icon={Rocket}
              label="Deploys (24h)"
              value={summary.deploys.today_24h_success + summary.deploys.today_24h_failed}
              sub={`${summary.deploys.today_24h_success} ok · ${summary.deploys.today_24h_failed} failed · ${summary.deploys.all_time_success} total`}
              color={summary.deploys.today_24h_failed > 0 ? "text-yellow-600" : "text-foreground"}
            />
          </>
        ) : null}
      </div>

      {/* Second row: success/failure badge row */}
      {summary && (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <Card className="bg-green-50 border-green-200 dark:bg-green-950/20 dark:border-green-900">
            <CardContent className="flex items-center gap-3 pt-4">
              <CheckCircle2 className="h-5 w-5 text-green-600 shrink-0" />
              <div>
                <p className="text-sm font-medium text-green-800 dark:text-green-300">All-time successful deploys</p>
                <p className="text-2xl font-bold text-green-700 dark:text-green-400">{summary.deploys.all_time_success}</p>
              </div>
            </CardContent>
          </Card>
          <Card className="bg-red-50 border-red-200 dark:bg-red-950/20 dark:border-red-900">
            <CardContent className="flex items-center gap-3 pt-4">
              <XCircle className="h-5 w-5 text-red-500 shrink-0" />
              <div>
                <p className="text-sm font-medium text-red-800 dark:text-red-300">All-time failed deploys</p>
                <p className="text-2xl font-bold text-red-600 dark:text-red-400">{summary.deploys.all_time_failed}</p>
              </div>
            </CardContent>
          </Card>
          <Card className="bg-blue-50 border-blue-200 dark:bg-blue-950/20 dark:border-blue-900">
            <CardContent className="flex items-center gap-3 pt-4">
              <Activity className="h-5 w-5 text-blue-600 shrink-0" />
              <div>
                <p className="text-sm font-medium text-blue-800 dark:text-blue-300">Sites monitored (uptime)</p>
                <p className="text-2xl font-bold text-blue-700 dark:text-blue-400">
                  {summary.uptime.sites_up + summary.uptime.sites_down}
                </p>
              </div>
            </CardContent>
          </Card>
          <Card className="bg-purple-50 border-purple-200 dark:bg-purple-950/20 dark:border-purple-900">
            <CardContent className="flex items-center gap-3 pt-4">
              <RefreshCw className="h-5 w-5 text-purple-600 shrink-0" />
              <div>
                <p className="text-sm font-medium text-purple-800 dark:text-purple-300">Deploying right now</p>
                <p className="text-2xl font-bold text-purple-700 dark:text-purple-400">{summary.sites.deploying}</p>
              </div>
            </CardContent>
          </Card>
        </div>
      )}

      {/* Charts row */}
      <div className="grid gap-6 lg:grid-cols-2">
        {/* Uptime checks — last 24h */}
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="flex items-center gap-2 text-sm font-medium">
              <Activity className="h-4 w-4 text-muted-foreground" />
              Uptime checks — last 24h
            </CardTitle>
          </CardHeader>
          <CardContent>
            {uptimeLoading ? (
              <Skeleton className="h-48 w-full" />
            ) : uptimeChartData.length === 0 ? (
              <p className="py-12 text-center text-sm text-muted-foreground">
                No uptime checks recorded yet.
              </p>
            ) : (
              <ResponsiveContainer width="100%" height={220}>
                <BarChart data={uptimeChartData} margin={{ top: 4, right: 8, left: -20, bottom: 0 }}>
                  <CartesianGrid strokeDasharray="3 3" stroke="hsl(var(--border))" />
                  <XAxis dataKey="time" tick={{ fontSize: 11 }} />
                  <YAxis tick={{ fontSize: 11 }} />
                  <Tooltip
                    contentStyle={{ fontSize: 12, borderRadius: 6 }}
                    labelStyle={{ fontWeight: 600 }}
                  />
                  <Legend wrapperStyle={{ fontSize: 12 }} />
                  <Bar dataKey="Up" fill="#22c55e" radius={[2, 2, 0, 0]} />
                  <Bar dataKey="Down" fill="#ef4444" radius={[2, 2, 0, 0]} />
                </BarChart>
              </ResponsiveContainer>
            )}
          </CardContent>
        </Card>

        {/* Deploy activity — last 7 days */}
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="flex items-center gap-2 text-sm font-medium">
              <Rocket className="h-4 w-4 text-muted-foreground" />
              Deploy activity — last 7 days
            </CardTitle>
          </CardHeader>
          <CardContent>
            {deployLoading ? (
              <Skeleton className="h-48 w-full" />
            ) : deployChartData.length === 0 ? (
              <p className="py-12 text-center text-sm text-muted-foreground">
                No deploys in the last 7 days.
              </p>
            ) : (
              <ResponsiveContainer width="100%" height={220}>
                <BarChart data={deployChartData} margin={{ top: 4, right: 8, left: -20, bottom: 0 }}>
                  <CartesianGrid strokeDasharray="3 3" stroke="hsl(var(--border))" />
                  <XAxis dataKey="day" tick={{ fontSize: 11 }} />
                  <YAxis allowDecimals={false} tick={{ fontSize: 11 }} />
                  <Tooltip
                    contentStyle={{ fontSize: 12, borderRadius: 6 }}
                    labelStyle={{ fontWeight: 600 }}
                  />
                  <Legend wrapperStyle={{ fontSize: 12 }} />
                  <Bar dataKey="Success" fill="#3b82f6" radius={[2, 2, 0, 0]} />
                  <Bar dataKey="Failed" fill="#f97316" radius={[2, 2, 0, 0]} />
                </BarChart>
              </ResponsiveContainer>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Latency sparkline */}
      {uptimeChartData.length > 0 && (
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="flex items-center gap-2 text-sm font-medium">
              <Activity className="h-4 w-4 text-muted-foreground" />
              Average response latency — last 24h (ms)
            </CardTitle>
          </CardHeader>
          <CardContent>
            <ResponsiveContainer width="100%" height={160}>
              <LineChart data={uptimeChartData} margin={{ top: 4, right: 8, left: -20, bottom: 0 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="hsl(var(--border))" />
                <XAxis dataKey="time" tick={{ fontSize: 11 }} />
                <YAxis tick={{ fontSize: 11 }} unit="ms" />
                <Tooltip
                  formatter={(v: number) => [`${v} ms`, "Latency"]}
                  contentStyle={{ fontSize: 12, borderRadius: 6 }}
                />
                <Line
                  type="monotone"
                  dataKey="Avg ms"
                  stroke="#8b5cf6"
                  strokeWidth={2}
                  dot={false}
                  activeDot={{ r: 4 }}
                />
              </LineChart>
            </ResponsiveContainer>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
