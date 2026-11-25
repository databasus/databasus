## Why

A worktree created by Codex, Claude Code or `git worktree add` has no `.env`, so `mise run run` stops at once and asks the developer to copy `.env.example`. A worktree that sits on a commit with other tool pins also needs a manual `mise install` before the app starts. Everything else a fresh worktree needs, such as the pnpm store and the Go caches, is already shared across the machine. These two manual steps are what is left, and they should not depend on which client created the worktree.

## What Changes

- mise loads `.env.example` and then `.env` into the environment of every task and of every shell with `mise activate` inside the repository. A key in `.env` overrides the same key from the example, and a missing `.env` is skipped.
- `.env` becomes an optional file of local overrides. A developer keeps only the values that differ from the example in it, such as OAuth credentials.
- `mise run run` no longer requires `.env`. It first runs a new hidden task that installs missing pinned tools from the lock file. `scripts/dev.sh` keeps installing the frontend and website dependencies from their lockfiles, as it does today.
- `scripts/test.sh` stops sourcing `.env`, because mise already supplies the same values. `mise run test` then works in a worktree without `.env`.
- The fresh-worktree instruction in `AGENTS.md`, the goose row of its command table, the setup and migration notes in `backend/README.md` and the error message of `backend/internal/lib/testdb/testdb.go` stop telling the reader to copy `.env.example` or to export `.env`.

No caller breaks. An existing full copy of `.env.example` in `.env` keeps working, because every key in it overrides the identical key from the example.

### Out of scope

- Hooks of a particular agent client: Claude Code's `WorktreeCreate` and `.worktreeinclude`, and Codex local environment setup scripts.
- A git `post-checkout` hook or a mise `enter` hook that prepares a worktree on creation or on `cd`.
- Copying `node_modules`, Go caches or other downloaded files between worktrees.
- Changes to the backend settings loader in `backend/internal/lib/config`.
- Automatic tool installation before `mise run test`, `mise run test-full`, `mise run lint` or any other task. `run_auto_install = false` stays, so CI keeps installing only the tools each job needs.
- CI, which keeps copying `.env.example` to `.env`.
- Masking secret values in task output.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `local-development-workflow`: `mise run run` works in a fresh worktree without `.env` and installs missing pinned tools itself. A new requirement covers local settings that fall back to `.env.example`, with `.env` overriding per key.

## Impact

- `mise.toml`: an `[env]` file entry and a hidden preparation task that `run` depends on.
- `scripts/dev.sh`: drops the `.env` check.
- `scripts/test.sh`: drops sourcing of `.env`. `scripts/test_sh_test.py` may need updates if it relies on that.
- `backend/internal/lib/testdb/testdb.go`: the error message for a missing `TEST_DATABASE_DSN`.
- `AGENTS.md`: the development environment paragraph and the goose row of the command table.
- `backend/README.md`: the setup note and the migration note.
- `scripts/local_settings_test.py` (new) and `scripts/development_commands_test.py`: tests of the settings layering and the preparation before `run`.
- The change answers to the root `AGENTS.md` and to `backend/AGENTS.md` for the one backend message.
