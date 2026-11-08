## Why

Password sign-in reports every credential refusal as an incorrect email or password. Users cannot tell whether they mistyped their address, entered the wrong password, have not completed registration, or have an OAuth account without a password, so the form gives them no useful next step.

## What Changes

- Distinguish an unknown email, an incorrect password, incomplete registration and an account without a password. Give each refusal translated guidance appropriate to its cause.
- For an account without a password, name its linked Google or GitHub providers only when they are configured on the instance. Reuse existing OAuth buttons, registration and password recovery rather than introducing a separate account-discovery screen.
- Keep deactivation hidden until a correct password has been verified. An inactive account without a password receives the same password-absence guidance as an otherwise equivalent active account.
- Preserve human verification, the existing ten-attempts-per-minute limit per submitted address, and the requirement that failed sign-ins issue neither a session nor a second-factor code.
- **BREAKING**: replace the user API's `invalid_credentials` code with specific refusal codes. Delete the old code, sentinel and translations instead of retaining compatibility aliases. This deliberately replaces the requirement that unknown addresses and wrong passwords be indistinguishable; it never suggests another account's address or the `admin` login.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `root-admin-account`: distinguish password sign-in refusals while preserving the password requirement for revealing deactivation and the prohibition on disclosing other logins or account counts.
- `two-factor-authentication`: retain password verification before issuing a code, while permitting the same specific refusal reasons with two-factor authentication on or off.
- `frontend-session`: show the specific translated refusal and the appropriate existing correction path without losing entered values or triggering another sign-in automatically.

## Impact

- Backend users sign-in logic, shared API error codes, the user OpenAPI contract and generated servers/clients; no database migration or new endpoint.
- Frontend sign-in feedback, API error translation, all seven dictionaries and affected API/screen tests.
- Existing backend sign-in, invitation, OAuth, password-reset, profile, administration and second-factor tests that assert the old code.
- Governed by root `AGENTS.md`, `backend/AGENTS.md` and `frontend/AGENTS.md`; the existing API and frontend architecture remain in place.

## Out of Scope

- Changing OAuth authentication, linking accounts, registration policy, password-reset delivery policy, two-factor verification or rate-limit settings.
- Exposing another user's address, account count, provider identifiers or secrets.
- Reporting deactivation before a correct password, including for an account without a password. Existing authenticated OAuth refusal behavior remains unchanged.
- A new email-discovery endpoint, a multi-step login redesign, new providers, database schema changes, or a general error-metadata framework.
