## 1. The agent contract and its clients

- [x] 1.1 Move the response check out of `internal/lib/apiclient/contract_checking_doer.go` into the test-only package `internal/lib/contracttest` per design.md decision 4, together with the tests that check the check itself, and make `apiclient.New` use it. Add `internal/lib/contracttest` to the test-library exemptions of `depguard` in `backend/.golangci.yml`. Verify with `mise run backend:test-package ./internal/lib/contracttest/... ./internal/lib/apiclient/... ./internal/modules/users/...` and `mise run backend:lint`.
- [x] 1.2 Write `backend/api/agent.yaml` per design.md decisions 1, 2 and 8: the four operations with the frozen paths, `{verificationId}` for the old `:id`, the tag `verification`, the bearer `securityScheme`, the heartbeat with optional `maxRamMb` and deprecated optional `maxRamGb`, the schemas `VerificationAssignment` and `VerificationReport`, the binary stream with `Content-Disposition`, and its own `Problem` and `ProblemCode` with `invalid_request`, `unauthenticated`, `rate_limited`, `verification_gone` and `internal`. Take the field names and types from `internal/agent/api/dto.go`, `internal/agent/api/constants.go` and `internal/features/verification/runs/dto.go`. Embed it as `contract.Agent` in `backend/api/api.go`. Make `backend:lint` in `mise.toml` run `vacuum` on `agent.yaml` too. Give `databasus-path-plural-nouns` an exception that lists the four frozen paths literally, with its reason next to it. Add `CodeVerificationGone` with status `410` to `internal/lib/apierror`, and change `code_test.go` to the two checks of decision 2. Verify with `mise run backend:lint` passing and `mise run backend:test-package ./internal/lib/apierror/...`. Then check two rules by hand and revert each edit. First, remove the `default` response of the heartbeat in `agent.yaml`, and see `mise run backend:lint` fail naming the heartbeat operation and `databasus-operation-has-default-response`. Second, add a path `/verification/{verificationId}/report` to `openapi.yaml`, and see it fail naming `databasus-path-plural-nouns` and that path.
- [x] 1.3 Generate the client and models of `agent.yaml` into `internal/lib/agentclient`, and the models and strict Gin server into `internal/modules/verification/internal/agentapi`, each with its `generate.go` and `oapi-codegen.yaml`. Add `internal/lib/agentclienttest` with `New(tb, baseURL, token)` on `contracttest`, and a test against an `httptest.Server`: a `410` with a JSON body that is not a `Problem` fails the test with a message that names the operation and the status, and a matching response passes. Verify with `mise run generate-api`, then `mise run backend:lint` passing, and `mise run backend:test-package ./internal/lib/agentclienttest/...`. Then edit a field in `agent.yaml` by hand, see `mise run backend:lint` name both stale generated files, and revert the edit.

## 2. The server

- [x] 2.1 Store memory in megabytes per design.md decisions 7 and 8, while the old controller still serves the agent. Add the migration that renames `verification_agents.max_ram_gb` to `max_ram_mb`, widens the column to `BIGINT` before multiplication to preserve the full stored range, multiplies stored values by 1024 and adds `CHECK (max_ram_mb >= 0)`, with its down migration, and a test in `backend/migrations` that a value of 2 reads 2048 after it. Rename the field of `agents.Agent` to `MaxRAMMb`. Give the agents service's capacity `MaxRAMMb` and `LegacyMaxRAMGigabytes`, and make `RecordHeartbeat` resolve them to megabytes, gigabytes times 1024 or zero. Make the old heartbeat controller pass the gigabytes it receives as `LegacyMaxRAMGigabytes`. Replace `maxRamGb` with `maxRamMb` in `VerificationAgent` of `openapi.yaml`, run `mise run generate-api`, and update:
  - `internal/modules/verification/internal/httpapi/agent.go`;
  - `internal/app/apptest/verification.go`, whose field becomes `MaxRAMMb`;
  - `internal/features/telemetry`, which sends `maxRamMb`, with its tests;
  - the two assertions of `internal/agent/e2e/update_test.go`, which expect `2048`.

  Verify with `mise run backend:test-package ./migrations/... ./internal/modules/verification/... ./internal/features/telemetry/... ./internal/agent/e2e/...` and `mise run backend:test-upgrade`. Also verify that `rg -l 'max_ram_gb|MaxRAMGb' backend --glob '!backend/migrations/**'` lists exactly these files, which tasks 2.2 and 3.1 change: `backend/internal/features/verification/agents/dto.go`, `backend/internal/features/verification/agents/agent_controller.go`, `backend/internal/agent/api/dto.go`, `backend/internal/agent/api/api_test.go` and `backend/internal/agent/heartbeat/heartbeat.go`.
- [x] 2.2 Serve `agent.yaml` from the module per design.md decisions 2, 3 and 7:
  - add the handler and the authentication middleware to `agentapi`, and `RegisterAgentRoutes` with the middleware on a Gin subgroup;
  - add `RateLimiter ratelimit.Counter` to the module's `Deps` and to the agents service, and make `AuthenticateAgent` take `AgentCredentials`, check the UUID, the rate limit and the token in that order, and log each refusal with its reason and the client's address;
  - return `ErrVerificationGone` as `verification_gone`, translate the other modules' problems as decision 2 lists, and turn `ErrAgentNotFound` of the heartbeat into `unauthenticated`;
  - validate a negative capacity in the heartbeat and the claim, an unknown report status and an unnamed table statistic in the services;
  - rename `verifications.JobAssignment` to `VerificationAssignment`.

  Build the second route group in `internal/app/routes.go` with `RequireContractOperation(contract.Agent, ...)`. Delete `internal/features/verification`, the five temporary methods and their comment from `verification/module.go`, the bridge installation in `internal/app/app.go` and the agent routes in `routes.go`. Add `NewAgentClient` to `apptest`. Switch the agent calls of the module's API tests from `CallOldAPI` to it. The one exception is the agent ID that is not a UUID, which uses a raw `http.Client`. Add the API tests of design.md decision 9 for every scenario of the `backup-verification` delta, with an `apptest` helper next to `DamageLocalBackup` that removes a local backup's stored file. Verify with `mise run backend:test-package ./internal/modules/verification/... ./internal/app/... ./tools/archlint` and `mise run backend:lint`. Also check that each of `rg -n 'features/verification|verification_bridge' backend --glob '*.go'` and `rg -n 'CallOldAPI|agentCall' backend/internal/modules/verification` prints nothing.
- [x] 2.3 Run the agent end-to-end suite against the server of task 2.2 while the agent still has its old client. This checks that the new server accepts the requests of an agent that sends `maxRamGb` and reads statuses only. Verify with `mise run backend:test-package ./internal/agent/e2e/...` exiting 0.

## 3. The agent

- [x] 3.1 Switch `internal/agent/api` to `lib/agentclient` per design.md decisions 5 and 6:
  - the heartbeat and the version check go through the `go-retryablehttp` client of decision 6 with the agent's logger, and the claim, the report and the stream go through a plain `http.Client`;
  - request editors add the token and the no-cache headers;
  - the stream goes through the plain generated method;
  - the two system calls are written by hand on `net/http`;
  - the report budget and the binary check timeout become constructor fields.

  Delete `api/dto.go` and `api/enums.go`. Switch `runner`, `heartbeat`, `upgrade` and `start` to the generated types, including `VerificationAssignment` and `VerificationReport`, and make the heartbeat send `maxRamMb` with the configured value. Make `github.com/hashicorp/go-retryablehttp` a direct dependency, and keep `github.com/go-resty/resty/v2` only as the indirect dependency required by rclone. Update `api_test.go`, `heartbeat_test.go` and the runner fakes for the new shapes. Verify with `mise run agent:test`, `mise run agent:build`, `mise run backend:lint` and `mise run backend:test-package ./internal/agent/e2e/... ./tools/archlint` exiting 0. Also check that `rg -n 'go-resty|api\.HeartbeatRequest|MaxRAMGb|JobAssignment' backend --glob '*.go'` prints nothing. From `backend/`, verify that `GOFLAGS= go list -deps ./cmd/verification-agent` does not list `go-resty`, and that `go mod why -m github.com/go-resty/resty/v2` traces the remaining dependency through rclone's Proton Drive backend.

## 4. The frontend

- [x] 4.1 Show the agent's memory with `formatBytes(maxRamMb * 1024 * 1024, formatNumber)` in `VerificationAgentsComponent.tsx` per design.md decision 8. Change the key `verification.agents.capacity.ram` to `{{size}} RAM` and its translations in the six dictionaries. Extend `VerificationAgentsComponent.test.tsx` so an agent with `maxRamMb` 512 shows "512 MB RAM" and one with 4096 shows "4 GB RAM". Verify with `pnpm --dir frontend typecheck`, `pnpm --dir frontend lint` and `pnpm --dir frontend test src/features/verification src/shared/i18n`.

## 5. Documents and checks

- [x] 5.1 Update the documents:
  - `docs/architecture.md`, section "HTTP API":
    - the strict server sentence at line 229 says that a module serving `agent.yaml` does so from `internal/agentapi`;
    - the operations of `agent.yaml` are served on their own route group, and the verification module authenticates the agent by its token;
    - the users middleware and the access mark apply to `openapi.yaml`;
  - `docs/architecture.md`, the test support line: `contracttest` is the response check that `apiclient` and `agentclienttest` share;
  - `docs/testing.md:53` names the shared check, and `docs/testing.md:59` drops the sentence about `CallOldAPI` until milestone 12e;
  - `docs/glossary.md`:
    - the row "API contract" defines the user API contract (`openapi.yaml`) and the agent API contract (`agent.yaml`);
    - the row "Contract-checking test client" names `apiclient` and `agentclienttest`;
    - a new row "Contract check" (`contracttest`) is the response check both test clients share;
  - `.agents/skills/api-design/SKILL.md`: step 3 of "Contract first" names `internal/agentapi` for `agent.yaml`, and the sentence at line 49 about new codes and the frontend's dictionary applies to codes of `openapi.yaml`.

  Verify that `rg -n 'CallOldAPI|12e' docs` prints nothing, and that `rg -n 'contracttest|agentclienttest|agentapi' docs .agents/skills/api-design/SKILL.md` shows `docs/architecture.md`, `docs/testing.md`, `docs/glossary.md` and the skill.
- [x] 5.2 Update the rewrite plan, which is git-ignored and lives only in the main checkout at `/home/rostislav/development/project/databasus/temp/refactor/plan.md`:
  - check `12e`;
  - strike through two table rows: the old agent routes over the temporary methods of `verification/module.go`, and the package-level durations of the agent;
  - in section 12e, correct the done criterion: the end-to-end suite changes only in its two memory assertions;
  - add to milestone 13 the removal of `maxRamGb` from the heartbeat of `agent.yaml` and of `LegacyMaxRAMGigabytes` from the service.

  Verify that `rg -n '12e|maxRamGb' /home/rostislav/development/project/databasus/temp/refactor/plan.md` shows the checked line, the struck rows and the milestone 13 item.
- [x] 5.3 Search the code, the documents and the skills for every mention of what changed, and bring each in line. Verify with `rg -n 'maxRamGb|max_ram_gb|features/verification|verification_bridge|go-resty|agent/api/dto' backend frontend/src docs AGENTS.md backend/AGENTS.md frontend/AGENTS.md .agents/skills --glob '!backend/migrations/**' --glob '!**/*.gen.go'`: it must name only the deprecated heartbeat field in `backend/api/agent.yaml` and the indirect `resty` dependency in `backend/go.mod` and `backend/go.sum`. Run `mise run format` and `mise run lint`, and fix what they report. Verify that `mise run lint` exits with status 0.

## Change verification

```sh
mise run backend:test-package ./internal/lib/contracttest/... ./internal/lib/apiclient/... ./internal/lib/agentclienttest/... ./internal/lib/apierror/... ./internal/modules/users/... ./internal/modules/verification/... ./internal/app/... ./internal/features/telemetry/... ./internal/features/system/healthcheck/... ./migrations/... ./tools/archlint ./internal/agent/e2e/...
mise run agent:test
pnpm --dir frontend test src/features/verification src/shared/i18n
```
