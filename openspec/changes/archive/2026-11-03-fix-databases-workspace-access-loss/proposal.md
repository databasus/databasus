## Why

When access to the selected workspace disappears, the default databases tab keeps showing its previously loaded databases and leaves the workspace in navigation. Repair this before release so a confirmed access refusal clears the obsolete content and refreshes accessible workspaces without a page reload.

## What Changes

- Connect the databases page to the application's existing workspace-access reconciliation.
- Treat a current HTTP 403 or 404 from any request composing the workspace database list as workspace unavailability, on both initial load and background refresh, even when a sibling request has already failed temporarily. Reconcile at most once per load.
- Clear the database collection and its details before reconciling accessible workspaces. Retain protection against obsolete responses after a workspace switch or session replacement.
- Preserve silent background recovery for network errors and other temporary failures: keep the list and selection, and let the next background refresh retry. Initial temporary failures retain their explicit Retry action.
- Clarify the database-screen specification so its silent-refresh policy excludes confirmed workspace unavailability, and add database-specific workspace-access scenarios.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `frontend-database-screens`: distinguish confirmed workspace unavailability from temporary background list failures.
- `frontend-workspace-screens`: explicitly cover database content and navigation recovery when the selected workspace becomes unavailable.

## Impact

- Frontend: `ApplicationScreen.tsx`, `DatabasesPage.tsx`, `DatabasesComponent.tsx`, and frontend screen tests, primarily `WorkspaceNavigation.test.tsx`.
- Reuse `useApplicationStartup.reconcileUnavailableWorkspace`; no new shared abstraction is needed.
- Follow root `AGENTS.md`, `frontend/AGENTS.md`, and the existing frontend ownership rules in `docs/architecture.md`.
- No HTTP contract, backend, dependency, or architecture changes. No breaking changes to existing API callers, persisted data, or configuration.

### Out of scope

- Adding Retry or error feedback for temporary background list failures.
- Changing permissions, server-side authorization, or resource mutation refusal behavior.
- Detecting access loss before an existing list refresh or page load receives a refusal.
- Refactoring workspace-access handling across all tabs.
