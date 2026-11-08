## Why

Frontend screens repeat loading flags, failure handling, refresh scheduling and protection against late responses even where their request lifecycles are the same. The owner has approved replacing that duplication across the frontend in one change, while preserving frequent server refreshes and avoiding reuse of responses between visits.

## What Changes

- Add `@tanstack/react-query` v5 and update the frontend package manifest and lockfile as part of the approved approach.
- Migrate ordinary screen reads and explicit server actions throughout the frontend, including resource pages, database settings and creation, healthcheck, verification settings, account and instance settings, workspace management, and application startup. This is a complete migration of suitable request lifecycles, not a pilot.
- Treat every result as immediately stale and remove unused results without a retention period. Reopening a screen starts a fresh read. Keep data needed by an open screen in memory; do not persist responses or introduce prefetching.
- Preserve existing polling intervals, visibility behavior, explicit retries and distinct first-load/background-error policies. Successful actions acknowledge server responses and refresh affected active data without allowing earlier reads to overwrite them.
- Propagate cancellation through migrated reads and complete-list traversal to HTTP, and isolate request state by session generation, workspace, resource and filters.
- Retain specialized pagination and WAL controllers, editor drafts and authentication-attempt ownership. Simplify ordinary requests around these owners without replacing their domain rules.
- Delete superseded request boilerplate and helpers with no remaining consumers. Measure the net production-source line change and production bundle gzip change against the pre-implementation baseline.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `frontend-api-requests`: Require fresh reads on new screen visits, cancellation of obsolete ordinary reads, refresh after successful actions, and request scheduling that preserves each screen's refresh policy.
- `frontend-session`: Require session-owned request results and pending completion effects to be discarded on session transitions, including signing back in as the same account.

## Impact

- Governed by root `AGENTS.md` and `frontend/AGENTS.md`.
- Affects `frontend/src/app`, request infrastructure and screen-test providers in `frontend/src/shared`, domain API facades in `frontend/src/entities` and `frontend/src/pages`, and request consumers in pages and features.
- Adds one production dependency and changes `frontend/package.json` and `frontend/pnpm-lock.yaml`. The owner accepted this dependency-bearing design during exploration.
- Updates `docs/architecture.md` for request ownership and lifetime, and `docs/testing.md` for isolated request providers and timer cleanup. Existing frontend behavior specs remain regression constraints unless the deltas above explicitly add a requirement.
- No HTTP operation, generated schema, persisted application setting or external caller changes are planned. Internal helpers that become unused are removed outright; no compatibility aliases are introduced.

### Out of scope

- Replacing `PagedListController`, `usePagedList`, `usePolledPagedList`, WAL controllers, editor draft hooks or authentication intent/OAuth state machines.
- Introducing `useInfiniteQuery`, response persistence, offline request queues, automatic mutation retries, optimistic writes or new refresh intervals.
- Changing backend operations, adding push/long-polling endpoints, redesigning UI, or reorganizing unrelated frontend slices.
- Shipping only a subset, leaving eligible screens for a later migration, or making rollout conditional on an estimated line saving.
