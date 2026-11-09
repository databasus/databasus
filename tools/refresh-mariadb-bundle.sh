#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
exec python3 -B "$REPO_ROOT/scripts/refresh_mariadb_bundle.py" "$@"
