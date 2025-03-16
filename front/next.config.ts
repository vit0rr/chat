import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  env: {
    BACKEND_ROOT_URL: process.env.BACKEND_ROOT_URL,
    BACKEND_WS_ROOT_URL: process.env.BACKEND_WS_ROOT_URL,
  },
};

export default nextConfig;
