import { useQuery } from "@tanstack/react-query";
import { fetchSites, type Site } from "@/lib/api";

export const SITES_REFETCH_INTERVAL = 15_000;

export function useSites() {
  return useQuery<Site[]>({
    queryKey: ["sites"],
    queryFn: fetchSites,
    refetchInterval: SITES_REFETCH_INTERVAL,
    refetchIntervalInBackground: false,
  });
}
