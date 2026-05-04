"use client";

import { useEffect, useMemo } from "react";
import { useRouter } from "next/navigation";
import { getRole } from "@/lib/api";

export interface AuthInfo {
  role: string;
  isAdmin: boolean;
  isEditor: boolean; // admin or editor
  canWrite: boolean; // admin or editor
}

export function useAuth(): AuthInfo {
  const role = useMemo(() => {
    if (typeof window === "undefined") return "";
    return getRole();
  }, []);

  return {
    role,
    isAdmin: role === "admin",
    isEditor: role === "editor",
    canWrite: role === "admin" || role === "editor",
  };
}

// Redirects non-admin users to "/" — use at the top of admin-only pages.
export function useAdminGuard() {
  const router = useRouter();
  const { isAdmin } = useAuth();
  useEffect(() => {
    if (typeof window !== "undefined" && getRole() && getRole() !== "admin") {
      router.replace("/");
    }
  }, [isAdmin, router]);
}
