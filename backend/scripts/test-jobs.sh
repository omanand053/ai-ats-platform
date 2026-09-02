#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8000}"
EMAIL="${EMAIL:-jobs-test@example.com}"
PASSWORD="${PASSWORD:-Password@123}"

echo "==> Login (or signup if needed)"
LOGIN=$(curl -s -X POST "$BASE_URL/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}")

TOKEN=$(echo "$LOGIN" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('data',{}).get('access_token',''))" 2>/dev/null || true)

if [ -z "$TOKEN" ]; then
  echo "Signup first..."
  SIGNUP=$(curl -s -X POST "$BASE_URL/api/v1/auth/signup" \
    -H "Content-Type: application/json" \
    -d "{\"company_name\":\"Jobs Test Co\",\"company_slug\":\"jobs-test-$(date +%s)\",\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\",\"first_name\":\"Jobs\",\"last_name\":\"Tester\"}")
  echo "$SIGNUP" | python3 -m json.tool
  TOKEN=$(echo "$SIGNUP" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['access_token'])")
fi

echo "==> Create job"
CREATE=$(curl -s -X POST "$BASE_URL/api/v1/jobs" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"title":"Senior Go Engineer","department":"Engineering","location":"Remote","employment_type":"full_time","experience_required":"5+ years","required_skills":["Go","PostgreSQL"],"status":"open"}')
echo "$CREATE" | python3 -m json.tool

JOB_ID=$(echo "$CREATE" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('data',{}).get('job',{}).get('id',''))" 2>/dev/null || true)

echo "==> List jobs"
curl -s "$BASE_URL/api/v1/jobs?page=1&limit=10" \
  -H "Authorization: Bearer $TOKEN" | python3 -m json.tool

if [ -n "$JOB_ID" ]; then
  echo "==> Get job by ID"
  curl -s "$BASE_URL/api/v1/jobs/$JOB_ID" \
    -H "Authorization: Bearer $TOKEN" | python3 -m json.tool
fi
