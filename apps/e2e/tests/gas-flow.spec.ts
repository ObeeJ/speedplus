import { test, expect } from '@playwright/test';
import os from 'os';
import path from 'path';
import fs from 'fs';

const API = 'http://localhost:8000/api/v1';

function loadFixtures() {
  const fixturePath = path.join(os.tmpdir(), 'fourdat-e2e-fixtures.env');
  const env: Record<string, string> = {};
  if (fs.existsSync(fixturePath)) {
    for (const line of fs.readFileSync(fixturePath, 'utf8').split('\n')) {
      const [k, v] = line.split('=');
      if (k && v) env[k.trim()] = v.trim();
    }
  }
  return {
    customerPhone: env['CUSTOMER_PHONE'] ?? process.env['E2E_CUSTOMER_PHONE'] ?? '+2349000000001',
    password:      env['SEED_PASSWORD']  ?? process.env['E2E_SEED_PASSWORD']  ?? 'Test1234!',
  };
}

async function apiLogin(phone: string, password: string): Promise<string> {
  const res = await fetch(`${API}/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ phone, password }),
  });
  const body = await res.json() as { data: { accessToken: string } };
  return body.data.accessToken;
}

// Gas vertical: cylinder + mode select → delivery address → live LPG price
// quote → order confirm. The merchant and cylinder products for this vertical
// are seeded by migration 022 (Fourdat Gas, deterministic UUID
// 00000000-0000-0000-0000-000000000004), and the cylinder specs the picker
// renders come from migration 028 (cylinder_specs table) — both applied by
// the seed script's migrate-up step, independent of the E2E fixture rows.
test('customer completes a gas cylinder order end-to-end', async ({ page }) => {
  const { customerPhone, password } = loadFixtures();

  await page.goto('/login');
  await page.getByLabel('Phone number').fill(customerPhone);
  await page.locator('input[type="password"]').fill(password);
  await page.getByRole('button', { name: /sign in/i }).click();
  await page.waitForURL('/');

  // ── 1. Cylinder + mode select ──────────────────────────────────────────────
  await page.goto('/gas/cylinder');
  // Label text comes from migration 028's cylinder_specs seed rows.
  await page.getByText(/12\.5 kg — most popular/i).click();
  await page.getByText(/Swap it/i).click();
  await page.getByRole('button', { name: /continue/i }).click();
  await page.waitForURL('/gas/deliver');

  // ── 2. Delivery address ──────────────────────────────────────────────────
  await page.getByText(/E2E Home/i).click();
  await page.getByRole('button', { name: /continue/i }).click();
  await page.waitForURL('/gas/price');

  // ── 3. Price + confirm ───────────────────────────────────────────────────
  await expect(page.getByRole('button', { name: /confirm — find a rider/i })).toBeEnabled({ timeout: 15_000 });
  await page.getByRole('button', { name: /confirm — find a rider/i }).click();
  await page.waitForURL('/gas/finding');

  // ── 4. Verify the order actually landed in Postgres via the API ──────────
  const token = await apiLogin(customerPhone, password);
  await expect.poll(async () => {
    const res = await fetch(`${API}/orders?vertical=gas`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    const body = await res.json() as { data: { orders: { vertical: string; status: string; gasMode?: string }[] } };
    return body.data.orders[0]?.status;
  }, { timeout: 15_000, intervals: [1000] }).not.toBeUndefined();

  const res = await fetch(`${API}/orders?vertical=gas`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  const body = await res.json() as { data: { orders: { id: string; vertical: string; status: string }[] } };
  const order = body.data.orders[0];
  expect(order).toBeTruthy();
  expect(order!.vertical).toBe('gas');
  // Straight after creation the order must not be cancelled/failed — it's
  // either awaiting merchant confirmation or already matching/assigned a rider.
  expect(order!.status).not.toBe('cancelled');
  expect(order!.status).toBeTruthy();
});
