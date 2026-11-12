## Context

See proposal.md for the motivation. This section records the current schedule and the isolation gaps that block a concurrent one.

### Current schedule

`scripts/test.sh` runs three suites at once: the slow backend packages with `-p=8` (`scripts/test.sh:125`), the other backend packages with `-p=16` (`scripts/test.sh:130`) and the frontend tests (`scripts/test.sh:134`). The full run adds the backup and agent end-to-end packages to the slow group. Only after all three pass does it start the extra suites, one after another (`scripts/test.sh:150-163`). The comment at `scripts/test.sh:156` gives the reason: "Image and browser suites run sequentially because they disturb each other's timing under load."

Each suite was measured alone on the development host (AMD Ryzen 9 7950X, 32 logical processors, 62 GB of memory) before the change, with the shared test PostgreSQL server running, the test images present and a warm build cache. Host memory was sampled every two seconds; "peak" is the highest used memory during the suite, and "added" is that peak minus used memory at the suite's start, which was 12-18 GB of desktop, editor and long-lived services.

| Stage | Command | Duration | Peak / added memory |
|---|---|---|---|
| Backend slow group with end-to-end packages, `-p=8` | `go -C backend test -p=8 -count=1 -timeout 15m <slow packages>` | 618 s | 34.0 / 20.0 GB |
| Backend other group, `-p=16` | `go -C backend test -p=16 -count=1 -timeout 15m <other packages>` | 27 s | 16.8 / 2.7 GB |
| Frontend tests | `pnpm --dir frontend test` | 54 s | 23.8 / 9.5 GB |
| Script tests | `python3 -m unittest discover -s scripts -p '*_test.py'` | 119 s | 13.0 / 1.0 GB |
| Review hook tests | `python3 -m unittest discover -s .agent-hooks -p '*_test.py'` | 153 s | 13.6 / 3.0 GB |
| Browser suite unit tests | `mise run e2e:unit` | 3 s | no change |
| Browser UI, two workers | `pnpm --dir frontend test:browser --project=ui` | 220 s | 19.4 / 4.5 GB |
| Upgrade test, image build included | `mise run backend:test-upgrade` | 228 s, of which 176 s the Go test | 18.0 / 1.7 GB |
| Browser end-to-end scenarios, image build included | `mise run e2e:browser` | 74 s | 17.7 / 1.4 GB |
| Lifecycle `failures` | `mise run e2e:lifecycle failures` | 219 s | 18.1 / 1.5 GB |
| Lifecycle `interruption` | `mise run e2e:lifecycle interruption` | 196 s | 18.9 / 1.7 GB |
| Lifecycle `isolation` | `mise run e2e:lifecycle isolation` | 473 s | 19.0 / 1.3 GB |
| Lifecycle interruption checks | `uv run --locked --project e2e/browser pytest e2e/browser/lifecycle_interruption` | 39 s | 18.2 / 0.8 GB |

The sum of the old schedule is the slow group plus every extra stage: 618 s + 1725 s, about 39 minutes, which matches the observed full run.

The slow group's critical path is `internal/agent/e2e` at 617 s. The next longest packages are `internal/modules/backups/e2e/postgresqlphysical/patroni` at 502 s and `internal/modules/storages/internal/providers` at 367 s. In this measurement the Patroni package failed: under the group's load the Docker daemon did not answer container stop, log and network removal requests within the fixture's one-minute cleanup deadline (`backend/internal/lib/testcontainers/patroni.go:82`). Docker keeps its data in an ext4 image on a loop device without direct I/O on this host.

The script tests failed in the measurement only because the sampler started them in the background of a non-interactive shell, which ignores SIGINT; the two interruption tests pass when run in the foreground.

The current critical path is the sum of the slow group and every extra stage. The target critical path is the longer of two chains: the slow group, and the probes that must run alone followed by the browser runs that may not overlap them.

```
current (about 40 min)
0                 ~12                                                          ~40
|== backend-slow ==|
|== backend-other =|
|== frontend ==|   |
                   |scripts|hooks|unit|== UI ==|== upgrade ==|== e2e ==|====== lifecycle ======|intr|

target (about 10 min, limit 20, measured about 14)
0                                                       ~11     ~12   ~14
|== backend-slow =======================================|
|== backend-other ==|
|== frontend ==|
|scripts|hooks|unit|
|== UI (workers from the runner) ==|
|== build ==|== upgrade ==|
|== crashed-run probes =========================================|intr|
                                                                     |== e2e:browser, lifecycle in 7 parts ==|
```

### Isolation gaps

Concurrency exposes these gaps. Each one either breaks a suite when another runs beside it, or lets one run test the wrong thing.

- **Fixed image tag.** The upgrade command builds `databasus-candidate:local` (`mise-tasks/backend/test-upgrade:16`). A direct upgrade run in another worktree retags it while this run's suites still start containers from it.
- **Default address pools.** The browser compose project uses Compose's default network (`e2e/browser/compose.yaml` has no `networks` section), and the upgrade test creates its network without a subnet (`backend/e2e/upgrade/docker_test.go:57-58`). Both take networks from Docker's default pools. Verification agents also allocate from those pools and cannot choose a range, which is why backend fixtures avoid them (`backend/internal/lib/testcontainers/network_subnet.go:16-34`). Agents run in the agent end-to-end package and inside the upgrade test, which waits for the released and current agents to remove their restore networks (`backend/e2e/upgrade/agent_container_test.go:238-292`). On the development host, about 10 of the roughly 31 networks the default pools provide are already taken by long-lived networks. Three lifecycle processes, the browser scenarios, the upgrade test's own network and the restore networks of both agent suites together could exhaust them. Moving the networks the suites create themselves into the test range leaves the default pools to the agents alone.
- **Machine-wide sweep.** Before starting, every browser run removes the projects of runs whose supervisor is gone, including projects with no record at all (`e2e/browser/leftover_sweep.py:33-87`). Crashed-run probes kill a run, or the process recorded as its supervisor, and then assert that their own later run removed it, for example `assert_sweep_came_before` at `e2e/browser/lifecycle/test_crashed_runs.py:81` and `e2e/browser/lifecycle/test_built_image.py:88`. The probe at `test_crashed_runs.py:24` expects a refused removal to stop its later run. `test_probe_cleanup.py:106` writes a supervisor record for a fake supervisor and kills it. Any concurrent browser run would remove these crashed runs first, and the probes would fail. The lifecycle interruption checks kill the lifecycle command itself and expect the next lifecycle run to remove what it left, so they have the same dependency.
- **Load-sensitive timing.** The comment at `scripts/test.sh:156` admits that some suites fail under load. Which tests fail is not yet known; the browser UI limits of 30 seconds per test and 5 seconds per assertion (`frontend/playwright.config.ts:11-12`) are the first candidates.

## Goals / Non-Goals

**Goals:**

- Shorten the full run's critical path to the target in the specs, without removing a check.
- Make every concurrent suite own its image, networks and cleanup, so no suite relies on running alone.
- Leave the time each suite takes visible after every full run.

**Non-Goals:**

- A general scheduler or a dependency language for suites. The run has three chains and one barrier; plain shell job control covers that.
- Changing CI jobs, shards or groups.
- Rewriting a slow package. If measurement shows one package alone exceeds the budget, the change reports it with its time and leaves the rewrite to a separate change unless the fix is small.

## Decisions

### Keep the runner in Bash and express the schedule as chains

`scripts/test.sh` keeps `start_suite`, which already gives each suite its own session, prefixes its output and records its process ID. The full run adds two chains that start together with the Go and frontend suites:

1. build the image, then run the upgrade test;
2. run the lifecycle command with `--only-crashed-run-probes`, then the lifecycle interruption checks, even when the crashed-run probes failed, and after both the browser end-to-end scenarios and the lifecycle probes with `--without-crashed-run-probes`, split into seven parts that run at the same time.

The exclusive stage goes first because it runs one probe after another, while the browser runs after it share the machine. The rounds below measured both orders.

Script tests, hook tests and `e2e:unit` start at time zero beside everything else. Each suite records its start and end time; the summary prints them, the build's time or `skipped` when the developer named an image, the total and every failed or not-started suite. The image build is the only step whose failure keeps other suites from starting: the upgrade test and the browser end-to-end scenarios are then reported as not started. Every other failure only marks its own suite as failed.

On interruption the existing `cleanup` trap already terminates each suite's session and removes containers, networks and volumes labelled with the run ID (`scripts/test.sh:22-47`). Browser runs and lifecycle commands receive the termination and remove their own compose projects, as the browser spec requires. The trap is extended to remove the run's image.

Alternatives:

- Rewrite the runner in Python with a dependency graph. It would read better, but it replaces a script with existing script tests (`scripts/test_sh_test.py`) and gains nothing for three chains.
- Use `make -j` or mise task dependencies. Neither prints per-suite timing in our format, and neither handles our session-based interruption, which the spec on interrupting the backend suite relies on.

### Build the image once, at the start, under a tag only the full run uses

The full run builds the image from the checkout at time zero under the fixed tag `databasus-full-test:local` and passes it as `DATABASUS_IMAGE` to the upgrade test and the browser end-to-end scenarios. The `cleanup` trap removes it at exit. Full runs hold the machine-wide lock, so no other full run can use the tag at the same time. The direct upgrade command gets its own run-unique tag, so it cannot touch this one either. When a killed full run leaves the image behind, the next full run's build replaces the tag, so at most one stale image waits until then.

The lifecycle command keeps building its own probes image. Five of its probes test a run that builds an image itself and are skipped when an image is named: the three in `e2e/browser/lifecycle/test_built_image.py` and the two in `e2e/browser/lifecycle/test_fresh_checkout.py`, both skipped at module level on `DATABASUS_IMAGE`. BuildKit shares the identical layers between concurrent builds, so the extra builds mostly cost cache lookups.

The direct `mise run backend:test-upgrade` tags its build `databasus-upgrade-test:<run ID>`, taking the run ID from `DATABASUS_TEST_RUN_ID` when a runner sets it and generating one otherwise. It prints the tag and removes the image on exit, failure and interruption through a shell trap. It keeps building only when `DATABASUS_IMAGE` is empty, and it never removes a named image.

Alternatives:

- A run-unique tag with a label and a startup sweep for images of killed full runs. The lock already rules out collisions between full runs, so the label, the sweep and their tests buy nothing.
- Pass the run's image to the lifecycle command too. It saves one build but skips those five probes, which loses coverage the browser spec requires locally.

### Recognize crashed-run probes by the teardown fixture they must use

A crashed-run probe is a lifecycle probe that kills a run, or the process recorded as a run's supervisor, or prevents a run's cleanup. The browser spec already requires such probes to register unconditional teardown, which `register_run_cleanup` (`e2e/browser/lifecycle/run_cleanup.py:16`) provides. That fixture becomes the only way to register it. The fixture accepts a run ID and an owned image as well as a run object, so `test_probe_cleanup.py:106`, which now uses `request.addfinalizer` with `remove_run_resources`, moves to it.

A collection hook in the lifecycle `conftest.py` adds the `crashed_run` marker to every probe that requests the fixture. The lifecycle command gains two mutually exclusive options, `--only-crashed-run-probes` and `--without-crashed-run-probes`, that combine with the existing group argument. The default with neither option stays as it is, so CI groups and the complete local command keep their behavior.

A unit test under `e2e/browser/unit` keeps the rule from drifting. It reads the lifecycle probe sources and fails when a probe sends `SIGKILL`, writes a supervisor record, or calls `remove_run_resources` or `request.addfinalizer` without requesting `register_run_cleanup`.

The fixture also marks the interruption probes in `test_interruption.py` and the self-cleaning probe at `test_built_image.py:35`, which register teardown only as a safety net. Running them in the exclusive stage costs a few probes of concurrency and keeps the rule automatic. The lifecycle interruption checks live in their own directory and command, so the full run places them in the exclusive stage directly.

The term "crashed-run probe" joins the Code table of `docs/glossary.md`, beside "Lifecycle probe", and names the marker, the options and the documentation.

Alternatives:

- A hand-placed marker on each probe. A new probe without the marker would pass alone and fail intermittently in a full run.
- Isolate the sweep with a namespace per lifecycle process, so probes see only their own crashed runs. It needs a new recovery path for a killed lifecycle process's namespace, which the browser spec does not have today, and it changes production supervisor behavior for a test scheduling problem.
- Run the three lifecycle groups concurrently and keep the crashed-run probes inside them. Each group's crashed-run probes would race the other groups' runs.

### Take browser and upgrade networks from the backend test range

Backend fixtures pick a random `/24` from `10.160.0.0/11` and retry when Docker reports an overlap (`backend/internal/lib/testcontainers/network_subnet.go:16-34`). The upgrade test, which already imports backend test libraries, creates its own network through the same helper, and a test asserts that the created network's subnet lies in that range. The browser suite cannot import backend code (`docs/architecture.md`, End-to-end suites), so the supervisor picks a subnet from the same range, passes it to Compose as `E2E_NETWORK_SUBNET` for the default network's IPAM, and retries `up` on an overlap error before any container starts. `docs/testing.md` names the range as the test network range that both implementations follow, so the shared value has one documented owner.

The restore networks that verification agents create keep coming from Docker's default pools. The agents cannot choose a range, and leaving the pools to them is the point of this decision.

Alternatives:

- Enlarge Docker's default pools in `daemon.json`. It requires host configuration, which `docs/testing.md` avoids, and it still leaves the agents competing with test networks.
- Have the supervisor create the network and attach Compose to it as external. Compose would no longer remove it, so cleanup and the crashed-run sweep would need a new resource type.

### Let the runner choose the browser UI worker count

`frontend/playwright.config.ts` drops `workers: 2`. `frontend/tooling/runBrowser.ts` passes a worker count into the container: two when it runs in CI, which the CI spec requires per shard, and a count derived from the host's processors otherwise. The runner states the reason for the local count beside it, as `docs/testing.md` requires for any limit. The container sets `CI=1` for every run (`frontend/tooling/runBrowser.ts:154`), so the decision is made on the host, before the container starts.

Alternative: Playwright's own default of half the logical processors inside the container. It ignores CI's requirement of exactly two workers and cannot be explained beside the code.

### Fix timing under load in the tests, measured run by run

Concurrency comes first; then the concurrent schedule is repeated. Each test that fails only under load is fixed per the root `AGENTS.md` rule on flaky tests: wait for the condition it depends on, or raise a limit the operation legitimately needs, with the reason in this design. Retries and moving the suite back into a sequential stage are not allowed.

The rounds exposed four such tests, each fixed at its cause:

- The slow group itself: with Go's default of one test per logical processor, the Patroni package started dozens of clusters at once. The Docker daemon's log showed containers that did not exit within ten seconds of a kill and iptables lock contention; stops and network removals outlasted their deadlines, and leaked clusters slowed the next packages. The group now runs with `-parallel=8`, as the limits decision below records.
- `TestWalSourceReconnect` (`backend/internal/app/apptest/physicaltest/source_reconnect.go`) read the WAL queue directory while it polled, and failed at once when the stream had not created that directory yet, which a loaded machine delays. A missing directory now counts as an empty queue, so the wait keeps polling.
- `RunPatroniSwitchovers` (`backend/internal/app/apptest/physicaltest/patroni_switchover.go`) expected archived WAL that ended before the first full backup. That WAL exists only when the stream connects before the full backup starts, which a loaded machine reorders. The test now looks for WAL that ends before the first retained full backup, which the incremental's activity always leaves, and waits for it to be archived.
- `TestWalTransactionTimeBoundary` (`backend/internal/engines/postgresql/physical/wal_transaction_time_test.go`) expected no completed transaction in the first segment. Autovacuum committed one there when the steps between the WAL switch and the large record took longer. Its server now runs without autovacuum, through a new `WithoutAutovacuum` option of the physical PostgreSQL fixture.

One failure had an external cause: in round F a browser run's `recovery` container could not install packages because the PostgreSQL apt mirror was in the middle of a sync. The container installs packages at start, so such a run depends on the mirror; that is recorded as a follow-up rather than fixed here.

Alternative: keep the browser UI and image suites sequential with each other, as `scripts/test.sh:156` does now. That keeps the run long and hides the defects `docs/testing.md` asks us to fix.

### Set package and suite limits from measurements

The slow group's `-p=8` was justified by exhausting the network pools. Backend fixtures now avoid those pools, so the limit is re-measured: the run raises it while the full run gets shorter and peak memory stays under 32 GB, and the comment beside it states the measured reason. The slow limit stays below the other group's `-p=16`, as the slow-package requirement says. The same applies to the number of lifecycle processes.

Measurements, each of the slow group alone with the end-to-end packages:

| Limits | Duration | Added memory | Result |
|---|---|---|---|
| `-p=8`, Go's default `-parallel` of 32 | 618 s | 20.0 GB | Patroni failed on Docker deadlines and leaked clusters |
| `-p=8 -parallel=8` | 493 s | 10.9 GB | passed, at most 88 containers |
| `-p=8 -parallel=16` | 475 s | 14.8 GB | a Patroni switchover scenario failed |

The group lasts as long as `internal/agent/e2e` (about 430 s) and the Patroni package (about 490 s), so a higher package limit only adds memory. `-p=8 -parallel=8` stays.

The same limit holds for `mise run test`, whose slow group lacks the end-to-end packages. Measured alone, that group took 236 s at Go's default `-parallel` and failed: an FTP login outlasted the provider's ten-second connection test. At `-parallel=8` it took 308 s and passed, and the whole `mise run test` took 419 s.

The lifecycle probes without crashed runs take 490 s in sum alone, and the longest single probe 93 s. Three groups took 229-259 s together alone and 511-570 s under the full load. Split round-robin into seven parts with `--shard`, they took 118 s together alone, and 52-115 s after the exclusive stage of a round.

The browser UI suite with 16 workers took 118 s alone, against 220 s with two.

Alternative: keep the current values. They were chosen for `mise run test` without the end-to-end packages and without concurrent browser suites, so they are not evidence for the new schedule.

### Module rules that constrain the approach

- Root `AGENTS.md`: no constant concurrency caps in tests, flaky tests are fixed at the cause, agents never start `mise run test-full` on their own, and other worktrees' runs must not be disturbed.
- `backend/AGENTS.md`: the upgrade test change uses existing backend test libraries and the backend's lint and format.
- `frontend/AGENTS.md`: the browser UI configuration and runner changes follow frontend lint, types and tests.
- `docs/architecture.md`, End-to-end suites: the browser suite imports nothing from `backend/`, so the subnet range is duplicated by convention, not by import.
- `docs/glossary.md`: one term per concept, so "crashed-run probe" names the marker, the options and the prose.

### Measured rounds

Each round ran the full run's suites through their own commands at once, sampling host memory and pressure every few seconds; "peak" is the highest used memory of the host, whose desktop and services used 11-15 GB before each round.

| Round | Order | Duration | Peak memory | Result |
|---|---|---|---|---|
| A | three lifecycle groups first, slow group at Go's default `-parallel` | 1261 s | 41.0 GB | slow group failed on Docker deadlines |
| B | as A, with A's leaked clusters still present | 1765 s | 36.8 GB | slow group, upgrade test and one lifecycle group failed |
| C | three lifecycle groups first, `-parallel=8` | 970 s | 30.3 GB | passed, too slow for the target: groups ended at 570 s, then the exclusive stage took 400 s |
| D | seven lifecycle parts first | 856 s | 32.8 GB | `TestWalSourceReconnect` failed |
| E | seven lifecycle parts first | 930 s | 30.7 GB | `RunPatroniSwitchovers` failed; one part took 451 s under load |
| F | exclusive stage first | 838 s | 29.7 GB | `TestWalTransactionTimeBoundary` failed; a lifecycle part hit the apt mirror sync |
| G | exclusive stage first, all fixes | 868 s | 29.8 GB | passed |
| H | exclusive stage first, all fixes | 838 s | 31.3 GB | passed |

With the exclusive stage first, the crashed-run probes take about eleven minutes under the slow group's load, and the browser runs after them about two. With it last, the lifecycle parts take five to seven and a half minutes under load, and the exclusive stage after them another eight to nine. In round C, CPU pressure peaked near 57 % in the first five minutes and stayed near zero once the slow group ended, so the critical path is the serial exclusive stage, not the machine's capacity.

## Risks / Trade-offs

- [Memory exhaustion: the slow Go packages, two complete browser applications with private Docker daemons, the crashed-run probes or seven lifecycle parts, and the upgrade test may together exceed 32 GB] → Task 1 measures each suite's peak alone; the concurrent rounds sample host memory, and the runner lowers the largest consumer's concurrency with the measured reason.
- [Disk space: concurrent builds and private Docker daemons grow Docker's storage] → the run removes its image at exit; `docs/testing.md` keeps the wide disk margin.
- [New flakes appear once suites overlap] → they are fixed in the tests, and the concurrent schedule must pass twice in a row.
- [Another worktree's browser run removes a crashed run during the exclusive stage] → this race exists today. The browser suite forbids a machine-wide lock, so `docs/testing.md` documents the limit instead.
- [Fail-fast removed: a broken backend now costs a full run before its failure is reported with the others] → every suite's result arrives in one run of about 14 minutes, instead of one failure per 40-minute run.
- [The critical path is the serial exclusive stage of about eleven minutes under load, so the run lands near 14 minutes, above the target of about ten. The limit is 20 minutes, because the measured rounds took 14 to 15.5 minutes and a limit of 15 would fail a slightly slower run] → a shorter stage needs either fewer probes in it or probes that tolerate other runs' sweeps; both change the browser suite's lifecycle rules and are left as a follow-up.

## Migration Plan

The change affects only local test commands and their documentation. Rollback is a revert of the change.

Task 5.2 checks the time target by running the suites of the new schedule concurrently through their own commands and recording each round's wall-clock time, image build included. Agents start `mise run test-full` only on the user's direct request; the user has asked for measurements during implementation, so a confirming `mise run test-full` can follow on that request, but no task depends on it. The measured durations and peak memory replace the estimates above.

## Open Questions

- The exclusive stage holds the critical path. Moving the probes that register teardown only as a safety net, such as graceful interruptions, out of it would save about a minute alone; a separate change can weigh that against the automatic rule.
- The `recovery` container of a browser run installs packages from the PostgreSQL apt mirror at start, so a mirror sync can fail a run. Baking those packages into its image would remove the dependency.
