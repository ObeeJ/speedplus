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

// Package delivery, single drop-off branch: pickup/drop-off address + recipient
// → size/weight → price + wallet payment confirm → order placed.
//
// The package merchant (Fourdat Logistics, migration 015) is referenced by
// NEXT_PUBLIC_PACKAGE_MERCHANT_ID in apps/customer/.env.local
// (00000000-0000-0000-0000-000000000002) — if that env var is ever unset,
// PACKAGE_MERCHANT_ID in package/price/page.tsx silently falls back to '' and
// order creation fails with a merchant-not-found error from the API, so this
// spec doubles as a regression check for that env wiring.
test('customer completes a single drop-off package order end-to-end', async ({ page }) => {
  const { customerPhone, password } = loadFixtures();

  await page.goto('/login');
  await page.getByLabel('Phone number').fill(customerPhone);
  await page.locator('input[type="password"]').fill(password);
  await page.getByRole('button', { name: /sign in/i }).click();
  await page.waitForURL('/');

  // ── 1. Pickup / drop-off / recipient ─────────────────────────────────────
  await page.goto('/package/where');
  // Both pickup and drop-off pickers list the same saved-address set — the
  // only saved address is "E2E Home", so pick it for both roles in order.
  const addressButtons = page.getByText(/E2E Home/i);
  await addressButtons.first().click(); // pickup
  await addressButtons.nth(1).click();  // drop-off

  await page.getByLabel(/Recipient name/i).fill('Amaka Obi');
  await page.getByLabel(/Recipient phone/i).fill('08012345678');
  await page.getByRole('button', { name: /continue/i }).click();
  await page.waitForURL('/package/what');

  // ── 2. Size + weight ──────────────────────────────────────────────────────
  // "Medium" is ambiguous (both a size and a weight bucket render that
  // label) — disambiguate by clicking each SelectionCard's description text.
  await page.getByText(/Shoebox or bag/i).click();
  await page.getByText(/3 – 10 kg/i).click();
  await page.getByRole('button', { name: /see price/i }).click();
  await page.waitForURL('/package/price');

  // ── 3. Price, wallet payment, consent, confirm ────────────────────────────
  await expect(page.getByText(/Available balance/i)).toBeVisible({ timeout: 15_000 });
  await page.getByText(/Pay from wallet/i).click();
  await page.getByText(/I confirm the recipient has agreed/i).click();

  const confirmBtn = page.getByRole('button', { name: /confirm & find rider/i });
  await expect(confirmBtn).toBeEnabled({ timeout: 15_000 });
  await confirmBtn.click();
  await page.waitForURL('/package/finding');

  // ── 4. Verify the order actually landed in Postgres via the API ──────────
  const token = await apiLogin(customerPhone, password);
  await expect.poll(async () => {
    const res = await fetch(`${API}/orders?vertical=package`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    const body = await res.json() as { data: { orders: { status: string }[] } };
    return body.data.orders[0]?.status;
  }, { timeout: 15_000, intervals: [1000] }).toBeTruthy();

  const res = await fetch(`${API}/orders?vertical=package`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  const body = await res.json() as {
    data: { orders: { id: string; vertical: string; status: string; items: { name: string }[] }[] };
  };
  const order = body.data.orders[0];
  expect(order).toBeTruthy();
  expect(order!.vertical).toBe('package');
  expect(order!.status).not.toBe('cancelled');
  // Note: the recipient name/phone entered in the UI is not present on the
  // customer-facing GET /orders response (apps/api/internal/dto/order.go's
  // OrderResponse has no recipientName/recipientPhone field, even though the
  // MerchantOrder type used by the merchant portal does) — see report.
  expect(order!.items.length).toBeGreaterThan(0);
});
