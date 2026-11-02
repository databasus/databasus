## 1. Backend selections

- [x] 1.1 Rewrite the coverage test in `scripts/test_group_test.py` per design.md decision 3 as `test_every_package_except_the_upgrade_test_belongs_to_exactly_one_group`, and add a test that runs the `other` selector against a `go` stub that returns a package in a new top-level directory only for `list ./...`, and asserts that the package is selected. Run both on the current code and see them fail: `migrations` and `tools/archlint` belong to no group, and the stubbed package is not selected. Add an assertion that `mise-tasks/backend/test-upgrade` selects `./e2e/upgrade/...`, which already holds. Verification: `python3 -m unittest discover -s scripts -p 'test_group_test.py'` fails for those reasons.
- [x] 1.2 Change the `other` group in `mise-tasks/backend/test-group-packages` per design.md decisions 1 and 2, and update its comment. Verification: `python3 -m unittest discover -s scripts -p 'test_group_test.py'` passes, and `mise run backend:test-group-packages other | rg 'migrations|tools/archlint'` prints both packages.
- [x] 1.3 In `scripts/test_sh_test.py`, make the `go` stub record its `list` arguments and, only for `list ./...`, also return the migration, architecture linter, upgrade and upgrade helper packages and a package in a new top-level directory. Then assert that the full run lists `./...`, selects the migration, architecture linter and new-directory packages, and drops the upgrade, upgrade helper and browser packages. See the test fail on the current `scripts/test.sh`, then change the selection per design.md decisions 1 and 2. Verification: `python3 -m unittest discover -s scripts -p 'test_sh_test.py'` passes.
- [x] 1.4 Run the newly selected packages through the project command, and repair the cause of any failure as the Tests section of `AGENTS.md` requires. Verification: `mise run backend:test-package ./migrations/... ./tools/...` passes.

- [x] 1.5 Add a test to `scripts/test_group_test.py` that the `test-backend` job installs `golangci-lint`, see it fail, and change the job per design.md decision 6. Verification: `python3 -m unittest discover -s scripts -p 'test_group_test.py'` passes.
- [x] 1.6 Anchor the upgrade exclusion to the module root in `mise-tasks/backend/test-group-packages`, `scripts/test.sh` and the coverage test, and pin it with a stub package `internal/lib/e2e/upgrade` per design.md decision 2. Verification: `python3 -m unittest discover -s scripts -p 'test_group_test.py'` and `python3 -m unittest discover -s scripts -p 'test_sh_test.py'` pass.

## 2. Shard runner tests

- [x] 2.1 Move the five assertions named in design.md decision 5 into `scripts/test_group_test.py`, and delete `.agent-hooks/test_group_test.py`. Verification: `python3 -m unittest discover -s scripts -p 'test_group_test.py'` passes, `test ! -e .agent-hooks/test_group_test.py` succeeds, and `rg -n 'returncode, 143|returncode, 1\b|running physical|-of-[0-9] other|databasus-test-shards' scripts/test_group_test.py` finds each moved assertion.

## 3. Documents

- [x] 3.1 Update `docs/testing.md`: add rows to the Layers table for the migration tests and for the architecture linter tests, and say in Runs that the full run and CI run both. State that the review hook tests run by hand when an agent changes the hooks, and add their command to the Commands table. Reword the sentences "Every check runs locally, and none of them is manual" and "CI runs everything on every pull request" so that they except these tests. Update the comment in `mise-tasks/backend/test-group-packages` if task 1.2 left it out of date. Verification: `rg -n "discover -s \.agent-hooks" docs/testing.md` finds the command, `rg -n 'backend/migrations|tools/archlint' docs/testing.md` finds the Layers rows and the Runs statement, `rg -n 'none of them is manual|CI runs everything on' docs/testing.md` finds nothing, `rg -c 'review hook tests' docs/testing.md` prints at least 3, and `rg -n '\.agent-hooks/test_group_test' AGENTS.md docs .agents/skills` finds nothing.

## 4. Check and hand over

- [x] 4.1 Run the checks of the touched areas. Verification: `mise run lint-shell` and `python3 -m unittest discover -s scripts -p '*_test.py'` pass.
