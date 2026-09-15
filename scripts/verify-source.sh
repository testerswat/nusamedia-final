#!/usr/bin/env sh
set -eu

fail=0
for f in apps/api/go.mod installer/go.mod; do
  if ! grep -q '^go 1.26$' "$f"; then
    echo "FAIL: $f is not pinned to Go 1.26" >&2
    fail=1
  fi
done
if find . -type d \( -name node_modules -o -name dist \) -print -quit | grep -q .; then
  echo "FAIL: generated dependency/build directories are present" >&2
  fail=1
fi
if find . -type f -iname '*appdeploy*' -o -iname '*snapshot*' | grep -q .; then
  echo "FAIL: deployment/snapshot artifacts are present" >&2
  fail=1
fi
if [ "$fail" -ne 0 ]; then exit 1; fi
echo "NusaMedia Final source verification: PASS"
