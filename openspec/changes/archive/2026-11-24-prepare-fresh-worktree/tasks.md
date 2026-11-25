## 1. Local settings through mise

- [x] 1.1 Add `_.file = [".env.example", ".env"]` to the `[env]` section of `mise.toml`. Add `scripts/local_settings_test.py`, which copies `mise.toml` and `.env.example` into a temporary trusted directory and checks, through `mise env --json` and a task that prints values, that:
  - without `.env`, a task sees the example values;
  - a one-key `.env` overrides that key and leaves the other example values;
  - a full copy of the example with one changed value yields that value and the example values for the rest;
  - a key added to the example after `.env` was written reaches the next task;
  - an edit to `.env` between two runs reaches the second run without an export;
  - the activated-shell environment holds `GOOSE_DRIVER` and `GOOSE_DBSTRING` with the example values.

  Verify with `mise exec -- python3 -m unittest scripts/local_settings_test.py`.
- [x] 1.2 Remove the `.env` sourcing block from `scripts/test.sh`, and update `scripts/test_sh_test.py` where it relies on that block or on a parent environment without the example settings. Verify with `mise exec -- uv sync --locked --project e2e/browser` followed by `mise exec -- python3 -m unittest discover -s scripts -p '*_test.py'`.
- [x] 1.3 Remove the `.env` check from `scripts/dev.sh`, and verify with `mise run lint-shell`.

## 2. Tool installation before the local app

- [x] 2.1 Add a hidden `dev:prepare` task to the internal section of `mise.toml` that runs `mise install --locked --include-task-tools`, and make `[tasks.run]` depend on it. Update the section comment if it no longer describes every task in it. In `scripts/development_commands_test.py`, add tests that `mise tasks deps run` lists `dev:prepare`, and that a temporary mise project reusing the real `dev:prepare` command, with a pin the lock file cannot resolve, makes `run` exit with a non-zero status before its body writes a marker file. The existing `test_only_the_supported_user_commands_are_visible_from_every_module` keeps checking that only the nine everyday tasks are visible. Verify with `mise exec -- uv sync --locked --project e2e/browser` followed by `mise exec -- python3 -m unittest scripts/development_commands_test.py`.
- [x] 2.2 Check that a tool installed by `dev:prepare` reaches the `PATH` of `run`. In a temporary trusted mise project that pins `jq` at a version absent from `~/.local/share/mise/installs/jq`, a `run` task that depends on the real `dev:prepare` command must print that version from `jq --version`. Afterwards leave the host's installs as they were before the check: remove that version with `mise uninstall` only if the check installed it, and reinstall it if it was present before. Verify by the printed version and by comparing the installs directory before and after.

Scenario 'Worktree pins a tool version the host lacks' has no regression test, because a positive test needs network access to download the tool; task 2.2 checked it once. `scripts/development_commands_test.py` covers the failure path.

## 3. A fresh worktree end to end

- [x] 3.1 Create a temporary worktree with `git worktree add --detach <temporary directory> HEAD`, and copy into it every file this change creates or modifies, so it has neither `.env` nor any `node_modules`. If ports 4005, 5173 and 3000 are free, run `mise run run -- --tui=false` there in the background. Wait until the backend health endpoint, the frontend on 5173 and the website on 3000 answer and the seed process reports success, and run `goose -dir backend/migrations status` there through the mise environment without exporting anything, checking that it exits with status 0. Then interrupt the run and check that the development containers stopped. If one of those ports is busy, another app run is active and would be disturbed: skip the app part and report why. When the app part is skipped, run the same `goose` check against the development database of the other run, which uses the same settings. In either case, run `mise run test` in the temporary worktree and verify it exits with status 0, which shows that `scripts/test.sh` gets the example `TEST_DATABASE_DSN` without `.env`. Finally remove the temporary worktree with `git worktree remove --force` and verify with `git worktree list` that it is gone.

During this apply the ports were held by another app run, so the app part of task 3.1 was skipped: scenario 'Fresh worktree' of `mise run run` is still unverified for the backend, frontend, website and seed. `mise run test` and `goose status` did run in the fresh worktree without `.env`.

## 4. Documentation and messages

- [x] 4.1 In `AGENTS.md`, replace "In a fresh worktree, copy `.env.example` to `.env`" with a sentence saying that settings come from `.env.example`, that an optional `.env` overrides them per key, and that `mise run run` installs missing pinned tools. Change the goose row of the command table so it relies on mise loading the settings rather than on an exported `.env`. In `backend/README.md`, change line 3, "Copy .env.example to .env in the repo root", and the migration note on line 41 in the same way. Verify by reading the changed lines.
- [x] 4.2 Change the error in `backend/internal/lib/testdb/testdb.go:166` so it says that `TEST_DATABASE_DSN` comes from `.env.example` or `.env` through mise. Verify with `mise run backend:lint` and `go -C backend test ./internal/lib/testdb/...`.
- [x] 4.3 Format the files of this change with `mise run python:format` and `golangci-lint fmt` on the changed backend package, leaving other agents' files alone, then run `mise run lint` and verify that it exits with status 0, which includes the formatting checks of every module. Then verify that `rg -n -i 'cop(y|ies).{0,3}\.env\.example|\.env.{0,2} exported' --glob '!openspec/**' .` prints nothing.

## Change verification

```sh
mise run test
```
