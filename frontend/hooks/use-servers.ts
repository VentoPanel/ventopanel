import { useQuery } from "@tanstack/react-query";
import { fetchServers, type Server } from "@/lib/api";

export function useServers() {
  return useQuery<Server[]>({
    queryKey: ["servers"],
    queryFn: fetchServers,
  });
}
