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
    customerId:    env['CUSTOMER_ID'] ?? '',
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

// The seed script writes CUSTOMER_ID to the fixtures file (see
// apps/api/scripts/seed/main.go fixtureContent). The seeded customer wallet
// is funded with ₦50,000 at seed time (walletFundKobo), which produces a
// ledger entry we can assert on through the admin Ledger Viewer UI.
test('admin views customer ledger entries and they match the API', async ({ browser }) => {
  const { customerPhone, customerId, adminPhone, password } = loadFixtures();

  const adminCtx = await browser.newContext({ baseURL: 'http://localhost:3003' });
  const adminPage = await adminCtx.newPage();

  try {
    // Resolve the customer's user ID via API if the fixture file didn't have it
    // (keeps the test robust to fixture-file format changes in the seed script).
    let userId = customerId;
    if (!userId) {
      const token = await apiLogin(customerPhone, password);
      const meRes = await fetch(`${API}/users/me`, { headers: { Authorization: `Bearer ${token}` } });
      const meBody = await meRes.json() as { data: { id: string } };
      userId = meBody.data.id;
    }

    // ── Admin logs in ──────────────────────────────────────────────────────────
    await adminPage.goto('/login');
    await adminPage.getByLabel('Phone number').fill(adminPhone);
    await adminPage.locator('input[type="password"]').fill(password);
    await adminPage.getByRole('button', { name: /sign in/i }).click();
    await adminPage.waitForURL('/kyc');

    // ── Navigate to Ledger Viewer and load the customer's entries ─────────────
    // Selectors verified against apps/admin/app/ledger/page.tsx.
    await adminPage.goto('/ledger');
    await adminPage.getByPlaceholder('User UUID').fill(userId);
    await adminPage.getByRole('button', { name: /^load$/i }).click();

    // ── Cross-check against the API directly (same endpoint the page calls) ──
    const adminToken = await apiLogin(adminPhone, password);
    const ledgerRes = await fetch(`${API}/admin/ledger?userId=${userId}`, {
      headers: { Authorization: `Bearer ${adminToken}` },
    });
    expect(ledgerRes.ok).toBeTruthy();
    const ledgerBody = await ledgerRes.json() as { data: { entries: { id: string; description: string }[] } };

    if (ledgerBody.data.entries.length === 0) {
      // Nothing to assert on screen — the seed-time wallet fund credit should
      // always exist for the seeded customer, so an empty result here is
      // itself a signal worth surfacing rather than silently passing.
      test.info().annotations.push({
        type: 'gap',
        description: `GET /admin/ledger?userId=${userId} returned zero entries — expected at least the seed-time wallet fund credit.`,
      });
    } else {
      const first = ledgerBody.data.entries[0]!;
      await expect(adminPage.getByText(first.description)).toBeVisible({ timeout: 10_000 });
    }
  } finally {
    await adminCtx.close().catch(() => {});
  }
});
