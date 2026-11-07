## Why

Storage and notifier lists duplicate request handling and presentation, and their behavior has diverged: only notifiers offer name search. Their transfer dialogs also repeat the same interaction around different usage checks. Sharing these mechanics will let fixes reach both screens while keeping domain-specific behavior easy to change.

## What Changes

- Extract a small, domain-independent collection hook and list presentation alongside the existing shared list/detail and selection primitives. Keep page adapters responsible for domain operations, permissions, selection scope and refresh policy.
- Add client-side name search to storages with the notifier list's existing threshold of five items, matching behavior and translated feedback on both screens.
- Share transfer dialog mechanics while keeping usage checks, workspace loading/filtering and transfer calls in page adapters.
- Preserve notifier polling every five minutes and keep storage loading event-driven.
- Keep cards, detail screens, creation forms and provider behavior owned by their existing domains. Shared parts accept composition and callbacks rather than resource-kind switches or a configurable screen framework.

Out of scope:

- Unifying storage and notifier API facades, changing HTTP operations or backend behavior.
- Merging `StorageComponent`, `NotifierComponent`, their editors or provider settings.
- Changing database transfer, logical/physical backup screens or restore flows.
- Adding storage polling, changing notifier polling cadence, redesigning pages or targeting a mandatory line-count reduction.
- Building generic CRUD, plugin registries or speculative extension points.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `frontend-workspace-screens`: Specify consistent resource name search and retain existing collection refresh behavior through the refactor.

## Impact

The change affects storage and notifier list and transfer page adapters, domain-independent shared hooks/UI, frontend dictionaries and relevant screen tests. API facades and contract-generated types retain their domain ownership. No external callers, schemas, configurations or HTTP APIs break, and no dependencies are added.

Implementation follows root `AGENTS.md` and `frontend/AGENTS.md`. `docs/architecture.md` must clarify that pages retain collection policy while delegating domain-independent request state to shared mechanics; shared code cannot import entity APIs. All shipped dictionaries receive the storage search strings.
