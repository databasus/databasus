## Why

The browser critical path proves that a logical backup restores, but it does not prove that the physical restore commands shown to a user work. A valid recovery script is not enough if the interface supplies the wrong directory, image, tool path or command arguments.

## What Changes

- Extend the existing critical path after its logical restore, reusing one running Databasus instance, its first account, workspace and local storage.
- Add a physical PostgreSQL 18 connection through the interface, complete a full backup and one incremental backup, then archive at least two distinct WAL segments containing later test writes.
- Restore that chain twice, through the Host and Docker tabs of the existing Script method. Fill the restore fields, read each rendered command and execute it unchanged in a prepared recovery environment.
- Start each restored cluster using the instructions displayed by the interface and compare its actual rows with independently recorded writes from every stage.
- Repair command-generation defects exercised by these paths, including quoting directories containing spaces, using the selected Host tool directory and Docker image, and mounting the selected restore directory correctly.
- Extend run cleanup, failure evidence and secret masking to the recovery resources, and keep the scenario in the existing CI browser job.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `browser-e2e`: Extend the deployment-image critical path with physical full, incremental and WAL recovery using the actual Host and Docker commands rendered by the interface.
- `physical-backups/operations`: Require the Script restore instructions to use the form's directory, PostgreSQL tools and image consistently through preparation and startup.

## Impact

- The independent root browser suite, its Compose recovery fixtures, supervisor and lifecycle checks.
- Physical restore command builders and their frontend tests; any necessary wording changes must update all six interface dictionaries.
- The existing `test-e2e-browser` CI job and `docs/testing.md`. Existing application and agent HTTP operations, stored schemas and verification agent behavior do not change.
- Root `AGENTS.md` applies throughout; frontend repairs also follow `frontend/AGENTS.md`. No backend change is planned; any defect exposed there follows `backend/AGENTS.md` and the repository's standing repair authorization.

## Out of Scope

- Physical backup support in the verification agent.
- The Manual restore method, PostgreSQL 17 coverage, other storage providers, SSH, TLS and failover scenarios.
- Restoring to a selected timestamp; these two restores use the latest archived state.
- A separate Databasus deployment or account for each restore, new CI triggers, and unrelated workflow or documentation corrections.
- API or schema changes and broader redesign of the recovery interface.
