import { useQuery } from "@tanstack/react-query";
import { fetchServerStats, type ServerStats } from "@/lib/api";

export function useServerStats(serverId: string | null) {
  return useQuery<ServerStats>({
    queryKey: ["server-stats", serverId],
    queryFn: () => fetchServerStats(serverId!),
    enabled: Boolean(serverId),
    refetchInterval: 30_000,
    refetchIntervalInBackground: false,
    // Keep last result visible while the next SSH fetch runs in background.
    staleTime: 25_000,
    retry: 1,
    retryDelay: 2000,
  });
}
