## Why

A password reset code has six digits and lives for an hour, and a wrong guess does not spend it. Anyone who can reach the server can request a code for the administrator's address and keep guessing, without an account. Each guess also makes the server compute an expensive password hash before it even looks up the user, so the same public operation can load the CPU.

## What Changes

- A reset code accepts five attempts. After that it stops working, even for the correct value, and the user requests a new one.
- Only the newest code of a user can complete a reset. An older code stops working as soon as a newer one exists, and it does not come back when the newest is exhausted or used.
- Attempts are recorded durably at the moment they are made, so a failed request and a restart both keep the count, and simultaneous guesses cannot exceed five.
- The new password is hashed only after the code is verified.

### Out of scope

- A human-verification challenge on code entry.
- Limits per IP address.
- A rate limit per address on code entry. The attempt cap and hashing after the code check already do the protective work: three codes an hour at five attempts each allow only 15 guesses, a request without a usable code costs no bcrypt, and an attacker could vary the address anyway. Without it there is no 429 answer on code entry and no second line of defence if the counter has a bug.
- Changes to the frontend screens. The current `invalid_reset_code` message already tells the user what to do.
- Making the response time equal for an address with a usable code and one without. That difference exists today and this change does not widen it.
- Refusing passwords longer than bcrypt accepts. That failure exists today; this change only moves it behind the code check.
- Changing how many codes can be requested per hour.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `root-admin-account`: adds requirements on how many attempts a reset code accepts, which code counts and when the new password is processed. The reset request requirements stay as they are.

## Impact

- Answers to `backend/AGENTS.md`, the root `AGENTS.md` and the `testing` and `api-design` skills.
- Database: a migration adds an attempt counter to `users_password_reset_codes`. Codes that exist during the upgrade start at zero attempts.
- API: `completePasswordReset` in `backend/api/openapi.yaml` documents the attempt cap and the rule about the newest code. Its responses stay as they are. Regenerating updates the generated descriptions.
- Code: the `users` module, in `internal/passwordreset`.
- Users: somebody who mistypes a code five times must request a new one. This also creates a new accepted risk. An unauthenticated caller can request a code for a victim's address and spend it with five wrong guesses, up to three times an hour, which can deny the victim the email reset for as long as the attack runs. Today a wrong guess spends nothing, so the victim can still use the newest mailed code. Sign-in with the current password and the console password reset stay available.
