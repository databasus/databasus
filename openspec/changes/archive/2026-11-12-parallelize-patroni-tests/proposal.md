## Why

The Patroni physical backup package takes about 21 minutes locally: its last recorded complete run took 1259 seconds for 32 backup scenarios and 11 fixture lifecycle scenarios. A probe of six scenarios on PostgreSQL 18 at the current limit used on average 2.7% of a 32-thread machine and about 1.5 GB of memory. The hardware sits idle because each of the package's two top-level tests allows only two highly available clusters at a time, so at most four run at once. The short lifecycle test finishes within the first minutes, after which only two backup clusters run. The total time is therefore close to the backup scenarios' summed time divided by two: about 42 minutes of backup scenario work, plus about 3 minutes of lifecycle work.

CI has never run this package. All of its backup scenarios sit in one top-level test that has no recorded weight, so the shard balancer counts it as one second and places the whole matrix on one 4 vCPU runner. That runner also runs other physical tests, with the race detector and the 40-minute limit. Expected outcome, not yet observed: that shard takes far longer than today's 2 to 10 minute backend checks and risks its timeout.

The project also has no written resource budget for tests. Nothing says what machine a test run may assume, that saturating it is the goal, or how long a package may take.

## What Changes

- The Patroni package drops its two fixed two-cluster limits. Its scenarios run as Go's parallel test limit allows, so locally the hardware sets the pace.
- The Patroni backup matrix is split into several top-level tests by scenario family and PostgreSQL major, each with a recorded duration weight.
- CI gets a dedicated `patroni` backend group with its own shards on both architectures, and the `physical` group no longer selects the Patroni package.
- The local runner returns its slow-package timeout from 40 to 15 minutes, which the existing local workflow requirement already demands. The slow-package list is reordered by a new measurement.
- `docs/testing.md` gains a section on resource usage and parallelism. It describes the reference developer machine, saturation as the goal with CPU oversubscription accepted and memory as the bound, the parallelism layers of `mise run test`, `mise run test-full` and direct package runs, the CI strategy of parallel jobs, and time targets of about 5 minutes per package with a limit of 10. Inherently long tests remain allowed.
- The testing skill gains one rule: test concurrency is never capped by a constant below what the resource budget allows.

Nothing breaks callers, schemas, configs or APIs. CI check names change: the `physical` shards no longer contain Patroni tests, and new `patroni` shard checks appear.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `ci-pipeline`: the backend groups gain a `patroni` group with its own shards, excluded from `physical`.
- `local-development-workflow`: the Patroni suite finishes within ten minutes on the reference developer machine when run alone.

## Impact

- Patroni test package: `backend/internal/modules/backups/e2e/postgresqlphysical/patroni`.
- Patroni scenario helpers in `backend/internal/app/apptest/physicaltest`, only if a measurement shows that fixture start-up contention misses the limit.
- Runner and CI selection: `scripts/test.sh`, `backend/test-slow-packages.txt`, `backend/test-shard-weights.txt`, `mise-tasks/backend/test-group-packages`, `.github/workflows/ci-release.yml` and their script tests under `scripts/`.
- Documentation: `docs/testing.md` and `.agents/skills/testing/SKILL.md`.
- The change answers to the root `AGENTS.md` and `backend/AGENTS.md`.

## Out of scope

- Changing what any Patroni scenario checks, including its assertions, real-time waits and transfer rate cap.
- Shortening the real-time cron wait of the routing outage scenarios.
- Parallelism of other packages, suites or CI groups beyond documenting how they work now.
- Changing CI runner sizes or the per-runner parallel limit of existing groups.
- Running `mise run test-full`; the user runs it when they want its memory peak measured.
- Observing the first CI run; it needs a push, which the user does.
