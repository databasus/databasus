## Context

See `proposal.md` for motivation and scope. This design is constrained by root `AGENTS.md`, `frontend/AGENTS.md`, and `docs/architecture.md:330-377`: dependencies point downward, domain operations remain with their owners, and pages own collection policy.

Current code establishes the useful boundaries:

- `frontend/src/pages/storages/ui/StoragesComponent.tsx:22-100` and `frontend/src/pages/notifiers/ui/NotifiersComponent.tsx:22-106` repeat collection state, latest-read acceptance, mutation reconciliation and selection wiring. Shared `useScopedSelection` already owns preference handling; `shared/ui/ResourceListDetail.tsx:13` already owns responsive list/detail layout.
- `frontend/src/pages/notifiers/ui/NotifiersComponent.tsx:99-106` polls every five minutes; `:121-156` filters names locally and displays search at five items. `frontend/src/pages/storages/ui/StoragesComponent.tsx:98-100` loads initially without polling; `:125-138` renders cards without search.
- `frontend/src/pages/storages/ui/StorageTransferDialogComponent.tsx:39-54` checks one backup config before loading target workspaces. `frontend/src/pages/notifiers/ui/NotifierTransferDialogComponent.tsx:36-50` checks databases filtered by notifier before loading targets. Their remaining pending/error/submit presentation is substantially alike.
- API facades already reuse shared transport and `collectCursorPages`; storage deletion has its own cleanup policy (`frontend/src/entities/storages/api/storageApi.ts:59-66`).

## Goals / Non-Goals

**Goals:** Give both pages one implementation of collection request mechanics, searchable list presentation and transfer interaction. Keep domain differences directly editable in their page adapters. Preserve existing detail/editor contracts, preference keys, mutation ordering and stale-response protection.

**Non-Goals:** See the proposal's exclusions. In particular, there is no target percentage of shared code: a few lines of page composition are preferable to switches, casts or extension machinery introduced solely to eliminate repetition.

## Decisions

### 1. Share collection mechanics; leave policy in pages

Introduce `useResourceCollection` under `shared/hooks` for typed items, initial readiness, read errors, loading, reload and local collection updates. It accepts a caller-provided read callback and only requires item identity for operations that use it. It owns request revision/session acceptance and invalidates pending work on cleanup. Separate initial readiness from background loading so refresh does not replace the page with a spinner.

Pages choose when to load and refresh. They retain workspace checks on accepted mutations, API error classification, unavailable-workspace callbacks, selection updates and the distinction between persisted and fully acknowledged editor completion. Invoke page error handlers only for a current request, so a late refusal cannot evict the new workspace. Provide explicit collection clearing so a current access refusal can remove old items. Reuse `useScopedSelection`, `useSessionGuard` and request revisions rather than inventing parallel identity systems. Effect cleanup must permit setup-cleanup-setup, and async handlers must read current options using the existing latest-ref convention where needed.

Keep the notifier timer in its page; the storage adapter has no timer. Do not put polling options or HTTP status policy into the hook. Update the resource-page paragraph of `docs/architecture.md` during implementation to distinguish shared request state from page-owned collection policy.

Rejected alternatives: a universal resource controller would absorb domain behavior and make later exceptions harder; extracting only a fetch helper would leave the duplicated loading/error/reconciliation state in both pages; leaving the duplication would keep fixes dependent on two edits.

### 2. Compose list presentation around domain-owned cards and editors

Add a focused `ResourceCollectionList` under `shared/ui` alongside `ResourceListDetail`. Its inputs are typed items, identity/name accessors, a card-render callback, a caller-supplied add action and description, and translated search labels/feedback. It owns local filtering and the shared five-item search threshold. Query state stays in each page and is passed as a controlled value so it survives the list unmounting while a mobile detail is open. Keep active search clearable when a collection shrinks. Filtering never writes to selection or drops the underlying items.

Each page continues composing `ResourceListDetail`, its own detail component, creation modal and editor callbacks. The page decides whether the add action exists and owns its ref/focus restoration. Existing add-button placement remains based on the unfiltered collection count. Translation stays reactive: the adapter translates from its own domain keys during render, including no-results text with interpolation, rather than passing a resource kind or concatenated key prefix.

This is the concrete extension boundary: a storage-only notice can be rendered next to the shared list in the storage page; storage-only card information goes in `StorageCardComponent`; new editor behavior stays in `EditStorageComponent`. None needs a change to notifier code or a resource-kind branch in shared code. Add a new slot only when an actual caller needs insertion inside shared markup; do not prebuild an extension API.

Rejected alternatives: a component that owns the entire screen would need knowledge of domain editors and permissions; a configuration schema or plugin registry would obscure ordinary JSX composition; changing `ResourceListDetail` to handle loading and domain collections would burden unrelated consumers.

### 3. Keep transfer rules in thin adapters and share the interaction

Retain `StorageTransferDialogComponent` and `NotifierTransferDialogComponent` as page adapters. They supply a context identity, a data-loading callback, a transfer callback, translated content and close/completion handlers to a focused shared transfer dialog. The data-loading callback returns a discriminated result: blocked, or available targets expressed as plain label/value options. Shared code does not import storage, notifier, database, backup-config or workspace entities.

Storage keeps the one-config usage check; notifier keeps the filtered database check. Each adapter loads workspaces only when the resource is unused and excludes the source workspace. Existing backend permissions and conflicts remain authoritative. The shared dialog owns read retry, selected target, transfer pending/error state, duplicate-submit prevention and current-context response acceptance. Retry reloads reads only; transfer failures retain the chosen target and require an explicit resubmission. Closing or changing context invalidates pending work; a new context resets old target/error state. Closing restores focus to a connected invoker even when the parent immediately unmounts the dialog. Controls wrap long labels and placeholders at 320 CSS pixels with 200 percent text enlargement. Error objects remain untranslated until render.

A later storage-only eligibility rule belongs in the storage loading callback, and a changed storage transfer request belongs in its transfer callback. If an interaction eventually becomes materially different, its page can compose the lower-level controls directly; it need not force another mode into the shared dialog.

Rejected alternatives: passing `resourceType` would move business branches into shared; moving workspace API calls into shared would violate dependency boundaries; merging database transfer into this dialog would require a different workflow and additional flags.

### 4. Keep API facades and supported language coverage

Leave `storageApi` and `notifierApi` unchanged. Retain domain-owned generated types and existing backend operations. Add storage search keys to every shipped dictionary (`en`, `ru`, `es`, `pt`, `zh`, `fr`, `id`) and keep notifier keys in place. Preserve live language switching, literal query rendering, focus, narrow layouts and theme roles.

Rejected alternatives: generic CRUD construction saves explicit request wrappers at the cost of weaker endpoint typing and awkward cleanup differences; renaming all keys into a generic resource namespace creates unrelated translation churn.

### 5. Verify through existing screen surfaces

Start the storage search addition with a screen regression that fails on current code, then implement it. Retain the existing list and transfer suites and their real providers and domain APIs; replace only HTTP. Add matching search cases for both domains, query clearing, the five-item boundary, shrinking collections, selection/draft preservation and language switching. Advance fake time to verify notifier polling and its cleanup, and absence of storage polling.

Exercise overlapping requests, unmount/session/workspace invalidation, current access refusal and setup-cleanup-setup through the existing screen/app harness. Transfer coverage must distinguish each usage-check request, unused targets, read retry, successful transfer, retained target on refusal, duplicate submit and obsolete completion. Use existing browser resource scenarios for keyboard/focus and narrow presentation checks. Run frontend formatting, lint, build, relevant screen/dictionary tests, the production browser UI suite and `mise run test` before implementation review.

Rejected alternatives: mocking the new hook would hide adapter wiring and request ordering; tests mirroring internal state would make future implementation changes expensive without establishing user behavior.

## Risks / Trade-offs

- [A shared abstraction grows flags for each domain] -> Keep domain imports, timers, policy and editors in adapters; accept small duplication and add only extension points needed by these two real callers.
- [Extraction changes async acceptance or persisted-creation behavior] -> Preserve existing callback contracts and prove late success/error, draft and selection behavior through screen tests before removing duplicated code.
- [Search hides the selected card or the collection shrinks] -> Keep selection independent of filtered cards and leave an active query clearable.
- [Shared markup accidentally changes other list/detail pages] -> Add focused components without changing `ResourceListDetail`'s public contract or unrelated callers.

## Migration Plan

No data migration, API generation or backend deployment change is needed. First extract collection mechanics and migrate the two pages without changing refresh policy, then share list/search presentation, then transfer mechanics. Remove only duplicated implementation replaced by those consumers. Update architecture documentation in the same implementation change. Deploy as a normal frontend build; rollback uses the prior frontend build and needs no data conversion.
