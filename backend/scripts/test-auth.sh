#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8000}"
SUFFIX="${SUFFIX:-$(date +%s)}"
EMAIL="${EMAIL:-demo-${SUFFIX}@acme.com}"
PASSWORD="${PASSWORD:-securepass123}"
COMPANY_NAME="${COMPANY_NAME:-Demo Corp ${SUFFIX}}"

echo "==> Health check"
curl -s "$BASE_URL/health" | python3 -m json.tool
echo

echo "==> Signup ($EMAIL)"
SIGNUP_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/auth/signup" \
  -H "Content-Type: application/json" \
  -d "{\"company_name\":\"$COMPANY_NAME\",\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\",\"first_name\":\"Jane\",\"last_name\":\"Doe\"}")
echo "$SIGNUP_RESPONSE" | python3 -m json.tool

TOKEN=$(echo "$SIGNUP_RESPONSE" | python3 -c "import sys, json; print(json.load(sys.stdin)['data']['access_token'])" 2>/dev/null || true)

if [ -z "$TOKEN" ]; then
  echo
  echo "==> Signup failed (email may already exist). Trying login..."
  LOGIN_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}")
  echo "$LOGIN_RESPONSE" | python3 -m json.tool
  TOKEN=$(echo "$LOGIN_RESPONSE" | python3 -c "import sys, json; print(json.load(sys.stdin)['data']['access_token'])")
fi

echo
echo "==> Protected /me (JWT middleware)"
curl -s "$BASE_URL/api/v1/auth/me" \
  -H "Authorization: Bearer $TOKEN" | python3 -m json.tool
