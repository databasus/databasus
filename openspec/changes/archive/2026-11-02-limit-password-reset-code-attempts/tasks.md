## 1. Storage and contract

- [x] 1.1 Create the migration with `goose -dir backend/migrations create add_failed_attempt_count_to_users_password_reset_codes sql`, adding `failed_attempt_count INTEGER NOT NULL DEFAULT 0` with a non-negative check, and a down section that removes both. Verify that the existing scenarios still pass with `mise run backend:test-package ./internal/modules/users/...`, whose test application applies migrations.
- [x] 1.2 Update `completePasswordReset` in `backend/api/openapi.yaml`: describe the five attempts, and the rule that only the newest code works, as the `api-design` skill says. Run `mise run generate-api` and verify with `mise run backend:lint` and `pnpm --dir frontend typecheck`.

## 2. Tests on the unfixed code

Each two-code scenario calls `app.Clock.Advance` between the two code requests, because the test clock stands still and two codes would otherwise share a creation time. Run all of this section with `mise run backend:test-package ./internal/modules/users/...` before section 3.

- [x] 2.1 In `TestCompletePasswordReset` (`backend/internal/modules/users/password_reset_test.go`) add the attempt scenarios. Verify that these fail on the unfixed code because the code is not spent: "refuses the correct code after five wrong attempts", "keeps the attempt count when the attempt fails", which reads the stored count, and "refuses the correct code after eight simultaneous wrong attempts", which asserts the refusal first and then a stored count of five. Verify that "accepts the correct code after four wrong attempts" and "completes one reset when two requests carry the same correct code at once", which compares the two answers and signs in with the accepted password, pass as pins.
- [x] 2.2 Add the scenarios about older codes. Verify that "refuses an older code after a completed reset" fails on the unfixed code. Verify that these pass as pins: "refuses an older code while the newest one is usable", "refuses an older code after the newest one is exhausted" and "accepts a new code after four wrong attempts on it when an exhausted one came before".
- [x] 2.3 Add the remaining scenarios. Verify that "returns invalid_reset_code for a wrong code with a password too long to hash" fails on the unfixed code. Verify that these pass as pins: "answers alike for an unknown address, a deactivated user and a wrong code", comparing status, code and detail, and "returns 400 naming the password without spending an attempt when a wrong code comes with a short password".

## 3. Implementation

- [x] 3.1 Add `FailedAttemptCount` to `ResetCode`, with `maxCodeAttempts`. Replace `getNewestUsable` and `markUsed` with `getNewest`, `claimAttempt` and `spend` in `reset_code_repository.go`, and reorder `CompleteReset` as `design.md` says: length, user lookup, newest code, claim, compare, hash, then a transaction that spends the code and replaces the password. Log a warning with the user ID and without the code when a code gets its fifth wrong attempt. Rewrite the comment above `CompleteReset`, and the comment and detail of `ErrInvalidResetCode`, to state the current behavior. Verify that every scenario from section 2 and the existing scenarios of `TestCompletePasswordReset` pass with `mise run backend:test-package ./internal/modules/users/...`.

## 4. Finish

- [x] 4.1 Search the repository for mentions of the changed behavior with `rg -n -i "reset code" README.md docs backend/AGENTS.md AGENTS.md` and bring any statement that no longer holds in line. Run `mise run backend:format` and verify with `mise run backend:lint`.

## Change verification

```sh
mise run backend:test-package ./internal/modules/users/...
```
