import type { NextConfig } from "next";
import path from "path";
import { fileURLToPath } from "url";

const backendUrl = process.env.API_BACKEND_URL ?? "http://localhost:8000";
const frontendRoot = path.dirname(fileURLToPath(import.meta.url));

const nextConfig: NextConfig = {
  // Prevent Next from picking a parent lockfile (e.g. C:\Users\SWAYAM) as the app root,
  // which makes /dashboard/jobs and /dashboard/candidates return 404.
  turbopack: {
    root: frontendRoot,
  },
  async rewrites() {
    return [
      {
        source: "/api/v1/:path*",
        destination: `${backendUrl}/api/v1/:path*`,
      },
    ];
  },
};

export default nextConfig;
