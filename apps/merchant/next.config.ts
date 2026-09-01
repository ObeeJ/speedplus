import path from 'node:path';
import type { NextConfig } from 'next';

const nextConfig: NextConfig = {
  transpilePackages: ['@fourdat/ui', '@fourdat/types', '@fourdat/api-client', '@fourdat/utils'],
  // See apps/customer/next.config.ts for why this is pinned explicitly.
  turbopack: {
    root: path.join(__dirname, '../..'),
  },
};

export default nextConfig;
