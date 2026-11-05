## Context

See `proposal.md` for the problem and agreed scope.

- The critical path currently creates its workspace and storage and finishes after checking a logical restore (`e2e/browser/scenarios/test_critical_path.py:93`). Account registration is already shared across the suite (`e2e/browser/scenarios/conftest.py:38`).
- Compose starts one application image, a pinned PostgreSQL 18 source and a browser. The application is addressed as `http://databasus:4005`; only the browser publishes a host port (`e2e/browser/compose.yaml:1`, `e2e/browser/supervisor.py:31`).
- The restore dialog selects Script or Manual and Host or Docker, and shows generated commands as page text (`frontend/src/pages/databases/ui/physical-backups/PhysicalRestoreComponent.tsx:34`, `frontend/src/pages/databases/ui/physical-backups/CopyableCommand.tsx:47`). Its token is single-use (`backend/api/openapi.yaml:2242`); switching environment tabs does not request another token (`frontend/src/pages/databases/ui/physical-backups/usePhysicalRestoreToken.ts:16`).
- Host ownership and startup commands interpolate unquoted directories. Host startup ignores the selected tools directory, and Docker startup prepends the working directory even to an absolute path and uses the default image rather than the selected image (`frontend/src/pages/databases/lib/physical-backups/restoreCommands.ts:40`, `frontend/src/pages/databases/ui/physical-backups/PhysicalRestoreEnvironmentPanel.tsx:186`).
- Backend physical tests already wait for WAL summaries before requesting an incremental backup (`backend/internal/app/apptest/physicaltest/backup_chain.go:82`). Their internal helpers cannot be imported by the independent browser suite.
- CI already runs the browser command and its lifecycle checks against the storage candidate (`.github/workflows/ci-release.yml:187`).

The design follows `docs/architecture.md` sections "Physical backups", "Frontend" and "End-to-end suites": scenarios use the image's public surfaces, while shell command construction remains in the existing frontend page slice. Root `AGENTS.md`, `frontend/AGENTS.md` and the testing, frontend-design and frontend-localization skills govern the affected areas. No production architecture or backend contract change is needed.

## Goals / Non-Goals

**Goals:** Prove that both Script tabs lead from the user's filled form to a running cluster with the expected rows. Preserve the suite's independence, one application deployment and ownership of every temporary resource.

**Non-Goals:** See the proposal's out-of-scope list. In particular, this does not start or extend the verification agent and does not add a general-purpose remote command runner to the product.

## Decisions

### 1. Extend one ordered critical path

Keep the logical and physical stages in one test invocation, with small domain helpers for the separate stages. Reuse the existing account, workspace, storage and PostgreSQL source. Add a distinct marker table for physical recovery after the logical restore has passed. Configure the source for replication and `summarize_wal=on` before it starts; these settings also permit the existing logical scenario.

Rejected: separate parametrized application deployments or relying on pytest file ordering. The user explicitly wants a long path on one instance, and one test makes its dependency on the logical restore explicit.

### 2. Use a source-controlled chain and wait for observable progress

Write markers A before the full backup and B after it. Add the physical connection through the wizard with `FULL_INCREMENTAL_WAL_STREAM`, a passing connection test and the existing local storage. Allow the wizard's requested full backup to finish rather than requesting a duplicate. Configure future scheduled runs so they cannot introduce a new full backup during the scenario.

After B, generate bounded source WAL activity, checkpoint and rotate WAL. Poll the source's public WAL-summary information until it covers the full backup's stop LSN, then request an incremental backup through the UI. Require a completed incremental with `rootFullBackupId` equal to the completed full backup's ID and `parentIncrementalBackupId` absent. The public list omits the incremental-parent field when the direct parent is the root full backup (`backend/internal/modules/backups/internal/physical/catalog/catalog_repository.go:52`). Do not accept a full-backup fallback.

After the incremental completes, write C with a transactional WAL marker, record the marker's WAL filename and the post-commit position, and rotate WAL. The filename comes from the marker's returned LSN so an automatic WAL switch cannot change it between observations. Repeat for D and require a different filename. Wait through authenticated read-only public application operations until archived ranges cover both recorded files and archive progress covers the final write. Also require a completed WAL range on the screen. A WAL range may contain multiple segments, so counting displayed rows is insufficient. Use bounded condition waits, never a fixed delay or a retry of a failed backup.

Rejected: waiting for natural rotation, guessing a sleep duration, or only checking an increasing segment counter. Those choices either slow CI or can pass on WAL produced before C and D.

### 3. Prepare isolated recovery infrastructure without changing the commands

Add two Compose fixture services: an idle PostgreSQL 18 recovery runner and a private Docker daemon for this run. Reuse the source image's exact digest for the runner. Pin the daemon image by version and digest. During runner preparation, install curl, zstd and a Docker CLI; keep these tools out of the application image and do not build another application or fixture image. Expose readiness only after these tools and the private daemon are usable. Tool installation requires package-repository access, just as pulling the fixtures does; failures must fail setup with a clear diagnostic.

The runner joins the existing Compose network, so the UI's original application address resolves without publishing or rewriting it. Its `DOCKER_HOST` points only to the private daemon's Unix socket, shared through a run-owned volume. Never mount the host Docker socket. Share a run-owned recovery volume at `/restore-work` in both runner and daemon so paths passed to nested bind mounts mean the same thing to both. Keep the daemon's data in a separate run-owned volume. The private daemon requires a privileged fixture container; use its isolated storage and network, and expose no TCP listener or host port.

Prepare the runner as root, make the recovery work directory writable by its postgres user, grant that user access to this run's private engine socket, then keep it idle without initializing a PostgreSQL cluster. Execute recovery as that user. Execute the UI's ownership command as root and its Host startup command as postgres, as the instructions require. Remove the PostgreSQL tools directory from the runner's command search path for Host command execution, so ignoring the filled bin-path field cannot accidentally pass. Pull the source's pinned PostgreSQL image into the private daemon before executing Docker recovery; nested containers need no account or source access.

The Host directory is `./host restore`, relative to `/restore-work`, and the tools field is `/usr/lib/postgresql/18/bin`. The Docker directory is `/restore-work/docker restore`, and its image field contains the source's complete digest-pinned image reference. Neither target contains a cluster at startup.

Rejected: installing PostgreSQL on each developer's host, publishing an extra application port, rewriting its URL, or sharing the host Docker socket with unlabelled nested commands. A private daemon lets the exact UI commands create arbitrary helper containers while the outer Compose project still owns their complete lifetime. It also avoids Docker path mismatches between a runner container and a host daemon.

### 4. Execute page text, including the startup instructions

Leave the target time empty and choose Script. Fill the Host fields, wait until the rendered command reflects them, and read the visible command block from the active tab. Send its text unchanged to a shell through standard input, with a known working directory. Configure that shell's standard output and error to write run-owned regular log files in the runner; PostgreSQL started by `pg_ctl` must not inherit a pipe that prevents the command client from finishing. Read these logs for masked diagnostics. Do not interpolate the command into an outer shell command, import a builder, call the restore-token operation directly or replace its URLs, flags or paths. Execute the visible ownership and startup commands under the users indicated by the page.

After Host recovery passes, close the restore dialog and reopen it through the UI before choosing Docker and filling that form. This obtains a new single-use token. Read and execute that tab's recovery command and its displayed Docker startup command unchanged. Keep the foreground Docker startup process under supervisor control, wait for readiness separately, and query the resulting container through the private engine. Verify that its configured image is exactly the reference filled into the form. Do not add lifecycle flags to the displayed command; all of its resources are already isolated inside the run's private daemon.

The source fixture uses the official image layout with configuration inside PGDATA, so the separate source-configuration repair instructions do not apply. The Docker environment needs only the source role's existing credentials to query the recovered cluster; the displayed password placeholder is not used to reinitialize an existing cluster.

Rejected: assembling an equivalent command in Python, downloading a bundle before executing the command, or validating only command strings. These can pass while the interface supplies an unusable command, or spend its single-use token prematurely. Clipboard behavior is outside this change; the visible command text is the execution input.

### 5. Repair the command builders where the scenario exposes defects

Keep construction in the existing physical restore command unit. Quote user-entered directories, tool paths and image references as shell arguments. Pass the selected tools directory into the Host startup builder and use its `pg_ctl` when nonempty. Pass the selected image into the Docker startup builder, and distinguish relative and absolute mount sources instead of always prepending `$PWD`. Preserve PostgreSQL 17 and 18 layouts in the existing shared builders; this change adds end-to-end coverage only for 18.

Reproduce the known defects through focused frontend tests before repairing them. The browser path also exposed that the explicit backup menu sent lowercase kinds despite the public contract requiring `FULL` and `INCREMENTAL`. Use the contract values as menu keys and cover both selections through the real screen. Add form-level coverage that changes the directory and tools/image fields and verifies that both preparation and startup instructions change. Do not add duplicate command builders to the test suite. Update all six dictionaries only if a necessary instruction changes; shell commands remain language-independent technical values.

Rejected: using default fields to hide the existing bugs, setting PATH so the ignored field happens to work, or patching the command after reading it. These defeat the purpose of this change. No API, generated contract or stored schema changes are involved.

### 6. Prove the recovered data and retain useful failure evidence

Record expected markers independently before executing recovery. Check that the downloaded public restore bundle includes the selected full and incremental chain and the recorded WAL segments; do not issue a second download for this check. For each running target, wait until `pg_is_in_recovery()` is false and compare the marker table's complete contents with A, B, C and D. Query the source again after both restores. Report the stage, tab, exit status and row difference on failure, rather than treating a prepared directory or a success label as a completed restore.

Extend the existing output masker to cover restore-token query values, including encoded forms that diagnostics can print. Apply masking to child output, assertion diagnostics and both outer and nested container logs before emitting them. Do not log the raw command or use shell tracing. Preserve the current ban on traces, screenshots and retained artifacts.

Rejected: relying only on row counts or statuses, comparing two live databases without independently recorded writes, or printing raw failed commands for diagnosis. Those choices can hide data loss or expose the single-use credential.

### 7. Keep ownership in the supervisor and use the existing CI job

Put runner, private daemon, sockets, recovery data and engine data in the same run's Compose project. Register ownership before starting them, so the existing project cleanup and crashed-run sweep can remove them even when pytest never reaches a teardown callback. Keep polling subprocesses and foreground Docker startup tied to the scenario/supervisor lifetime. Collect Host PostgreSQL and nested Docker logs before removing the outer containers; masked failure evidence must survive partial teardown. Bound and collect outer and recovery log streams independently, retaining completed and partial output if a source times out or fails. Cleanup failure remains a failing run.

Add lifecycle probes that start the recovery infrastructure and a nested container, then fail, interrupt or crash the run. Assert removal of the outer project and its volumes, and verify that an overlapping run and the unrelated sentinel container still answer. Stop only a run's private daemon to prove that log collection preserves available masked output and cleanup still completes. These probes need not repeat the expensive backup chain. Extend masking probes with restore URLs in all relevant streams and prove that a deliberately unusable displayed command fails rather than being replaced by test code.

Continue using `mise run e2e:browser` in `test-e2e-browser`; the added physical path runs automatically against `DATABASUS_IMAGE`. Preserve the current CI triggers and candidate-image build contract. Update `docs/testing.md` with the coverage, recovery tooling, privileged fixture requirement and commands. The existing architecture remains intact, so no architecture rule or agent instruction changes.

Rejected: a second CI job with a second application deployment, application-specific lifecycle labels inserted into copied commands, or relying only on pytest finalizers. Those would duplicate setup, alter the commands under test or leak resources when a process is killed.

## Risks / Trade-offs

- **Nested Docker needs privileged container support.** Verify this fixture early through the public command and document the requirement. It is test infrastructure only; do not relax application container permissions or fall back to the host Docker socket.
- **The critical path becomes longer and fails as one sequence.** This is the user's chosen trade-off. Keep stage-specific diagnostics and bounded waits. Measure the local public run against the existing 30-minute CI job budget; change that job's timeout only if measured legitimate work requires it.
- **Runtime tools need package-repository access.** Perform installation once per run before scenarios, fail clearly on setup errors and leave the checkout and application image unchanged.
- **A restore can appear complete while PostgreSQL uses another directory.** Execute the UI's startup command, wait for recovery completion and query the expected nonempty marker table.
- **Both tabs initially share a spent token.** Reopen the dialog between restores and never prefetch the bundle or replay a consumed command.
- **Shared frontend builders also render Manual instructions and PostgreSQL 17 paths.** Keep focused existing coverage passing without expanding the new browser scenario to those cases.

## Migration Plan

No data migration or API rollout is required. The frontend fixes ship with the normal image build, and the existing browser CI job gains the physical stages. Reverting this change removes the additional test infrastructure and command fixes without changing stored application data or the verification-agent protocol.
