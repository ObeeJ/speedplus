import path from 'node:path';
import type { NextConfig } from 'next';

const nextConfig: NextConfig = {
  transpilePackages: ['@speedplus/ui', '@speedplus/types', '@speedplus/api-client', '@speedplus/utils'],
  // See apps/customer/next.config.ts for why this is pinned explicitly.
  turbopack: {
    root: path.join(__dirname, '../..'),
  },
};

export default nextConfig;
