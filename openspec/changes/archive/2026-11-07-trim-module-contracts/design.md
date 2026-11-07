## Context

See `proposal.md` for motivation and scope. The root `AGENTS.md` and `backend/AGENTS.md` govern this change. In particular, domain code stays under its module's `internal/`, consumers use the root contract, and application wiring belongs to `internal/app`. The existing module layout and interfaces remain as documented in `docs/architecture.md:57` and `docs/architecture.md:91`.

The current contracts mix declarations used by consumers with redundant re-exports. Examples include backup configuration and retention aliases at `backend/internal/modules/backups/module.go:49`, unused user role and status aliases at `backend/internal/modules/users/module.go:34`, and storage provider constants at `backend/internal/modules/storages/module.go:71`.

An initial search accounting for qualified imports, the module's own tests and references inside the contract found these deletion candidates:

| Module | Candidate aliases and constants | Starting location |
| --- | ---: | --- |
| users | 4 | `backend/internal/modules/users/module.go:34` |
| notifiers | 8 | `backend/internal/modules/notifiers/module.go:57` |
| storages | 8 | `backend/internal/modules/storages/module.go:71` |
| workspaces | 10 | `backend/internal/modules/workspaces/module.go:31` |
| backups | 41 | `backend/internal/modules/backups/module.go:49` |

These 71 candidates require reference verification during implementation. The original review's count is not an acceptance criterion. The referenced `temp/refactor/full-review.md` is absent from this checkout.

Four additional forwarding methods have no references and are absent from the public interface: `RequestFullBackup`, `ClearFullBackupRequest`, `RequestIncrementalBackup` and `ClearIncrementalBackupRequest` at `backend/internal/modules/backups/internal/module/module.go:127`. Their corresponding configuration service methods are used by backup request handling and scheduling, for example at `backend/internal/modules/backups/internal/physical/backup_request_service.go:77` and `backend/internal/modules/backups/internal/physical/backuping/scheduler.go:198`.

## Goals / Non-Goals

**Goals:**

- Determine removability from resolved references and the remaining contract's needs, including test consumers.
- Keep the cleanup limited to declarations and the four unused forwarding methods, with any imports or comments made redundant by those deletions removed as well.
- Preserve every live operation and the existing dependency graph.

**Non-Goals:**

The exclusions in `proposal.md` apply. In particular, this design introduces no new representation of a module and no rule or automated check for unused exports.

## Decisions

### Verify references before deleting each declaration

Use text search to enumerate declarations and candidate consumers, then `gopls references` at the declaration to verify references across production and test code. Inspect references in retained signatures and other declarations as well. Keep anything a real consumer or retained declaration needs; do not rewrite a caller or signature to make a declaration removable. If a candidate has gained a caller, retain it and report the adjusted removal count.

The expected small groups are `Role`, `Status`, `StatusActive` and `StatusInactive` in users; the seven `Type*` constants and `CategoryAll` in notifiers; the eight `Type*` constants in storages; and `Role`, the four `Role*` constants and the five workspace-management `Action*` constants in workspaces. The backup group consists of unused configuration, encryption and retention re-exports, plus unused logical backup status declarations. Verify the individual members rather than deleting an entire block by its subject.

Keep types that signatures need even when consumers never name them directly. For example, `DatabaseCreatedHandler` is used by `SubscribeToDatabaseCreated` at `backend/internal/modules/databases/module.go:141`, and the logical backup handler aliases occur in subscription signatures at `backend/internal/modules/backups/module.go:157`. Likewise, retain `PhysicalBackup`, used as a return type at `backend/internal/modules/backups/module.go:145`.

Rejected alternatives: deleting every name without an external selector would remove necessary signature types; relying on regular expressions alone would miss import aliases and indirect uses. Compilation provides a second check after deletion.

### Retain the implementation packages and working forwarding methods

Keep the existing call path:

```text
Consumer --> Module interface --> internal/module --> Service
```

This preserves the interfaces required by `docs/architecture.md:91` and the module boundaries in `backend/AGENTS.md`. Implementation packages also hold orchestration: backups starts restores, logical backups and physical backups in order at `backend/internal/modules/backups/internal/module/module.go:91`, and coordinates deletion at line 156. Those operations remain intact.

Rejected alternatives: moving implementations into the root contract would enlarge that file without eliminating forwarding; exposing concrete services would change the dependency model. Removing interfaces would also affect the existing `stuckStorage` test double, which embeds `storages.Module` at `backend/internal/modules/backups/internal/physical/backuping/stream_test.go:193`.

Delete only the four confirmed unused backup forwarding methods and their unused `time` import. Their service implementations remain available to current internal consumers.

### Use compilation and existing tests to verify the cleanup

Compile all backend packages and all test packages, including packages outside the routine test selection. Then run backend formatting, backend lint and the routine test command. Backend lint already runs `archlint`, as shown in `mise-tasks/backend/lint:15`; no new lint rule is needed.

Rejected alternatives: adding tests for the presence or absence of exported names would mirror the declarations rather than test behavior; adding a regression scenario would imply a behavior change this refactor does not make. The testing skill's requirement for a reproducing test applies to bug fixes, while this change removes unused declarations.

## Risks / Trade-offs

- A search reports an alias as unused even though a retained signature or test needs it -> verify resolved references, compile all test packages, and preserve required declarations without replacing their callers.
- Removing a similarly named domain method would change behavior -> limit method deletion to the four declarations on the implementation's `Module` receiver; preserve configuration service methods and their consumers.
- A broad cleanup could grow into a module redesign -> keep `internal/module`, module interfaces and `App` methods outside the deletion set.
- Compilation alone cannot verify runtime ordering -> leave orchestration untouched and run existing tests. No startup, event or background-job code moves.

## Migration Plan

No data migration, API regeneration, dependency update or configuration change is required. The implementation consists of source deletions followed by verification. A rollback restores those source declarations through the normal version-control workflow.

Architecture documentation and behavioral specs need no structural or behavioral update. Five entries in `docs/glossary.md` refer to deleted aliases for workspace roles, backup configuration and notification triggers; update those references to the owning `access` and `config` types without changing the terms or definitions. The change artifacts record the scope, decisions and verification tasks; `.openspec.yaml` explicitly skips delta specs because behavior is unchanged.
