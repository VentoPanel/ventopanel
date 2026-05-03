import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "standalone",
  // API requests are proxied via app/api/v1/[...path]/route.ts at runtime,
  // so the backend URL is read from process.env.API_URL (server-side, runtime).
  //
  // SWC minifier in Next.js 15.3.x has a known bug ("returnNaN is not defined")
  // that crashes the production server. Disable webpack minimisation as a workaround
  // until the upstream fix lands.
  webpack: (config, { dev }) => {
    if (!dev) {
      config.optimization.minimize = false;
    }
    return config;
  },
};

export default nextConfig;
