#!/usr/bin/env bash
set -euo pipefail

root="$(cd "${BASH_SOURCE[0]%/*}/.." && pwd)"
exec python3 -B "$root/scripts/verify_dependencies.py"
