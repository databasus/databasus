## Context

See proposal.md for the motivation. The state this design starts from:

- `GET /api/v1/system/version` is served by the old `system_version` controller, which writes `{"version": GetAppVersion()}` with `ctx.JSON` (`backend/internal/features/system/version/controller.go:15-30`). `GetAppVersion` reads `APP_VERSION` and falls back to a fixed default. The verification agent calls this path (`agent/verification/internal/features/api/constants.go:6`), and so does the frontend (`frontend/src/entity/system/api/systemApi.ts:6`, used by `shared/hooks/useVersionCheck.ts:19`).
- `app.newRouter` (`backend/internal/app/server.go:76-100`) calls `registerRoutes`, which registers every old controller on one `/api/v1` group (`backend/internal/app/routes.go:28-68`), and registers `httpserver.HandleUnknownPath` through `router.NoRoute`. That handler answers an unknown `/api/` path with `{"error": "not found"}` (`backend/internal/lib/httpserver/middleware.go:70-76`). `App.Serve` builds the router and starts the background loops together (`backend/internal/app/server.go:33-100`).
- `app.New` installs the old code's process-wide bridge (`old_config.Install`, `storage.Install`) and calls `setUpDependencies`, which registers listeners (`backend/internal/app/app.go:50-60`). Calling it twice in one process registers every listener twice.
- The frontend throws `ApiError` from `apiHelper` and shows errors only through `translateApiError`, which maps a `code` to a dictionary key (`frontend/src/shared/i18n/translateApiError.ts:9-36`). Frontend tests run in the `node` environment without network mocks (`frontend/vitest.config.ts:12`).
- `backend/.golangci.yml` already allows test libraries in `internal/lib/apiclient` and `internal/app/apptest` (`:69-82`) and names the target layout (`:131`). `importas` is off.
- Cloud (`/home/rostislav/development/project/cloud`) builds the same pipeline: per-tag `oapi-codegen` configs next to a `generate.go`, a freshness check in its backend `Justfile`, `contractcheck` on `kin-openapi`, `openapi-typescript` through `pnpm dlx`, and `mockApi` on `msw`.

The change answers to the root `AGENTS.md`, `backend/AGENTS.md`, `frontend/AGENTS.md`, the `api-design` and `testing` skills, and `docs/architecture.md` ("Backend layout", "HTTP API", "Automatic checks").

### Development tooling

- The root `Justfile` holds the cross-module recipes: `run` (`Justfile:3`), `test`, `pull-testcontainers`, `generate-api`, `check-generated`, `lint`, `lint-shell`, `doctor`, `format` and `test-filesystem`. It delegates to the modules through `just --justfile backend/Justfile ...`.
- `backend/Justfile` holds `lint` with its dependency `check-generated` (`backend/Justfile:17`), `format`, `generate-api`, `test-package` (`:45`), the CI sharding recipes `test-group-packages` (`:55`) and `test-group-tests` (`:75`), `test-upgrade` (`:120`), `clean-testcontainers` and `pull-testcontainers`. Several of them are bash scripts of twenty lines or more. `agent/verification/Justfile` holds `run`, `build`, `test`, `lint`, `format`, `e2e`, `e2e-podman` and `e2e-clean`.
- The recipes `generate-api`, `check-generated` and the `vacuum` line of the backend `lint` come from task groups 1 to 5 of this change, which were built on just.
- `lefthook.yml` formats staged Go and frontend files with `stage_fixed: true` (`lefthook.yml:15`, `:27`, `:39`), then runs the module's `just lint && go mod tidy -diff` (`:17`, `:29`) or `pnpm lint && pnpm typecheck`. It starts itself through `mise exec` (`lefthook.yml:2`), because Git clients in an IDE run hooks without a mise-activated shell.
- CI lints the backend and the verification agent with `actions/setup-go` and `go install golangci-lint@v2.11.3`, then calls `golangci-lint run` directly (`.github/workflows/ci-release.yml:34-62`, `:202-230`). It runs neither `golangci-lint fmt --diff` nor the contract checks this change adds. The frontend lint job calls pnpm scripts one by one (`:91-114`). Seven steps call `just`, among them the CI-only sharding at `:449-450`.
- `.agent-hooks/review_gate.py:172-211` accepts exactly three command forms in a change's verification block, two of them `just` forms, and checks every argument against a character set, because `just test-package` interpolates its arguments into a bash recipe. `.agent-hooks/run.sh:12` and the hook messages point to `just doctor`.
- mise 2026.10.6 is the installed version. hk 2.8.1 is the current release, and it evaluates `hk.pkl` with a built-in Pkl evaluator, without the `pkl` CLI.
- The archived change `migrate-dev-tooling-to-just` (`openspec/changes/archive/2026-10-10-migrate-dev-tooling-to-just/design.md`) chose just to mirror the team's other project. It recorded no reason against mise tasks.
- `scripts/test.sh` itself calls two backend recipes: `just --justfile "$root/backend/Justfile" pull-testcontainers` (`scripts/test.sh:76`) and `... test-upgrade` (`scripts/test.sh:96`). `scripts/test_sh_test.py` stubs `just` to observe them.
- `docs/architecture.md` "Automatic checks" names the root `just check-generated` in its last table row (`docs/architecture.md:275`) and `just lint` at the root and in `backend/` in the paragraph below (`:277`).

The tooling part answers to the root `AGENTS.md` (the development environment section, the command table rules and "Other agents' changes in the working tree"), `backend/AGENTS.md`, `frontend/AGENTS.md` and `agent/verification/AGENTS.md`, all of which name commands. No architecture rule changes. Only the names of the commands in `docs/architecture.md` change: the row at `:275` names `backend:check-generated`, which `backend:lint` runs, and `check:api`, which `frontend:lint` runs, and the paragraph at `:277` names `mise run lint` and `mise run backend:lint`.

## Goals / Non-Goals

**Goals:**

- Every step the next milestones repeat per module works on one real operation: contract, rules, generation, strict server, `Problem`, contract-checked test through `apptest`, frontend call and mock.
- Installed verification agents see no difference on `/system/version`.
- One definition per check. A developer, an agent, the commit hook and CI run the same task.
- The commit hook cannot touch changes that are not staged.
- Every command keeps its behavior, so only the names change for the people who call them.

**Non-Goals:**

- `x-access` enforcement, auth on the frontend client, request body validation, `apptest` fakes. Each arrives with its first consumer, as the proposal lists.
- Changing what a check checks. The single deliberate additions are `golangci-lint fmt --diff` and the API contract checks in CI, which come from running the local lint task there, and `go mod tidy -diff` moving from the hook into the module lint task.
- Making `scripts/test.sh` or `scripts/dev.sh` do anything new beyond calling the new task names.

## Decisions

### The contract and its tools

- `backend/api/openapi.yaml` uses OpenAPI 3.0.3, the version `oapi-codegen` supports fully. `backend/api/api.go` embeds it as `api.OpenAPI`. Error codes are an enum `ProblemCode` with `invalid_request`, `not_found` and `internal`, the codes this change can return. They are lower `snake_case`, as the error table of the `api-design` skill prescribes, while its "Fields" section says enum values are `UPPER_SNAKE_CASE`. This change resolves that contradiction in the skill: error codes are named as an exception there, and the `vacuum` casing rule skips `ProblemCode`. `Problem` has `type`, `title`, `status`, `detail`, `code` and an optional `errors` list for field errors, as the `api-design` skill describes.
- `oapi-codegen` is a `tool` dependency in `backend/go.mod`, so `go tool oapi-codegen` runs the pinned version without a separate install. `vacuum` is pinned in `mise.toml`, and its rules live in `backend/api/vacuum.yaml`. They port cloud's ruleset, adjusted to the `api-design` skill: a default `Problem` response, a tag and an operation ID on every operation, `camelCase` fields, and no `x-access` value other than `public` and `admin`. The plural-noun path rule lists `/system/version` and `/system/verification-agent` as exceptions, because those paths never change.
- `mise run generate-api` runs `go generate ./...` in `backend/` and `pnpm generate:api` in `frontend/`. `pnpm generate:api` runs `openapi-typescript` through `pnpm dlx` at a pinned version, as cloud does, so the generator adds nothing to `pnpm-lock.yaml`.
- `backend:lint` gains `vacuum lint` and `backend:check-generated`, the freshness check ported from cloud's `_check-generated`. Each config is regenerated into a temporary file and compared with the committed output. `frontend:lint` gains `check:api`, which regenerates `schema.gen.ts` and compares it. The root `mise run lint` runs both through the module lints.

Rejected alternative: one generator config for the whole contract. Operations are assigned to modules by tag (`docs/architecture.md` "HTTP API"), so each server package generates only its own tag from the first operation on.

### The version operation lives in `internal/app/systemhttpapi`

The `system` tag is served by `internal/app/systemhttpapi`: `oapi-codegen.yaml`, `generate.go`, the generated `api.gen.go` and `handler.go`. The handler implements the strict interface and takes the version string in its `Deps`. `docs/architecture.md` "HTTP API" and the `api-design` skill ("Contract first") say that the tag names the module and the server lives in the module's `internal/httpapi`. This change adds the one exception to both: the `system` tag is served by `internal/app/systemhttpapi`, because `system` dissolves into `app` and is not a module. `app` passes `system_version.GetAppVersion()`, so the fallback for an unset `APP_VERSION` stays one function shared with the agent download and telemetry. The old route is removed from `routes.go`, and the `VersionController` type, its DTO and its `di.go` are deleted. `GetAppVersion` stays in the `system_version` package until milestone 13 dissolves `system` into `app`.

Rejected alternatives:

- A demo module like cloud's `ping`. It would stay in production as dead code, and it would prove nothing about a path that real clients depend on.
- A module `system`. Milestone 13 dissolves `system` into `app`. A module created now would be torn down there.

### `Problem` comes from `lib/apierror`

`lib/apierror` holds `Code` constants that mirror the contract enum, `Problem` and `Write(ctx, problem)`, which sets `application/problem+json` and fills `type`, `title` and `status` from the code. `apierror` builds the three error hooks of a generated Gin server: the strict server's request and response error handlers, and `GinServerOptions.ErrorHandler`, which receives malformed path and query parameters from the non-strict wrapper and would otherwise write its own `{"msg": ...}` body. A decoding or parameter failure becomes 400 `invalid_request`. Any other error is logged with its text and answered as 500 `internal`, with the fixed detail "an internal error occurred". `httpserver.HandleUnknownPath` answers 404 `not_found` through `apierror`. A test in `apierror` checks that the Go constants equal the contract enum, so the two cannot drift.

### The contract route guard

`lib/httpserver.RequireContractOperation(contract)` loads the embedded contract into a `kin-openapi` router once. For each request it looks up the operation by method and path. When there is none, it logs the method and the path and answers 500 `internal` without calling the handler. `app` mounts every generated server on a `/api/v1` group that carries this middleware. Old controllers stay on the plain group and are not checked. A row in the plan's table of temporary relaxations tracks that gap.

Milestone 4 reuses the found operation: the `users` middleware reads `x-access` from it. That is why the guard runs per request instead of comparing routes once at start-up.

Rejected alternative: a start-up check that every route of the contract group appears in the contract. It catches the same mounting mistake earlier, but it does not give the request the operation that the access check needs next.

### `apiclient` checks every response

`internal/lib/apiclient` holds the generated client (`oapi-codegen` with `client: true`) and `New(t, baseURL)`. `New` wraps the HTTP client with a doer that validates each response against `api.OpenAPI` through `openapi3filter.ValidateResponse`, and calls `t.Errorf` with the operation, the status and the validator's message on a mismatch. The validation code stays inside `apiclient` until `agentclienttest` needs the same doer in milestone 12. It moves to a shared package then.

### `apptest` builds the real application, one instance at a time

`internal/app/apptest.New(t)` does the following:

1. builds an `app.Environment` with a database from `testdb.Open(t)`;
2. calls `app.New`;
3. serves `App.NewHandler()` through `httptest.NewServer`;
4. returns `*apptest.App`, whose `URL` field holds the server address. Tests call `apiclient.New(t, testApp.URL)`. Milestone 4 adds the clock and the fakes to the same struct, so callers keep their signature.

`App.NewHandler()` is split out of `Serve`, so the server and the test share one router, and the test starts no background loop. `Serve` keeps opening the port and starting the loops.

While old code is wired through process-wide globals, these rules apply:

- `setUpDependencies` runs once per process through `sync.Once` inside `app.New`. The console commands and the server call `app.New` once, so for them nothing changes.
- Each `app.New` still installs its config and database into the bridge. The repositories of old code are empty structs that call `storage.GetDb()` on every query, so they follow the bridge to the current test's database.
- The data directory of the old code is created once per process under `os.TempDir()`. Lazy singletons in `di.go` may copy a path from the configuration when they are first built, and a per-test `t.TempDir()` would be deleted under them after the first test.
- `apptest.New` holds a process-wide mutex until the test's cleanup. A second `New` while one instance is alive fails the test with a message saying that `apptest` tests of one package run one at a time until milestone 13. The rule is therefore enforced, not merely documented.

Rejected alternative: one database per process for `apptest`, shared by all its tests. Tests would then see each other's data. Module tests in milestone 4 would depend on their order, and `apptest` would have to change again when old code is gone.

### Frontend client and mocks

- `shared/api/apiClient.ts` creates the `openapi-fetch` client with `baseUrl` from `getApplicationServer()` plus `/api/v1`. `shared/api/requireData.ts` turns a `{ data, error, response }` result into the data, or into an `ApiError` with the `Problem`'s `code`, `detail` and `status`. A result without a parseable `Problem` becomes `request_failed`, as in `apiHelper`. The auth header middleware waits for the first authenticated operation.
- `systemApi.getVersion` calls `GET /system/version` through the client with `cache: 'no-store'`, and returns the generated response type. `entity/system/model/VersionResponse.ts` is deleted, and `entity/system/index.ts` re-exports the generated response type under the same name, so `VersionResponse` keeps one meaning.
- `shared/lib/test/mockApi.ts` follows cloud's helper on `msw/node`: a typed `mockApi.get(path, status, body)` whose status and body come from `paths` in `schema.gen.ts`. `msw` becomes a dev dependency.
- `translateApiError` maps `not_found`, `internal` and `invalid_request` to new keys under `errors` in all six dictionaries. The contract codes get their own table, declared with `satisfies Record<components['schemas']['ProblemCode'], TranslationKey>`. `openapi-typescript` emits only types, so the check lives in the type checker: a new contract code without a key makes `pnpm typecheck` name the missing property.

### `importas` aliases

`importas` is enabled with `no-unaliased: true` for these patterns:

- `databasus-backend/internal/modules/(\w+)/internal/httpapi` as `${1}httpapi`;
- `databasus-backend/internal/app/systemhttpapi` as `systemhttpapi`;
- `databasus-backend/internal/lib/apiclient` as `apiclient`;
- `databasus-backend/api` as `contract`.

The last alias avoids reading `api.OpenAPI` next to the `httpapi` packages. In `docs/architecture.md` "Automatic checks", the sentence on what arrives later is rewritten to keep only `gochecknoglobals`, which still arrives once no package-level dependencies remain. `docs/testing.md` loses the sentence saying that `mockApi` arrives with the contract.

### The glossary gains the error terms

`docs/glossary.md` "Code" gains `Problem` (the RFC 9457 error body of every contract operation), `Error code` (`code`, the machine-readable value the frontend translates) and `Contract-checking test client` (`apiclient`, which fails a test on a response that differs from the contract).

### Every task lives at the root, with module prefixes

`mise.toml` holds the short tasks. A script longer than a few lines becomes a file task under `mise-tasks/`, and mise names a file in a subdirectory after its path: `mise-tasks/backend/test-group-packages` becomes `backend:test-group-packages`. Every module task sets its directory to the module (`dir = "backend"` in TOML, `#MISE dir="backend"` in a file task), so `mise run backend:test-package ./internal/...` resolves the packages in `backend/` from any directory. Task arguments are declared with a `usage` spec, a package list as a variadic argument, and scripts read them from `$usage_<name>`.

The root `lint` and `format` depend on the module tasks, and mise runs the dependencies in parallel. `lint` must report every failing module. The first task therefore checks whether mise cancels the other dependencies when one fails. If it does, `lint` runs the module tasks with mise's continue-on-error behavior instead of plain dependencies.

`check-generated` is not kept at the root. `backend:lint` depends on `backend:check-generated`, and `frontend:lint` runs `check:api`, so the root `lint` already names every stale file.

`scripts/test.sh` calls `mise run backend:pull-testcontainers` and `mise run backend:test-upgrade` instead of the two `just --justfile` calls, and its test stubs `mise` instead of `just`.

Alternatives rejected:
- mise monorepo mode, with a `mise.toml` per module and tasks such as `//backend:lint`, or `:lint` from inside `backend/`. It adds three configuration files with settings of their own. The same short name would mean different tasks in different directories, so the review hook and the commands in `tasks.md` would have to depend on the directory. Nested monorepo roots also have their own rule: the nearest one wins, and agents work in worktrees.
- Keeping just and lefthook, pointing the CI lint jobs at the module `lint` recipes, as the test jobs already call recipes, and making the lefthook jobs check-only. This fixes the CI drift and the hook writing to the index with far fewer edits and no renames. It keeps three tools for one job, though, and CI keeps depending on both mise and just. The team chose a single tool, so this baseline is the fallback if the migration turns out more costly than it is worth.

### Commands keep their behavior under new names

`just run` becomes `mise run dev`: `mise run run` reads badly, and `dev` names what it starts. Its arguments go on to `scripts/dev.sh` as before, so the `run-app` skill's `-t=false` keeps working. Every other root recipe keeps its name as a task. Module recipes keep their names behind the module prefix. `agent` is the prefix of the verification agent, as `agent/` is its directory.

Alternative rejected: renaming tasks to new conventions in the same change, which would make every document change twice as large for no gain.

### hk checks and never writes

`hk.pkl` defines the steps at the top level, as hk recommends, and the `pre-commit` hook sets `fix = false` and `stash = "none"`. The steps are:

- a formatting check per module on the staged files of that module: `golangci-lint fmt --diff` for the backend and the agent, `oxfmt --check` for the frontend. A failing step prints the file and `mise run format`;
- the module's lint task, `mise run backend:lint`, `mise run agent:lint` or `mise run frontend:lint`, gated by a glob on the module directory and run once per commit, not per file.

hk's default `pre-commit` hook built from top-level steps stashes unstaged changes with `git stash` while it fixes files. In a working tree that several agents edit, that would take away another agent's edits for the duration of the hook and could fail to restore them. The root `AGENTS.md` forbids touching another agent's changes, so the hook does not write at all.

The check reads the working tree, not the index. A file whose staged content is unformatted while its unstaged content is formatted passes the check. This gap already exists with lefthook, and closing it means stashing, which this decision rules out.

`hk install --mise` installs the hook per repository and starts it through mise, replacing `lefthook: mise exec -- lefthook`.

Alternatives rejected:
- hk with `fix = true` and `stash = "none"`. It matches today's `stage_fixed`, but adding a fixed file back to the index also adds its unstaged hunks, so another agent's half-written edit can enter the commit.
- hk's default with `stash = "git"`, for the reason above.
- `hk install --global`. It needs Git 2.54 or newer and installs the hook into every repository of the user.

### CI lint jobs call the module lint tasks

The backend, agent and frontend lint jobs install their tools with the pinned `jdx/mise-action` and run `mise run backend:lint`, `mise run agent:lint` and `mise run frontend:lint`. The shell job runs `mise run lint-shell`. `actions/setup-go` and the `go install` of golangci-lint leave these jobs. The Go cache step stays. `go mod verify`, `govulncheck` and the frontend build stay separate steps, because they are not part of the local lint. The test jobs replace each `just` call with its task, such as `mise run backend:test-group-packages "$TEST_GROUP"`, and keep their own `go test` line and timeout.

Each job installs only the tools its task needs through `install_args`, so a lint job does not download Node, Python or the browser server. A new check that uses a tool the job does not install yet adds that tool to the job's `install_args`, as the `ci-pipeline` delta allows.

Alternative rejected: one `mise run lint` job for the whole repository. It would run every module on one runner, one after another for the slowest parts, and its check name would not say which module failed. `ci-pipeline` also keeps the backend lint as its own check, which publishing depends on.

### The review hook's command forms follow the tasks

`VERIFICATION_FORMS` in `.agent-hooks/review_gate.py` becomes `mise run backend:test-package <packages>` and `mise run agent:test`, both run from the repository root, plus the unchanged `pnpm --dir frontend test`. The character check stays, because `backend:test-package` hands the package list to bash. The hook messages and `.agent-hooks/run.sh` name `mise run doctor`.

Every active change's `## Change verification` block must use the new forms when the hook changes, or the hook blocks its next turn. This change's own block switches in the same task that changes the hook, because before that task the hook accepts only the old forms.

Alternative rejected: accepting both the old and the new forms for a transition period. The root `AGENTS.md` forbids shims of the old shape.

### The contract pipeline moves to the tasks with everything else

Task groups 1 to 5 built the contract pipeline on just recipes, and groups 6 to 10 move those recipes to mise tasks together with the older ones. `generate-api` stays a root task, the backend freshness check becomes `backend:check-generated`, which `backend:lint` depends on, and the root `check-generated` recipe does not become a task, because the root `lint` already runs both freshness checks through the module lints.

Alternative rejected: rewriting the completed tasks of groups 1 to 5 to mise names. They record what was built and verified at the time, and groups 6 to 10 say how it moved.


## Risks / Trade-offs

- [A strict-server rewrite changes the version body byte for byte, for example by adding fields] → The contract declares exactly one required field. An `apptest` test also asserts the raw JSON body, because agents parse it with their own struct.
- [Moving the route changes middleware order for `/system/version`] → The guard runs only on the contract group, after the request ID, access log, recovery and compression middlewares that every route gets. The `apptest` test checks that `X-Request-Id` is present.
- [`apptest` tests in one package are serial until milestone 13] → Packages still run in parallel. The mutex makes the limit visible instead of producing flaky shared state.
- [`msw` and `openapi-fetch` change `pnpm-lock.yaml`] → They are the two libraries the pipeline needs, the same ones cloud uses. The lockfile changes in the same change that uses them.
- [A script that parsed `{"error": "not found"}` breaks] → The proposal marks the break for the release notes.
- [mise cancels the remaining dependencies of `lint` when one fails, and the root lint names only the first failing module] → The first task checks this behavior, and `lint` uses continue-on-error when needed. The scenario "Two modules fail lint" checks the result.
- [hk fetches the Pkl packages that `hk.pkl` amends from GitHub on its first run, and a commit fails offline] → A task checks a commit with the network off after the first run. If hk does not cache the packages, the hk Pkl package is copied into the repository and `hk.pkl` amends the local copy.
- [A worktree created before this change keeps a lefthook hook that calls `just`, which no longer exists] → The hook fails loudly on the missing command, not silently. `AGENTS.md` names `lefthook uninstall` and `hk install --mise` as the one-time step.
- [Commits no longer format staged files] → The refusal names `mise run format`, and `mise run format` is the only writer. People who relied on the hook formatting for them run one more command.
- [The frontend lint task in the hook also runs `check:api`, which downloads `openapi-typescript` through `pnpm dlx` on its first run] → pnpm caches the download. A first commit without network fails with the pnpm error.
- [CI lint jobs now run `golangci-lint fmt --diff`, `vacuum` and the freshness check, which they skipped before] → Run `mise run lint` on the branch before pushing. A failure there is a real finding that CI used to miss.

## Migration Plan

Nothing to migrate in data. The release notes mention the new 404 body. Each contributor runs `mise install`, `lefthook uninstall` and `hk install --mise` once per clone. A worktree shares the clone's hooks directory, so one installation covers its worktrees. Rollback is a revert of the change, followed by `lefthook install`.
