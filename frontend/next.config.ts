import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "standalone",
  // API requests are proxied via app/api/v1/[...path]/route.ts at runtime,
  // so the backend URL is read from process.env.API_URL (server-side, runtime).
};

export default nextConfig;
