#!/usr/bin/env bash
set -euo pipefail

pnpm exec oxfmt --check || {
    echo 'Run mise run format to format the frontend' >&2
    exit 1
}
pnpm exec oxlint
node lint/fsd.ts
node lint/styles.ts
pnpm typecheck
pnpm --silent generate:api --check || {
    echo 'src/shared/api/schema.gen.ts is out of date, run mise run generate-api' >&2
    exit 1
}
