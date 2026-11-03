Backend test commands run in `backend/` in the full form of `docs/testing.md` (Commands), with the libedit shim applied and `TESTCONTAINERS_RYUK_DISABLED=true`. Below, `go test` stands for that form.

## 1. Failing tests

- [x] 1.1 Add `TestCancelPhysicalBackup` under `backend/internal/modules/backups/physicalapitests/catalog` with the cases "returns the backup in its final status" for a full backup and for an incremental backup whose parent is the root full backup. Each asserts `200`, the status `CANCELED`, and equality with the row the list returns right after. Verify that `go test -count=1 -run 'TestCancelPhysicalBackup' ./internal/modules/backups/physicalapitests/catalog` fails because the response is `202`.
- [x] 1.2 Rename `TestTriggerPhysicalBackup` (`access_test.go:84`) to `TestStartPhysicalBackup`, matching the operation. Add the cases "returns invalid_request when backups are turned off", "returns invalid_request when backups are turned off and an incremental has no chain", "requests no incremental when backups are turned off and a chain exists", "returns invalid_request when the backup config was never saved", "returns invalid_request when backups are on and the config names no storage", "returns permission_denied to a viewer when backups are turned off" and "returns not_found to an outsider when backups are turned off". The turned-off, no-storage, viewer and outsider cases also assert that `isFullBackupPending` is false. Verify that `go test -count=1 -run 'TestStartPhysicalBackup' ./internal/modules/backups/physicalapitests/catalog` fails on the cases that expect `400`.

## 2. Contract

- [x] 2.1 In `backend/api/openapi.yaml`, make `cancelPhysicalBackup` answer `200` with `PhysicalBackupListRow`, delete its `202`, and state the final status in its description. Extend the `startPhysicalBackup` description with the `400` for backups turned off and for no storage, and with the reason a start answers without a body. Run `mise run generate-api` and verify with `mise run backend:check-generated`.

## 3. Backend

- [x] 3.1 Add the optional `ID` to `catalog.BackupListFilter`, make `Service.CancelBackup` return the item read through `ListBackups` inside its transaction, and answer it from `CancelPhysicalBackup` through `toPhysicalBackupListRow`, per design decisions 2 and 3. Verify that the test from 1.1 passes with its `go test` command.
- [x] 3.2 Rework `StartBackup` per design decision 4: one authorization, one config read, the refusals in `physical/errors.go`, then private branches by kind that replace the public `RequestBackup`, `RequestFullBackup` and `RequestIncrementalBackup`. Verify that the test from 1.2 passes with its `go test` command.
- [x] 3.3 Change the tests and helpers that expect `202` from a physical cancel to expect `200`: `physicalapitests/backuping/shutdown_test.go`, `physicalapitests/backuping/running_backup_test.go` and `physicalapitests/catalog/operations_test.go`. Where a test starts a backup on a database with backups off or no storage and is not about the refusal, turn backups on or set a storage in its setup. Verify with `go test -count=1 ./internal/modules/backups/physicalapitests/...`.

## 4. Frontend

- [x] 4.1 In `frontend/src/pages/databases/ui/physical-backups/PhysicalBackupsComponent.test.tsx`, give the shared `config` fixture a storage ID, then add cases: the start is disabled with the explanation when the loaded config has backups off, the same when backups are on and it names no storage, and the start stays enabled without the explanation when the config fails to load. Verify with `pnpm --dir frontend exec vitest run src/pages/databases/ui/physical-backups/PhysicalBackupsComponent.test.tsx` that the new disabled cases fail and the existing start tests pass.
- [x] 4.2 Disable "Back up now" and show the explanation in `PhysicalBackupsComponent` per design decision 5, adding one key under `backups.physical.list` to all six dictionaries as the `frontend-localization` skill says. Update test mocks that answer a physical cancel with `202` so they match the contract. Verify with the same command, `pnpm --dir frontend typecheck` and `pnpm --dir frontend lint`.
- [x] 4.3 In `frontend/browser/scenarios/physicalBackups.spec.ts`, give the config fixture at line 92 a storage ID, so the click on "Back up now" at line 334 still works. Verify with `pnpm --dir frontend test:browser --project=ui`.
- [x] 4.4 Repair the flaky `databaseTransfer.spec.ts` scenario that the browser check exposed, per design decision 7: make `enlargeText` wait for CSS transitions, let the storage type row shrink in `EditStorageComponent`, and exempt a select value's ellipsis in `assertAccountReflow`. Verify with `pnpm --dir frontend test:browser --project=ui`, also while `mise run test` runs alongside it.

## 5. Documentation

- [x] 5.1 In `docs/architecture.md`, section "HTTP API", record the physical start as the exception to the long-operation rule with its reason and its limit, and record that a physical list page carries `totalUsageMb` with the reason. In `.agents/skills/api-design/SKILL.md`, link "Long operations" to that exception, say that a cancel with a final status answers `200`, make clear in "Pagination" that the ban on totals concerns paging counts, and add to "Errors" that a start whose target config cannot back up answers `400 invalid_request`, citing `startLogicalBackup` as the precedent. Verify with `rg -n "202|invalid_request" docs/architecture.md .agents/skills/api-design/SKILL.md` that no statement contradicts these.
- [x] 5.2 Search for other mentions of the changed answers with `rg -n "physical.*(cancel|start)|cancelPhysicalBackup|startPhysicalBackup" docs README.md backend/AGENTS.md frontend/AGENTS.md .agents/skills` and bring any that are out of date in line. Verify that a repeated search shows none.

## 6. Checks

- [x] 6.1 Run `mise run format` and `mise run lint` and fix what they report, touching only this change's files. Verify that `mise run lint` exits 0.

## Change verification

```sh
mise run test
pnpm --dir frontend test:browser --project=ui
```
