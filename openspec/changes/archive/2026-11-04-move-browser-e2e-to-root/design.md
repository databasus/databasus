## Context

See `proposal.md` for the motivation. The current state that shapes this design:

- The current suite starts in `frontend/tooling/runBrowser.ts:17`, goes through `go test` in `frontend/tooling/runBrowserE2E.ts:26` and builds the application with `apptest` in `backend/internal/app/browsertests/browser_test.go:48`. A gateway in `backend/internal/app/browsertests/gateway_test.go:21` serves the frontend and hard-codes `IS_EMAIL_CONFIGURED: "true"`.
- `backend/internal/app/apptest` lives under `backend/internal`, so Go refuses to import it from outside the `backend` module.
- `.dockerignore:43` excludes `.env`, and it also excludes `node_modules` and `databasus-data`, so a developer's settings do not reach an image built from the checkout.
- A production frontend calls its own origin, `frontend/src/shared/config/deploymentConfig.ts:18`, so the address `http://databasus:4005` on the compose network works without configuring the application.
- The image publishes `runtime-config.js` at startup and reports email as configured only with `SMTP_HOST`, `docker/start.sh:149`. The sign-in form shows `Forgot password?` only then, `frontend/src/pages/auth/ui/SignInComponent.tsx:215`.
- The wizard starts the first backup itself, `frontend/src/pages/databases/model/startFirstBackup.ts:9`.
- CI already builds and shares a candidate image for `test-upgrade`, `.github/workflows/ci-release.yml:437`.
- The review hook accepts an exact list of verification commands, `.agent-hooks/review_gate.py:237`.

The approach answers to the root `AGENTS.md` rules on several agents sharing one machine, on security and on naming, and to the `frontend/AGENTS.md` rules for what stays of the frontend's tooling. It keeps `docs/architecture.md`'s rule that test-only code never reaches the production binary, and adds the root `e2e/` directory to its layout.

## Goals / Non-Goals

**Goals:**

- The suite tests an application image of the commit, built separately from the published one from the same commit and Dockerfile, and nothing else.
- A crash on any step leaves nothing behind, or leaves something the next run removes, including a run-unique image.

**Non-Goals:**

- A framework for other suites. The upgrade test keeps its own Docker helpers.
- Parallel scenarios inside one run. A run has one instance and few scenarios, and running them in order removes a class of flaky tests.

## Decisions

### Python with the official Playwright binding

The user chose Python for this suite when asked, and asked for a suite independent of the backend and the frontend. It is written with `pytest` and Microsoft's `playwright` package, pinned at `1.64.1`, the version the frontend's fixture suite uses. The cost is rewriting the two existing scenarios and pinning `uv`.

Rejected: Go with `playwright-go`. It is a community binding, and its latest release pins Playwright `1.62.1`, so it would hold back upgrades. Its advantage, sharing Docker helpers with the upgrade test, amounts to a few `docker` calls.

Rejected: keeping TypeScript in the frontend's Playwright project. The suite would stay inside the frontend's package and lockfile, which contradicts the independence this change is for.

Rejected: a standalone TypeScript package in `e2e/browser`. It would move the existing scenarios nearly as they are and add no new package manager, but the user chose Python.

### A supervisor process owns the run, and `pytest` is its child

Scenarios and their fixtures live in `e2e/browser/scenarios` and hold no path into `backend/` or `frontend/`. The supervisor uses the checkout only as the context of an image build.

`e2e/browser/supervisor.py` sweeps leftovers, prepares the image, runs `pytest` as a child and cleans up in `finally`, after turning `SIGTERM` into the same path as `SIGINT`. Scenarios get the browser's address, the application's address on the compose network and credentials through the environment.

Rejected: doing everything in `conftest.py` session fixtures. A collection error or a hard termination skips or truncates fixture teardown, and cleanup is what this suite must guarantee. The start of a run also has to come before `pytest` can report anything useful.

### A per-run compose project holds the containers

`e2e/browser/compose.yaml` runs under the project name `databasus-e2e-<runId>`, with these services:

- `databasus`: a copy of the production compose from `README.md`, with a named volume and its health check;
- `databases`: the server holding `e2e_backed_up` and the empty `e2e_restore_target`;
- `browser`: the Chromium server, on the same compose network.

The browser opens the application at `http://databasus:4005`, and the wizard reaches the databases at `databases:5432`. Only the browser's websocket port is published, on a port Docker chooses, and the supervisor reads it with `docker compose port`. Readiness comes from `docker compose up --wait` and the health check. `docker compose -p <project> down -v --remove-orphans` then removes everything by project.

The page runs on a plain-HTTP origin, as on a plain-HTTP production deployment, so scenarios do not rely on the clipboard API.

Rejected: running the browser on the host network and publishing the application on a free host port. The suite would then choose ports and retry on a collision, with nothing on the host needing the application.

Rejected: the `databasus-local` service of `docker-compose.yml:9`. Its container name and port are fixed, so two runs collide, and it bind-mounts `./databasus-data` into the checkout.

Rejected: the Docker helpers of `backend/e2e/upgrade`. They are Go test files in another module, and this suite would have to reimplement them anyway. Compose gives the same result with less code.

Rejected: `testcontainers`. Its cleanup depends on a reaper the repository disables for parallel runs, as the Commands section of `docs/testing.md` says.

### Every run tests the image, and no local mode exists

`mise run e2e:browser` has no option. It uses `DATABASUS_IMAGE` when set and otherwise builds the `Dockerfile` under the name `databasus-e2e-<runId>:candidate`. The layer cache shortens repeated builds, and the name is removed after the run while the cache stays.

Rejected: a second mode that builds the backend and the frontend from the checkout and runs them on the host. It would avoid the image build, but it needs a configurable server port, a temporary directory that imitates the image's layout, its own internal database and a `libedit` shim, and CI would not run it. The user chose to run on the image only.

Rejected: running on top of `mise run run`. It seeds an account, which rules out the first registration, and it is one shared instance, which breaks isolation between worktrees.

Rejected: keeping `apptest` as a local mode. It cannot be imported from `e2e/`, and it would keep a second, test-only way of assembling the application.

Rejected: the fixed `databasus-candidate:local` tag of `mise-tasks/backend/test-upgrade`. Two worktrees would overwrite each other's image during a run.

### Settings come from `mise` and the supervisor, with no `.env`

The compose file sets what never changes: `IS_DISABLE_ANONYMOUS_TELEMETRY=true` on the application, and the `e2e:browser` task's `env` holds the suite's own fixed settings. The `e2e:browser` task also sets `PYTHONDONTWRITEBYTECODE=1`, and `pytest` runs with `-p no:cacheprovider`, so a run writes no cache into the checkout beside `.venv`. The supervisor generates what is unique to a run: its project name and passwords. Docker chooses the published port.

Rejected: writing a `.env` into the run. A file holding passwords is one more thing to protect and to clean up, and it adds nothing to process variables.

### Row comparison runs after the browser, through `psql` in the database container

`docker exec` on the run's `databases` container runs `psql`, which reads `e2e_backed_up` and `e2e_restore_target`, and the test compares the output. A unit test covers the comparison's difference report, because proving it through the whole stack adds nothing.

Rejected: a Python PostgreSQL driver, a dependency that a single query does not justify. Rejected: passing on a `COMPLETED` status, because a restore that reports success without the data would stay green.

### The browser runs in a pinned container

`pytest` runs on the host and connects over a websocket to a Playwright server in `mcr.microsoft.com/playwright:v1.64.1-noble`, pinned by digest, the image the fixture suite already uses. That image holds the browsers but no driver, and neither does its Python variant, so the supervisor mounts the driver of the locked Python package read-only and runs its `run-server`. Client and server are therefore the same version by construction, and nothing is downloaded at run time. The two pins carry a comment naming each other, because a browser image of another version fails on a missing browser executable.

Rejected: `npx playwright@1.64.1 run-server`, which downloads at run time without a hash check. Rejected: a locally built image with the package installed, which adds a build step to every run.

Rejected: `playwright install` on the host. Playwright does not support Fedora, so browser libraries would differ between developers and CI.

Rejected: running `pytest` inside that container too. It would then need the Docker socket for its row comparison and cleanup.

### Leftovers are found by a dead supervisor

The supervisor writes `supervisor.json` with its PID, its process start time and the run-unique image name into `~/.cache/databasus-e2e/<runId>/` before it starts anything. The parent is fixed rather than under `TMPDIR`, which the repository lets a run change, so every run on a machine finds every other run's file. The sweep lists these directories and the `databasus-e2e-*` compose projects, and it removes those of a dead supervisor: the project with its networks and volumes, the run-unique image and the directory. It removes only the image name it finds in `supervisor.json`, which a run records only when it builds that image itself. The start time guards against a reused PID.

A compose project with no such directory is a crash in the short window of the directory's removal, so it counts as dead. The directory is written before the project exists, so a live run always has one.

An exited process awaiting reaping also counts as dead. Its `/proc` entry still holds its start time, so the liveness check rejects zombie and dead states before comparing that time.

Failure to remove a crashed run's project or built image stops startup and names the remaining resource. Its supervisor record stays for later recovery. Standalone pytest processes own recorded temporary roots, so zero-retention cleanup cannot delete another process's temporary files and a later run can recover a killed process's directory. The supervisor gives its scenario child an explicit temporary root inside the supervisor's directory. It removes that root after the child terminates, even when a signal bypasses pytest's cleanup callbacks.

`mise run e2e:lifecycle` writes the same file for the image it builds for its probes, so the sweep also recovers that image after the command is killed.

Rejected: storing the PID and start time as compose labels. A run-unique image and the time before the first container would need labels of their own.

Rejected: removing every `databasus-e2e-*` project at the start, which would kill runs from other worktrees. Rejected: an age limit, which leaves a crash's containers running for that long and can kill a slow live run.

### The first registration is a session fixture on an empty instance

A session fixture registers the instance admin through the interface, so every scenario can depend on it in any order. Scenarios use unique workspace names as a safeguard, though no other run shares the instance.

Rejected: a test that must run first. An order dependency makes a single scenario impossible to repeat on its own.

The sign-in scenario now asserts that `Forgot password?` is hidden, because an instance without `SMTP_HOST` correctly hides it. The frontend's screen tests keep covering the visible button.

### The databases are named by their role

`docs/glossary.md` holds one term per concept and already defines "Restore target", so the database a restore writes into is `e2e_restore_target`, and the other is `e2e_backed_up`.

### CI gives the new check the candidate image and no fallback

`test-e2e-browser` has `needs: [build-storage-image]`, keeps the old job's `if`, so it skips `develop`, downloads and loads that artifact like `test-upgrade`, and runs `mise run e2e:unit`, then `mise run e2e:browser` and `mise run e2e:lifecycle` with `DATABASUS_IMAGE` set. The supervisor refuses to run in CI without `DATABASUS_IMAGE`, so a broken job cannot silently test something else. The check needs no `backend-db-test`.

The old job uploaded `status.json`. With traces, video and screenshots off, it held only the run ID, the exit status and an interrupted flag, which the job's result and log already report, so the upload and the file are not carried over.

### The change verification of this change gains the new command with the hook edit

The review hook accepts `mise run e2e:browser` only after this change edits it, and it validates `tasks.md` when reviewing this proposal. So the tasks that build the suite run that command themselves, and task 7.2 adds it to the section in the same step that teaches the hook to accept it. The section never names `mise run test-full`, which the hook rejects.

### The routine run stays short and `mise run test-full` runs the Docker suites

`mise run test` selects no package under `internal/agent/e2e` or `internal/modules/backups/e2e` and starts no upgrade test, so it answers in about five minutes. `mise run test-full` adds them, script and review hook tests, frontend browser UI, and the browser suite with its unit, lifecycle and interruption checks. Podman and filesystem tests remain separate because they require additional host setup. The other packages that start containers stay in the routine run, because they are the only local check of engines and storages against real servers.

`scripts/test.sh --full` waits for the backend and frontend packages to pass, then runs script tests, review hook tests, browser e2e unit tests, frontend browser UI, the upgrade test, browser end-to-end scenarios, lifecycle checks and lifecycle interruption checks, in that order. A failure skips every later stage. Image and browser suites run sequentially because they disturb each other's timing under load. CI calls the hidden `mise run e2e:browser` on its candidate image, because `mise run test-full` would repeat the checks CI already runs in other jobs.

Rejected: leaving the backup end-to-end packages in the routine run. They add little to the time, but they were named as slow by the user, and a break in the backup pipeline still reaches `mise run test-full` and CI. The cost is that such a break can pass `mise run test`.

### User commands precede CI/CD commands and dependencies

`mise.toml` lists `run`, `test`, `test-full`, `lint`, `format`, `doctor`, `review` and `generate-api` first, hidden CI/CD tasks next, and tools and environment settings last. File tasks keep the multi-line CI helpers out of that configuration. The old `dev` and `full` names have no aliases. Redundant pull, generation and agent wrappers are removed.

The frontend exposes `dev`, `build`, `lint`, `format`, `typecheck`, `test`, `test:browser` and `generate:api`. Its lint command checks formatting, source rules, architecture, styles, types and generated API types. Commit hooks use the same module checks without a second formatting pass.

### One wait in a backend test has a limit of two minutes

`testwait.Until` used to wait up to the deadline of the whole package, so one hung condition took 28 minutes of a run. It now stops after two minutes. In a measured run of every backend test on a loaded machine, load average about 40, the longest ordinary wait took 91 seconds. The one wait above the limit, 179 seconds, belongs to `TestIdlePhysicalWalSource`, which exists to observe 150 seconds of samples and states its own limit through `testwait.Within`. The same measurement put the slowest package at 7 minutes, so the local package timeout goes from 30 to 15 minutes. CI keeps its limits, 40 minutes for backend groups and 30 for the upgrade test in `mise-tasks/backend/test-upgrade`, because it runs with the race detector and the 15 minutes were not measured there. Container start-up waits in `internal/lib/testcontainers` and those of the upgrade test stay outside this limit: they allow up to five minutes and were not measured against it. An upgraded instance also runs its migrations during that wait.

Rejected: a limit on one whole test. Go has no such setting. A check over `go test -json` could report tests above two minutes, but it would change the output of every run for a report nobody asked to act on.

### Agents do not start the full run on their own

An agent runs `mise run test-full` only on the user's direct request. At the end of a change it runs `mise run test` and, by hand, the specific long or end-to-end tests the change touches, so the hook accepts `mise run e2e:browser` and rejects `mise run test-full`. CI's existing application suites still gate the release; review hook tests and lifecycle interruption checks remain local.

### Lifecycle probes run through the public command

`e2e/browser/lifecycle/` starts the public command as a subprocess. The secrets probe fails after sign-in and writes the generated password and a synthetic bearer token into the `pytest` output and, by writing to the main process's output in the `databases` container, into a container log. It also puts both values into the application's log, through the `databasus` container's main process output. A failed run prints the end of every container's log through the same masking, so a stream that bypasses masking fails the check. It selects a probe with `E2E_PROBE`, which the suite reads, and then checks the exit status, the diagnostic and that nothing remains. This is the same technique as `frontend/tooling/runBrowserE2E.test.ts`, with less code because compose owns the containers.

Without `DATABASUS_IMAGE`, the command builds one run-unique image for its probes and removes it after every outcome. With `CI` set and no `DATABASUS_IMAGE`, it refuses, so a broken CI job builds nothing. Three probes are skipped when `DATABASUS_IMAGE` names an image: a run that builds its own image, a killed run whose built image the next run removes, and the fresh checkout. Only these need a run to own an image or a clean tree.

The fresh-checkout probe copies tracked and untracked files into a temporary directory rather than touching the developer's `.env`. It compares the file list there before and after a run, ignored files included, because other agents may be writing to the real checkout.

Before deliberately killing a run or blocking its cleanup, a probe registers an independent finalizer for that run's exact project, owned image and directory. The finalizer still runs if the recovery command or an assertion fails; it retains the supervisor record when resource removal fails. The fixture that drives lifecycle-command interruption saves its own record before yielding, then removes the project, owned image, unrelated container and directory during teardown independently of recovery. A subprocess regression forces recovery to fail after SIGKILL and verifies that teardown leaves no resources.

The unrelated service is a small container the command starts itself, outside any run's compose project, and queries after each probe. The shared test PostgreSQL server and the development services may not exist on a CI runner, so they cannot serve as this check.

The pure parts, the row comparison, output masking, the sweep's decision, the lifecycle command's leftover check, the independence scan and the check of published ports, run without a Docker daemon through `mise run e2e:unit`. The test that interrupts and kills the lifecycle command itself needs a full build, so it lives in `e2e/browser/lifecycle_interruption/` and runs in `test-full` or directly when the lifecycle command changes.

Rejected: dropping these probes. A regression in cleanup is otherwise invisible until a machine fills with containers.

## Risks / Trade-offs

- Every local run pays for an image build, which is slow after a code change → The layer cache shortens it, `DATABASUS_IMAGE` skips it, and `mise run test` stays the routine check.
- A session token could reach printed output through a failure message → Scenarios read the API inside the page, so the token stays out of Python. Output also passes through bearer token masking, and `pytest` runs with short tracebacks and no local values.
- `mise run e2e:browser` no longer has the contract's types, so renaming a button or an API field shows up only at run time → The scenarios stay few, and the check gates every release.
- The suite tests a candidate rather than the published bytes → The commit and the Dockerfile are the same; only `APP_VERSION` differs. Changing this means reworking the release flow, which is out of scope.
- `run-server` and Python on 3.14 are not yet proven together → The first implementation tasks run the path on the image before anything is deleted.
- Lifecycle probes run the whole stack, so their limits must allow for it → Limits are set by measurement, per the testing rules.

## Migration Plan

The new suite is built and passing before the old one is deleted, inside this one change. Nothing is deployed, and the application does not change. Rollback is reverting the commit.
