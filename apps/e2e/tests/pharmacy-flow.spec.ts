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

// Pharmacy vertical, OTC (over-the-counter) branch: pharmacy picker → item
// select → delivery address → price confirm → order placed.
//
// This exercises the "E2E Pharmacy" merchant + "E2E Paracetamol 500mg" OTC
// product added to apps/api/scripts/seed/main.go for this task — no pharmacy
// merchant existed anywhere in the migration history before that change, so
// apps/customer/app/pharmacy/page.tsx (catalogApi.listMerchants('pharmacy'))
// previously always rendered its "No pharmacies are available" empty state
// and this flow could not be completed at all.
test('customer completes an OTC pharmacy order end-to-end', async ({ page }) => {
  const { customerPhone, password } = loadFixtures();

  await page.goto('/login');
  await page.getByLabel('Phone number').fill(customerPhone);
  await page.locator('input[type="password"]').fill(password);
  await page.getByRole('button', { name: /sign in/i }).click();
  await page.waitForURL('/');

  // ── 1. Pharmacy picker ───────────────────────────────────────────────────
  await page.goto('/pharmacy');
  await page.getByText(/E2E Pharmacy/i).click();
  await page.waitForURL('/pharmacy/items');

  // ── 2. OTC item select (default tab) ─────────────────────────────────────
  await expect(page.getByText(/Everyday medicine/i)).toBeVisible();
  await page.getByText(/E2E Paracetamol 500mg/i).click();
  await page.getByRole('button', { name: /continue/i }).click();
  await page.waitForURL('/pharmacy/deliver');

  // ── 3. Delivery address ──────────────────────────────────────────────────
  await page.getByText(/E2E Home/i).click();
  await page.getByRole('button', { name: /continue/i }).click();
  await page.waitForURL('/pharmacy/price');

  // ── 4. Price + confirm ───────────────────────────────────────────────────
  await expect(page.getByRole('button', { name: /confirm order/i })).toBeEnabled({ timeout: 15_000 });
  await page.getByRole('button', { name: /confirm order/i }).click();
  await page.waitForURL('/pharmacy/finding');

  // ── 5. Verify the order actually landed in Postgres via the API ──────────
  const token = await apiLogin(customerPhone, password);
  await expect.poll(async () => {
    const res = await fetch(`${API}/orders?vertical=pharmacy`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    const body = await res.json() as { data: { orders: { status: string }[] } };
    return body.data.orders[0]?.status;
  }, { timeout: 15_000, intervals: [1000] }).toBeTruthy();

  const res = await fetch(`${API}/orders?vertical=pharmacy`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  const body = await res.json() as {
    data: { orders: { id: string; vertical: string; status: string; items: { name: string }[] }[] };
  };
  const order = body.data.orders[0];
  expect(order).toBeTruthy();
  expect(order!.vertical).toBe('pharmacy');
  expect(order!.status).not.toBe('cancelled');
  expect(order!.items.some((i) => /Paracetamol/i.test(i.name))).toBe(true);
});
