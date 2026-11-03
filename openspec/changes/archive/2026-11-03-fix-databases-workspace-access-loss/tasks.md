## 1. Reproduce the stale database content

- [x] 1.1 Add an app-level test named `clears the databases and refreshes workspaces when a refresh answers 403` in `WorkspaceNavigation.test.tsx`, using real screens and controlled HTTP responses. Load a database and its detail, trigger a visibility refresh, return 403, and hold the accessible-workspace refresh pending while asserting the old content is removed. Run `pnpm --dir frontend exec vitest run src/app/ui/WorkspaceNavigation.test.tsx -t 'clears the databases and refreshes workspaces when a refresh answers 403'` before production edits and record that it fails because content remains or reconciliation never starts.

## 2. Reconcile confirmed workspace unavailability

- [x] 2.1 Pass the existing application reconciliation callback through `DatabasesPage` into `DatabasesComponent`. Observe each required request's rejection before aggregation: reject obsolete responses, then reset the complete collection, clear local feedback, and invoke reconciliation at most once per load for a current 403/404, including silent loads. Keep sibling observers active after an aggregate temporary failure; prevent the aggregate handlers from accepting content or restoring feedback after unavailability. Preserve temporary-error behavior and cleanup. Verify the regression from task 1.1 passes with the same command and run `pnpm --dir frontend typecheck`.
- [x] 2.2 Extend `WorkspaceNavigation.test.tsx` to cover 403 and 404 on initial and background loads of the databases, workspace backup configurations, and storages. Verify content removal before reconciliation finishes, removal from workspace navigation, selection of another accessible workspace, and the permitted empty state when none remains. Run `pnpm --dir frontend test src/app/ui/WorkspaceNavigation.test.tsx` and require all cases to pass.
- [x] 2.3 Add database-specific regressions for failed accessible-workspace refresh followed by successful application Retry, a late list refusal after switching workspaces, a late refusal after session replacement, and an older refusal after a newer list refresh succeeds. Verify old content never returns and obsolete responses do not trigger reconciliation or replace current selection. Run `pnpm --dir frontend test src/app/ui/WorkspaceNavigation.test.tsx src/app/ui/ApplicationSession.test.tsx` and require all cases to pass.
- [x] 2.4 Add controlled-response regressions for initial and background loads where one required collection request first fails with a network error or 500, that failure is processed, and a sibling later returns 403 or 404 while the third request remains pending. Verify content is cleared and workspace reconciliation starts before the third response, with no local Retry left for the unavailable workspace. Cover multiple sibling refusals and require at most one reconciliation per load. Release all held responses explicitly, including on failure. Run `pnpm --dir frontend test src/app/ui/WorkspaceNavigation.test.tsx` and require all cases to pass.

## 3. Preserve temporary-failure recovery and finish verification

- [x] 3.1 Add a background network-error regression in `DatabasesComponent.test.tsx` that retains cards and selection without feedback or Retry and recovers on a later background refresh. Retain coverage for background 500 and initial temporary-error Retry, and verify network/500 background errors do not refresh workspace navigation in the app-level tests. Run `pnpm --dir frontend test src/pages/databases/ui/DatabasesComponent.test.tsx src/app/ui/WorkspaceNavigation.test.tsx` and require all cases to pass.
- [x] 3.2 Format only the frontend files changed by this task with `pnpm --dir frontend exec oxfmt <changed-frontend-file-paths-relative-to-frontend>`, then run `mise run frontend:lint` and `pnpm --dir frontend build`. Require both checks to pass before implementation review. Run the full local test command in Change verification and retain its result for review.

## Verification evidence

- The focused 403 regression failed on the unfixed code because the database card remained visible, then passed after the repair. Run with `pnpm --dir frontend exec vitest run src/app/ui/WorkspaceNavigation.test.tsx -t 'clears the databases and refreshes workspaces when a refresh answers 403'`.
- Focused database/navigation tests passed, including ordered sibling failures, duplicate refusals, and temporary network recovery. The two additional stale-refresh cases also passed with `pnpm --dir frontend exec vitest run src/app/ui/WorkspaceNavigation.test.tsx -t 'keeps a newer database list when an older refresh answers HTTP'`.
- Changed frontend files were formatted with oxfmt. `mise run frontend:lint` and `pnpm --dir frontend build` passed.
- `mise run test` passed: both backend package groups, 183 frontend test files with 2415 tests, and both upgrade-test packages. The full run included every new regression.

## Change verification

```sh
mise run test
```
