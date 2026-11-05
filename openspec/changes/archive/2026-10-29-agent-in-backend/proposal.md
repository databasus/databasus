## Why

This is the first change of milestone 12. The verification agent lives in its own Go module, `agent/verification`, with its own `go.mod`, its own lint configuration and its own CI lint job. The target layout in `docs/architecture.md` already places it at `backend/cmd/verification-agent` and `backend/internal/agent`, and the later changes of the milestone depend on that move: the shared restore code in `engines` (12c) and the generated agent client (12e) can only be imported from inside the backend module.

The separate module also holds the agent to weaker rules than the backend. Its lint configuration enables the standard linters and nine more, and it skips test files. Run under the backend's configuration, the agent's code and tests produce 149 findings, and the architecture check would add four bare goroutine starts. Two of the gaps reach users:

- The agent writes its configuration file, which holds the agent token, with mode `0644`, so every user of the machine can read the token.
- The agent's logger is a package-level variable that the configuration package initializes at import time. Every command, even `version`, therefore creates the log file in the current directory.

## What Changes

- **The code moves without changing behavior**, apart from the file permissions and the logger construction below:
  - `agent/verification/cmd` becomes `backend/cmd/verification-agent`;
  - `internal/config`, `internal/logger` and `internal/features/*` become `backend/internal/agent/{api,config,container,dbconn,heartbeat,logging,restore,runner,start,upgrade,verifier}`, without the `features` level;
  - `internal/testutil` is deleted, and its two helpers move into the tests that use them;
  - `images/postgres`, the image of the restore containers, moves with the agent to `backend/cmd/verification-agent/images/postgres`.
- **One `go.mod`.** The agent's dependencies join `backend/go.mod`. `moby` and `pgx` take the backend's versions. `resty` stays until 12e replaces it with the generated client.
- **The agent passes the whole backend lint and the architecture check**, test files included. The rules that already name the agent in `backend/.golangci.yml` and `tools/archlint` start applying to real code:
  - the daemon start keeps `os/exec` as the one documented exception, because the daemon outlives the command that started it; the check of a downloaded binary runs through `lib/process`;
  - `context.Background()` gives way to the caller's context everywhere except the entry point;
  - `time.Sleep` in the daemon's start and stop gives way to waiting for the condition with a deadline;
  - the four bare `go` statements start through `lib/background`;
  - printing and exit codes stay only in `cmd/verification-agent`;
  - log calls take the `...Context` form, constant messages and `snake_case` attributes;
  - tests wait for conditions instead of sleeping, and take the names the testing skill prescribes.
- **The logger is built once, in `cmd/verification-agent`, and passed down.** No package does work at import time. The log format, the file names and the 5 MB rotation stay as they are.
- **The configuration file holding the agent token gets mode `0600`.** The agent creates and rewrites it with `0600`. At every start it narrows an existing file to `0600`, so installations from earlier releases are closed on their first start after the upgrade. When it cannot narrow the mode, it keeps running and prints a warning that names the file, its current mode and owner, the operating system's error and the command that fixes it.
- **Tasks and hooks.**
  - `mise run agent:build`, `agent:test` and `agent:run` work from `backend/`.
  - **BREAKING (developer tooling):** `mise run agent:lint` and `agent:format` are deleted, because `backend:lint` and `backend:format` already cover the agent's packages. `mise run lint` and `mise run format` lose their agent part. The `agent-format` and `agent-lint` steps of the commit hook are deleted.
  - `agent:run` takes the agent ID and token from the environment, sourcing the root `.env` when it exists; `agent/verification/.env.example` is deleted, and the root `.env.example` does not change.
  - `agent:e2e` and `agent:e2e-podman` still run the docker compose suite. Its agent builder builds from `backend/`.
- **CI.** The `lint-verification-agent` job is deleted. `lint-backend` already verifies module checksums and scans for vulnerabilities, and now covers the agent. Jobs that waited for the deleted job wait for `lint-backend`. `test-verification-agent` runs `mise run agent:test`. The image scan and the two image builds take the new path of `images/postgres`.
- **`Dockerfile`** builds the agent from `backend/` in its own stage, on the same base image and module cache as the server's stage.
- **Deleted from `agent/verification`:** `cmd`, `internal`, `images`, `go.mod`, `go.sum`, `.golangci.yml`, `AGENTS.md`, `CLAUDE.md`, `.gitignore`, `.dockerignore` and `.env.example`. Only `e2e/` remains, until 12b deletes it.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `secret-handling`: the verification agent keeps the file with its token readable only by its owner, narrows an older file at start, and warns visibly when it cannot.
- `local-development-workflow`: the root lint and format commands cover two modules, the backend with the verification agent and the frontend, and there is no separate agent lint or format task. A commit that touches the agent runs the backend checks.
- `ci-pipeline`: the backend lint job checks the agent's dependencies and code; there is no separate agent lint job.
- `agent-workflow`: an agent changing verification agent code reads `backend/AGENTS.md`, because the agent no longer has its own module doc.

## Out of scope

- The end-to-end suite. Docker compose, the mock server, the fixture generator and the shell scenarios stay in `agent/verification/e2e` until 12b rewrites them as Go tests against the test harness.
- The restore code in `engines` (12c), the backend's `verification` module (12d) and `agent.yaml` with the generated client (12e).
- Any change to the agent protocol, the restore, the heartbeat or the CLI commands and flags.
- Redacting secrets in the agent's log handler. The agent logs no token today; using the redaction of `lib/logging` comes with a later change.
- Moving the agent to the backend's `lib/logging` handler, which rotates differently and names files differently.
- The token passed as a `--token` flag, which other users can see in the process list while `start` runs.
- Lint, formatting and commit checks for the mock server in `agent/verification/e2e/mock-server`. It loses its module with `agent/verification/go.mod`, keeps building through its own Dockerfile, and goes unchecked until 12b deletes it.
- Windows support. The agent builds only for Linux, and the unused `lock_windows.go` is deleted with the move.

## Impact

- **Backend, new:** `backend/cmd/verification-agent/...` and `backend/internal/agent/...` with their tests.
- **Backend, changed:** `backend/go.mod` and `go.sum`. The agent binary and the files `agent:run` creates go to `backend/temp/`, which is already ignored.
- **Deleted:** everything under `agent/verification` except `e2e/`.
- **Tooling:** `mise.toml`, `mise-tasks/agent/run`, `hk.pkl`, `.github/workflows/ci-release.yml`, `Dockerfile`, `agent/verification/e2e/Dockerfile.agent-builder` and `docker-compose.yml`. The root `.gitignore` keeps local agent binaries and the old e2e artifacts ignored.
- **Not affected:** the backend's code, the API, the agent protocol, the agent binary's commands and flags, its log format and the release artifacts users download.
- **Running instances:** an agent of the new release narrows its configuration file to `0600` on its first start, which follows its self-update. Nothing else changes for a running agent.
- **Documents:**
  - the root `AGENTS.md`: the line asking to read `agent/verification/AGENTS.md` and the agent rows of the command table;
  - `docs/architecture.md`, "Long-running work" and "Automatic checks", and the matching line of `backend/AGENTS.md`: the one exception to starting processes through `lib/process`, the agent's daemon, with its reason;
  - `docs/testing.md`, if it names the agent's test command;
  - the rewrite plan `temp/refactor/plan.md`, which lives only in the main checkout: the 12a line, the claim that the agent's lint configuration differs only in `gofumpt.module-path`, and the claim that the agent's unit tests need Docker.
- **Rules the change answers to:** the root `AGENTS.md` and `backend/AGENTS.md`, which already holds the verification agent's rules. `agent/verification/AGENTS.md` adds nothing beyond a pointer to them and is deleted.
