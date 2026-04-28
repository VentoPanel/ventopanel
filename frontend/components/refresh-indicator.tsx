"use client";

import { useEffect, useState } from "react";
import { Loader2, RefreshCw } from "lucide-react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";

interface RefreshIndicatorProps {
  isFetching: boolean;
  dataUpdatedAt: number;
  onRefresh: () => void;
  intervalSeconds?: number;
  className?: string;
}

function formatTime(ts: number): string {
  if (!ts) return "—";
  return new Date(ts).toLocaleTimeString();
}

export function RefreshIndicator({
  isFetching,
  dataUpdatedAt,
  onRefresh,
  intervalSeconds = 15,
  className,
}: RefreshIndicatorProps) {
  const [, tick] = useState(0);

  // Re-render every second to keep "X s ago" fresh
  useEffect(() => {
    const id = setInterval(() => tick((n) => n + 1), 1000);
    return () => clearInterval(id);
  }, []);

  const secondsAgo = dataUpdatedAt
    ? Math.floor((Date.now() - dataUpdatedAt) / 1000)
    : null;

  return (
    <div
      className={cn(
        "flex items-center gap-2 text-xs text-muted-foreground",
        className,
      )}
    >
      {isFetching ? (
        <>
          <Loader2 className="h-3 w-3 animate-spin" />
          <span>Refreshing…</span>
        </>
      ) : (
        <>
          <span className="h-2 w-2 rounded-full bg-green-400" />
          <span>
            {secondsAgo !== null
              ? secondsAgo < 5
                ? "Updated just now"
                : `Updated ${secondsAgo}s ago`
              : `Auto-refresh: every ${intervalSeconds}s`}
          </span>
        </>
      )}
      <Button
        variant="ghost"
        size="icon"
        className="h-6 w-6"
        onClick={onRefresh}
        disabled={isFetching}
        title="Refresh now"
      >
        <RefreshCw className={cn("h-3 w-3", isFetching && "animate-spin")} />
      </Button>
    </div>
  );
}
