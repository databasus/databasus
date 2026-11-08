## Context

See `proposal.md` for the motivation and scope.

- `backend/cmd/seed_dev/main.go:48-72` reads settings through `config.ReadVariables`, requires development mode and fixes the dev email as `test@email.com`.
- `backend/cmd/seed_dev/seed.go:47-58` checks health and authenticates before creating the workspace. Authentication currently returns only the token (`:102-139`), although the existing session responses also contain `userId` (`backend/api/openapi.yaml:3682-3733`).
- `backend/internal/modules/users/internal/accounts/user_record.go:23-29` stores the role separately from `IsBootstrapAdmin`. The unique bootstrap index is identified in `user_creation_service.go:14-16` of the same package.
- `backend/internal/modules/users/internal/accounts/administration_service.go:122-128` refuses ordinary role changes to the bootstrap administrator and reserves administrator management for that account. There is no public bootstrap-transfer operation.
- `backend/internal/modules/users/internal/signin/authentication_service.go:16-33` reloads the account on each request, so an existing session observes role changes without another sign-in.

Root `AGENTS.md`, `backend/AGENTS.md`, and the ownership, composition and transaction rules in `docs/architecture.md` constrain this change. Queries belong to users repositories, domain behavior stays in users services, and dependency wiring stays in `internal/app`. The approach preserves those boundaries and needs no architecture-rule change.

## Goals / Non-Goals

**Goals:** Make the existing dev account usable for all administrator screens after a successful seed, including when another account claimed the instance first. Keep the transfer atomic, auditable and repeatable.

**Non-Goals:** See the proposal's out-of-scope list. The seed will not build a second full application, run migrations, start jobs or expose another HTTP operation.

## Decisions

### 1. Keep account authentication before a dedicated local operation

Retain the existing health check and one-attempt sign-in/sign-up flow. Extend the seed's response projections to read the existing `userId`, and return the authenticated account identity together with its token. After authentication, call the local assignment operation before finding or creating the workspace. Require a nonzero user ID; use the configured seed email as the second identity check.

Read `DATABASE_DSN` through the existing configuration loader and carry it and the environment mode in seed parameters. The command still targets its fixed local backend. Connect only after successful authentication; require the DSN rather than introducing another default or connection source. Close the pool after the local operation. Tests supply their own app's `GetServerConfig().DatabaseDsn` and development mode.

Rejected alternatives: assigning by email before authentication would change privileges after a failed sign-in; registering directly in SQL would bypass registration policy; making every development backend startup transfer privileges would broaden the trigger beyond seeding.

### 2. Expose a narrow users-module service through lightweight app composition

Declare a separate `DevSeedAdmin` interface and its dependency/request types in `users/module.go`, with an `EnsureBootstrapAdmin` operation returning whether anything changed. Keep the existing bootstrap-administrator terminology in new identifiers and test names. Its implementation lives under `users/internal/accounts`, using a dedicated service and the users repository. Do not add it to the ordinary `users.Module` interface or wire it into routes or console commands.

Add a lightweight entry point in `internal/app` for the seed. It rejects modes other than `development`, opens the configured database, and constructs only the transactor, audit log and users dev-seed service. Supply the clock and logger explicitly; do not call `app.New`. The service accepts the authenticated ID and configured email together as a named request. It verifies that both identify the same active row before modifying anything. A mismatch fails the seed.

Rejected alternatives: SQL inside `cmd/seed_dev` would violate data ownership; a new HTTP endpoint would unnecessarily expand the authorization surface; constructing the full application would run unrelated initialization against an already-running instance. A new general console command would change production behavior outside the request.

### 3. Serialize and commit the assignment as one users transaction

Use `database.Transactor.Transaction`. Add one repository operation for a short `SHARE ROW EXCLUSIVE` lock on the users table. Both the seed transfer and the existing `administerUser` transaction acquire this same lock as their first database operation, before reading the actor, target or current bootstrap designation. Keep the lock until the transaction commits or rolls back. It serializes the transfer with the authorization checks and writes of role changes, activation and deactivation, as well as with concurrent seeds and registration writes. Do not hold it across network requests.

The ordinary administration path must reread and validate the actor's active status and administrator role under this lock, then apply its existing bootstrap-protection and administrator-management checks to the locked transaction's current actor and target records. A request whose middleware authenticated the previous bootstrap administrator before the transfer must still see that account's loss of bootstrap powers. If administration commits first, the seed validates its resulting target state and refuses an inactive target; if the seed commits first, administration checks the new designation and refuses to demote or deactivate it. A lock taken only when writing does not provide this guarantee: `administration_service.go:158-179` currently reads both records before `user_repository.go` updates role or status by ID.

After validating the target, return unchanged if it already has both `ADMIN` and the bootstrap flag. Otherwise clear the previous holder's bootstrap flag, set the target's role to `ADMIN` if needed, and set its bootstrap flag. Keep the existing unique index. Each repository update changes only the intended column; do not save whole account records. Save an audit event in the same transaction, naming the previous and new account IDs and addresses. Treat this as a host action with no impersonated audit author. If there was no previous designation, record an assignment rather than inventing a previous holder.

Any statement or audit failure rolls back all privilege changes. A later failure to create a workspace or database does not undo the completed assignment or the account registration. Leave passwords, password versions, status, settings and all unrelated roles unchanged. A no-op writes no transfer audit event.

Rejected alternatives: separate transactions could leave no bootstrap administrator; a single update of both flags can conflict with the immediate unique index; only locking the current holder misses the case where no holder exists. A table lock used only by the seed leaves administration free to read stale permissions before its write waits. Updating the existing administration transaction to take the same lock before reading is simpler than coordinating ordered row locks across actors, targets and a potentially absent bootstrap holder.

### 4. Verify through the seed and existing administrator operations

Use the real seed against isolated `apptest` applications. Prepare accounts and settings through the existing API. Check flags through SQL only where the API does not expose them; verify actual powers through administrator role-management requests from the dev account and previous holder, including sessions issued before the transfer. Keep existing authentication, health, edited-item and overlapping-run coverage.

Start with a regression scenario where an existing dev member fails to become the bootstrap administrator on the current code. Replace the existing forbidden-workspace expectation with successful seeding when member workspace creation is disabled. Test rollback by adding a temporary failing database trigger to the test's own database for the final bootstrap grant, then check that both roles and the old designation remain unchanged. Exercise audit-write rollback with the same isolated-database technique. Do not mock our modules, repositories or database.

Add controlled concurrency scenarios under `TestSeedBootstrapAdmin`. In the isolated test database, pause the transfer just before the final bootstrap grant using a target-specific temporary trigger waiting on an advisory lock held by a test transaction. While the seed owns the users-table lock but has not committed, send a role-change or deactivation request from the previous bootstrap administrator. Wait through `testwait.Until` for PostgreSQL lock diagnostics to show that request blocked by the seed, then release the advisory barrier and let the seed commit. Without the administration lock, the request reads the old records and applies its stale decision after the transfer; with the shared protocol, it blocks before reading, sees the committed designation and is refused. Cover both demotion and deactivation of the dev account, and the previous holder trying to grant administrator privileges to a third account. Assert that the dev account remains active, `ADMIN` and the sole bootstrap administrator, and that no forbidden administration audit event was committed. Clean up the trigger and barrier on every outcome; use no sleeps, scheduling assumptions or production test switches.

Rejected alternatives: checking only `role=ADMIN` misses the bootstrap powers; testing only the new service misses seed ordering and wrong-database wiring; adding production failure switches solely for tests would enlarge the implementation unnecessarily.

## Risks / Trade-offs

- The command now needs direct internal-database access in addition to HTTP access. Use the existing checkout DSN, verify the authenticated ID and email in that database, and never print the DSN, credentials or session token. Connection failures must report a sanitized step-level error.
- The users table lock briefly delays account writes and serializes ordinary user-administration transactions, including in production. This limited coordination change is necessary to preserve bootstrap protection during a dev transfer; it adds no production transfer capability. Keep only validation, the few updates and the audit write under it; connection setup and HTTP requests stay outside.
- Existing requirements forbid role changes and tie the bootstrap designation to account creation. The two delta specs explicitly describe the development-seed exception. Update the seed package documentation, including its claim that existing accounts are only signed in to, and the bootstrap field description if it implies the designation can never move. Search active documentation for affected promises; do not rewrite historical changes or agent instructions.

## Migration Plan

No schema migration, API generation or data backfill is needed. The next successful `mise run run` or standalone dev-seed invocation performs the assignment. Production startup and host-console behavior stay unchanged. Reverting the code stops future reassignment but does not reverse a transfer already committed; the previous holder remains an ordinary administrator.
