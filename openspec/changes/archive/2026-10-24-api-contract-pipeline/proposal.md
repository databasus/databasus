## Why

The backend API has no contract. Error bodies come in five shapes, the frontend keeps 155 hand-written type files that already drift from the backend, and tests check responses only where someone wrote an assertion. Every module that moves in milestones 4 to 13 has to land its operations in a contract with generated code and contract-checked tests. Building that pipeline inside the first module move would double the size of that change, so this change builds it end to end on one existing operation.

The repository needs three tools to run its own commands: mise pins the tool versions, `just` holds the commands and lefthook runs the commit hook. CI lints with its own command lists on top of that: the backend lint job installs golangci-lint with `go install` at a second recorded version and calls the linter directly, so the contract checks this change adds to the lint, `vacuum` and the generated-code freshness check, would never run in CI. The team decided to keep one tool for both: mise already installs every tool, its tasks can hold the commands, and hk, built by the same author, runs the hook through mise. CI then depends on mise alone and calls the same tasks a developer runs, and the hook becomes a check that never writes to the working tree.

## What Changes

- `backend/api/openapi.yaml` describes `GET /system/version`, its `x-access: public` mark, the `Problem` error schema and the error codes this change uses. `vacuum` checks the contract against the rules of the `api-design` skill.
- `mise run generate-api` generates the strict Gin server, the Go test client and `frontend/src/shared/api/schema.gen.ts`. `mise run lint` fails when any generated file is out of date.
- `GET /api/v1/system/version` is served by the generated server. Its path, its public access and its body `{"version": "..."}` stay exactly as installed verification agents expect them.
- `lib/apierror` writes every error of a contract operation as `Problem` with a stable `code`. A 500 never carries the text of an internal error.
- **BREAKING** for scripts that parse the 404 body: an unknown path under `/api/` answers `Problem` with `code: not_found` instead of `{"error": "not found"}`.
- A route mounted for the contract but missing from it answers 500 `internal`, so a mounting mistake cannot expose an operation without its access check. Routes of code outside the target layout are not checked yet; each is checked once its module moves.
- `apptest.New(t)` builds the application through `app.New` with its own database from `testdb` and returns it with its URL, and `apiclient.New(t, url)` fails the test on any response that differs from the contract. While old code is wired through process-wide globals, `apptest` tests of one package run one at a time.
- The frontend calls the version operation through a client on `openapi-fetch` with types from `schema.gen.ts`, and a failed call becomes an `ApiError` carrying the `Problem` code. Frontend tests mock the network through `mockApi`, which is typed by the contract. Every error code in the contract has a translation in every dictionary.
- `importas` is turned on with fixed aliases for the generated packages.

Development tooling moves from just and lefthook to mise and hk:

- **BREAKING** The three `Justfile`s are deleted, and `just` leaves `mise.toml`. Their recipes become mise tasks: root tasks in `mise.toml`, longer scripts as file tasks under `mise-tasks/`, and module tasks named `backend:<task>`, `agent:<task>` and `frontend:<task>`. A task always runs in its module's directory, whatever directory it is started from.
- **BREAKING** The local commands change names:

  | Before | After |
  |---|---|
  | `just run` | `mise run dev` |
  | `just test`, `just test-filesystem`, `just pull-testcontainers`, `just lint`, `just format`, `just doctor`, `just generate-api`, `just lint-shell` | `mise run test`, `mise run test-filesystem`, `mise run pull-testcontainers`, `mise run lint`, `mise run format`, `mise run doctor`, `mise run generate-api`, `mise run lint-shell` |
  | `just test-package <packages>` in `backend/` | `mise run backend:test-package <packages>` from any directory |
  | `just lint`, `just format` in a module directory | `mise run backend:lint`, `mise run agent:lint`, `mise run frontend:lint` and the matching `format` tasks |
  | `just --justfile agent/verification/Justfile <recipe>` | `mise run agent:<recipe>` |

  The root `check-generated` recipe of this change does not become a task. The root `mise run lint` runs every module's lint, and each module's lint checks its own generated files, so one run still names every stale file.
- **BREAKING** hk replaces lefthook. `lefthook.yml` is deleted, and `hk.pkl` defines the pre-commit hook. The hook only checks: it no longer rewrites staged files or adds them back to the commit, and it never stashes unstaged changes, so another agent's edits in the same working tree stay untouched. A commit with an unformatted file is refused with the command that formats it.
- `mise.toml` pins hk and sets a minimum mise version. `go mod tidy -diff`, which only the lefthook job ran, moves into `backend:lint` and `agent:lint`.
- Every CI job that runs a check installs its tools through mise and calls the same task as a local run: the lint jobs call `backend:lint`, `agent:lint`, `frontend:lint` and `lint-shell`, and the test jobs call the tasks that replace their `just` recipes. The separate `go install` of golangci-lint goes.
- The review hook accepts `mise run backend:test-package <packages>` and `mise run agent:test` in a change's verification block instead of the two `just` forms, and `mise run doctor` replaces `just doctor` in its messages.

## Capabilities

### New Capabilities

- `api-contract`: the contract as the single source of the API shape, the checks that keep the code generated from it current, and the test client that holds responses to it.

### Modified Capabilities

- `api-http-server`: unknown API paths answer `Problem`; contract operations answer errors as `Problem` without internal details; a contract route without an operation answers 500; the version operation keeps its path and body.
- `frontend-localization`: every error code the contract declares has a translation in every language.
- `local-development-workflow`: every command is named as a mise task. The commit check stops rewriting staged files and leaves unstaged changes alone. The pinned tool set loses the command runner and names hk as the commit hook runner, and the repository sets a minimum mise version.
- `docker-storage-compatibility`: the filesystem suite's entry point becomes `mise run test-filesystem`.
- `ci-pipeline`: CI checks run through the same tasks as the local checks.
- `review-hook-launch`: the environment check is `mise run doctor`.
- `implementation-review`: the allowed change verification forms become the mise task forms.
- `proposal-review`: the scenario of a forbidden full suite run names `mise run test`.
- `agent-workflow`: the browser check starts and recognizes the app through `mise run dev`.

## Impact

- Backend: new `backend/api` (contract, `vacuum` rules, the embedding package), `internal/lib/apierror`, `internal/lib/apiclient`, `internal/app/apptest`, the generated server and the version handler under `internal/app`, `internal/lib/httpserver` (`Problem` 404 and the contract route guard), `internal/app/routes.go` (the old version route is removed).
- Dependencies: `oapi-codegen` as a Go tool, `kin-openapi` and the `oapi-codegen` runtime in `backend/go.mod`; `vacuum` in `mise.toml`; `openapi-fetch` and the dev dependency `msw` in `frontend/package.json`, which also changes `frontend/pnpm-lock.yaml`.
- Frontend: `shared/api` (generated types, the client, Problem handling), `entity/system/api/systemApi.ts`, a `mockApi` test helper, the `errors` keys of all six dictionaries.
- Tooling and docs: `backend/.golangci.yml` (`importas`), `docs/architecture.md` ("HTTP API": the `system` tag exception; "Automatic checks"), `docs/testing.md` (the sentence about `mockApi` arriving later), `docs/glossary.md` (`Problem`, error code, contract-checking test client), the `api-design` skill (the `system` tag exception, error code casing).
- Development tooling, deleted: `Justfile`, `backend/Justfile`, `agent/verification/Justfile`, `lefthook.yml`.
- Development tooling, new: `hk.pkl`, the `[tasks]` of `mise.toml`, file tasks under `mise-tasks/`.
- Development tooling, changed: `mise.toml` (tools, minimum version, tasks), `.github/workflows/ci-release.yml`, `.agent-hooks/review_gate.py` with its test, prompts and `run.sh`, `scripts/test.sh` (its calls of the pull and upgrade recipes) and `scripts/test_sh_test.py`, comments that name recipes in `backend/.air.toml`, `docker-compose.yml` and `agent/verification/e2e/docker-compose.yml`.
- Documents and agent instructions: the root `AGENTS.md` (development environment, command table), `backend/AGENTS.md`, `frontend/AGENTS.md`, `agent/verification/AGENTS.md`, the `testing`, `run-app` and `api-design` skills, `docs/testing.md`, `docs/architecture.md`, `backend/README.md`, `README.md` and its five translations under `assets/readme/` where they name commands, `assets/tools/README.md`, `openspec/config.yaml`.
- Every contributor runs `lefthook uninstall` and `hk install --mise` once. A worktree created before the change keeps a lefthook hook until then.
- Answers to the root `AGENTS.md`, `backend/AGENTS.md`, `frontend/AGENTS.md` and `agent/verification/AGENTS.md`.
- Depends on change `isolated-test-databases` for `testdb.Open`.

### Out of scope

- Moving any other operation to the contract.
- `agent.yaml`, `lib/agentclient` and `agentclienttest`, moved to milestone 12.
- `lib/clock`, moved to milestone 4.
- Enforcing `x-access` marks and request authentication through the contract, which the `users` middleware does in milestone 4, and the client's auth header, which arrives with the first authenticated operation.
- Validating request bodies against the contract at runtime; the version operation has no body.
- Fakes of external systems in `apptest`; each arrives with the first module that sends email, calls a webhook or writes to a storage.
- `website/`, which keeps its own npm scripts.
- The logic of `scripts/dev.sh`, `scripts/test.sh` and `e2e/docker-storage/run.sh`. Their callers change, and so do the two calls of `just` inside `scripts/test.sh`.
- The read-only module checks the review hook runs before its reviewers. They keep calling golangci-lint and pnpm directly.
- CI jobs that build, push or release images, and the job steps that are not checks, such as `govulncheck` and `go mod verify`.
- Pinning govulncheck or other CI-only tools in `mise.toml`.
- mise monorepo mode and per-module `mise.toml` files.
- Automatic formatting in the commit hook.
