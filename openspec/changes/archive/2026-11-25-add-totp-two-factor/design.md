## Context

See proposal.md for the motivation. The current second factor is built into the `users` module (paths below are under `backend/internal/modules/users/internal/` unless they start with `backend/`):

- The instance requirement is one boolean, `Settings.IsTwoFactorAuthRequired` (`settings/settings.go:17`), stored in `users_settings.is_two_factor_auth_required` (`backend/migrations/20260920215925_add_two_factor_auth.sql:4`).
- The password step decides between a session and an emailed code in `SignIn` (`signin/password_sign_in_service.go:66`). The emailed code lives in a pending sign-in row (`signin/pending_sign_in.go:14`) that stores a bcrypt hash of the code, the password version, the expiry, the used flag and the failed-attempt count. Concurrent password steps for one user are serialized by an advisory lock (`signin/pending_sign_in_repository.go:32`), and the hourly cap counts every row created in the last hour (`signin/two_factor_service.go:122`).
- The verification claims one of five attempts before checking the code and spends the row in the transaction that issues the session (`signin/two_factor_verification_service.go:53`).
- Master-key export and recovery apply ask `users` whether the instance requires two-factor sign-in (`backend/internal/modules/users/module.go:120`, `backend/internal/modules/instancebackups/internal/keyexport/service.go:168`, `backend/internal/modules/instancebackups/internal/recovery/service.go:248`) and then verify an emailed identity code bound to the session and purpose (`identityproof/service.go:158`). Inside `identityproof`, `canEmail` (`identityproof/service.go:240`) is the only place that checks the requirement.
- The lost-instance state cleanup after a recovery rotates password versions and deletes pending sign-ins and codes (`instancerecovery/state_repository.go:66`). It also reads the settings row column by column (`instancerecovery/state_repository.go:56`).
- The console flags are parsed and validated before any database work in `backend/cmd/databasus/options.go` (with `ErrEmailMissing` in `backend/cmd/databasus/errors.go`, tested in `backend/cmd/databasus/binary_test.go`); the selected command runs in `backend/internal/app/console.go`, where `--disable-2fa` calls `DisableTwoFactorAuth` (`backend/internal/app/console.go:94`). Its audit event has no author (`settings/service.go:185`).
- Rate limits use `backend/internal/lib/ratelimit`, an in-memory counter of attempts (`backend/internal/lib/ratelimit/memory_counter.go:36`). It counts every attempt, cannot count only failures, and starts empty after a restart.
- Stored secrets are encrypted with the master key through `crypto.FieldEncryptor` (`backend/internal/lib/crypto/field_encryptor.go:17`); the users module already receives the master key through its `Deps` (`backend/internal/modules/users/module.go:156`).
- Every signed-in user reads the instance settings (`getInstanceSettings` in `backend/api/openapi.yaml`), so the profile can learn the required method without a new field.

The approach is constrained by `docs/architecture.md` (module boundaries: other modules reach `users` only through `module.go`; shared code without domain knowledge goes into `internal/lib`; Data and Migrations, including invariants in the schema) and by `backend/AGENTS.md` (Deps with `exhaustruct`, repositories through `database.Conn`, one-column updates, errors in `errors.go`, the clock from Deps, audit inside the transaction, enums as `TEXT` with structural constraints kept). Frontend work follows `frontend/AGENTS.md` and the frontend-localization, frontend-design and feature-sliced-design skills; documentation follows `website/AGENTS.md` and `assets/readme/AGENTS.md`; the contract follows the api-design skill.

## Goals / Non-Goals

**Goals:**

- One place, inside `users`, decides which second factor an account presents, and every consumer (sign-in, profile, identity proof) asks it.
- TOTP verification is constant-time, replay-safe and serialized per account, so the per-account limit and the replay rule hold under concurrent requests.
- The emailed code flow keeps its current behavior and tests; it only stops being the default when the account has an app.

**Non-Goals:**

- No general multi-factor framework with pluggable methods. Two methods exist, and the code names them directly.
- No changes to the session token format: a session issued after an app code looks the same as any other.

## Decisions

### 1. The requirement becomes one enum: `OFF`, `EMAIL`, `AUTHENTICATOR_APP`

`users_settings.two_factor_requirement TEXT NOT NULL DEFAULT 'OFF'` replaces the boolean, and the API field `twoFactorRequirement` replaces `isTwoFactorAuthRequired` in `InstanceSettings` and `InstanceSettingsUpdate`. The interface still draws a switch with a method choice under it and maps both to this one value: switch off is `OFF`.

Alternatives considered:

- Keep the boolean and add a `twoFactorMethod` field. This leaves a state, "off with method app", that means nothing and must be ignored everywhere; two fields also make the "changing to EMAIL" precondition check harder to state. Rejected.
- Keep the boolean and add the method as an optional field so the API does not break. The compatibility policy in `docs/architecture.md` (HTTP API, Compatibility) allows breaking the user API when the frontend changes in the same change, and the root `AGENTS.md` forbids unrequested shims. Rejected.

`users.Module.IsTwoFactorAuthRequired` stays as a predicate that answers `requirement != OFF`, because master-key export and recovery only need to know whether a confirmation is required at all; the method of the confirmation is decided per account (decision 7).

### 2. The algorithm goes to `lib/totp`; the domain goes to `users/internal/authenticator`

`backend/internal/lib/totp` holds what has no domain knowledge: generating a secret, computing and checking RFC 6238 codes for a given time, and building the `otpauth://` address. `users/internal/authenticator` holds the domain: the repositories of the new tables, the serialized code check with replay protection and the per-account limit, recovery codes, and the operations of the profile, the administrator reset and the console reset. `signin` and `identityproof` call `authenticator` to check a code; they never read its tables.

Alternatives considered:

- Everything inside `users/internal/authenticator`. The algorithm carries no tables or rules, and `backend/AGENTS.md` (Where code goes) sends such code to `internal/lib`. Rejected.
- The domain inside `signin` (mixes profile management into the sign-in package and makes `identityproof` depend on `signin`) or inside `accounts` (the account package would gain encryption and code checks it has nothing to do with). Rejected.

### 3. TOTP over the standard library; the QR code is drawn by the frontend

RFC 6238 with HMAC-SHA1, six digits and a 30-second step is about thirty lines over `crypto/hmac`, `crypto/sha1` and `encoding/binary`, checked against the RFC 6238 test vectors. Code comparison uses `crypto/subtle`. The backend returns the base32 secret and the `otpauth://totp/Databasus:<email>?secret=…&issuer=Databasus&algorithm=SHA1&digits=6&period=30` address; the frontend draws the QR code from the address with a small QR library added to `frontend/package.json`, chosen and pinned as `docs/how-to-upgrade-deps.md` requires.

Alternatives considered:

- `github.com/pquerna/otp`. It covers the same thirty lines plus HOTP, key generation options and PNG QR rendering we do not need, and adds a dependency to the security-sensitive path. Rejected; the root `AGENTS.md` prefers the standard library when it is enough.
- Rendering the QR code on the backend as a PNG. It needs an image library on the server and a second way to transport the secret. Rejected.

### 4. Data model

```
users_authenticator_apps          one connected app per user
  id                     UUID PK
  user_id                UUID NOT NULL UNIQUE  → users(id) ON DELETE CASCADE
  encrypted_secret       TEXT NOT NULL
  last_used_step         BIGINT NOT NULL DEFAULT 0
  connected_at           TIMESTAMPTZ NOT NULL

users_authenticator_setups        at most one unconfirmed setup per user
  id                     UUID PK
  user_id                UUID NOT NULL UNIQUE  → users(id) ON DELETE CASCADE
  encrypted_secret       TEXT NOT NULL
  failed_attempt_count   INT NOT NULL DEFAULT 0
  expires_at             TIMESTAMPTZ NOT NULL
  created_at             TIMESTAMPTZ NOT NULL

users_recovery_codes
  id                     UUID PK
  authenticator_app_id   UUID NOT NULL → users_authenticator_apps(id) ON DELETE CASCADE
  hashed_code            TEXT NOT NULL
  used_at                TIMESTAMPTZ NULL

users_second_factor_failures      wrong app or recovery codes, kept one hour
  id                     UUID PK
  user_id                UUID NOT NULL → users(id) ON DELETE CASCADE
  failed_at              TIMESTAMPTZ NOT NULL

users_pending_sign_ins            gains
  kind                   TEXT NOT NULL DEFAULT 'EMAIL_CODE'
                         EMAIL_CODE | AUTHENTICATOR_CODE | AUTHENTICATOR_SETUP
  authenticator_app_id   UUID NULL → users_authenticator_apps(id) ON DELETE CASCADE
  authenticator_setup_id UUID NULL → users_authenticator_setups(id) ON DELETE CASCADE
  hashed_code            becomes NULL-able
  CHECK (num_nonnulls(hashed_code, authenticator_app_id, authenticator_setup_id) = 1)
```

The `CHECK` keeps the invariant in the schema, as `docs/architecture.md` (Data) requires for a row that references one of several things: an emailed-code row has a hash, an app-code row has its app, a setup row has its setup. Existing rows all have `hashed_code` and satisfy it. The `kind` names use the API's word `AUTHENTICATOR`, so code and contract say the same thing.

The connected app and the unconfirmed setup are separate tables because a replacement keeps the old app working until the new one is confirmed: both exist at once. Confirming a setup deletes the old app row (its recovery codes and `AUTHENTICATOR_CODE` pending sign-ins go with it through the cascade), inserts the new app with a new ID and new recovery codes, and deletes the setup, in one transaction. A reset deletes the app row, which cascades the same way. Binding pending sign-ins to the app ID is what makes "a reset or replacement makes waiting sign-ins unusable" hold without extra code.

The setup row has one owner at a time: whoever started it last. A profile start and a password step that answers `AUTHENTICATOR_SETUP_REQUIRED` both replace the account's unconfirmed setup with a new one and a new secret; replacing it cascades to the pending sign-in that waited for the old setup. A setup started by the password step lives thirty minutes, and its `AUTHENTICATOR_SETUP` pending sign-in gets the same `expires_at`. The setup is never reused across password steps, so its secret appears in exactly one response (decision 8).

The hourly cap on emailed codes counts only `kind = 'EMAIL_CODE'` rows, so app sign-ins never spend mail allowance. Reuse of a live pending sign-in on a repeated password step looks only for the kind the method choice picks now, and never for `AUTHENTICATOR_SETUP`. The retention job in `users/internal/module/jobs.go` already deletes pending sign-ins older than an hour of every kind; it additionally deletes failure rows older than an hour and expired setups.

Alternatives considered:

- A `confirmed_at` column on a single app table instead of a setups table. It cannot represent a replacement in progress without a second row per user and a partial unique index on confirmed rows; two tables say the same thing plainly. Rejected.
- Storing recovery codes as a JSON array on the app row. Spending one code then rewrites the whole array, and the one-column-per-update rule of `backend/AGENTS.md` is harder to keep. Rejected.
- A `kind`-specific `CHECK` such as `(kind = 'EMAIL_CODE') = (hashed_code IS NOT NULL)` for each kind. Three constraints that say what one `num_nonnulls` plus the application's kind mapping already says. Rejected.

### 5. Secrets are encrypted with the master key; recovery codes are hashed with SHA-256

The TOTP secret must be read back to compute codes, so it is encrypted, not hashed, through `crypto.FieldEncryptor`. A Databasus recovery installs the source key together with the snapshot, so the restored apps keep working.

Recovery codes have 50 bits of randomness (ten base32 characters, shown as `xxxxx-xxxxx`), so a fast hash cannot be brute-forced offline in useful time, and SHA-256 lets a check compare all ten codes without the ten bcrypt rounds (about 0.7 s) a slow hash would cost on every sign-in. The emailed six-digit codes keep bcrypt, because a six-digit space needs the slow hash.

Alternative considered: bcrypt for recovery codes. Same safety for this entropy, ten times slower per check, and a timing difference between the first and the last code. Rejected.

### 6. One serialized check for every app or recovery code, with a failure table

Every app or recovery code check, at sign-in, in the profile or in a confirmation, runs in one transaction under an advisory lock on the user (`users_second_factor:<user id>`):

1. Count `users_second_factor_failures` for the user in the last hour; at ten, refuse with 429 `too_many_wrong_codes` without checking.
2. Normalize the input. Six digits are an app code: compute the codes for steps `now-1`, `now`, `now+1`, compare each in constant time, and accept the highest matching step only if it is greater than `last_used_step`; then update `last_used_step` with `WHERE last_used_step < ?`. Otherwise treat it as a recovery code: hash it, find an unused row of the current app and set `used_at` with `WHERE used_at IS NULL`.
3. On failure insert a failure row; the caller then counts its own attempt (the pending sign-in's five).

Setup confirmation does not go through this check: it compares against the setup secret, counts against the setup's own five attempts, and inserts no failure row, as the `authenticator-app` spec says.

The lock makes "two requests with the same code, one wins" and "no more than ten failures checked" hold without relying on retries. It is a different lock key from the existing pending sign-in lock, and the order is always pending-sign-in work first, then this lock, so the two cannot deadlock.

Alternatives considered:

- Capping `AUTHENTICATOR_CODE` pending sign-ins per hour by counting rows, as emailed codes are capped. That bounds guessing through sign-in only. The profile changes and the master-key and recovery confirmations check app codes outside any pending sign-in, and they exist precisely to stop someone who holds a stolen session; a row-count cap does not see them, and `lib/ratelimit` would leave hundreds of guesses an hour that reset on every restart. A small table that every check path shares is the simplest bound that covers all of them. Rejected.
- `lib/ratelimit` for the per-account limit. It is in memory, counts attempts instead of failures (a correct code would spend the allowance), and loses its counts on restart. Rejected for this limit; it still guards the operations per pending sign-in and per IP as today.
- A failure counter column on the app row with a window start. It cannot express a sliding hour and resets awkwardly; ten small rows per account per hour are cheap. Rejected.

### 7. Identity proof checks the requirement first, then asks `authenticator`

`identityproof.Service` decides in this order. First, if the requirement is `OFF`, a confirmation request answers 409 `identity_proof_unavailable` and no confirmation is asked, whether or not root admin has an app, as today. Next, if root admin has a connected app, `SendCode` sends nothing and answers that the app code is expected, and `Verify` checks the submitted value through the check of decision 6. Otherwise both behave as today, including 409 when mail cannot reach root admin. The requirement check moves out of `canEmail` into this first step, and `canEmail` keeps only the mail and address conditions.

The API reflects this by adding `method: EMAIL | AUTHENTICATOR_APP` to `IdentityConfirmation` and making `expiresAt` present only for `EMAIL`. The frontend reads the signed-in account first: with an app it shows the app code field and sends no confirmation request, and otherwise it keeps the emailed code flow with its send action and switches to the app code field when that request answers `method: AUTHENTICATOR_APP` (decision 11). An app code confirmation is not bound to the session or purpose the way an emailed code is; the replay rule of decision 6 makes it single-use across all purposes instead.

Alternative considered: a separate operation for app confirmation. The frontend would have to learn the method from another call before choosing which operation to use, and both purposes already pass through `SendCode` and `Verify`. Rejected.

### 8. HTTP operations

The sign-in answer `SignInOutcome` (a `oneOf` on `outcome`) gains two variants:

- `AUTHENTICATOR_CODE_REQUIRED` with `pendingSignInId`.
- `AUTHENTICATOR_SETUP_REQUIRED` with `pendingSignInId`, `secret`, `otpauthUri` and `expiresAt`. Each such answer issues a new setup, so the secret is a credential returned once in the response of the operation that issued it, which the api-design skill allows. No operation returns a stored secret.

Existing `POST /pending-sign-ins/{pendingSignInId}/verify` accepts the app code or a recovery code for an `AUTHENTICATOR_CODE` pending sign-in through its existing `code` field (decision 6 tells them apart by form), and answers a wrong one with the same `sign_in_code_incorrect` as a wrong emailed code. Resend answers 409 `pending_sign_in_not_resendable` for the other kinds.

New operations, all under existing collections and using verb sub-paths for actions, as the api-design skill requires:

| Operation | Access | Purpose |
|---|---|---|
| `POST /pending-sign-ins/{pendingSignInId}/confirm-authenticator-app-setup` | public, Turnstile | confirm the required setup; answers the session and `recoveryCodes` |
| `POST /users/{userId}/authenticator-app-setups` | own account | start a setup; with an app connected, requires `code` (replacement); answers `setupId`, `secret`, `otpauthUri`, `expiresAt` |
| `POST /users/{userId}/authenticator-app-setups/{setupId}/confirm` | own account | confirm with `code`; answers `recoveryCodes` |
| `POST /users/{userId}/disconnect-authenticator-app` | own account | requires `code`; 409 `authenticator_app_required` under the app method |
| `POST /users/{userId}/recovery-codes` | own account | requires `code`; creates a new set and answers `recoveryCodes` |
| `POST /users/{userId}/reset-authenticator-app` | admin | administrator reset under the deactivation rules; answers the `User` |

"Own account" follows `changeUserPassword`: another user's ID answers 403 `permission_denied`. `User` gains the nullable `authenticatorAppConnectedAt` and `unusedRecoveryCodeCount`, both null without an app, so the profile and the administration list read the state from the user they already load. The profile learns the required method from `getInstanceSettings`, which every signed-in user reads.

Problem codes and where they answer:

- `second_factor_invalid` (400): a wrong or missing current code in the profile operations (replacement start, disconnect, regeneration) and a wrong code confirming a profile setup.
- `sign_in_code_incorrect` (existing): a wrong app or recovery code at sign-in and a wrong code confirming the sign-in setup.
- `identity_proof_invalid` (existing): any refused confirmation of a master-key export or recovery apply.
- `too_many_wrong_codes` (429): the per-account limit of decision 6, wherever the check runs. Its name sets it apart from the existing `too_many_sign_in_codes`, which counts requested emails.
- `authenticator_app_required` (409) and `pending_sign_in_not_resendable` (409) as above.
- `authenticator_setup_not_usable` (410): a setup that expired, was replaced or was destroyed by wrong codes, in the profile; at sign-in the existing `pending_sign_in_not_usable` covers it.

### 9. Console command `--reset-2fa --email=<address>`

`backend/cmd/databasus/options.go` gains the flag, its place after `--disable-2fa` in the command order, and the `ErrEmailMissing` refusal before any database work, tested in `backend/cmd/databasus/binary_test.go`. The usage text of `--disable-2fa` ("Stop requiring a second factor at sign-in") and `--email` (also naming the reset) is updated. `backend/internal/app/console.go` runs it through a new `users.Module.ResetAuthenticatorAppFromConsole`, which shares the reset with the administrator path and records the audit event without an author, like `DisableTwoFactorAuth`. The `FromConsole` suffix follows `CreateUserFromConsole` and `ChangePasswordFromConsole`, the majority of the console methods.

The name pairs with `--disable-2fa`, which operators already know; `--reset-authenticator` was considered and rejected because the README and troubleshooting docs speak of "2FA".

### 10. Recovery normalization drops setups and failures, keeps apps

`endSessionsAndCodes` additionally deletes `users_authenticator_setups` and `users_second_factor_failures`. Connected apps and their recovery codes are part of the account and stay, so a recovered instance asks the same users for the same apps. `listSettings` reads `two_factor_requirement` instead of the removed column.

### 11. Shared second-factor UI lives in the users entity

The QR setup step, the recovery codes display with copy and download, the app code input with its recovery-code switch, and the read of whether the signed-in account has an app are used by the auth page, the profile, the recovery dialog and the `export-master-key` feature that master-key onboarding and export share. `docs/architecture.md` (Frontend) forbids imports between slices of one layer, so a feature slice would be unreachable from `export-master-key`. These parts therefore go into `frontend/src/entities/users` with its public API, next to the account model and API they display. Alternatives: a `features/authenticator-app` slice, which `export-master-key` could not import; or copies in each consumer. Both rejected; four copies of a security-relevant flow drift apart.

A confirmation shows the app code field without a send action when the signed-in account has an app, and the emailed code flow otherwise; when the request for an emailed code answers `method: AUTHENTICATOR_APP`, it switches to the app code field. Knowing the method before any request keeps the email flow's explicit send action, which an automatic confirmation request on opening would replace with an unasked email.

## Risks / Trade-offs

- [An attacker who knows the password of an account without an app can complete the required setup with their own phone] → This is inherent to any "set up at next sign-in" requirement. Administrators see who has no app in the user list (`authenticatorAppConnectedAt`) and can deactivate or reset; the risk ends once the real user has connected an app.
- [Someone who knows the password can lock the real user out of password sign-in for an hour by spending the ten wrong codes] → Accepted, as the spec states: existing sessions keep working, and the alternative is unlimited guessing. The limit is per account, not per IP, deliberately.
- [A reload of the setup screen shows a new secret, so an already scanned entry stops working] → Accepted to keep every secret in exactly one response. The setup screen says that the newest QR code is the one to scan.
- [A lost phone plus lost recovery codes on a single-admin instance] → `--reset-2fa` from the host, documented beside the password recovery commands.
- [Clock drift beyond 30 seconds on the server] → Codes fail for everyone with an app. Docker hosts normally sync time; the password page of the website mentions keeping the server clock synchronized. Widening the window was rejected because it weakens every account to fix a misconfigured host.
- [The setup secret is returned by an unauthenticated operation answer] → It is returned only after a correct password, only for an account without an app, and the setup it belongs to completes only with a code that the holder's app computes; it does not let anyone sign in without the password.
- [Turnstile on the setup confirmation adds friction on the second screen] → Same as the existing code verification step; consistency outweighs the extra challenge.

## Migration Plan

One migration, applied by the server at startup:

1. Add `users_settings.two_factor_requirement`, set it to `'EMAIL'` where `is_two_factor_auth_required` is true, then drop the boolean.
2. Create the four new tables, add the three pending sign-in columns, make `hashed_code` nullable, and add the `num_nonnulls` check, which every existing row satisfies.

The `Down` section first deletes pending sign-ins whose `kind` is not `EMAIL_CODE`, then drops the check and the new columns, makes `hashed_code` NOT NULL again, drops the new tables, and restores the boolean as `two_factor_requirement <> 'OFF'`, losing the method. Behavior after the upgrade is unchanged for every instance: nobody has an app yet, and `EMAIL` behaves as the old `true`. Rollback to the previous release restores the old behavior; connected apps are lost, and accounts sign in with the emailed code or without a second factor as the boolean says.

The migration gets a migration test in the style of `backend/migrations/*_migration_test.go`, covering the boolean mapping both ways and a `Down` with app-code pending sign-ins present.
