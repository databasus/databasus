## Why

The instance health endpoint observes only the logical backup scheduler, and telemetry is the last backend periodic loop outside the shared runner. Milestone 13b makes background work observable through one lifecycle and moves the remaining system operations behind the API contract before the legacy teardown.

## What Changes

- Move anonymous telemetry into a module with no inbound HTTP operations or tables. Preserve its report, persistent instance identity, opt-out, warmup, daily jitter and retry backoff; read dependencies through module contracts in batches and run `telemetry_ping` through `background.Runner`.
- Record job starts, completions and outcomes in the runner. Extend instance health with every registered job, startup grace and explicit stall budgets that account for legitimate pass duration. A completed pass with an error does not by itself make the instance unhealthy.
- Serve health, disk usage, version and verification-agent downloads through `app/systemhttpapi` and the generated contract. Preserve public discovery/download paths, architecture selection, version headers and health CORS. Keep checks of the data volume, internal database, required client tools and stale verification agents.
- **BREAKING:** Replace `/api/v1/disk/usage` with authenticated `/api/v1/system/disk-usages`, update the frontend in the same change, and delete the old route and handwritten response type. Health keeps its path and 200/503 meanings, but gains structured diagnostics; its 503 response becomes a `Problem` with safe diagnostic extensions instead of the old free-text status body.
- **BREAKING:** Require `maxRamMb` in heartbeats and remove `maxRamGb`, `LegacyMaxRAMGigabytes`, the conversion fallback and legacy-heartbeat acceptance tests. Rejected heartbeats change neither capacity nor last-seen time. Keep persisted memory and the public self-update operations intact.
- Delete `features/telemetry`, `features/system`, `features/disk`, the backups bridge, `app/background_loops.go`, the legacy route group and `apptest.CallOldAPI` after migrating their remaining callers. This includes both seed-test backups-bridge calls: route registration and inspection of the requested physical FULL backup. Retain the HTTP server's supervised lifecycle.

## Capabilities

### New Capabilities

- `anonymous-telemetry`: The existing anonymous report, privacy boundary, persistent identity, opt-out and sending schedule, including retry behavior.
- `instance-health`: Public health diagnostics, dependency checks, job outcomes, startup grace and stall detection.

### Modified Capabilities

- `api-http-server`: Contract-only route mounting, the stable agent download operation, authenticated system disk usage and a version response independent of deep health.
- `backup-verification`: Mandatory memory in megabytes, rejection of incomplete heartbeats and removal of gigabyte-only heartbeat compatibility, with persisted memory and independent self-update preserved.

## Impact

The change answers to the root `AGENTS.md`, `backend/AGENTS.md` and `frontend/AGENTS.md`. It touches backend composition, the runner, telemetry, backup batch reads, verification heartbeat handling, both API contracts and their generated consumers, the frontend disk call and the affected tests. It adds no tables, database migration, runtime dependency or deployment service.

Update `docs/architecture.md` and the API design skill's compatibility reference to describe the explicitly accepted heartbeat break without relaxing the frozen update endpoints. Add glossary entries for instance health, job snapshot and permitted pass duration, including their code/API names; distinguish them from database healthchecks and execution timeouts. Correct the milestone 13b description: Helm probes already use `/system/version`, and `background.Group` also supervises the HTTP server. The scope of telemetry follows `temp/refactor/plan.md`, section 13b.

### Out of scope

- New telemetry fields, a different receiver protocol, a telemetry dashboard or a new operator setting.
- Changes to backup, restore, database probing, file deletion or verification scheduling behavior.
- Converting the verification agent's own loops to the backend runner.
- Changing Helm probe targets or introducing health-triggered restarts.
- The general legacy cleanup, global-state removal, test parallelization and lint tightening assigned to milestone 13c.
- Compatibility shims for the removed disk route or gigabyte-only heartbeats.
