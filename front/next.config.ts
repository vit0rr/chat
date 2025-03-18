import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  experimental: {
    reactCompiler: true,
  },
  env: {
    BACKEND_ROOT_URL: process.env.BACKEND_ROOT_URL,
    BACKEND_WS_ROOT_URL: process.env.BACKEND_WS_ROOT_URL,
    API_KEY: process.env.API_KEY,
  },
};

export default nextConfig;
