## Context

See proposal.md for the motivation. The state that shapes the approach:

- **The backend is already prepared for the agent.**
  - `docs/architecture.md` ("Backend layout") and `backend/AGENTS.md` ("Verification agent") describe the target location and rules.
  - `tools/archlint` restricts the agent's imports (`backend/tools/archlint/layers.go:24`), and `layers_test.go:52` pins that rule on a fixture.
  - `./cmd/verification-agent` is already an entry point of the production-binary check (`archlint.go:18`), which skips it while the directory does not exist (`production_dependencies.go:39`).
  - `backend/.golangci.yml` holds the `depguard` rule `verification-agent` and lists `cmd/verification-agent` and `internal/agent` in the target layout.
  - The exceptions for printing, exit and `context.Background()` already cover `^(cmd|tools)/` and `^(cmd|internal/app)/`, so they apply to `cmd/verification-agent` without a change.
- **The agent's lint is weaker than the backend's.** `agent/verification/.golangci.yml` enables `default: standard` plus nine linters and sets `tests: false`. A trial run of the backend's configuration, with its exclusions removed, over the agent's code and tests reported 149 findings: 70 `sloglint`, 38 `forbidigo`, 17 `testifylint`, 10 `revive`, 5 `funcorder`, 4 `usetesting`, 3 `errcheck`, 1 `noctx` and 1 `modernize`. The architecture check adds four bare `go` statements (`start/start.go:67`, `start/start.go:101`, `container/dockerengine.go:248`, `runner/runner.go:218`) and one sentinel error outside `errors.go` (`runner/runner.go:41`).
- **The daemon outlives its parent.** `start` starts its own executable with `_run`, `Setsid` and `exec.CommandContext(context.Background(), ...)`, then exits (`start/daemon.go:97-112`). `lib/process` starts every tool with `Setpgid` and `Pdeathsig: SIGKILL` (`backend/internal/lib/process/process.go:53`), which kills the child when the parent thread exits.
- **The logger is created at import time.** `logger.GetLogger()` builds a package-level logger once through `sync.OnceFunc` (`internal/logger/logger.go:67-69`), and `config.go:15` calls it in a package-level `var`. The handler drops the level attribute (`logger.go:85-87`), so a warning looks like any other line.
- **The configuration file is written with `0644`** (`config/config.go:91`). `os.WriteFile` keeps the mode of an existing file, so no rewrite narrows it. `start` and `run` load and then save it (`cmd/main.go:53-55`, `cmd/main.go:80-82`); the daemon's `_run` only loads it (`cmd/main.go:111`).
- **The agent's unit tests need no Docker.** Every container test uses a fake engine (`container/fake_engine_test.go`, `container/purge_test.go`). The rewrite plan says they need Docker; that is wrong.
- **169 test functions** exist, 145 of them named `Test_What_When_Expect`. Eight `time.Sleep` and five `require.Eventually` calls sit in tests and production code: `start/daemon.go:49,112`, `api/idle_timeout_reader_test.go`, `runner/diskwatch_test.go`, `runner/abort_midrestore_integration_test.go`, `runner/concurrent_restores_integration_test.go` and `runner/report_giveup_integration_test.go`.
- **Dependency versions differ.** The agent uses `moby/moby/api v1.54.2`, `moby/moby/client v0.4.1` and `pgx v5.9.2`; the backend uses `v1.55.0`, `v0.5.0` and `v5.10.0` (`backend/go.mod:16,22,23`). `client` is below 1.0, so its API may differ between minor versions.

The change answers to the root `AGENTS.md` and `backend/AGENTS.md`.

## Goals / Non-Goals

**Goals:**

- One Go module, one lint configuration and one lint task for the server and the agent, with no exclusion for the agent's paths.
- The agent binary keeps every command, flag, output line, exit code, file name and protocol call it has.
- The move and the lint fixes are reviewable as behavior-preserving: logic changes only where a rule demands it, and the agent's tests stay green at each step.

**Non-Goals:**

- Package-level durations that tests shrink: `verifyBinaryTimeout` (`upgrade/upgrader.go:19`), `diskWatchInterval` (`runner/diskwatch.go:10`) and `reportRetryBudget` (`api/constants.go:36`). They are settings, not dependencies, so the rule against package-level dependencies does not cover them. 12c and 12e rewrite the code they sit in. The two function seams beside them are dependencies and move in this change, as decision 12 says.
- `resty`, which 12e replaces.
- Logging through `lib/logging`, and redaction in the agent's handler.

## Decisions

### 1. Package layout

`agent/verification/cmd` becomes `backend/cmd/verification-agent`. `internal/config` becomes `internal/agent/config`, `internal/logger` becomes `internal/agent/logging`, the name the rewrite plan gives it, and each `internal/features/<name>` becomes `internal/agent/<name>`. The import path changes from `databasus-verification-agent/...` to `databasus-backend/...`. `images/postgres` moves to `backend/cmd/verification-agent/images/postgres`, next to the binary that names the image it builds.

`internal/testutil` is deleted. `DiscardLogger` becomes `slog.New(slog.DiscardHandler)` at each call site, and `CaptureLogger` moves into a test file of the one package whose tests capture logs, or into each such package if there are several.

Alternatives:

- Keep `features` as a level, as in `internal/agent/features/<name>`. Rejected: the target layout in `docs/architecture.md` has no such level, and the backend is dropping `features`.
- Move `testutil` into `lib`. Rejected: one helper is a one-liner over the standard library, and the other is used by a few tests. A `lib` package for them would be a `utils` package by another name.
- Put the image under `backend/images` or `assets/`. Rejected: the root `.dockerignore` excludes `backend/images`, and `assets/` holds vendored client tools.

### 2. Dependencies take the backend's versions

The agent's direct dependencies join `backend/go.mod` at the backend's versions. The first task builds and tests the moved agent before any lint fix, so a break from `moby client v0.5.0` shows in its own diff. If v0.5.0 changed an API the agent calls, the agent adapts to it. Raising the backend's own `moby` and `pgx` usage to meet the agent is not considered.

Alternative: keep the agent's versions through a `replace` or a second module. Rejected: one module has one version of each dependency, which is the reason for the move.

### 3. The daemon keeps `os/exec`; everything else goes through `lib/process`

The daemon start stays on `os/exec` with `Setsid`, under one `//nolint:forbidigo` that names the reason: the daemon must outlive the command that starts it, and `lib/process` exists to prevent exactly that. It takes `context.WithoutCancel` of the caller's context instead of `context.Background()`. `exec.Cmd` kills the process when its context ends, even after `Start`, and a Ctrl-C during the 500 ms startup check must not kill a daemon that has already detached.

The check of a downloaded binary (`upgrade/upgrader.go:103`) runs through `process.Run` with stdout written to a buffer. `process.CombinedOutput` does not fit, because the check compares only the version the binary prints on stdout.

The re-exec after an upgrade (`cmd/reexec.go:20`) passes `os.Environ()` to `syscall.Exec`. It is not reading a setting, so it gets one `//nolint:forbidigo` with that reason instead of a `lib/config` call.

This is a deliberate exception to the rule that production code starts external processes only through `lib/process`. "Long-running work" and the "Automatic checks" row in `docs/architecture.md`, and the line in `backend/AGENTS.md` that states the rule, gain this one exception and its reason.

Alternatives:

- Add `process.StartDetached` to `lib/process`. Rejected: the package's contract is that a tool dies with its context, and a function that breaks that contract for one caller makes the package promise two opposite things.
- Exclude `internal/agent/start` from `forbidigo` in `backend/.golangci.yml`. Rejected: a path exclusion hides every future `os/exec` in that package, while a `nolint` covers one line and states its reason.

### 4. Waits in the daemon become polls with a deadline

`Stop` polls the process every 500 ms for up to 30 seconds, and `spawnDaemon` sleeps 500 ms before checking that the child is alive (`start/daemon.go:42-49,112`). Both become a loop over a `time.Ticker` that ends on the condition, the deadline or the context, in a private helper of `start`. `Stop` keeps its 30 seconds and its error. `spawnDaemon` checks the child for 500 ms and fails as soon as it is gone; a child still alive after 500 ms counts as started, as today.

Alternatives:

- `testwait.Until`. Rejected: it is a test library and takes a `testing.TB`.
- A polling helper in `lib/clock`. Rejected: `lib/clock` only reads the time today, and two callers in one package do not justify a shared helper.

### 5. Goroutines start through `lib/background`

The lock watcher and the background upgrader start in a `background.Group` that `start` creates, and the agent waits for the group before it exits. The disk watcher of each job starts in the same group, passed to the runner through its constructor. The stdin copy in `container/dockerengine.go:248` runs in a `background.Group` local to that exec call, and the call waits for the group before it returns. `background.Group` logs a returned error. The copy's error is still reported to the caller through the existing channel, so its behavior stays the same.

The heartbeater and the verification runner, which `start.go` starts with `sync.WaitGroup.Go`, move into the same group. The architecture check does not see `WaitGroup.Go`, but a panic in either would still take the agent down unlogged, and the group logs it with its stack.

Verification jobs also start through this group. The pool retains a separate wait counter so the runner can drain only its jobs without waiting on itself in the shared group. A job panic becomes a `background.PanicError`, cancels the runner's context and stops further claims. The runner drains all in-flight jobs and returns the error; the daemon cancels its remaining services, waits for the shared group and exits with that error. Container cleanup runs before exit, and any container it cannot remove is purged at the next startup.

Alternative: keep the goroutines and add `nolint`. Rejected: the architecture check has no suppression, and the rule exists so that a panic in a goroutine does not take the process down unlogged.

### 6. The logger is built in `cmd/verification-agent`

`internal/agent/logging` exports a constructor that returns the logger and its rotating file. `main` calls it once for the commands that log (`start`, `run`, `_run`, `stop`, `status`) and passes the logger down. `version` and the usage text build no logger, so they create no log file. `config` takes the logger as a parameter instead of a package-level `var`. The handler, the format, the file names and the 5 MB rotation stay as they are.

Log calls move to the `...Context` methods with constant messages and `snake_case` attributes, as `sloglint` requires. Messages keep their meaning; their wording becomes lowercase constant phrases, so a log line reads differently than before. The shell upgrade scenario checks two log messages; its assertions follow their lowercase form. The agent protocol and command output stay unchanged.

Alternatives:

- Use `lib/logging`. Rejected for this change: it rotates through lumberjack under other file names and takes the backend's configuration, so users would find their logs in another file.
- Keep the global and only remove the import-time call. Rejected: the architecture document forbids a package-level variable that holds a dependency.

### 7. The token file is narrowed at load and written private

`config` gets one function that ensures the file's mode before it is read: it reads the mode, does nothing when the file does not exist or when no group or other permission bit is set, and otherwise calls `os.Chmod(path, 0o600)`. The trigger is the same one the spec uses: a file with mode `0400` or `0600` is left alone, and `0640` or `0644` is narrowed. A failed `os.Chmod` returns a result, not an error, that carries the path, the old mode, the owner's user name (the numeric ID when the name cannot be resolved), the OS error and the user the agent runs as. The function takes the chmod call as a parameter, so a test can fail it without a file that belongs to another user. Saving writes with `0o600`. It needs no narrowing of its own: `start` and `run` load the file, which narrows it, before they save it (`cmd/main.go:53-55,80-82`), and `os.WriteFile` keeps the mode of an existing file.

`cmd/verification-agent` turns a failed result into the warning the spec describes. It prints the warning to standard error, with a first line that starts with `WARNING:` because the agent's handler shows no level. It also writes one log record whose constant message starts with `warning:` (`warning: agent config file is readable by other users`), with the path, mode, owner, OS error and applicable commands as attributes, so the log copy carries the mark too. It compares the numeric owner and agent UIDs, and includes `chown_command` only when the owner UID is known and differs from the agent UID. The text names the path, the mode in octal, the owner, the OS error, that other users can read the agent token, and two commands: `chmod 600 <path>`, and `sudo chown <agent user> <path>` when the owner is another user. It runs at every start of `start`, `run` and `_run`.

Each output gets the warning once. For `start` and `run`, standard error is the user's terminal, and the log record goes to the terminal through the handler's stdout and to `databasus-verification.log`. For `_run`, standard error is `databasus-verification.daemon.log` (`start/daemon.go:84,99`), a different file from the agent log, and the daemon's stdout is not connected, so the block lands in the daemon log and the record in the agent log.

Only the configuration file changes. The lock file and the log files stay `0644`, because they hold no secret.

Alternatives:

- Refuse to start when the mode cannot be narrowed. Rejected: agents update themselves, so a refusal would stop verification after an automatic update on a machine where nobody changed anything.
- Write the file through a temporary file and a rename. Rejected: the rename would also change the owner and the inode of a file that another user or a tool manages, which is a larger behavior change than narrowing the mode.
- Warn only in the log. Rejected: the user asked to see the problem, and the agent log does not mark levels.

### 8. Tasks: `agent:lint` and `agent:format` go, `agent:test` stays

`backend:lint` and `backend:format` already run over `./cmd/...` and `./internal/...` (`mise.toml:76-91`), which include the agent after the move. A separate `agent:lint` would run the same linters twice in `mise run lint`, and the commit hook would run them twice for an agent change. Both agent tasks are deleted, and the `agent-format` and `agent-lint` steps of `hk.pkl` with them.

`agent:test` stays and runs `go test -race -count=1 -failfast ./internal/agent/... ./cmd/verification-agent/...` in `backend/`. The review hook accepts it as a verification command (`.agent-hooks/review_gate.py:188`), and the tasks rule of `openspec/config.yaml` names it, so keeping it leaves both unchanged.

`agent:build` defaults the version to `dev` when `VERSION` is unset, so a local run does not start the production self-updater. It writes the binary to `backend/temp/verification-agent/`, which the backend's `.gitignore` and the root `.dockerignore` already exclude through `temp/` and `backend/temp`. `agent:run` builds into the same directory and starts the agent there, so the configuration, lock and log files it creates stay out of the source tree. It takes `AGENT_ID` and `AGENT_TOKEN` from the environment, sourcing the root `.env` first when it exists, and fails with a message naming the missing variable when either is empty. The backend address comes from `VERIFICATION_AGENT_DATABASUS_URL` and defaults to the dev backend, `http://localhost:4005`. `DATABASUS_URL` is not reused, because in `.env` it is the domain for email links. The task's description names the three variables. The root `.env.example` does not change.

Alternatives:

- Keep `agent:lint` as a filtered `golangci-lint run ./cmd/verification-agent/... ./internal/agent/...`, as the rewrite plan says. Rejected: it duplicates part of `backend:lint` and would need its own spec requirement for no check the backend lint lacks.
- Replace `agent:test` with `backend:test-package ./internal/agent/...`. Rejected: it would change the review hook and its tests for no gain.
- Keep a separate `.env` for `agent:run` under `backend/`. Rejected: the repository already has one `.env` per worktree.
- Add the two keys to the root `.env.example`. Rejected: the repository rules ask for the user's consent before `.env*` files change, and the task works without it.

The mock server in `agent/verification/e2e/mock-server/main.go` is Go code with no module once `agent/verification/go.mod` is deleted; its Dockerfile builds it on its own (`e2e/Dockerfile.mock-server`). No lint, format or commit check covers it until 12b deletes it. It is a frozen test fixture, and giving it a module or lint task for one change would be work that 12b throws away.

### 9. CI and `Dockerfile`

The `lint-verification-agent` job is deleted. `lint-backend` already verifies module checksums and runs `govulncheck` (`.github/workflows/ci-release.yml:56,64`). The jobs that needed the deleted job (`ci-release.yml:209,239,265,875,1004`) need `lint-backend` instead, unless they already do. `test-verification-agent` runs `mise run agent:test` in place of its own `go test` line, so CI and developers run the same command. The image scan (`ci-release.yml:133`) and the two image builds (`ci-release.yml:1016,1312`) take the new image path.

The `Dockerfile`'s `verification-agent-build` stage copies `backend/go.mod` and `backend/go.sum`, downloads modules, copies `backend/` and builds `./cmd/verification-agent` for both architectures with the same `-ldflags`. It stays a separate stage from the server's build, so the two builds keep running in parallel, but it uses the same base image and module cache mount.

`agent/verification/e2e/Dockerfile.agent-builder` builds from the backend module: `docker-compose.yml` sets its context to `../../../backend`, and it builds `./cmd/verification-agent`. The shell upgrade assertions follow the lowercase log messages required by backend lint. That scenario resets mock state and always stops its agent, including when an assertion fails, so a failure cannot consume later scenarios' jobs. The scenarios and protocol stay unchanged.

Alternative: build the agent in the server's stage. Rejected: the stages build in parallel today, and merging them makes the image build slower for no gain.

### 10. Printing and exit codes stay in `cmd/verification-agent`

`start.Status` prints the agent's state with `fmt.Println` (`start/daemon.go:58-73`). It becomes `start.GetDaemonStatus`, which returns the state, and `cmd/verification-agent` prints the same three lines. `LoadFromJSONAndArgs` calls `os.Exit(1)` when parsing fails (`config/config.go:71-73`), which never happens, because its flag sets use `flag.ExitOnError` and `Parse` exits on its own. It returns the error instead, and `cmd` exits with code 1 as before.

Alternative: allow `fmt.Print*` in `internal/agent/start`. Rejected: the exception for commands and tools exists so that only entry points talk to the console, and the agent has one.

### 11. Test names and waits

Test functions take the name of what they test, and each scenario becomes a sentence in `t.Run`, as the testing skill says. Tests that differ only in their inputs become a table. Assertions stay as they are, except where `testifylint` asks for another assertion of the same meaning.

In tests, `time.Sleep` and `require.Eventually` become `testwait.Until` on the condition the test waited for. A sleep that gave a goroutine time to *not* do something, as in `idle_timeout_reader_test.go:87,110` and `diskwatch_test.go:67,83`, becomes a check after a step the test controls, such as a fake clock or a channel that the fake closes. A sleep inside the program text that a test compiles (`upgrade/upgrader_test.go:51`) is part of a fixture, not a wait, and stays.

Alternative: leave test names to 12b. Rejected: 12b replaces the shell scenarios, not these unit tests, and the lint already applies to test files once they are in the target layout.

### 12. Function seams become fields

`timeAfterFn` (`api/constants.go:35`) and `backupDownloadBackoffFn` (`runner/runner.go:44`) are package-level functions that tests overwrite (`api/api_test.go:25-33`, `runner/runner_test.go:105-107,142`). The first decides when a report retry fires and the second how long a download retry waits, so both are dependencies. Each becomes an unexported field of the API client and of the runner, set to `time.After` and to `backupDownloadBackoff` by their constructors. Tests in the same package set the field on the instance they build, so they no longer change state other tests see.

Alternative: record them as a temporary exception in `docs/architecture.md` until 12c and 12e. Rejected: moving them is a few lines per seam, smaller than the exception's text.

## Risks / Trade-offs

- [The daemon start, the stop and the self-update are the fragile parts of the move, and only the shell scenarios `test-start-stop-status` and `test-heartbeat-and-upgrade` check them until 12b] → The implementation runs `mise run agent:e2e` after the lint fixes, as a check outside the task list. The unit tests of `start` and `upgrade` keep covering the lock, the PID file and the binary check.
- [`moby client v0.5.0` may change an API the agent calls] → Task 1 moves the code and builds it before any lint fix, so a compile error or a test failure points at the version change alone.
- [The lint fixes touch almost every file] → Each task covers a group of packages and ends with their tests and `backend:lint`, so a regression shows in the task that caused it.
- [Rewording log messages changes what users see in the agent log] → Messages keep their meaning; no program parses them.
- [The warning prints the owner's user name and the agent's user name] → Both are local account names that the user can already see on the machine; the token never appears.

## Migration Plan

An agent of the new release narrows its configuration file on its first start, which follows its self-update. Nothing else changes for a running agent: the protocol, the binary's name and the download paths stay the same. Rolling back to the previous release leaves the file at `0600`, which the previous release reads and writes without trouble.
