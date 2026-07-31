#!/usr/bin/env bash
# scripts/run_app.sh — local dev bootstrap for SpeedPlus API
# Usage: bash scripts/run_app.sh
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
API_DIR="$(dirname "$SCRIPT_DIR")"
cd "$API_DIR"

# ── 1. Env ────────────────────────────────────────────────────────────────────
# Override with Docker Compose local credentials regardless of .env contents
DATABASE_URL="postgres://speedplus:speedplus@localhost:5433/speedplus?sslmode=disable"
REDIS_URL="redis://localhost:6379"
PORT="${PORT:-8000}"
ENVIRONMENT="development"

# Pull remaining secrets from docker-compose.yml defaults
JWT_SECRET="6465766c6f63616c6a77747365637265746b65796d757374626532636861727"
PAYCODE_SECRET="6465766c6f63616c7061796c6f61647365637265746b65796d757374626532"
ENCRYPTION_KEY="devlocalencryptionkey32byteslong"
JWT_ACCESS_TTL_MIN="15"
JWT_REFRESH_TTL_DAYS="30"
OSRM_URL="http://router.project-osrm.org"
ALLOWED_ORIGINS="http://localhost:3000,http://localhost:3001,http://localhost:3002,http://localhost:3003"
PIN_THRESHOLD_KOBO="5000000"

export DATABASE_URL REDIS_URL PORT ENVIRONMENT \
       JWT_SECRET PAYCODE_SECRET ENCRYPTION_KEY \
       JWT_ACCESS_TTL_MIN JWT_REFRESH_TTL_DAYS \
       OSRM_URL ALLOWED_ORIGINS PIN_THRESHOLD_KOBO

# ── 2. Dependency check ───────────────────────────────────────────────────────
for cmd in go psql docker; do
  if ! command -v "$cmd" &>/dev/null; then
    echo "[run_app] ERROR: '$cmd' not found in PATH"
    exit 1
  fi
done

# ── 3. Wait for Postgres ──────────────────────────────────────────────────────
echo "[run_app] Waiting for Postgres on localhost:5433..."
for i in $(seq 1 20); do
  if psql "$DATABASE_URL" -c "SELECT 1" &>/dev/null 2>&1; then
    echo "[run_app] Postgres ready"
    break
  fi
  sleep 1
  if [[ $i -eq 20 ]]; then
    echo "[run_app] ERROR: Postgres not reachable — is 'docker compose up' running?"
    exit 1
  fi
done

# ── 4. Wait for Redis ─────────────────────────────────────────────────────────
echo "[run_app] Waiting for Redis..."
for i in $(seq 1 10); do
  if docker exec speedplus-redis-1 redis-cli ping &>/dev/null 2>&1; then
    echo "[run_app] Redis ready"
    break
  fi
  sleep 1
  if [[ $i -eq 10 ]]; then
    echo "[run_app] ERROR: Redis not reachable — is 'docker compose up' running?"
    exit 1
  fi
done

# ── 5. Migrations ─────────────────────────────────────────────────────────────
echo "[run_app] Running migrations..."
go run ./cmd/server/migrate.go 2>&1 | tail -10

# ── 6. Seed ───────────────────────────────────────────────────────────────────
echo "[run_app] Seeding test data..."
psql "$DATABASE_URL" <<'SQL'
DO $$
DECLARE
  customer_id UUID := '11111111-1111-1111-1111-111111111111';
  driver_id   UUID := '22222222-2222-2222-2222-222222222222';
  merchant_id UUID := '33333333-3333-3333-3333-333333333333';
BEGIN
  -- Seed users with a known argon2id hash.
  -- The hash below encodes "TestPass1!" using the same params as auth.go
  -- (m=65536,t=1,p=4). For dev only — never use in production.
  INSERT INTO users (id, role, first_name, last_name, phone, password_hash, referral_code, is_verified, is_active)
  VALUES
    (customer_id, 'customer', 'Test', 'Customer', '+2348000000001',
     '$argon2id$v=19$m=65536,t=1,p=4$c2FsdHNhbHRzYWx0c2FsdA$aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa', 'CUST001', true, true),
    (driver_id,   'driver',   'Test', 'Driver',   '+2348000000002',
     '$argon2id$v=19$m=65536,t=1,p=4$c2FsdHNhbHRzYWx0c2FsdA$aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa', 'DRVR001', true, true),
    (merchant_id, 'merchant', 'Test', 'Merchant', '+2348000000003',
     '$argon2id$v=19$m=65536,t=1,p=4$c2FsdHNhbHRzYWx0c2FsdA$aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa', 'MRCH001', true, true)
  ON CONFLICT (id) DO NOTHING;

  -- Wallet ledger accounts
  INSERT INTO ledger_accounts (id, owner_id, type, currency)
  VALUES
    ('aaaa0001-0000-0000-0000-000000000001', customer_id, 'wallet', 'NGN'),
    ('aaaa0002-0000-0000-0000-000000000002', driver_id,   'wallet', 'NGN'),
    ('aaaa0003-0000-0000-0000-000000000003', merchant_id, 'wallet', 'NGN')
  ON CONFLICT (id) DO NOTHING;

  -- Balances in kobo: customer=₦50,000  driver=₦10,000  merchant=₦0
  INSERT INTO wallet_balances (account_id, balance_kobo)
  VALUES
    ('aaaa0001-0000-0000-0000-000000000001', 5000000),
    ('aaaa0002-0000-0000-0000-000000000002', 1000000),
    ('aaaa0003-0000-0000-0000-000000000003', 0)
  ON CONFLICT (account_id) DO NOTHING;

  -- Platform accounts
  INSERT INTO ledger_accounts (id, owner_id, type, currency)
  VALUES
    ('bbbb0001-0000-0000-0000-000000000001', NULL, 'escrow',            'NGN'),
    ('bbbb0002-0000-0000-0000-000000000002', NULL, 'revenue',           'NGN'),
    ('bbbb0003-0000-0000-0000-000000000003', NULL, 'provider_clearing', 'NGN'),
    ('bbbb0004-0000-0000-0000-000000000004', NULL, 'earnings',          'NGN')
  ON CONFLICT (id) DO NOTHING;

  INSERT INTO wallet_balances (account_id, balance_kobo)
  VALUES
    ('bbbb0001-0000-0000-0000-000000000001', 0),
    ('bbbb0002-0000-0000-0000-000000000002', 0),
    ('bbbb0003-0000-0000-0000-000000000003', 0),
    ('bbbb0004-0000-0000-0000-000000000004', 0)
  ON CONFLICT (account_id) DO NOTHING;
END $$;
SQL
echo "[run_app] Seed complete"

# ── 7. Start server ───────────────────────────────────────────────────────────
echo "[run_app] Starting API server on :$PORT (go run — uses latest source)"
exec go run ./cmd/server/main.go
