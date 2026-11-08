## 1. Confirm the baseline and move healthcheck

- [x] 1.1 Inspect the working tree, worktrees and locally available branches for overlapping edits; record an ownership inventory using the design's mapping before moving files. Leave other contributors' changes untouched. Verify the current baseline with `mise run frontend:lint`, `pnpm --dir frontend typecheck` and `pnpm --dir frontend test src/pages/databases`.
- [x] 1.2 Move the healthcheck UI and its existing tests into `pages/databases/healthcheck/ui`, update all consumers, and retain existing entity ownership. Verify with `pnpm --dir frontend test src/pages/databases/healthcheck` and `pnpm --dir frontend typecheck`.

## 2. Move verification and WAL monitoring

- [x] 2.1 Move verification UI, model, API facades and existing tests into `pages/databases/verification`, preserving declarations and runtime bodies. Update all consumers and verify with `pnpm --dir frontend test src/pages/databases/verification` and `pnpm --dir frontend typecheck`.
- [x] 2.2 Move all WAL UI, model, supporting logic, fixtures, existing tests and `physicalWalStreamStatisticsApi` into `pages/databases/wal-monitoring`. Preserve its existing call to the physical backup facade and update the tab composition imports. Verify with `pnpm --dir frontend test src/pages/databases/wal-monitoring` and `pnpm --dir frontend typecheck`.

## 3. Move configuration, backups and restores

- [x] 3.1 Move backup configuration UI, `BackupStorageSelection`, configuration-specific logical helpers, physical draft/conversion/validation/retention/notification units and their existing tests into `pages/databases/backup-config`. Update settings, wizard and backup-list consumers without duplicating entity contracts. Verify with `pnpm --dir frontend test src/pages/databases/backup-config` and `pnpm --dir frontend typecheck`.
- [x] 3.2 Move the remaining physical backup model, API, supporting logic, list and physical restore UI with their existing tests into `pages/databases/physical-backups`. Update WAL's physical API import. Verify with `pnpm --dir frontend test src/pages/databases/physical-backups` and `pnpm --dir frontend typecheck`.
- [x] 3.3 Move logical backup UI, its API facade, root backup/list/status/download models, verification badge styles, filter helper and existing tests into `pages/databases/logical-backups`. Verify with `pnpm --dir frontend test src/pages/databases/logical-backups` and `pnpm --dir frontend typecheck`.
- [x] 3.4 Move logical restore UI, restore target draft/conversion units and existing tests into `pages/databases/restores`, preserving physical restore ownership and entity-owned operations. Verify with `pnpm --dir frontend test src/pages/databases/restores` and `pnpm --dir frontend typecheck`.

## 4. Move settings and the creation wizard

- [x] 4.1 Move settings composition, editors, shared settings forms, display components, transfer dialog, notifier comparison helper and existing tests into `pages/databases/settings` as mapped in the design. Update wizard consumers to import reusable settings forms directly. Verify with `pnpm --dir frontend test src/pages/databases/settings` and `pnpm --dir frontend typecheck`.
- [x] 4.2 Move wizard composition, engine/backup-type choice, notifier step, first-backup operation and existing tests into `pages/databases/create-wizard`. Keep configuration steps in `backup-config`, shared page controls in root `ui`, `formatBackupSize` in root `lib`, and the public page export unchanged. Verify with `pnpm --dir frontend test src/pages/databases/create-wizard` and `pnpm --dir frontend typecheck`.
- [x] 4.3 Audit the completed relocation for stale module paths, accidental compatibility re-exports, changed runtime statements and changed test assertions. Preserve source-fixture and CSS resource path semantics. Verify the intermediate layout with `mise run frontend:lint`, `pnpm --dir frontend typecheck` and `pnpm --dir frontend test src/pages/databases` before consolidating declarations.

## 5. Consolidate related declarations

- [x] 5.1 Combine related WAL generated response aliases in `walStreamContract.ts`, move label mappings beside their corresponding enums, and place tiny view aliases with their owning unit, including `MonitoringTextParam` with `MonitoringText`. Preserve identifiers, schema references, values and type-only imports; delete redundant files and update consumers. Verify with `pnpm --dir frontend test src/pages/databases/wal-monitoring` and `pnpm --dir frontend typecheck`.
- [x] 5.2 Apply the same ownership-based consolidation to logical backups, physical backups, verification and configuration, using focused domain-named contracts only where multiple related aliases justify them. Preserve independent model units and existing test assertions, and inspect dependencies for new runtime cycles. Verify with `mise run frontend:lint`, `pnpm --dir frontend typecheck` and `pnpm --dir frontend test src/pages/databases`.

## 6. Document and verify the complete change

- [x] 6.1 Document the database page's local areas, composition modules and unchanged slice boundary in `docs/architecture.md`. Search repository references to moved modules and update affected descriptions or literal references without editing agent instructions or introducing tooling exceptions. Verify with `mise run frontend:lint` and `pnpm --dir frontend build`.
- [x] 6.2 Format only files changed by this refactor, preserve others' edits, and inspect the diff for structural-only changes, unchanged scenario assertions, obsolete paths and unchanged public exports. Run `mise run frontend:lint`, `pnpm --dir frontend typecheck`, `pnpm --dir frontend build` and `pnpm --dir frontend test src/pages/databases`; record the commands and outcomes for mandatory implementation review. Complete the change only after required checks and review accept the working tree, before any commit.

## Change verification

```sh
mise run test
```
