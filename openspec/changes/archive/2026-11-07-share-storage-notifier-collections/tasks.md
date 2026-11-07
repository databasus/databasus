## 1. Shared collection mechanics

- [x] 1.1 Establish the current behavior and identify reusable lifecycle coverage before extraction. Run `pnpm --dir frontend test src/pages/storages/ui/StoragesComponent.test.tsx src/pages/notifiers/ui/NotifiersComponent.test.tsx src/app/ui/WorkspaceResources.test.tsx src/app/ui/ApplicationSession.test.tsx`; retain all existing scenarios and add only missing acceptance cases in subsequent tasks.
- [x] 1.2 Add the domain-independent collection hook and migrate storages first. Keep API calls, access-refusal policy, selection and editor acknowledgment handling in the page. Cover current load failure/retry, stale success/error, overlapping reads, unmount and setup-cleanup-setup through the screen harness; verify with `pnpm --dir frontend test src/pages/storages/ui/StoragesComponent.test.tsx src/app/ui/WorkspaceResources.test.tsx src/app/ui/ApplicationSession.test.tsx`.
- [x] 1.3 Migrate notifiers to the same hook while retaining the five-minute timer in its page. Verify mutation reconciliation, newer draft preservation, current access-refusal clearing, obsolete workspace/session results, polling and timer cleanup, and absence of storage polling using controlled responses and fake time. Run `pnpm --dir frontend test src/pages/storages/ui/StoragesComponent.test.tsx src/pages/notifiers/ui/NotifiersComponent.test.tsx src/app/ui/WorkspaceResources.test.tsx src/app/ui/ApplicationSession.test.tsx`.

## 2. Shared list presentation and search

- [x] 2.1 Add a storage screen regression for case-insensitive local name search and run `pnpm --dir frontend test src/pages/storages/ui/StoragesComponent.test.tsx` to observe its intended failure before implementing search. Add the focused shared list, migrate both adapters and add storage search keys to all seven shipped dictionaries. Verify matching/no-match/clear behavior, four/five-item threshold, shrinking collections, no search requests, independent selection/drafts, language switching and existing add-action permissions/placement with `pnpm --dir frontend test src/pages/storages/ui/StoragesComponent.test.tsx src/pages/notifiers/ui/NotifiersComponent.test.tsx src/shared/i18n/dictionaries.test.ts`.

## 3. Shared transfer interaction

- [x] 3.1 Extract the shared transfer dialog with context-scoped request acceptance and migrate storage first. Keep the one-config usage check, workspace API/filtering and transfer API in the adapter; preserve read retry, selected target after refusal, duplicate-submit protection and close/context invalidation. Verify with `pnpm --dir frontend test src/pages/storages/ui/StorageTransferDialogComponent.test.tsx` using real domain APIs and HTTP-only replacement.
- [x] 3.2 Migrate notifier transfer to the shared dialog while retaining its notifier-filtered database usage check. Cover blocked and unused resources, target choices, successful transfer, read retry, retained target on refusal, duplicate submission, and late success/error after closing or changing context for both adapters. Run `pnpm --dir frontend test src/pages/storages/ui/StorageTransferDialogComponent.test.tsx src/pages/notifiers/ui/NotifierTransferDialogComponent.test.tsx`.

## 4. Integration and documentation

- [x] 4.1 Clarify shared mechanics versus page-owned policy in `docs/architecture.md` and check affected references for consistency. Keep API facades, domain detail/editor contracts, existing preference keys and `ResourceListDetail`'s public contract intact. Verify dependency boundaries and types with `mise run frontend:lint` and `pnpm --dir frontend typecheck`; remove only duplicated code replaced by the two consumers.
- [x] 4.2 Retain or extend the existing frontend browser resource scenarios to check keyboard search, focus restoration, translated feedback and narrow list/dialog presentation. Format touched frontend files with the repository formatter without changing unrelated work; verify `mise run frontend:lint`, `pnpm --dir frontend build`, the affected screen suites and `pnpm --dir frontend test:browser --project=ui`. Run the change verification commands below before mandatory implementation review and resolve blocking findings before handover.

## Change verification

```sh
mise run test
pnpm --dir frontend test:browser --project=ui
```
