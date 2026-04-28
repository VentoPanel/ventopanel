import { useQuery } from "@tanstack/react-query";
import { fetchServerStats, type ServerStats } from "@/lib/api";

export function useServerStats(serverId: string | null) {
  return useQuery<ServerStats>({
    queryKey: ["server-stats", serverId],
    queryFn: () => fetchServerStats(serverId!),
    enabled: Boolean(serverId),
    refetchInterval: 30_000,
    refetchIntervalInBackground: false,
    retry: 1,
  });
}
