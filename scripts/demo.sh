#!/usr/bin/env bash
set -euo pipefail

BASE="http://localhost:8080/api/v1"

reg()  { curl -s -X POST "$BASE/auth/register" -H "Content-Type: application/json" -d "$1"; }
login(){ curl -s -X POST "$BASE/auth/login"    -H "Content-Type: application/json" -d "$1" \
        | python -c 'import sys,json; print(json.load(sys.stdin)["access_token"])'; }

# her koşuda benzersiz idempotency key
STAMP="$(date +%s)"
IDEM_CREDIT="credit-$STAMP"
IDEM_TRANSFER="transfer-$STAMP"

echo "# register alice"
reg '{"username":"alice","email":"a@a.com","password":"pass"}' || true

echo "# login alice"
TOKEN_ALICE=$(login '{"email":"a@a.com","password":"pass"}')

echo "# me"
curl -s "$BASE/me" -H "Authorization: Bearer $TOKEN_ALICE"; echo

echo "# register bob"
reg '{"username":"bob","email":"b@b.com","password":"pass"}' || true
TOKEN_BOB=$(login '{"email":"b@b.com","password":"pass"}')

BOB_ID=$(docker compose exec -T db psql -U postgres -d insider -tAc "SELECT id FROM users WHERE email='b@b.com';")
echo "BOB_ID=$BOB_ID"

echo "# credit 1000 to alice (Idempotency-Key: $IDEM_CREDIT)"
curl -s -X POST "$BASE/transactions/credit" \
  -H "Authorization: Bearer $TOKEN_ALICE" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: $IDEM_CREDIT" \
  -d '{"amount":1000}'; echo

echo "# transfer 500 alice -> bob (Idempotency-Key: $IDEM_TRANSFER)"
curl -s -X POST "$BASE/transactions/transfer" \
  -H "Authorization: Bearer $TOKEN_ALICE" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: $IDEM_TRANSFER" \
  -d "{\"to_user_id\":\"$BOB_ID\",\"amount\":500}"; echo

echo "# balances"
echo "alice:"; curl -s "$BASE/balances/current" -H "Authorization: Bearer $TOKEN_ALICE"; echo
echo "bob:  "; curl -s "$BASE/balances/current" -H "Authorization: Bearer $TOKEN_BOB"; echo

echo "# history (alice)"
curl -s "$BASE/transactions/history?limit=10" -H "Authorization: Bearer $TOKEN_ALICE"; echo
