import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: './tests',
  timeout: 120_000,
  retries: 0,
  workers: 1,

  globalSetup: './fixtures/global-setup.ts',

  use: {
    baseURL: 'http://localhost:3000',
    trace: 'on-first-retry',
  },

  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],

  webServer: [
    {
      // Go API — must be up before the Next.js apps make API calls on boot.
      command: 'go run github.com/speedplus/api/cmd/server',
      url: 'http://localhost:8000/healthz',
      reuseExistingServer: true,
      timeout: 60_000,
      cwd: '../api',
      env: {
        PORT: '8000',
        ENVIRONMENT: 'development',
        DATABASE_URL: process.env['DATABASE_URL'] ?? 'postgres://speedplus:speedplus@localhost:5433/speedplus?sslmode=disable',
        REDIS_URL: process.env['REDIS_URL'] ?? 'redis://localhost:6379',
        JWT_SECRET: process.env['JWT_SECRET'] ?? '6465766c6f63616c6a77747365637265746b65796d757374626532636861727',
        PAYCODE_SECRET: process.env['PAYCODE_SECRET'] ?? 'e67ab93ef03d304e88186fce474400b8dfc3833f28b38ab8394153ff2a104ac2',
        QUOTE_SECRET: process.env['QUOTE_SECRET'] ?? '679165bc06f47ca9034fc078930c93b64d5c4778d95826146be24aa66bdb28de',
        // 32 raw bytes — matches docker-compose ENCRYPTION_KEY
        ENCRYPTION_KEY: process.env['ENCRYPTION_KEY'] ?? 'devlocalencryptionkey32byteslong',
        OSRM_URL: process.env['OSRM_URL'] ?? 'http://router.project-osrm.org',
        ALLOWED_ORIGINS: 'http://localhost:3000,http://localhost:3001',
        JWT_ACCESS_TTL_MIN: '15',
        JWT_REFRESH_TTL_DAYS: '30',
      },
    },
    {
      command: 'pnpm --filter @speedplus/customer dev',
      url: 'http://localhost:3000',
      reuseExistingServer: true,
      timeout: 120_000,
      cwd: '../..',
    },
    {
      command: 'pnpm --filter @speedplus/driver exec next dev --turbopack --port 3001',
      url: 'http://localhost:3001',
      reuseExistingServer: true,
      timeout: 120_000,
      cwd: '../..',
    },
  ],
});
