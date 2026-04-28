import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  createSite,
  updateSite,
  deleteSite,
  deploySite,
  type SiteInput,
} from "@/lib/api";

export function useCreateSite() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: SiteInput) => createSite(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["sites"] }),
  });
}

export function useUpdateSite() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: SiteInput }) =>
      updateSite(id, input),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["sites"] }),
  });
}

export function useDeleteSite() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => deleteSite(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["sites"] }),
  });
}

export function useDeploySite() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => deploySite(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["sites"] }),
  });
}
