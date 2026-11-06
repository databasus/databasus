## Why

Release CI builds an ARM image but runs its test suites on x64, so a release can pass without proving that logical backups, physical backups and recovery work on ARM. The latest investigated run also exposed a stale upgrade-test client requiring repair and a MongoDB backup timeout through SSH requiring investigation.

## What Changes

- Run every existing release-CI test suite natively on both amd64 and arm64, on automatic pushes to `main` and manual runs on `develop`. Include backend groups and shards, verification agent unit tests and Docker/Podman e2e, frontend tests, browser UI, real-backend browser e2e and its harness checks, script tests, filesystem tests and upgrade tests.
- Require successful results from both architectures before release version selection or publication. Keep manual verification runs free of release publication and keep unrelated checks running after a test failure.
- Build and distribute a candidate application image for each architecture, and make each image-consuming suite use the matching candidate from the same commit. Separate architecture-specific artifacts and browser failure evidence.
- Run the existing logical backup/restore matrix and physical full, incremental and WAL recovery scenarios on ARM with their data assertions intact. Exclude only MySQL 5.7 and MariaDB 5.5 server-container scenarios on arm64 because their official images have no ARM variant. Keep both on amd64 and keep container-free unsupported-version checks on ARM.
- Make image preparation, database-client dependencies and the browser UI toolchain architecture-aware. Replace the amd64-only NFS fixture while retaining the NFS permission scenario.
- Repair the upgrade harness against the actual newest release API. Investigate the MongoDB-over-SSH timeout with source/bastion diagnostics and repeated concurrent tests. Record its cause as unestablished when those checks do not reproduce it, as accepted by the user. Fix demonstrated reliability defects without hiding failures through retries or weaker assertions.
- Update testing documentation and automated checks of workflow coverage and publication dependencies.

No product HTTP operation, persisted schema or configuration changes are planned. There is no product compatibility break.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `ci-pipeline`: Require native, complete release testing on both architectures, matching candidate images, explicit unavailable-image exclusions and publication dependencies on every required result. Remove the existing architecture and develop exclusions from the affected image/browser requirements.
- `frontend-database-screens`: Let the scheduler own the first physical backup after creation, preventing a redundant manual request from queuing another backup while retaining logical startup and explicit manual queueing.
- `frontend-screen-testing`: Run the pinned browser UI environment natively on either supported architecture while preserving explicit visual-reference updates and strict comparison failures.

## Impact

- CI: release workflow matrices, runner selection, image artifacts, client-library setup, script-test execution and release dependency checks. The dev-image publishing workflow and its trigger rules remain outside this change.
- Backend: shared test-container support, engine-version scenarios, logical e2e, the upgrade harness and any backup/tunnel code whose defect is established by a regression test.
- Frontend: browser runner, pinned environment, image construction, harness tests and visual-reference selection if native rendering requires architecture-specific references. The creation wizard stops sending a redundant first-physical-backup request; layout and translations remain unchanged.
- End-to-end infrastructure: browser image consumers and the network-filesystem fixture; existing isolation, cleanup and secret masking remain required.
- Documentation: `docs/testing.md` and any descriptions of changed test commands or fixture setup. The application architecture is unchanged.
- Constraints: root `AGENTS.md`, `backend/AGENTS.md` and `frontend/AGENTS.md`, including their test, dependency and verification rules.
- Infrastructure prerequisite already satisfied: Namespace profiles `4vcpu-8ram-ubuntu-22-arm` and `8vcpu-16ram-ubuntu-22-arm` have been created and their saved architecture, resources and Ubuntu version checked.

## Out of Scope

- Adding support for MySQL 5.7 or MariaDB 5.5 on ARM, cross-architecture physical recovery, or emulating x64 tests on ARM.
- Duplicating architecture-independent lint and security scans, or adding the review-hook suite and standalone browser lifecycle-interruption suite that release CI currently omits.
- Changing release numbering, release eligibility, dev-image publication, supported product versions or the backup format.
- Rewriting test suites, unrelated product work or broad refactoring beyond code needed for the architecture coverage and demonstrated reliability repairs.
- Committing, pushing, merging, publishing or retaining a separate permanent verification workflow as part of the implementation checklist. A temporary hosted verification run is an additional evidence-gathering activity, not a prerequisite for completing a local implementation task.
