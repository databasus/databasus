Direct Go test commands below run from the repository root with the pinned tools active. Prepare the Fedora client-library environment once in that shell as described in `docs/testing.md`:

```sh
source scripts/fedora.sh
apply_libedit_shim
export TESTCONTAINERS_RYUK_DISABLED=true
```

## 1. Reproduce and implement bootstrap assignment

- [x] 1.1 Add a regression scenario under `TestSeedBootstrapAdmin` that creates the old bootstrap administrator and an existing dev member through the API, runs the current seed, and expects the dev account to acquire bootstrap powers while the old account stays `ADMIN`. Run `go -C backend test -count=1 -run '^TestSeedBootstrapAdmin' ./cmd/seed_dev` before implementation and confirm it fails because the dev account remains a member; rerun it after task 1.3 and require success.
- [x] 1.2 Add the separate users-module dev-seed interface, dedicated service and repository operations described in design decisions 2 and 3. Wire only the necessary dependencies through a lightweight app entry point. Implement development-mode refusal, matching active account ID and email, the transfer-side transaction lock, narrow updates, audit recording and no-op behavior. The matching ordinary-administration guard is added in task 2.5 after its concurrency regression fails. Verify after seed wiring with `go -C backend test -count=1 -run '^TestSeedBootstrapAdmin' ./cmd/seed_dev` and `mise run backend:lint`.
- [x] 1.3 Extend seed authentication response projections to retain `userId`; carry environment mode and the existing `DATABASE_DSN` from configuration, and call the app entry point after authentication and before workspace creation. Close the database pool and sanitize connection diagnostics. Pass each test app's own DSN through seed fixtures without reading the developer's database. Verify with `go -C backend test -count=1 -run '^TestSeedBootstrapAdmin' ./cmd/seed_dev`.

## 2. Cover permissions, failures and repeatability

- [x] 2.1 Extend `TestSeedBootstrapAdmin` for an empty instance, a new dev account after another user, an existing ordinary admin, an already-correct root, and concurrent assignments for the same account. Check that exactly one bootstrap designation exists, actual transfers produce an audit entry, repeat assignments add none, the previous root remains `ADMIN`, and API role-management requests reflect both users' new powers with preexisting sessions. Verify with `go -C backend test -count=1 -run '^TestSeedBootstrapAdmin' ./cmd/seed_dev`.
- [x] 2.2 Cover a mismatched database with the same email but a different user ID, an email mismatch, an inactive or missing target, an unreachable database, and a non-development invocation. Induce final-grant and transfer-audit failures with narrowly targeted temporary database triggers in the test's isolated database; prove rollback preserves both roles and the old root and that no later dev items are created. Check diagnostics contain neither the connection string nor credentials. Verify with `go -C backend test -count=1 -run '^TestSeedBootstrapAdmin' ./cmd/seed_dev`.
- [x] 2.3 Adapt existing seed tests: member workspace restrictions now allow seeding after promotion; wrong passwords, closed registration, mandatory two-factor and failed health checks preserve privileges; later item-creation failure keeps the completed transfer; repeated runs preserve edited objects and backup settings. Retain the existing overlapping-instance and stale-agent scenarios. Verify with `go -C backend test -count=1 ./cmd/seed_dev ./internal/modules/users ./internal/app`.

- [x] 2.4 Add the controlled concurrency regressions from design decision 4 under `TestSeedBootstrapAdmin`: pause the final grant in the isolated database with a temporary trigger and advisory-lock barrier, send an overlapping demotion or deactivation of the dev account or administrator grant to a third account from the previous holder, and wait for PostgreSQL lock diagnostics before releasing the transfer. Run `go -C backend test -count=1 -run '^TestSeedBootstrapAdmin' ./cmd/seed_dev` before task 2.5 and confirm failure because stale authorization allows the forbidden update. Ensure barriers and triggers are cleaned up on failure and no fixed waits are used.
- [x] 2.5 Acquire the shared repository table lock at the start of `administerUser`, before its actor and target reads, and revalidate the actor's current active status and administrator role under it. Keep the existing bootstrap-protection checks and updates within that same transaction. Verify the task 2.4 regressions now refuse all three forbidden requests, preserve the active dev bootstrap administrator and do not commit a forbidden administration audit entry. Cover the reverse order where deactivation wins and the seed refuses the inactive target. Run `go -C backend test -count=1 ./cmd/seed_dev ./internal/modules/users ./internal/app` and `mise run backend:lint`.

## 3. Document and verify the completed change

- [x] 3.1 Update the seed package documentation to describe the local DSN requirement, bootstrap assignment, unchanged previous admin role, failure boundaries and continued use of the public API for ordinary seed items. Update affected backend README text and the bootstrap-field description, and search active documentation for obsolete promises that existing account roles never change. Leave historical OpenSpec changes and agent instructions alone. Verify the touched backend with `mise run backend:lint` and the change contract with `openspec validate ensure-dev-seed-root-admin --strict`.
- [x] 3.2 Run `mise run backend:format`, `mise run backend:lint` and the change-verification command below, resolve failures and finish with the required automatic implementation review. Verify with `mise run backend:lint` and `mise run test`; record their results in the handoff. Leave the completed change ready to commit without committing it.

## Verification evidence

- The initial member regression failed on the original code: seeding succeeded but the dev account remained `MEMBER` without the bootstrap designation.
- The controlled concurrency regressions failed before administration joined the lock protocol: stale requests demoted or deactivated the new bootstrap administrator, or let the previous holder grant another administrator role. All three scenarios pass with the shared lock.
- `go -C backend test -count=1 ./cmd/seed_dev ./internal/modules/users ./internal/app` passed. After consolidating the seed test runner, `go -C backend test -count=1 ./cmd/seed_dev` passed again. These direct commands used the Fedora library shim and disabled Ryuk as documented above.
- `mise run backend:format` completed and `mise run backend:lint` passed with zero issues.
- `mise run test` passed, including the seed tests and all 196 frontend test files with 2908 tests.
- `openspec validate ensure-dev-seed-root-admin --strict` and whitespace checks passed.

## Change verification

```sh
mise run test
```
