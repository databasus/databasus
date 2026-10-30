## Why

The refactoring is complete, but its progress checklist and several comments still describe the migration period. An unused session middleware remains exposed, and the architecture document overstates the production test-dependency restriction despite two deliberate rclone exceptions.

## What Changes

- Mark milestones 13 and 13c complete in the temporary refactoring plan after checking the archived legacy-teardown tasks.
- Remove GetSessionMiddleware from the users module interface, implementation and access adapter, retaining contract-based access enforcement.
- Remove or rewrite comments about deleted old test helpers and migration bridges in users, databases and backups.
- Document the existing Files.com and Proton Drive test-library import exceptions without removing rclone backends.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

None. This is documentation and dead-code cleanup; `skip_specs: true` records that no product requirement changes.

## Impact

Constrained by root `AGENTS.md`, `backend/AGENTS.md` and the testing skill. Affects the temporary refactoring plan, users module public interface and access code, stale module comments, and `docs/architecture.md`. The public HTTP contract and access rules remain unchanged. The existing archlint exceptions and `backend-code-checks` specification already agree with the intended dependency policy.

### Out of scope

- Removing rclone, Files.com or Proton Drive support, or strengthening the production dependency rule.
- Changing authentication behavior or deleting module methods merely because their comments are stale.
- Fixing R1, R2, F1, F2 or F3 in this change.
- Rewriting the full temporary refactoring analysis or publishing release migration instructions.
