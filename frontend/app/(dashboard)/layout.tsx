"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { isTokenValid } from "@/lib/api";
import { Nav } from "@/components/nav";
import { CommandPalette } from "@/components/command-palette";

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const router = useRouter();

  useEffect(() => {
    if (!isTokenValid()) {
      router.replace("/login");
    }
  }, [router]);

  return (
    // Desktop: side-by-side (flex-row). Mobile: stacked (flex-col).
    <div className="flex h-screen flex-col overflow-hidden lg:flex-row">
      <Nav />
      <main className="flex-1 overflow-y-auto p-4 lg:p-8">{children}</main>
      <CommandPalette />
    </div>
  );
}
