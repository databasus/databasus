## Why

The bundled `pg_dump` 18.4 writes the internal column option `pg_attver` that PostgreSQL 18.7 stores in `attoptions`. A server older than the fixed minor rejects that option on restore (databasus/databasus#921). The fixed clients 14.25, 15.20, 16.16, 17.12 and 18.7 stop writing it. The verification agent already restores into those minors. The bundled clients still produce dumps that older servers reject. Other dependencies have also fallen behind, including security releases of pgx, the FTP client, `golang.org/x/crypto`, Trivy and the frontend's `source-map-js`. One maintenance pass refreshes them together under `docs/how-to-upgrade-deps.md`.

## What Changes

- Refresh the PostgreSQL client bundles 14 to 18 to 14.25, 15.20, 16.16, 17.12 and 18.7 on both architectures. Each bundle gets its whole tool set, including `pg_basebackup`, `pg_receivewal`, `pg_combinebackup` and `pg_waldump` where the bundle has them. The fixed minors were published on 2026-11-12. The fourteen-day waiting period is shortened by three days, because the release fixes a defect that breaks users' verification. Bundles 12 and 13 stay, because their lines are EOL and received no fix.
- Refresh the MySQL client bundles to 8.4.13, 9.7.4 and 26.10.0, and MongoDB Database Tools to 100.20.1.
- Move the PostgreSQL test fixtures within their release lines to the fixed minors. Add a test that pins #921: a dump produced by the bundled client of a fixed minor contains no `pg_attver`.
- Upgrade backend Go modules to their newest eligible patch and minor releases, including pgx 5.13, the tagged FTP client v0.2.4, `golang.org/x/crypto` v0.59, gin 1.12, gorm 1.31, kin-openapi 0.153, testcontainers 0.45 with its moby modules, and goose 3.30 with its mise pin.
- Upgrade frontend packages within their current majors, including Mantine 9.8, Tailwind 4.3.5, Vite 8.3, oxlint, oxfmt and steiger, plus React 19.2.9 to match the website. Upgrade website ESLint and its lock.
- Upgrade pnpm 12.21, mise, hk, Trivy, vacuum, Python tooling and the other mise tools with their locks and sidecars, the browser-test Python dependency `pytest`, and the pinned GitHub Actions.
- Adapt code, configuration and tests to the behavior changes of these releases where they surface. Main specs stay unchanged.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

None. Backup, restore and verification behavior keeps satisfying the main specs. `skip_specs: true` records that the change moves dependency versions, not requirements.

## Out of scope

- Major migrations: TypeScript 6/7 in the frontend and 7 in the website, i18next 27 and react-i18next 18, msw 3, Node 26, `@redocly/openapi-core` 2, OpenSpec 2, ripgrep 16, `pkg/sftp` v2.
- React 19.3. Its transitions now render independently, which changes UI behavior and deserves its own check.
- MongoDB Go driver 2.9 and newer. They drop MongoDB 4.2, which Databasus supports. The driver stays on v2.8.2; design.md explains the advisory this leaves open.
- MariaDB 13.0 to the 13.1 rolling line. It is a line change, not a patch.
- Releases still inside their waiting period on 2026-11-24: Node 24.22.3, Playwright 1.65.1, govulncheck 1.10.0, `golang.org/x/tools` v0.53.0, `golang.org/x/net` v0.63.0, mongo-driver v2.10.1, `klauspost/compress` v1.20.5, `go-sql-driver/mysql` v1.10.2, `oapi-codegen` v2.9.0 and its runtime, MariaDB 12.3.4 and the MySQL 8.4.13.1/9.7.4.1/26.10.0.1 updates.
- Replacing `hirochachacha/go-smb2`. It is abandoned, GO-2026-5051 has no fix, and govulncheck finds no reachable call.
- Cleaning up fallback `postgres:<version>` and TimescaleDB images on verification agents.

## Impact

Bundled database clients under `assets/tools/` and their records in `tools/dependencies/clients.json` and `assets/tools/README.md`; `backend/go.mod` and the Go test fixtures; `frontend/` and `website/` manifests and locks; `mise.toml`, `mise.lock`, `.mise/locks/`, `hk.pkl`, `tools/pnpm/`, `tools/dependencies/mise-bootstrap.json`, `frontend/browser/environment.json`; `e2e/browser/` Python locks; workflow action pins under `.github/workflows/`; PostgreSQL fixtures in Compose files and test Dockerfiles. The change answers to `AGENTS.md`, `backend/AGENTS.md`, `frontend/AGENTS.md` and `website/AGENTS.md`. No HTTP API, persisted data or supported server version changes. Contributors reinstall the pinned tools with `mise install --locked --include-task-tools`.
