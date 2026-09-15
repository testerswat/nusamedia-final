#!/usr/bin/env bash
set -euo pipefail
BASE="${BASE_URL:-http://localhost:8080}"
echo '[1] health'; curl -fsS "$BASE/api/health" >/dev/null
echo '[2] public settings'; curl -fsS "$BASE/api/v1/public/settings" >/dev/null
echo '[3] discover validation'; code=$(curl -sS -o /dev/null -w '%{http_code}' "$BASE/api/v1/discover/nearby"); test "$code" = "422"
echo 'NusaMedia smoke checks passed.'
