import type { NextConfig } from 'next';

const nextConfig: NextConfig = {
  transpilePackages: ['@speedplus/ui', '@speedplus/types', '@speedplus/api-client', '@speedplus/utils'],
};

export default nextConfig;
