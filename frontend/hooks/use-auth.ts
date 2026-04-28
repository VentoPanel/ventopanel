"use client";

import { useMemo } from "react";
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
