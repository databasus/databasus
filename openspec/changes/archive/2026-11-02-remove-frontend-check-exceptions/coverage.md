# Coverage

## Baseline

`pnpm --dir frontend test` before this change: 171 test files, 2007 tests, all passing.

## Request and firing changes

- `pages/databases/ui/healthcheck/HealthcheckAttemptsComponent.tsx`: polling now starts after a successful load instead of right after the config answers, and it does not start after a failed load, so a tick cannot replace a shown failure. A failed refresh still leaves polling running. Tests: "shows the reason without the old attempts when the load after a period change fails, and a later tick does not replace it", "keeps refreshing after a failed refresh".
- `pages/databases/ui/healthcheck/HealthcheckAttemptsComponent.tsx`: the load effect no longer depends on `t` or on `onVisibilityChange`. A language change no longer restarts the load and its polling, and a re-render of `DatabaseComponent`, which passes a new `onVisibilityChange` each time, no longer repeats the config and attempts requests. `ShowHealthcheckConfigComponent` also stopped reloading on a language change, because the reason translates at render. Tests: "repeats no request when the parent passes a new callback or the language changes", "requests nothing new and keeps the configuration when the language changes".

## Removed tests

- `tooling/styleBaseline.test.ts`, "tracks exact counts and rejects stale exceptions when source occurrences change": it tested `compareStyleBaseline`, which is deleted. The process cases in `tooling/styleChecks.test.ts` now prove that one finding fails `lint:styles`.
- `frontendArchitecture.test.ts`, "tracks exact occurrences and stale entries when the baseline changes": it tested `compareArchitectureBaseline`, which is deleted. Process cases in the same file now prove that one finding fails `lint:fsd`.
- `pages/databases/ui/DatabaseConfigComponent.test.tsx`, "shows no notifier or verification error after the screen unmounts": it only asserted that `window.alert` stayed unused, which the code no longer calls. The held-response cases in `ShowDatabaseNotifiersComponent.test.tsx` and `ShowVerificationConfigComponent.test.tsx` cover these late answers.
- `pages/databases/ui/DatabaseConfigComponent.test.tsx`, "ignores pending configuration errors after the screen unmounts" and "shows no failure of a physical backup configuration after the screen unmounts": they too asserted only that `window.alert` stayed unused. "ignores a late backup configuration failure from an earlier load of the same database" still covers a late answer of that load.

## Effect repairs

No `exhaustive-deps` finding needed a firing change, so effects fire as before apart from the healthcheck changes above. Functions an effect only calls went into `useEffectEvent`. A timer that outlives its effect now runs the current handler instead of the one from the render that started it, for example in `NotifiersComponent`, `StorageSwitchHistoryComponent` and `WorkspaceMembersComponent`.

## Changed tests

- `pages/databases/ui/DatabaseComponent.test.tsx`: tab assertions moved from the `bg-surface` class to `aria-selected`, name and action queries moved to roles and translated names, and a keyboard tab change was added.
- `pages/databases/ui/DatabasesComponent.test.tsx`: button queries moved to the new names, the refused copy asserts the reason on the screen instead of `window.alert`, the transfer submit is queried inside its dialog, and a block of late-answer scenarios was added.
- `pages/databases/ui/DatabaseConfigComponent.test.tsx`: name queries follow the new markup, and action feedback scenarios were added.
- `pages/databases/ui/healthcheck/EditHealthcheckConfigComponent.test.tsx`: the refused save asserts the reason on the screen instead of `window.alert`. Load, retry, late-answer and label scenarios were added.
- `pages/databases/ui/healthcheck/HealthcheckAttemptsComponent.test.tsx`: failure, refresh, period and late-answer scenarios were added.
- `pages/instance-settings/ui/verification-agents/VerificationAgentsComponent.test.tsx`: the agent name is queried by role and name.
- `src/shared/lib/test/frontendArchitecture.test.ts` and `tooling/styleChecks.test.ts`: process cases and the layer rule cases were added, as "Removed tests" describes.
- `browser/scenarios/databaseResources.spec.ts`: dark-theme notice and focus indicator scenarios were added.

New test files: `shared/hooks/useLatestRef.test.tsx`, `shared/hooks/useRequestRevision.test.tsx`, `tooling/frontendLintRules.test.ts`, and screen tests for `ShowHealthcheckConfigComponent`, `ShowDatabaseNotifiersComponent`, `EditDatabaseBaseInfoComponent`, `ChoosePostgresBackupTypeComponent`, `CreateLimitedUserComponent`, the logical and physical backup filters, `EditAzureBlobStorageComponent` and `AuthNavbarComponent`. `AppShell.test.tsx` gained a logo link case.

## Final counts

`pnpm --dir frontend test` after this change: 182 test files, 2122 tests, all passing. Two baseline tests and three tests without a remaining assertion were removed, as "Removed tests" lists. `pnpm --dir frontend test:browser --project=ui`: 109 passed.
