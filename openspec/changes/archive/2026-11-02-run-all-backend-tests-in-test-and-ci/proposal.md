## Why

A defective migration changes user data during an upgrade, and its failure would first appear in a release, yet the migration tests run only when someone remembers them: neither CI nor `mise run test` selects them. The architecture linter's tests are in the same position, so a defect in either can reach a release before anyone sees a failing test.

## What Changes

- The backend test selections of CI and of `mise run test` start from every package of the backend module and drop only what runs elsewhere. `backend/migrations` and `backend/tools/archlint` join the `other` group and the full local run, and a package in any new directory joins them on its own.
- The `test-backend` CI job gets the pinned `golangci-lint`, which the architecture linter tests run.
- The coverage test compares the groups with the whole module instead of repeating the same partial list.
- `.agent-hooks/test_group_test.py` is deleted. It tests the shard runner, not the hooks, and `scripts/test_group_test.py`, which CI runs, already covers the same three scenarios. Its few unique assertions move there.
- `docs/testing.md` describes the new coverage. Its statements that every check runs automatically and that CI runs everything now except the review hook tests, which run by hand when an agent changes the hooks.

This change breaks no callers, schemas, configs or APIs.

### Out of scope

- Running the review hook tests in CI or in `mise run test`. They exist for an agent that changes the hooks and runs them by hand.
- Running the script, frontend or filesystem tests from `mise run test`.
- New tests for migrations or the architecture linter beyond those needed to pin the selections.
- Moving the upgrade test or the real-API browser package into the ordinary selections.
- Changing how the backend groups are sharded, and changing which CI jobs depend on `lint-shell`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `ci-pipeline`: the CI backend groups cover the whole backend module.
- `local-development-workflow`: `mise run test` selects its backend packages from the whole module.

## Impact

- Selection: `mise-tasks/backend/test-group-packages`, `scripts/test.sh`.
- CI: the `test-backend` job in `.github/workflows/ci-release.yml` installs the pinned `golangci-lint`, because the architecture linter tests run it.
- Tests: `scripts/test_group_test.py`, `scripts/test_sh_test.py`; `.agent-hooks/test_group_test.py` is removed.
- Documents: `docs/testing.md`.
- Rules: the change answers to the root `AGENTS.md` and `backend/AGENTS.md`.
- No production code or migration changes. If a newly selected test fails, the cause is repaired, as the Tests section of `AGENTS.md` requires.
