## Why

`mise run test-full` takes about 40 minutes on the reference machine, while CPU and memory stay mostly idle. The backend and frontend tests run together, but everything after them runs one suite at a time: script and hook tests, browser UI, the upgrade test, the browser end-to-end scenarios and 27 lifecycle probes, each of which starts a complete application. `docs/testing.md` already treats a run that waits on an idle machine as a defect, and it forbids constant concurrency caps, yet the full run keeps both. The goal is a full run of about 10 minutes and at most 20, with suites that tolerate running beside each other instead of being kept apart.

## What Changes

- The full run builds the application image once, at the start, under a tag only the full command uses, and removes it at the end. The upgrade test and the browser end-to-end scenarios use that image.
- Every suite of the full run starts as soon as what it needs is ready, rather than after the backend and frontend tests. Only the crashed-run probes and the lifecycle interruption checks wait until no other browser run of the full run is active. A crashed-run probe is a lifecycle probe that kills a run or prevents its cleanup, so that a later run must remove what remains.
- **BREAKING** (developer workflow): a failed suite no longer stops the full run from starting the others. Every suite finishes, and the summary names each failed suite. The rule that a backend or frontend failure prevents the image build and the upgrade test is deleted.
- The full run prints how long each suite took, so a regression in the run's length is visible without a separate investigation.
- The browser UI suite takes its local worker count from the runner instead of a constant of two in its configuration. CI keeps two workers per shard.
- Crashed-run probes are recognized automatically, so a runner can run the other probes concurrently and the crashed-run probes alone. The lifecycle command can split a selection into parts, so several processes share the other probes. `docs/glossary.md` defines the term. CI groups stay as they are.
- Isolation fixes needed for concurrency:
  - the upgrade test and browser end-to-end runs create their own Docker networks inside the backend test network range, outside Docker's default address pools, as backend tests already do;
  - the direct upgrade command builds and tags its image under a run-unique name instead of the fixed `databasus-candidate:local`, and removes it afterwards;
  - tests that fail only because other suites load the machine are fixed, by waiting on the right condition or by a timeout that matches the operation, rather than by running them alone.
- The backend package limits in the runner and the concurrency of browser-level suites are set from measurements taken during implementation, with the reason beside each limit.
- `docs/testing.md` records the full run's time target, its scheduling, the shared-image rule and the isolation rules for concurrent suites.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `local-development-workflow`: the full command's ordering, failure handling, time target and per-suite timing; the upgrade test no longer waits for the backend and frontend suites.
- `upgrade-test`: the test's image and network do not collide with concurrent runs.
- `browser-e2e`: runs take networks from the test network range, and crashed-run probes can run apart from the other probes.

## Out of scope

- CI jobs, shards, groups and their worker counts. CI already parallelizes by jobs, and this change keeps its two-worker browser UI shards and three lifecycle groups.
- `mise run test`: it runs the same backend and frontend suites, so it benefits from re-measured package limits, but its composition does not change.
- The Podman subset and the filesystem suite, which stay outside the full run.
- Making `TestVerifyBackup` scenarios concurrent; the agent still allocates networks from Docker's default pools.
- Speeding up individual tests or packages beyond what the full run's critical path requires. A single package that alone exceeds the full run's budget is reported, not rewritten, unless the fix is small.
- Isolation from another worktree's browser end-to-end runs during the crashed-run probes. Their machine-wide cleanup already races that way today; the change documents the limit rather than adding a machine-wide lock, which the browser suite forbids.
- The machine-wide lock that makes `mise run test` and `mise run test-full` take turns.

## Impact

- Answers to the root `AGENTS.md`, `backend/AGENTS.md` (upgrade test and runner limits) and `frontend/AGENTS.md` (browser UI configuration and runner).
- Code: `scripts/test.sh` and its script tests, `mise-tasks/backend/test-upgrade`, `backend/e2e/upgrade`, `e2e/browser` (supervisor, compose project, lifecycle command and probes), `frontend/playwright.config.ts` and `frontend/tooling/runBrowser.ts`, `backend/test-slow-packages.txt` if measurement moves packages.
- Documentation: `docs/testing.md` and `docs/glossary.md`.
- Resources: the full run holds more containers at once, so peak memory and disk use rise toward the 32 GB the reference machine reserves for tests.
