import { test, expect } from '@playwright/test';
import os from 'os';
import path from 'path';
import fs from 'fs';

const API = 'http://localhost:8000/api/v1';

// Read from the fixture file the seed script writes, falling back to env vars.
function loadFixtures() {
  const fixturePath = path.join(os.tmpdir(), 'speedplus-e2e-fixtures.env');
  const env: Record<string, string> = {};
  if (fs.existsSync(fixturePath)) {
    for (const line of fs.readFileSync(fixturePath, 'utf8').split('\n')) {
      const [k, v] = line.split('=');
      if (k && v) env[k.trim()] = v.trim();
    }
  }
  return {
    customerPhone: env['CUSTOMER_PHONE'] ?? process.env['E2E_CUSTOMER_PHONE'] ?? '+2349000000001',
    driverPhone:   env['DRIVER_PHONE']   ?? process.env['E2E_DRIVER_PHONE']   ?? '+2349000000002',
    password:      env['SEED_PASSWORD']  ?? process.env['E2E_SEED_PASSWORD']  ?? 'Test1234!',
  };
}

// Logs in via the API and returns the access token.
async function apiLogin(phone: string, password: string): Promise<string> {
  const res = await fetch(`${API}/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ phone, password }),
  });
  const body = await res.json() as { data: { accessToken: string } };
  return body.data.accessToken;
}

test('customer places food order → driver receives and accepts offer', async ({ browser }) => {
  const { customerPhone, driverPhone, password } = loadFixtures();

  // Two isolated browser contexts — one per actor.
  const customerCtx = await browser.newContext({ baseURL: 'http://localhost:3000' });
  const driverCtx   = await browser.newContext({ baseURL: 'http://localhost:3001' });

  const customerPage = await customerCtx.newPage();
  const driverPage   = await driverCtx.newPage();

  try {
    // ── 1. Driver logs in and goes online ────────────────────────────────────
    await driverPage.goto('/login');
    await driverPage.getByLabel('Phone number').fill(driverPhone);
    await driverPage.locator('input[type="password"]').fill(password);
    await driverPage.getByRole('button', { name: /sign in/i }).click();
    await driverPage.waitForURL('/');

    await driverPage.getByRole('button', { name: /offline/i }).click();
    await expect(driverPage.getByRole('button', { name: /online/i })).toBeVisible();

    // ── 2. Customer logs in ──────────────────────────────────────────────────
    await customerPage.goto('/login');
    await customerPage.getByLabel('Phone number').fill(customerPhone);
    await customerPage.locator('input[type="password"]').fill(password);
    await customerPage.getByRole('button', { name: /sign in/i }).click();
    await customerPage.waitForURL('/');

    // ── 3. Customer navigates the food flow ──────────────────────────────────
    await customerPage.goto('/food/menu');

    // data-testid selectors match the buttons rendered in /food/menu and /food/items
    await customerPage.getByTestId('merchant-card').filter({ hasText: /E2E Food Shop/i }).first().click();
    await customerPage.waitForURL('/food/items');

    await customerPage.getByTestId('product-add-btn').filter({ hasText: /E2E Jollof Rice/i }).first().click();
    await customerPage.getByRole('button', { name: /continue/i }).click();
    await customerPage.waitForURL('/food/deliver');

    await customerPage.getByText(/E2E Home/i).click();
    await customerPage.getByRole('button', { name: /continue/i }).click();
    await customerPage.waitForURL('/food/price');

    await expect(customerPage.getByRole('button', { name: /confirm order/i })).toBeEnabled({ timeout: 15_000 });
    await customerPage.getByRole('button', { name: /confirm order/i }).click();
    await customerPage.waitForURL('/food/finding');

    // ── 4. Driver receives the offer ─────────────────────────────────────────
    await expect(driverPage.getByText(/New delivery/i)).toBeVisible({ timeout: 20_000 });

    // ── 5. Driver accepts the offer ──────────────────────────────────────────
    await driverPage.getByRole('button', { name: /accept/i }).click();
    await expect(driverPage.getByText(/Accepted — ride to pickup/i)).toBeVisible({ timeout: 10_000 });

    // ── 6. Confirm order status via API ──────────────────────────────────────
    const customerToken = await apiLogin(customerPhone, password);

    await expect.poll(async () => {
      const res = await customerPage.request.get(`${API}/orders?vertical=food&status=driver_assigned`, {
        headers: { Authorization: `Bearer ${customerToken}` },
      });
      const body = await res.json() as { data: { orders: { status: string }[] } };
      return body.data.orders[0]?.status;
    }, { timeout: 15_000, intervals: [1000] }).toBe('driver_assigned');
  } finally {
    await customerCtx.close().catch(() => {});
    await driverCtx.close().catch(() => {});
  }
});
