## Why

This is the second change of milestone 12, after `agent-in-backend` moved the verification agent into the backend module. The agent's end-to-end suite still runs against `mock-server`, a 400-line stand-in for the backend with its own untyped copy of the agent protocol. The scenarios restore dumps prepared in advance, not backups the backend took. They therefore cannot catch a mismatch between the agent and the real backend: how the backend hands out a job, decrypts and streams the backup, and accepts and stores the report.

12c and 12d change code on both sides of the agent protocol. Before that, the agent needs a safety net that runs it against the real backend and stays unchanged while they land.

## What Changes

- **The ten shell scenarios become Go tests in `backend/internal/agent/e2e`**, the place `docs/testing.md` gives to agent tests against a real backend. The scenarios keep their coverage:
  - a valid backup restores on PostgreSQL 12 to 18;
  - a TimescaleDB backup on PostgreSQL 17;
  - a backup that includes the `public` schema, on PostgreSQL 18;
  - a broken backup;
  - the per-job disk budget;
  - a retry over a network an earlier attempt left behind;
  - startup removal of leftovers;
  - the privileges of the restore container;
  - `start`, `stop` and `status`;
  - heartbeat and self-update.
- **The agent runs against the real backend from `apptest`.** The test builds the agent binary twice, for the server's version and for an older one, and starts it as a process. Users, the workspace, the storage, the source database and its backup are created through the API. The agent, its token and the manual verification are created through the old routes with `apptest.CallOldAPI`, because the contract does not describe them yet. The agent restores a backup the backend has just taken, so the whole path from backup to report is checked. No verification config is created: a manual verification does not need one.
- **A test gateway sits between the agent and the backend.** It forwards every request to the real backend unchanged, with three exceptions that the real backend cannot produce on demand:
  - it holds the agent's claim requests until the test releases them;
  - it holds the backup stream request until the test releases it;
  - in the disk budget scenario it lowers the disk budget in the claim answer.

  The gateway knows two paths and one JSON field, not the protocol.
- **The backend serves the agent binary from a directory given by the application environment**, not from the path `agent-binaries/...` hard-coded in `features/system/agent/controller.go:18`. `apptest` gets an option that sets the directory, so the self-update scenario downloads a binary the test built with the server's version. The default equals the current path, so the image does not change. The version source does not change: the test reads the version the server already reports.
- **A broken backup is a real backup whose stored bytes the test damages** in the local storage directory through an `apptest` helper, the way corruption on a storage would. It is taken without encryption, so the restore, not the decryption, rejects it.
- **The suite runs on Docker, and a subset of it on Podman.** The subset is a valid backup on PostgreSQL 18, a broken backup and the container privileges, as today. An environment variable points only the agent and the test's own engine checks at Podman; the source databases stay on Docker.
- **Tasks and CI:**
  - `mise run agent:e2e` and `agent:e2e-podman` run the Go tests;
  - `mise run agent:test` no longer runs the e2e package;
  - `mise run pull-testcontainers` also pulls the agent's restore images;
  - the e2e package becomes the `agent-e2e` CI group, a `test-backend` check like the other groups;
  - the `e2e-verification-agent-podman` job runs the Podman subset and no longer waits for the backend lint, like the other backend test checks.
  - **BREAKING (developer tooling):** `mise run agent:e2e-clean` is deleted, because the tests remove what they start. The `e2e-verification-agent` CI job is replaced by the `test-backend (agent-e2e)` check.
- **Deleted:** the whole `agent/` directory, with `mock-server`, docker compose, the Dockerfiles, the shell scenarios, `fixture-gen` and the prepared dumps.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `local-development-workflow`: a developer runs the agent's end-to-end suite against the real backend with one command on Docker and with another on Podman, and the suite cleans up after itself. Test images that can be pulled ahead include the agent's restore images. The lint requirement loses its mention of the old suite outside both modules. The commit-check requirement is replaced by one without the exemption for that suite, because a delta cannot drop one of its scenarios.
- `ci-pipeline`: the agent's end-to-end tests are a backend test group that covers its package once, and a Podman check runs the subset; publishing requires both. The lint requirement loses the old suite's Go code.
- `agent-workflow`: the requirement that module rules reach every agent is replaced by one without the scenario for the old end-to-end suite outside the backend module, because the suite no longer exists there.

## Out of scope

- Changes to the agent protocol, commands and flags, or backend verification logic. The disk-budget scenario exposed a repair needed in the agent: measure `/var/lib/postgresql` rather than the pre-18 data path so PostgreSQL 18 data counts toward the budget.
- The restore code shared through `engines` (12c), the `verification` module (12d) and `agent.yaml` with the generated agent client (12e). In 12d the scenarios move from `CallOldAPI` to the generated client for user operations; nothing else in them changes until the end of the milestone.
- New scenarios. The suite keeps today's coverage; it does not add tests for MySQL, MariaDB or MongoDB verification, which the agent does not support.
- Any change to where the server's version comes from. The version endpoint and the agent download keep reporting `system_version.GetAppVersion()`.
- A setting that operators can change for the agent binary directory. The directory stays where the image puts it.
- Rootless Podman. The spec requires rootful Podman, and the subset keeps running on it.

## Impact

- **Backend, new:** `backend/internal/agent/e2e/` with the scenarios, the gateway and the binary build; `apptest` helpers for creating the agent, enqueuing a verification, reading it, damaging a stored backup file, and the option for the agent binary directory.
- **Backend, changed:** `internal/app` (an environment field for the agent binary directory, routes that pass it on, and access to the wired backups module for the damage helper), `features/system/agent` (the controller takes the directory).
- **Deleted:** `agent/` in full.
- **Tooling:** `mise.toml` (`agent:test`), `mise-tasks/agent/e2e`, `e2e-podman`, `e2e-clean`, `mise-tasks/backend/test-group-packages` and `pull-testcontainers`, `.github/workflows/ci-release.yml`, the root `.gitignore` entries for the old artifacts; `scripts/fedora.sh` and its regression test, so Ubuntu keeps its native libedit library.
- **Agent repair:** count the PostgreSQL 18 data directory in the per-job disk budget, as exposed by the new scenario.
- **Not affected:** the API contract, the agent protocol, image layout and download paths.
- **Running instances:** the agent binary directory keeps its current default. After agents update, the disk-budget probe also counts PostgreSQL 18 data.
- **Documents:** `docs/testing.md` (how the agent's end-to-end suite runs, on Docker and on Podman, and the gateway as the one allowed stand-in on the agent's link); the rewrite plan `temp/refactor/plan.md`, which lives only in the main checkout (the 12b line and section, and the relaxation row for the old e2e). The root `AGENTS.md`, `docs/architecture.md` and review hooks stop naming the deleted top-level agent directory; hook tests check the updated prompts.
- **Rules the change answers to:** the root `AGENTS.md` and `backend/AGENTS.md`.
