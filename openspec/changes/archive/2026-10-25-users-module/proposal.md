## Why

`users` is the lowest module in `docs/architecture.md`, and every later module depends on it. Today it is an old feature: global getters, controllers that answer errors in four shapes and leak internal error text, rate limits and the human-verification check inside the controllers, `time.Now()` everywhere, and no description of its operations. Milestone 4 of the backend rewrite plan moves it into `internal/modules/users`, the first module in the target layout, with its operations in the contract. This change is the last of four. It needs `lib/crypto` and `lib/mailer` from `lib-crypto-mailer`, `lib/audit`, `lib/clock` and `background.Runner` from `audit-in-lib`, and the `Actor` from `users-actor`.

## What Changes

- The module `internal/modules/users` appears with `module.go`, `Deps`, `New` and its code under `internal/`. It owns users, password sign-in with the emailed second factor, sign-up and the first-user rule, GitHub and Google sign-in, password reset, invitations, a user's own profile and password, user administration, instance settings, the test email and the console commands. Its services take the time from the clock, open transactions through the `database.Transactor` from `Deps` and write audit events through `lib/audit` in the same transaction, with the author passed explicitly.
- `module.go` grows from the `Actor` of `users-actor` into the module's contract. It offers other modules what they use today: looking users up by ID and by email, inviting a user, the two permission questions of `workspaces`, the user count for telemetry and the console commands for `internal/app`.
- 19 operations enter `backend/api/openapi.yaml` under the tag `users`, and the module serves them from a generated strict server. The contract gets a bearer security scheme, `x-access: public` on the operations that run without a session and `x-access: admin` on the administration operations.
- The users middleware enforces the access mark of each contract operation. A missing or invalid session answers 401 `unauthenticated`, and a user who is not an instance admin calling an admin operation gets 403 `permission_denied`. The same session check guards the routes of the old code, which keep their current groups.
- Every error of these operations is a `Problem` with a code. The contract gains the codes the frontend tells apart, among them `unauthenticated`, `permission_denied`, `rate_limited`, `human_verification_failed`, `invalid_credentials`, `user_deactivated`, `registration_closed`, `email_taken`, `invalid_reset_code` and `oauth_not_configured`. The five sign-in codes and the two test-email codes that the old controllers send today enter the contract with their current values and meanings.
- A failed password sign-in answers `invalid_credentials` alike for an unknown address, a user who has not finished sign-up, a user without a password and a wrong password. Today the answers differ and tell a caller which addresses have accounts. A deactivated user is named only after the correct password. A password reset request answers success for every address that is not rate limited, including addresses of users who are not active.
- The user list pages with a cursor (`limit`, `after`, `nextCursor`) and returns no total. The administration screen loads more users as before but no longer shows "N of M".
- Rate limits and the Cloudflare Turnstile check move from the controllers into the services. `util/ratelimiter` and `util/cache` move into `lib/ratelimit` and `lib/cache`, and their other callers switch imports. The Turnstile client moves inside the module.
- The cleanup of pending sign-ins and expired reset codes becomes the `background.Runner` job `users_expired_codes_cleanup` and loses its hand-written loop.
- The tables `password_reset_codes` and `two_factor_codes` become `users_password_reset_codes` and `users_pending_sign_ins`, because a row of the second table is a pending sign-in and its ID is the `pendingSignInId` of the contract. `users.role` and `users.status` get `CHECK` constraints, after the migration repairs any row outside the allowed values.
- The old code reaches the module through a bridge: `features/users` keeps only the installed `users.Module`, an admin check for the old routes, the remaining `WorkspaceRole` enum and the test helpers behind the `testhelpers` tag. `workspaces`, `telemetry`, the audit bridge and the console commands call the module.
- The frontend calls the 19 operations through the generated client and types. The shared client attaches the session token and handles 401 as `apiHelper` does today. The hand-written types of `entity/users` are deleted, and every new code gets a translation in the six interface dictionaries.
- The tests of `users` become module API tests through `apptest` and the contract-checking client. `apptest` gets helpers that create an administrator and a user who is not an admin through the API, a recorder for outgoing mail, and fakes for the OAuth providers and Turnstile.
- The api-design skill states that an operation that issues a credential, such as a session token, returns it once in its own response.
- **BREAKING** for users' scripts: the paths of the users operations change. For example, `POST /users/signin` becomes `POST /sessions`, `GET /users/me` becomes `GET /users/{userId}` and `GET /users/settings` becomes `GET /instance-settings`. Design decision 3 lists every path. The sign-in response becomes a `oneOf` with an `outcome` field. Errors are `Problem` bodies. The user list drops `offset`, `beforeDate` and `total`. The operation `POST /users/invite` is removed: no screen calls it, and `workspaces` invites users when it adds an unknown address to a workspace, through `Module.InviteUser`.
- **BREAKING** for the schema: two tables are renamed and two `CHECK` constraints are added.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `root-admin-account`: a failed password sign-in gives one answer whatever the reason before the password is verified, and a password reset request gives one answer whatever user the address belongs to.
- `api-http-server`: a contract operation answers 401 or 403 according to its access mark before its handler runs.

## Out of scope

- Changing the session mechanism. Tokens stay JWT signed with the master key for ten years and stored in `localStorage`, as the plan's list of things not done says.
- Moving the audit log views. They stay in the old code and move to `workspaces` in milestone 5, as `audit-in-lib` decides.
- Taking the audit author from the request context: every writer passes it explicitly.
- `WorkspaceRole`, memberships, the invitation email of `workspaces` and its join with `users`: milestone 5.
- Changing who may do what. Every permission rule keeps its current meaning.
- Moving the old routes of other features behind the contract guard. They keep their routes and groups until their modules move.
- Job states in the health endpoint: milestone 13.

## Impact

- Backend: new `backend/internal/modules/users`; new `backend/internal/lib/ratelimit` and `backend/internal/lib/cache`; `backend/api/openapi.yaml`; `backend/internal/lib/apierror` (codes and statuses); `backend/internal/lib/httpserver` (the access mark and the client IP in the context); `backend/internal/lib/apiclient` (the session token); `backend/internal/app` (wiring, routes, console, jobs) and `backend/internal/app/apptest`; `backend/internal/features/users` reduced to the bridge, the admin check, `WorkspaceRole` and the test helpers; `backend/internal/features/workspaces`, `features/telemetry`, `features/audit_logs`, `features/verification/agents`, the nine test routers that mount the session check, and the other callers of the rate limiter and the cache; deleted `backend/internal/util/ratelimiter`, `util/cache` and `util/cloudflare_turnstile`; `backend/cmd/seed_dev`; `backend/e2e/upgrade` (the sign-in of the current release).
- Frontend: `frontend/src/shared/api` (auth middleware of the generated client), `frontend/src/entity/users`, the screens of `features/users`, `features/settings` and `pages` that call these operations, the six dictionaries in `frontend/src/shared/i18n/locales`, and tests for the screens.
- Migrations: one migration renames two tables and adds the two `CHECK` constraints.
- Running instances: users, settings, pending sign-ins and reset codes keep their data. Sessions issued before the upgrade stay valid, because the token format and the key do not change.
- Documents: `docs/glossary.md` (user, session, pending sign-in, instance settings, sign-in options, invitation, password reset, access mark), `.agents/skills/api-design/SKILL.md` (credentials an operation issues), `temp/refactor/plan.md` (the checklist and the table of temporary relaxations). `docs/architecture.md` does not change: the module follows it. No README or website page shows a users operation.
- Rules the change answers to: the root `AGENTS.md`, `backend/AGENTS.md` and `frontend/AGENTS.md`.
