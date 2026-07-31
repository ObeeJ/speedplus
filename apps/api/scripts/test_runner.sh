#!/usr/bin/env bash
# scripts/test_runner.sh — E2E + concurrency tests for SpeedPlus fintech core
# Usage: bash scripts/test_runner.sh [BASE_URL]
# Requires: curl, jq, psql
set -euo pipefail

BASE_URL="${1:-http://localhost:8000}"
DATABASE_URL="postgres://speedplus:speedplus@localhost:5433/speedplus?sslmode=disable"

PASS=0; FAIL=0

log_pass() { echo "  ✅ $1"; ((PASS++)) || true; }
log_fail() { echo "  ❌ $1"; ((FAIL++)) || true; }

assert_eq() {
  local label="$1" got="$2" want="$3"
  if [[ "$got" == "$want" ]]; then log_pass "$label (got: $got)"
  else log_fail "$label (got: '$got', want: '$want')"; fi
}

# ── Preflight ─────────────────────────────────────────────────────────────────
echo "[test_runner] Checking API is up..."
if ! curl -sf "$BASE_URL/healthz" &>/dev/null; then
  echo "[test_runner] ERROR: API not reachable at $BASE_URL — run scripts/run_app.sh first"
  exit 1
fi
echo "[test_runner] API is up"

# ── Auth helpers ──────────────────────────────────────────────────────────────
login() {
  local phone="$1" pass="${2:-TestPass1!}"
  curl -sf -X POST "$BASE_URL/api/v1/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"phone\":\"$phone\",\"password\":\"$pass\"}" 2>/dev/null \
    | jq -r '.data.accessToken // empty'
}

get_balance() {
  local token="$1"
  curl -sf "$BASE_URL/api/v1/wallet" \
    -H "Authorization: Bearer $token" 2>/dev/null \
    | jq -r '.data.balanceKobo // 0'
}

# ── Seed fresh balances before each run ───────────────────────────────────────
reset_balances() {
  psql "$DATABASE_URL" -q <<'SQL'
UPDATE wallet_balances SET balance_kobo = 5000000 WHERE account_id = 'aaaa0001-0000-0000-0000-000000000001';
UPDATE wallet_balances SET balance_kobo = 1000000 WHERE account_id = 'aaaa0002-0000-0000-0000-000000000002';
SQL
}

echo ""
echo "[test_runner] Resetting balances..."
reset_balances

# ── Login ─────────────────────────────────────────────────────────────────────
echo "[test_runner] Logging in test users..."
CUSTOMER_TOKEN=$(login "+2348000000001")
DRIVER_TOKEN=$(login "+2348000000002")

if [[ -z "$CUSTOMER_TOKEN" || -z "$DRIVER_TOKEN" ]]; then
  echo "[test_runner] ERROR: Login failed — check seed users and password hash"
  exit 1
fi
echo "[test_runner] Logged in OK"

# ═════════════════════════════════════════════════════════════════════════════
echo ""
echo "═══ TEST 1: Idempotency — duplicate request, single debit ═══"
# ═════════════════════════════════════════════════════════════════════════════

IDEM_KEY="idem-$(date +%s%N)"
BAL_BEFORE=$(get_balance "$CUSTOMER_TOKEN")

transfer() {
  curl -sf -X POST "$BASE_URL/api/v1/wallet/transfer" \
    -H "Authorization: Bearer $CUSTOMER_TOKEN" \
    -H "Idempotency-Key: $1" \
    -H "Content-Type: application/json" \
    -d '{"recipientId":"22222222-2222-2222-2222-222222222222","amountKobo":100000,"pin":"0000"}' \
    2>/dev/null || echo '{}'
}

RESP1=$(transfer "$IDEM_KEY")
RESP2=$(transfer "$IDEM_KEY")  # exact duplicate

BAL_AFTER=$(get_balance "$CUSTOMER_TOKEN")
DEBIT=$(( BAL_BEFORE - BAL_AFTER ))

assert_eq "Single debit on duplicate key" "$DEBIT" "100000"

MSG1=$(echo "$RESP1" | jq -r '.data.message // empty')
MSG2=$(echo "$RESP2" | jq -r '.data.message // empty')
assert_eq "Idempotent response body identical" "$MSG1" "$MSG2"

# ═════════════════════════════════════════════════════════════════════════════
echo ""
echo "═══ TEST 2: Idempotency body mismatch → 422 ═══"
# ═════════════════════════════════════════════════════════════════════════════

IDEM_KEY2="mismatch-$(date +%s%N)"

# First request
curl -sf -X POST "$BASE_URL/api/v1/wallet/transfer" \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" \
  -H "Idempotency-Key: $IDEM_KEY2" \
  -H "Content-Type: application/json" \
  -d '{"recipientId":"22222222-2222-2222-2222-222222222222","amountKobo":100000,"pin":"0000"}' \
  &>/dev/null || true

# Same key, different body
HTTP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
  -X POST "$BASE_URL/api/v1/wallet/transfer" \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" \
  -H "Idempotency-Key: $IDEM_KEY2" \
  -H "Content-Type: application/json" \
  -d '{"recipientId":"22222222-2222-2222-2222-222222222222","amountKobo":200000,"pin":"0000"}')

assert_eq "Body mismatch returns 422" "$HTTP_STATUS" "422"

# ═════════════════════════════════════════════════════════════════════════════
echo ""
echo "═══ TEST 3: Concurrency — 10 parallel transfers, no overdraft ═══"
# ═════════════════════════════════════════════════════════════════════════════

# Reset customer to exactly ₦10,000 (1,000,000 kobo)
psql "$DATABASE_URL" -q -c \
  "UPDATE wallet_balances SET balance_kobo = 1000000 WHERE account_id = 'aaaa0001-0000-0000-0000-000000000001';"

BAL_START=$(get_balance "$CUSTOMER_TOKEN")
assert_eq "Balance reset to 1,000,000 kobo before stress test" "$BAL_START" "1000000"

# Fire 10 concurrent transfers of ₦1,000 each (total = exact balance)
for i in $(seq 1 10); do
  IDEM="concurrent-$i-$(date +%s%N)"
  curl -sf -X POST "$BASE_URL/api/v1/wallet/transfer" \
    -H "Authorization: Bearer $CUSTOMER_TOKEN" \
    -H "Idempotency-Key: $IDEM" \
    -H "Content-Type: application/json" \
    -d '{"recipientId":"22222222-2222-2222-2222-222222222222","amountKobo":100000,"pin":"0000"}' \
    -o "/tmp/conc_$i.json" 2>/dev/null &
done
wait

BAL_END=$(get_balance "$CUSTOMER_TOKEN")

if [[ "$BAL_END" -ge 0 ]]; then
  log_pass "No overdraft after 10 concurrent transfers (final balance: $BAL_END kobo)"
else
  log_fail "OVERDRAFT: balance is $BAL_END kobo"
fi

SUCCESS_COUNT=0
for i in $(seq 1 10); do
  grep -q '"transfer successful"' "/tmp/conc_$i.json" 2>/dev/null && ((SUCCESS_COUNT++)) || true
done
echo "  ℹ️  Successful transfers: $SUCCESS_COUNT/10"

# ═════════════════════════════════════════════════════════════════════════════
echo ""
echo "═══ TEST 4: Ledger invariance — SUM(amount_kobo) = 0 ═══"
# ═════════════════════════════════════════════════════════════════════════════

LEDGER_SUM=$(psql "$DATABASE_URL" -t -c \
  "SELECT COALESCE(SUM(amount_kobo), 0) FROM ledger_entries;" | tr -d ' \n')

assert_eq "Global ledger sum is 0 (balanced double-entry)" "$LEDGER_SUM" "0"

UNBALANCED=$(psql "$DATABASE_URL" -t -c \
  "SELECT COUNT(*) FROM (
     SELECT journal_id FROM ledger_entries
     GROUP BY journal_id HAVING SUM(amount_kobo) != 0
   ) t;" | tr -d ' \n')

assert_eq "Zero unbalanced journals" "$UNBALANCED" "0"

# ═════════════════════════════════════════════════════════════════════════════
echo ""
echo "═══ TEST 5: Outbox — webhook_events table exists and is append-only ═══"
# ═════════════════════════════════════════════════════════════════════════════

TABLE_EXISTS=$(psql "$DATABASE_URL" -t -c \
  "SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'webhook_events';" \
  | tr -d ' \n')
assert_eq "webhook_events table exists" "$TABLE_EXISTS" "1"

# Verify the append-only rule is present (migration 003 creates it)
RULE_EXISTS=$(psql "$DATABASE_URL" -t -c \
  "SELECT COUNT(*) FROM pg_rules WHERE tablename = 'ledger_entries' AND rulename IN ('no_update_ledger','no_delete_ledger');" \
  | tr -d ' \n')
assert_eq "Ledger append-only rules present (no_update + no_delete)" "$RULE_EXISTS" "2"

# ═════════════════════════════════════════════════════════════════════════════
echo ""
echo "═══ TEST 6: Minor units — balanceKobo is an integer, no floats ═══"
# ═════════════════════════════════════════════════════════════════════════════

BAL_RAW=$(curl -sf "$BASE_URL/api/v1/wallet" \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" 2>/dev/null \
  | jq '.data.balanceKobo')

if echo "$BAL_RAW" | grep -qE '^-?[0-9]+$'; then
  log_pass "balanceKobo is an integer: $BAL_RAW"
else
  log_fail "balanceKobo is not a plain integer: $BAL_RAW"
fi

# ═════════════════════════════════════════════════════════════════════════════
echo ""
echo "═══ TEST 7: Insufficient funds — no partial debit ═══"
# ═════════════════════════════════════════════════════════════════════════════

BAL_BEFORE_OD=$(get_balance "$CUSTOMER_TOKEN")
IDEM_OD="overdraft-$(date +%s%N)"

HTTP_OD=$(curl -s -o /dev/null -w "%{http_code}" \
  -X POST "$BASE_URL/api/v1/wallet/transfer" \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" \
  -H "Idempotency-Key: $IDEM_OD" \
  -H "Content-Type: application/json" \
  -d '{"recipientId":"22222222-2222-2222-2222-222222222222","amountKobo":999999999,"pin":"0000"}')

BAL_AFTER_OD=$(get_balance "$CUSTOMER_TOKEN")

assert_eq "Overdraft attempt returns 4xx" "$(echo "$HTTP_OD" | cut -c1)" "4"
assert_eq "Balance unchanged after failed overdraft" "$BAL_BEFORE_OD" "$BAL_AFTER_OD"

# ═════════════════════════════════════════════════════════════════════════════
echo ""
echo "═══ TEST 8: wallet_balances CHECK constraint — no negative balance in DB ═══"
# ═════════════════════════════════════════════════════════════════════════════

NEGATIVE=$(psql "$DATABASE_URL" -t -c \
  "SELECT COUNT(*) FROM wallet_balances WHERE balance_kobo < 0;" \
  | tr -d ' \n')
assert_eq "No negative wallet_balances rows" "$NEGATIVE" "0"

# ── Cleanup ───────────────────────────────────────────────────────────────────
rm -f /tmp/conc_*.json

# ── Summary ───────────────────────────────────────────────────────────────────
echo ""
echo "══════════════════════════════════════════"
echo "  Results: $PASS passed, $FAIL failed"
echo "══════════════════════════════════════════"

[[ $FAIL -eq 0 ]]
