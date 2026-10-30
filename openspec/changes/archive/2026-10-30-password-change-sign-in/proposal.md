## Why

The profile reports a successful sign-in after changing a password even when a second factor is still required. The repair must also close a server-side gap: time-based token invalidation leaves earlier sessions valid when the password changes within the same second.

## What Changes

- After a successful password change, clear local authentication and display the actual ordinary sign-in form with a persistent-through-transition password-change notice.
- Make AuthPageComponent select signIn after loading options for an instance with existing users; an unclaimed instance still starts with signUp.
- Remove profile auto-sign-in and reuse the ordinary emailed-code and Turnstile flow.
- Replace timestamp-based password identity with a random password version rotated atomically on every successful password replacement, including multiple replacements at the same clock instant. Bind both issued sessions and pending sign-ins to that version.
- **BREAKING**: invalidate existing timestamp-only sessions and pending sign-ins at upgrade; users sign in again. Delete old claim acceptance rather than adding a compatibility fallback, while retaining pending-sign-in history needed for rate limits.
- Add same-second API regressions and screen coverage of the real profile-to-auth transition.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `two-factor-authentication`: password changes invalidate all earlier authentication state and return the interface to complete ordinary sign-in, even within one clock tick.

## Impact

Constrained by root `AGENTS.md`, `frontend/AGENTS.md`, `backend/AGENTS.md`, the testing skill and the existing session/transaction architecture. Affects the profile, AuthPageComponent and auth transition, six dictionaries, users account/sign-in repositories and services, session claims, pending-sign-in storage, a transactional migration, and API/frontend tests. Update the session glossary definition and add Password version / PasswordVersion with its canonical meaning.

HTTP operation shapes and session expiry remain unchanged; existing opaque tokens stop authorizing requests after upgrade. No runtime dependency is added.

### Out of scope

- Embedding a second sign-in or code form inside the profile.
- Changing password strength, session lifetime, bot protection or rate limits.
- Changing the product flow for reset-password or invitation acceptance, except rotating the password version whenever those existing flows replace a password.
- Preserving old timestamp-only authentication or automatically replacing a session after a password change.
