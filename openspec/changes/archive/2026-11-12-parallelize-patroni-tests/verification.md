# Verification

Host: AMD Ryzen 9 7950X (16 cores, 32 threads), 62 GiB memory, Fedora, Docker with overlayfs. Go commands ran from the repository root after `source scripts/fedora.sh` and `apply_libedit_shim`, with `.env` exported and `TESTCONTAINERS_RYUK_DISABLED=true`.

## Baseline before the change

A probe of six PostgreSQL 18 backup scenarios with the two-slot semaphores passed in 297 seconds. Over that run the host averaged 2.7% CPU across 32 threads with an 8.7% peak, and used memory rose by about 1.5 GB over the idle baseline. The 6% iowait seen during the run was the host's background level; it stayed the same after the run ended.

## Package split and semaphore removal

- `go -C backend test -count=1 -list '^TestPatroni' ./internal/modules/backups/e2e/postgresqlphysical/patroni` lists the ten family-by-major tests, `TestPatroniFixtureChild` and `TestPatroniFixtureLifecycle`.
- `mise run backend:lint` passed with 0 issues.
- `go -C backend test -v -count=1 -timeout 15m -run '^TestPatroniFixtureLifecycle$' ./internal/modules/backups/e2e/postgresqlphysical/patroni` passed all 11 lifecycle cases in 50.4 seconds, against 94.3 seconds recorded with the semaphore.
- `go -C backend test -v -count=1 -timeout 15m -run '^TestPatroniRecoveryPostgres18$' ./internal/modules/backups/e2e/postgresqlphysical/patroni` passed its three scenarios in 158.8 seconds.

## The runner's limit decides concurrency

`rg -n 'make\(chan struct\{\}|semaphore|capacity' backend/internal/modules/backups/e2e/postgresqlphysical/patroni` returned no match. The PostgreSQL 18 slot family ran twice while a sampler counted distinct running `databasus.patroni-fixture` labels every second:

| `-parallel` | Peak fixtures | Wall time | Scenario durations reported |
| --- | --- | --- | --- |
| 4 | 4 | 63.9 s | 48.8 to 63.8 s |
| 2 | 2 | 137.2 s | 55.6 to 69.6 s |

At `-parallel=2` the reported scenario durations stay near one scenario's length, while the wall time doubles. This confirms that the durations `go test -v` reports leave out the wait for a slot, so they serve as shard weights directly.

## Complete package on this machine

The first attempt at two full runs with the default limit was interrupted by the host, not by the tests. Docker keeps its data in a sparse loop image on the root filesystem, and the root filesystem, already nearly full of Docker caches, ran out of space at 21:49; the host then rebooted. The user pruned the Docker caches. After the reboot, a run failed every backup scenario only in cleanup, with `flush test application logs: ... localhost:9428: connect: connection refused`: the local `.env` sends test-application logs to a VictoriaLogs container that the prune removed and that the repository does not define. The runs below export `OPEN_TELEMETRY_URL=` for the test process only, as `.env.example` leaves it, and do not change `.env`.

A sampler recorded CPU, memory, disk use on `/`, on Docker's filesystem and in `/tmp`, distinct `databasus.patroni-fixture` labels and container count every 2 seconds. It would have stopped the run if free space on `/` or on Docker's filesystem fell below 25 GB; it never did. Default `-parallel` is 32 on this host.

Command, run twice:

```sh
TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -v -count=1 -timeout 15m ./internal/modules/backups/e2e/postgresqlphysical/patroni
```

| Run | Result | Wall time | Peak fixtures | Memory above idle | Docker storage | `/tmp` | CPU average / peak |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | 43 passed, none failed or skipped | 197 s (package 192.2 s) | 32 | +11.0 GB | +8 GB | +0.9 GB | 13.1% / 70.9% |
| 2 | 43 passed, none failed or skipped | 180 s (package 176.4 s) | 32 | +10.4 GB | +8 GB | +0.9 GB | 13.4% / 77.9% |

Each run passed 32 backup scenarios and 11 lifecycle scenarios. No connection errors to the shared test server appeared. After each run no `databasus.patroni-fixture` container, network or volume remained.

Both runs finish in about 3 minutes, within the 5-minute target. Task 2.2 was therefore not needed: start-up contention did not push the run over 10 minutes. Task 2.3 was not needed either: peak memory stayed far below the 32 GB budget. The run is now bound by its longest scenarios, the routing outages with their real cron wait and independent recovery, not by the hardware: CPU stays well below saturation.

## Slow group as the runner runs it

To reorder the slow-package list and check the restored 15-minute limit, every package in `backend/test-slow-packages.txt` ran the way `scripts/test.sh` runs its slow group in a full run, with the same sampler:

```sh
cd backend
TESTCONTAINERS_RYUK_DISABLED=true go test -p=8 -count=1 -timeout 15m <packages in list order>
```

All 34 packages passed in 464 seconds. The longest were `internal/agent/e2e` at 459.5 s, the Patroni package at 272.2 s and `physicalapitests/walstreaming` at 226.6 s. Peak memory rose by 19.1 GB over the idle baseline, Docker storage by 9 GB and `/tmp` by 2.0 GB, with 32 Patroni fixtures and 177 containers at the peak. Average CPU was 25.8% and the peak 93.9%. The list is now ordered by these times. `internal/agent/e2e` stays within the 10-minute limit but above the 5-minute target; it is outside this change.

## Runner and CI selection

- `python3 -m unittest discover -s scripts -p test_sh_test.py` passed 44 tests with the slow and other limits both at 15 minutes.
- `python3 -m unittest discover -s scripts -p test_group_test.py` passed 13 tests. They include the `patroni` group taking its nested packages while `physical` takes the rest, and real coverage of every non-upgrade package exactly once. They also compute the five real `patroni` shards and check that each holds a backup family test and that every Patroni scenario test has a weight.
- Weights are the mean of the two complete runs above. The five shards carry 776, 764, 606, 641 and 676 seconds of local scenario time.

## Documentation and checks

- `rg -n 'Speed\]|#speed|capacity limit|two active' docs AGENTS.md backend/AGENTS.md .agents/skills backend/internal/lib/testcontainers/testdata/patroni/README.md` returned no match.
- `rg -n -- '-timeout 40m' docs/testing.md` returned only the CI remainder-shard command and the provider-root command, which match the CI workflow.
- No CI subsection names individual groups; the remainder-shard commands now say that the same commands repeat a `patroni` shard by name.
- `mise run backend:format` and `mise run python:format` ran; the Python formatter reformatted one changed test file.
- `mise run lint` passed, including backend, frontend, Python, shell and workflow lint.
- `python3 -m unittest discover -s scripts -p '*_test.py'` passed 207 tests.
- `openspec validate parallelize-patroni-tests --strict` passed.

## Change verification

`mise run test` exited 0: every selected backend package passed in the slow and other groups, and the frontend passed 194 files with 2866 tests. The local `.env` sends test-application logs to `localhost:9428`, and the VictoriaLogs container that served it was removed by the Docker prune. A temporary process accepted and discarded those log posts on port 9428 for this run only and was stopped afterwards; nothing in the repository or `.env` changed. No CI run is part of this verification: the five `patroni` shards are first observed after a push.
