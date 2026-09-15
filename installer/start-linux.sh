#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
exec "$ROOT/installer/bin/nusamedia-installer-linux-amd64" --root "$ROOT"
