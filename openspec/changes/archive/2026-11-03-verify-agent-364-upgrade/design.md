## Context

See `proposal.md` for the motivation and scope. This design follows root `AGENTS.md` and `backend/AGENTS.md`; HTTP changes follow the API-design skill and tests follow the testing skill and `docs/testing.md`.

- `backend/internal/agent/e2e/binaries_test.go:23` defines the older version as `0.0.1`; line 50 builds both binaries from current sources. `backend/internal/agent/e2e/update_test.go:19` therefore exercises replacement and restart without exercising the released protocol.
- `backend/internal/modules/verification/internal/agentapi/heartbeat_validation.go:34` requires `maxRamMb`. `backend/internal/modules/verification/agent_contract_test.go:54` includes a gigabyte-only request among the refused heartbeats.
- The historical source at tag `v3.64.0`, `agent/verification/internal/features/heartbeat/heartbeat.go`, sends `maxRamGb`. Its independent upgrader in `agent/verification/internal/features/upgrade/background_upgrader.go` checks every ten seconds, while `upgrader.go` writes a sibling `.update` file and renames it over the executable.
- `backend/e2e/upgrade/flow_test.go:176` checks migrated verification records before finishing its restore checks. `backend/e2e/upgrade/previous_release_verification_test.go:17` currently retains only the created agent's ID, not its token.
- `mise-tasks/backend/test-upgrade:9` selects the newest release tag for the general upgrade test. That moving baseline must not determine the historical agent fixture.
- `docs/architecture.md:280` already permits explicitly approved agent protocol breaks while freezing discovery and download paths. The unconditional compatibility description at `backend/api/agent.yaml:5` and the shared-contract claim at `docs/architecture.md:322` need qualification.
- Recovery guidance stays in the repository documentation. The user explicitly excluded website pages and their translations from this change.

## Goals / Non-Goals

**Goals:** Prove the released executable crosses the rejected-heartbeat interval, preserves its configuration and executes a real restore verification. Make the failure fixture represent executable replacement permissions rather than failure to write all agent state.

**Non-Goals:** The companion claim-eligibility change owns assignment admission. This test must remain useful with either admission behavior. No historical source modifications or simulated legacy heartbeat stand in for the released binary.

## Decisions

### 1. Extract the fixture from the published 3.64.0 image

During implementation, resolve and record the immutable digest of `databasus/databasus:v3.64.0`, then extract the architecture-appropriate executable from its `agent-binaries` directory into test-owned temporary storage. Verify its `version` output identifies 3.64.0 before starting it. Keep the digest with the fixture preparation code and log only nonsensitive provenance and version data. Use the current image's public download for the replacement binary.

The fixed historical fixture is separate from the newest-release server image. Do not add another full server migration matrix: reuse the existing upgrade server/source/backup preparation and add the agent scenarios to `backend/e2e/upgrade/` after preservation assertions have completed. Each scenario uses its own registration and saved token. Hold its claim requests until the replacement agent has sent an accepted heartbeat, so the old executable cannot consume the preserved pending verification during either update scenario.

Rejected alternatives: rebuilding current code with `3.64.0` changes only metadata; rebuilding the historical source is not the distributed artifact; committing the executable adds a large opaque asset; letting the fixture follow the newest tag eventually loses this regression.

### 2. Observe rejection before releasing the independent updater

Run the old agent with the startup update check skipped, while leaving its background updater enabled. An HTTP gateway forwards agent operations to the real current server and temporarily holds version discovery and claims. Observe a forwarded old heartbeat receiving `400`, release discovery, and wait for the executable on disk to report the current version and the running process to send an accepted heartbeat. Held claims must end when their old request context is cancelled during restart; claims by the replacement stay held until the test releases them. The gateway never supplies a synthetic heartbeat response, assignment or report.

This follows the real-backend gateway pattern described in `docs/testing.md:99`. Use condition-based waits, not fixed ten-second sleeps. Engine startup checks, binary version probing and restart must execute for real. Ensure the agent is outside development-mode detection and use a current server version distinct from 3.64.0.

Then release claims and let the replacement agent complete the preserved pending verification of the existing encrypted PostgreSQL backup. If that verification has already been used by the other scenario, request a new manual verification through the current user API. Assert completion, the original agent identity and resource settings, and table/row statistics matching the rows captured by the backup, excluding rows added after backup. Configuration migration is exercised by restarting the actual replacement binary with the old saved configuration and arguments.

Rejected alternatives: a normal startup update can skip the old heartbeat entirely; waiting for any eventual successful heartbeat does not prove there was a rejection; a mock server cannot prove acceptance or real verification by the new backend.

### 3. Deny writes to the executable directory while keeping agent state writable

Give the test agent a separate executable directory and writable working directory. Use a read-only executable-directory mount and an unprivileged agent identity with access to the test's Docker Engine socket. The fixture must reach its heartbeat and background update; failure at config, log, lock or engine startup is not the intended result.

Wait for the forwarded rejected heartbeat and the actual background replacement failure, then verify the executable still reports 3.64.0. Stop the process cleanly. As the test's installation identity, install a newly downloaded current binary through a temporary file and atomic replacement, and restart as the agent identity from its existing working directory. Separate the initial flag-based setup from restart: the restart requires the saved JSON and supplies no replacement credentials or resource settings. Temporarily move the configuration aside and assert that the restart helper rejects it. Also run the actual replacement binary without credential or capacity flags, require a nonzero exit and a missing-configuration diagnostic, and restore the saved JSON before recovery. Assert accepted heartbeat, preserved settings and completed verification with known backup contents.

Hold claim requests throughout the failure phase so neither scheduler retries nor pre-eligibility assignment behavior decide whether this scenario passes. Release them only after the replacement is running and its heartbeat is accepted, then complete the preserved pending verification or a new manual verification if the earlier scenario already completed it. The second change tests withholding pending work explicitly.

Rejected alternatives: making only the executable read-only still permits replacement by rename; running as root can defeat permission-bit fixtures; making the whole working directory read-only fails for unrelated reasons; asserting old-agent assignment churn would couple this change to behavior the companion change removes.

### 4. Reuse the same recovery procedure in documentation and tests

Describe automatic replacement as requiring creation of the sibling temporary executable and replacement in the executable's directory. Explain that heartbeat contact may fail until update completes. Document stopping the built-in daemon from its working directory, or stopping the supervising service/container; downloading the correct architecture from the connected Databasus server to a temporary file; making it executable; replacing it with suitable installation permissions; and restarting using the saved configuration. Invoke the installed executable by its absolute path while keeping the existing working directory for daemon stop and start; use that same installed path for replacement. For a container-managed agent, explain replacing the deployed binary or image and restarting through that supervisor.

Do not recommend bypassing updates as a repair for protocol incompatibility. Preserve tokens and resource limits. Verification after recovery includes server-visible heartbeat contact and a completed verification; `status` or `version` alone is insufficient. Exercise the same stop, download, atomic replacement and restart sequence in the automated recovery scenario. Keep this guidance in `docs/architecture.md`; do not change website pages or translations.

Correct the agent contract description and regenerate artifacts with `mise run generate-api`. Qualify architecture prose with successful self-update and document the real-release fixture in `docs/testing.md`. Search affected documents for overlapping compatibility and update claims.

Rejected alternatives: documenting a normal restart alone leaves the failing old executable in place; downloading directly over a running executable is unreliable. Website edits are excluded by the user.

## Risks / Trade-offs

- Registry or architecture-specific fixture availability can block setup -> pin the resolved image digest, verify the binary version and fail explicitly rather than substituting a newly built binary.
- A gateway or root process can accidentally remove the condition under test -> forward protocol outcomes unchanged, observe the rejection and update failure, and run the agent unprivileged with writable state.
- Background claims can race migration assertions -> start agents only after data-preservation checks, and enqueue recovery work after a valid replacement heartbeat.
- New and old log wording differs -> prefer forwarded HTTP outcomes, version output and verification state for assertions; inspect old-agent logs only to establish the replacement failure and on test failure.
- Agents and restore containers may outlive an assertion failure -> extend existing unique resource names and cleanup for every agent process, gateway, fixture container, mount, network and engine artifact; capture image-declared volume mounts and assert their deletion on normal and failure cleanup; do not touch shared services or prune Docker.
- These scenarios add a released binary and real verification to a costly suite -> reuse its existing backend and backup fixtures, keep the fixed 3.64.0 fixture separate from the moving migration baseline, and run through the existing upgrade command and full verification workflow.

## Full-suite repair

The required full run exposed a WAL shutdown test setup race: a sampled receiver reading can be stale by the time the test queries PostgreSQL for an active replication slot. Both shutdown scenarios wait for that actual PostgreSQL condition before recording the slot identity. The same-slot equality, slot-count and thirty-second resumption checks stay in place. This repairs test synchronization without changing WAL streaming behavior.

## Migration Plan

No schema or data migration is required. Implementation completes when the automated upgrade/recovery scenarios, contract regeneration, documentation, relevant lint/format checks and `mise run test` pass and mandatory proposal/implementation review accepts the result. A failure of the real-release scenario is a release blocker to investigate, not a reason to weaken the scenario or add unrequested compatibility behavior. Preserve the two-change boundary when repairing defects it exposes.
