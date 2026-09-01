import { execSync } from 'child_process';
import path from 'path';

export default function globalSetup() {
  const dbUrl = process.env['DATABASE_URL'] ?? 'postgres://fourdat:fourdat@localhost:5433/fourdat?sslmode=disable';

  if (!process.env['DATABASE_URL']) {
    console.warn('[e2e] DATABASE_URL not set — falling back to localhost:5433. Set it explicitly in CI.');
  }

  const apiDir = path.resolve(__dirname, '../../api');
  execSync('go run ./scripts/seed/main.go', {
    cwd: apiDir,
    stdio: 'inherit',
    env: {
      ...process.env,
      DATABASE_URL: dbUrl,
      ALLOW_DESTRUCTIVE_SEED: 'true',
      E2E_SEED_PASSWORD:  process.env['E2E_SEED_PASSWORD']  ?? 'Test1234!',
      E2E_CUSTOMER_PHONE: process.env['E2E_CUSTOMER_PHONE'] ?? '+2349000000001',
      E2E_DRIVER_PHONE:   process.env['E2E_DRIVER_PHONE']   ?? '+2349000000002',
      E2E_MERCHANT_PHONE: process.env['E2E_MERCHANT_PHONE'] ?? '+2349000000003',
    },
  });
}
