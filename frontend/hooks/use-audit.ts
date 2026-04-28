import { useInfiniteQuery } from "@tanstack/react-query";
import { fetchAuditEvents, type AuditPage } from "@/lib/api";

const PAGE_SIZE = 50;

export function useAuditEvents(filters?: {
  resource_type?: string;
  resource_id?: string;
}) {
  return useInfiniteQuery<AuditPage, Error>({
    queryKey: ["audit", filters],
    queryFn: ({ pageParam }) =>
      fetchAuditEvents({
        ...filters,
        limit: PAGE_SIZE,
        before: (pageParam as string) || undefined,
      }),
    initialPageParam: "",
    getNextPageParam: (lastPage) => lastPage.next_cursor || undefined,
  });
}
