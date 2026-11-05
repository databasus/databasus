## Why

Backend test packages share eight internal databases, one per worker slot, and a full run starts by wiping those slots, every test container on the machine and the shared database servers the tests use. A full run therefore breaks every test run of every other worktree, so agents are forbidden to run `just test` at all. Milestone 3 of the backend rebuild needs a database per test before `apptest` can exist, and agents need a full run that cannot hurt anyone else.

## What Changes

- A test gets a fresh internal database copied from a template that has every migration applied. The template is built once per set of migrations. A passing test drops its database, a failing test keeps it and prints its address, and databases left behind for over an hour are removed.
- Tests of code outside the target layout get one such database per test process instead of a worker slot. **BREAKING** for local setups: the slot claim, `TEST_PARALLEL_WORKERS`, `GOOSE_TEST_DBSTRING`, the slot reset in `cmd/cleanup_test_db` and the per-slot `goose` loops in `scripts/test.sh` and CI are deleted. The number of packages that run at once is set by `-p` alone.
- A full run removes only the test containers it started. Removing every labelled container on the machine, at the start of a run and on interrupt, is deleted.
- The reset of the three shared test PostgreSQL servers (their databases, roles and replication slots) stays, because old tests change the shared `testdb` database on them. It now waits for every running test process that uses those servers to finish, and a test process that starts during the reset waits for it.
- Full runs take turns through a machine-wide lock. The rule that an agent never runs `just test` is lifted; the agent runs it only when the user asks, as the target rules already say.
- The test PostgreSQL server allows enough connections for packages that each hold their own database pool.

## Capabilities

### New Capabilities

- `backend-test-isolation`: how backend tests get their internal databases and how test runs from several worktrees share one machine without breaking each other.

### Modified Capabilities

- `local-development-workflow`: `just test` prepares no per-worker databases, waits for another full run, and on interrupt removes only its own containers.

## Impact

- Code: `backend/internal/lib/testdb`, `backend/internal/config/load_without_app_testhelpers.go`, `backend/internal/util/testing/containers`, `backend/cmd/cleanup_test_db`, `backend/internal/app` and `backend/cmd/databasus` tests, `backend/internal/lib/database` tests.
- Tooling: `scripts/test.sh` and its new test `scripts/test_sh_test.py`, `backend/Justfile`, `docker-compose.yml`, `.env.example`, the backend test job and the `lint-shell` job in `.github/workflows/ci-release.yml`.
- Agent rules and docs: `AGENTS.md`, `docs/testing.md`, `openspec/config.yaml`, the comment on the verification forms in `.agent-hooks/review_gate.py`. The hooks still never run the full suite, because it does not fit their time limit.
- Answers to the root `AGENTS.md` and `backend/AGENTS.md`.

### Out of scope

- `apptest`, the contract and its clients: change `api-contract-pipeline`.
- The upgrade test and the refusal to start on a newer schema: change `upgrade-safety`.
- Making old tests use unique objects on the shared engine servers, which would remove the reset. Their tests move in milestones 8 to 10.
- Running tests of one package in parallel.
- A cap on the number of test containers beyond `-p`.
