"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { fetchTemplates, type SiteTemplate } from "@/lib/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";
import { Layers, ChevronDown, ChevronUp, Copy, Check } from "lucide-react";

const RUNTIME_COLOR: Record<string, string> = {
  node: "bg-green-100 text-green-700 border-green-200",
  python: "bg-blue-100 text-blue-700 border-blue-200",
  php: "bg-purple-100 text-purple-700 border-purple-200",
  go: "bg-cyan-100 text-cyan-700 border-cyan-200",
  static: "bg-gray-100 text-gray-700 border-gray-200",
};

const RUNTIME_BORDER: Record<string, string> = {
  node: "border-l-green-400",
  python: "border-l-blue-400",
  php: "border-l-purple-400",
  go: "border-l-cyan-400",
  static: "border-l-gray-400",
};

function TemplateCard({ t }: { t: SiteTemplate }) {
  const [expanded, setExpanded] = useState(false);
  const [copied, setCopied] = useState(false);

  function copyDockerfile() {
    navigator.clipboard.writeText(t.dockerfile).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    });
  }

  return (
    <Card className={cn("border-l-4", RUNTIME_BORDER[t.runtime] ?? "border-l-muted")}>
      <CardHeader className="pb-2">
        <div className="flex items-start justify-between gap-3">
          <div className="space-y-1">
            <CardTitle className="flex items-center gap-2 text-base">
              <span
                className={cn(
                  "rounded-full border px-2.5 py-0.5 text-xs font-medium",
                  RUNTIME_COLOR[t.runtime] ?? "bg-muted text-muted-foreground",
                )}
              >
                {t.runtime}
              </span>
              {t.name}
            </CardTitle>
            <p className="text-sm text-muted-foreground leading-relaxed">{t.description}</p>
          </div>
        </div>
        <div className="flex flex-wrap gap-1 pt-1">
          {t.tags.map((tag) => (
            <span
              key={tag}
              className="rounded bg-muted px-2 py-0.5 text-xs text-muted-foreground font-mono"
            >
              {tag}
            </span>
          ))}
        </div>
      </CardHeader>
      <CardContent className="space-y-2 pt-0">
        <div className="flex items-center gap-2 text-xs text-muted-foreground">
          <span>Healthcheck:</span>
          <code className="rounded bg-muted px-1.5 py-0.5 font-mono text-foreground">
            {t.healthcheck_path}
          </code>
        </div>

        <button
          onClick={() => setExpanded((v) => !v)}
          className="flex items-center gap-1.5 text-xs text-muted-foreground hover:text-foreground transition-colors"
        >
          {expanded ? <ChevronUp className="h-3.5 w-3.5" /> : <ChevronDown className="h-3.5 w-3.5" />}
          {expanded ? "Hide Dockerfile" : "Show Dockerfile"}
        </button>

        {expanded && (
          <div className="relative">
            <pre className="rounded-md bg-black p-4 font-mono text-xs text-green-300 leading-relaxed overflow-x-auto max-h-80 overflow-y-auto whitespace-pre">
              {t.dockerfile}
            </pre>
            <Button
              variant="ghost"
              size="sm"
              className="absolute right-2 top-2 h-7 gap-1.5 bg-white/10 text-white hover:bg-white/20 text-xs"
              onClick={copyDockerfile}
            >
              {copied ? <Check className="h-3.5 w-3.5" /> : <Copy className="h-3.5 w-3.5" />}
              {copied ? "Copied" : "Copy"}
            </Button>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

const FILTERS = ["all", "node", "python", "php", "go", "static"];

export default function TemplatesPage() {
  const [filter, setFilter] = useState("all");

  const { data: templates = [], isLoading } = useQuery<SiteTemplate[]>({
    queryKey: ["templates"],
    queryFn: fetchTemplates,
    staleTime: Infinity,
  });

  const filtered = filter === "all" ? templates : templates.filter((t) => t.runtime === filter);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold tracking-tight">
          <Layers className="h-6 w-6" />
          Framework Templates
        </h1>
        <p className="mt-1 text-sm text-muted-foreground">
          Pre-built, production-ready Dockerfiles for popular frameworks. Select a template when
          creating a site — VentoPanel will write the Dockerfile automatically on every deploy.
        </p>
      </div>

      {/* Runtime filter */}
      <div className="flex flex-wrap gap-2">
        {FILTERS.map((f) => (
          <button
            key={f}
            onClick={() => setFilter(f)}
            className={cn(
              "rounded-full px-3 py-1 text-sm font-medium transition-colors capitalize",
              filter === f
                ? "bg-primary text-primary-foreground"
                : "bg-muted text-muted-foreground hover:bg-muted/80",
            )}
          >
            {f === "all" ? `All (${templates.length})` : f}
          </button>
        ))}
      </div>

      {isLoading ? (
        <div className="grid gap-4 sm:grid-cols-2">
          {[0, 1, 2, 3].map((i) => (
            <Card key={i} className="border-l-4 border-l-muted">
              <CardHeader className="pb-2 space-y-2">
                <Skeleton className="h-5 w-32" />
                <Skeleton className="h-4 w-full" />
                <Skeleton className="h-4 w-3/4" />
              </CardHeader>
            </Card>
          ))}
        </div>
      ) : filtered.length === 0 ? (
        <p className="text-sm text-muted-foreground">No templates for this runtime.</p>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2">
          {filtered.map((t) => (
            <TemplateCard key={t.id} t={t} />
          ))}
        </div>
      )}
    </div>
  );
}
