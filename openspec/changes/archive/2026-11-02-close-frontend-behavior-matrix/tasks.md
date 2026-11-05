## 1. Finish the audit

- [x] 1.1 Run `pnpm --dir frontend test` and record the file and test counts in a new `coverage.md` of this change. Copy the 26 items from `design.md` into it as a table to complete. Verification: `coverage.md` states both numbers and lists 26 items.
- [x] 1.2 Read the test bodies behind the matrix rows the audit checked partly: profile and settings, storage and notifier, database connections, database composition, backup config, logical, physical and WAL, verification, and i18n and theme. Add any further gap as a new row. Verification: each row of the matrix in `temp/refactor/frontend-analysis.md` appears in `coverage.md` with its owning tests.
- [x] 1.3 List test titles removed since the migration began with `git log -p 5424a6c4^..HEAD -- 'frontend/**/*.test.*' 'frontend/**/*.spec.*'`, filtered to removed `it(`, `test(`, `it.each(` and `test.each(` lines and the title line of each table. Compare parameterized titles as templates. Commit `5424a6c4` established the foundation, and the range includes `sweep-code-spacing-and-test-waits`, whose archive arrived in `9831c495`. For every title no current test file holds, record its new place or reason, with `407b31e1` as Decision 4 describes. A title whose behavior no current test owns becomes a new row of the table. Verification: each such title from that output appears in `coverage.md` with a new place, a reason or a row.

## 2. Authentication and settings

- [x] 2.1 Add items 1 to 3. Reproduce the shared-workspace selection and backup filter defects first, then clear those keys in `publishSession` as Decision 6 describes. Verification: `pnpm --dir frontend test src/pages/auth-callback src/pages/auth src/app/ui/ApplicationSession.test.tsx src/shared/auth src/shared/hooks src/pages/databases/ui/logical-backups src/pages/databases/ui/physical-backups`.
- [x] 2.2 Add item 4. Verification: `pnpm --dir frontend test src/pages/instance-settings/ui/MailServer.test.tsx`.

## 3. Storages and notifiers

- [x] 3.1 Add item 5 as one table, as Decision 3 describes. Verification: `pnpm --dir frontend test src/features/edit-storage`.
- [x] 3.2 Add item 6 as one table. Verification: `pnpm --dir frontend test src/features/edit-notifier`.
- [x] 3.3 Add item 7. Verification: `pnpm --dir frontend test src/pages/storages src/pages/notifiers`.

## 4. Databases and backup configuration

- [x] 4.1 Add item 8. Verification: `pnpm --dir frontend test src/pages/databases/ui/DatabasesComponent.test.tsx`.
- [x] 4.2 Add items 9 to 11 for logical and physical configuration. Verification: `pnpm --dir frontend test src/pages/databases/ui/backup-config src/pages/databases/ui/DatabaseConfigComponent.test.tsx`.

## 5. Backups and WAL monitoring

- [x] 5.1 Add items 12 to 14. Verification: `pnpm --dir frontend test src/pages/databases/ui/logical-backups`.
- [x] 5.2 Add the screen test of item 16. Verification: `pnpm --dir frontend test src/pages/databases/ui/physical-backups`.
- [x] 5.3 Add items 17 and 18. Verification: `pnpm --dir frontend test src/pages/databases/ui/wal-stream-monitoring`.

## 6. Verification agents and shared failures

- [x] 6.1 Add items 21 and 22. Verification: `pnpm --dir frontend test src/pages/instance-settings`.
- [x] 6.2 Add items 25 and 26. Verification: `pnpm --dir frontend test src/app/ui/WorkspaceResources.test.tsx`.

## 7. Browser scenarios

- [x] 7.1 Strengthen `browser/scenarios/auth.spec.ts`, `browser/e2e/signIn.spec.ts`, `browser/scenarios/physicalBackups.spec.ts` and `browser/scenarios/wal.spec.ts` for items 2, 3, 15, 19 and 20, with no new reference image. Verification: `pnpm --dir frontend test:browser --project=ui` and `pnpm --dir frontend test:browser --project=e2e`.
- [x] 7.2 Add `browser/scenarios/browserRegion.spec.ts` for items 23 and 24. Verification: `pnpm --dir frontend test:browser --project=ui`.
- [x] 7.3 Add a test for each row tasks 1.2 and 1.3 added, at the layer Decision 1 assigns. Verification: `pnpm --dir frontend test` on the owning paths named in `coverage.md`, and the matching `test:browser` project where a row needs a browser scenario.

## 8. Documents and acceptance

- [x] 8.1 Update the "Workspace resource screens" section of `frontend/README.md` to the final structure. Search for old paths, commands and migration wording outside archived changes, and bring any finding in line. Verification: `rg -n "features/(databases|backups|restores|verification)|entities/(backups|verification|intervals)([^-a-z]|$)|toUnsavedDatabase|apiHelper|fsd-baseline|style-baseline|ToastHelper" docs .agents/skills frontend/AGENTS.md frontend/README.md frontend/src frontend/lint frontend/tooling AGENTS.md --glob '!**/upstream/**'` finds nothing; `rg -n 'physical|verification' frontend/README.md` shows the section naming those screens' owners; `rg -n -i 'migrat(ed|ion) (baseline|residue|checks|milestone)|legacy|transitional|during the migration' docs/architecture.md docs/testing.md docs/frontend-design.md docs/frontend-localization.md .agents/skills/feature-sliced-design/SKILL.md .agents/skills/frontend-design/SKILL.md .agents/skills/frontend-localization/SKILL.md .agents/skills/testing/SKILL.md frontend/AGENTS.md frontend/README.md` finds nothing about the frontend migration, and a backend hit is acceptable.
- [x] 8.2 Check the final state. Verification: `rg -n 'uses:' .github/workflows | rg -v '@[0-9a-f]{40}'` finds nothing, `rg -n 'test-frontend-browser-e2e|test-frontend-browser-ui' .github/workflows/ci-release.yml` shows both jobs among the release dependencies, and `find frontend -maxdepth 2 -name '*baseline*' -not -path '*/node_modules/*'` finds nothing.
- [x] 8.3 Format the changed files and run `mise run frontend:lint`, `pnpm --dir frontend build`, `pnpm --dir frontend test`, `pnpm --dir frontend test:browser --project=ui` and `pnpm --dir frontend test:browser --project=e2e`. Record each command's result, test count and duration in `coverage.md`, with any limitation. Mark item 8 done in `temp/refactor/frontend-plan.md`. Verification: all five pass, every row of the table in `coverage.md` names its test, and `rg -n '\[x\] \*\*8\.' temp/refactor/frontend-plan.md` finds the line.

## Change verification

```sh
pnpm --dir frontend test
pnpm --dir frontend test:browser --project=ui
pnpm --dir frontend test:browser --project=e2e
```
