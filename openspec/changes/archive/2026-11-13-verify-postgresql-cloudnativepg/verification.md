## Environment and coverage

Verification ran on 2026-11-13 on native Linux amd64: Fedora kernel
`7.2.11-200.fc44.x86_64`, 32 logical CPUs and 64 GB RAM. The test budget is
32 GB for the entire local run. Go was `go1.27.4 linux/amd64`; the Docker
client and server were both `29.9.2`. The host had
`fs.inotify.max_user_instances=1024` and
`fs.inotify.max_user_watches=1048576`.

The lifecycle checks observed k3s `v1.36.5+k3s1` and CloudNativePG `1.30.1`
(commit `2a35abb46`). Executing `postgres --version` inside the running CNPG
pods reported PostgreSQL `17.11 (Debian 17.11-1.pgdg13+2)` and
`18.6 (Debian 18.6-1.pgdg13+2)`. Recovery used the catalog's official
PostgreSQL image of the matching major on the same Debian release. The image
catalog and fixture README own the immutable references and platform digests.

These are native amd64 scenario results. The workflow selects native amd64 and
arm64 runners, but no native arm64 scenario run was performed in this session.
Other Kubernetes distributions, cloud storage and multi-node Kubernetes were
not exercised.

## Complete CNPG measurements

Commands ran from the repository root after `source scripts/fedora.sh`,
`apply_libedit_shim` and exporting `.env`. Each measurement set a unique
`DATABASUS_TEST_RUN_ID` for resource inspection and disabled Ryuk. The command
was:

```sh
TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -count=1 -timeout 15m -json \
  ./internal/modules/backups/e2e/postgresqlphysical/cnpg
```

The three initial measurements used Go's default parallel test limit of 32, with no other test
suite running. Each passed all 60 leaf scenarios across the recovery, access,
slots,
independent recovery, interrupted-backup, routing-outage and fixture families,
plus the no-op parent-process entry of the child-process fixture. None
reported a skipped or failed test.

| Measurement | Package duration | Host memory baseline / peak, MiB | Increase, GiB | Running CNPG clusters | Observed k3s |
| --- | ---: | ---: | ---: | ---: | ---: |
| `default-5` | 586.317 s | 14815 / 29950 | 14.78 | 26 | 10 |
| `default-6` | 549.716 s | 15439 / 33188 | 17.33 | 32 | 12 |
| `default-7`, after the race fix | 489.705 s | 14195 / 29426 | 14.87 | 30 | 16 |

`default-6` took 552 s including command startup. The post-fix `default-7`
took 491 s including command startup. A daemon restart interrupted
the wrapper of `default-5`; its Go process continued, and its final JSON
package event established success and the duration above. The orphaned sampler
was stopped explicitly after completion.

The two-second sampler recorded host CPU, memory, filesystem usage and the
run-labelled k3s containers. A running CNPG cluster is counted as a distinct
namespace with a CNPG instance pod in Kubernetes' `Running` phase. In
`default-5` and `default-6`, both cluster counts exclude lifecycle children that
intentionally replace the outer run ID to test isolation. Those children also
create PostgreSQL clusters. Both columns therefore describe outer-labelled
resources, not the complete nested fixture peak; host memory includes every
child. The additional post-fix `default-7` sampled every fixture label,
including those children, and observed total peaks of 16 k3s and 30 CNPG
clusters. All three resource types were empty before that run. Filesystem
values describe the filesystems containing Docker storage and `/tmp`, not a sum of fixture directory sizes.

| Measurement | Peak busy CPU | Peak load average | Docker filesystem baseline / peak, MiB | `/tmp` filesystem baseline / peak, MiB |
| --- | ---: | ---: | ---: | ---: |
| `default-5` | 77% | 41.80 | 168558 / 168667 | 2728 / 3302 |
| `default-6` | 82% | 47.30 | 168558 / 168677 | 2520 / 3478 |
| `default-7` | 81% | 57.11 | 168558 / 168671 | 2551 / 3121 |

All three package durations meet the ten-minute limit, with little time margin;
they do not meet the five-minute target. All three measured memory increases stay
below the 32 GB test budget. Four CNPG clusters per pooled k3s remained usable
through the complete matrix without startup or scenario-wait failures.

Earlier successful comparison runs with `-parallel=4 -timeout 90m` and
`-parallel=2 -timeout 150m` took 1683 s and 3433 s. They justify removing the
fixed CI limit of two, not using those slower settings as acceptance criteria.
Earlier default-limit runs at 512 inotify instances failed lifecycle probes
with `error creating fsnotify watcher: too many open files`; they establish no
successful complete-package result. The measurements above used the documented
1024-instance limit.

Weights in `backend/test-shard-weights.txt` use the rounded-up sum of leaf durations
from the latest complete run (`coverage-cnpg`). The child entry receives the
minimum weight of one. All 19 `TestCNPG*` entries have a weight. The five
current shard weights are 1985, 1955, 1976, 1972 and 2001 seconds. Script tests
prove that every shard receives backup scenarios and that selection remains
complete and disjoint. Earlier assignments and their checks below are retained
as preceding measurements; `coverage-shards.json` records the current selection.

## Complete CNPG run after the first review repairs

The same documented command passed the implementation after the first review in **481.764 s**
(484 s including startup), with **62 leaf scenarios**, the no-op child entry
and all 16 top-level tests passing. There were no skips or failures. This run
includes completed incremental artifacts in initial recovery, observed WAL TLS
refusal, physical plaintext refusal, cancellation after cluster readiness and
a killed parent whose child also exits.

The sampler began with no k3s fixtures and recorded all fixture-labelled
resources. Host memory rose from 12195 to 28893 MiB, **16.31 GiB** above the
baseline, within the 32 GB test budget. Peaks were **20 k3s** and **32 running
CNPG clusters**, measured separately from Go's default limit of 32 parallel
tests. Busy CPU reached 75%, load average 43.04; the Docker filesystem rose
from 168558 to 168688 MiB and `/tmp` from 2509 to 3326 MiB. No other test
suite overlapped; format and lint checks ran during startup. Logs and samples
are `/tmp/cnpg-measure/final-cnpg.jsonl` and `final-cnpg.tsv`.

The earlier whole-package race run (`repairs-cnpg`) reproduced the mutable
TLS-proxy connection race and showed that an active replacement slot does not
establish continuity of an older WAL interval. It also exposed an observer
that watched only the receiver's later connection, while TLS failed during
the WAL worker's earlier slot inspection. These failures were repaired rather
than counted as successful coverage. Four certificate cases subsequently
passed with `-race` in 143.082 s (`repairs-certificates`), and the Docker exec
and TLS startup cancellation tests passed with `-race` in 2.945 s
(`proxy-exec-race.log`). `design.md` records the reproduced defects and their
causes.

## CI configuration and race check

The CNPG jobs use 8-vCPU / 16-GB runners on both architectures and retain five
shards. Their Go command has no `-parallel` override. Smaller existing jobs keep
their runner sizes. Runners up to 16 vCPUs / 32 GB remain available if measured
load requires them.

Local shard checks use `GOMAXPROCS=8` to match the default test parallel limit
of that runner. They do not emulate a native CI runner's CPU quota or establish
native ARM performance. The selected command for shard 1 is:

```sh
pattern=$(mise run --quiet backend:test-group-tests cnpg-1-of-5)
GOMAXPROCS=8 TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -race -p=8 \
  -count=1 -failfast -timeout 15m -json -run "$pattern" \
  ./internal/modules/backups/e2e/postgresqlphysical/cnpg
```

The first execution reproduced the pool's startup-error publication race and
failed. The fix and reproducer are recorded in `design.md`; the original log
is `/tmp/cnpg-measure/ci-1-before-fix.jsonl`. The identical post-fix shard
passed in 326.337 s (329 s including command startup), with no race, failed test or skipped test. Its host memory increased
from 13980 to at most 21299 MiB, about 7.15 GiB. `mise run lint` overlapped
part of this run, so that increase also includes the lint processes and is not
a fixture-only measurement.

The same command with `cnpg-2-of-5` and `cnpg-3-of-5` ran the fixture-heavy
shards, each with `GOMAXPROCS=8`. Both ran at the same time on the local host:

| Shard | Package duration | Command wall time | Result |
| --- | ---: | ---: | --- |
| 1 | 326.337 s | 329 s | passed, no race or skip |
| 2 | 484.562 s | 485 s | passed, no race or skip |
| 3 | 329.252 s | 330 s | passed, no race or skip |

In that preceding assignment, shard 3 had the largest recorded weight and shard 2 contained fixture controls,
including the two-k3s pool-capacity scenario. The aggregate host memory during
shards 2 and 3 rose from about 15206 MiB to at most 25676 MiB, about 10.23 GiB
for both checks together. This is an aggregate measurement, not a per-shard
memory allocation. It supports keeping the 16-GB runner rather than selecting
the available 32-GB size. Both checks removed their resources: after both
finished, the three fixture-label queries below again returned no IDs.

After refreshing all weights from `final-cnpg`, the preceding assignment was
checked again with the same race-enabled command and `GOMAXPROCS=8`:

| Preceding shard | Selected families | Package duration | Command wall time | Result |
| --- | --- | ---: | ---: | --- |
| 1 | fixture cleanup, PostgreSQL 17 slots, PostgreSQL 18 interrupted backups | 316.887 s | 320 s | passed, no race or skip |
| 5 | fixture controls, PostgreSQL 17 and 18 recovery, child entry | 429.771 s | 433 s | passed, no race or skip |

Both ran concurrently. Aggregate host memory rose from about 12723 to at most
24174 MiB, **11.18 GiB** for both checks together, within a single selected
runner's 16-GB budget. Their separate run-label counts included all lifecycle
children: shard 1 peaked at 8 k3s and 8 running CNPG clusters, and shard 5 at
4 k3s and 8 clusters. Those are separate peaks, not a claim that they occurred
at the same instant. Both durations meet the ten-minute limit. Evidence is in
`/tmp/cnpg-measure/final-ci-{1,5}.jsonl`, the matching TSV files and
`final-shards.json` (those exact preceding selections). These checks cover the new
parent-death and live-cancellation probes, the repaired slot oracle and TLS
proxy, and the incremental bundle on both majors.

The CI job itself keeps its 40-minute failure timeout; the local checks used
15 minutes. Neither is a claim that native CI runner timing has been measured.

## Complete Patroni regression run

The complete Patroni package ran separately, using the same Fedora preparation
and environment as CNPG:

```sh
TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -count=1 -timeout 15m -json \
  ./internal/modules/backups/e2e/postgresqlphysical/patroni
```

The initial run passed on PostgreSQL 17 and 18 in 161.245 s, or 162 s
including command startup, with no failed or skipped test. The fixture output reported Patroni
`4.1.5`, etcd `3.6.16` and HAProxy `3.2.26-3e0f11a`. This checks the shared
scenario refactoring and recovery behavior against the existing manager. Host
memory rose from 14784 to at most 26174 MiB, about 11.12 GiB. The log and
samples are `/tmp/cnpg-measure/patroni-final.jsonl` and
`/tmp/cnpg-measure/patroni-final.tsv`.

After the review repairs, the identical whole-package command passed again in
**199.999 s**, or **202 s** including startup, with all **55 test pass events**
and no failures or skips. This includes the shared FULL → INCREMENTAL → WAL
initial recovery on both majors and the repaired TLS proxy. Memory increased
from 12072 to 23808 MiB, about **11.46 GiB**. Evidence is retained in
`/tmp/cnpg-measure/final-patroni.jsonl` and `final-patroni.tsv`.

## Earlier completed checks

- `mise run test`: the final post-review run passed in **406 s** total; slow
  backend packages took 378 s, other backend packages 74 s, frontend 75 s.
  All 194 frontend files and 2870 tests passed. Aggregate host memory rose
  from 12253 to at most 26445 MiB, **13.86 GiB** above baseline,
  within the entire run's 32 GB test budget. Transcript and samples are
  `/tmp/cnpg-measure/mise-test-post-review.log` and
  `mise-test-post-review.tsv`. The earlier pre-review run also passed in 429 s.
- `mise exec -- python3 -m unittest discover -s scripts -p '*_test.py'`:
  219 tests passed in 33.267 s after refreshing the weights; the subsequently
  refactored CI execution tests also passed all seven cases in 0.515 s.
- `mise run backend:format` and `mise run python:format`: passed.
- `mise run lint`: passed, including backend, frontend, Python, shell and
  workflow checks; the backend reported zero issues.
- `mise run verify-deps`: all known declaration and client-file checks passed.
- `openspec validate verify-postgresql-cloudnativepg --strict`: passed.
- `git diff --check`: passed.

After organizing access scenarios and the six CNPG test families into separate
files, their test names, scenario bodies and all five shard selections stayed
unchanged. The CNPG package compiled and ran its no-op child entry in 0.063 s;
this was a compilation check, not another full scenario run. The 16 group
selection tests passed in 1.823 s, and the real-container cancellation/proxy
checks passed again with `-race` in 2.386 s. Backend formatting and lint passed
with zero issues. Evidence is in `/tmp/cnpg-measure/*-structure.log`.

## Cleanup evidence

After `default-5`, the following queries returned no IDs:

```sh
docker ps -aq --filter label=databasus.k3s-fixture
docker network ls -q --filter label=databasus.k3s-fixture
docker volume ls -q --filter label=databasus.k3s-fixture
```

The full runs also passed the lifecycle probes for successful cleanup, partial
startup failure, cancellation, a killed child, run-label propagation and two
child runs that leave each other's resources intact. Deliberately missing
prerequisites failed in the child and were asserted by the parent; no scenario
was skipped. The startup/cleanup probes check their own temporary directories
as well as Docker resources. Measurement logs and sample tables are retained
under `/tmp/cnpg-measure` as verification evidence.

After the post-fix full CNPG run and the full Patroni run, queries for
`databasus.k3s-fixture`,
`databasus.test-run=cnpg-measure-default-7-1794555140` and
`databasus.test-run=cnpg-measure-patroni-final-1794555647` each found zero
containers, zero networks and zero volumes. The shared test PostgreSQL server
was left running.

After the final post-review CNPG, Patroni and CI checks, the fixture-label
query and the exact run labels below each returned **zero containers, zero
networks and zero volumes**:

- `cnpg-measure-final-cnpg-1794558381`
- `cnpg-measure-final-patroni-1794558889`
- `cnpg-measure-final-ci-1-1794559115`
- `cnpg-measure-final-ci-5-1794559115`

The complete query results are `/tmp/cnpg-measure/cleanup-final.json`. The
shared PostgreSQL server and unrelated development containers remained running.

`mise run test-full` was not run. The complete CNPG and Patroni packages above
were executed directly in addition to the routine command.

## Follow-up review repairs

The fixture checks now separate Kubernetes lifecycle, CNPG controls and pool
capacity into distinct files and top-level tests. The shared slot lookup lives
with the existing replication-slot checks. At that stage there were 17 top-level CNPG
tests; the follow-up weights and selections superseded the
16-test assignment above.

The review's claim that promoted fields in the source constructors do not
compile was rejected. The repository's pinned Go 1.27.4 compiled both the
package and a minimal independent reproduction
(`/tmp/cnpg-measure/promoted-literal.go`). Moreover, the backend linter's
`modernize/embedlit` check requires the existing promoted-field form and
rejects an explicit nested embedded literal. Both source constructors keep
the form accepted by the pinned compiler and linter.

The additional same-timeline slot-gap cases passed on PostgreSQL 17 and 18
with `-race` in 171.970 s (173 s command wall time), with no failure or skip.
The eligible FULL and reachable prefix remain positive controls, while later
archived transactions share the refused target's timeline. The API reports
`WAL gap before target` for the gap even after a later FULL. Evidence is
`/tmp/cnpg-measure/gap-fixed.jsonl` and its matching TSV.

The stalled-API regression first proves that Kubernetes answered, then stops
only the owned server process and proves that its next request times out.
Without the diagnostics deadline, the child spent 104.89 s in startup failure
and the regression failed because diagnostics did not report its context
deadline (`stalled-causal-before`). The parent now also requires completion
within one minute after stalling the API. With the ten-second diagnostics
deadline, the same real-container case passed with `-race` in 16.913 s
(`stalled-threads`), removed its resources and retained the unrelated network.
Earlier failed probe preparations are not counted as defect reproductions.

### Complete CNPG after the follow-up repairs

The documented whole-package command passed again in **460.186 s** (462 s
including startup), with **65 scenario leaves**, the no-op child entry and
all **17 top-level tests** passing. There were no failures or skips. This run
includes the same-timeline gap cases on both majors and bounded startup
diagnostics. The subsequent one-minute completion bound on the stalled-API
probe is also exercised by the current CI shard check.

Host memory rose from 12101 to 29372 MiB, **16.87 GiB** above the baseline.
The sampler recorded peaks of **21 k3s** and **31 running CNPG clusters**.
Busy CPU reached 94%, load average 46.42; the Docker filesystem rose from
168559 to 168723 MiB, and `/tmp` from 2566 to 3430 MiB. Backend and repository
lint overlapped startup; no other test suite overlapped. Logs and samples are
`/tmp/cnpg-measure/followup-cnpg.jsonl` and `followup-cnpg.tsv`.

The full script suite passed 219 tests in 31.232 s before the latest weight
refresh. The group-selection suite was repeated after that refresh. Repository
lint passed its other areas; backend lint passed separately with zero issues
after formatting the new long call. Dependency verification, strict OpenSpec
validation and `git diff --check` passed again.

The complete Patroni package also passed after extracting the shared WAL
recycling wait: **175.123 s** (176 s command wall time), all **55 pass events**,
no failure or skip. Host memory rose from 12053 to 22633 MiB, **10.33 GiB**
above baseline. It ran separately from CNPG. Evidence is
`/tmp/cnpg-measure/followup-patroni.jsonl` and `followup-patroni.tsv`.

### Preceding CI shard assignment

The refreshed assignment was checked using the same CI-style command, once
for `cnpg-2-of-5` and once for `cnpg-3-of-5`, started concurrently:

```sh
pattern=$(mise run --quiet backend:test-group-tests cnpg-2-of-5)
GOMAXPROCS=8 TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -race -p=8 \
  -count=1 -failfast -timeout 15m -json -run "$pattern" \
  ./internal/modules/backups/e2e/postgresqlphysical/cnpg
```

| Current shard | Selected families | Package duration | Command wall time | Result |
| --- | --- | ---: | ---: | --- |
| 2, largest weight | fixture cleanup, PostgreSQL 17 independent recovery, PostgreSQL 18 interrupted backups | 368.690 s | 370 s | passed, no race or skip |
| 3 | PostgreSQL 17 slots and access, fixture lifecycle | 281.203 s | 283 s | passed, no race or skip |

This includes the one-minute bound on stalled-API cleanup and the new
same-timeline WAL-gap case. Both meet the ten-minute limit. Host memory rose
from 12310 to at most 22594 MiB across the two checks, **10.04 GiB** for both
together, within one selected runner's 16-GB budget. Shard 2 separately peaked
at 11 k3s and 8 running CNPG clusters; shard 3 at 5 k3s and 8 clusters.
Repository lint overlapped startup. `GOMAXPROCS=8` sets the test concurrency;
it does not emulate a native runner's CPU quota. No native ARM execution or
native CI timing is claimed.

Evidence is `/tmp/cnpg-measure/followup-ci-{2,3}.jsonl`, the matching TSV files
and `followup-shards.json`. After both finished, all fixture resources and
every follow-up run label returned **zero containers, zero networks and zero
volumes**, including the failed diagnostic preparations and the intentional
before-fix regression. Exact results are `cleanup-followup.json`. The shared
PostgreSQL server stayed running.

### Routine command before the upstream repair

`mise run test` passed again after all follow-up repairs in **397 s**: slow
backend packages 371 s, other backend packages 82 s, frontend 79 s. All 194
frontend files and 2870 tests passed. Host memory rose from 12073 to 27308 MiB,
**14.88 GiB** above baseline, within the entire local run's 32-GB
budget. Evidence is `/tmp/cnpg-measure/mise-test-followup.log` and its TSV.
`mise run lint` passed as a complete command (`lint-current.log`), and the
refreshed group-selection checks passed all 16 tests in 2.130 s.

After moving the unchanged Kubernetes-version query into its own file and
renaming it to `GetK3sVersion`, gopls confirmed the single lifecycle caller.
That real-k3s lifecycle scenario passed with `-race` in **36.970 s** (40 s
command wall time). Backend formatting and lint passed with zero issues,
strict OpenSpec validation and `git diff --check` passed again. The other
follow-up edits only separate logical steps with blank lines; scenario names,
behavior, weights and shard assignments are unchanged. Evidence is
`/tmp/cnpg-measure/version-structure.jsonl` and `lint-version-structure.log`.

### Upstream SSL cancellation repair

The real-TCP regression completes client TLS startup, reads the proxy's
SSLRequest at the upstream peer and deliberately withholds the SSL response.
Before the fix, cancellation did not complete forwarding and the five-second
probe failed (`upstream-before.log`). Closing the peer in failure cleanup
still joined the probe's tracked goroutine. After registering cancellation
against the immutable raw upstream connection before negotiation, all proxy
and Docker exec cases passed with `-race` in 2.203 s. The final test form also
passed in 17.170 s under concurrent suite load (`upstream-final.log`). It
requires EOF on the upstream socket after cancellation.

Gopls renamed the manager constructors to `NewCNPGManager` and
`NewPatroniManager`, and the WAL filename getter to `getCurrentSegment`,
including every caller. These names do not change scenario selection.
Backend formatting and lint passed with zero issues (`lint-upstream-fixed.log`).
The requested independent guards now have blank lines between them.

The routing-outage families were repeated with `-race` on PostgreSQL 17 and
18 for both managers:

```sh
TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -race -count=1 -timeout 15m \
  -run '^TestCNPGRoutingOutagesPostgres1[78]$' \
  ./internal/modules/backups/e2e/postgresqlphysical/cnpg
TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -race -count=1 -timeout 15m \
  -run '^TestPatroniRoutingOutagesPostgres1[78]$' \
  ./internal/modules/backups/e2e/postgresqlphysical/patroni
```

CNPG passed in **293.640 s** (297 s command wall time), all 6 test pass
events; Patroni passed in **182.089 s** (185 s command wall time), all
6 pass events. Neither reported a race, failure or skip. They ran
concurrently with the final routine command below; these are focused
regressions, not replacement isolated performance measurements. Evidence is
`/tmp/cnpg-measure/upstream-{cnpg,patroni}.jsonl` and the matching TSV files.

The final `mise run test` passed in **457 s** (458 s wrapper wall time):
slow backend packages 443 s, other backend packages 84 s, frontend 84 s.
All 194 frontend files and 2870 tests passed. It overlapped the focused CNPG
and Patroni regressions above. Across all three samplers, host memory rose
from 12104 to 31380 MiB, **18.82 GiB** above baseline,
within the shared 32-GB local test budget for the combined checks. Evidence is
`/tmp/cnpg-measure/mise-test-upstream.log` and the three matching TSV files.
All owned fixture and follow-up run-label queries again returned zero
containers, networks and volumes (`cleanup-followup.json`).

The final structure pass renamed `ExecNode` to `ExecPostgresNodeCommand`
through gopls, removed redundant comments and separated independent guards
and assertions with blank lines. The real-node lifecycle cases on PostgreSQL
17 and 18 passed with `-race` in **137.623 s** (141 s command wall time),
including per-node SQL through the renamed method. Proxy and Docker exec
checks passed again with `-race` in **2.316 s**. Backend formatting and lint
passed with zero issues (`lint-structure-complete.log`). These edits change
names, comments and whitespace only; the latest full routine run and focused
upstream regressions above cover the final behavior. Evidence is
`/tmp/cnpg-measure/node-command-structure.jsonl` and
`proxy-node-command-structure.log`. Cleanup queries remain empty.


## Final coverage and resource-ownership checks

Fixture switchover, failover and slot controls now have separate top-level
tests and files. The package has 19 top-level tests and retains all 65
scenario leaves. The declared backup role actually attempts
`pg_read_file` and `pg_read_binary_file` on both `postgresql.conf` and
`pg_hba.conf`; every attempt must return SQLSTATE `42501`, before and after
promotion. Attribute and table-read restrictions remain checked separately.

The TLS transfer-gate test consumes the first backup data message, requires
a read deadline while the gate is held, then requires further backup data
after release. It passed with `-race` in 3.167 s (`gate-observed.log`).
Removing only the gate's blocking wait made that same test fail in 1.166 s
because the held transfer delivered data (`gate-without-wait.log`). The
blocking implementation was restored before subsequent runs.

The overlap-cleanup case waits until both independently labelled child runs
hold live, ready fixtures before releasing either. After one exits, its
resources must be absent while the other child's resources still exist.
The second child then exits successfully and its resources must also be
absent. Each live container, network and volume is inspected for the child's
requested `databasus.test-run` ID. Children with their own IDs retain the
enclosing ID as `databasus.test-root-run`. The shell runner exports that root
ID and removes the union of resources matching either exact label with its
own run ID, deduplicating IDs. It never selects another run's resources.
This also retains ownership of existing Patroni children that replace their
own run IDs. The script regression failed before root-label cleanup existed
(`root-run-before.log`); the final 51 shell tests passed in 15.974 s
(`root-run-export.log`).

The entire final CNPG cleanup family passed with `-race` in **161.104 s**
(164 s command wall time), including distinct child IDs, overlapping live
fixtures, cancellation, partial failure, killed-parent handling and the
stalled-API case. Evidence is `root-run-cleanup.jsonl` and its TSV.

### Latest complete CNPG measurement

The documented complete-package command passed in **407.613 s** (409 s wall)
with **84 test pass events** covering 65 scenario leaves and
19 top-level tests, including the no-op child entry. There were no failures or skips. Go used its default
parallel limit of 32. No other test suite ran concurrently; routine lint
and script checks overlapped startup.

Global fixture sampling observed **18 k3s** and **32 running CNPG clusters**.
Host memory rose from **12209 to 30184 MiB**, an increase of **17.55 GiB**
within the total 32-GB test budget. Busy CPU peaked at 93%, load average at
57.32, Docker filesystem usage at 168687 MiB from 168561, and `/tmp` at
3349 MiB from 2596. Evidence is `/tmp/cnpg-measure/coverage-cnpg.jsonl` and
`coverage-cnpg.tsv`. This full run includes the actual denied file reads,
observable transfer gate and both-ready overlap barrier. The final child/root
ownership metadata was added afterward and verified by the complete cleanup
family and current CI shards below; no new full-package timing is claimed
for that metadata-only repair.

Every current weight is the rounded-up sum of this run's scenario durations;
the no-op child entry receives one. The exact current selections are retained
in `/tmp/cnpg-measure/coverage-shards.json`:

| Shard | Weight, s | Selected top-level families |
| --- | ---: | --- |
| 1 | 1985 | cleanup, fixture failover, PostgreSQL 17 interrupted backups |
| 2 | 1955 | PostgreSQL 17 independent recovery and routing outages, fixture slots, child entry |
| 3 | 1976 | PostgreSQL 17 slots, PostgreSQL 18 routing outages, lifecycle, pool |
| 4 | 1972 | PostgreSQL 17 access, PostgreSQL 18 slots and recovery, fixture switchover |
| 5 | 2001 | PostgreSQL 18 access and independent recovery, PostgreSQL 17 recovery, PostgreSQL 18 interrupted backups |

### Current CI candidate checks

The selected five-shard configuration remains **8 vCPU / 16 GB per runner**,
with Go's default test parallelism. Shards 1 and 5 (the largest current
weight) passed with `GOMAXPROCS=8`, `-race -p=8 -failfast -count=1 -timeout 15m`.
The exact commands and frozen regular expressions are retained in
`/tmp/cnpg-measure/run-coverage-ci-{1,5}.sh`.

| Shard | Package duration | Command wall time | Test pass events |
| --- | ---: | ---: | ---: |
| 1 | 267.890 s | 269 s | 20 |
| 5 | 306.329 s | 307 s | 18 |

Neither reported a race, failure or skip. The checks ran concurrently and
increased aggregate host memory from **12156 to 23369 MiB**, **10.95 GiB**
for both together, below one selected runner's 16-GB budget. Busy CPU reached
81%, load average 21.84, Docker filesystem usage 168639 MiB from 168561, and
`/tmp` 3332 MiB from 2616. The per-run sampler now includes the union of own
and root ownership labels, so it counts children with explicit different IDs.
Shard 1 peaked at 7 k3s and 7 CNPG clusters; shard 5 at 2 k3s and 8 clusters.
Evidence is `coverage-ci-{1,5}.jsonl` and their TSV files. These are local
candidate measurements, not native CI or arm64 execution results.

### Latest complete Patroni regression

The complete separate Patroni command from task 8.3 passed in **224.884 s**
(226 s wall), all **55 test pass events**, without failures or skips.
This run includes the shared actual server-file denial assertions before
and after promotion. Host memory rose from 12281 to 23349 MiB, **10.81 GiB**
above baseline. No other test suite overlapped it. Evidence is
`/tmp/cnpg-measure/coverage-patroni.jsonl` and `coverage-patroni.tsv`.


### FTP authentication repair discovered by the final routine run

The `mise-test-coverage` routine command failed after **404 s** because the
successful unsaved FTP-settings case returned `IsConnected=false` when its
request exhausted the ten-second connection deadline. All other packages
and all 2870 frontend tests passed. The original assertion printed a string
pointer rather than the error text, so the exact underlying error from that
run cannot be recovered. The assertion now compares the complete response
and reports its string value on failure. Five isolated repeats of that
unchanged successful FTP flow passed in 27.467 s (`ftp-diagnostic.log`).

A controlled experiment used two real containers from the same FTP image
and the same credentials. Each generated its hash before receiving the
same 0.2-CPU limit. With default calibration, login timed out at **10.012 s**;
with `pure-pw -C 32`, login succeeded in **4.705 s**. An earlier comparison
at 0.25 CPU took 9.182 s and 2.605 s. A stricter 0.1-CPU experiment timed out
for both configurations; no success is claimed for that budget. These
experiments establish CPU-sensitive hash verification as a cause of this
kind of deadline failure, without claiming the missing original error text.
Evidence is `/tmp/cnpg-measure/ftp-hashing-{fifth,quarter}-probe.json` and
`ftp-hashing-probe.json`. Every probe removes its own container in `finally`.

The fixture now supplies `-C 32`, accounting for concurrent test-server
logins when calibrating the real password hash. It keeps successful and
wrong-password authentication checks and the existing application deadlines.
No runtime FTP behavior or runner parallel limit changed.


The diagnostic routine repeat passed in **417 s** before the FTP calibration
repair was exercised by that run's storage containers: slow backend 394 s,
other backend 68 s, frontend 85 s. It is an intermittent pre-repair success,
not evidence that the deadline defect was absent. The full storage API
package then passed with the repaired fixture and `-race -count=1 -timeout 5m`
in **24.353 s** (`ftp-storages-fixed.log`). Repository formatting and all
linters passed again (`format-ftp.log`, `lint-ftp.log`). The latest full
Python script suite passed **220 tests in 54.127 s**
(`python-coverage-final.log`). Dependency verification, strict OpenSpec
validation and diff whitespace checks also passed.


### WAL publication cleanup order repair

The post-FTP-repair routine run completed after **426 s** but failed the
late WAL publication case during cleanup (`mise-test-final-ftp.log`). The
FTP storage module passed in 14.319 s, as did the other packages and all
2870 frontend tests. The failing WAL test body passed, but its application
backup-run drain exhausted the existing 30-second deadline. The following
notification drain shared that expired context and failed too.

The applications had registered shutdown before the MinIO storages were
created. LIFO cleanup could thus remove a storage while a replacement FULL
was still writing to it. Both late-publication cases now register shutdown
after their target storage is created, so applications stop and drain while
their dependencies remain live. The existing shutdown method was renamed
from private `stop` to public `Stop` through gopls for this explicit ordering;
its implementation and deadlines are unchanged. Normal cleanup invokes it
again, exercising repeated shutdown too.

Both cases passed with `-race -count=1 -timeout 5m -run
'^TestLateWalStoragePublication$'` in **41.962 s**
(`/tmp/cnpg-measure/wal-cleanup-fixed.log`). Backend formatting passed. The
linter requested the exported method be placed before private methods;
that ordering was corrected before the repeated lint and routine checks.


## Final verification of the completed implementation

After both routine-run repairs, **`mise run test` passed in 412 s**:
slow backend packages 388 s, other backend packages 66 s, frontend 73 s.
All **194 frontend files and 2870 tests** passed. The fixed FTP and WAL
publication packages passed in this complete run, along with the application
lifecycle checks. No other test suite ran concurrently; backend lint overlapped
startup. Host memory rose from **13901 to 26143 MiB**, an observed increase
of **11.96 GiB**, within the total local 32-GB test budget. Busy CPU reached
100%, load average 96.74, Docker filesystem usage 168698 MiB from 168563, and
`/tmp` 3714 MiB from 2593. Evidence is
`/tmp/cnpg-measure/mise-test-final-cleanup.log` and its TSV. This result
supersedes the failed 404-s and 426-s attempts and the diagnostic pre-repair
success; those remain documented as defect evidence.

Backend formatting and lint passed after the shutdown-method ordering fix
(`format-wal-cleanup-order.log`, `lint-wal-cleanup-order.log`). Full repository
lint had passed after the FTP repair (`lint-ftp.log`); subsequent code edits
were confined to the backend. Strict OpenSpec validation and diff whitespace
checks passed again. The latest script suite has 220 passing tests; dependency
verification passed with no dependency changes afterward.

The final cleanup inspection queries all fixture-labelled resources and both
own/root labels for the recorded CNPG and Patroni follow-up run IDs. Every
query found **zero containers, networks and volumes**
(`/tmp/cnpg-measure/cleanup-followup.json`). Exact labels for both diagnostic
FTP probes were empty too. The shared test PostgreSQL server was preserved.
`mise run test-full` and native arm64 scenarios were not run.

## Final abrupt-loss and structure review repairs

The earlier primary-loss runs established promotion and recovery, but
force-deleting a pod alone did not guarantee an unclean shutdown. Kubelet
can still grant a minimum termination window. The fixture now obtains the
serving pod's `postgres` container ID, requires successful
`ctr -n k8s.io tasks kill --all --signal SIGKILL <containerID>`, and records
that exact container ID before deleting the pod. This kills the instance
manager and PostgreSQL processes together. Replay, different serving node,
newer timeline, unchanged system identifier and old pod UID checks remain.
Design decision 8 and the tasks describe the same operation.

The focused race-enabled command was:

```sh
TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -race -count=1 -timeout 15m -json \
  -run '^(TestCNPGFixtureFailover|TestCNPGRecoveryPostgres1[78])$/(routes_to_the_other_node|recovers_after_primary_loss)' \
  ./internal/modules/backups/e2e/postgresqlphysical/cnpg
```

It passed in **174.372 s** (**178 s** command wall time): all four scenario
leaves, covering fixture failover and booted recovery on both PostgreSQL
majors, and their three top-level tests. No race, failure or skip occurred.
`/tmp/cnpg-measure/sigkill-cnpg.jsonl` records four successful SIGKILL
operations with their actual container IDs and all subsequent scenario
passes; its TSV contains resource samples. Both primary-loss callers are
covered by this command. Earlier full-package timings and CI measurements
remain preceding measurements; no full-package timing with the final
SIGKILL control is claimed.

The transfer probe uses the domain name `copyData`, certificate-refusal
assertions separate their steps, and the restored-log reader is named
`readRestoredServerLogs`. The TCP-authentication rationale now sits above
`waitForRecoveryToFinish`. These name, comment and whitespace changes do not
change runtime behavior. All proxy cases passed again with `-race` in
**3.189 s** (`proxy-sigkill-structure.log`); the focused recovery cases above
compile and exercise the shared restored-server path.

Backend formatting and lint passed (`format-sigkill-final.log`,
`lint-sigkill-final.log`), as did strict OpenSpec validation and diff
whitespace checks. The successful 412-s routine run remains the full
non-end-to-end verification; the final behavioral change is confined to the
CNPG primary-loss fixture and is covered by the focused command above.
Final own/root label inspection includes
`cnpg-measure-sigkill-cnpg-1794571961` and again found no fixture containers,
networks or volumes (`cleanup-followup.json`).

## Final credential-safety regression and verification

Review exposed an exception to preserving the driver's connection diagnostic:
`pgconn.ParseConfigError` embeds conninfo. With an invalid root CA and the
synthetic password `prefix' PRIVATE_SUFFIX`, pinned pgx's masking left
`PRIVATE_SUFFIX` visible. The new public-source regression failed on the
unfixed implementation for that reason; a password with both a backslash
and apostrophe also leaked its suffix. The backslash-only case still exposed
the connection string. Evidence is `escaped-password-before.log`.

The classifier now recognizes wrapped configuration parse errors and returns
`CONNECTION_FAILED` with `invalid PostgreSQL connection configuration`,
without formatting the original error or retaining its error chain. Actual
connection failures retain their certificate and host diagnostics. All three
regressions passed with `-race` in 1.104 s after the fix; the final form using
the linter-required `errors.AsType` passed in **1.215 s**
(`escaped-password-final.log`). The command was:

```sh
go -C backend test -race -count=1 -timeout 2m \
  -run '^TestTestReplicationConnectionConfiguration$' \
  ./internal/engines/postgresql/physical
```

`NodeCommand` and `ExecPostgresNodeCommand` moved to
`cnpg_node_command.go`; query types and SQL methods remain in
`cnpg_node_sql.go`. The following focused command verifies certificate
refusal diagnostics and both independent target-recovery callers after
that file split:

```sh
TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -race -count=1 -timeout 15m -json \
  -run '^(TestCNPGAccessPostgres1[78]|TestCNPGIndependentRecoveryPostgres1[78])$/(refuses|recovers_independently_with_target)' \
  ./internal/modules/backups/e2e/postgresqlphysical/cnpg
```

It passed in **280.133 s** (**286 s** wall), all six scenario leaves and
four top-level tests, with no failure, skip or race. Both certificate reasons
remain visible on PostgreSQL 17 and 18. Both independent recoveries booted
the requested state while their source was hibernated. Evidence is
`/tmp/cnpg-measure/credentials-cnpg.jsonl` and its TSV.

The final **`mise run test` passed in 344 s**, superseding the 412-s routine
result: slow backend 331 s, other backend 90 s, frontend 78 s. All 194 frontend
files and 2870 tests passed. It overlapped the focused CNPG command above.
Across both samplers, host memory rose from **12743 to 27429 MiB**, an
increase of **14.34 GiB** for the combined checks, within the entire 32-GB
local test budget. The CNPG sample observed two k3s and six CNPG clusters.
CPU reached 100% and load average 103.81. Docker filesystem usage reached
168709 MiB from 168564, `/tmp` 3682 MiB from 2614 in that sample. Evidence is
`mise-test-credentials.log` and both TSV files.

Backend formatting and lint passed again (`format-credentials-final.log`,
`lint-credentials-final.log`), along with strict OpenSpec validation and diff
whitespace checks. No other module's code changed after its previous lint
check. Final fixture and own/root label inspection includes
`cnpg-measure-credentials-cnpg-1794572800`; every container, network and
volume query remained empty (`cleanup-followup.json`).

## Final recovery-scenario organization

Independent recovery remains in `cnpg_independent.go`; external-configuration
preservation and copied-authentication enforcement now live in
`cnpg_external_configuration.go` and `cnpg_copied_authentication.go`. Their
shared serving-configuration observation lives in
`cnpg_serving_configuration.go`. Preparation, execution and assertions have
separate logical steps, including the scenario driver and role-probe setup.
Redundant narration was removed from the moved code. Test names, runtime
behavior, all 19 CNPG weights and current shard selections are unchanged.

The moved scenarios were repeated on both PostgreSQL majors:

```sh
TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -race -count=1 -timeout 15m -json \
  -run '^TestCNPGIndependentRecoveryPostgres1[78]$/(boots_with_external|enforces_the_authentication)' \
  ./internal/modules/backups/e2e/postgresqlphysical/cnpg
```

All four scenario leaves and both top-level tests passed in **154.565 s**
(**158 s** wall), without a failure, skip or race. The source retained its
TLS and access configuration, and the recovered server enforced its copied
`pg_hba.conf` with accepted and rejected connections. Evidence is
`/tmp/cnpg-measure/recovery-units-cnpg.jsonl` and its TSV.

Backend formatting and lint passed (`format-recovery-units-final.log`,
`lint-recovery-units-final.log`), along with strict OpenSpec validation and
`git diff --check`. These file, comment and whitespace changes preserve the
behavior covered by the latest successful 344-s routine run. Final
fixture and own/root label queries include
`cnpg-measure-recovery-units-cnpg-1794573712` and again found zero containers,
networks and volumes (`cleanup-followup.json`).

## Final routing-outage slot precondition

Review found that `NewCNPGManager` discarded `ShouldKeepSlotOnEveryNode`.
Enabling synchronization alone had not established the required candidate
slot before promotion; later recreation could conceal its absence.
The requested mode now wraps the switch with an observed precondition.
After completed management cycles, it reads both slots, requires the
candidate's nonempty `restart_lsn` to be no later than the primary slot's
current required position, and observes a current segment absent from the
archive. It records the positions and segment before switching and requires
that candidate to serve afterward. The check never creates Databasus's slot.

WAL reception remains active: blocking every replication connection would
also terminate the held `BASE_BACKUP` before the intended promotion. The
separate synchronized-slot case continues to block reception to establish
its controlled unreceived interval. Design decision 9 and task 4.2 now
state this observed routing-outage precondition explicitly.

The focused command was:

```sh
TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -race -count=1 -timeout 15m -json \
  -run '^(TestCNPGRoutingOutagesPostgres1[78]|TestCNPGSlotsPostgres1[78])$/(preserves|keeps_the_synchronized)' \
  ./internal/modules/backups/e2e/postgresqlphysical/cnpg
```

All six scenario leaves and four top-level tests passed in **286.509 s**
(**290 s** wall), without a failure, skip or race. The log contains four
pre-promotion slot observations, one for each FULL/INCREMENTAL outage on
both PostgreSQL majors. Candidate positions were `0/9FE0000` or `0/AFE0000`,
equal to the observed primary position, covering unarchived segments
`00000001000000000000000A` or `00000001000000000000000B` respectively.
The remaining checks established the original failure, refused recheck,
UTC-scheduled replacement and booted recovery. Evidence is
`/tmp/cnpg-measure/routing-retention-cnpg.jsonl` and its TSV.

Gopls renamed `serverCertificateFiles` to `getServerCertificateFiles` and
`lastLine` to `getLastLine`, including all callers. All proxy cases passed
again with `-race` in **3.188 s** (`proxy-certificate-getter.log`). Backend
formatting and lint passed (`format-routing-retention-final.log`,
`lint-routing-retention-final.log`), along with strict OpenSpec validation
and `git diff --check`. Test names, weights and shard selections remain
unchanged. The 344-s routine run remains the latest complete non-end-to-end
check; subsequent behavioral changes affect the CNPG fixture only and are
covered by the focused commands. Final own/root label queries include
`cnpg-measure-routing-retention-cnpg-1794575019` and found zero fixture
containers, networks and volumes (`cleanup-followup.json`).
