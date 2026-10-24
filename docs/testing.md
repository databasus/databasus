# Testing

This document describes the kinds of tests Databasus has and what each kind checks. It also explains how tests stay isolated from each other and how runs are organised. How to write a single test is in the [testing skill](../.agents/skills/testing/SKILL.md). The code layout the tests rely on is in [architecture.md](architecture.md).

## Principles

- A test checks what a caller observes: the HTTP API of the backend, the behavior of an engine against a real database server, or a screen of the frontend.
- Our own database, modules and services are never mocked. Fakes replace only systems we cannot run locally and failures that are unsafe to cause for real.
- Every test can run in parallel with any other test of the same run. Single packages and tests also run alongside runs from other worktrees on the same machine; only full runs take turns through a machine-wide lock. A test never takes a shared resource under a fixed name.
- Every check runs locally, and none of them is manual. CI repeats the same tests on a clean machine.

## Layers

Paths under `internal/` are inside `backend/`.

| Layer | What it checks | Where it lives | What is real |
|---|---|---|---|
| Module API tests | the behavior of each module through its HTTP operations, and that every response matches the contract | `internal/modules/<name>/*_test.go`, package `<name>_test` | the application, the internal database, HTTP; external systems are fakes |
| Engine tests | connecting, privileges, dumping and restoring against every supported server version | `internal/engines/<engine>/*_test.go` | the database servers in containers |
| Backup and restore tests | a full backup → restore cycle per engine and version, logical and physical, including SSH tunnels, TLS and failover | `internal/modules/backups/e2e/` | the application, source and target servers, storages in containers, the recovery script |
| Agent tests | the verification agent's job pipeline, and the agent against a real backend | `internal/agent/**`, `internal/agent/e2e/` | the backend from the test harness, Docker or Podman |
| Upgrade tests | an instance of the previous release keeps its data after an upgrade to the current code, and its backups restore | `backend/e2e/upgrade/` | both images, real data written through the API |
| Filesystem tests | the image on the host filesystems and permission setups that users run | `e2e/docker-storage/` | Docker on the host |
| Frontend tests | screens for every answer the contract allows | `frontend/src/**/*.test.tsx` | the components; the network is mocked through `mockApi`, typed by the contract |

Until the contract in `backend/api` exists, frontend tests render components without network mocks, and `mockApi` arrives with the contract.

Module API tests cover most behavior. A unit test without the application is used only for pure logic that has no HTTP entry point, or that is too slow to enumerate through HTTP. Examples are the scheduler's decision function, retention arithmetic, WAL position math, manifest reconstruction and parsers.

## The test application

`apptest.New(t)` builds the whole backend with `app.New`, the same function the server uses. A test therefore sees the real routing, middleware, modules and wiring. The test owns everything the instance uses:

- its own internal database from `testdb.Open(t)`;
- a clock that the test moves forward, so schedules and deadlines need no real waiting;
- fakes for external systems: an email recorder, an HTTP server for webhook and chat notifiers, and a local directory or a MinIO bucket for storages;
- background jobs that never run on their own; the test runs each pass of a job explicitly.

The test talks to the instance over real HTTP through `apiclient.New(t, url)`. The client checks every response against `openapi.yaml` and fails the test on any mismatch. It checks the status, the content type, the body and the field formats. The frontend tests mock responses by the contract, so they rely on this check. The agent tests use `agentclienttest`, which checks responses against `agent.yaml` in the same way.

A test creates its data through the API, the same way a user does. The testing skill lists the few cases where direct SQL is allowed.

## Isolation

- **Internal database.** Databases for tests live on one test PostgreSQL server, shared by every worktree on the machine and never stopped by a test run. `testdb.Open(t)` creates a fresh database with a unique name from a template that has every migration applied. The template is built once per set of migrations, under an advisory lock, and it is keyed by a hash of the migrations. A passing test drops its database. A failing test keeps it and prints its address. Each test process removes databases that are older than an hour and have no connections.
- **Database servers for engines.** Each test package boots one container per engine version, and the package's tests reuse it. The container stops before the next version boots. Booting a server per test is too slow, and booting every version at once does not fit into memory. The servers keep their data in memory with durability turned off. Tests that share a server give a unique name to every object they create: database, role, schema and replication slot.
- **Storages.** Each test uses its own bucket, container or directory.
- **Everything else.** Ports come from `127.0.0.1:0` or `httptest`, and files come from `t.TempDir()`. Containers and processes that a test starts get unique names and labels, and `t.Cleanup` removes them.

## Waiting

Tests never wait for a fixed time. They wait for a condition with `testwait.Until`, up to the test's overall timeout, and the linter rejects `time.Sleep` and `Eventually` in tests. Time that drives behavior comes from the clock in `Deps`. A test of a schedule or a timeout therefore moves the clock instead of sleeping. The testing skill describes how to write a negative check.

## Choosing the tests to run

While working, run the package or the test you need right now. Before handing the work over, run the tests of every package the change touched and of the packages that test the same feature, with `just test-package <packages>` in `backend/` and `pnpm test` in `frontend/`. A package that tests the same feature is one whose tests exercise the changed behavior through the API, such as the backup and restore tests for a change in an engine.

The full run is `just test`. It holds a machine-wide lock, so only one full run goes at a time and the others wait. Single packages and single tests run without the lock. CI runs everything on every pull request.

## Runs

- Backend tests run with `-race`, and a data race fails the test.
- CI runs the same groups as the full run, split into shards by the measured duration of tests. Physical backup tests and WAL streaming tests are the slow groups and get their own shards.
- The upgrade test and the filesystem tests are part of `just test`, and CI runs them on every pull request.

Until the tests of old code move into modules, the upgrade test does not exist yet and the filesystem tests run only through `just test-filesystem`. Old tests still change the shared `testdb` database on the three shared test PostgreSQL servers, so `just test` resets those servers before its tests. The reset waits up to ten minutes for running test processes to end, and a test process that starts during the reset waits for it. The linter rejects `time.Sleep` and `Eventually` only in the target layout, and tests of the packages outside it run with the `testhelpers` build tag, as "Until the migration finishes" in [`AGENTS.md`](../AGENTS.md#until-the-migration-finishes) says.

## Commands

| What | Command |
|---|---|
| Given backend packages | `just test-package <packages>` in `backend/` |
| Everything | `just test` at the root |
| One backend package | `go test -race -count=1 ./internal/modules/<module>/...` in `backend/` |
| One backend test | `go test -race -count=1 -run 'TestName/scenario text' ./internal/modules/<module>/...` in `backend/` |
| Frontend | `pnpm test` in `frontend/`, or `pnpm test -t 'scenario text'` |
| Filesystem tests | `just test-filesystem` at the root |
