## Context

See `proposal.md` for the motivation and `specs/root-admin-account/spec.md` for the required behavior.

`CompleteReset` (`backend/internal/modules/users/internal/passwordreset/service.go:142`) checks the password length, hashes the new password (`service.go:150`) and only then opens a transaction (`service.go:155`). Inside it, `getNewestUsable` (`reset_code_repository.go:54`) reads the newest unused, unexpired code with `FOR UPDATE`, and the code is compared with bcrypt while that lock is held.

Two-factor sign-in solves the same problem in another order (`signin/two_factor_verification_service.go:27`): a rate limit, then an atomic attempt claim outside any transaction (`two_factor_verification_service.go:53`), then the bcrypt comparison, then a transaction that spends the row with a conditional `UPDATE` (`two_factor_verification_service.go:69`). The claim and the spend are in `signin/pending_sign_in_repository.go:140` and `:125`.

The code request allowance stays as it is: three codes an hour per user, counted in the database (`service.go:31`).

The design is constrained by `backend/AGENTS.md` (roles of `service.go`, `*_repository.go`, `<entity>.go` and `errors.go`), by the root `AGENTS.md` rule against "how it was" comments, which the comment above `CompleteReset` breaks today, and by `docs/architecture.md`, which keeps this logic inside the `users` module. No decision changes the architecture.

## Goals / Non-Goals

**Goals:**

- Bound guessing to five attempts per code in a way a rollback, a restart or a burst of requests cannot get around.
- Keep every bcrypt computation behind something the caller must earn.

**Non-Goals:**

- Sharing code between `signin` and `passwordreset`. The two tables, their columns and their refusals differ, and a shared helper would have to take the table name as a parameter. Each package keeps its own small claim and spend.

## Decisions

### Claim an attempt before comparing, outside a transaction

`claimAttempt` runs one atomic `UPDATE ... SET failed_attempt_count = failed_attempt_count + 1 WHERE id = ? AND is_used = FALSE AND failed_attempt_count < 5 AND expires_at > ? RETURNING failed_attempt_count`. It commits at once, so returning an error afterwards cannot undo it. The correct code also takes an attempt, so the column counts checks that were made, and the fifth attempt may still be correct. The name `failed_attempt_count` is not exact for a successful attempt. It stays, because `users_pending_sign_ins` uses the same name with the same meaning and two names for one concept would mislead more.

A claim that changes no row means the code is used, exhausted or expired, and it answers `ErrInvalidResetCode`.

Alternatives rejected:

- **Increment in a separate transaction while the current one stays open.** The open transaction holds `FOR UPDATE` on the same row, so the increment waits for it, and it waits for the increment. Postgres cannot see the cycle because one side waits in Go, so the request hangs.
- **Increment inside the transaction and commit it by returning the refusal after the commit.** This works and keeps the row lock, but bcrypt would run under a lock and an open database connection, and every guess would hold both for the length of a comparison.
- **Compare first, then count.** A burst of simultaneous guesses would all compare before any of them counted, which breaks the cap.

### Only the newest code counts

A new `getNewest` reads the row of the user with the latest `created_at`, with no filter and no lock, and `claimAttempt` alone decides whether it may be attempted. `getNewestUsable` is deleted. `PendingSignIn.IsUsable` (`signin/pending_sign_in.go:30`) has no counterpart here, because it exists for a second reader that never claims an attempt, and `CompleteReset` is the only reader of `getNewest`.

Real requests get distinct timestamps in practice. If two simultaneous requests did tie, either row was issued at that moment, so no separate tie-break is added. The test clock stands still, so tests with two codes advance it between the requests.

Alternatives rejected:

- **Add the attempt count to the filter in `getNewestUsable`.** Exhausting the newest code would bring the previous one back to life. Today the same filter already lets an older unused code change the password again after a completed reset.
- **Mark older codes used when a code is issued or a reset completes.** It gives the same behavior with an extra write on each path and more ways to get it wrong.

### Spend with a conditional update instead of a row lock

`markUsed` becomes `spend`, an `UPDATE ... SET is_used = TRUE WHERE id = ? AND is_used = FALSE AND expires_at > ?` that reports whether it changed a row. It runs in a transaction with `ReplacePassword`. A second request with the same correct code changes no row and gets `ErrInvalidResetCode`, so one code still cannot complete two resets without the `FOR UPDATE` lock.

The alternative of keeping `FOR UPDATE` was rejected because the claim no longer needs it and the lock would only cover the two final writes.

### Hash after the comparison

The order becomes: password length, user lookup, newest code, claim, compare, hash, then the transaction that spends the code and replaces the password. Hashing stays outside the transaction so no lock or connection is held during bcrypt.

The alternative of hashing inside the transaction was rejected for that reason. Hashing before the comparison, as today, is what lets an unauthenticated caller load the CPU.

The length check stays first because it costs nothing and needs no counted attempt.

### No rate limit on code entry

`CompleteReset` gets no limiter, and `checkRateLimit` and `ErrRateLimited` stay as the code request uses them. The attempt cap bounds guessing at 15 attempts an hour, and a request costs bcrypt only while the newest code has attempts left, so one code causes at most five comparisons and five hashes. Several hashes happen only when simultaneous requests all carry the correct code.

Alternatives rejected:

- **Limit per address, with the limiter sign-in code verification uses (`two_factor_verification_service.go:38`).** It would bound neither guessing nor hashing beyond the cap, because the attacker can vary the address. A victim's correct code can be refused either way, and the limiter would only add a different answer.
- **Limit per IP address.** The reasoning about proxies and trusted headers is a separate decision.

### One refusal for every unusable code

`ErrInvalidResetCode` keeps its code and one shared detail. Its comment and detail are reworded to include exhausted and older codes, because `detail` is for logs and the frontend chooses its text by `code`.

### Warn in the log at the fifth wrong attempt

The refusal path logs a warning with the user ID and without the code when the fifth claimed attempt is wrong. This is an operational detail, so no spec requirement covers it. The alternative of an audit event, as `refuseWrongSignInCode` records, was rejected because it needs a new audit dependency in this service for a signal the victim already sees as unexpected emails. It can be added later without changing the behavior specified here.

## Risks / Trade-offs

- [Guessing stays possible at a low rate: 15 attempts an hour, about 12% over a year of continuous attack by calculation, not measurement] → The attacker triggers three emails an hour to the victim, and the fifth wrong attempt on each code logs a warning.
- [An unauthenticated caller can request a code and spend it with five wrong guesses, up to three times an hour, which can deny the victim the email reset for as long as the attack runs. This is new: today a wrong guess spends nothing, so the newest mailed code keeps working] → Accepted as the cost of the cap. Sign-in with the current password and the console password reset stay available.
- [A correct older code can still complete a reset if a newer code is inserted between `getNewest` and `spend`, during the comparison and the hash] → Accepted. The window is short, the caller must hold a real mailed code, and closing it would add a subquery to `spend` for no gain against guessing.
- [A failed email delivery still replaces the older code, because `RequestReset` stores the new code before sending] → This already happens today, because only the newest usable code is compared. The user requests another code.
- [Code entry has no rate limit, so a flood of requests still reaches the database, and a bug in the counter has no second line of defence] → Once a code's attempts are spent, such a request costs two reads and one `UPDATE` that changes no row, and no bcrypt. Scenarios pin the cap, including a burst of simultaneous attempts.
- [A correct code with a password longer than bcrypt accepts spends an attempt and fails without burning the code] → This failure exists today and is out of scope. The user has four more attempts.

## Migration Plan

A goose migration adds `failed_attempt_count INTEGER NOT NULL DEFAULT 0` with a non-negative check, matching `users_pending_sign_ins`. Codes live for an hour at most, so codes present during the upgrade simply start at zero. Rolling back drops the constraint and the column.
