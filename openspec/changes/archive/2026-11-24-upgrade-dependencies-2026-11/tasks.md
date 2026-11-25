## 1. Database clients

- [x] 1.1 Add a test next to the PostgreSQL logical backup scenarios, named `TestBackupPostgresql18` with the issue number #921 next to it, that backs up a table from a PostgreSQL 18 server on the pinned minor and checks that the dump contains no `pg_attver`. Write it while the 18.4 bundle is still committed and confirm it fails with `source scripts/fedora.sh && apply_libedit_shim && TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -count=1 -run '^TestBackupPostgresql18$' ./internal/modules/backups/e2e/postgresql/...`; task 1.2 makes it pass.
- [x] 1.2 Refresh the PostgreSQL 14 to 18 bundles to 14.25-1.pgdg12+1, 15.20-1.pgdg12+1, 16.16-1.pgdg12+1, 17.12-1.pgdg12+1 and 18.7-1.pgdg12+2 on both architectures, with every tool each bundle ships, its records in `tools/dependencies/clients.json` and the table in `assets/tools/README.md`, which marks 14 as EOL after 14.25; verify with `mise run verify-deps`, `--version` of every refreshed binary in the runtime image, `100755` modes in the index and the test from 1.1 passing with the same command.
- [x] 1.3 Refresh the MySQL bundles to 8.4.13, 9.7.4 and 26.10.0 and MongoDB Database Tools to 100.20.1 `ubuntu2204` on both architectures, with their records and tables, after checking the oldest server MongoDB tools 100.20 support; verify with `mise run verify-deps` and `--version` of every refreshed binary in the runtime image.
- [x] 1.4 Move the PostgreSQL fixtures to the fixed minors within their lines: `docker-compose.yml`, `backend/internal/modules/backups/e2e/postgresql/testdata/{ssl,mtls}/Dockerfile`, `backend/internal/modules/backups/physicalapitests/testdata/mtls/Dockerfile`, `e2e/browser/compose.yaml` and `backend/internal/lib/testcontainers/testdata/cnpg/README.md`; verify with `mise run verify-deps`, `source scripts/fedora.sh && apply_libedit_shim && TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -count=1 -run 'OverTls|OverMutualTls' ./internal/modules/backups/e2e/postgresql/...`, `TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -count=1 ./internal/modules/backups/physicalapitests/backuping/... ./internal/modules/backups/physicalapitests/walstreaming/...`, `mise run e2e:unit` and `mise run e2e:browser`.

## 2. Backend Go modules

- [x] 2.1 Upgrade pgx to v5.13.0 with `gorm.io/driver/postgres` v1.6.3, `github.com/jlaffaye/ftp` to v0.2.4 and `golang.org/x/crypto` to v0.59.0; verify with `mise run backend:lint` and the Go tests of the repository, migration and FTP storage packages.
- [x] 2.2 Upgrade gorm to v1.31.2, gin to v1.12.0 with `gin-contrib/cors` v1.7.11 and `gin-contrib/gzip` v1.2.10, and kin-openapi to v0.153.0, adapting code to their behavior changes; verify with `mise run backend:lint` and `mise run test`.
- [x] 2.3 Upgrade testcontainers to v0.45.0 with `moby/moby/api` v1.57.0 and `moby/moby/client` v0.6.3, adapting the `wait.ForSQL` callback in `backend/internal/lib/testcontainers/mysql.go`; verify with `mise run backend:lint`, `mise run agent:test`, `mise run agent:e2e` and `mise run backend:test-upgrade`.
- [x] 2.4 Upgrade goose to v3.30.1 in `backend/go.mod` and `mise.toml`, `lib/pq` to v1.12.3, minio-go to v7.4.1, azcore to v1.24.0, azblob to v1.8.3, and `go-sql-driver/mysql` v1.10.1, `klauspost/compress` v1.20.4, `pkg/sftp` v1.13.12, `golang.org/x/tools` v0.52.1-0.20261106145633-d407941a339a, the commit goose v3.30.1 requires, `golang.org/x/oauth2` v0.38.0, gopsutil v4.26.10, `caarlos0/env` v11.4.2 and `go.opentelemetry.io/proto/otlp` v1.11.1; verify with `go -C backend mod tidy -diff`, `mise run backend:lint`, `mise run security:go` and the Go tests of the S3, Azure and SFTP storage packages.

## 3. Frontend and website

- [x] 3.1 Upgrade Mantine to 9.8.1, `tailwindcss` and `@tailwindcss/vite` to 4.3.5 with `source-map-js` 1.2.2 in the lock, `vite` to 8.3.8, `@vitejs/plugin-react` to 6.3.0, React and React DOM to 19.2.9, `lucide-react` 1.73.0, `recharts` 3.10.1, `cron-parser` 5.10.6, `dayjs` 1.11.23, `i18next` 26.7.2, `react-i18next` 17.1.13, `@testing-library/user-event` 14.6.12, `postcss` 8.5.33, `color-name` 2.1.1 and the `@redocly/openapi-core` override 1.34.24; verify with `pnpm --dir frontend install --frozen-lockfile --ignore-scripts`, `pnpm --dir frontend audit --audit-level high`, `pnpm --dir frontend typecheck` and `pnpm --dir frontend build`.
- [x] 3.2 Upgrade `oxlint` to 1.94.0, `oxfmt` to 0.79.0, `steiger` to 0.10.0 and `@feature-sliced/steiger-plugin` to 0.8.2, then run `mise run frontend:format`; verify with `mise run frontend:lint`.
- [x] 3.3 Upgrade the website's `eslint` to 10.15.0, `tailwindcss` and `@tailwindcss/postcss` to 4.3.5 and `@types/react*` to the newest eligible 19.2 typings; verify with `pnpm --dir website install --frozen-lockfile --ignore-scripts`, `pnpm --dir website audit --audit-level high`, `pnpm --dir website lint` and `pnpm --dir website build`.

## 4. Tooling and CI

- [x] 4.1 Upgrade pnpm to 12.21.0 in `mise.toml`, `tools/pnpm/package.json` with its lock, both `packageManager` fields and `frontend/browser/environment.json`; verify with `mise run verify-deps` and `pnpm --dir frontend install --frozen-lockfile --ignore-scripts`.
- [x] 4.2 Upgrade mise to 2026.11.9 in `min_version` and `tools/dependencies/mise-bootstrap.json` with hashes checked against the release digests, `hk` to 2.13.0 with `hk.pkl`, `trivy` 0.76.2, `vacuum` 0.41.0, `process-compose` 1.124.0, `python` 3.14.10, `uv` 0.13.8, `ruff` 0.17.5, `ty` 0.0.104, `gh` 2.104.0, `helm` 4.3.2, `chrome-devtools-mcp` 1.13.0, `typescript-language-server` 6.1.0, `semgrep` 1.185.0 and `@fission-ai/openspec` 1.17.0, refreshing `mise.lock` and its sidecars; verify with `mise install --locked --include-task-tools`, `mise run verify-deps` and `mise run lint`.
- [x] 4.3 Upgrade `pytest` to 9.2.0 in `e2e/browser` with `uv lock`; verify with `uv sync --locked --project e2e/browser` and `mise run e2e:unit`.
- [x] 4.4 Pin `actions/download-artifact` v8.0.2, `actions/upload-artifact` v7.0.2, `docker/login-action` v4.8.0, `github/codeql-action` v4.39.3 and `jdx/mise-action` v5.2.9 by full commit SHA, checking that the action's own install step does not conflict with the workflows' install arguments; verify with `mise run lint-workflows` and `mise run verify-deps`.

## 5. Backup pipeline on the final dependency set

- [x] 5.1 Run the backup and restore matrices after sections 1 to 4, from the repository root with `source scripts/fedora.sh` and `apply_libedit_shim`: `TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -count=1 -timeout 30m ./internal/modules/backups/e2e/postgresql/... ./internal/modules/backups/e2e/mysql/... ./internal/modules/backups/e2e/mariadb/... ./internal/modules/backups/e2e/mongodb/...`, then the physical suites `./internal/modules/backups/e2e/postgresqlphysical/pg17`, `pg18`, `patroni` and `cnpg` with the commands and prerequisites in `docs/testing.md`, and `mise run backend:test-upgrade`, which builds the image with the refreshed clients.

## 6. Report

- [x] 6.1 Record in this change the selected versions with their sources and publication dates, the PostgreSQL client exception, the deferred releases with their eligibility dates, the open mongo-driver advisory and the EOL lines PostgreSQL 12 to 14, MySQL 5.7 and 8.0 and MariaDB 13.0; verify that every version in `tools/dependencies/clients.json`, `mise.toml` and the manifests appears in the record.

## Change verification

```sh
mise run test
```
