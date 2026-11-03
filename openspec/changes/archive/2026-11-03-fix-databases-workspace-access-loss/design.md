## Context

See `proposal.md` for the problem and scope. This design resolves the distinction between confirmed workspace unavailability and temporary refresh failures before implementation.

- `frontend/src/app/ui/ApplicationScreen.tsx:210` mounts `DatabasesPage` without the workspace-unavailable callback that adjacent tabs receive. `frontend/src/pages/databases/ui/DatabasesPage.tsx:5` also lacks that prop.
- `frontend/src/pages/databases/ui/DatabasesComponent.tsx:89` loads databases, workspace backup configurations, and storages together. Its catch at line 105 suppresses every background failure.
- `frontend/src/pages/storages/ui/StoragesComponent.tsx:48` already clears its collection and calls `onUnavailable` for a current `ApiError` with status 403 or 404.
- `frontend/src/app/model/useApplicationStartup.ts:97` removes the unavailable workspace before refreshing navigation. Session and access revisions reject obsolete reconciliation results, and failures reach application recovery.
- `frontend/src/pages/databases/ui/DatabasesComponent.test.tsx:443` already pins silent recovery after a background 500. `openspec/specs/frontend-database-screens/spec.md:148` requires that policy, while `openspec/specs/frontend-workspace-screens/spec.md:31` requires clearing unavailable workspace content.

Root `AGENTS.md` and `frontend/AGENTS.md` govern the change. The approach follows `docs/architecture.md` under Frontend: app owns workspace selection and navigation, pages own collection policy, and obsolete asynchronous work must be invalidated. No architecture document or agent-instruction changes are needed.

## Goals / Non-Goals

**Goals:** keep error classification within the database page, reuse application reconciliation, and make initial and background reads agree on confirmed workspace unavailability.

**Non-Goals:** changing shared transport behavior, introducing a common resource controller, or treating a refused database mutation as proof that workspace access disappeared. Product exclusions are listed in `proposal.md`.

## Decisions

### Pass the existing callback through the page

Add `onUnavailable?: (workspaceId: string) => void` to the page and collection component props, and provide `startup.reconcileUnavailableWorkspace` in `ApplicationScreen`. This matches the existing storage pattern and keeps navigation in app. Reject importing app state into the page because it would reverse layer dependencies; reject a new shared controller because the existing callback covers the need.

### Classify confirmed access loss before applying silent-refresh policy

Observe rejection of each of the three collection requests before passing their promises to `Promise.all`. Each observer first rejects an obsolete session or request revision. For a current `ApiError` with status 403 or 404, mark the load as unavailable, reset the collection with `createEmptyCollection(workspaceId)`, clear local load feedback, and invoke the callback for that request's workspace. Use one flag shared by the observers for this load so several refusals invoke reconciliation at most once. Each observer rethrows its original error to preserve aggregate success and failure behavior.

The aggregate catch handles temporary failures only while the load remains current and has not been marked unavailable. A network error or 500 that rejects the aggregate first must not invalidate the request revision or detach the sibling observers. A later current 403/404 still clears the collection and invokes reconciliation immediately, even if the third request remains pending. If a newer load, workspace switch, unmount, or session replacement has made that refusal obsolete, its observer does nothing. The aggregate success path also checks the load's availability before accepting content; the catch must not restore local feedback after an observer has cleared it.

This removes items, storage mappings, readiness, and the rendered detail selection before workspace reconciliation completes. Application recovery owns any subsequent navigation-refresh failure. Keep existing effect cleanup protections; sibling completions cannot repopulate a rejected collection.

Reject relying on the `Promise.all` catch for access classification because it sees only the first rejection and misses a later access refusal after a temporary failure. Reject waiting for `Promise.allSettled` because a pending sibling would delay clearing confirmed unavailable content. Reject checking access loss only inside the non-silent branch because it misses the reported defect. Reject handling only the database request because the other required workspace reads can establish the same refusal. Reject handling all 403/404 responses globally because a resource-specific refusal need not mean the whole workspace is unavailable.

### Preserve temporary background recovery

Network failures and other errors outside 403/404 keep the accepted list and selection without feedback on background refresh. The next existing interval or visibility refresh retries. Initial failures still set the translated load error and offer Retry.

Reject adding Retry for background network failures in this repair because that changes the separately specified refresh policy. The requested network-error test becomes a regression for keeping the list, retaining workspace navigation, and recovering through a later background refresh; it must not expect Retry.

### Verify through real screens and the HTTP boundary

Start with an app-level regression in `WorkspaceNavigation.test.tsx` that loads visible databases, refuses a background read with 403, and expects content removal and workspace reconciliation. Run it before changing production code and confirm that it fails for the stale-content defect.

Extend coverage with initial and background 403/404 cases for the required collection reads, another-workspace and no-workspace outcomes, failed accessible-workspace refresh with successful Retry, and obsolete refusals after switching workspaces or replacing the session. Use controlled HTTP responses to hold reconciliation pending and prove old list and detail content disappear before it completes. Use the existing screen tests to retain initial temporary-error Retry and silent background network/500 recovery.

Add ordered-response regressions for initial and background loads: let one required request fail with a network error or 500, wait for that failure to be processed, then release a sibling with 403 or 404 while the third request remains pending. Assert that the later refusal still clears content and starts workspace reconciliation without waiting for the third response. Also release multiple access refusals from one load and verify that reconciliation starts at most once. Release every held response explicitly and preserve cleanup on test failure.

Reject mocking the callback or domain APIs as the sole regression because that would not detect missing wiring in `ApplicationScreen` or `DatabasesPage`. Tests replace only HTTP and use the existing renderers, as the testing skill requires.

## Risks / Trade-offs

- A late refusal could remove the wrong workspace -> guard the error before any state change or callback, retain cleanup invalidation, and test obsolete responses.
- An earlier temporary failure could hide a later refusal -> observe each request independently and test temporary-error-first ordering while another request remains pending.
- Several sibling refusals could launch duplicate reconciliation -> share an availability flag within the load and assert one reconciliation request.
- Clearing only visible cards could leave selected details or storage mappings -> reset the complete collection and assert both list and detail removal while navigation refresh is pending.
- A network outage could be mistaken for revoked access -> classify only structured HTTP 403/404; test network and 500 failures without workspace reconciliation.
- A workspace-list refresh can fail after content is removed -> reuse the existing translated application Retry and verify recovery without obsolete content.
- Access loss remains undetected until an existing workspace list load receives a refusal -> retain the current refresh cadence; push updates and additional polling are outside scope.

## Migration Plan

No data migration or API rollout is required. Ship with the frontend bundle after the focused regressions, frontend lint/build, and full local test run pass. Reverting the frontend change restores prior behavior without persisted-data changes.
