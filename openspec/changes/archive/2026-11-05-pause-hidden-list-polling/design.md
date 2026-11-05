## Context

See `proposal.md` for motivation and scope. A design is useful here because preserving unchanged state interacts with cursor invalidation and concurrent page requests.

- `frontend/src/shared/hooks/usePolledPagedList.ts:23` installs an unconditional interval. `frontend/src/shared/hooks/usePagedList.ts:46` owns initial loading and lifecycle activation; its stable refresh callback is defined at line 69.
- `frontend/src/shared/lib/PagedListController.ts:78` prevents overlapping refreshes and rejects refreshes before the first load succeeds. Lines 88-96 merge the response, invalidate incompatible pending pagination and publish state. Lines 161-163 always create and publish a new state object.
- `frontend/src/shared/lib/pagedList.ts:29` aligns the refreshed first page with loaded rows and preserves a usable tail. The controller currently uses the merge result's identity to distinguish replacement by the first page from tail preservation.
- The five consumers are logical backups (`frontend/src/pages/databases/ui/logical-backups/LogicalBackupsComponent.tsx:69`), physical backups (`frontend/src/pages/databases/ui/physical-backups/PhysicalBackupsComponent.tsx:84`), restores (`frontend/src/pages/databases/ui/restores/RestoresComponent.tsx:59`), verifications (`frontend/src/pages/databases/ui/verification/VerificationsComponent.tsx:76`) and verification agents (`frontend/src/pages/instance-settings/ui/verification-agents/VerificationAgentsComponent.tsx:123`). Physical-backup metadata contains total usage at line 71 of its component.
- `frontend/src/pages/databases/ui/DatabasesComponent.tsx:207` already guards automatic refresh by document visibility and subscribes to visibility changes.

Root `AGENTS.md` and `frontend/AGENTS.md` constrain the implementation, especially effect dependencies, cleanup and regression testing. `docs/architecture.md`, under Frontend / State and effects, assigns generic pagination machinery to shared and requires cleanup to invalidate late work. This design keeps those boundaries.

## Goals / Non-Goals

**Goals:** Put browser visibility in the polling lifecycle, keep response acceptance in the existing controller and eliminate refresh publications whose effective state is unchanged.

**Non-Goals:** Introduce no polling configuration, shared application-wide scheduler, generic caching system or cancellation protocol. Keep explicit operations outside the visibility gate.

## Decisions

### 1. Gate automatic refresh in the existing hook

Use one callback for interval ticks and `visibilitychange`. It calls the stable `refreshFirstPage` callback only when `document.visibilityState !== 'hidden'`. Register and remove the listener alongside the interval in the existing effect, declaring the callback and interval dependencies. Do not call it on effect setup: `usePagedList` already starts the initial load.

The controller's readiness and in-flight guards handle visibility changes during initial loading, after a first-load error or during a pending refresh. A visibility event starts a refresh without waiting for a tick; subsequent ticks keep the existing schedule. Requests already sent may finish under the existing generation guards.

Rejected alternatives: gating the controller would also block explicit refreshes, and gating every screen would duplicate one policy across five consumers. Clearing and restarting the timer on every visibility change adds lifecycle branches without reducing network requests beyond a visibility guard. No focus listener is needed because focus and browser-tab visibility represent different conditions.

### 2. Compare the effective refresh result before publishing it

Keep the existing merge and calculate cursor invalidation from its original result before replacing any reference. Compare the merged list with the current list by value, including ordered row content, the effective continuation cursor and metadata. Use a small private structural comparison of the JSON-compatible response values: short-circuit identical references, compare arrays in order and compare object keys and values independently of property order. Reused tail rows can short-circuit by identity. Keep this logic with the controller unless the merge itself needs a tightly scoped adjustment.

If the list is equal, retain its reference. Still perform required request invalidation and loading-state changes. If the effective list and refresh-related loading state are both unchanged, retain the full state reference and do not invoke `onStateChange`. Limit this suppression to refresh publication; initial loads, explicit reloads and action-driven item replacement retain their current lifecycle.

Rejected alternatives: comparing only row keys hides status changes; comparing only items misses metadata and cursor changes; comparing raw first pages ignores the loaded tail and current merges. Comparing serialized JSON makes property order affect equality and allocates serialized copies. Adding a comparison dependency or a server revision field is unnecessary for these response values.

### 3. Prove behavior at the existing test surfaces

Use the real polling hook through `PolledPagedListScreen` and its typed HTTP boundary, fake timers for polling and controlled responses for overlap. Controller tests assert reference retention and the absence of `onStateChange`, plus positive cases for changed rows, order, metadata, cursor and loading state. Keep the existing merge and lifecycle regressions, extending them where the new behavior introduces a case.

Rejected alternatives: a render snapshot does not prove that requests stop, and a same-reference assertion alone does not prove that the controller avoided publishing a new state. A new end-to-end or browser-layout suite would add setup without improving these request and state assertions.

## Risks / Trade-offs

- Structural comparison costs work proportional to the visited values. Identity checks avoid inspecting unchanged tail objects, but this change does not claim a measured rendering speedup or solve growth of the loaded list.
- An early equality return could skip cursor invalidation or leave a loading indicator wrong. Compute invalidation first and test compatible and incompatible pending pagination before suppressing publication.
- Adding a visibility listener can leave duplicate work after cleanup. Test unmount, reset, changed options, interval changes and Strict Mode setup-cleanup-setup, with no requests or state publications from disposed lifecycles.
- Hidden tabs may still receive a response to a request already sent, and explicit operations or unrelated polling may send requests. The visibility guarantee covers new automatic refreshes from this hook.

## Migration Plan

Ship through the normal frontend build. No data migration, API generation or caller changes are required. Reverting the frontend change restores the previous polling behavior.
