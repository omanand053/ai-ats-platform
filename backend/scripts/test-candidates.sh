#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8000}"
EMAIL="${EMAIL:-candidates-test@example.com}"
PASSWORD="${PASSWORD:-Password@123}"

echo "==> Login (or signup)"
LOGIN=$(curl -s -X POST "$BASE_URL/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}")

TOKEN=$(echo "$LOGIN" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('data',{}).get('access_token',''))" 2>/dev/null || true)

if [ -z "$TOKEN" ]; then
  SIGNUP=$(curl -s -X POST "$BASE_URL/api/v1/auth/signup" \
    -H "Content-Type: application/json" \
    -d "{\"company_name\":\"Candidates Test\",\"company_slug\":\"candidates-test-$(date +%s)\",\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\",\"first_name\":\"Test\",\"last_name\":\"User\"}")
  TOKEN=$(echo "$SIGNUP" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['access_token'])")
fi

CANDIDATE_EMAIL="candidate-$(date +%s)@example.com"

echo "==> Create candidate"
CREATE=$(curl -s -X POST "$BASE_URL/api/v1/candidates" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"name\":\"Jane Doe\",\"email\":\"$CANDIDATE_EMAIL\",\"experience_years\":4,\"current_company\":\"TechNova\",\"current_designation\":\"Software Engineer\",\"location\":\"Bengaluru\",\"skills\":[\"Go\",\"PostgreSQL\",\"Docker\",\"Redis\",\"REST API\"],\"resume_summary\":\"Backend Engineer with 4 years of experience in Go, PostgreSQL and distributed systems.\",\"resume_text\":\"Experienced Backend Engineer with 4 years of experience building scalable REST APIs using Go, Gin, PostgreSQL, Redis and Docker. Designed authentication systems, optimized SQL queries, implemented caching, integrated CI/CD pipelines and deployed applications on Kubernetes. Strong understanding of distributed systems, microservices, concurrency and API security.\",\"status\":\"applied\",\"source\":\"Website\"}")
echo "$CREATE" | python3 -m json.tool

CID=$(echo "$CREATE" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('data',{}).get('candidate',{}).get('id',''))" 2>/dev/null || true)

echo "==> List candidates"
curl -s "$BASE_URL/api/v1/candidates?page=1&limit=10&search=jane" \
  -H "Authorization: Bearer $TOKEN" | python3 -m json.tool

if [ -n "$CID" ]; then
  echo "==> Get by ID"
  curl -s "$BASE_URL/api/v1/candidates/$CID" \
    -H "Authorization: Bearer $TOKEN" | python3 -m json.tool
fi
