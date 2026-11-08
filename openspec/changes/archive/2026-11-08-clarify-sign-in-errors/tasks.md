## 1. Backend refusal reasons

- [x] 1.1 Add regression coverage distinguishing an unknown email, an incorrect password and incomplete registration; observe the behavioral failure before implementing it. Replace the user contract's `invalid_credentials` with the seven codes in the design, regenerate the API, update shared status/title mappings and sign-in sentinels, implement refusal classification and update all existing assertions of the removed code. Keep the Problem body unchanged. Verify with `TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -count=1 ./internal/lib/apierror ./internal/modules/users/...`, using the Fedora preparation in `docs/testing.md`.
- [x] 1.2 Extend users API coverage for Google, GitHub, both providers, disabled and unlinked providers, nil/empty passwords, and accounts with both a password and OAuth. Verify that active/inactive passwordless accounts give equivalent guidance, while deactivation is reported only after a matching password. Prove existing email recovery can set the first password without changing its policy. Verify with `TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -count=1 ./internal/modules/users/...`, using the same Fedora preparation.
- [x] 1.3 Cover challenge and limiter precedence for every refusal category and equivalent refusals with two-factor authentication on/off; assert no session, new pending sign-in or mail is produced. Add a regression proving that an incorrect password after a successful password step leaves the existing pending sign-in unchanged, that the next correct password step reuses it, and that its original code still completes sign-in. Verify with `TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -count=1 ./internal/modules/users/...`, using the same Fedora preparation.

## 2. Translated correction paths

- [x] 2.1 Replace the old error mapping and dictionary key with the new complete messages in all seven locales; add recovery/administrator guidance and the conditional "Set a password" action label. Update API client and translation tests. Verify with `pnpm --dir frontend test src/shared/i18n/dictionaries.test.ts src/shared/i18n/translateApiError.test.ts src/shared/api/apiClient.test.ts`.
- [x] 2.2 Show unknown-email and incorrect-password feedback beside the corresponding controls, and other reasons at form level. Reuse existing provider, sign-up and recovery controls, preserve drafts, clear the refusal on credential edits, and retain authentication-operation guards. Add screen tests for every reason, configured/unavailable recovery, action callbacks, locale switching, a single submitted request and obsolete responses. Verify with `pnpm --dir frontend test src/pages/auth/ui/SignInComponent.test.tsx`.
- [x] 2.3 Extend the existing browser UI authentication scenarios to cover translated field refusals and passwordless guidance at 320px and 200 percent text enlargement, with accessible field descriptions and retained input. Preserve the existing human-verification scenario. Verify with `pnpm --dir frontend test:browser --project=ui`.

## 3. Integration checks

- [x] 3.1 Remove stale claims of uniform sign-in errors from touched production comments and update any other affected live documentation. Check that no user API caller, fixture, dictionary or assertion still depends on `invalid_credentials`, without changing the verification module's unrelated internal logging reason. Format only this change's files, then verify with `mise run backend:lint`, `mise run frontend:lint` and `pnpm --dir frontend build`.
- [x] 3.2 Run the change verification commands below, resolve failures and mandatory implementation-review findings, and update task completion only for work actually verified. Leave the implementation and artifacts ready for a commit; verify integration with `mise run test` and `pnpm --dir frontend test:browser --project=ui`.

## Verification evidence

- `mise run test` passed, including all backend groups and 2,905 frontend tests.
- Backend and frontend lint, the frontend production build and strict OpenSpec validation passed.
- The browser UI command passed all authentication scenarios in the shared working tree; two unrelated WAL screenshot comparisons failed because concurrent outline-tab styling changed the selected tab's color. Those changes and the concurrent profile form edits belong to another task and were left untouched.
- The complete browser UI suite then passed all 334 tests in a temporary source snapshot using the same pinned runner and `--project=ui` argument. Only the concurrent edits to `src/shared/theme/controls.module.css`, `src/shared/theme/mantineTheme.ts` and `src/pages/profile/ui/ProfilePage.tsx` were replaced with their HEAD versions in that snapshot. No screenshot references were changed. Browser run: `62c063e4-fa7c-4553-8bc5-15414bb1e279`.

## Change verification

```sh
mise run test
pnpm --dir frontend test:browser --project=ui
```
