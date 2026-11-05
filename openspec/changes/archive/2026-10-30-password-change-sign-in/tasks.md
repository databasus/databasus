## 1. Reproduce both authentication failures

- [x] 1.1 Add ProfileComponent submission/failure tests, AuthPageComponent form-selection tests and an App-layer integration screen test of the actual profile-to-auth transition with real auth helpers/forms and the typed network. Keep higher-layer imports in the App test. Assert a visible sign-in form, no default sign-up form after options load, retained password-change notice and no automatic POST /sessions, with/without 2FA and on failure. Run `pnpm --dir frontend test src/app/ui/App.test.tsx src/features/users/ui/ProfileComponent.test.tsx src/pages/AuthPageComponent.test.tsx` on the unfixed code and record the incorrect form/session behavior.
- [x] 1.2 Add real users API regressions with the clock unchanged between initial password setting and replacement, multiple replacements at that exact instant, and a pending sign-in from the previous password. Assert old sessions/code steps fail and new credentials can complete sign-in. Run `mise run backend:test-package ./internal/modules/users/...` on unfixed code and record same-second session acceptance.

## 2. Make invalidation independent of time

- [x] 2.1 Add a transactional migration for per-user UUID password versions and pending-sign-in snapshots, backfilling unusable historical pending versions without deleting rate-limit history. Cover fresh/upgrade state and per-user uniqueness with `mise run backend:test-package ./migrations`.
- [x] 2.2 Initialize versions on every account creation path, rotate atomically with every successful password replacement, replace timestamp JWT claims/checks with exact UUID versions, and bind pending reuse/resend/verification to the same version. Remove old claim acceptance. Cover profile changes, password/console resets, invitation activation, exact-same-instant replacements, rollback, stale pending steps and unchanged rate limits through existing entry points. Verify with `mise run backend:test-package ./internal/modules/users/... ./cmd/databasus`.

## 3. Return to the actual ordinary sign-in form

- [x] 3.1 Remove profile auto-sign-in, clear token and stored user identity only after success, notify listeners and preserve the localized notice. Initialize AuthPageComponent from loaded options: signIn for existing users and signUp for an unclaimed instance, without overriding later mode navigation. Replace the obsolete success key in all dictionaries. Verify actual rendered forms and the notice with `pnpm --dir frontend test src/app/ui/App.test.tsx src/features/users/ui/ProfileComponent.test.tsx src/pages/AuthPageComponent.test.tsx src/shared/i18n/dictionaries.test.ts` and `pnpm --dir frontend typecheck`.
- [x] 3.2 Cover the resulting ordinary sign-in path with required emailed code and Turnstile, preserving reset/sign-up/code navigation and failed-password-change behavior. Verify with `pnpm --dir frontend test src/features/users/ui/SignInComponent.test.tsx src/features/users/ui/SignInCodeComponent.test.tsx src/pages/AuthPageComponent.test.tsx`.

## 4. Complete verification and documentation

- [x] 4.1 Update docs/glossary.md for Password version / PasswordVersion and the Session definition, and docs/architecture.md for clock-independent invalidation. Describe the breaking sign-out of timestamp-only sessions and pending steps in the change documentation. Run `mise run backend:format`, `mise run frontend:format`, `mise run lint` and all Change verification commands. Resolve mandatory review findings and leave the change ready to commit.

## Change verification

```sh
mise run backend:test-package ./migrations ./internal/modules/users/... ./cmd/databasus
pnpm --dir frontend test src/app/ui/App.test.tsx src/features/users/ui/ProfileComponent.test.tsx src/pages/AuthPageComponent.test.tsx src/features/users/ui/SignInComponent.test.tsx src/features/users/ui/SignInCodeComponent.test.tsx src/shared/i18n/dictionaries.test.ts src/shared/api/apiClient.test.ts src/shared/api/apiHelper.test.ts
```
