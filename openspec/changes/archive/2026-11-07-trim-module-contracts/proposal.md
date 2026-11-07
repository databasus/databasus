## Why

Unused aliases and constants in module contracts obscure which declarations consumers actually need. Removing them and four unused backup forwarding methods makes those boundaries easier to read without changing application behavior.

## What Changes

- Audit the declarations in every `backend/internal/modules/*/module.go`, including references from tests and from the contract itself.
- Delete unused aliases and constants after confirming that no caller or retained declaration needs them. Exploration identified 71 candidates across backups, users, workspaces, notifiers and storages; this count is an audit starting point, not a deletion quota.
- Delete `RequestFullBackup`, `ClearFullBackupRequest`, `RequestIncrementalBackup` and `ClearIncrementalBackupRequest` from the backups module implementation. Keep the service methods that the backup scheduler and request handling use.
- Preserve the module interfaces, types needed by their signatures, module implementation packages and working forwarding methods.

### Out of Scope

- Removing or relocating `internal/module`, replacing module interfaces, or exposing internal services.
- Adding a modules getter to `App` or changing its test-support methods.
- Deleting domain operations, changing startup or event ordering, or reorganizing background jobs.
- Changing HTTP contracts, database schemas, frontend behavior, architecture checks or agent instructions.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

None. This is an implementation-only cleanup with `skip_specs: true`; existing behavioral requirements remain applicable.

## Impact

- Expected implementation changes: the root contracts of backups, users, workspaces, notifiers and storages, and `backend/internal/modules/backups/internal/module/module.go`.
- The remaining module contracts are audited; declarations needed by their consumers or signatures remain.
- `docs/glossary.md` names the owning internal types where it referred to deleted contract aliases. Product terms and definitions remain unchanged.
- The change follows the root `AGENTS.md`, `backend/AGENTS.md`, the module boundaries in `docs/architecture.md`, and the testing skill.
- No existing callers, HTTP APIs, persisted data or configuration formats change. Unused declarations are deleted outright without compatibility aliases.
- Existing tests provide behavioral coverage. Verification includes backend and test-package compilation, backend formatting and lint, the architecture check, and the routine test run. No new tests or dependencies are needed.
