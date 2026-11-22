#!/usr/bin/env bash
# RailShack · 50 Microservices Project by DevOps Shack
# Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
# SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.
# End-to-end smoke test: health of all 50 services, sign up / sign in, then search → book → PNR → My Trips → cancel,
# checking that booking, My Trips and cancelling refuse requests without a login.
# Usage: scripts/smoke-test.sh <base-url>   e.g. https://railshack.prod-bat.site (CI passes the runner's address)
# On Windows use the PowerShell version: .\scripts\smoke-test.ps1 [base-url]
set -euo pipefail
BASE="${1:?usage: scripts/smoke-test.sh <base-url>}"
DATE="$(date -d tomorrow +%F 2>/dev/null || date -v+1d +%F)"
pass() { printf '  \033[32m✔\033[0m %s\n' "$1"; }
fail() { printf '  \033[31m✘\033[0m %s\n' "$1"; exit 1; }
j() { python3 -c "import json,sys;d=json.load(sys.stdin);print($1)"; }

echo "RailShack smoke test → $BASE"
UP=$(curl -sf "$BASE/api/platform/status" | j "d['up']") || fail "platform status unreachable"
[ "$UP" = "50" ] && pass "all 50 microservices healthy" || fail "only $UP/50 services healthy"

SMOKE_USER="smoke$(date +%s)$RANDOM"
TOKEN=$(curl -sf -XPOST "$BASE/api/auth/signup" -d "{\"user\":\"$SMOKE_USER\",\"password\":\"smoke-pass-123\",\"name\":\"Smoke Test\"}" | j "d['token']") || fail "sign up"
pass "sign up created $SMOKE_USER and issued a token"
curl -sf -XPOST "$BASE/api/auth/login" -d "{\"user\":\"$SMOKE_USER\",\"password\":\"smoke-pass-123\"}" >/dev/null || fail "sign in"
CODE=$(curl -s -o /dev/null -w '%{http_code}' -XPOST "$BASE/api/auth/login" -d "{\"user\":\"$SMOKE_USER\",\"password\":\"wrong-password\"}")
[ "$CODE" = "401" ] && pass "sign in works, wrong password is refused" || fail "wrong password returned $CODE"
AUTH="Authorization: Bearer $TOKEN"

COUNT=$(curl -sf "$BASE/api/search/trains?from=NDLS&to=BSB&date=$DATE" | j "d['count']") || fail "search"
[ "$COUNT" -ge 1 ] && pass "search NDLS → BSB returned $COUNT trains" || fail "search returned no trains"

BODY="{\"train\":\"22436\",\"from\":\"NDLS\",\"to\":\"BSB\",\"date\":\"$DATE\",\"class\":\"CC\",\"quota\":\"GN\",\"passengers\":[{\"name\":\"Asha\",\"age\":31,\"gender\":\"F\"},{\"name\":\"Ravi\",\"age\":64,\"gender\":\"M\"}],\"payment\":\"wallet\",\"coupon\":\"BLACKFRIDAY\",\"insurance\":true}"
CODE=$(curl -s -o /dev/null -w '%{http_code}' -XPOST "$BASE/api/booking/book" -d "$BODY")
[ "$CODE" = "401" ] && pass "booking without signing in is refused" || fail "booking without login returned $CODE"
BOOK=$(curl -sf -XPOST "$BASE/api/booking/book" -H "$AUTH" -d "$BODY" -D /tmp/rs_headers) || fail "booking"
PNR=$(echo "$BOOK" | j "d['pnr']")
HOPS=$(grep -i '^x-trace' /tmp/rs_headers | tr ',' '\n' | wc -l)
pass "booked PNR $PNR (status $(echo "$BOOK" | j "d['status']"), total ₹$(echo "$BOOK" | j "d['total']"), touched $HOPS services)"

STATE=$(curl -sf "$BASE/api/pnr/$PNR" | j "d['state']") || fail "pnr lookup"
[ "$STATE" = "ACTIVE" ] && pass "PNR lookup works" || fail "unexpected PNR state $STATE"
SRC=$(curl -sf "$BASE/api/pnr/$PNR" | j "d.get('servedFrom','')")
case "$SRC" in
  redis)  pass "PNR saved in PostgreSQL, second read served from the Redis cache" ;;
  memory) pass "PNR stored in memory (no DATABASE_URL: local mode)" ;;
  *)      fail "second PNR read came from '$SRC', expected the Redis cache" ;;
esac

CODE=$(curl -s -o /dev/null -w '%{http_code}' "$BASE/api/booking-history/$SMOKE_USER")
[ "$CODE" = "401" ] && pass "My Trips without signing in is refused" || fail "My Trips without login returned $CODE"
CODE=$(curl -s -o /dev/null -w '%{http_code}' -H "$AUTH" "$BASE/api/booking-history/devopsshack")
[ "$CODE" = "403" ] && pass "someone else's trips are refused" || fail "other user's trips returned $CODE"
for i in 1 2 3 4 5; do   # booking-history is written in the background
  TRIPS=$(curl -sf -H "$AUTH" "$BASE/api/booking-history/$SMOKE_USER" | j "len(d['trips'])") || fail "trips"
  [ "$TRIPS" -ge 1 ] && break; sleep 1
done
[ "$TRIPS" -ge 1 ] && pass "trip saved in My Trips" || fail "trip not saved"

CODE=$(curl -s -o /dev/null -w '%{http_code}' -XPOST "$BASE/api/cancellation/cancel" -d "{\"pnr\":\"$PNR\"}")
[ "$CODE" = "401" ] && pass "cancelling without signing in is refused" || fail "cancel without login returned $CODE"
REFUND=$(curl -sf -XPOST "$BASE/api/cancellation/cancel" -H "$AUTH" -d "{\"pnr\":\"$PNR\"}" | j "d['refund']['amount']") || fail "cancellation"
pass "cancelled with refund ₹$REFUND"

curl -sf "$BASE/api/live-status/12301" >/dev/null && pass "live train status" || fail "live status"
curl -sf "$BASE/api/admin/dashboard" >/dev/null && pass "ops dashboard" || fail "admin dashboard"
curl -sf "$BASE/" -o /tmp/rs_index.html && grep -q "RailShack" /tmp/rs_index.html && pass "web UI served" || fail "web UI"
echo "🎉 smoke test passed"
