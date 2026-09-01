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
    merchantPhone: env['MERCHANT_PHONE'] ?? process.env['E2E_MERCHANT_PHONE'] ?? '+2349000000003',
    password:      env['SEED_PASSWORD']  ?? process.env['E2E_SEED_PASSWORD']  ?? 'Test1234!',
    merchantId:    env['MERCHANT_ID'],
    productId:     env['PRODUCT_ID'],
    addressId:     env['ADDRESS_ID'],
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

// Merchant portal: an order arrives → merchant confirms it → preps it →
// marks it ready for pickup, with each UI click verified against the real
// order status in Postgres via the API. The order itself is created directly
// via the API (POST /quotes → POST /orders) rather than through the customer
// UI — order-flow.spec.ts and delivery-completion.spec.ts already cover the
// customer-side UI journey into an order; this spec's purpose is the
// merchant-side state machine, so seeding the order via API keeps it focused
// and fast while still exercising the real handler → service → repo →
// Postgres path for order creation.
test('merchant confirms, prepares, and readies an incoming order via the UI', async ({ page }) => {
  const { customerPhone, merchantPhone, password, merchantId, productId, addressId } = loadFixtures();
  if (!merchantId || !productId || !addressId) {
    test.skip(true, 'seed fixture file missing MERCHANT_ID/PRODUCT_ID/ADDRESS_ID — run the seed script first');
  }

  const customerToken = await apiLogin(customerPhone, password);

  // ── 1. Create a food order directly via the API (customer side) ──────────
  const quoteRes = await fetch(`${API}/quotes`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${customerToken}` },
    body: JSON.stringify({
      merchantId, vertical: 'food', subtotalKobo: 2500_00,
      originLat: 6.4531, originLng: 3.3958, destLat: 6.4550, destLng: 3.3980,
    }),
  });
  const quoteBody = await quoteRes.json() as { data: { id: string } };
  expect(quoteRes.ok, `quote request failed: ${JSON.stringify(quoteBody)}`).toBe(true);

  const orderRes = await fetch(`${API}/orders`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${customerToken}`,
      'Idempotency-Key': `merchant-orders-e2e-${Date.now()}`,
    },
    body: JSON.stringify({
      merchantId, quoteId: quoteBody.data.id, vertical: 'food',
      items: [{ productId, quantity: 1 }],
      deliveryAddressId: addressId, paymentMethod: 'wallet',
    }),
  });
  const orderBody = await orderRes.json() as { data: { id: string; status: string } };
  expect(orderRes.ok, `order create failed: ${JSON.stringify(orderBody)}`).toBe(true);
  const orderId = orderBody.data.id;
  expect(orderBody.data.status).toBe('pending');

  // ── 2. Merchant logs in and sees the order on /orders ─────────────────────
  await page.goto('/login');
  await page.getByLabel('Phone number').fill(merchantPhone);
  await page.locator('input[type="password"]').fill(password);
  await page.getByRole('button', { name: /sign in to partner portal/i }).click();
  await page.waitForURL('/');

  await page.goto('/orders');
  await expect(page.getByText(`#${orderId.slice(0, 8)}`)).toBeVisible({ timeout: 15_000 });

  const merchantToken = await apiLogin(merchantPhone, password);

  // ── 3. "Confirm & prepare" (pending → confirmed) ──────────────────────────
  await page.getByRole('button', { name: /confirm & prepare/i }).click();
  await expect.poll(async () => {
    const res = await fetch(`${API}/orders/${orderId}`, { headers: { Authorization: `Bearer ${customerToken}` } });
    const body = await res.json() as { data: { status: string } };
    return body.data.status;
  }, { timeout: 15_000, intervals: [1000] }).toBe('confirmed');

  // ── 4. "Start preparing" (confirmed → preparing) ──────────────────────────
  await page.getByRole('button', { name: /start preparing/i }).click();
  await expect.poll(async () => {
    const res = await fetch(`${API}/orders/${orderId}`, { headers: { Authorization: `Bearer ${customerToken}` } });
    const body = await res.json() as { data: { status: string } };
    return body.data.status;
  }, { timeout: 15_000, intervals: [1000] }).toBe('preparing');

  // ── 5. "Mark ready" (preparing → ready_for_pickup) ────────────────────────
  await page.getByRole('button', { name: /mark ready/i }).click();
  await expect.poll(async () => {
    const res = await fetch(`${API}/orders/${orderId}`, { headers: { Authorization: `Bearer ${customerToken}` } });
    const body = await res.json() as { data: { status: string } };
    return body.data.status;
  }, { timeout: 15_000, intervals: [1000] }).toBe('ready_for_pickup');

  // The card should now show "Rider on the way" with no further action button.
  await expect(page.getByText(/rider on the way/i)).toBeVisible({ timeout: 10_000 });

  // ── 6. Earnings page reflects the merchant's real wallet balance ─────────
  await page.goto('/earnings');
  const walletRes = await fetch(`${API}/merchant/wallet`, { headers: { Authorization: `Bearer ${merchantToken}` } });
  const walletBody = await walletRes.json() as { data: { balanceKobo: number } };
  const naira = `₦${(walletBody.data.balanceKobo / 100).toLocaleString('en-NG', { minimumFractionDigits: 0 })}`;
  await expect(page.getByText(naira, { exact: false })).toBeVisible({ timeout: 15_000 });
});
