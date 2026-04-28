"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { isTokenValid } from "@/lib/api";
import { Nav } from "@/components/nav";

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
    <div className="flex h-screen overflow-hidden">
      <Nav />
      <main className="flex-1 overflow-y-auto p-8">{children}</main>
    </div>
  );
}
