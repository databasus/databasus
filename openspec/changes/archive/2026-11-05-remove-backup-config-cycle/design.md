## Context

See `proposal.md` for the motivation and scope. The current dependencies are visible in these entry points:

- `backend/internal/modules/backups/module.go:181` declares logical and physical service pointers before constructing `config.Service`, then supplies callbacks that dereference those pointers. Logical and physical constructors receive that same config service.
- `backend/internal/modules/backups/internal/config/service.go:273` owns the authorized save transaction. It locks the configuration, re-reads and authorizes the database, selects the backup method, saves it and attaches current cleanup counts before returning.
- `backend/internal/modules/backups/internal/config/service.go:434` combines physical validation, previous-state loading, storage cleanup, persistence, generation changes, request resets and registration of a reaction after commit.
- `backend/internal/modules/backups/internal/config/storage_switch_service.go:31` provides a second save entry point for workspace transfers. Moving only the HTTP save would leave its dependency on the backup services intact.
- `backend/internal/modules/backups/internal/config/storage_change_cleanup_service.go:22` calls the history-removal callbacks and attributes pending file deletion to the database and accepting workspace.
- `backend/internal/modules/backups/internal/config/storage_publication_service.go:9` checks storage selection and generation before publication. It is used by running backups and must remain below their services.

Root `AGENTS.md` and `backend/AGENTS.md` constrain dependency injection, repository ownership and removal of compatibility adapters. `docs/architecture.md`, sections "Modules", "Roles inside a module" and "Storage switches", requires service-owned transactions, repository-owned SQL, atomic cleanup and physical-history-before-database-before-config lock order. This design applies those conventions inside the existing backups module; it introduces no new module or architectural rule.

## Goals / Non-Goals

**Goals:** Make service construction follow one dependency order and separate configuration-save orchestration from the config operations needed by running backups. Preserve both the ordinary save and workspace-transfer paths, including their rollback and concurrency guarantees.

**Non-Goals:** Do not redesign transactions, reduce lock acquisition, change scheduler or cancellation behavior, move unrelated event handlers, or add abstraction layers for future callers.

## Decisions

### 1. Put both save entry points in `configsave`

Construct `config.Service` first, `logical.Service` and `physical.Service` next, and `configsave.Service` last. The new service takes the ready config, logical and physical services through `Deps`, along with the existing transaction, access, storage and clock dependencies it needs.

```text
httpapi ----> configsave <---- transfer
                  |
       +----------+----------+
       v          v          v
    logical    physical    config
       |          |
       +----------+-------> config
```

Move `SaveBackupConfig`, `SwitchStorage` and storage-change cleanup orchestration into `configsave`. The HTTP handler keeps its config dependency for reads and gains the save service for writes. The transfer service keeps its config dependency for locks and reads and calls the save service for a storage switch. Keep configuration requests, publication checks and database/storage event handling in `config`.

Delete `BackupHistoryRemoverFunc`, `PhysicalConfigSavedReactionFunc`, the three callback dependencies and the late-bound service variables after updating their callers. `configsave` calls the backup services directly. Ordinary transaction and after-commit closures remain: they do not close a dependency cycle.

Rejected alternatives: interfaces, setters or method-value adapters still make config depend on services that depend on it. Moving only `SaveBackupConfig` leaves the cycle through `SwitchStorage`. Moving orchestration into `httpapi` would violate the handler/service boundary and duplicate the transfer path.

### 2. Reuse the config repository and domain rules

Keep SQL in the existing config repository and expose the repository and only the methods required by `configsave` through the existing repository bundle. Construct the bundle once and pass it to both services. Do not pass `*gorm.DB` to the save service or duplicate queries in the new package.

Keep defaults, validation and normalization with config models, exposing the operations the save service needs. Reuse config's cleanup decoration for the saved value inside the save transaction; do not replace it with a later config read that could acknowledge a concurrent save instead. Keep locking entry points in `config`.

Keep `HistoryRemoval` and `RemovedBackupHistory` in `config` as shared lower-level contracts. Moving them into `configsave` would make logical and physical services import the orchestration package and recreate a cycle.

Rejected alternatives: a second repository duplicates table ownership and SQL. A generic repository interface or forwarding service adds indirection without an independent implementation. Moving models and publication rules into the save package reverses dependencies needed by backup execution.

### 3. Split physical saves along transaction steps

Use a short physical-save coordinator with private helpers for preparing the config and previous state, applying storage-change cleanup, persisting the config and request changes, and registering the saved-config reaction. Group a phase's values in a named struct when its argument list requires one under the naming rules.

Preserve the existing sequence and conditions:

1. Normalize and validate the submitted settings; under the existing locks, read the current database backup type, validate against it and retain the pending manual backup requests.
2. Check storage ownership using the database workspace read and authorized under locks. Retain last-selected-storage lookup and distinguish an ordinary initial assignment from reassignment with retained history and from a transfer.
3. Invoke the appropriate backup service directly to remove history and collect exact file references. Record scoped deletion requests in the same transaction.
4. Save the configuration and apply the existing generation-increment conditions. For physical reassignment, reset database backup history and physical backup requests under their existing conditions.
5. Register the physical saved-config reaction with `database.AfterCommit`, preserving its logging and failure behavior. The helper at `backend/internal/lib/database/transaction.go:34` defers nested hooks until the outermost transaction commits and discards them on rollback; a workspace transfer must retain that guarantee.

Keep config's storage publication generation check unchanged. Preserve rejection of logical storage changes during an active logical backup or restore, and physical cancellation only after commit. Clearing storage still retains history; a later reassignment still cleans it up.

Rejected alternatives: moving the long method unchanged improves dependencies but leaves unrelated phases intertwined. Running cancellation immediately after an inner save would cancel active work even if a later transfer step rolls back. Replacing the transaction structure or normalizing generation rules would expand this refactor into a behavior change.

### 4. Verify through the existing API tests

Run the backups root package, internal packages and every `physicalapitests` package unchanged. Relevant coverage includes config access and validation, concurrent workspace/storage transfers, cleanup rollback, physical request preservation, disabling backups, storage reassignment, and late FULL/WAL/history publication. Backend formatting and lint verify wiring and dependency rules.

Rejected alternatives: mocks of config or backup services would hide the transaction and cancellation behavior at risk. New tests that assert constructor shape would mirror the implementation. Backup/restore end-to-end suites are unnecessary because this change does not alter the backup execution pipeline.

## Risks / Trade-offs

- A storage switch during transfer could keep calling config's old save path. Mitigation: move both entry points, remove the old callbacks and forwarding entry points, and compile all backups packages.
- Cleanup could gain stale workspace attribution or lose rollback behavior. Mitigation: preserve locked reauthorization, scope selection and the existing outer transaction; run transfer and rollback API tests.
- Cancellation could occur before the outer commit, or a stale writer could regain publication rights. Mitigation: preserve after-commit registration, storage-generation updates and publication checks; run all physical API tests.
- Exporting repository operations increases the internal package API. Mitigation: expose only the operations used by the two services; the enclosing module's Go `internal` boundary still protects them from other modules.

## Migration Plan

No schema, data or HTTP migration is needed. Introduce the minimal shared config operations, move both orchestration entry points and update construction and callers in one implementation change. Update package and service comments to match their responsibilities. `docs/architecture.md` and agent instructions already describe the intended boundaries and need no rule changes. Rollback is a code revert with no data conversion.
