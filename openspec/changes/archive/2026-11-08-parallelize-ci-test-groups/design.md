## Context

See [proposal.md](proposal.md) for motivation and scope. The baseline is the successful manual [CI and Release run 42495867813](https://github.com/databasus/databasus/actions/runs/42495867813), commit `b67abaee`, on native amd64 and arm64. Its required checks passed on their first attempt. The following times are observed job or step durations, not estimates of the new arrangement:

| ARM work | Observed duration | Proposed arrangement | Estimated longest part |
| --- | --- | --- | --- |
| Backend `other`, including provider setup | 19.5 minutes | Three test shards plus one isolated provider leg | 6–9 minutes |
| Browser UI job | 12.6 minutes | Three UI shards | 4–6 minutes |
| Browser lifecycle step | 15.9 minutes | Three lifecycle groups | 6–8 minutes |
| Main browser scenarios with environment setup and cleanup | 1.6 minutes | One job, separate from lifecycle | About the current duration |

These estimates include repeated setup only approximately. The baseline UI job waited almost ten minutes to start, and the final lifecycle job determined the workflow's completion time. Extra runners can shorten execution while leaving queue time unchanged or increasing it.

Baseline implementation examined before this change:

- `.github/workflows/ci-release.yml:450` defines nine backend matrix legs per architecture. Its test step at `:548` runs the root provider package alone before the remaining packages with `-race`, `-p=8` and two concurrent scenarios per binary.
- `mise-tasks/backend/test-group-packages:10` discovers package ownership; `mise-tasks/backend/test-group-tests:32` assigns top-level test names using recorded weights and a default weight for unknown names. It already supports splitting tests within a package.
- `.github/workflows/ci-release.yml:206` runs one UI job per architecture. `frontend/tooling/runBrowser.ts:18` rejects shard arguments, and `frontend/tooling/browserInside.ts:31` always runs the whole UI project. `frontend/playwright.config.ts:8` enables fully parallel tests with two workers and no retries.
- `.github/workflows/ci-release.yml:244` combines browser unit checks, main scenarios and lifecycle checks in one job per architecture. `mise.toml:66` starts the lifecycle suite directly through pytest. Lifecycle probes drive the public command and retain independent cleanup guarantees.
- `scripts/backend_ci_execution_test.py:18`, `scripts/frontend_ci_test.py:15`, `scripts/native_release_test.py:1` and `scripts/release_gate_test.py:1` exercise workflow execution, native coverage and release prerequisites; they must be updated with the workflow rather than bypassed.

The approach follows `docs/architecture.md:404` and `:412`: browser e2e stays outside the application trees, while CI tooling remains outside production modules. Root `AGENTS.md`, `backend/AGENTS.md` and `frontend/AGENTS.md` constrain the touched areas. No application architecture change is needed.

## Goals / Non-Goals

**Goals:**

- Give the longest work independent runners, using the same partition on both architectures.
- Prove complete, disjoint selection at the test level and require every architecture and shard before publication.
- Preserve the complete local commands, failure cleanup, native execution, race detection and immutable screenshot references.
- Make each selected part directly reproducible locally without a push.

**Non-Goals:**

- Enforce fixed runtime targets in assertions or change runner profiles to achieve the estimates.
- Introduce a general scheduling service, a new dependency, or higher concurrency inside existing runners.
- Change product scenarios or the suite's security and cleanup guarantees.

## Decisions

### 1. Split `other` at top-level tests and isolate the provider root once

Add `other-1-of-3`, `other-2-of-3`, `other-3-of-3` and `storage-providers` to the backend matrix in place of the whole `other` entry. All four use the existing eight-CPU Ubuntu 22 runner profiles for their architecture initially. Other backend matrix assignments remain intact.

The `storage-providers` selection contains exactly `databasus-backend/internal/modules/storages/internal/providers`. Its subpackages stay in `other`. Exclude the root package from every `other` selection before forming shard patterns. The provider leg runs only that package with `-race -p=1 -parallel=1`; it must not fall through into a second test invocation. The three `other` legs retain `-race -p=8 -parallel=2`, existing time limits, resource diagnostics and failure propagation.

Reuse the weighted test selector and pin its collation to `LC_ALL=C`, so runner locale cannot change test ordering or shard assignment. Record timings through a local race-enabled run of the affected selection and populate weights for slow `other` tests. The existing name-based selector can place identically named tests from different packages together; account for their combined cost when recording a shared name. Coverage verification uses `(package, test)` identities, so repeated names cannot hide an omitted package. Unknown tests retain a default weight and stale weights never exclude a test. Each existing non-upgrade package belongs to exactly one logical group; shards partition that group's tests.

**Alternatives rejected:** Whole-package distribution leaves the observed twelve-minute `backups` package as a lower bound. Increasing in-runner concurrency changes resource pressure and can revive timing failures. Running the provider root at the start of every shard repeats its tests and setup.

### 2. Add one validated UI shard argument to the existing command

Support `pnpm --dir frontend test:browser --project=ui --shard=1/3` and its other two selections. The host runner validates the optional argument before building or starting a container: positive integer index and count, index no greater than count, one occurrence only. Reject malformed values and unknown options. Forward the validated shard into the container and append it to the UI test invocation. The command without a shard still runs the complete UI suite.

Keep two workers, no retries, the same Docker environment and exact screenshot comparison. Reject combining shard selection with reference updates; reference updates retain the existing complete-suite path. CI artifact names identify architecture and shard, while local run directories remain unique. Each shard retains the existing browser harness probes.

Use test enumeration to compare the full UI project with the union of its three shard selections, checking full test identities and a disjoint union. Add argument and forwarding tests to the existing browser tooling tests. Do not add a second runner or a second set of screenshot references.

**Alternatives rejected:** Three hand-maintained file lists can omit newly added scenarios. More workers on one runner retain the same resource bottleneck. Arbitrary argument passthrough expands the command's behavior beyond the required selection.

### 3. Separate main browser scenarios from three lifecycle groups

Keep `test-e2e-browser` for the suite's unit checks and main real-application scenarios. Add `test-e2e-browser-lifecycle` with architecture and group axes. Both jobs depend directly on `build-storage-image`; they do not depend on each other, backend lint or backend groups. They use the existing four-CPU architecture-specific runner profiles.

Every image-consuming job downloads and checks the candidate for its own architecture and commit. Reuse the existing artifact-transfer steps and their failure handling, without rebuilding the application image. Keep the lifecycle suite sequential inside each job; its existing overlapping-run probes still create their intentional overlap.

The initial lifecycle partition is based on file boundaries and the baseline's log timestamps. Those boundaries approximate group cost; they are not individual-test profiling:

| Group | Files assigned explicitly | Baseline active cases with a named image | Approximate time |
| --- | --- | --- | --- |
| `failures` | `test_failing_runs.py` | 9 | 5m 53s |
| `interruption` | `test_crashed_runs.py`, `test_interruption.py` | 6 | 5m 45s |
| `isolation` | Every other lifecycle test file | 7 | 4m 16s |

The complement includes named-image checks, overlapping runs, probe cleanup and CI-without-image refusal. It also retains the build-image and fresh-checkout tests that already skip when CI supplies a candidate. New files join `isolation` automatically, and new tests in an existing file inherit that file's group. Coverage tests compare collected node identities under the same environment, including existing conditional skips; selection must neither introduce skips nor discard new cases.

Add an optional group to `mise run e2e:lifecycle`, such as `mise run e2e:lifecycle failures`. With no group, it executes the full lifecycle directory exactly as today. A small typed selector discovers files, validates the group and replaces its process with the existing pytest invocation, preserving direct signal delivery and exit status. Invalid groups fail before any Docker work; a selected group with no collected cases fails. Preserve each group's unrelated-container guard and unconditional cleanup of deliberately killed runs. The existing standalone lifecycle-interruption suite must still exercise the public default command successfully.

**Alternatives rejected:** Keeping lifecycle behind the main scenario job retains an unnecessary serial dependency. Parallelizing all lifecycle cases in one process or daemon increases interference and resource pressure. Selecting arbitrary keywords through an environment variable makes exact coverage harder to establish and can collide with filters inherited by nested probe runs.

### 4. Keep every new result in the release gate

The backend and UI matrix jobs retain their existing job IDs; their complete matrices are required. Add the new lifecycle job ID to version-selection prerequisites and the automated gate checks. Every architecture/group pair is mandatory, with matrix fail-fast disabled and no allowed test failures. A failed, cancelled, skipped, missing or timed-out shard prevents version selection and publication. Download retry steps retain only their existing transfer-specific handling.

Add mutation checks that remove or duplicate a matrix entry, disconnect the lifecycle prerequisite, or permit a test failure. Validate backend test selection, UI enumeration and lifecycle collection independently of textual workflow assertions.

This replaces nine backend legs with twelve, one UI leg with three, and one combined browser e2e leg with a main-scenario leg plus three lifecycle legs per architecture: sixteen additional matrix legs overall. It does not duplicate architecture-independent security scans or frontend lint.

**Alternatives rejected:** An optional performance-only workflow would leave the release critical path unchanged. A successful result on one architecture cannot replace the other. Retaining the old whole-suite CI entries would duplicate work.

## Risks / Trade-offs

- **More setup and runner demand** → Preserve runner sizes and in-runner concurrency for the first comparison. Treat per-part runtime ranges as estimates, and compare queue time separately from execution time. Native hosted timing evidence follows implementation; it is not a prerequisite that requires a push to complete a local implementation task.
- **Unknown weights or one indivisible slow test** → Measure individual top-level tests locally, retain inclusion for unknown names, and record any remaining lower bound rather than hiding it with retries or smaller assertions.
- **Omitted or repeated tests** → Assert package ownership and complete disjoint unions of test identities; automatically include new packages, tests and lifecycle files.
- **Provider timing changes** → Keep its complete root package on a dedicated runner with one package and one scenario at a time; keep its nested packages in the sharded remainder.
- **Signal or cleanup regression in the lifecycle selector** → Replace the selector process with pytest, retain default-command behavior, and run the standalone interruption suite as well as all lifecycle groups.
- **Reference races or artifact overwrites** → Reject sharded reference updates and give CI failure artifacts distinct architecture/shard names.
- **Branch rules refer to old check names** → The workflow removes old whole-group matrix entries. Assess any external required-check names before adoption; no compatibility jobs are added.

## Migration Plan

Implement selectors and their coverage checks before wiring the new matrices. Update the local verification commands and `docs/testing.md` in the same change. Verify all parts locally, including interruption and failure propagation, and leave the implementation ready for review and commit.

The following hosted comparison can use a manual `develop` run after adoption. Compare required coverage, each architecture's shard durations, repeated setup, queue time and total completion against the baseline. A green run proves execution on the hosted architectures; local amd64 tests and workflow checks alone do not prove native ARM runtime success.

If additional runners make queues or resource cost unacceptable, revert the partition and workflow changes together, including their CI delta spec. Do not retain partial matrices or bypass the release gate to obtain a shorter run.

## Local implementation verification

On 2026-11-08, native amd64 runs with race detection passed all three `other` selections. The initial measurement runs took 142.0, 124.3 and 133.4 seconds. Ninety slow-test weights were added; identically named tests contribute their combined package durations, and existing physical-group weights were unchanged. The longest individual top-level test observed in that measurement was MariaDB `TestSource`, at 44.6 seconds. The shared `TestSource` name across engine packages has a summed weight of 111 seconds.

The isolated provider package passed with race detection, one package and one scenario at a time, without overlapping test work. Go reported 119.288 seconds; the complete command took 122.05 seconds. Final shard execution is checked against one saved source inventory and weight set, so concurrent changes to the shared checkout cannot silently mix different partitions. Local wall times do not predict hosted ARM duration.

The three real UI shard runs passed 110, 109 and 109 tests. Playwright reported test-step durations of about 1.3 minutes, 48.0 seconds and 50.1 seconds; these exclude the outer Docker setup. The ordinary complete UI command also passed all 328 tests. Reference images were unchanged. Enumeration verification runs in an isolated copy and confirms complete disjoint coverage after adding a scenario.

The routine `mise run test` passed, including 2,879 frontend tests. The script suite passed all 172 tests. Workflow mutations cover missing and duplicate matrix legs, a missing lifecycle release prerequisite, incorrect candidate transfer/architecture, skipped required jobs and allowed test failures. Formatting, lint, shell checks and the production frontend build passed.

Final backend verification used commit `97dabdaa` and a fixed source/weight digest for all three selections. The shards completed 289, 285 and 302 top-level cases in 174.8, 105.1 and 99.4 seconds. Comparing Go syntax-tree declarations with actual test events confirmed all 876 `(package, test)` identities exactly once, with no failures. One existing child-process helper, `TestOpenAsChildProcess`, skips its direct invocation; its parent test exercises it. The provider root passed separately. These are local amd64 observations under concurrent verification load, not hosted performance targets.

All three lifecycle groups passed against the same checkout-built candidate, `databasus-ci-partition:local-20261108`: `failures` passed 9 cases in 261.91 seconds, `interruption` passed 6 in 229.99 seconds, and `isolation` passed 7 with the existing 5 image-build exclusions in 260.34 seconds. Each group retained its unrelated-container guard and registered run cleanup. Collection checks compare their disjoint union with the full directory under both named-image and local-build environments; they do not pin the suite's growing case count.

GitHub branch settings were inspected on 2026-11-08 before adoption. The API reported no required status-check contexts for `develop` or `main`. `develop` was unprotected; the active default-branch ruleset prevented deletion and force pushes and contained no required-check rule. No observed branch rule referenced the replaced whole-group check names. These settings can change independently of the workflow.

A locale regression check reproduced different backend assignments under `C` and `en_US.UTF-8`. The selector now exports `LC_ALL=C`; a regression test verifies the environment of the real sorting command. All three resulting selections exactly match the saved, fully executed selections above, so the race-enabled execution evidence remains applicable. The same source and weights now produce the same partition across runner locales.

The complete default lifecycle command passed all 27 cases without the named-image exclusions in 1,055.07 seconds. Its standalone interruption suite passed all 3 cases in 47.44 seconds, including termination and later cleanup after a killed lifecycle command. All six physical and physical API shard patterns were also compared against the baseline selector and duration weights and remained unchanged.

The complete `mise run e2e:browser` command passed all 6 main application scenarios in 121.92 seconds (147.2 seconds including setup and cleanup). Its 5 dedicated probe cases kept their existing `E2E_PROBE` selection skips; lifecycle verification exercises those probes separately. After all runs, no browser/e2e containers, networks, volumes, built run images or supervisor records remained. The separately prepared named candidate tag was removed after verification. New native ARM and complete hosted-workflow results still require adoption and a manual CI run; no commit or push was performed during apply.

The browser command also verifies the actual inner entry point inside its pinned container before running application scenarios. An isolated static fixture supplies six recording cases: a complete run must execute all six, while the three shard selections must execute cases `[1, 2]`, `[3, 4]` and `[5, 6]`. Each invocation runs the real build command, preview and browser harness probes. An isolated mutation removing inner `BROWSER_SHARD` forwarding failed because the first shard executed all six cases instead of two. This supplements full-suite enumeration and host argument checks, and adds fixture builds and probe setup to each UI job. With forwarding intact, all four fixture selections and the complete 328-case UI suite passed. All 208 frontend tooling tests, frontend lint and the production build passed.
