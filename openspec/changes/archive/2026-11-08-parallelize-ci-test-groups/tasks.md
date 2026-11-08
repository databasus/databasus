## 1. Backend ownership and test selection

- [x] 1.1 Add the `storage-providers` package selection and exclude only its exact root package from `other`, leaving nested provider packages and automatic discovery intact. Extend package-ownership tests to cover the complete union, duplicates, exclusions and a newly discovered package. Verify with `mise exec -- python3 -m unittest discover -s scripts -p 'test_group_test.py'`.
- [x] 1.2 Extend weighted selection tests for three `other` shards, including multiple tests in one slow package, identical names in different packages, new tests, stale weights, empty selections, invalid shard indexes and runner-locale variation. Assert complete disjoint `(package, test)` coverage and unchanged physical-group selection. Verify with `mise exec -- python3 -m unittest discover -s scripts -p 'test_group_test.py'`.

## 2. Browser UI shard selection

- [x] 2.1 Add validated optional `--shard=index/count` handling to the existing browser command and forward it into the container's test invocation. Preserve complete runs, two workers, zero retries and existing cleanup; reject duplicates, malformed selections, unknown options and sharded reference updates before Docker work. Verify argument handling, forwarding, failure status and interruption through `pnpm --dir frontend test tooling/runBrowser.test.ts`. The production browser command must also execute the actual inner entry point with a small fixture for full and all three sharded selections, and reject an isolated mutation removing inner forwarding.
- [x] 2.2 Add an automated enumeration check comparing the full UI project with all three shard selections, proving no omissions or duplicates and automatic inclusion of a new test. Verify with `pnpm --dir frontend test tooling` and run `pnpm --dir frontend test:browser --project=ui --shard=1/3`, repeating for `2/3` and `3/3` against the unchanged references.

## 3. Backend execution and duration balance

- [x] 3.1 Replace the whole `other` matrix entries with three `other` shards and one isolated `storage-providers` leg on each architecture, retaining current runner profiles. Update execution tests to prove the provider root runs exactly once with `-race -p=1 -parallel=1`, the remaining shards use `-race -p=8 -parallel=2`, failures propagate, and the provider leg cannot fall through into another test run. Verify with `mise exec -- python3 -m unittest discover -s scripts -p 'backend_ci_execution_test.py'`.
- [x] 3.2 Measure the affected backend tests locally and update duration weights, accounting for shared test names. Use the documented Fedora libedit preparation where needed. Verify each shard with `TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -race -p=8 -parallel=2 -count=1 -timeout 40m -json -run "$(mise run --quiet backend:test-group-tests other-1-of-3)" $(mise run --quiet backend:test-group-packages other-1-of-3)`, repeating for `other-2-of-3` and `other-3-of-3`. Verify the isolated package separately with `TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -race -p=1 -parallel=1 -count=1 -timeout 40m ./internal/modules/storages/internal/providers`; run it without overlapping test work. Record measured shard durations and any indivisible slow test rather than claiming ARM performance from local results.

## 4. Lifecycle group selection

- [x] 4.1 Add the typed lifecycle selector and optional public command group. Assign failing-run tests to `failures`, crashed-run and interruption tests to `interruption`, and every other discovered lifecycle test file to `isolation`. Keep the default full-suite command and direct pytest signal delivery. Cover invalid groups, no collected cases, new files, new tests, conditional image skips, unchanged environment and exit status. Verify with `mise run e2e:unit`.
- [x] 4.2 Add collection-level checks that compare all three groups with the default suite under matching environments, proving a disjoint union without added skips. Run the actual groups locally through `mise run e2e:lifecycle failures`, `mise run e2e:lifecycle interruption` and `mise run e2e:lifecycle isolation` with the same locally prepared candidate named through `DATABASUS_IMAGE`; record group durations and verify each run removes its own resources.
- [x] 4.3 Verify default-command behavior and signal cleanup after changing the public entry point. Run `mise run e2e:lifecycle` and `mise exec -- uv run --locked --project e2e/browser pytest e2e/browser/lifecycle_interruption`, preserving the cases that build their own image locally and their unconditional teardown.

## 5. Workflow matrices and release requirements

- [x] 5.1 Expand browser UI to three shards on each architecture and distinguish failure artifact names by architecture and shard. Keep the existing native runners, pinned dependencies, harness probes and matrix failure behavior. Update workflow tests for forwarding, artifacts and all six required selections. Verify with `mise exec -- python3 -m unittest discover -s scripts -p 'frontend_ci_test.py'` and `mise run lint-workflows`.
- [x] 5.2 Keep browser unit checks and complete main scenarios in `test-e2e-browser`; add the six `test-e2e-browser-lifecycle` architecture/group legs with direct candidate dependencies and the existing native architecture checks and artifact-transfer steps. Require the complete lifecycle matrix before version selection. Verify with `mise exec -- python3 -m unittest discover -s scripts -p 'native_release_test.py'`, `mise exec -- python3 -m unittest discover -s scripts -p 'release_gate_test.py'` and `mise exec -- python3 -m unittest discover -s scripts -p 'frontend_ci_test.py'`.
- [x] 5.3 Extend mutation tests to reject an omitted or duplicated backend/UI/lifecycle matrix entry, a disconnected lifecycle prerequisite, allowed test failures, incorrect candidate architecture and missing required results. Preserve manual-run publication restrictions and all unrelated prerequisites. Verify through the native-release and release-gate test commands from task 5.2, plus `mise exec -- python3 -m unittest discover -s scripts -p 'ci_events_test.py'`.

## 6. Documentation and integrated verification

- [x] 6.1 Update `docs/testing.md` with the twelve backend legs per architecture, three UI shards, separate lifecycle groups, complete local defaults and focused commands. Keep measurements distinct from projected runtime ranges and explain queue/setup costs. Search for affected commands and group counts, then verify their documented selections with the package, UI enumeration and lifecycle collection tests from the preceding tasks.
- [x] 6.2 Run the full script checks after locked dependency preparation with `mise exec -- uv sync --locked --project e2e/browser` and `mise exec -- python3 -m unittest discover -s scripts -p '*_test.py'`. Run `mise run lint`, `pnpm --dir frontend build`, and `mise run e2e:unit`; format only changed files with the relevant repository formatters and check shell changes with `shellcheck`. Resolve blocking review findings and keep verification evidence ready with the implementation.

## Change verification

```sh
mise run test
pnpm --dir frontend test:browser --project=ui
mise run e2e:browser
```
