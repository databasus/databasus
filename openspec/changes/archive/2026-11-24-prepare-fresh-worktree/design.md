## Context

See proposal.md - Why. The approach is shaped by how each consumer reads local settings today:

- `scripts/dev.sh:47-50` refuses to start without `.env`, and `scripts/dev.sh:58-59` installs the frontend and website dependencies with `pnpm install --frozen-lockfile` before any process starts.
- `docker-compose.yml:38-40` interpolates `DEV_DB_NAME`, `DEV_DB_USERNAME` and `DEV_DB_PASSWORD` with no fallback. Compose reads `.env` from the project directory, and a variable in the calling environment wins over it.
- The backend settings loader reads the process environment first and lets `.env` fill only keys that are still unset (`backend/internal/lib/config/load.go:77-82`, `:91-95`). The seed uses the same loader (`backend/cmd/seed_dev/main.go:47-53`).
- Vite reads `.env` from the repository root (`frontend/vite.config.ts:9`) and also copies every `VITE_*` variable from `process.env` (`frontend/node_modules/vite/dist/node` loadEnv).
- `scripts/test.sh:124-128` sources `.env` when it exists, and `backend/internal/lib/testdb/testdb.go:166` asks the reader to copy `.env.example` when `TEST_DATABASE_DSN` is missing.
- `mise.toml` sets `[settings.task] run_auto_install = false`. Commit `ef5e3653` made that choice, and `.github/workflows/ci-release.yml:14-16` relies on it so that each CI job installs only the tools its tasks need.
- Tools, the pnpm store and the Go caches already live in per-user directories shared by every worktree, so the remaining per-worktree state is `.env` and `node_modules`.

The change answers to the root `AGENTS.md`, the development environment section and the documentation consistency rule in particular, and to `backend/AGENTS.md` for the testdb message. It changes no module boundary in `docs/architecture.md`. Its rule that settings come from environment variables with `.env` as an optional local source (`docs/architecture.md:331`) still holds.

## Goals / Non-Goals

**Goals:**

- One mechanism, owned by mise, supplies local settings to every task and to activated shells.
- `mise run run` prepares whatever a fresh or re-pinned worktree lacks, with no hook of any agent client or of git.

**Non-Goals:**

- Running outside mise, for example from an IDE that does not load mise environments. Such a run gets settings only from a `.env` file, as today.
- Speeding up the first `pnpm install` of a worktree beyond what the shared store already gives.

## Decisions

### Layer `.env.example` and `.env` through mise `[env]`

`mise.toml` gets `_.file = [".env.example", ".env"]` in its `[env]` section. A check on mise 2026.11.9 showed three things: a missing later file is skipped without an error, a later file overrides an earlier one per key, and quoted values containing `=` parse correctly. Since every consumer listed in Context lets the process environment win, they all see the same merged values whether or not `.env` exists.

Alternatives rejected:

- Copying `.env.example` to `.env` in `scripts/dev.sh` when `.env` is missing. It is one line, but the copy stops following the example from the moment it is made, and `mise run test` and an activated shell would still have no settings in a worktree where `mise run run` has never run.
- Teaching the backend loader to read `.env.example` as a fallback. That brings a development-only file into runtime configuration logic, and Docker Compose and Vite would still need their own fallback.
- Per-client worktree hooks such as Claude Code's `WorktreeCreate` or `.worktreeinclude` and Codex setup scripts. These would need one setup per client, and a worktree from plain `git worktree add` would still be left unprepared.

### A hidden `dev:prepare` task that `run` depends on

A new hidden task in the internal section of `mise.toml` runs `mise install --locked --include-task-tools`, and `[tasks.run]` lists it in `depends`. The frozen pnpm installs stay in `scripts/dev.sh:58-59`, because they already run before any process starts and need the tools that `dev:prepare` has just installed. A check on mise 2026.11.9 confirmed that a task resolves its tool `PATH` after its dependencies finish, so a tool that `dev:prepare` installs is already on the `PATH` of `run`. The task stays hidden, which keeps the nine everyday tasks of the `local-development-workflow` spec unchanged.

Alternatives rejected:

- Setting `run_auto_install = true`. That would install every tool before any task, including in CI jobs that install only what they need.
- Calling `mise install` from `scripts/dev.sh`. By then the script's `PATH` has already been computed without the newly installed tools.
- A mise `enter` hook or a git `post-checkout` hook. An `enter` hook runs only in activated shells, so agents never trigger it, and it runs on every entry into the repository. A git `post-checkout` hook runs on every branch switch, not only when a worktree is created, and would have to be installed in every clone.
- Making `test` and `test-full` depend on `dev:prepare`. CI runs those tasks and installs tools per job, so the dependency would undo that.

### Drop the `.env` sourcing in `scripts/test.sh`

With mise supplying the merged settings, the `source "$root/.env"` block re-reads the same values with a second parser, bash instead of mise's dotenv reader, and ignores `.env.example`. Removing it leaves one source of truth.

Alternative rejected: keeping it as a fallback for running `scripts/test.sh` outside mise. The repository runs it only through `mise run test`, and a second loader would bring back the drift this change removes.

## Risks / Trade-offs

- [Tests that copy the whole parent environment, such as `scripts/test_sh_test.py:201-234`, now inherit example settings when run from an activated shell] → run the Python tests of `scripts/` and `backend/internal/lib/config`. A test that needs a clean environment drops the keys explicitly, as `backend/cmd/databasus/binary_test.go:423-433` already does.
- [Secrets in `.env` reach the environment of every process started from an activated shell in the repository, agent child processes included, instead of only the processes that read the file] → accepted. Masking is out of scope, and a developer who does not want this exposure keeps secrets out of `.env`.
- [`dev:prepare` adds a step to every `mise run run`] → `mise install` with nothing to install returns at once.
- [A first run in a re-pinned worktree needs network access to install tools] → the same as the manual `mise install` it replaces. A failure stops the run before any service starts.

## Migration Plan

No migration is needed. An existing `.env` that is a full copy of the example keeps working, because each of its keys overrides the identical key. Developers may trim it to the keys they changed. Rollback is a revert of the commit.
