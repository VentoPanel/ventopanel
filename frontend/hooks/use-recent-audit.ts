import { useQuery } from "@tanstack/react-query";
import { fetchAuditEvents, type AuditPage } from "@/lib/api";

export function useRecentAudit(limit = 8) {
  return useQuery<AuditPage>({
    queryKey: ["audit-recent", limit],
    queryFn: () => fetchAuditEvents({ limit }),
    refetchInterval: 30_000,
    refetchIntervalInBackground: false,
  });
}
