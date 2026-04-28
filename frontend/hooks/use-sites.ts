import { useQuery } from "@tanstack/react-query";
import { fetchSites, type Site } from "@/lib/api";

export function useSites() {
  return useQuery<Site[]>({
    queryKey: ["sites"],
    queryFn: fetchSites,
  });
}
