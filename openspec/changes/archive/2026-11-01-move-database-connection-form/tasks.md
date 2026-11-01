## 1. Map coverage and export the parsers

- [x] 1.1 Run `pnpm --dir frontend test` and record the file and test counts. Then create `coverage.md` in this change. It maps every test file under `features/databases` and the restore and page tests that reach the moved forms to their final paths and retained guarantees. Verification: `coverage.md` lists each of those test files.
- [x] 1.2 Rename `ConnectionStringParser` and its test to `PostgresqlConnectionStringParser`. Export the four parsers from `entities/databases/index.ts`, point the forms at the public API, and remove the five `project/private-slice-import` entries from `fsd-baseline.json`. Verification: `pnpm --dir frontend test src/entities/databases src/features/databases` and `mise run frontend:lint`.

## 2. Move the forms

- [x] 2.1 Move `EditDatabaseBaseInfoComponent` and `ChoosePostgresBackupTypeComponent` to `pages/databases/ui`, changing only their import paths. Remove their exports from `features/databases/index.ts`, point the wizard at the new paths, and update their entries in `style-baseline.json` to the new source and owner `pages/databases`. Verification: `pnpm --dir frontend test src/pages/databases` and `mise run frontend:lint`.
- [x] 2.2 Move the remaining files and tests of `features/databases/ui/edit`, including `CreateReadOnlyComponent` and `toDatabaseSettingsTarget`, to `features/database-connection/ui`, changing only their import paths. Create `features/database-connection/index.ts` with the `EditDatabaseSpecificDataComponent` and `CreateReadOnlyComponent` exports, and delete `features/databases` with no forwarding exports. Point the page and `features/restores` at the new public API. In `fsd-baseline.json`, set the restore entry's target to `features/database-connection/index.ts`. In `style-baseline.json`, update the remaining entries to the new source and owner `features/database-connection`. Verification: `pnpm --dir frontend test src/features/database-connection src/features/restores src/pages/databases` and `mise run frontend:lint`.

## 3. Documentation and handover

- [x] 3.1 Update `docs/architecture.md` ("Database pages compose reusable editing") and `frontend/README.md`. In `temp/refactor/frontend-plan.md`, record 5.3 as this change and add 5.4 as `refactor-database-connection-form`, and mark 5.3 done. Complete `coverage.md` and confirm that no test was dropped. Format the changed files and run `mise run frontend:lint`, `pnpm --dir frontend build` and `pnpm --dir frontend test`. Verification: all three commands pass, the test count is at least the one recorded in 1.1, and `rg -n 'features/databases|\bConnectionStringParser\b' frontend docs .agents --glob '!frontend/node_modules' --glob '!frontend/dist'` finds nothing.

## Change verification

```sh
pnpm --dir frontend test
```
