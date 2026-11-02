## Context

See `proposal.md` for the motivation.

Two places select the backend packages, and both name only `./internal/...` and `./cmd/...`: the full run at `scripts/test.sh:65` and the `other` group at `mise-tasks/backend/test-group-packages:19-22`. The coverage test at `scripts/test_group_test.py:34` builds its expected list from the same two directories, so it cannot notice what they leave out. `go list ./...` in `backend/` also returns `api`, `e2e/upgrade`, `e2e/upgrade/upgradecheck`, `migrations` and `tools/archlint`. `api` has no tests. `mise-tasks/backend/test-upgrade:20` already runs `./e2e/upgrade/...`, so the two upgrade packages are covered.

CI runs `scripts/*_test.py` in the `lint-shell` job at `.github/workflows/ci-release.yml:109`. `.agent-hooks/test_group_test.py` tests the shard runner and sits in a directory nothing runs.

The newly selected tests pass locally: the migration package took 11 seconds and `tools/archlint` 5 seconds. `backend/tools/archlint/lint_config_test.go:126,138` runs `golangci-lint`, which a local run gets from `mise`. The `test-backend` job has `install: false` on its `mise` step, so it has no such tool.

The rules that constrain this design are the root `AGENTS.md` sections on tests, documentation consistency and agent instructions, and the test selection rules in `docs/testing.md`. No architecture rule in `docs/architecture.md` or `backend/AGENTS.md` changes.

## Goals / Non-Goals

**Goals:**

- A new backend package needs no selection edit to run, in any directory of the module.

**Non-Goals:**

- Sharing one selection function between `scripts/test.sh` and `test-group-packages`. They exclude different packages, and each is a single line.
- Any automatic run of the review hook tests.

## Decisions

### 1. Select from `go list ./...` and drop named subtrees

`other` becomes `go list ./...` minus the subtrees of the named groups and `e2e/upgrade`. `scripts/test.sh` becomes `go list ./...` minus `internal/app/browsertests` and `e2e/upgrade`. This keeps the rule the main spec already states: a package outside the named groups runs in `other` without a configuration change.

Alternative rejected: appending `./migrations/...` and `./tools/...` to both commands, as the task suggested. The next new top-level directory would run in neither place. The coverage test would catch that for CI only, and `mise run test` would keep missing it.

### 2. Exclude `e2e/upgrade` as a whole subtree

Both selections drop `databasus-backend/e2e/upgrade` and everything below it, `upgradecheck` included. The pattern is anchored to the module root, so a future package such as `internal/lib/e2e/upgrade` is still selected. The upgrade step keeps selecting the subtree, and a test pins that selection, because otherwise a narrowed step would leave `upgradecheck` running nowhere with every test still passing.

Alternatives rejected: excluding only the exact `e2e/upgrade` package runs `upgradecheck` twice per commit, which breaks the "exactly once" rule of the spec. Selecting the upgrade package would fail, since it needs the images of two releases.

### 3. A coverage test that does not repeat the selection

`scripts/test_group_test.py` compares the combined groups with `go list ./...` minus the `e2e/upgrade` subtree, and also names `migrations` and `tools/archlint` as members of `other`. Naming them means that a later broad exclusion fails the test even if the two lists still agree. A second test runs the `other` selector against a `go` stub that returns a package in a new top-level directory, because the real module cannot show that a directory added later is selected.

Alternative rejected: keeping only the set comparison. A too-wide exclusion in the test and in the task together would pass unnoticed, and appending two directories to `other` would pass as well.

### 4. The review hook tests stay a manual check

Neither CI nor `mise run test` runs `.agent-hooks/*_test.py`. These tests exist for an agent that changes the hooks, and that agent runs them by hand as its task's verification. `docs/testing.md` states this and gives the command, and its two general statements change to except these tests: "Every check runs locally, and none of them is manual" at line 10 and "CI runs everything on every pull request" at line 120.

Alternatives rejected: a CI step or a part of `mise run test` costs about 2.5 minutes per run to guard tooling that changes rarely and does not reach users, and a failure there would block backend work for an unrelated reason.

### 5. Delete `.agent-hooks/test_group_test.py` and keep its unique assertions

`scripts/test_group_test.py` already has the concurrent run, the rejected selections and the interruption. Five assertions exist only in the file being removed and move across: the interrupted exit code is exactly 143, the exit code after a failed shard is exactly 1, the output names `running <shard>`, a named shard mixed with `other` is rejected, and no temporary log directory remains after a shard fails. The shard runner is backend test infrastructure, so its tests belong in the directory CI runs.

Alternative rejected: moving the file as it is, as the task suggested. A file of that name is already in `scripts/`, and the two would test the same scenarios twice. The removed file also has a 10-second limit, against 30 seconds in the file that stays, which raises the risk of false failures on a loaded runner. Leaving it in place keeps those five assertions outside every automatic run.

### 6. Install `golangci-lint` in the `test-backend` job

The `mise` step of the job takes `install_args: 'golangci-lint'` instead of `install: false`, and a test in `scripts/test_group_test.py` pins it. The version stays the one in `mise.toml`, and `mise-action` exports the tool's directory on `PATH`, so the plain `go test` step finds it.

Alternatives rejected: installing it only for the `other` group adds a condition to save a download of one binary. Skipping `TestLintConfiguration` when the tool is missing would silently drop the check in the one place that runs it automatically.

## Risks / Trade-offs

- [A defect in the review hooks shows up only when someone runs their tests or uses the hooks] → Accepted. The hooks do not reach users, and a change to them names these tests as its verification.
- [The newly selected tests have never run in the `other` group next to its other packages] → Each test takes its own database, as `docs/testing.md` describes, so the risk is low. A failure is repaired at its cause.
- [The real CI failure cannot be shown inside the tasks, because the project does not allow a CI run as a task verification] → The tasks prove the selection locally. After the first push, the user can break one migration test on a branch and watch `test-backend (other)` fail.

## Migration Plan

There is nothing to migrate. Reverting the commit restores the old selections.
