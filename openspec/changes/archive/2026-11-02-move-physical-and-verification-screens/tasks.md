## 1. Map coverage

- [x] 1.1 Run `pnpm --dir frontend test` and record the file and test counts. Then create `coverage.md` in this change, mapping every test file under `features/backups`, `features/verification` and `entities/backups`, and `pages/databases/ui/EditPhysicalBackupConfigComponent.test.tsx`, to its final path and retained guarantee. Verification: `coverage.md` lists each test file that `rg --files frontend/src/features/backups frontend/src/features/verification frontend/src/entities/backups | rg 'test\.tsx?$'` prints, and it lists `pages/databases/ui/EditPhysicalBackupConfigComponent.test.tsx`.

## 2. Move WAL stream monitoring and physical backups

- [x] 2.1 Move `features/backups/physical/monitoring` to `pages/databases/{ui,lib,model}/wal-stream-monitoring`, and `entities/backups/physical/model/WalStream*` and `physicalWalStreamStatisticsApi` next to it, changing only import paths. Point `DatabaseComponent` at the new files, update the palette path in `frontend/lint/styleChecks.ts` and `frontend/tooling/styleChecks.test.ts`, and update the moved entries of `style-baseline.json` to the new source and owner `pages/databases`. Verification: `pnpm --dir frontend test src/pages/databases/lib src/pages/databases/ui/wal-stream-monitoring src/pages/databases/ui/DatabaseComponent.test.tsx tooling/styleChecks` and `mise run frontend:lint`.
- [x] 2.2 Move the physical list, filters, type cell, restore screen, editor, show view and their tests, `restoreCommands`, and `physicalBackupStatus.ts` under its new name `physicalBackupStatusBadgeStyles.ts`, according to the table in `design.md`, changing only import paths and that name. Delete `features/backups` with no forwarding exports, and update the `style-baseline.json` entries. These files keep importing `entities/backups/physical` for now, which the layer direction allows. Verification: `pnpm --dir frontend test src/pages/databases` and `mise run frontend:lint`.
- [x] 2.3 Move the rest of `entities/backups/physical` to `pages/databases/api` and `model/physical-backups`, changing only import paths. Delete `entities/backups` with no forwarding exports. Verification: `pnpm --dir frontend test src/pages/databases` and `mise run frontend:lint`.
- [x] 2.4 Remove `renderStorageSelection` from the physical editor, which renders `BackupStorageSelection` directly, and from its call sites and tests. Make `StorageSelectionOptions` the `Props` of `BackupStorageSelection`, and delete `entities/storages/ui/StorageSelectionOptions.ts` and its export. Verification: `pnpm --dir frontend test src/pages/databases/ui/backup-config src/pages/databases/ui/CreateDatabaseComponent.test.tsx src/pages/databases/ui/DatabaseConfigComponent.test.tsx` passes, and `rg -n 'renderStorageSelection|StorageSelectionOptions' frontend/src` finds nothing.

## 3. Move verification

- [x] 3.1 Create `entities/verification-agents` from `entities/verification/agents`, with `models` renamed to `model`, and point its consumers at the new public API. Verification: `pnpm --dir frontend test src/features/verification src/pages/instance-settings` and `mise run frontend:lint`.
- [x] 3.2 Move the agent screen, its test, `verificationAgentCommands` and `agentStatus` to `pages/instance-settings` according to the table in `design.md`. Update the `style-baseline.json` entries. Verification: `pnpm --dir frontend test src/pages/instance-settings` and `mise run frontend:lint`.
- [x] 3.3 Move the verification config and list screens, their APIs and types to `pages/databases` according to the table in `design.md`. Delete `features/verification` and `entities/verification` with no forwarding exports, update the `style-baseline.json` entries, and remove the two `fsd/public-api` entries of `fsd-baseline.json`, because `lint:fsd` reports a stale entry. Verification: `pnpm --dir frontend test src/pages/databases` and `mise run frontend:lint`.

## 4. Checks and documentation

- [x] 4.1 Delete `NESTED_SLICES` and `MIGRATION_GROUPS` from `frontend/lint/checkProjectDependencies.ts`, and delete the `nested group` and `separates new sibling scenarios inside an existing migration group` cases from `frontendArchitecture.test.ts`. Record both cases with the reason in `coverage.md`. The delta spec of `frontend-code-checks` removes the matching scenario. Verification: `pnpm --dir frontend test src/shared/lib/test/frontendArchitecture.test.ts` and `mise run frontend:lint`.
- [x] 4.2 Update "Database pages compose reusable editing" in `docs/architecture.md`: the page owns the physical backup screens, WAL stream monitoring and verification, and the physical editor uses the storage selection directly, like the logical editor. Add that the instance settings page owns the agent screen and that `entities/verification-agents` exists for the two pages. Remove the sentence about nested domain groups. Verification: `rg -n 'through their public APIs\. The wizard|as a render prop|Nested domain groups' docs/architecture.md` finds nothing, and `rg -n 'entities/verification-agents' docs/architecture.md` finds a line.
- [x] 4.3 Mark this item done in `temp/refactor/frontend-plan.md`, and complete `coverage.md` with each test changed beyond its imports. Verification: `rg -n '\[x\] \*\*7\.2\.' temp/refactor/frontend-plan.md` finds the line.
- [x] 4.4 Format the changed files and run `mise run frontend:lint`, `pnpm --dir frontend build`, `pnpm --dir frontend test` and `pnpm --dir frontend test:browser --project=ui`. Verification: all four pass, the test count equals the one recorded in 1.1 minus the two cases of 4.1, and `rg -n 'features/(backups|verification)|entities/(backups|verification)/' frontend docs .agents --glob '!frontend/node_modules' --glob '!frontend/dist' --glob '!frontend/browser-results'` finds nothing outside architecture test fixtures.

## Change verification

```sh
pnpm --dir frontend test
pnpm --dir frontend test:browser --project=ui
```
