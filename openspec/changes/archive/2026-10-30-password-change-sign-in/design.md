## Context

See proposal.md for the two observed failures. ProfileComponent.tsx:100 ignores the signIn outcome; userApi.ts:49 stores a token only for COMPLETED. App.tsx:26 already listens for auth changes, but AuthPageComponent.tsx:21 initializes authMode to signUp, and its options loader never changes it.

backend/internal/modules/users/internal/signin/session.go:50 encodes passwordCreationTime in whole seconds; authentication_service.go:32 truncates both values to seconds. profile_test.go:126 advances the clock by two seconds before testing invalidation. accounts/profile_service.go:111 replaces the password using the current clock, so even increased timestamp precision would not distinguish two changes at the exact same instant. Pending sign-ins compare password timestamps at microsecond precision in signin/two_factor_verification_service.go:162 and need the same clock-independent identity.

Follow frontend/AGENTS.md, backend/AGENTS.md, the testing skill and the session/transaction rules in docs/architecture.md. No HTTP shape or session-lifetime change is needed.

## Goals / Non-Goals

**Goals:** show the actual sign-in form after a password change, invalidate earlier sessions and pending sign-ins independently of clock resolution, and reuse existing authentication steps.

**Non-Goals:** add another auth UI, change expiry/rate limits, or retain timestamp-only tokens.

## Decisions

### Use a random password version instead of time as authentication identity

Add a non-null UUID password_version to users and a password-version snapshot to pending sign-ins. New accounts receive a random version. Every successful password replacement rotates it in the same database update/transaction as the password hash and its audit event. Cover profile changes, password resets, console resets and invitation activation; a rejected or rolled-back replacement rotates nothing.

Issue a passwordVersion string claim, parse it as a nonzero UUID, and compare it exactly to the user's current version in Authenticate. The token retains ordinary issued-at/expiry times, but passwordCreationTime is no longer an authentication claim. Pending-sign-in reuse, resend and code verification also match the password version; stale pending steps cannot issue a session after any password replacement. Keep password creation time as historical data, not an identity.

Rejected: whole-second or microsecond comparison, because frozen clocks and same-tick replacements can still collide. Rejected: artificially advance creation timestamps, which turns a historical time into an implicit version and complicates concurrency. Rejected: embed a password hash in a token or build another password-derived secret.

### Migrate authentication state without a compatibility path

A transactional migration initializes a fresh password version for each existing user. Existing pending-sign-in records receive nonmatching historical versions and remain unusable; retain the records so hourly send limits still count them. Remove acceptance of the old numeric passwordCreationTime claim. All existing opaque sessions require a new sign-in after upgrade, explicitly marked BREAKING in the proposal.

Backfill one version per row, never a shared constant. Initialize versions at all account-creation sites and rotate in every password-writing repository path. A new session is issued from the persisted current version; a stale concurrent sign-in cannot authorize after the version rotates.

Rejected: accept timestamp-only tokens until expiry, which preserves the same-second vulnerability. Rejected: delete pending history wholesale, which could reset security rate limits.

### Select signIn from authoritative sign-in options

After the auth listener clears local authentication, App renders AuthPageComponent. While sign-in options load, retain the existing loading view. Once loaded, set authMode to signIn when hasAnyUser is true and signUp when false. Explicit sign-up, reset and code-step navigation then proceeds as before; do not run a later effect that overrides the user's selected mode.

This gives the post-password-change transition an actual sign-in form without a persisted redirect flag. It also makes ordinary unauthenticated visits to an existing instance start with sign-in, while a fresh instance still offers its first registration.

Rejected: notify auth listeners without changing the page's initialization; that mounts registration. Rejected: a transient flag stored with the password or a second profile-local form.

### Preserve the notice and failed-change behavior

After a successful password response, clear token and stored user identity, erase form password values and notify auth listeners. Use the existing application-level AntD message provider for localized copy that the password changed and signing in again is required. The provider survives the profile-to-auth transition. Replace the misleading success key in all six dictionaries.

On a failed response, display translateApiError and keep the profile/session. Do not put the password into navigation, persistent state or a toast payload.

Rejected: reload immediately, which can erase the notice. Rejected: logout in finally, which discards a session after validation or network failure.

### Test observable authentication with a frozen clock

Before the repair, reproduce both the profile auto-sign-in and the same-second old-token acceptance. Through the real users API, keep the clock fixed through account creation, password replacement and sign-in, and prove that earlier tokens fail while newly issued tokens work. Repeat replacement at the exact same instant; cover stale pending-code verification and all password-writing flows, preserving failed-change rollback and rate-limit behavior.

An App integration screen test renders the actual profile/auth transition and actual auth forms with typed mocked network responses. Keep this test at the app layer so it imports lower layers without violating frontend/AGENTS.md. Assert the sign-in form is visible after options load, no sign-up form is selected, the notice remains, and no automatic POST /sessions occurs. Profile tests cover submission/failure behavior; AuthPage tests also prove first-user registration remains available. Existing normal sign-in tests cover emailed codes and Turnstile.

Rejected: advancing the clock to make the invalidation test pass or checking only that logout was called.

## Risks / Trade-offs

- Upgrade signs out current users -> explicitly document the breaking token transition; no aliases or fallback.
- A password-writing path omits rotation -> audit every hash write and verify profile/reset/console/invitation paths through their observable entry points.
- Deleting historical pending records changes rate limits -> preserve unusable historical entries and existing retention.
- Auth options finish after the user changes modes -> initialization runs only for the page's initial options result; test existing navigation.
- Auth state unmounts the profile -> keep the notice in the shared provider and assert visible form/message together.

## Migration Plan

Add the transactional UUID version migration without editing released SQL. Update account/pending repositories and every issuance/verification path together, then ship frontend and all six translations. Update docs/glossary.md for Password version / PasswordVersion and the Session definition, and docs/architecture.md for clock-independent password invalidation. Existing sessions and pending steps become unusable; users authenticate normally again. Follow the existing one-way schema policy.


## Reproduction evidence

Before the fix, the required frontend regression command failed: the successful profile change started POST /sessions, left authentication active and never rendered ordinary sign-in. AuthPage rendered the registration password confirmation field on an instance with users. Failed-change and first-registration controls passed.

The users API regression command failed with HTTP 200 instead of 401 for old sessions after a fixed-clock password change and repeated replacements. A correct code from the prior password still returned HTTP 200 instead of 410, and the password step reused the earlier pending sign-in.

## Verification evidence

On 2026-10-30, `mise run backend:test-package ./migrations ./internal/modules/users/... ./cmd/databasus` passed with race checks. Coverage includes fixed-clock repeated replacements, old claims, pending-code reuse/resend/verification, preserved send limits, password-reset and console invalidation, invitation activation, and rollback on audit failure. Migration tests passed for fresh and upgraded databases with distinct user and historical pending versions.

The frontend Change verification command passed all 47 tests across eight files. It covers the actual App transition, retained notice, authentication storage, ordinary emailed-code sign-in with Turnstile, explicit auth navigation, refusals, dictionaries and API clients. `pnpm --dir frontend typecheck`, both module formatters, `mise run lint`, `git diff --check` and strict OpenSpec validation passed.
