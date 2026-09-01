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
    adminPhone: process.env['E2E_ADMIN_PHONE'] ?? '+2349000000004',
    password:   env['SEED_PASSWORD'] ?? process.env['E2E_SEED_PASSWORD'] ?? 'Test1234!',
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

// The seed script (apps/api/scripts/seed/main.go) provisions one merchant
// with status='pending' ("E2E Pending Merchant", phone +2349000000008) and
// one driver_profile with status='pending' ("Pending Driver",
// phone +2349000000009) specifically so the admin approval UI has real
// fixtures to act on — these did not exist in the seed set before this pass.
test('admin approves a pending merchant', async ({ browser }) => {
  const { adminPhone, password } = loadFixtures();
  const adminCtx = await browser.newContext({ baseURL: 'http://localhost:3003' });
  const adminPage = await adminCtx.newPage();

  try {
    await adminPage.goto('/login');
    await adminPage.getByLabel('Phone number').fill(adminPhone);
    await adminPage.locator('input[type="password"]').fill(password);
    await adminPage.getByRole('button', { name: /sign in/i }).click();
    await adminPage.waitForURL('/kyc');

    // Selectors verified against apps/admin/app/merchants/page.tsx.
    // Default filter is 'pending' so the seeded row should already be visible.
    await adminPage.goto('/merchants');
    const row = adminPage.locator('div', { hasText: 'E2E Pending Merchant' }).filter({ has: adminPage.getByRole('button', { name: /approve/i }) }).first();
    await expect(row).toBeVisible({ timeout: 10_000 });
    await row.getByRole('button', { name: /approve/i }).click();

    // UI patches the cached row's status badge optimistically on success.
    await expect(adminPage.locator('div', { hasText: 'E2E Pending Merchant' }).getByText(/^active$/i)).toBeVisible({ timeout: 10_000 });

    // Verify server-side via the same admin API the page calls.
    const adminToken = await apiLogin(adminPhone, password);
    const res = await fetch(`${API}/admin/merchants?status=active`, {
      headers: { Authorization: `Bearer ${adminToken}` },
    });
    const body = await res.json() as { data: { merchants: { businessName: string; status: string }[] } };
    const merchant = body.data.merchants.find((m) => m.businessName === 'E2E Pending Merchant');
    expect(merchant?.status).toBe('active');
  } finally {
    await adminCtx.close().catch(() => {});
  }
});

test('admin approves a pending driver', async ({ browser }) => {
  const { adminPhone, password } = loadFixtures();
  const adminCtx = await browser.newContext({ baseURL: 'http://localhost:3003' });
  const adminPage = await adminCtx.newPage();

  try {
    await adminPage.goto('/login');
    await adminPage.getByLabel('Phone number').fill(adminPhone);
    await adminPage.locator('input[type="password"]').fill(password);
    await adminPage.getByRole('button', { name: /sign in/i }).click();
    await adminPage.waitForURL('/kyc');

    // Selectors verified against apps/admin/app/drivers/page.tsx.
    // The seeded pending driver has vehicle plate LAG-E2E-99 — rows don't
    // render a name, only vehicleType/vehiclePlate + rating, so match on plate.
    await adminPage.goto('/drivers');
    const row = adminPage.locator('div', { hasText: 'LAG-E2E-99' }).filter({ has: adminPage.getByRole('button', { name: /approve/i }) }).first();
    await expect(row).toBeVisible({ timeout: 10_000 });
    await row.getByRole('button', { name: /approve/i }).click();

    await expect(adminPage.locator('div', { hasText: 'LAG-E2E-99' }).getByText(/^approved$/i)).toBeVisible({ timeout: 10_000 });

    const adminToken = await apiLogin(adminPhone, password);
    const res = await fetch(`${API}/admin/drivers?status=approved`, {
      headers: { Authorization: `Bearer ${adminToken}` },
    });
    const body = await res.json() as { data: { drivers: { vehiclePlate: string; status: string }[] } };
    const driver = body.data.drivers.find((d) => d.vehiclePlate === 'LAG-E2E-99');
    expect(driver?.status).toBe('approved');
  } finally {
    await adminCtx.close().catch(() => {});
  }
});
