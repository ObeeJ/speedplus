import type { NextConfig } from 'next';

const nextConfig: NextConfig = {
  transpilePackages: ['@speedplus/ui', '@speedplus/types', '@speedplus/api-client', '@speedplus/utils'],
  images: {
    remotePatterns: [{ protocol: 'https', hostname: '**.speedplus.ng' }],
  },
};

export default nextConfig;
