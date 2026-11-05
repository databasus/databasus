Run the direct Go commands below from `backend/`, using the pinned tools and the Fedora setup from `docs/testing.md`:

```sh
source ../scripts/fedora.sh
apply_libedit_shim
set -a
source ../.env
set +a
```

## 1. Expose the shared configuration operations

- [x] 1.1 Expose only the existing config repository, validation, normalization and cleanup-decoration operations needed by the save service, keeping SQL, defaults, locks and publication checks in `config`. Build one repository bundle for reuse. Verify existing config behavior with `TESTCONTAINERS_RYUK_DISABLED=true go test -count=1 -run '^Test(ReadBackupConfig|SaveBackupConfig|ListBackupConfigs|BackupConfigAccess)' ./internal/modules/backups`.

## 2. Move save orchestration and remove the cycle

- [x] 2.1 Add `internal/configsave` with `SaveBackupConfig`, `SwitchStorage` and storage-change cleanup; call ready logical and physical services directly, split physical saving into the phases in `design.md`, and update both the HTTP handler and transfer service to use it. Preserve locked reauthorization, request preservation/reset rules, cleanup scope, storage-generation conditions and reactions after the outermost commit. Verify the integration with `TESTCONTAINERS_RYUK_DISABLED=true go test -count=1 ./internal/modules/backups`.
- [x] 2.2 Construct config, logical/physical and config-save services in dependency order; remove late-bound variables, callback types/dependencies and old config save/switch entry points. Keep `HistoryRemoval` and `RemovedBackupHistory` in `config`, and update package/service comments to describe their resulting responsibilities. Run `mise run backend:format` and verify dependencies, construction and formatting with `mise run backend:lint`.

## 3. Verify the complete refactor

- [x] 3.1 Run the existing backups root tests, internal tests and every physical API test without adding or changing scenarios: `TESTCONTAINERS_RYUK_DISABLED=true go test -count=1 ./internal/modules/backups ./internal/modules/backups/internal/... ./internal/modules/backups/physicalapitests/...`. This covers configuration access and validation, workspace/storage transfers, cleanup rollback, physical cancellation and stale FULL/WAL/history publication. If verification exposes a defect that requires a test change, add the required `mise run test` change-verification section and run it before review, as `openspec/config.yaml` requires.
