## Context

See proposal.md for the motivation. The state that shapes the approach:

- **The old suite controls a fake backend.** `agent/verification/e2e/mock-server/main.go:38-57` serves the agent routes plus `/mock/set-claim`, `set-version`, `set-binary-path`, `set-backup-fixture`, `set-stream-delay` and `reports`. Each script in `scripts/` sets a claim, waits for a report and checks it with `jq`. `fixture-gen/entrypoint.sh` dumps PostgreSQL 12 to 18, a TimescaleDB database and a `-n public` database ahead of the run.
- **The test application is reachable over TCP.** `apptest.New` serves the app through `httptest.NewServer` (`internal/app/apptest/apptest.go:200`), so a separate agent process can call `App.URL`. Only one `apptest` instance may live in a process at a time (`docs/testing.md`, "The test application"), so the scenarios of one package run one after another.
- **The backup side already exists.** `enginetest.Team` creates a database through the contract and takes a backup with a chosen encryption (`internal/app/apptest/enginetest/team.go:89-165`). `containers.StartPostgres` and `StartTimescaleDB` boot the source servers (`internal/util/testing/containers/postgres.go`).
- **The verification routes are still old code.** Creating an agent is `POST /verification/agents`, admin only (`internal/features/verification/agents/controller.go:18-29`). A manual verification is `POST /verifications/enqueue`, which needs no verification config (`runs/service.go:42-117`). `GET /verifications/:id` and `GET /verification/agents` return the stored verification and the agents with `maxRamGb` and `lastSeenAt` (`runs/model.go:9-33`, `agents/model.go:9-22`).
- **A failed report stores only the message.** `RequeueOrFail` writes `fail_message` and the status (`runs/service.go:437-466`). A report with a `pg_restore` exit code or with the disk failure kind is terminal on the first attempt, and a failure without either is requeued up to three attempts (`runs/service.go:591-614`, `runs/constants.go:8`).
- **The backend cannot hand out a small disk budget.** The budget is the archive plus the restored size plus at least 1536 MB (`runs/capacity.go:15-19`, `:48-58`). A test database of a few megabytes never reaches it.
- **The agent waits 15 seconds after an empty claim** (`internal/agent/runner/runner.go:27`). A claim call times out after 30 seconds (`api/constants.go:14`); the backup stream has no timeout (`api/api.go:93`).
- **Background self-update ignores `--skip-update`.** The flag skips only the check at start (`cmd/verification-agent/main.go:174-177`). The background updater runs whenever the version is not `dev` and the agent is not in development mode (`internal/agent/start/start.go:99`), and it checks every 10 seconds (`upgrade/background_upgrader.go:13`). Development mode is a `.env` with `ENV_MODE=development` in the agent's directory or any parent (`cmd/verification-agent/main.go:197`). The update compares versions only for equality (`upgrade/upgrader.go:43-49`).
- **The agent binary path is relative to the working directory.** `system_version.GetAppVersion` reads `APP_VERSION` with a default of `3.26.0` (`internal/features/system/version/version.go:9`). The version endpoint gets it in `internal/app/routes.go:39`, and the download controller sends the same value in its header. The controller serves files from `agent-binaries/databasus-verification-agent-linux-<arch>` relative to the working directory (`internal/features/system/agent/controller.go:18-37`). The image puts the binaries in `./agent-binaries` (`Dockerfile:154-155`).
- **Leftover cleanup filters by labels.** The startup purge removes everything labeled with the agent's ID. The cleanup before an attempt removes what carries both the agent's and the verification's labels (`internal/agent/container/leftovers.go:87-93`, `manager.go:158-164`). The attempt's network is named `databasus-verif-<verification ID>`.
- **The agent restores into `databasus/verification-postgres:<major>`** by default (`internal/agent/config/config.go:22`), or into `timescale/timescaledb:<version>-pg<major>` when the job names a TimescaleDB version.
- **The agent's import rules.** `tools/archlint` checks the imports of production files only (`tools/archlint/layers.go:57-64`). The agent's `depguard` rule covers every file under `internal/agent`, test files included, and denies Gin and GORM (`backend/.golangci.yml:114-122`).
- **A failed verification stores only its message.** For a completed report the backend stores `pgRestoreExitCode` and `dbSizeBytesAfterRestore` with the statistics (`runs/repository.go:139-152`), and `GET /verifications/:id` returns them (`runs/model.go:27-28`). For a failed report `RequeueOrFail` stores only `fail_message` and the status (`runs/service.go:437-466`).
- **The local storage keeps a file under its name in the data folder.** `apptest` already reads that layout through `old_config.GetEnv().DataFolder` (`internal/app/apptest/storage_files.go`). A logical backup's stored file is its dump as `pg_dump` wrote it, or the encrypted dump; the read path only decrypts (`internal/modules/backups/internal/pipeline/backup_file.go:12-31`).
- **Tasks and CI.** `agent:test` runs `./internal/agent/...` (`mise.toml:121`), which would include a new e2e package. `test-group-packages` puts every package outside the named groups into `other` (`mise-tasks/backend/test-group-packages:16-20`). The CI jobs `e2e-verification-agent` and `e2e-verification-agent-podman` run docker compose (`.github/workflows/ci-release.yml:196-252`), and the release job waits for both (`:986`).

The change answers to the root `AGENTS.md` and `backend/AGENTS.md`, and to the testing skill for the tests it writes.

## Goals / Non-Goals

**Goals:**

- Every behavior the ten shell scenarios check is checked again, with the agent process, the backend, the database servers and the container engine all real.
- The scenarios change only when the user operations move to the generated client in 12d. The engine and module rewrites of 12c and 12d run under them unchanged.
- A failure points at the step that broke: each wait names its condition, and a failed scenario prints the agent's output and log.

**Non-Goals:**

- Faster scenarios through parallelism inside the package. The `apptest` limit forbids it until milestone 13.
- Testing the agent's update at start. The old suite checked only the background update, and so does this one.
- Testing more than the shell scenarios did. Agent behavior that only the unit tests of `internal/agent` cover stays there.
- The wire shape of the heartbeat. The old scenario also checked that the heartbeat sends `"currentVerificationIds":[]`. The backend does not expose the field, so the new scenario checks the stored capacity and `lastSeenAt` instead. The field's encoding stays covered by the unit tests of `internal/agent/heartbeat` and the backend's heartbeat controller tests.

## Decisions

### 1. One test-only package, `internal/agent/e2e`

Every file of the package ends in `_test.go`. The package imports `apptest`, `enginetest` and the containers helpers, which the agent's production code may not import. The architecture check judges the imports of production files only, and no binary links a test-only package, so the agent's import rule stays intact. `depguard` does check these test files, so the package imports neither Gin nor GORM directly; it reaches them only through `apptest`. `docs/testing.md:21` already names this path for agent tests against a real backend.

Alternatives:

- `internal/modules/verification/e2e`. Rejected: the module appears only in 12d, and the scenarios would have to move then. Moving them in the same diff as the code they guard defeats the safety net.
- `backend/e2e/agent`, next to the upgrade test. Rejected: that directory holds tests against built images, and `docs/testing.md` already places agent tests under `internal/agent`.

### 2. The agent runs as a built binary, built once per test process

`TestMain` builds the agent twice with `CGO_ENABLED=0 go build -ldflags "-X main.Version=<version>" ./cmd/verification-agent` into a temporary directory: once with the server's version, read through `system_version.GetAppVersion()`, and once with a different, older string. Both binaries are named `databasus-verification-agent-linux-<GOARCH>`, the name the download route serves, and live in separate directories. Each scenario copies the binary it needs into its own `t.TempDir()`, which becomes the agent's working directory with its config, lock file and log.

The scenarios start the agent through `lib/process`, so the process dies with the test's context. The daemon that `start` detaches is the exception: the scenario reads its PID from the output and kills it in `t.Cleanup` if it is still alive. Test files may start the binary under test directly, as the `backend-code-checks` spec allows, so building with `go` uses `os/exec`.

`t.TempDir()` lives under the system temporary directory, outside the repository. The agent therefore never finds the repository's `.env` with `ENV_MODE=development`, which would switch off its updates.

Alternatives:

- Run the agent in process by calling `start.RunDaemon`. Rejected: the daemon, lock and self-update scenarios check a process, a lock file and the replacement of the executable, which exist only for a real binary.
- Build the binaries in the mise task and pass their paths in the environment. Rejected: `go test ./internal/agent/e2e` would then fail on its own. The Go build cache makes the second build of the same commit a link step.

### 3. The agent binary directory comes from the application environment

`app.Environment` gets `AgentBinariesDir`, which the production environment fills with `agent-binaries`, today's path. `routes.go` builds the download controller with it, and the controller stops using its hard-coded paths. `apptest.WithAgentBinariesDir(directory)` sets the field for a test, next to the existing `WithClientToolsDir`. The version stays where it is: the version endpoint and the download header already report `system_version.GetAppVersion()`, and the test builds its "current" binary with the same string.

Agents built with the server's version never update. The self-update scenario starts the "older" binary with `--skip-update`. That skips the check at start, and 10 seconds later the background updater downloads the current binary from the backend, verifies it and re-executes. This is the path the old scenario covered, and it needs no change of the server's version during the test. `TestMain` reads the version in the same process that serves it; the first scenario also compares it with the version endpoint's answer, so a mismatch fails with both values instead of an unexpected update.

Alternatives:

- Move the version into the environment as well and set it through an `apptest` option. Rejected: the test can read the version the server already reports, so the option would change production wiring for nothing.
- An `apptest` method that changes the server's version during the test. Rejected: the background path is reachable without it, and it would put a mutable version into production wiring.
- An operator setting for the binary directory. Rejected: nobody needs to move it, and a setting would join the runtime configuration spec.

### 4. A gateway on the agent's link

The agent calls a gateway, an `httputil.ReverseProxy` in the test package that listens on `127.0.0.1:0` and forwards to `App.URL`. It forwards every request unchanged unless the scenario enabled one of three behaviors:

- **Hold claims.** `POST /api/v1/agent/verifications/{agentId}/claim` waits until the scenario opens the gate, then goes to the backend. The retry scenario needs a moment after the startup purge and before the claim, which the real backend cannot give on demand.
- **Hold the backup stream.** `GET .../{verificationId}/backup-stream` signals the scenario that it arrived and waits for the release. The restore container is running by then, so the privileges scenario reads `/proc/1/status` from it without a race.
- **Cap the disk budget.** In the answer to a claim with status 200, the gateway decodes the JSON into a map, sets `maxContainerDiskMb` to the cap and encodes it again. The disk scenario needs a budget below the size of an empty cluster, and the backend's floor is 1536 MB.

A held request also ends when the test's context ends, so a failed scenario does not hang. The gateway knows no other field and no other path, and never answers in the backend's place.

Alternatives:

- No gateway. For the retry scenario, enqueue and create the network right after the first heartbeat and rely on the 15-second pause after an empty claim. Rejected: the first claim can arrive after the enqueue, and then the scenario either fails on an existing network name or passes without testing anything. For the privileges scenario, catch the container while it starts. Rejected: the result would depend on how long initdb takes.
- An `apptest` option that changes the disk gap of the old verification code. Rejected: a test-only knob in code that 12d rewrites, and the knob would have to survive that rewrite.
- Move the disk scenario to the `runner` package with a fake server and a real engine. Rejected by the user: the scenario would then leave the real backend, and the backend's handling of the disk failure kind would no longer be checked from end to end.

The claim response can be gzip-compressed. The disk-cap branch decodes that encoding before changing the field and returns the modified JSON with matching headers. The other branches keep the upstream body and headers unchanged.

The disk-budget scenario on PostgreSQL 18 exposed an existing defect: the agent measured `/var/lib/postgresql/data`, while PostgreSQL 18 keeps its data under `/var/lib/postgresql/18/docker`. The repair measures `/var/lib/postgresql` and `/tmp`, covering the data directory on every supported major. The scenario failed with `COMPLETED` before the repair and must reach `FAILED` after it.

### 5. Data through the API, verification through the old routes

Every restore scenario does the following:

1. Creates the owner, the workspace and a local storage with `enginetest.NewTeam`.
2. Fills a source database with `t_a` and `t_b`, or with the hypertable for TimescaleDB, under a unique database name on the version's server.
3. Creates the database through the contract and takes a backup with `Team.BackUp`.
4. Creates the agent through `CallOldAPI` and starts it with its ID and token.
5. Enqueues a manual verification through `CallOldAPI`.

The successful scenarios take encrypted backups, so the backend's decryption on the stream path is covered. New `apptest` helpers wrap the agent and verification calls and decode their answers into local structs. In 12d only these helpers move to the generated client.

The scenarios assert what a user sees:

- `GET /verifications/:id` for the status and the attempt count; for a completed verification `pgRestoreExitCode` 0, `dbSizeBytesAfterRestore` above zero, `tableCount`, `schemaCount` and `tableStats`; for a failed one `failMessage`;
- `GET /verification/agents` for `maxRamGb` and `lastSeenAt`;
- the engine for labels, leftovers and `/proc/1/status`.

A failure is checked by its status and its attempt count. A rejected restore or an exceeded budget is `FAILED` after one attempt, while a failure before `pg_restore` would be retried up to three times. A failed verification does not store the exit code of `pg_restore`, so the scenario cannot check it; the attempt count shows that the agent reported one. The disk scenario also checks that `failMessage` names the disk budget, which tells it apart from a restore rejection.

Alternatives:

- Enable an after-backup verification config, as the rewrite plan suggested. Rejected: it adds a request and a wait for nothing, because the manual route returns the verification ID directly. `after_backup_verification_test.go` already covers the automatic path.
- Read the stored rows with SQL. Rejected: the API returns everything the scenarios check.

### 6. A broken backup is a real backup whose bytes are damaged on disk

The scenario takes a backup without encryption into a local storage, then overwrites the stored file in the data folder with bytes that are not a `pg_restore` archive. This is what corruption on the storage looks like: Databasus wrote and published the file, and its bytes changed afterwards. A new `apptest` helper reads the backup's file name through the application instance, which delegates to its wired backups module and writes `filepath.Join(DataFolder, fileName)` with `os.WriteFile`, the layout `IsLocalStorageFilePresent` already relies on. The file store's records stay as the backup left them, so no pending deletion appears and the file stays in place. The read path only decrypts, and an unencrypted file is passed through, so the agent receives the damaged bytes and `pg_restore` rejects them. The scenario then expects `FAILED` after one attempt, the sign of a restore rejection.

Alternatives:

- Damage an encrypted backup. Rejected: the backend's decryption fails first, the agent gets a broken stream and reports a failure without an exit code, and the backend retries it. That is a different scenario.
- Rewrite the file through `App.WriteStorageFile`. Rejected: the write records a pending deletion and the receipt is never spent, so the deletion worker removes the backup's file after the commit window. The agent would get a stream error and the backend would retry.
- Rewrite it through `App.PublishStorageFile`, which writes and confirms. Rejected: it reuses a published name, which the "Stored files" rule in `docs/architecture.md` forbids, and the scenario needs only damaged bytes, not a new publication.

### 7. The scenarios and their grouping

Top-level tests name the agent behavior, and each scenario is a sentence in `t.Run`, as the testing skill asks:

| Test | Scenarios |
|---|---|
| `TestVerifyBackup` | a valid backup on PostgreSQL 12 to 18, one source server per version booted in turn; TimescaleDB on 17; the `public` schema on 18; a broken backup; the disk budget; a retry over a leftover network |
| `TestRestoreContainer` | the privileges of the restore container |
| `TestAgentStartup` | removal of leftovers, keeping another agent's network |
| `TestAgentDaemon` | `start`, `status`, a second instance refused, `stop` |
| `TestAgentUpdate` | the heartbeat with capacity, the background update and the heartbeat after it |

The startup scenario seeds its stray container from the agent's restore image with a long `sleep`, an image the suite pulls anyway, instead of `alpine`.

Each scenario ends with a leftover check: no container or network with its agent's label remains on the agent's engine. The agent's output and log file are printed when the scenario fails.

Alternative: one top-level test per old script. Rejected: the testing skill groups scenarios under the behavior they test, and a version table fits one test.

### 8. The engine: Docker by default, Podman through one variable

`DATABASUS_AGENT_E2E_ENGINE_HOST` holds the address of the agent's engine, empty for the default Docker socket. The scenario passes it to the agent process as `DOCKER_HOST`, and the test builds its own engine client for the leftover and privileges checks from it. The process environment of the test keeps no `DOCKER_HOST`, so testcontainers and the source servers stay on Docker.

`mise run agent:e2e-podman` sets the variable to the rootful socket, `/run/podman/podman.sock` unless `E2E_ENGINE_SOCKET` names another. The task keeps the name of the variable it reads today. It fails before any test when the socket is missing or the user cannot open it. A missing socket also gets the command to start rootful Podman's socket. The library shim returns without changes when the host already provides `libedit.so.2`, so the task works on Ubuntu CI as well as Fedora. The failure names the path and how to make it reachable: start rootful Podman's socket when it is missing, or grant access with `sudo setfacl -m u:$USER:rw <socket>` when it cannot be opened. The task selects the subset with one `-run` expression over the three scenario names.

Alternatives:

- Set `DOCKER_HOST` for the whole test process. Rejected: the source servers and the backend's other test containers would move to Podman, where they have never run, and the subset would test more than the agent's engine.
- Rootless Podman. Rejected: the `verification-container-resources` spec names rootful Podman, and the subset must check that.
- Run the agent in a root container, as the compose runner did, to reach the socket. Rejected: the agent would no longer be a process the test owns, and the daemon scenarios could not run.

### 9. Tasks and CI

- `test-group-packages` gets the group `agent-e2e`, `./internal/agent/e2e/...`, and `other` leaves it out.
- `agent:e2e` runs `mise run backend:test-package` on that package, because the task already applies the Fedora client library shim and the 30-minute timeout. `agent:e2e-clean` is deleted, because each scenario removes what it started.
- `agent:test` lists `./internal/agent/...` and `./cmd/verification-agent/...` without `/e2e`, so it needs no engine and no test database.
- `pull-testcontainers` also pulls `databasus/verification-postgres:12` to `:18` and the TimescaleDB restore image. The e2e package names them as literals the task's pattern finds, and the pattern learns the `databasus/verification-postgres` name.
- In CI, `agent-e2e` joins the `test-backend` matrix on a 4 vCPU runner. That job already starts the shared test fixtures and installs the client libraries. The `e2e-verification-agent` job is deleted, and the release waits for the `test-backend` matrix as before.
- `e2e-verification-agent-podman` stays on `ubuntu-24.04`. It gets the steps a backend test needs: Go, client libraries, `.env` and the fixtures. It grants the runner's user access to the rootful socket and runs `mise run agent:e2e-podman`. It no longer waits for `lint-backend`, like the `test-backend` groups. Its cleanup, which runs whether the tests passed or failed, removes labeled containers and networks from Podman, as it does now.

Alternative: keep `e2e-verification-agent` as its own job running `agent:e2e`. Rejected: it would repeat the setup steps of `test-backend`, and the `ci-pipeline` spec wants every package in exactly one group.

## Risks / Trade-offs

- [The suite runs slower than the compose suite, because every restore scenario first takes a real backup and the scenarios run in sequence] → The estimate is 6 to 10 minutes, measured during apply and recorded in `docs/testing.md`. The limit per package is 30 minutes. If the measurement comes close to it, the restore table splits into a second package under `internal/agent/e2e/`, which joins the group without a CI change.
- [The gateway's disk cap hides a change of the budget field's name] → If the field is renamed, the cap stops working, the restore completes, and the scenario fails on `COMPLETED`. 12e must keep the field name for compatibility with older agents anyway.
- [Replacing the stored file may trip a size or checksum check on the backend's read path, so the agent gets a broken stream instead of bad bytes] → The scenario asserts one attempt, so this shows up as a failure, not as a false pass. The helper then writes content of the original length.
- [Podman access in CI needs `setfacl` on the socket] → The CI step and the task's check name the same command. If the runner lacks ACL support, `chmod 666` on the CI runner's socket is the fallback; locally the task never changes permissions itself.
- [First runs pull many images] → `pull-testcontainers` pulls them ahead. The Docker restore scenarios wait for the outcome up to the test timeout, not a fixed budget.

## Migration Plan

The old suite stays until the Go scenarios pass on Docker and on Podman. It is deleted in the last task together with its CI job. Rollback is a revert of the change; the image and the protocol do not change, so there is nothing to migrate for running instances.
