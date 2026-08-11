import path from 'node:path';
import type { NextConfig } from 'next';

const nextConfig: NextConfig = {
  transpilePackages: ['@speedplus/ui', '@speedplus/types', '@speedplus/api-client', '@speedplus/utils'],
  images: {
    remotePatterns: [{ protocol: 'https', hostname: '**.speedplus.ng' }],
  },
  // Pin the workspace root to the monorepo, not inferred. An unrelated
  // package.json sitting in the home directory (outside this repo) makes
  // Next.js/Turbopack stop its upward directory walk there instead of
  // reaching pnpm-workspace.yaml, which resolves `next` against the wrong
  // node_modules tree and crashes with "Next.js package not found" after
  // running for a while (module resolution caches, then a fresh lookup
  // hits the misresolved path).
  turbopack: {
    root: path.join(__dirname, '../..'),
  },
};

export default nextConfig;
