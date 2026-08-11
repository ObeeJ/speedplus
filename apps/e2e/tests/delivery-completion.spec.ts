import { test, expect } from '@playwright/test';
import os from 'os';
import path from 'path';
import fs from 'fs';

const API = 'http://localhost:8000/api/v1';

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

async function apiLogin(phone: string, password: string): Promise<string> {
  const res = await fetch(`${API}/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ phone, password }),
  });
  const body = await res.json() as { data: { accessToken: string } };
  return body.data.accessToken;
}

// Polls until the predicate returns a truthy value or timeout is reached.
async function pollUntil<T>(
  fn: () => Promise<T>,
  predicate: (v: T) => boolean,
  { timeout = 20_000, interval = 1_000 } = {},
): Promise<T> {
  const deadline = Date.now() + timeout;
  while (Date.now() < deadline) {
    const v = await fn();
    if (predicate(v)) return v;
    await new Promise((r) => setTimeout(r, interval));
  }
  throw new Error('pollUntil timed out');
}

test('driver completes delivery → escrow releases → order is delivered', async ({ browser }) => {
  const { customerPhone, driverPhone, password } = loadFixtures();

  const customerCtx = await browser.newContext({ baseURL: 'http://localhost:3000' });
  const driverCtx   = await browser.newContext({ baseURL: 'http://localhost:3001' });
  const customerPage = await customerCtx.newPage();
  const driverPage   = await driverCtx.newPage();

  try {
    // ── 1. Driver logs in and goes online ─────────────────────────────────────
    await driverPage.goto('/login');
    await driverPage.getByLabel('Phone number').fill(driverPhone);
    await driverPage.locator('input[type="password"]').fill(password);
    await driverPage.getByRole('button', { name: /sign in/i }).click();
    await driverPage.waitForURL('/');
    await driverPage.getByRole('button', { name: /offline/i }).click();
    await expect(driverPage.getByRole('button', { name: /online/i })).toBeVisible();

    // ── 2. Customer logs in and places a food order ───────────────────────────
    await customerPage.goto('/login');
    await customerPage.getByLabel('Phone number').fill(customerPhone);
    await customerPage.locator('input[type="password"]').fill(password);
    await customerPage.getByRole('button', { name: /sign in/i }).click();
    await customerPage.waitForURL('/');

    // ── 3. Get tokens — API is guaranteed up after page navigations succeed ───
    const customerToken = await apiLogin(customerPhone, password);
    const driverToken   = await apiLogin(driverPhone, password);

    await customerPage.goto('/food/menu');
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

    // ── 4. Driver accepts the offer ───────────────────────────────────────────
    await expect(driverPage.getByText(/New delivery/i)).toBeVisible({ timeout: 20_000 });
    await driverPage.getByRole('button', { name: /accept/i }).click();
    await expect(driverPage.getByText(/Accepted — ride to pickup/i)).toBeVisible({ timeout: 10_000 });

    // ── 5. Fetch the active order ID via API ──────────────────────────────────
    const orderId = await pollUntil(
      async () => {
        const res = await fetch(`${API}/orders?vertical=food&status=driver_assigned`, {
          headers: { Authorization: `Bearer ${customerToken}` },
        });
        const body = await res.json() as { data: { orders: { id: string; status: string }[] } };
        return body.data.orders[0]?.id ?? null;
      },
      (id) => id !== null,
      { timeout: 15_000 },
    );

    // ── 6. Advance driver through stages to POD ───────────────────────────────
    // Stage 1 → 2: arrived at pickup
    await driverPage.getByRole('button', { name: /arrived at pickup/i }).click();
    // Stage 2 → 3: picked up package
    await driverPage.getByRole('button', { name: /I have the package/i }).click();
    // Stage 3 → 4: arrived at drop-off
    await driverPage.getByRole('button', { name: /arrived at drop-off/i }).click();

    // POD screen should now be visible
    await expect(driverPage.getByText(/Proof of delivery/i)).toBeVisible({ timeout: 10_000 });

    // ── 7. Fetch the delivery code via API (driver endpoint) ──────────────────
    // The delivery code is sent to the customer — in E2E we read it directly
    // from the API using the driver token to simulate the customer sharing it.
    const deliveryCode = await pollUntil(
      async () => {
        const res = await fetch(`${API}/orders/${orderId}`, {
          headers: { Authorization: `Bearer ${driverToken}` },
        });
        const body = await res.json() as { data: { deliveryCode?: string } };
        return body.data.deliveryCode ?? null;
      },
      (code) => code !== null && code.length === 6,
      { timeout: 15_000 },
    );

    // ── 8. Driver enters the delivery code ────────────────────────────────────
    await driverPage.locator('input[inputmode="numeric"][maxlength="6"]').fill(deliveryCode!);
    await driverPage.getByRole('button', { name: /confirm delivery/i }).click();

    // ── 9. Assert order is delivered ─────────────────────────────────────────
    await expect.poll(async () => {
      const res = await fetch(`${API}/orders/${orderId}`, {
        headers: { Authorization: `Bearer ${customerToken}` },
      });
      const body = await res.json() as { data: { status: string } };
      return body.data.status;
    }, { timeout: 15_000, intervals: [1000] }).toBe('delivered');

    // ── 10. Assert escrow was released (driver earnings > 0) ─────────────────
    await expect.poll(async () => {
      const res = await fetch(`${API}/wallet`, {
        headers: { Authorization: `Bearer ${driverToken}` },
      });
      const body = await res.json() as { data: { balanceKobo: number } };
      return body.data.balanceKobo;
    }, { timeout: 15_000, intervals: [1000] }).toBeGreaterThan(0);

  } finally {
    await customerCtx.close().catch(() => {});
    await driverCtx.close().catch(() => {});
  }
});
