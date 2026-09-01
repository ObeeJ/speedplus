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

// Package delivery E2E: a single-drop, non-food vertical with a distinct
// merchant model — the customer app's /package flow hits a platform-owned
// "Fourdat Logistics" merchant at the deterministic ID
// 00000000-0000-0000-0000-000000000002 (seeded via migration 015), rather
// than a merchant the customer chooses from a list like food/gas/pharmacy.
test('driver completes a package delivery → escrow releases → wallet shows earnings', async ({ browser }) => {
  const { customerPhone, driverPhone, password } = loadFixtures();

  // geolocation + permissions: package/where's "Use my current location" drop-off
  // button (there's only one seeded address, "E2E Home", so pickup uses that
  // address card and drop-off must come from GPS) calls navigator.geolocation
  // then reverse-geocodes via nominatim.openstreetmap.org — if that fetch fails
  // (e.g. no network in CI) the page code catches it and falls back to a
  // synthetic "Current location" / "Lagos" address using the same coords, so
  // this works offline too.
  const customerCtx = await browser.newContext({
    baseURL: 'http://localhost:3000',
    geolocation: { latitude: 6.46, longitude: 3.4 },
    permissions: ['geolocation'],
  });
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

    // ── 2. Customer logs in ────────────────────────────────────────────────────
    await customerPage.goto('/login');
    await customerPage.getByLabel('Phone number').fill(customerPhone);
    await customerPage.locator('input[type="password"]').fill(password);
    await customerPage.getByRole('button', { name: /sign in/i }).click();
    await customerPage.waitForURL('/');

    const customerToken = await apiLogin(customerPhone, password);
    const driverToken   = await apiLogin(driverPhone, password);

    // ── 3. Customer walks the package flow: where → what → price ──────────────
    // Selectors verified against apps/customer/app/package/{where,what,price}/page.tsx.
    // Only one saved address exists ("E2E Home"), so pickup uses that address
    // card and drop-off uses the "Use my current location" GPS button (single
    // drop-off is the default delivery type — isMultiDrop starts false).
    await customerPage.goto('/package/where');

    // Pickup: the AddressCard under "Pickup address" renders the address label.
    await customerPage
      .locator('section', { hasText: 'Pickup address' })
      .getByText(/E2E Home/i)
      .click();

    // Drop-off: GPS button under the "Drop-off address" section.
    await customerPage
      .locator('section', { hasText: 'Drop-off address' })
      .getByRole('button', { name: /use my current location/i })
      .click();

    // Recipient details (required for singleOk to become true).
    await customerPage.getByLabel('Recipient name').fill('E2E Recipient');
    await customerPage.getByLabel('Recipient phone').fill('08099999999');

    const continueBtn = customerPage.getByRole('button', { name: /continue/i });
    await expect(continueBtn).toBeEnabled({ timeout: 10_000 });
    await continueBtn.click();
    await customerPage.waitForURL('/package/what');

    await customerPage.getByText(/^Small$/i).click();
    await customerPage.getByText(/^Light$/i).click();
    await customerPage.getByRole('button', { name: /continue/i }).click();
    await customerPage.waitForURL('/package/price');

    await expect(customerPage.getByRole('button', { name: /confirm order/i })).toBeEnabled({ timeout: 15_000 });
    await customerPage.getByRole('button', { name: /confirm order/i }).click();
    await customerPage.waitForURL('/package/finding');

    // ── 4. Driver receives and accepts the offer ───────────────────────────────
    await expect(driverPage.getByText(/New delivery/i)).toBeVisible({ timeout: 20_000 });
    await driverPage.getByRole('button', { name: /accept/i }).click();
    await expect(driverPage.getByText(/Accepted — ride to pickup/i)).toBeVisible({ timeout: 10_000 });

    const orderId = await pollUntil(
      async () => {
        const res = await fetch(`${API}/orders?vertical=package&status=driver_assigned`, {
          headers: { Authorization: `Bearer ${customerToken}` },
        });
        const body = await res.json() as { data: { orders: { id: string }[] } };
        return body.data.orders[0]?.id ?? null;
      },
      (id) => id !== null,
      { timeout: 15_000 },
    );

    // ── 5. Advance driver through stages to POD ────────────────────────────────
    await driverPage.getByRole('button', { name: /arrived at pickup/i }).click();
    await driverPage.getByRole('button', { name: /I have the package/i }).click();
    await driverPage.getByRole('button', { name: /arrived at drop-off/i }).click();
    await expect(driverPage.getByText(/Proof of delivery/i)).toBeVisible({ timeout: 10_000 });

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

    await driverPage.locator('input[inputmode="numeric"][maxlength="6"]').fill(deliveryCode!);
    await driverPage.getByRole('button', { name: /confirm delivery/i }).click();

    // ── 6. Assert order delivered + escrow released via API ───────────────────
    await expect.poll(async () => {
      const res = await fetch(`${API}/orders/${orderId}`, {
        headers: { Authorization: `Bearer ${customerToken}` },
      });
      const body = await res.json() as { data: { status: string } };
      return body.data.status;
    }, { timeout: 15_000, intervals: [1000] }).toBe('delivered');

    await expect.poll(async () => {
      const res = await fetch(`${API}/wallet`, {
        headers: { Authorization: `Bearer ${driverToken}` },
      });
      const body = await res.json() as { data: { balanceKobo: number } };
      return body.data.balanceKobo;
    }, { timeout: 15_000, intervals: [1000] }).toBeGreaterThan(0);

    // ── 7. Driver-side earnings tab reflects the same wallet balance ──────────
    await driverPage.getByRole('button', { name: /earnings/i }).click();
    await expect(driverPage.getByText(/WALLET BALANCE/i)).toBeVisible();
    const walletRes = await fetch(`${API}/wallet`, { headers: { Authorization: `Bearer ${driverToken}` } });
    const walletBody = await walletRes.json() as { data: { balanceKobo: number } };
    const naira = `₦${(walletBody.data.balanceKobo / 100).toLocaleString('en-NG', { minimumFractionDigits: 0 })}`;
    await expect(driverPage.getByText(naira, { exact: false })).toBeVisible({ timeout: 10_000 });
  } finally {
    await customerCtx.close().catch(() => {});
    await driverCtx.close().catch(() => {});
  }
});
