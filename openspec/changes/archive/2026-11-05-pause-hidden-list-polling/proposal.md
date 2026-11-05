## Why

Polled lists keep requesting their first page while the browser tab is hidden. Every successful refresh also publishes new list state even when the result is unchanged, causing unnecessary renders; the rendering cost has not been measured.

## What Changes

- Skip automatic list refreshes while the browser tab is hidden and refresh immediately when it becomes visible, without duplicating an in-flight refresh.
- Apply this policy to all five consumers of the shared polling hook: logical backups, physical backups, restores, verifications and verification agents. Keep their existing intervals while visible.
- Keep the existing list and state references and publish no state update when merging a refreshed page changes neither row content or order, continuation cursor, metadata nor loading state.
- Preserve initial and explicit loads, refresh error handling, pagination alignment and request lifecycle guards.

### Out of Scope

- Virtualization, row eviction, page-size changes and performance benchmarking.
- Other polling, including verification-agent availability and local clocks.
- Changes to WAL monitoring or database-list polling.
- HTTP contracts, backend behavior, visible copy, configuration and new dependencies.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `frontend-list-loading`: Define visibility-aware automatic refresh and preservation of unchanged list results across the five polled lists.
- `backup-list-paging`: Clarify that the existing refresh interval applies while the browser tab is visible, subject to the common list-loading policy.

## Impact

The implementation belongs in `frontend/src/shared/hooks/usePolledPagedList.ts` and `frontend/src/shared/lib/PagedListController.ts`, with pagination merge changes only if needed to retain equal results. The shared hook's five consumers inherit the policy without separate screen implementations. Regression coverage belongs in the existing polling-hook, controller and pagination tests.

The change follows root `AGENTS.md` and `frontend/AGENTS.md`. It preserves the generic pagination ownership established in `docs/architecture.md`. It changes no caller signatures, schemas, configuration or HTTP APIs.
