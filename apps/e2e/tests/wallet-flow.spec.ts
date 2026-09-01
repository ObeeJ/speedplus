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

async function getBalanceKobo(token: string): Promise<number> {
  const res = await fetch(`${API}/wallet`, { headers: { Authorization: `Bearer ${token}` } });
  const body = await res.json() as { data: { balanceKobo: number } };
  return body.data.balanceKobo;
}

// Wallet balance display + real-time balance read on /wallet.
//
// NOTE on /wallet/fund: it POSTs to Paystack/Bridge and does
// `window.location.href = result.authorizationUrl`, redirecting off-app to a
// hosted payment page E2E has no way to complete headlessly (see report for
// details) — so funding is not covered here. Instead this test drives the
// one wallet-mutating action that completes entirely in-app: customer→driver
// wallet transfer (/wallet/transfer, walletApi.transfer), which still goes
// through the real API → ledger service → Postgres wallet_balances update.
test('wallet page shows live balance, and a wallet-to-wallet transfer moves real money', async ({ page }) => {
  const { customerPhone, driverPhone, password } = loadFixtures();

  const customerToken = await apiLogin(customerPhone, password);
  const driverToken = await apiLogin(driverPhone, password);

  const customerBalanceBefore = await getBalanceKobo(customerToken);
  const driverBalanceBefore = await getBalanceKobo(driverToken);

  // ── 1. Wallet page shows the real balance from the API ──────────────────
  await page.goto('/login');
  await page.getByLabel('Phone number').fill(customerPhone);
  await page.locator('input[type="password"]').fill(password);
  await page.getByRole('button', { name: /sign in/i }).click();
  await page.waitForURL('/');

  await page.goto('/wallet');
  const naira = (kobo: number) => `₦${(kobo / 100).toLocaleString('en-NG', { minimumFractionDigits: 0 })}`;
  await expect(page.getByText(naira(customerBalanceBefore), { exact: false })).toBeVisible({ timeout: 15_000 });

  // ── 2. Transfer ₦500 to the driver via the UI ────────────────────────────
  const transferKobo = 50_000; // ₦500 — above the 10,000-kobo (₦100) minimum
  await page.getByRole('button', { name: /send/i }).click();
  await page.waitForURL('/wallet/transfer');

  await page.getByLabel(/Recipient phone number/i).fill(driverPhone);
  await page.getByText('₦500', { exact: true }).click();
  await page.getByLabel(/Wallet PIN/i).fill('1234');
  await page.getByRole('button', { name: /send ₦500/i }).click();

  // Success screen confirms the amount sent.
  await expect(page.getByText(`${naira(transferKobo)} sent`)).toBeVisible({ timeout: 15_000 });

  // ── 3. Verify both wallet balances actually moved server-side ───────────
  await expect.poll(async () => getBalanceKobo(customerToken), {
    timeout: 15_000, intervals: [1000],
  }).toBe(customerBalanceBefore - transferKobo);

  await expect.poll(async () => getBalanceKobo(driverToken), {
    timeout: 15_000, intervals: [1000],
  }).toBe(driverBalanceBefore + transferKobo);
});
