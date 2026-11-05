## Context

See proposal.md for the motivation. The state this design starts from:

- Old code reaches the internal database through `storage.GetDb()`. In a test build with the `testhelpers` tag, `config.loadWithoutApp` claims one of eight slots with an advisory lock and points the database at `databasus_w<slot>` (`backend/internal/config/load_without_app_testhelpers.go:84-86`, `:107-122`).
- `scripts/test.sh` prepares those slots before the tests: it removes every labelled test container on the daemon (`scripts/test.sh:47`), runs `cmd/cleanup_test_db` (`:51`), which recreates the slot databases and resets the shared engine servers, and migrates each slot with `goose` (`:52-56`). The exit trap removes every labelled container again (`:8-18`). CI repeats the reset and the slot loop (`.github/workflows/ci-release.yml:425-440`).
- The engine server reset exists because old tests change the shared `testdb` database. For example, provisioning a read-only user revokes `CREATE` on schema `public` from `PUBLIC`, and later `pg_dump` runs fail on that database (`backend/cmd/cleanup_test_db/main.go:1-12`).
- `testdb.CreateDatabase` gives a test an empty database without migrations (`backend/internal/lib/testdb/testdb.go:17`). The tests of `internal/app`, `cmd/databasus` and `lib/database` use it.
- Test containers are created in three places of `backend/internal/util/testing/containers`: `bootContainer` (`containers.go:117`), the traffic control sidecar (`netem.go:142`) and the networks (`containers.go:47`, `postgres_physical.go:235`).
- The test PostgreSQL server `backend-db-test` runs with the default 100 connections (`docker-compose.yml:47-58`). Each process opens a pool of up to 10 connections (`backend/internal/lib/database/database.go:13`).

The change answers to the root `AGENTS.md`, `backend/AGENTS.md`, `docs/testing.md` ("Isolation", "Choosing the tests to run") and `docs/architecture.md` ("Shared code": `testdb` lives in `lib`).

## Goals / Non-Goals

**Goals:**

- The `testdb` behavior `docs/testing.md` describes: a migrated copy per test, a kept database for a failed test, removal of abandoned databases.
- No step before `go test` prepares databases, so `go test`, `just test-package` and `just test` all work from a fresh checkout.
- A full run harms no other run on the machine.

**Non-Goals:**

- Making old tests independent of the shared engine servers' state.
- `testwait`, `apptest` and a `*gorm.DB` helper; they arrive with their first consumers.

## Decisions

### One template per set of migrations, copied per test

`testdb` builds a database `databasus_test_template_<hash>` once, where the hash covers the names and contents of the embedded migrations. The build runs under an advisory lock on the test server. It migrates a database named `..._building` and renames it only after every migration succeeded, so an interrupted build leaves no half-migrated template. Each test then runs `CREATE DATABASE ... TEMPLATE`, which copies files instead of replaying 90 migrations.

Alternatives rejected:

- Migrating every test database from scratch. That costs every test a full migration run.
- Keeping the `goose` step in `scripts/test.sh`. A single `go test` would then need a preparation step, and two worktrees with different migrations would fight over one schema.

### `testdb` returns connection strings

The names follow `docs/testing.md`, which already calls the per-test function `testdb.Open(t)`. `testdb.Open(t)` returns the connection string of a migrated copy, `testdb.OpenEmpty(t)` returns an empty database for the migration tests, and `testdb.OpenForProcess()` returns a migrated copy that lives as long as the process. Every current consumer opens its own pool: `app.New`, the binary under test in `cmd/databasus` and `database.Migrate`. Returning `*gorm.DB`, as in cloud, would open a pool nobody uses. `CreateDatabase` and `ReplaceDatabaseName` are replaced, and their callers move to the new functions.

### Old code gets one database per test process

In a test binary, `config.loadWithoutApp` takes its database from `testdb.OpenForProcess()` instead of claiming a slot. The slot claim, `TestParallelWorkers` and the slot part of `cmd/cleanup_test_db` are deleted. Old tests inside one package keep sharing one database, exactly as they share a slot today, so their behavior does not change.

The process has no hook at exit, because most packages have no `TestMain`. Its database is therefore left behind and removed by the abandoned-database cleanup an hour later. Each copy takes about 8 MB on the disk of the test server, which stays well within a disk volume over a day of full runs.

Alternatives rejected:

- Keeping the slots. Then `-p` stays tied to eight prepared databases, and a full run still has to wipe databases other runs use.
- A `TestMain` in every package that drops the database. That adds a file to over a hundred packages that move or disappear in the coming milestones.

### Abandoned databases are found by name and connections

A test database is named `databasus_test_<unix seconds>_<8 hex>`, because `pg_database` stores no creation time. Every test process runs the cleanup once, the first time it asks for a database, whether or not it builds the template. The cleanup drops every database that matches the pattern, is older than an hour and has no row in `pg_stat_activity`. It takes no lock and ignores errors: a neighbouring process may drop the same database, and a `DROP DATABASE IF EXISTS` that loses that race does no harm. Template names do not match the pattern, so templates stay. Templates of older migration sets are small and are dropped by hand when needed.

### A run labels its containers and removes only those

`scripts/test.sh` generates a run ID and exports it as `DATABASUS_TEST_RUN_ID`. The `containers` helpers add the label `databasus.test-run=<id>` to every container and network they create, when the variable is set. The exit trap removes containers and networks with that label only. The removal of every labelled container at the start of a run is deleted. `just clean-testcontainers` stays for removing leftovers by hand, and its comment says it removes the containers of every run on the machine.

Alternatives rejected:

- Turning Ryuk on. It reaps each test binary's containers when the binary dies, but the earlier tooling change kept it off on purpose without recording why, and CI mirrors that choice. Turning it on is a separate decision with its own risk.
- Filtering by the testcontainers session label. Each test binary gets its own session ID, and the script cannot learn them.

### The engine server reset is guarded by a readers-writer lock

Every old-code test binary takes a shared `flock` on `${TMPDIR:-/tmp}/databasus-test-fixtures.lock` when it loads its configuration, and holds it until the process exits. The exclusive side lives only in `scripts/test.sh`. CI runs `cmd/cleanup_test_db` directly without the lock, because a CI runner runs one group and nothing else. Only a test binary takes it, checked with `testing.Testing()`, so `cmd/cleanup_test_db` itself never waits for the lock that guards it. `scripts/test.sh` runs the reset under `flock -x` on the same file, waiting up to `DATABASUS_FIXTURE_RESET_WAIT_SECONDS`, 600 by default, so its test can shorten the wait. The reset thus waits for running package processes, and a process that starts during the reset blocks at configuration load until the reset ends. The exclusive lock is released before the run's own tests start.

Every old-code test process takes the lock, not only the ones that touch an engine server. This over-approximation only delays a reset. Finding exactly the processes that use the servers would require changes across about a hundred tests.

Alternatives rejected:

- Dropping the reset. State that old tests leave on the shared `testdb` database then accumulates and breaks later `pg_dump` runs.
- Moving old tests to unique databases on the shared servers. This is the target from `docs/testing.md`, but it touches about a hundred tests that move in milestones 8 to 10.

The plan's table of temporary relaxations gets a row for this reset; it is removed when those tests move.

### Full runs take turns through `flock`

`scripts/test.sh` takes an exclusive `flock` on `${TMPDIR:-/tmp}/databasus-full-test.lock` for its whole run. It tries without waiting first. When another run holds the lock, it prints that it waits for another full run and then blocks. Single package runs do not take this lock.

The alternative was a scheduler script like `run_exclusive.py` in cloud, which also reports queue positions. A one-line `flock` covers the need, and `flock` ships with util-linux on Fedora and Ubuntu.

### `-p` stays 8 and the test server allows 300 connections

`-p` keeps bounding how many packages, and therefore roughly how many engine containers, run at once. `TEST_PARALLEL_WORKERS` disappears, and `-p=8` stays a literal in `scripts/test.sh` and CI. `backend-db-test` gets `-c max_connections=300`. That covers eight processes with a pool of 10 for old code, the pools of `apptest` instances later, and other worktrees' package runs. The server stays on disk rather than on `tmpfs`, so leftover databases cost disk, not memory.

### Agent rules follow the lifted ban

- `AGENTS.md`: the bullet on per-worker databases in "Until the migration finishes" is deleted. "Several agents, one machine" already states the target rule: an agent runs `just test` only when the user asks.
- `docs/testing.md`: the last paragraph of "Runs" no longer says that `scripts/test.sh` takes no lock.
- `openspec/config.yaml`: the tasks rule keeps the full suite out of tasks, with a new reason. The suite takes too long for a task's verification and runs only when the user asks.
- `.agent-hooks/review_gate.py`: the comment over the verification forms gives the same reason. The forms and the hook specs do not change, because the hooks still never run the full suite.

## Risks / Trade-offs

- [A long package run delays a full run's reset by up to ten minutes] → `scripts/test.sh` gives up after 600 seconds with a message, resets nothing, and the developer reruns after the other run ends.
- [A test binary killed with SIGKILL keeps its lock until the kernel closes its file] → `flock` locks are released when the process dies, so nothing stays locked.
- [A run killed with SIGKILL leaves labelled containers] → `just clean-testcontainers` removes them by hand, as it does today.
- [Slot databases `databasus_w0..7` stay on developers' test servers] → They match neither the test pattern nor any code. Recreating the `backend-db-test` container drops them, and `docs/testing.md` does not mention them.
- [A neighbouring process drops a database another process is about to connect to] → Only databases older than an hour without connections are dropped, and a fresh copy is connected within seconds of its creation.

## Migration Plan

Developers copy the new `.env.example`, because `GOOSE_TEST_DBSTRING` and `TEST_PARALLEL_WORKERS` disappear. `docker compose up -d` recreates `backend-db-test` with the new connection limit. Rollback is a revert of the change.
