## Context

Why the change is needed is in proposal.md, section Why. This document covers only what shapes the approach. It assumes the three earlier changes of the milestone have landed: `lib/crypto` and `lib/mailer` (`lib-crypto-mailer`), `lib/audit`, `lib/clock`, `logging.With` and `background.Runner` (`audit-in-lib`), and `users.Actor` with the old code switched to it (`users-actor`).

**Operations.** `features/users` serves 21 operations from three controllers. They become 19 contract operations, see decision 3: `GET /users/me` and `GET /users/:id` merge into `getUser`, and `POST /users/invite` is dropped.

- `UserController` (`internal/features/users/controllers/user_controller.go:20`) serves nine public operations: sign-up, sign-in, the two second-factor steps, the account-existence check, the two password-reset steps and the two OAuth callbacks. It also serves four operations that need a session: read and update the profile, change the password and invite a user.
- `SettingsController` (`settings_controller.go:31`) reads and updates the instance settings and sends the test email.
- `ManagementController` (`management_controller.go:20`) lists users, reads one, activates and deactivates it, and changes its role.

The controllers check Turnstile (`controllers/turnstile.go:11`) and rate limits (`controllers/rate_limit.go:12`) themselves. Most errors go out as `{"error": err.Error()}` with status 400. Only the sign-in steps and the test email add a `code` (`user_controller.go:300`, `settings_controller.go:95`). Those seven codes exist only in the controllers' JSON: the contract's `ProblemCode` enum holds `invalid_request`, `not_found` and `internal` (`backend/api/openapi.yaml:73`). The user list pages with `limit`, `offset`, `beforeDate` and a total (`management_controller.go:40`), and the frontend shows "loaded N of M" (`frontend/src/features/users/ui/UsersComponent.tsx:375`).

**Sign-in refusals.** `UserService.SignIn` (`internal/features/users/services/user_services.go:145`) answers three different messages before it checks the password: no account (`:155`, `:161`), sign-up not finished (`:168`) and deactivated (`:174`). Then it answers "password is incorrect" (`:183`, `:190`). A caller can therefore tell which addresses have accounts. `SendResetPasswordCode` (`:520`) is silent for an unknown address but answers an error for an account that is not active (`:533`) and for an account past its hourly allowance (`:544`).

**Sessions.** `GetUserFromToken` (`:215`) verifies an HS256 JWT signed with the master key. It loads the user by `sub`, refuses a user who is not active (`:249`) and refuses a token whose `passwordCreationTime` claim differs from the user's (`:259`). `GenerateAccessToken` (`:271`) issues tokens for ten years. The middleware strips an optional `Bearer ` prefix, because the frontend sends the raw token (`frontend/src/shared/api/apiHelper.ts:82`). The frontend keeps the signed-in user's ID next to the token (`frontend/src/shared/api/accessTokenHelper.ts`).

**Time and transactions.** The users code calls `time.Now()` 25 times, including the expiry checks on the models (`models/two_factor_code.go`, `models/password_reset_code.go`). No service opens a transaction. Audit events go through the old bridge after the fact. The cleanup of pending sign-ins and expired reset codes is a hand-written loop every ten minutes (`services/background_service.go:40`).

**Who depends on `users`.** After `users-actor`, the old code uses these parts:

- `workspaces` calls `GetUserByEmail`, `GetUserByID`, `InviteUser`, `CanInviteUsers` and `CanCreateWorkspaces` (`internal/features/workspaces/services/membership_service.go`, `workspace_service.go`), and its membership repository joins `users` in SQL (`workspaces/repositories/membership_repository.go:55`);
- `telemetry` calls `GetUsersCount` (`internal/features/telemetry/service.go:102`);
- `internal/app/console.go` calls four console services;
- the audit bridge from `audit-in-lib` reads the author's email and name through the old users repository;
- the admin group of verification agents uses `RequireRole` (`internal/features/verification/agents/controller.go:23`);
- the routes use `AuthMiddleware` (`internal/app/routes.go:74`), and nine test routers build routers with it: `workspaces/testing/testing.go:32`, `tests/physical/postgresql/shared/helpers.go:218`, and the controller tests of `notifiers`, `databases`, `storages`, `verification/agents`, `verification/runs`, `audit_logs` and `backups/backups/controllers/physical`.

The tests of other packages call `users_testing.CreateTestUser` 656 times and use `RecreateInitialAdmin`, `DeleteAllUsers`, `ResetSettingsToDefaults` and the settings toggles. `cmd/seed_dev` (`seed.go:116`, `:133`) and the current-release half of the upgrade test sign in through `/users/signin`.

**The rate limiter and the cache.** `util/ratelimiter` builds on `util/cache`. Outside `users`, the rate limiter or the cache is used by `backups/backups/download` (including `restore_token` and `stream_guard`), `backups/backups/backuping/logical`, `restores/restoring` and `verification/agents`.

**The users tests.** 181 test functions in `features/users/controllers` and `features/users/services`, about 5,800 lines, drive the old controllers on a Gin router of their own.

**Contract plumbing.** The contract guard (`internal/lib/httpserver/contract_guard.go:19`) already finds the operation of each request, "because the access check reads the found operation's mark". `vacuum` accepts `x-access` only as `public` or `admin` (`backend/api/vacuum.yaml:140`). Its rule `databasus-path-plural-nouns` (`vacuum.yaml:76`) requires every resource segment to be a plural noun, optionally followed by a parameter and one segment after it. `apierror.Code` holds three codes and maps each to a status (`internal/lib/apierror/apierror.go:13`, `:55`). The generated strict handlers receive the `*gin.Context` as their `context.Context` (`internal/app/systemhttpapi/api.gen.go:244`). `app.NewHandler` builds the Gin engine (`internal/app/server.go:92`) without `ContextWithFallback`, so values put into the request context are invisible through the `*gin.Context`.

**Rules in force.** The module lands in the target layout and passes every linter and `archlint`: `module.go` alone in the root package, `Deps` and `New`, repositories with `database.Conn`, file roles, errors in `errors.go`, no package-level state. The handlers implement the strict server and hold no logic. The api-design skill says that a response never carries a secret. The frontend uses only types from `schema.gen.ts` and shows errors through `translateApiError`. Names follow the root `AGENTS.md` and `docs/glossary.md`: one word per concept, a name that states everything the unit does, a struct for two adjacent parameters of one type. The change answers to the root `AGENTS.md`, `backend/AGENTS.md` and `frontend/AGENTS.md`.

## Goals / Non-Goals

**Goals:**

- `features/users` holds only the bridge, `WorkspaceRole`, the admin check of the old routes and the test helpers. Every behavior of the 19 contract operations runs in the module.
- Every scenario of the old users tests has an API test through `apptest` before its old test is deleted.
- The old code keeps working unchanged in behavior. Its tests pass with only the edits this document lists.

**Non-Goals:**

- Moving the audit log views, memberships or anything else of `workspaces`.
- New product behavior beyond the two refusal rules of the `root-admin-account` delta and the access answers of the `api-http-server` delta.

## Decisions

### 1. The module's packages and its contract

```
internal/modules/users/
  module.go          Actor, Role, Status, User, Module, Deps, New, the console types
  internal/
    accounts/        the users table, profile, password change, invitations, administration, console commands
    signin/          sign-up and the first-user rule, password sign-in, the second factor, OAuth, sessions
    passwordreset/   reset codes
    settings/        instance settings, the test email, the two permission questions
    turnstile/       the Cloudflare Turnstile client
    httpapi/         the generated strict server and its handlers, the access middleware
```

`signin` and `passwordreset` depend on `accounts` and `settings`, and `accounts` depends on `settings`. Nothing depends back. The split follows the services that exist today: `UserService` holds sign-in, sessions, OAuth, reset and invitations in 1,167 lines, and the split separates them by what they own.

The contract calls the concept a user everywhere: the path `/users`, the operations `getUser` and `listUsers`, the field `userId`, and the glossary. `module.go` uses the same word. `Module` offers what the old code uses now and nothing more:

```go
type Module interface {
	GetAccessMiddleware() gin.HandlerFunc
	GetSessionMiddleware() gin.HandlerFunc
	RegisterRoutes(routes gin.IRouter, hooks *apierror.Hooks)
	GetJobs() []background.Job

	GetUser(ctx context.Context, userID uuid.UUID) (User, bool, error)
	GetUserByEmail(ctx context.Context, email string) (User, bool, error)
	InviteUser(ctx context.Context, actor Actor, invitation Invitation) (User, error)
	CanInviteUsers(ctx context.Context, actor Actor) (bool, error)
	CanCreateWorkspaces(ctx context.Context, actor Actor) (bool, error)
	GetUserCount(ctx context.Context) (int64, error)

	CreateUserFromConsole(ctx context.Context, request ConsoleUserRequest) (ConsoleUserOutcome, error)
	ChangePasswordFromConsole(ctx context.Context, change ConsolePasswordChange) error
	DisableTwoFactorAuth(ctx context.Context) (bool, error)
	RenderAdminListing(ctx context.Context) (string, error)
}
```

`User` holds `ID`, `Email`, `Name`, `Role`, `Status` and `CreatedAt`, never the password hash or the OAuth IDs. `ConsolePasswordChange` holds `Email` and `Password`, so the two strings cannot be swapped by position. `Deps` holds the database, the transactor, the clock, the logger, the master key, the mailer, the audit log, the rate limiter, and the settings the module reads from the configuration: the public URL, the OAuth client IDs and secrets with the providers' endpoints, and the Turnstile secret key with its verification URL.

*Rejected: one internal package for the whole module.* It would reproduce `UserService` with every dependency reachable from every function. *Rejected: a package per operation.* Sign-up, sign-in and OAuth share the first-user rule and session issuing, and one package per operation would export that shared code between siblings. *Rejected: `Account` as the name of the type.* The contract and the glossary say user, and two words for one concept is what the glossary forbids.

### 2. The guard records the access mark, and the users middleware enforces it

The contract guard already finds the operation. It now also puts the operation's `x-access` value into the request context, and `httpserver.GetAccessMark(ctx)` reads it: `public`, `admin` or empty for "signed in". `GetAccessMiddleware` runs right after the guard on the contract group:

- public: the request passes, and a token is ignored;
- signed in: a missing or invalid session answers 401 `unauthenticated`;
- admin: like signed in, and a role other than admin answers 403 `permission_denied`.

On success it puts the `Actor` into the context with `users.WithActor` and the user ID into the logging context with `logging.With`.

`GetSessionMiddleware` is the same session check without the mark. `app` mounts it on the protected group of the old routes, where `AuthMiddleware` sits now. The old routes keep their groups, so the same routes require a session as before. Their 401 answer becomes a `Problem` body. The frontend reacts to the status alone (`apiHelper.ts:10`), so nothing there changes. The admin group of verification agents gets `RequireAdmin()` from the `features/users` bridge in place of `RequireRole`. It reads the actor with `users.GetActor` and answers 403 for a non-admin, and it goes away when verification agents move in milestone 12.

The session check keeps today's rules, which the Context section lists, including the optional `Bearer ` prefix. A user who is not active, invited or deactivated, has no valid session. The generated client sends `Bearer`, and the prefix rule keeps scripts that send the raw token working.

`app.NewHandler` sets `ContextWithFallback` on the engine it builds. The strict handlers then read the actor, the logging attributes and the client IP through the `*gin.Context` they receive. A new `lib/httpserver` middleware, `RememberClientIP`, mounted by `app.NewHandler` next to the request ID middleware, puts the client IP into the request context. Turnstile verification needs it, and the services see only a context.

*Rejected: a per-operation middleware list in the generated server.* The mark would live in two places, in the contract and in the Go wiring, and a forgotten entry would expose an admin operation. The guard reads the contract itself, so an operation is protected by the line that describes it. *Rejected: keep `RequireRole` on admin routes.* A route mounted by mistake would run without it, which is exactly what the guard exists to prevent.

### 3. The operations follow the api-design rules, and their paths change

| Old | New | `operationId` | Access |
|---|---|---|---|
| `POST /users/signup` | `POST /users` | `signUp` | public |
| `POST /users/signin` | `POST /sessions` | `signIn` | public |
| `POST /users/verify-signin-code` | `POST /pending-sign-ins/{pendingSignInId}/verify` | `verifySignInCode` | public |
| `POST /users/resend-signin-code` | `POST /pending-sign-ins/{pendingSignInId}/resend` | `resendSignInCode` | public |
| `POST /auth/github/callback` | `POST /github-sessions` | `signInWithGitHub` | public |
| `POST /auth/google/callback` | `POST /google-sessions` | `signInWithGoogle` | public |
| `GET /users/is-any-user-exist` | `GET /sign-in-options` | `getSignInOptions` | public |
| `POST /users/send-reset-password-code` | `POST /password-resets` | `requestPasswordReset` | public |
| `POST /users/reset-password` | `POST /password-reset-completions` | `completePasswordReset` | public |
| `GET /users/me` | `GET /users/{userId}` | `getUser` | signed in |
| `PUT /users/me` | `PUT /users/{userId}` | `updateUser` | signed in |
| `PUT /users/change-password` | `PUT /users/{userId}/password` | `changeUserPassword` | signed in |
| `GET /users` | `GET /users` | `listUsers` | admin |
| `GET /users/:id` | `GET /users/{userId}` | `getUser` | signed in |
| `POST /users/:id/activate` | `POST /users/{userId}/activate` | `activateUser` | admin |
| `POST /users/:id/deactivate` | `POST /users/{userId}/deactivate` | `deactivateUser` | admin |
| `PUT /users/:id/role` | `PUT /users/{userId}/role` | `changeUserRole` | admin |
| `GET /users/settings` | `GET /instance-settings` | `getInstanceSettings` | signed in |
| `PUT /users/settings` | `PUT /instance-settings` | `updateInstanceSettings` | admin |
| `POST /users/settings/test-email` | `POST /test-emails` | `sendTestEmail` | admin |

Every path fits `databasus-path-plural-nouns`, so `vacuum` needs no exception. The description of each public operation states why it runs without a session, as the api-design skill asks: it creates or completes a session, or it serves the sign-in page. The current user's profile is `/users/{userId}` with the caller's own ID, which the frontend already keeps. `getUser` serves the caller's own profile and an administrator's view of any user. `updateUser` and `changeUserPassword` accept only the caller's own ID and answer 403 `permission_denied` for another user, as the old operations did by having no ID at all. `getUser` stays signed in, because a non-admin may read their own user and the service keeps answering 403 for anyone else's. `POST /users/invite` gets no successor. No screen calls it, and the only invitation flow the product offers is adding an unknown address to a workspace, which `workspaces` serves through `Module.InviteUser` with the same permission rule.

Body changes:

- `signIn` answers a `oneOf` discriminated by `outcome`: `COMPLETED` with `userId`, `email` and `token`, or `CODE_REQUIRED` with `pendingSignInId` and `email`. Today the client tells them apart by which fields are present (`frontend/src/entity/users/api/readSignInOutcome.ts`), and the contract cannot describe that without a discriminator.
- `getSignInOptions` answers `{hasAnyUser}` instead of `{isExist}`. The sign-in page uses it to offer the first registration.
- `requestPasswordReset` takes `email` as a plain string without `format: email`. The placeholder `admin` of an upgraded instance therefore gets the same answer as any other address and receives no code, as the `root-admin-account` delta requires.
- `Module.InviteUser` takes the actor and an `Invitation` with the email and the workspace, keeps the workspace on the audit event as today, and answers the invited user. It drops the old echo of `intendedWorkspaceRole`, which nothing stored.
- `listUsers` takes `limit`, `after` and `query` and answers `items` and `nextCursor`, ordered by creation time, newest first. The cursor encodes the creation time and the ID of the last user. It lives in the module until a second module needs one.
- Passwords are `writeOnly`. The session token is the credential that `signIn`, `verifySignInCode`, `signUp` and the two OAuth operations issue, and each returns it once in its own response. The api-design skill says that a response never carries a secret. This change adds the exception to the skill: a credential the operation itself issues, such as a session token or a single-use download token, is returned once in that operation's response and never in any other.

*Rejected: keep the old paths in the contract.* `vacuum` enforces the path rules of the api-design skill, so the old paths would need exceptions in the rule set that every later module could copy. The architecture allows a breaking release of `/api/v1` when the frontend changes in the same change, and the proposal marks it for the release notes. *Rejected: `/users/me` for the current user.* `me` is neither a plural noun nor a parameter, so it needs a `vacuum` exception, while `/users/{userId}` with the caller's ID describes the same resource.

### 4. Codes and statuses

`apierror.Code` grows to the codes below, and `describeCode` maps each code to one status. A test keeps the Go list equal to the contract's `ProblemCode` enum, as it does now. The seven codes that the old controllers already send keep their values and their meanings and enter the contract for the first time.

| Code | Status | Raised by |
|---|---|---|
| `unauthenticated` | 401 | access middleware |
| `permission_denied` | 403 | access middleware, another user's profile or password, administration rules such as deactivating yourself |
| `rate_limited` | 429 | every rate limit of the module; replaces `rate_limit_exceeded` |
| `human_verification_failed` | 400 | missing or rejected Turnstile answer |
| `invalid_credentials` | 401 | password sign-in, see decision 5 |
| `user_deactivated` | 403 | password sign-in after the correct password, OAuth sign-in to a deactivated user |
| `registration_closed` | 403 | sign-up and OAuth sign-up when external registration is off |
| `email_taken` | 409 | sign-up and profile update with an address in use |
| `invalid_reset_code` | 400 | wrong, used or expired reset code |
| `oauth_not_configured` | 409 | OAuth sign-in when the provider has no client ID or secret |
| `oauth_failed` | 400 | the provider refused the code or returned no usable email address |
| `pending_sign_in_not_usable` | 410 | unchanged |
| `sign_in_code_not_sent` | 503 | unchanged |
| `too_many_sign_in_codes` | 429 | unchanged |
| `sign_in_code_resent_too_soon` | 429 | unchanged |
| `sign_in_code_incorrect` | 400 | unchanged |
| `email_not_configured` | 409 | test email without a mail server; was 400 |
| `admin_email_missing` | 409 | test email to a user without a deliverable address; was 400 |
| `not_found` | 404 | a user ID that does not exist |
| `invalid_request` | 400 | everything the contract or a validation rule refuses, with `errors` naming the fields |

Two sign-in codes share 429, and three statuses carry more than one code. That is why the frontend reads the code, as it does today. `oauth_not_configured` moves from 501 to 409: the operation exists, and the instance's state forbids it.

Every service error that the client must tell apart is an `*apierror.Problem` built in the module's `errors.go`. Anything else becomes 500 `internal` through the existing hooks.

*Rejected: one code per old message.* Several old messages describe the same situation for the client, such as the six "insufficient permissions" texts. Those collapse into `permission_denied`, and `detail` keeps the specific text for logs and scripts.

### 5. Password sign-in refuses alike until the password is verified

`SignIn` loads the user by address and decides in this order:

1. no user, an invited user, or a user without a password: one bcrypt comparison against a fixed hash, then `invalid_credentials`;
2. a wrong password: `invalid_credentials`;
3. a deactivated user: `user_deactivated`;
4. otherwise the sign-in completes or asks for a code.

Every case of step 1 runs the same bcrypt comparison as step 2, so the response time does not single them out either. The `root-admin-account` delta specifies only the status and the code, which tests can observe. The equal timing is a hardening measure without a spec scenario. The log keeps each reason as today.

`requestPasswordReset` answers 204 for an unknown address and for a user who is not active, and only an active user gets a code. The rate limit per address stays in front and answers `rate_limited` alike for every address. The service keeps its own allowance of three codes per user an hour behind that limit. With equal numbers, a request reaches it only after the in-memory limiter lost its counts in a restart, and then it answers 204 without sending a code instead of today's error. No spec scenario covers that case, because the API cannot reach it within one process.

*Rejected: keep the distinct answers and give each its own code.* A code is a stable promise to clients, so it would turn the leak into part of the contract. The `root-admin-account` spec already requires sign-in to disclose nothing about the accounts an instance holds.

### 6. Rate limits and Turnstile live in the services

Each service method that today sits behind `checkRateLimitOrRespond` or `verifyTurnstileOrRespond` calls the limiter or the Turnstile client itself. The scopes and limits are the same:

- sign-in: 10 per minute per address;
- code verification: 10 per minute per pending sign-in;
- reset request: 3 per hour per address;
- test email: 5 per 10 minutes per administrator.

A limiter error still refuses the request, as the `local-rate-limiting` spec requires. The Turnstile client takes the client IP from the request context (decision 2) and its URL from `Deps`.

`util/ratelimiter` and `util/cache` move to `lib/ratelimit` and `lib/cache`. They are listed in `docs/architecture.md`, have no domain logic, and the module may not import `util`. Their package-level instances become values built in `app`. The old callers listed in the Context get them through bridge getters, as `lib-crypto-mailer` does for the master key. The Turnstile client has one consumer, so it moves into the module.

*Rejected: a rate-limit middleware per operation.* The identifier is the address or the pending sign-in ID from the body. A middleware would decode the body before the strict server does, and the limit would live apart from the rule it protects.

### 7. Time comes from the clock, and writes run in transactions with their audit event

Every expiry check takes `now` from the clock, and queries take it as a parameter. The model methods that call `time.Now()` become checks with a `now` argument. Sign-up and OAuth sign-up, the second-factor steps, the reset completion, invitations, administration changes and settings updates each run in one transaction of the `database.Transactor` from `Deps` together with their `auditLog.Record`. The author is the user the service already loaded. Sending mail stays outside the transaction, as today, so a slow mail server does not hold a row lock.

The cleanup loop becomes the job `users_expired_codes_cleanup` with the same ten-minute interval. Each pass deletes pending sign-ins past their retention and expired password reset codes. `GetJobs` returns it, and `app` registers it.

### 8. The old code reaches the module through a bridge

`features/users` keeps four things:

- `Install(users.Module)` and `GetModule()`. `app.New` installs the module it built. In a test build, `loadWithoutApp` builds the module from the old access points: the database, the configuration, the master key, the mailer and the audit log bridges.
- `RequireAdmin()` for the admin group of verification agents, as decision 2 describes.
- `enums.WorkspaceRole`, until milestone 5.
- `features/users/testing` behind the `testhelpers` tag.

The callers change:

- `workspaces` calls `GetUser`, `GetUserByEmail`, `InviteUser`, `CanInviteUsers` and `CanCreateWorkspaces`;
- `telemetry` calls `GetUserCount`;
- the console commands call the module's console methods;
- the audit bridge calls `GetUser` for the author;
- the routes mount `GetSessionMiddleware()`, and the nine test routers of the Context get it through `users_testing.GetSessionMiddleware()`.

The membership repository of `workspaces` keeps its SQL join with `users`. The main table keeps the name `users`, so the join works until milestone 5. `temp/refactor/plan.md` records that break of the "no join with another module's tables" rule.

The test helpers keep their signatures, so the 656 calls do not change. `CreateTestUser` inserts the user with SQL, as it does today, and signs its token with the master key from the bridge. It uses the same claims the module issues: `sub`, `exp`, `iat`, `role` and `passwordCreationTime`. The settings toggles and `RecreateInitialAdmin` stay SQL on the module's tables. Writing a module's tables from outside the module breaks the data ownership rule of `docs/architecture.md`, and `temp/refactor/plan.md` records it as a temporary relaxation until the old tests move to `apptest`, in milestone 13 at the latest.

*Rejected: add `IssueSessionToken` to `Module` for the helpers.* Only tests would call it, and `module.go` would then describe a contract that production never uses. *Rejected: create test users through the sign-up and sign-in operations.* Each of the 656 calls would pay two bcrypt rounds at the default cost, and the old tests do not run an HTTP server. The price of the chosen way is that the claim names live in two places. A drift shows at once, because every old test that signs in would get 401.

### 9. The test application can sign users up and observe mail

`apptest` gets the following, all built through the public operations except where stated:

- `CreateAdmin(t)`: the first sign-up of a fresh test database claims the instance;
- `CreateNonAdminUser(t)`: the administrator allows external registration through `updateInstanceSettings`, and a second user signs up;
- each returns a session with a contract-checking client that sends its token;
- `apiclient.New` gets an option for the token;
- `Mail`: a recorder that `app.Environment` takes in place of `lib/mailer`, so a test reads the sign-in and reset codes;
- fake GitHub, Google and Turnstile endpoints on `httptest` servers, which a test turns on through options of `apptest.New`.

States that need a later operation are built through that operation in the test of the task that adds it: an invited user through the old workspace member endpoint, which calls `Module.InviteUser` for an unknown address, a deactivated user through `deactivateUser`, a user without a password through GitHub sign-up.

`app.Environment` gets the mailer and the external endpoints as fields. `cmd/databasus` fills them with `lib/mailer` and the providers' real URLs.

The old controller tests turn into API tests in the module's root package. Each old scenario maps to a new test or to an `apptest` helper that covers it. Scenarios that only tested private functions are covered through the operation that uses them. The console tests of `internal/app` stay where they are and call the module through `app`.

### 10. The frontend calls the generated client with the session

`shared/api/apiClient.ts` adds an openapi-fetch middleware. It sets `Authorization: Bearer <token>` from `accessTokenHelper`, and on a 401 it clears the token and reloads, as `apiHelper` does. `entity/users/api` keeps its function names and implements them on the generated client, so the screens change only where the bodies change:

- the sign-in outcome;
- the current user read and updated by the stored user ID;
- the cursor of the user list and the removed counter;
- `hasAnyUser`;
- the error handling, which goes through `translateApiError`.

The hand-written request and response types of `entity/users/model` are deleted. Only the role label maps stay.

Every new code gets a key in `translateApiError` and a translation in `en`, `ru`, `zh`, `pt`, `fr` and `es`. The type check fails on a missing one, as the `frontend-localization` spec requires. `rate_limit_exceeded` is removed, because only `users` emitted it.

Screen tests run on `mockApi`:

- sign-in for both outcomes and for `invalid_credentials`;
- the code screen for `sign_in_code_incorrect`, `sign_in_code_resent_too_soon` and `pending_sign_in_not_usable`;
- the user list loading the next page with `nextCursor`;
- the test email for `email_not_configured`.

### 11. New terms enter the glossary

The contract and `module.go` bring terms the glossary does not define: user, session, pending sign-in, instance settings, sign-in options, invitation, password reset and access mark. Each gets a row in `docs/glossary.md` in this change, with the word the contract uses.

## Risks / Trade-offs

- [Scripts that sign in break on upgrade] → The proposal marks the path change as breaking for the release notes. The new paths follow one rule, so a script needs one mapping, and decision 3 is that mapping.
- [The change is large] → The module, its tests and the frontend move one operation group at a time: sign-up and sign-in, settings, second factor, OAuth, profile and invitations of `Module`, administration, password reset. Each group removes its old routes and tests in the same task. Two routes on one path never coexist, and Gin would refuse them at startup anyway.
- [The old test helpers sign tokens on their own] → Decision 8 explains why. A token-format drift fails every old test that signs in, so it cannot pass unnoticed.
- [An administrator loses the user total] → The list still loads every page. The total contradicts cursor paging, which the api-design skill requires for every list.
- [`ContextWithFallback` changes what `gin.Context.Value` returns] → Only the strict handlers rely on it. The old controllers already read the request context explicitly. A test checks that a strict handler sees the actor.

## Migration Plan

One migration, in its transaction:

1. rename `password_reset_codes` to `users_password_reset_codes` and `two_factor_codes` to `users_pending_sign_ins`, together with their indexes and constraints. A row of the second table is a pending sign-in: its ID is the `pendingSignInId` of the contract, and the glossary term is "pending sign-in";
2. repair `users.role` outside `ADMIN` and `MEMBER` to `MEMBER`, and `users.status` outside `INVITED`, `ACTIVE` and `INACTIVE` to `INACTIVE`, so a damaged row loses rights instead of stopping the upgrade;
3. add `CHECK` constraints on both columns.

Sessions stay valid across the upgrade: the token format, the claims and the master key do not change. The upgrade test signs in through the new `POST /sessions` with the password set on the previous release.
