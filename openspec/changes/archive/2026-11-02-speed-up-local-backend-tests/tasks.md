## 1. Shared test server

- [x] 1.1 Add `-c fsync=off -c synchronous_commit=off -c full_page_writes=off` to the `backend-db-test` command in `docker-compose.yml`, leaving `backend-db` unchanged, and verify with `docker compose config backend-db-test` that the command holds the flags.
- [x] 1.2 Recreate the shared `backend-db-test` container, which the user has authorized for this change as an exception to the rule against stopping the shared server. First confirm that no test run is active: `docker exec backend-db-test psql -U postgres -Atc "SELECT count(*) FROM pg_stat_activity WHERE datname LIKE 'databasus_test_%'"` returns `0`, and wait while it does not. Then run `docker compose up -d backend-db-test`, and verify that `SHOW fsync`, `SHOW synchronous_commit` and `SHOW full_page_writes` each return `off` on it, that a `databasus_test_template_%` database still exists, and that `SHOW fsync` returns `on` on `backend-db`.
- [x] 1.3 Add an assertion on these flags and on the unchanged `backend-db` to a script test in `scripts/`, and verify with `python3 -m unittest discover -s scripts -p '*_test.py'`.
- [x] 1.4 Start a uniquely named empty PostgreSQL 17 container with the new flags, select it through `TEST_DATABASE_DSN`, verify that `go test -count=1 ./internal/lib/testdb/...`, run in `backend/` with the libedit shim as `docs/testing.md` gives it, passes on it without any preparation step, and remove the container.

## 2. Race detector only in CI

- [x] 2.1 Replace `-race` with `${CI:+-race}` in `scripts/test.sh` and in `backend:test-package` and `mise-tasks/backend/test-group` while they still exist, make `scripts/test_group_test.py` set `CI` itself and assert both cases, and verify with `python3 -m unittest discover -s scripts -p '*_test.py'`.
- [x] 2.2 Replace `-race` with `${CI:+-race}` in `agent:test` in `mise.toml`, `mise-tasks/backend/test-upgrade` and `mise-tasks/agent/e2e-podman`, and verify with `mise run lint-shell` and `mise run agent:test`.
- [x] 2.3 Make `frontend/tooling/runBrowserE2E.ts` pass `-race` only when `process.env.CI` is set, and verify with `mise run frontend:lint` and `pnpm --dir frontend test:browser --project=e2e`.
- [x] 2.4 Add a script test in `scripts/` that pins `-race` in the `ci-release.yml` test group line and the `CI` condition in `scripts/test.sh`, the mise tasks above and the browser command, and verify with `python3 -m unittest discover -s scripts -p '*_test.py'`.

## 3. Contract parsed once per process

- [x] 3.1 In `internal/lib/httpserver/contract_guard.go`, add `NewAccessCheckedContractRouter` with one parse, validation, the access mark check and the router, and make `RequireContractOperation` take that router and the logger. Adapt `access_mark_test.go` and `server_test.go`, keeping the case that rejects an unknown access mark, and verify with `mise run backend:test-package ./internal/lib/httpserver/... ./internal/lib/contracttest/...`.
- [x] 3.2 Hold `loadUserContractRouter` and `loadAgentContractRouter` as `sync.OnceValues` in `internal/app`, use them in `routes.go`, keep the load error returned from `NewHandler`, and verify with `mise run backend:test-package ./internal/app/...`.

## 4. Failing and flaky tests

- [x] 4.1 Find why `TestCode/covers_both_contracts_and_appears_in_at_least_one_contract` fails on `instance_unhealthy`, a code in the contract and absent from `apierror`. Fix the code or the contract, and verify with `mise run backend:test-package ./internal/lib/apierror/...`.
- [x] 4.2 Fix `TestListen/closes_the_connection_when_the_request_line_stays_unfinished_past_the_header_limit`, which reported `9.999042003s` against a 10-second minimum because it starts its clock after the write. Take the time before the write, and verify with `mise run backend:test-package ./internal/lib/httpserver/...`.
- [x] 4.3 Make `internal/agent/e2e` fail and end when a network cannot be created, as in `TestAgentStartup` with `all predefined address pools have been fully subnetted`, instead of leaving its other scenarios waiting. Verify with a temporary failure injected into network creation that the package ends with the reported error, then remove the injection and run `mise run backend:test-package ./internal/agent/e2e/...`.
- [x] 4.4 Fix any other failing or flaky test that a later task exposes, in this change, and name it in the final report. Verify by rerunning its package.

## 5. Parallel subtests

For each task below, list the reported tests with `golangci-lint run --default=none --enable-only tparallel --max-issues-per-linter=0 --max-same-issues=0 <packages>` in `backend/`, and resolve each report as `design.md` says: add `t.Parallel()` to independent subtests or keep them sequential with `//nolint:tparallel // <shared state or dependency>`, add `t.Parallel()` to a sequential top-level test with parallel subtests or state what blocks it, and move a `defer` to `t.Cleanup`. Each of tasks 5.1 to 5.7 also verifies that this command reports nothing for its packages.

- [x] 5.1 Process `./internal/lib/...`, `./cmd/...`, `./internal/app/...`, `./internal/agent/runner/...` and `./internal/agent/restore/...`, and verify with `mise run backend:test-package ./internal/lib/... ./cmd/... ./internal/app/...` and `mise run agent:test`.
- [x] 5.2 Process `./internal/engines/...`, keeping tests on a shared engine-version fixture sequential, and verify with `mise run backend:test-package ./internal/engines/...`.
- [x] 5.3 Process the `workspaces`, `healthcheck`, `notifiers` and `storages` modules, and verify with `mise run backend:test-package ./internal/modules/workspaces/... ./internal/modules/healthcheck/... ./internal/modules/notifiers/... ./internal/modules/storages/...`.
- [x] 5.4 Process the `databases`, `users` and `verification` modules, and verify with `mise run backend:test-package ./internal/modules/databases/... ./internal/modules/users/... ./internal/modules/verification/...`.
- [x] 5.5 Process the `backups` module root and its internal packages, and verify with `mise run backend:test-package ./internal/modules/backups/ ./internal/modules/backups/internal/...`.
- [x] 5.6 Process `./internal/modules/backups/physicalapitests/...`, and verify with `mise run backend:test-group physical-api-1-of-3 physical-api-2-of-3 physical-api-3-of-3`.
- [x] 5.7 Process `./internal/modules/backups/e2e/...` and `./internal/agent/e2e/...`, and verify with `mise run backend:test-package ./internal/modules/backups/e2e/... ./internal/agent/e2e/...`.
- [x] 5.8 Enable `tparallel` in `backend/.golangci.yml`, and verify with `mise run backend:lint`. Then add a temporary test file with five tests: a parallel test with sequential subtests, a sequential test with parallel subtests, a parallel test with parallel subtests that uses `defer` and calls no `t.Cleanup`, a parallel test that mixes parallel and sequential subtests, and a parallel test with sequential subtests and a `//nolint:tparallel // <reason>`. Confirm that `mise run backend:lint` fails, names the first three and does not report the last two, and remove the file.

## 6. Local run scheduling

- [x] 6.1 Add `backend/test-slow-packages.txt`, longest first, from the measured times: `engines/postgresql/physical`, `storages/internal/providers`, `physicalapitests/walstreaming`, `apptest`, `agent/e2e`, the `backups/e2e` packages, the other packages measured at over 60 seconds and the packages whose tests start containers or networks, found with `rg 'testcontainers\.Start|containers\.Start' -g '*_test.go'`. Make `scripts/test.sh` install the frontend dependencies with `pnpm --dir frontend install --frozen-lockfile`, drop `-failfast`, run the listed packages found in its selection and all others in two concurrent `go test` processes and `pnpm --dir frontend test` in a third, wait for all of them, print each output under its process name, exit non-zero on any failure, start the upgrade test only after all pass, and stop all of them and their children on interruption. Extend `scripts/test_sh_test.py` for each of these, including the order and limits passed to `go`, a package missing from the list, an entry that no longer exists and an excluded entry, one failing process while the others finish, and interruption, and verify with `python3 -m unittest discover -s scripts -p '*_test.py'` and `mise run lint-shell`.
- [x] 6.2 Measure the backend part of `scripts/test.sh` with limits of 6, 8 and 12 for the slow process and 16 and 32 for the other process, recording the wall time, the failures and the longest packages. Keep the fastest pair that passes, and verify by running that configuration once more.
- [x] 6.3 Take the longest package from the timeline of 6.2, starting with `engines/postgresql/physical`, and find why its tests do not overlap. Make them overlap, or run it as local shards selected inside `scripts/test.sh`, and verify with a rerun of 6.2's configuration. When shards appear, also assert in `scripts/test_sh_test.py` that they cover every top-level test exactly once, including one without a weight, and that the package does not also run whole, and verify with `python3 -m unittest discover -s scripts -p '*_test.py'`. Repeat for the next package while the gain is at least 30 seconds, and record each result for the final report.

## 7. One routine command

- [x] 7.1 Delete `backend:test-package` from `mise.toml` and `mise-tasks/backend/test-group`, give `mise-tasks/agent/e2e` its own `go test ${CI:+-race}` line with the libedit shim, `TESTCONTAINERS_RYUK_DISABLED=true` and the 30-minute timeout, adapt `scripts/test_sh_test.py` and `scripts/test_group_test.py`, and extend the script test from 2.4 to pin `mise-tasks/agent/e2e` and the absence of both removed tasks. Verify with `python3 -m unittest discover -s scripts -p '*_test.py'`, `mise run lint-shell`, `mise run backend:test-group-packages physical-api`, and that `mise run backend:test-package ./internal/lib/...` reports a missing task.
- [x] 7.2 In `.agent-hooks/review_gate.py`, make `VERIFICATION_FORMS` accept exactly `mise run test` and the two browser commands, and remove the package and shard argument checks that have no remaining use. Adapt `.agent-hooks/review_gate_test.py` so that `mise run test` is accepted, `mise run test ./x`, `mise run test && rm -rf /tmp/x`, `mise run agent:test`, `pnpm --dir frontend test`, `mise run backend:test-package ./internal/lib/...` and `mise run backend:test-group physical-api-1-of-3` are rejected, and a rejected command starts no reviewer. Make the prompts in `.agent-hooks/` and the task rules in `openspec/config.yaml` ask for `mise run test` where a change adds or changes tests, with no finding for running it, then replace the block in this file's `## Change verification` section with `mise run test` and `pnpm --dir frontend test:browser --project=e2e`. Verify with `python3 -m unittest discover -s .agent-hooks -p '*_test.py'`.

## 8. Documentation

- [x] 8.1 In `docs/testing.md`, describe `mise run test` as the one routine command with its slow package list, state where race checks run and how to run one by hand with `go test -race`, give the full command with the libedit shim for repeating one test, replace the mentions of the removed commands, and describe the test server's lost durability and its recovery with `docker compose up -d --force-recreate --renew-anon-volumes backend-db-test`, only after a crash and while no test run is active. Verify with `rg -n -e '-race' -e 'test-package' -e 'test-group ' docs AGENTS.md backend .agents/skills` that every remaining mention is correct.
- [x] 8.2 In `AGENTS.md` and `backend/AGENTS.md`, update the command tables, the rules on which tests to run and on full runs, and the rule against stopping the shared test PostgreSQL server, with the crash recovery as its one exception. In `.agents/skills/testing/SKILL.md`, say that work is checked with `mise run test`, that a failing test is repeated directly, how to reproduce a race that CI reports, and that a subtest is sequential only with a stated reason, linking `docs/testing.md` instead of retelling it. Verify with `rg -n 'test-package|only when the user asks' AGENTS.md backend/AGENTS.md .agents/skills openspec/config.yaml`.

## 9. Result

- [x] 9.1 Run `time mise run test` twice. Verify that both runs pass and that the backend part takes less than the 594-second baseline in both, and report both times with the frontend and upgrade parts. Also report the gain of parallel subtests and of the single contract parse as far as the timelines show them.
- [x] 9.2 Run `mise run lint`, `python3 -m unittest discover -s scripts -p '*_test.py'` and `python3 -m unittest discover -s .agent-hooks -p '*_test.py'`, and verify that all pass.

## Change verification

```sh
mise run test
pnpm --dir frontend test:browser --project=e2e
```
