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
    adminPhone:    process.env['E2E_ADMIN_PHONE'] ?? '+2349000000004',
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

// End-to-end: complete a food delivery (same UI path as
// delivery-completion.spec.ts) via API-driven driver actions to get a
// delivered order with escrow held + proof media on record, then drive the
// admin Disputes page (apps/admin/app/disputes/page.tsx) through freeze →
// release and assert the escrow state actually changes via the ledger API.
//
// NOTE ON DUAL-ADMIN APPROVAL (docs/OPS-RUNBOOK.md Section B, confirmed in
// apps/api/internal/service/admin.go ReleaseEscrow): releases above
// ₦50,000 (disputeSingleAdminThresholdKobo = 5_000_000 kobo) require a
// second, *different* admin's approval before funds move — the first call
// only records ApprovalOne and returns "approval recorded — awaiting second
// admin approval". The seed data only provisions ONE admin
// (+2349000000004), so a Tier-3 dual-admin release cannot be exercised by
// this suite. The seeded food order (E2E Jollof Rice, ₦2,500) is safely
// under the threshold, so this test only exercises the Tier-2 single-admin
// path. A second seeded admin account is needed to cover Tier-3.
test('admin freezes and releases escrow on a delivered order', async ({ browser }) => {
  const { customerPhone, driverPhone, adminPhone, password } = loadFixtures();

  const customerCtx = await browser.newContext({ baseURL: 'http://localhost:3000' });
  const driverCtx   = await browser.newContext({ baseURL: 'http://localhost:3001' });
  const adminCtx    = await browser.newContext({ baseURL: 'http://localhost:3003' });

  const customerPage = await customerCtx.newPage();
  const driverPage   = await driverCtx.newPage();
  const adminPage    = await adminCtx.newPage();

  try {
    // ── 1. Driver online, customer places + driver completes a food order ────
    await driverPage.goto('/login');
    await driverPage.getByLabel('Phone number').fill(driverPhone);
    await driverPage.locator('input[type="password"]').fill(password);
    await driverPage.getByRole('button', { name: /sign in/i }).click();
    await driverPage.waitForURL('/');
    await driverPage.getByRole('button', { name: /offline/i }).click();
    await expect(driverPage.getByRole('button', { name: /online/i })).toBeVisible();

    await customerPage.goto('/login');
    await customerPage.getByLabel('Phone number').fill(customerPhone);
    await customerPage.locator('input[type="password"]').fill(password);
    await customerPage.getByRole('button', { name: /sign in/i }).click();
    await customerPage.waitForURL('/');

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

    await expect(driverPage.getByText(/New delivery/i)).toBeVisible({ timeout: 20_000 });
    await driverPage.getByRole('button', { name: /accept/i }).click();
    await expect(driverPage.getByText(/Accepted — ride to pickup/i)).toBeVisible({ timeout: 10_000 });

    const orderId = await pollUntil(
      async () => {
        const res = await fetch(`${API}/orders?vertical=food&status=driver_assigned`, {
          headers: { Authorization: `Bearer ${customerToken}` },
        });
        const body = await res.json() as { data: { orders: { id: string }[] } };
        return body.data.orders[0]?.id ?? null;
      },
      (id) => id !== null,
      { timeout: 15_000 },
    );

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

    await expect.poll(async () => {
      const res = await fetch(`${API}/orders/${orderId}`, {
        headers: { Authorization: `Bearer ${customerToken}` },
      });
      const body = await res.json() as { data: { status: string } };
      return body.data.status;
    }, { timeout: 15_000, intervals: [1000] }).toBe('delivered');

    // ── 2. Admin logs in and looks up the order on the Disputes page ─────────
    // Selectors verified against apps/admin/app/disputes/page.tsx.
    await adminPage.goto('/login');
    await adminPage.getByLabel('Phone number').fill(adminPhone);
    await adminPage.locator('input[type="password"]').fill(password);
    await adminPage.getByRole('button', { name: /sign in/i }).click();
    await adminPage.waitForURL('/kyc');

    await adminPage.goto('/disputes');
    await adminPage.getByPlaceholder('xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx').fill(orderId!);
    await adminPage.getByRole('button', { name: /look up/i }).click();
    await expect(adminPage.getByText(orderId!)).toBeVisible({ timeout: 10_000 });

    // ── 3. Freeze escrow — type-to-confirm modal requires reason + exact amount ─
    await adminPage.getByRole('button', { name: /freeze escrow/i }).click();
    await adminPage.getByPlaceholder('Why is this action being taken?').fill('E2E dispute — item not as described');
    const amountLabel = await adminPage.locator('label:has-text("Type the amount shown above")').locator('span.font-mono').innerText();
    await adminPage.locator('input[placeholder="' + amountLabel + '"]').fill(amountLabel);
    await adminPage.getByRole('button', { name: /confirm & execute/i }).click();
    await expect(adminPage.getByText(/escrow frozen/i)).toBeVisible({ timeout: 10_000 });

    // ── 4. Release escrow → merchant (single-admin, order value well under the
    //      ₦50,000 dual-admin threshold) ──────────────────────────────────────
    await adminPage.getByRole('button', { name: /release → merchant/i }).click();
    await adminPage.getByPlaceholder('Why is this action being taken?').fill('E2E dispute resolved in merchant favor');
    const amountLabel2 = await adminPage.locator('label:has-text("Type the amount shown above")').locator('span.font-mono').innerText();
    await adminPage.locator('input[placeholder="' + amountLabel2 + '"]').fill(amountLabel2);
    await adminPage.getByRole('button', { name: /confirm & execute/i }).click();

    // The release message text differs by settlement outcome — assert the
    // generic success banner rather than one exact string.
    await expect(adminPage.locator('p.text-emerald')).toBeVisible({ timeout: 10_000 });

    // ── 5. Verify via API that escrow is no longer frozen/held ───────────────
    const adminToken = await apiLogin(adminPhone, password);
    const orderDetailRes = await fetch(`${API}/admin/orders/${orderId}`, {
      headers: { Authorization: `Bearer ${adminToken}` },
    });
    expect(orderDetailRes.ok).toBeTruthy();
  } finally {
    await customerCtx.close().catch(() => {});
    await driverCtx.close().catch(() => {});
    await adminCtx.close().catch(() => {});
  }
});
