## 1. Visibility-aware polling

- [x] 1.1 Add regression scenarios to `usePolledPagedList.test.tsx` named `stops polling while the page is hidden` and `refreshes at once when the page becomes visible`. Use the real screen fixture, typed HTTP responses and fake timers; advance to a point between ticks for the return event. Run `pnpm --dir frontend test src/shared/hooks/usePolledPagedList.test.tsx` on the unfixed code and verify both fail because of the missing visibility policy.
- [x] 1.2 Gate automatic interval requests by browser visibility and add immediate refresh on visibility changes in `usePolledPagedList`. Clean up both the timer and listener. Extend regression coverage for mounting hidden, initial load pending or failed, explicit refresh/reload while hidden, a pending refresh across hide/show, reset with stale success/error, latest options, interval changes, unmount with pending work and Strict Mode setup-cleanup-setup. Restore visibility and timer overrides after each test. Verify `pnpm --dir frontend test src/shared/hooks/usePolledPagedList.test.tsx src/shared/hooks/usePagedList.test.tsx` passes with no duplicate requests or work from disposed lifecycles.

## 2. Unchanged refresh results

- [x] 2.1 Add `keeps the list reference when the refreshed page is unchanged` to `PagedListController.test.ts`, using separately allocated equal response objects. Assert the list and full state retain their references and the refresh invokes no state-change callback. Cover a first page alone and a preserved loaded tail, including an empty complete list. Run `pnpm --dir frontend test src/shared/lib/PagedListController.test.ts` before the fix and verify the new assertions fail because refresh always publishes new state.
- [x] 2.2 Preserve equal merged list references and suppress unchanged refresh publication in `PagedListController`, using structural value comparison without a new dependency. Compute cursor invalidation before retaining any old reference. Add positive coverage for changed row values, reordered rows, nested values, metadata-only changes, effective cursor changes and loading-state changes; equal objects with different property order must remain unchanged. Verify `pnpm --dir frontend test src/shared/lib/PagedListController.test.ts src/shared/lib/pagedList.test.ts` passes, including both completion orders for overlapping refresh and next-page requests, compatible cursor retention, incompatible cursor invalidation and authoritative empty/complete pages.

## 3. Integration and checks

- [x] 3.1 Verify shared behavior and all five consumers with `pnpm --dir frontend test src/shared/hooks/usePolledPagedList.test.tsx src/shared/hooks/usePagedList.test.tsx src/shared/lib/PagedListController.test.ts src/shared/lib/pagedList.test.ts src/pages/databases/ui/logical-backups/LogicalBackupsComponent.test.tsx src/pages/databases/ui/physical-backups/PhysicalBackupsComponent.test.tsx src/pages/databases/ui/restores/RestoresComponent.test.tsx src/pages/databases/ui/verification/VerificationsComponent.test.tsx src/pages/instance-settings/ui/verification-agents/VerificationAgentsComponent.test.tsx`. Keep unrelated polling and all visible intervals unchanged.
- [x] 3.2 Format only frontend files changed by this implementation with the repository formatter, then verify `mise run frontend:lint` and `pnpm --dir frontend build`. Run the change verification below before mandatory implementation review.

## Verification evidence

- Before the visibility fix, the polling-hook test file failed both new scenarios: hidden polling sent three extra requests, and returning to the tab waited for the next tick.
- Before the controller fix, all four new unchanged-result scenarios failed their reference assertions.
- The targeted integration command in task 3.1 passed all 212 tests across nine files. The final polling-hook tests, including exact listener cleanup checks, passed all 14 scenarios.
- The initial `mise run test` completed successfully for both backend suites and the frontend suite; the frontend portion passed all 2528 tests across 187 files.
- `mise run frontend:lint` and `pnpm --dir frontend build` passed. The production build reports runtime-script and chunk-size warnings.

- After adding direct controller coverage for a pending refresh settling after deactivation, the four affected test files passed all 78 tests. `mise run frontend:lint` and `pnpm --dir frontend build` passed again. The two new controller scenarios assert state references, `onStateChange` and error callbacks directly; the screen unmount scenarios cover request and listener cleanup.

## Change verification

```sh
mise run test
```
