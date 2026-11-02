## Why

Local backend tests are slow, while the machine stays mostly idle. A full run of the backend without the race detector takes 594 seconds with CPU busy on 10 to 35 percent of its 32 threads, and it needs little memory. The cost lies elsewhere:

| Cause | Measurement |
| --- | --- |
| The race detector | Four module packages: 105 s and 1353 CPU seconds with it, 54 s and 177 CPU seconds without it |
| The shared test server flushes every write | The same four packages under the race detector: 152 s, 105 s without flushing |
| Slow packages hold the 8 package slots, so others wait | `storages/internal/providers` starts at second 343 of the full run |
| One package is effectively sequential | `engines/postgresql/physical` takes 482 of the 594 seconds, and its top-level tests add up to 470 seconds |
| Sequential subtests | `workspaces` alone takes 41 s, or 25 s with parallel subtests under a temporary overlay |

Raising the limit alone does not help. With `-p=32` everything starts at once, the other 174 packages finish at 532 seconds, and the same tests take 3 to 7 times longer at low CPU, which points to a shared bottleneck, most likely the Docker daemon. `internal/agent/e2e` then fails because Docker runs out of network address pools, and it hangs instead of finishing.

Local checking also gets simpler. Choosing which packages a change affects is a judgment call that misses things, so one command should run everything.

## What Changes

- `mise run test` becomes the one routine local test command. It runs every backend package and the frontend unit and screen tests, as today followed by the upgrade test.
- **BREAKING** `mise run backend:test-package` and `mise run backend:test-group` are deleted. Agents and OpenSpec change verification run `mise run test`. To repeat one failing test, a developer or an agent calls the Go test command or the frontend runner directly.
- **BREAKING** The rule that a full run happens only when the user asks is deleted. The review hooks stop rejecting `mise run test`, and they stop accepting `backend:test-package`, `backend:test-group`, `agent:test` and `pnpm --dir frontend test` in the verification section.
- `mise run test` runs slow backend packages from a list in their own `go test` process with a lower package limit, longest first, and all other packages in a concurrent `go test` process, so that the others do not wait for slow ones.
- **BREAKING** Local test commands stop using the race detector, including OpenSpec verification. A local data race surfaces in CI, or in `go test -race` started by hand.
- The shared test server runs without durability, with the flags the engine test containers already use, and with `max_connections=1000`, because a full run with parallel subtests exhausted the limit of 300. The agent recreated the `backend-db-test` container twice for these two settings, as the user has authorized.
- `engines/postgresql/physical` and other packages on the critical path stop running their tests one after another, through parallel tests or local shards.
- The server parses the API contract once per process instead of twice per application.
- Independent subtests run in parallel, and the lint guards this. Subtests that share a server or a fixture stay sequential and state why.
- Tests that this work finds failing or flaky are fixed in this change: `apierror` `TestCode`, `httpserver` `TestListen`, and the failure and hang of `TestAgentStartup` when Docker runs out of networks.

## Out of scope

- CI: `.github/workflows` does not change. CI keeps its groups, shards and the race detector. Its disposable test server gets the same durability flags, because it starts from the same compose file.
- Skipping tests locally. Every test of the backend and frontend suites runs in `mise run test`; long tests start early instead.
- Cheaper password hashing in tests.
- The browser suites and the Docker filesystem suite, which stay separate commands.
- Docker daemon settings such as `default-address-pools`, and networks left by other projects.
- More memory for the machine, or tests that are rewritten to wait less.
- Frontend test speed.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `local-development-workflow`: `mise run test` is the one routine command, includes the frontend tests and schedules slow packages separately.
- `backend-code-checks`: local runs no longer use the race detector, and the lint checks that tests and their subtests use parallelism consistently.
- `ci-pipeline`: the race detector requirement covers every CI check that runs backend tests, and the local shard command is removed.
- `backend-test-isolation`: the shared test server gives up durability, and losing it costs only a recreation with an empty data directory.
- `implementation-review`: the verification section accepts `mise run test` and the browser commands, and no narrow command.
- `proposal-review`: `mise run test` is no longer a forbidden verification command.

## Impact

- Test infrastructure: `docker-compose.yml`, `mise.toml`, `scripts/test.sh`, `mise-tasks/backend/test-group` (deleted), `mise-tasks/agent/e2e`, `mise-tasks/agent/e2e-podman`, `mise-tasks/backend/test-upgrade`, `frontend/tooling/runBrowserE2E.ts`, a new slow package list in `backend/`, and the script tests in `scripts/`.
- Review hooks: `.agent-hooks/review_gate.py`, its tests and the reviewer prompts, and the task rules in `openspec/config.yaml`.
- Backend code: `internal/lib/httpserver/contract_guard.go` and `internal/app/routes.go`, with no change in production behavior, and the `apierror` codes or the contract, depending on what the failing test shows.
- Backend tests: up to 315 top-level tests that the `tparallel` linter reports today, the failing tests above, and the linter settings in `backend/.golangci.yml`.
- Documentation: `AGENTS.md`, `backend/AGENTS.md`, `docs/testing.md` and `.agents/skills/testing/SKILL.md`.
- Concurrent agents wait for each other, because full runs take turns through a machine-wide lock.

The change answers to the root `AGENTS.md`, `backend/AGENTS.md` and `frontend/AGENTS.md`.
