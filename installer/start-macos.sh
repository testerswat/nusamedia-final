#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
ARCH="$(uname -m)"
if [ "$ARCH" = "arm64" ]; then BIN="$ROOT/installer/bin/nusamedia-installer-macos-arm64"; else BIN="$ROOT/installer/bin/nusamedia-installer-macos-amd64"; fi
exec "$BIN" --root "$ROOT"
