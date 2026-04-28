import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  createServer,
  updateServer,
  deleteServer,
  connectServer,
  provisionServer,
  type ServerInput,
} from "@/lib/api";

export function useCreateServer() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: ServerInput) => createServer(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["servers"] }),
  });
}

export function useUpdateServer() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: ServerInput }) =>
      updateServer(id, input),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["servers"] }),
  });
}

export function useDeleteServer() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => deleteServer(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["servers"] }),
  });
}

export function useConnectServer() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => connectServer(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["servers"] }),
  });
}

export function useProvisionServer() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => provisionServer(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["servers"] }),
  });
}
