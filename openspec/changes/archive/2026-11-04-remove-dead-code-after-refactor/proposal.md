## Why

The refactor left unused code and files behind. A reader spends time on them, and the module contracts list connections that no longer exist. Three compose services nobody uses also keep running: the two physical fixtures are built and started on every `mise run dev` and `mise run test`, and `backend-victoria-logs`, which floats on its `latest` tag, starts on every `mise run test`.

## What Changes

Removed from the backend:

- Contract methods no other module calls: six in `backups`, `CanInviteUsers` in `users`, `DeleteDatabase` in `databases`.
- Code nothing calls: `GetVersionEnum`, `gfs.FormatTiers`, `chains.GetChainSpan`, both `HasPrivilege` methods of the connection settings, `apptest.WithFileTimings`, `DefaultForcedRotationInterval`, `mailpittest.Client.Clear`, the unused `FirstLSN` of `fixtureSegmentNo` in `streamstatistics`.
- Constants nothing reads: `ftpChunkSize`, `noRowLock` in the backups config repository, `BackupErrorCanceledByDbRemoval`.
- Code only tests call: `namelist.ParseUniqueNames`, `namelist.FormatUniqueNames`, `CancellationRegistry.WaitUntilEmpty` with its `isEmpty` channel, `cache.Store.Clear`, `CountByReferences`, `FindByReference`, `mysqlfamily.HasPrivilege`. The tests that used them read through existing code or, for `FindByReference`, an inline query in their fixture.

Corrected in the backend:

- Restore statuses in `restore_repository.go` and the physical backup types in `physical_backup_list_row.go` use their constants instead of string literals.
- Stale references go: `util/encryption` in `chunked_manifest.go`, the issue number in `archiveownerroles.go`, `cmd/cleanup_test_db` in `backend/.air.toml` and both `.gitignore` files.

Removed from the repository:

- **BREAKING (developer environment):** the compose services `test-physical-postgres-17`, `test-physical-postgres-18` and `backend-victoria-logs` are deleted, along with `TEST_PHYSICAL_POSTGRES_17_PORT` and `TEST_PHYSICAL_POSTGRES_18_PORT` in `.env.example`. No shim remains. A developer who ran them by hand loses them; the owner confirmed nobody does.
- `backend/ui/readme.md`.
- **BREAKING (developer environment):** `mise run test` and the CI jobs `e2e-verification-agent-podman` and `test-backend` start only the shared test PostgreSQL server, not every compose service.

Removed from the frontend and the website:

- 13 unused keys in all six interface dictionaries.
- `frontend/public/favicon.svg` and the icons `warn-red.svg`, `menu/arrow-down-gray.svg`, `menu/users-gray.svg`, `menu/users-white.svg`.
- The `@vitest/coverage-v8` dependency, which changes `frontend/pnpm-lock.yaml`.
- 13 of the 18 font files in `website/public/font/`.

### Out of scope

- Unused aliases and constants in the `module.go` files, beyond those that only the removed methods need.
- Re-exports in frontend `index.ts` files, `WalStreamHealth.ts` and the duplicate `formatBackupSize` in `VerificationDetailDrawer.tsx`.
- Stale `.dockerignore` entries, website images, and moving `.agent-hooks/test_group_test.py`.
- The three `containsIgnoreCase` copies, `mise-tasks/backend/clean-testcontainers`, `assets/logo-square.*` and the Postgresus volume check in `docker/start.sh`, which all stay. `design.md` gives the reasons.
- `WalSegmentNo.GetFilename`, which tests of two packages share.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `local-development-workflow`: `mise run test` starts only the shared test PostgreSQL server. What survives an interrupted test run and what `mise run agent:e2e` needs are named as that server rather than the shared test fixtures, and the reason the fixtures keep running after `mise run dev` no longer names the backend suite.

## Impact

- The change answers to `backend/AGENTS.md`, `frontend/AGENTS.md`, `website/AGENTS.md` and the root `AGENTS.md`.
- Users of Databasus see no change: no HTTP operation, schema, migration or interface text in use changes.
- Internal contracts of `backups`, `users` and `databases` shrink. The compiler reports anything that still depends on them.
- `frontend/pnpm-lock.yaml` and `.env.example` change as part of this change. Starting apply is the owner's confirmation of both; an owner who declines cuts those parts from the change first.
- `.github/workflows/ci-release.yml` changes in a way only a CI run fully proves; `design.md` describes the risk.
