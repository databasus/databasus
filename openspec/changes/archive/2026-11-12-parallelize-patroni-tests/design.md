## Context

See proposal.md for the motivation and the measurements behind it.

The Patroni package bounds its own concurrency with two semaphores of two slots, one for backup scenarios and one for fixture lifecycle scenarios (`backend/internal/modules/backups/e2e/postgresqlphysical/patroni/backups_test.go:17`, `fixture_test.go:14`). No other backend package has such a cap. Go's `-parallel` flag already limits how many parallel tests run at once in one test binary. A direct package run leaves it at its default, which equals GOMAXPROCS. CI sets it to 2 through `TEST_PARALLELISM` (`.github/workflows/ci-release.yml:733`, `:759`).

One scenario starts its own cluster: etcd, two Patroni nodes and HAProxy, plus its own test application and, for recovery checks, a restore target. In the probe, one active scenario used about 0.7 GB and well under one processor thread. Approximate scenario durations on PostgreSQL 18 were 45 to 170 seconds. The longest is the routing outage case, because its schedule waits for a real cron minute 60 to 120 seconds ahead (`backend/internal/app/apptest/physicaltest/patroni_retry.go:106`). The fixture starts its nodes one after the other and builds the node image from its Dockerfile for each node (`backend/internal/lib/testcontainers/patroni.go:112`, `:125`). That start took about 20 seconds per scenario.

Locally, `scripts/test.sh` runs three suites at once: the slow backend packages with `-p=8`, the other backend packages with `-p=16` and the frontend tests (`scripts/test.sh:122-131`). The Patroni change raised the slow group's timeout to 40 minutes (`scripts/test.sh:126`). The existing local workflow requirement still says every package gets 15 minutes.

In CI, `mise-tasks/backend/test-group-packages:15` puts the Patroni package into `physical`. Its shards are split by top-level test with the weights in `backend/test-shard-weights.txt`. Neither Patroni top-level test has a weight there. Recent `physical` shards took 2 to 3 minutes, and the other backend checks took up to 10.

The change answers to the root `AGENTS.md` and `backend/AGENTS.md`, including the testing skill's `t.Parallel()` rule. It changes no application code and no architecture described in `docs/architecture.md`.

## Goals / Non-Goals

**Goals:**

- The Patroni package run alone finishes within 10 minutes on the reference machine, with about 5 minutes as the target.
- CI spreads the Patroni matrix over its own shards, so no backend check carries the whole matrix.
- `docs/testing.md` states the resource budget and the parallelism model for every local and CI test run.

**Non-Goals:**

- Rebalancing other slow packages or changing their limits.
- Measuring `mise run test-full`. An agent does not run it on its own, so the user measures its memory peak when they choose to.

## Decisions

### Concurrency comes from the test runner, not from the package

Both semaphores are deleted. Every Patroni top-level test and scenario subtest calls `t.Parallel()`, and `-parallel` alone decides how many run at once. The CI limit of 2 is unchanged by this, while a local run uses the machine's GOMAXPROCS. On this 32-thread machine that is 32 scenarios at about 0.7 GB each, roughly 22 GB, which fits the 32 GB budget.

Alternatives:

- **A larger constant, such as 12.** Rejected because it repeats the current problem at a different number: one machine still idles and another may overload.
- **A cap computed from free memory at run time.** Rejected because the measured footprint fits the budget at GOMAXPROCS, and this adds code that guesses the host's state. If the measurement in tasks shows a peak over the budget, the runner gets an explicit `-parallel` for the slow group. The package itself stays uncapped, as the spec requires.

### The matrix splits by family and PostgreSQL major

`TestPatroniPhysicalBackups` becomes ten top-level tests: five scenario families, each for PostgreSQL 17 and 18. The names follow `TestPatroni<Family>Postgres<major>`, and scenario subtest names stay as they are today.

| Family | Top-level tests | Scenarios per major |
| --- | --- | --- |
| Recovery | `TestPatroniRecoveryPostgres17`, `TestPatroniRecoveryPostgres18` | initial schema and sequences, two switchovers, primary loss |
| Independent recovery | `TestPatroniIndependentRecoveryPostgres17`, `TestPatroniIndependentRecoveryPostgres18` | target, latest, backup, external configuration, referenced authentication |
| Slots | `TestPatroniSlotsPostgres17`, `TestPatroniSlotsPostgres18` | protected, absent, unprotected, lost |
| Interrupted backups | `TestPatroniInterruptedBackupsPostgres17`, `TestPatroniInterruptedBackupsPostgres18` | interrupted FULL, interrupted INCREMENTAL |
| Routing outages | `TestPatroniRoutingOutagesPostgres17`, `TestPatroniRoutingOutagesPostgres18` | outage during FULL, outage during INCREMENTAL |

`TestPatroniFixtureLifecycle` stays as it is, apart from losing its semaphore. `TestPatroniFixtureChild` stays as the child-process entry point. In a normal run it returns at once, so its shard weight is negligible.

Alternatives:

- **Families only, with the major as a subtest.** Rejected because five pieces of 5 to 11 minutes of scenario time each balance poorly across five shards.
- **One top-level test per scenario.** Rejected because 32 names add weight-file entries without improving balance, since the family pieces are already shorter than a shard.

### A dedicated `patroni` CI group with five shards

`test-group-packages` gains a `patroni` group that selects the Patroni package and any package nested under it, and `physical` excludes that subtree. The workflow matrix gets five `patroni` shards per architecture on the same runner profile, with `TEST_PARALLELISM` at 2.

The shard count comes from an estimate, not an observed run. The last complete local run implies about 45 minutes of scenario time: 1259 seconds at two backup scenarios at a time gives about 42 minutes, and the lifecycle tests add about 3. Assuming the race detector and a 4 vCPU runner make it 1.5 times slower, that is about 68 minutes. At two scenarios at a time this is about 34 minutes of shard time. Five shards give about 7 minutes each, plus job setup, which is in line with today's backend checks.

Alternatives:

- **Keep Patroni in `physical` and add `physical` shards.** Rejected because durable, disk-backed HA fixtures would share 8 GB runners with the PostgreSQL 17 and 18 physical binaries. A failure would also be harder to attribute, and `physical` would gain shards for tests unrelated to its own.
- **A higher `TEST_PARALLELISM` for `patroni` with fewer shards.** Rejected for now because the switchover and transfer-overlap scenarios are timing-sensitive on a 4 vCPU runner under the race detector. Changing per-runner limits is out of scope.
- **One unsharded `patroni` job.** Rejected because it is estimated at about 34 minutes.

### Shard weights are the sum of scenario durations

For each new top-level test, the weight is the sum of the durations `go test -v` reports for its scenario subtests in one local run, in seconds. Go leaves the wait for a `-parallel` slot out of a parallel subtest's reported duration: `t.Parallel()` records the elapsed time before blocking and restarts the clock after release. The reported durations therefore measure scenario work, while contention between top-level tests running concurrently inflates their own wall time. The comment at the head of `backend/test-shard-weights.txt` will say how the weight is measured for a test with parallel subtests.

Alternative: a top-level test's wall time from a package run. Rejected because it depends on what else ran beside it.

### The slow-package timeout returns to 15 minutes

`scripts/test.sh` gives the slow group `-timeout 15m` again, matching the existing requirement, and its script test asserts that. The Patroni command in `docs/testing.md` uses the same limit. A package that needs more than 15 minutes now fails loudly instead of being absorbed.

Alternative: keep 40 minutes. Rejected because it contradicts the spec and would hide a regression of the new target.

### Fixture start-up is optimized only if the limit is missed

The fixture keeps building and starting its nodes as it does today. If a measured package-alone run exceeds 10 minutes and the logs show start-up contention, the implementation builds the node image once per test process and starts both nodes together. Those optimizations save an estimated 5 to 10 seconds per scenario, while the semaphore costs minutes. A run between 5 and 10 minutes is accepted and reported.

Alternative: do them unconditionally. Rejected because the need is unproven and they touch shared fixture code that the lifecycle tests guard.

### Resource usage gets its own section in `docs/testing.md`

A new section, "Resource usage and parallelism", replaces the "Speed" subsection and states:

- the reference machine: about 64 GB in total with 32 GB free for tests, the rest for compilation and the desktop, and at least 10 performance cores;
- saturation as the goal: CPU load above the core count is acceptable, memory free for tests is the bound, and no constant limits concurrency below that budget;
- the time targets: a package, or a few packages run together, take about 5 minutes, with 10 minutes as the limit, and inherently long tests stay allowed but must not stretch the critical path;
- the local layers: the three concurrent suites of `mise run test`, the slow and other package limits, `-parallel` inside each binary, the per-test databases and containers, sequential engine-version scopes, and the sequential image and browser suites of `mise run test-full`, with the reasons each limit exists;
- CI: parallel jobs on separate runners, groups and shards balanced by weights, and a small per-runner `-parallel`.

The principle bullet about parallel tests links to that section. The link to `machine-setup.md` moves into it. The testing skill's `t.Parallel()` rule gains one sentence: concurrency is not capped by a constant below the resource budget.

Alternative: put the budget in `AGENTS.md`. Rejected because the root rules already route test topics to `docs/testing.md` and the testing skill.

## Risks / Trade-offs

- [Timing-sensitive scenarios flake under heavy CPU oversubscription] → They wait for explicit evidence such as backend PIDs, streamed bytes and observed role changes. The implementation runs the complete package twice and treats any flake as a defect to fix at its cause.
- [Peak memory of a local package run exceeds the budget at a high GOMAXPROCS] → The measurement task records peak memory. Above the budget, the runner sets `-parallel` for its slow group and the docs state why.
- [`mise run test-full` runs Patroni beside seven other slow packages and its peak is unmeasured] → Out of agent scope; the docs state the budget, and the user measures it with the documented command.
- [Simultaneous container starts slow down the Docker daemon] → The conditional fixture start-up decision covers it.
- [Many test applications at once exhaust connections to the shared test PostgreSQL server] → The server allows 1000 connections. The measurement run checks for connection errors.
- [The CI estimate is wrong] → The first CI run after the user's push shows the shard times. The shard count is one number in the workflow and the group script, and the weights rebalance automatically.

## Migration Plan

Nothing is deployed. CI check names change: five `patroni` shard checks per architecture appear, and release gating picks them up through the backend matrix. Rollback is a revert of the commit.
