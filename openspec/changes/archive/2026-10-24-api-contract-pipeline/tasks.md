## 1. Contract and generation

- [x] 1.1 Add `backend/api/openapi.yaml` with `GET /system/version`, the `Problem` schema and the `ProblemCode` enum, plus `backend/api/api.go`, which embeds it as `api.OpenAPI`, as design.md "The contract and its tools" describes. Add `backend/api/vacuum.yaml` and pin `vacuum` in `mise.toml`. Verify that `vacuum lint -r backend/api/vacuum.yaml backend/api/openapi.yaml` passes, and that on a copy of the contract with the default response removed it fails with output that names the `getVersion` operation and the rule from `vacuum.yaml`.
- [x] 1.2 Add `oapi-codegen` as a `tool` dependency in `backend/go.mod`, and add `internal/app/systemhttpapi` with its `oapi-codegen.yaml` and `generate.go`, generating the `system` tag as a strict Gin server. Add `generate-api` to `backend/Justfile` and to the root `Justfile`, plus `generate:api` and `check:api` to `frontend/package.json`, and generate `frontend/src/shared/api/schema.gen.ts`. Verify that `just generate-api` at the root leaves `git status` unchanged when run a second time.
- [x] 1.3 Add `vacuum lint` and the generated-code freshness check to `lint` in `backend/Justfile`, and `pnpm --dir frontend check:api` to `lint` in the root `Justfile`. Verify that `just lint` at the root passes, and that it fails naming the file after a field is added to the contract without regeneration. Then run `just generate-api` with that field still added and check with `rg` that it appears in `internal/app/systemhttpapi/api.gen.go`, and in `frontend/src/shared/api/schema.gen.ts`; revert the field and regenerate.

## 2. Problem errors and the contract guard

- [x] 2.1 Add `internal/lib/apierror` as design.md "`Problem` comes from `lib/apierror`" describes, with tests for `Write`, the strict request handler (400), the parameter error handler (400 `invalid_request` for a malformed path parameter on a test route), the response handler (500 with the fixed detail), and the equality of the Go codes with the contract enum. The 500 test fails a handler with an error that names a table and a value, captures the log, and checks that neither appears in the body and that the logged `request_id` equals the response's `X-Request-Id`. Verify with `just test-package ./internal/lib/apierror/...` in `backend/`.
- [x] 2.2 Make `httpserver.HandleUnknownPath` answer `Problem` 404 `not_found`, and add `RequireContractOperation` as design.md "The contract route guard" describes. Add tests: an unknown `/api/` path gets `Problem` 404 and a frontend path gets the page; a route mounted under the guard but missing from the contract answers 500 `internal`, does not run its handler and logs the method and path. Verify with `just test-package ./internal/lib/httpserver/...` in `backend/`.

## 3. Test harness and the version operation

- [x] 3.1 Add `internal/lib/apiclient` with the generated client and `New(t, baseURL)` as design.md "`apiclient` checks every response" describes. Its tests run against `httptest` servers that return a correct version body, a version as a number, an error with a non-`Problem` body and a `201` with the version body, and check that only the first passes and that the failures name the operation, the status and the field. Verify with `just test-package ./internal/lib/apiclient/...` in `backend/`. Then add a field to the version response in the contract, run `just generate-api`, check with `rg` that the field appears in the generated file of `internal/lib/apiclient`, revert the field and run `just generate-api` again.
- [x] 3.2 Split `App.NewHandler()` out of `App.Serve`, run `setUpDependencies` once per process, and add `internal/app/apptest` returning `*apptest.App` with `URL`, as design.md "`apptest` builds the real application, one instance at a time" describes. Add tests for the three scenarios of the `api-contract` requirement "Backend API tests run against the real application": the response through `apptest` carries `X-Request-Id`, a row written in one test's database is absent in the next test's database, and a second `apptest.New` while one is alive fails with the message about serial tests. Verify with `just test-package ./internal/app/... ./cmd/databasus/...` in `backend/`.
- [x] 3.3 Implement the `systemhttpapi` handler, mount the generated server on the guarded group in `internal/app`, remove the old route, `VersionController`, its DTO and `di.go`, and keep `GetAppVersion`. Add `apptest` tests through `apiclient` for every scenario of the requirement "The version operation keeps its path and body": no session, the raw body equal to `{"version":"<version>"}`, and `X-Request-Id` present. Update the assertions of `internal/app/server_test.go` on the 404 body and the former Swagger paths to `Problem` with `not_found`. Verify with `just test-package ./internal/app/... ./internal/features/system/...` in `backend/`, and with `just --justfile agent/verification/Justfile test` at the root, because the agent reads this body.

## 4. Frontend

- [x] 4.1 Add `openapi-fetch` and the dev dependency `msw` to `frontend/package.json`, which also updates `pnpm-lock.yaml`. Add `shared/api/apiClient.ts` and `shared/api/requireData.ts` as design.md "Frontend client and mocks" describes, and move `systemApi.getVersion` to them, deleting `entity/system/model/VersionResponse.ts` and changing the `VersionResponse` re-export in `entity/system/index.ts` to the generated type under that name. Verify with `pnpm --dir frontend typecheck`.
- [x] 4.2 Add `shared/lib/test/mockApi.ts` on `msw/node`, and tests for `systemApi.getVersion`: a 200 returns the version, a `Problem` 500 throws an `ApiError` with `code` `internal` and status 500, and an answer without a body throws `request_failed`. Add a type-level test that a mock without `version` does not compile, as a `@ts-expect-error` line checked by `pnpm typecheck`. Verify with `pnpm --dir frontend test src/entity/system src/shared/api` and `pnpm --dir frontend typecheck`.
- [x] 4.3 Add translations for `not_found`, `internal` and `invalid_request` to all six dictionaries, and a table of contract codes in `translateApiError` declared with `satisfies Record<components['schemas']['ProblemCode'], TranslationKey>`. Add a case to `translateApiError.test.ts`, following its `createI18n('ru')` pattern, that an `ApiError` with code `internal` returns the Russian text of its key, not the raw code. Verify with `pnpm --dir frontend test src/shared/i18n` and `pnpm --dir frontend typecheck`, and check once that removing the `internal` entry makes `pnpm --dir frontend typecheck` fail naming `internal`.

## 5. Checks and documents

- [x] 5.1 Enable `importas` in `backend/.golangci.yml` with the aliases from design.md "`importas` aliases", and update the backend imports to them. Verify with `just lint` in `backend/`.
- [x] 5.2 Update the documents: `docs/architecture.md` "HTTP API" gets the `system` tag exception, and its "Automatic checks" sentence on what arrives later keeps only `gochecknoglobals`; `docs/testing.md` loses the sentence about `mockApi` arriving with the contract; the `api-design` skill gets the `system` tag exception in "Contract first" and the error code exception in "Fields"; `docs/glossary.md` "Code" gets the three terms from design.md "The glossary gains the error terms". Verify that `rg -n "arrives with the generated API packages|arrives with the contract" docs` prints nothing, that `rg -n "systemhttpapi" docs/architecture.md .agents/skills/api-design/SKILL.md` shows both exceptions, that `rg -n "gochecknoglobals" docs/architecture.md` still shows the sentence, and that `rg -n "^\| Problem|^\| Error code|^\| Contract-checking test client" docs/glossary.md` shows three rows.
- [x] 5.3 Run `just lint` at the root and fix what it reports. Verify that it exits with status 0.

## 6. mise tasks

Groups 1 to 5 were built on just recipes. Groups 6 to 10 move every recipe, the ones groups 1 to 5 added included, to mise tasks.

- [x] 6.1 Add `min_version` and an exact `hk` pin to `mise.toml`, and the backend tasks as design.md "Every task lives at the root, with module prefixes" describes:
  - `backend:lint` with `go mod tidy -diff`, depending on `backend:check-generated`;
  - `backend:format`, `backend:generate-api` and `backend:check-generated`;
  - `backend:test-package` with a variadic package argument;
  - the file tasks `backend:test-group-packages`, `backend:test-group-tests`, `backend:test-upgrade`, `backend:clean-testcontainers` and `backend:pull-testcontainers`.

  Verify:
  - `mise run backend:lint` started from `frontend/` exits with status 0;
  - `mise run backend:test-package ./internal/lib/apierror/...` started from the root passes;
  - for the groups `other`, `logical` and `physical-2-of-3`, `mise run backend:test-group-packages <group>` and `mise run backend:test-group-tests <group>` print the same output as the matching `just` recipes, which `diff` shows;
  - a mise binary older than `min_version`, for example one run through `mise exec aqua:jdx/mise@<older version> --`, refuses `mise run backend:lint` without running a check, and its output names the minimum version.
- [x] 6.2 Add the agent tasks `agent:run`, `agent:build`, `agent:test`, `agent:lint` (with `go mod tidy -diff`), `agent:format`, `agent:e2e`, `agent:e2e-podman` and `agent:e2e-clean`, and the frontend tasks `frontend:lint` (`format:check`, `lint`, `typecheck`, `check:api`) and `frontend:format`. Verify that `mise run agent:lint`, `mise run agent:test`, `mise run agent:build` and `mise run frontend:lint` exit with status 0.
- [x] 6.3 Add the root tasks `dev`, `test`, `test-filesystem`, `pull-testcontainers`, `generate-api`, `lint`, `format`, `lint-shell` and `doctor`.
  - First check whether mise cancels the remaining dependencies of a task when one fails. Then make `lint` report every failing module, as design.md says.
  - Verify that `mise run lint` and `mise run doctor` exit with status 0.
  - Verify that `mise run generate-api` run twice leaves the second `git status --porcelain` equal to the first.
  - Verify that `mise tasks info dev` shows that `dev` passes its arguments to `scripts/dev.sh`.
  - Verify that `test-filesystem` keeps the selection through the `CASE` environment variable and builds no image when `DATABASUS_IMAGE` names one: with `DATABASUS_IMAGE` set to an image present locally, `CASE=no-such-group mise run test-filesystem` exits with a non-zero status, names `no-such-group` and starts no container (`docker ps -a` lists none of the suite's containers before and after).
  - Add a lint error to one backend Go file and one frontend file at once, check that `mise run lint` exits with a non-zero status and that its output names both files, and revert both edits.
  - Add a formatting difference to one file in each of the backend, the agent and the frontend, run `mise run format`, check that `mise run lint` then exits with status 0, and revert the three files.
  - Add a field to the version response in `backend/api/openapi.yaml` without regenerating, and check that `mise run lint` fails and names `internal/app/systemhttpapi/api.gen.go`, `internal/lib/apiclient/client.gen.go` and `src/shared/api/schema.gen.ts`. Then remove the `default` response of `getVersion`, and check that the lint names `getVersion` and `databasus-operation-has-default-response`. Revert both edits.

## 7. Commit hook

- [x] 7.1 Add `hk.pkl` as design.md "hk checks and never writes" describes. Check its behavior in a temporary clone of the repository under `/tmp`, where `hk install --mise` runs, not in the shared working tree:
  - a staged, unformatted Go file and, separately, a staged, unformatted frontend file each make `git commit` fail, and the output names the file and `mise run format`;
  - a staged backend file that is formatted but fails `backend:lint` makes `git commit` fail, and the output shows the lint error;
  - a second backend file with unstaged edits has the same `sha256sum` before and after a refused commit and after a passing one, `git diff --cached` is unchanged after the refused one, and `git show --stat HEAD` after the passing one lists only the staged file;
  - a staged file with further unstaged edits keeps both: after the commit attempt, `git diff` and `git diff --cached` for that file print what they printed before;
  - a staged unformatted Go file that is formatted afterwards in the working tree only passes the formatting check, as the scenario "Staged content differs from the working tree" says;
  - a commit that touches only the backend runs no frontend or agent step, as `hk run pre-commit --plan` shows;
  - after one run, a commit attempt with `HTTPS_PROXY=http://127.0.0.1:9` set behaves the same. If it does not, copy hk's Pkl package into the repository and point `hk.pkl` at it, as design.md "Risks" says.

  Delete the temporary clone afterwards.
- [x] 7.2 Delete `lefthook.yml` and remove `lefthook` from `mise.toml`. Verify that `rg -n lefthook --glob '!openspec/changes/archive/**' --glob '!openspec/changes/api-contract-pipeline/**'` prints nothing.

## 8. Review hook and scripts

- [x] 8.1 Change `VERIFICATION_FORMS`, `ALLOWED_VERIFICATION_FORMS`, the comments that name `just`, `.agent-hooks/run.sh` and the hook prompts to the forms of design.md "The review hook's command forms follow the tasks".
  - Update `.agent-hooks/review_gate_test.py`, with cases for an accepted `mise run backend:test-package ./internal/...`, an accepted `mise run agent:test`, a refused `mise run agent:test ./x`, a refused `mise run backend:test-package ./internal/a;rm` and a refused old `just test-package ./internal/...`.
  - In the same edit, change this change's `## Change verification` block to `mise run backend:test-package ...` and `mise run agent:test`, because from this task on the hook accepts only the new forms.
  - Verify with `python3 -m unittest discover -s .agent-hooks -p '*_test.py'`, and check that the hook's parser reads this change's verification block without an error: `python3 -c "import sys; sys.path.insert(0, '.agent-hooks'); import review_gate as g; [g.parse_verification_command(l) for l in g.read_change_verification_block(open('openspec/changes/api-contract-pipeline/tasks.md').read()) if l.strip()]"`.
- [x] 8.2 Change the two `just --justfile` calls in `scripts/test.sh` to `mise run backend:pull-testcontainers` and `mise run backend:test-upgrade`, and its comments that name recipes. Change `scripts/test_sh_test.py` to stub and expect `mise` in their place. Verify with `python3 -m unittest discover -s scripts -p '*_test.py'` and `mise run lint-shell`.

## 9. CI

- [x] 9.1 Change `.github/workflows/ci-release.yml` as design.md "CI lint jobs call the module lint tasks" describes. The lint jobs of the backend, the agent and the frontend run their module lint task through the pinned `jdx/mise-action`, and the shell job runs `mise run lint-shell`. The test jobs call the tasks that replace their `just` recipes. Remove `actions/setup-go` and the `go install` of golangci-lint from the lint jobs.
  - Verify that `rg -n '\bjust\b|golangci-lint@' .github/workflows/ci-release.yml` prints nothing.
  - Verify that every `jdx/mise-action` line carries a full commit SHA with a version comment.
  - Verify that the file parses as YAML with `pnpm --package=yaml@2.8.1 dlx yaml valid < .github/workflows/ci-release.yml`.

## 10. Documents and removal

- [x] 10.1 Update every document that names a recipe or the hook:
  - the root `AGENTS.md`: the development environment, with `hk install --mise` and `lefthook uninstall` as the one-time setup, and the command table;
  - `backend/AGENTS.md`, `frontend/AGENTS.md`, `agent/verification/AGENTS.md`;
  - the `testing`, `run-app` and `api-design` skills;
  - `docs/testing.md` and `docs/architecture.md`;
  - `backend/README.md`, `README.md` and its five translations under `assets/readme/` where they name a command, `assets/tools/README.md`;
  - the task rules in `openspec/config.yaml`;
  - the comments in `backend/.air.toml`, `docker-compose.yml` and `agent/verification/e2e/docker-compose.yml`.

  Verify that `rg -n '\bjust (run|test|lint|format|doctor|generate-api|check-generated|pull-testcontainers|lint-shell|e2e|build)|Justfile|--justfile' --glob '!openspec/changes/archive/**' --glob '!openspec/specs/**' --glob '!openspec/changes/*/specs/**' --glob '!openspec/changes/api-contract-pipeline/**'` prints nothing.
- [x] 10.2 Delete `Justfile`, `backend/Justfile` and `agent/verification/Justfile`, and remove `just` from `mise.toml`. Verify that `mise run lint` at the root exits with status 0 and that `mise tasks ls` lists every task from groups 6 to 8.

## Change verification

```sh
mise run backend:test-package ./internal/lib/apierror/... ./internal/lib/httpserver/... ./internal/lib/apiclient/... ./internal/app/... ./cmd/databasus/... ./internal/features/system/... ./tools/archlint/... ./cmd/seed_dev/...
mise run agent:test
pnpm --dir frontend test src/entity/system src/shared/api src/shared/i18n
```
