## Why

Databasus offers one second factor today: a six-digit code emailed at password sign-in. An instance without a working mail server cannot require it at all, and a mail outage locks everyone out while it is on. Self-hosted installations in private networks and Kubernetes clusters often have no SMTP, yet they guard backups of production databases, so they need a second factor that depends on nothing outside the instance. GitHub issue databasus/databasus#901 asks for exactly that: codes from an authenticator app (TOTP, RFC 6238).

## What Changes

- Any active user can connect an authenticator app in the profile. Databasus shows a QR code and the secret for manual entry, and the app is connected only after the user enters a code it generated. A user with a connected app is asked for the app's code at password sign-in instead of an emailed code, whether or not the instance requires a second factor and whichever method it requires. No mail is sent in that case.
- The instance setting that requires a second factor still covers every account, administrators included. When it is on, an administrator chooses the method: an emailed code, with today's preconditions (a mail server and a real address for every active administrator), or an authenticator app, with no mail preconditions. Under the app method, a user without a connected app sets one up right after the password step, and the session is issued only after that setup is confirmed with a code.
- Connecting an app issues ten single-use recovery codes, shown once. A recovery code completes a sign-in in place of the app's code. The signed-in user can see how many remain and can replace the set, which invalidates the old one.
- A user can disconnect their app, replace it with another one, or regenerate recovery codes only after entering a current app code or an unused recovery code. A secret is shown once: each start of a setup, including each password step that requires one, issues a new secret. Disconnecting is refused while the instance requires the app method; replacing the app stays possible.
- An administrator can reset another user's authenticator app under the same rules that govern deactivating that user. A new host console command, `--reset-2fa --email=<address>`, resets one account's app and recovery codes for an owner who has lost the device. Both write audit events.
- Confirming a master-key export or a Databasus recovery apply, which today requires an emailed code while the instance requires a second factor, accepts the root admin's app code instead when root admin has a connected app. A root admin with a connected app therefore no longer needs a mail server for this confirmation. A root admin without an app, for example one who signs in only through Google or GitHub, still needs the emailed code.
- Wrong app and recovery codes are limited per pending sign-in, as emailed codes are, and also per account over an hour, because app codes can be tried without sending any mail, at sign-in and in the profile alike.
- **BREAKING** (API): the boolean `isTwoFactorAuthRequired` in the instance settings schemas is replaced by the enum `twoFactorRequirement` with the values `OFF`, `EMAIL` and `AUTHENTICATOR_APP`. The sign-in answer gains new outcomes for an app code and for a required app setup. The boolean column `users_settings.is_two_factor_auth_required` is replaced by a text column; the migration maps `true` to `EMAIL` and `false` to `OFF`. The old field and column are deleted, not kept as aliases.
- The README in all seven languages, the website security page and home-page security answer in every language, and the password recovery documentation describe the authenticator app and the new console command.

## Out of scope

- A second factor for sign-in through Google or GitHub. These providers run their own checks; the existing exception stays.
- Any second-factor method other than the emailed code and TOTP, such as WebAuthn, passkeys, SMS or push approval.
- Requiring a second factor per role, per workspace or per user. The requirement stays one setting for the whole instance.
- Letting the user choose between email and app on each sign-in. A connected app is always the method for that user.
- Remembering a trusted device to skip the second factor.
- Notifying a user by email when their app is reset or a recovery code is used.
- Ending existing sessions when the requirement is turned on, when the method changes or when an app is reset.
- Step-up confirmation for operations other than master-key export and recovery apply.
- Configurable TOTP parameters. Codes are always six digits, SHA-1 and 30 seconds, the defaults every listed authenticator app supports.

## Capabilities

### New Capabilities

- `authenticator-app`: connecting, replacing and disconnecting an authenticator app; the code check with clock drift and replay protection; recovery codes; resetting an app by an administrator or from the host.

### Modified Capabilities

- `two-factor-authentication`: the instance requirement gains a method choice; the password step chooses between an app code, an emailed code and a required app setup; app codes share the pending sign-in limits and add a per-account limit; the console recovery path names the new reset command; the documentation requirement covers the app.
- `host-console-commands`: the stable command list gains `--reset-2fa` with `--email`, and its audit record.
- `master-key-export`: the export confirmation accepts root admin's app code.
- `databasus-recovery`: the recovery apply confirmation accepts root admin's app code.
- `frontend-user-screens`: the sign-in screens add the app code step, the recovery code alternative and the app setup step; the profile adds the authenticator section; the instance settings add the method choice; user administration adds the reset action.

## Impact

- Backend: the `users` module gains the authenticator app data, its service and its HTTP operations; the sign-in, settings, administration and identity proof services change; `internal/app/console.go` gains the reset command. Migrations add the authenticator app and recovery code tables, add a method column to pending sign-ins and replace the settings column. The TOTP algorithm is written over the Go standard library, so no new Go dependency is added. The secret is encrypted with the master key through `lib/crypto`.
- API: `backend/api/openapi.yaml` gains operations for the profile authenticator, the sign-in setup and app code steps, and the administrator reset; the settings schemas break as described above. Servers, test clients and frontend types are regenerated.
- Frontend: the auth page, profile, instance settings, user administration, master-key export and recovery confirmation screens; dictionaries in every interface language. A QR code library is added to `frontend/package.json` to draw the `otpauth://` URI.
- Documentation: `README.md` and the six `assets/readme/README.<locale>.md` copies; the website security, home page and password pages in every language; `docs/glossary.md` and `docs/architecture.md` where they describe two-factor sign-in.
- This change answers to the root `AGENTS.md`, `backend/AGENTS.md`, `frontend/AGENTS.md`, `website/AGENTS.md` and `assets/readme/AGENTS.md`.
