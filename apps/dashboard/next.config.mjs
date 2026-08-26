/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  output: 'standalone',
  env: {
    TRIAGE_ENGINE_URL: process.env.TRIAGE_ENGINE_URL || 'http://localhost:8080',
  },
};

export default nextConfig;
