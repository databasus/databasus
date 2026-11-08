## Context

See `proposal.md` for the user problem and scope. This change spans the user API and frontend feedback, and changes intentional disclosure at sign-in, so it needs an explicit design.

- Password sign-in checks Turnstile and the per-address limiter before account lookup, combines unknown/invited/passwordless cases, and checks deactivation only after a successful password comparison (`backend/internal/modules/users/internal/signin/password_sign_in_service.go:37`, `:86`).
- The account record already carries a nullable password and independent Google/GitHub links (`backend/internal/modules/users/internal/accounts/user_record.go:16`). OAuth registration leaves the password unset, while linking a provider does not remove a stored password (`backend/internal/modules/users/internal/signin/oauth_service.go:144`, `:162`).
- Password recovery admits active accounts without requiring an existing password; inactive accounts receive no mail (`backend/internal/modules/users/internal/passwordreset/service.go:105`). Its delivery policy remains unchanged.
- The form retains API errors and translates them during rendering, and already has OAuth, registration and optional recovery controls (`frontend/src/pages/auth/ui/SignInComponent.tsx:91`, `:111`, `:195`). Provider buttons follow deployment configuration (`frontend/src/features/oauth-sign-in/ui/OAuthSignInButtons.tsx:35`).
- `docs/architecture.md`, sections "Modules", "HTTP API" and "Frontend", keep account rules in the users service, errors in the shared Problem format, and translations in the interface. Root `AGENTS.md`, `backend/AGENTS.md` and `frontend/AGENTS.md` constrain the implementation. No architectural boundary changes.

## Goals / Non-Goals

**Goals:** give a specific refusal and useful existing correction path; retain deactivation secrecy until password proof; keep the error transport and sign-in flow small.

**Non-Goals:** add error metadata, an account-discovery request, new authentication flows, or changes to password recovery, linking or two-factor policy. See the proposal for the complete exclusions.

## Decisions

### 1. Use the existing Problem code and translation mechanism

Replace `invalid_credentials` in the user contract and shared error catalogue with these 401 codes. Each has its own complete translated sentence; the body shape remains unchanged.

| Code | English message |
| --- | --- |
| `email_not_found` | No account was found with this email address. Check the address. |
| `password_incorrect` | The password is incorrect. |
| `registration_incomplete` | Registration is not complete. Sign up with this email address to accept your invitation. |
| `password_not_set` | This account has no password. |
| `password_not_set_google` | This account has no password. Sign in with Google. |
| `password_not_set_github` | This account has no password. Sign in with GitHub. |
| `password_not_set_google_github` | This account has no password. Sign in with Google or GitHub. |

Keep 403 `user_deactivated` for a successfully verified password and translate it as an instruction to contact an administrator. Add the new codes to `openapi.yaml` first and regenerate with `mise run generate-api`. Update shared status/title mappings, sentinels and every affected assertion. Remove the old user API code and dictionary key outright. The verification module's unrelated internal logging reason with the same text is not a user API error and is outside this removal.

**Alternatives rejected:** parsing backend `detail` would break localization and stable error handling. Adding provider arrays to the generic Problem and carrying them through every error layer costs more than the four fixed guidance variants for two existing providers. Keeping `invalid_credentials` for wrong passwords preserves the ambiguous old meaning and conflicts with the explicit replacement.

### 2. Classify the account before password comparison; reveal deactivation only after it

Keep this order:

```text
Turnstile --> Per-address rate limit --> Account lookup
                                            |
                +---------------------------+-----------------------+
                |                           |                       |
             Missing                     Invited               No password
                |                           |                       |
         email_not_found         registration_incomplete    password_not_set*

With a stored password --> Password comparison
                                  |
                       +----------+----------+
                       |                     |
                    Mismatch                Match
                       |                     |
              password_incorrect      Active-state check
                                             |
                                  +----------+----------+
                                  |                     |
                               Inactive                Active
                                  |                     |
                          user_deactivated       Session or 2FA
```

For an account without a password, choose the provider variant from the intersection of its stored OAuth links and the providers whose backend clients report `IsConfigured()`. A single available provider yields its specific code; two yield the combined code; none yields `password_not_set`. Do not inspect active status when choosing this guidance. Nil and empty password hashes both mean no password. Invited status takes precedence over absence of a password. A linked provider never bypasses comparison when a password exists.

Keep the existing dummy bcrypt comparison for branches without a usable password; the change does not need to alter password-comparison workload. Preserve reason-specific logging without credentials and remove comments that claim refusal codes are identical. Every refusal returns before session issuance or second-factor creation.

**Alternatives rejected:** reporting inactive status before password comparison violates the user's explicit choice. Treating inactive passwordless accounts differently would reveal the same state through another code or action. Recommending every configured provider can point to an unlinked method; recommending every linked provider can point to a disabled integration. Routing all OAuth-linked accounts away from passwords breaks accounts that support both methods.

### 3. Reuse form controls and translate feedback at render time

Use `translateApiError` for all refusal text and update all seven dictionaries together. Put `email_not_found` beside the email control and `password_incorrect` beside the password control; distinguish these from existing local empty/format validation. Show registration, password-absence, deactivation and infrastructure errors as form-level feedback. Do not repeat the same error both beside a field and below the form.

Keep the existing provider buttons as explicit actions; a provider recommendation starts no navigation by itself. Keep the existing sign-up action for incomplete registration. For `password_not_set` without a provider recommendation, show a separately translated whole-sentence hint: use password recovery to set a password when `IS_EMAIL_CONFIGURED` is true, otherwise contact an administrator. In that recovery case label the existing recovery action "Set a password" and route it through the existing callback. Do not promise delivery or infer that the account is active. Other cases retain the ordinary recovery label and availability.

Store the API error, not its translated text. Retain values on failure and clear the displayed refusal and its guidance when either credential changes. Keep the existing authentication-operation checks so responses from an abandoned form cannot publish feedback or activate a session. No new requests, automatic retries or automatic OAuth transitions.

**Alternatives rejected:** changing only copy cannot recover reasons the server merges. A separate email-first screen adds a discovery API and changes the interaction without improving these refusal cases. New duplicate OAuth/recovery controls would add UI work where the existing controls already perform the required actions.

### 4. Verify through the existing HTTP and screen surfaces

Update the users module's API tests and the shared API error tests. Cover each refusal, both providers, both linked providers, disabled/unlinked providers, nil and empty password hashes, a password plus OAuth, and matching active/inactive accounts without a password. Verify wrong/correct passwords for inactive accounts, per-address limits across refusal branches, and unchanged challenge ordering. With two-factor authentication enabled, assert the same refusal code, no mail and no newly created pending sign-in. Also start a pending sign-in through a successful password step, submit an incorrect password, and prove that the existing pending sign-in remains unchanged, is reused by the next correct password step and accepts its original code under the existing verification rules. Keep successful password and OAuth scenarios passing, including setting the first password through existing recovery.

Update sign-in screen tests using the real locale provider and typed HTTP mocks: field placement, provider text, existing action callbacks, recovery availability, retained values, clearing on edit, language switching and obsolete responses. Update API client assertions that still mention the removed code, and run dictionary completeness tests. Use existing browser UI authentication scenarios to check narrow translated feedback and accessible field descriptions; update any affected reference deliberately. Finish with module format/lint, the frontend build, the browser UI suite and `mise run test`.

**Alternatives rejected:** tests of private classifiers would not verify API codes or the correction path. Manual-only verification cannot establish the behavior across the refusal and locale cases. New browser end-to-end authentication infrastructure is unnecessary for this bounded change.

## Risks / Trade-offs

- Specific refusals disclose account existence and recommended linked providers for the submitted address. This is the requested product behavior. Existing attempt controls remain first; deactivation, other addresses, provider IDs and secrets remain undisclosed.
- Recovery can be offered without proving that mail will be sent, particularly for an inactive passwordless account. Preserve the existing neutral recovery response and word the hint as an available action, never a delivery or activation guarantee.
- Backend provider configuration and frontend deployment configuration must agree for provider buttons to work. Use the existing deployment configuration and test consistent enabled/disabled variants; changing deployment configuration ownership is outside this change.

## Migration Plan

No persisted data changes or migration are required. Ship the regenerated user API and frontend together under the existing user API compatibility policy. External clients matching `invalid_credentials` must adopt the specific codes; no alias or transition layer is added. Rollback restores the previous application version as a unit, without a data rollback. Main specs are synchronized when this change is finalized through the usual OpenSpec workflow.
