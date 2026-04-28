import { useQuery } from "@tanstack/react-query";
import { fetchServers, type Server } from "@/lib/api";

export const SERVERS_REFETCH_INTERVAL = 15_000;

export function useServers() {
  return useQuery<Server[]>({
    queryKey: ["servers"],
    queryFn: fetchServers,
    refetchInterval: SERVERS_REFETCH_INTERVAL,
    refetchIntervalInBackground: false,
  });
}
