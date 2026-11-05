## Context

See `proposal.md` for motivation and scope. The implementation follows the module boundaries, batch reads, contract routing, injected time and supervised work in `docs/architecture.md:35`, `docs/architecture.md:57`, `docs/architecture.md:89` and `docs/architecture.md:238`. The applicable module rules are `backend/AGENTS.md` and `frontend/AGENTS.md`, in addition to the root rules. The API design and testing skills constrain operations and verification.

The current implementation supplies these constraints:

| Evidence | Consequence |
| --- | --- |
| `backend/internal/features/telemetry/background_service.go:19` defines warmup, daily jitter and backoff; `backend/internal/app/background_loops.go:14` starts that loop separately | Preserve reporting behavior while replacing its loop with a runner pass |
| `backend/internal/features/telemetry/service.go:140` reads backups inside the database loop and uses wall-clock time | Batch the reads and use one injected time for collection |
| `backend/internal/modules/backups/module.go:161` already offers latest completed logical backups in a batch | Reuse that contract rather than add a second logical telemetry query |
| `backend/internal/lib/background/runner.go:126` serializes passes and logs errors without keeping results | Add independently readable snapshots without changing scheduling |
| `backend/internal/features/system/healthcheck/service.go:29` returns the first failed check and checks only the logical scheduler | Collect all dependency results and use runner snapshots for job freshness |
| `backend/internal/modules/storages/module.go:78` permits 50 deletion attempts with a two-minute lease; `backend/internal/modules/storages/internal/files/deletion_job.go:72` processes them sequentially | Scheduling interval alone cannot define a safe stall limit |
| `backend/internal/modules/healthcheck/internal/attempts/probe_job_service.go:37` limits concurrent probes to 32 and waits for the whole pass | A busy probe pass needs a larger monitoring budget than a short scheduler pass |
| `backend/internal/app/routes.go:62` mounts the remaining uncontracted controllers | Move these operations to the generated system adapter and delete that group |
| `backend/internal/app/server.go:71` supervises the HTTP server with `background.Group` | Removing telemetry does not make the group unused |
| `deploy/helm/values.yaml:136` and `deploy/helm/README.md:303` target version for Kubernetes probes | Preserve probe targets and keep version independent of deep health |
| `backend/api/agent.yaml:141` leaves both memory fields optional; `backend/internal/modules/verification/internal/agents/service.go:197` converts gigabytes or stores zero | Require megabytes and delete the fallback without altering persisted rows |
| `backend/internal/modules/notifiers/transfer_test.go:181` still calls `CallOldAPI`; `backend/internal/agent/e2e/binaries_test.go:13` imports the old version package | Migrate these callers before deleting the helpers and packages |
| `backend/cmd/seed_dev/main_test.go:87` reads the physical config through the backups bridge and `backend/cmd/seed_dev/main_test.go:337` mounts its routes | Migrate both calls before deleting the bridge, preserving the FULL-request assertion and route coverage |

## Goals / Non-Goals

**Goals:** Keep domain data behind module contracts, expose safe and complete diagnostics, retain the existing telemetry receiver protocol, and make the change independently implementable before milestone 13c.

**Non-Goals:** Introduce a second scheduler, persist runner results, add operator tuning screens or environment settings, impose execution cancellation through health budgets, or remove unrelated legacy bridges.

## Decisions

### 1. Telemetry owns report assembly and due-time state

Create `internal/modules/telemetry/module.go` with `Deps`, `New` and a `Module` interface exposing `GetJobs`. Keep implementation under `internal/`, split by report assembly, report payload, sending schedule, identity persistence and the HTTP receiver client. The module has no HTTP adapter and takes no database connection. Dependencies are the public contracts of users, databases, storages, notifiers, backups and verification, plus the clock, logger and required configuration values.

Register `telemetry_ping` at a 15-second interval. Its first pass initializes the one-minute warmup deadline. Subsequent early passes return nil without reading identity, querying modules or sending HTTP. A due pass collects and sends once, then records the next deadline from the clock at completion. Keep the existing jitter range, success interval, retry sequence and five-second HTTP timeout. Do not follow redirects; any non-2xx receiver response triggers retry backoff. A skipped identity schedules another ordinary daily opportunity. When disabled, passes are cheap successful no-ops and do not access identity or report dependencies.

The opt-out and identity path come from `lib/config`. Add the receiver address to `Environment.ExternalEndpoints`, with the current production URL as its default, so tests can select a local HTTP receiver without a new operator setting. Pass the configured application version through `Deps`. Keep the identity file's shape, path, UUID and installation date; inject the clock when creating its UTC installation date. Never overwrite a corrupt identity.

Normalize `APP_VERSION` once through `lib/config`, preserving the existing nonempty default from `features/system/version/version.go:5` when the variable is unset. This preserves the current version response and telemetry user agent in development as well as production. Agent e2e setup can read that configuration to select the version used to build its test binaries, without importing the application or a deleted feature package.

**Alternatives rejected:** Keeping the custom loop leaves one job outside shared supervision. Giving the runner a daily interval prevents the module's shorter retries and makes a daily send look indistinguishable from a stopped loop. Adding a telemetry database table changes persistence without a need. Adding a public report endpoint expands the privacy and API surface solely for tests.

### 2. Reuse logical batch reads and add a physical batch read

Read database facts once, separate logical and physical database IDs, then use `GetLatestCompletedLogicalBackups` for logical backup facts. Its latest successful backup's creation time supplies the same inclusive seven-day check as the existing success-existence query. Add `GetLatestCompletedPhysicalFullBackups` to the backups public contract, with a batch repository query returning at most one completed FULL per requested database, ordered by creation time and ID for ties. Keep physical activity based on the database's recorded `LastBackupTime`, with its existing strict boundary. Join backup facts and verification configs in memory.

After migrating callers, remove single-database telemetry-only reads and their implementations if no consumer remains. Keep the batch API already used by verification. Preserve existing payload keys, omission rules, size rounding, sorting, duplicates, engine facts, array caps and capacity units.

**Alternatives rejected:** Per-database public calls retain the current N+1 work. Reading another module's tables from telemetry violates ownership. A new combined telemetry query in backups would duplicate the existing logical read and mix report formatting with backup storage.

### 3. Runner snapshots have a separate lock from pass execution

Keep `Job` scheduling unchanged. Add `background.JobSnapshot` and `Runner.GetSnapshots` in a separate file in `lib/background`. A job snapshot includes job name, interval, activation time, current running state, last start, last completion, last completed outcome and the corresponding internal error. Return copied snapshots sorted by job name. Store metadata behind a short-lived mutex independent of the existing pass mutex; never hold that metadata lock while invoking a job, formatting an error or doing I/O.

Record the start after obtaining exclusive execution. In a deferred completion path, record the completion time from the injected clock, outcome and error after panic recovery. Starting another pass keeps the previous completed result. Both automatic passes and `RunPass` use this path. Snapshot reads therefore finish even when a job blocks. A failed or panicking pass remains completed work and the loop continues on its existing schedule.

Capture activation at runner construction for deterministic applications that execute passes manually. Production `Start` establishes the actual activation time for jobs that have not run yet, so startup checks before the runner starts do not consume their first-pass grace. Time arithmetic uses UTC clock values; presence is explicit rather than inferred from a successful outcome.

**Alternatives rejected:** Taking the pass mutex to read diagnostics hangs health behind a stalled task. Keeping the last result in each module duplicates synchronization and cannot report a never-completed first pass. Persisting snapshots adds tables and upgrade behavior to process-local monitoring.

### 4. Health uses explicit pass budgets without changing execution

Keep monitoring policy in `app`, where all jobs are assembled. The generic runner records facts and does not decide application health. Build a policy for every registered job; default permitted pass duration is five minutes. Use these overrides:

| Job | Permitted pass duration | Reason |
| --- | --- | --- |
| `telemetry_ping` | One minute | A five-second HTTP send plus local batched reads; warmup and backoff are completed no-op passes |
| `storage_file_deletion` | Maximum of five minutes and `ClaimBatchSize * AttemptLease + one minute` | One pass can legitimately consume every sequential provider attempt's lease |
| `healthcheck_probe` | Thirty minutes | One pass waits for batches of at most 32 remote probes; over this duration the probing service is considered degraded |
| All other registered jobs | Five minutes | Scheduling, reconciliation and local cleanup passes, rather than the duration of backups or restores they dispatch |

Keep these values as application policy, with dependency injection for tests, not environment variables. Expose the budget in diagnostics so operators can interpret a long-running pass. A legitimate pass that exceeds the configured budget is reported as degraded; health does not claim to distinguish a blocked call from an overloaded service indefinitely.

Before the first completion, deadline is activation plus permitted duration. Afterwards it is last completion plus interval plus permitted duration. Also flag a currently running pass whose age reaches its permitted duration, even if it started earlier than its ordinary interval. Completion, including a failed completion, advances the deadline. At an exact deadline, mark stale. A health request observes one current time and a coherent copy of each snapshot. Budget expiration only changes diagnostics; it cancels nothing and changes no scheduling, backup or deletion policy.

**Alternatives rejected:** Three scheduling intervals would misclassify ordinary deletion work and one-second scheduler passes. The same five-minute budget for every pass ignores known sequential external calls. Automatic cancellation would turn monitoring into a new execution policy and change feature behavior. Detecting only stale starts misses a task that starts once and never finishes.

### 5. System handlers remain thin and health returns structured Problem extensions

Extend `app/systemhttpapi` through its generated strict interface. Keep services and the internal-database connectivity repository in separate role files, with the handler calling one service method for each operation. Inject the runner snapshots, job policies, clock, verification contract, client-tools checker, data-volume path, database repository, binary directory and configured version from `app`. The database query uses the request context and an independent bounded context for its check. Disk inspection reuses `lib/diskusage`; read the data-volume root, not the process working directory.

Contract operations:

| Method and path relative to `/api/v1` | Access | Responses |
| --- | --- | --- |
| `GET /system/version` | Public | Existing 200 JSON body; unchanged |
| `GET /system/health` | Public | 200 health JSON, 503 `HealthProblem`, default `Problem` |
| `GET /system/verification-agent?arch=...` | Public | 200 binary, 400/404/default `Problem` |
| `GET /system/disk-usages` | Signed-in user | 200 disk JSON, 401/default `Problem` |

`HealthReport` has `status: HEALTHY`, `checks` and `jobs`. The 503 `HealthProblem` extends `Problem` with the same `checks` and `jobs` and uses `instance_unhealthy` with a fixed safe detail. Its `status` remains the integer HTTP status required by `Problem`. Add the new problem code and its translations to all six frontend dictionaries. Every dependency result has a stable uppercase identifier, an outcome and, when relevant, a safe failure code or a stale-agent count. Job diagnostics include `name`, `intervalSeconds`, `maxPassDurationSeconds`, `isRunning`, `isStale`, optional timestamps and an optional last-completed outcome. Keep absent timestamps absent before their event. Do not serialize an error object or its text.

Collect all dependency results rather than return on the first failure. Give the internal database and agent-freshness query independent five-second child contexts so a connectivity failure cannot indefinitely withhold the rest of the report. Reuse the cached client-tools checker rather than launch every tool for every request; retain its existing distinction between fatal and nonfatal bundles. Log detailed failures with request/job context through the existing redacting logger. Attach wildcard CORS to the health operation before response writing, including its 503 result. Preserve simple cross-origin GET monitoring without adding an unrelated OPTIONS operation.

Add exact plural-path lint exceptions for `/system/health` and `/system/disk-usages`: the `system` namespace describes the single running instance. Do not broaden the exception to arbitrary `/system/*` operations. Document binary success headers and streaming in the contract. Validate architecture through its enum, stream the opened file through the generated binary response, and map an absent file to `not_found`; unexpected file errors stay generic internal errors.

**Alternatives rejected:** A bare free-text 503 violates the existing error contract and exposes internal messages. A new monitoring port changes deployments. A public disk endpoint changes access. Moving system operations into a domain module conflicts with the existing `system` exception in the architecture. Broad CORS or lint exceptions would affect unrelated operations.

### 6. Make the heartbeat memory break explicit and retain discovery

In `agent.yaml`, make `maxRamMb` required, remove the gigabyte property, regenerate consumers, and replace pointer/fallback capacity handling with a required integer. Validate the body before any heartbeat write. Do not add a replacement alias, infer a default, or change the database schema. Remove legacy acceptance scenarios and add raw-request tests for a missing field and gigabyte-only memory, checking both capacity and last-seen time remain unchanged. The new generated request no longer represents the removed field; its presence as an unrelated unknown JSON property is not a compatibility promise.

Retain the stable version and binary-download paths, response body and headers. The agent's updater is started independently of its heartbeat in `backend/internal/agent/start/start.go:109`, and polls separately in `backend/internal/agent/upgrade/background_upgrader.go:14`. Test discovery/download while heartbeats are rejected. An old agent can affect health after five minutes until it updates; installing a new backend does not reset its freshness timestamps or memory to conceal this.

The requested break supersedes the blanket previous-release guarantee for this heartbeat shape. During implementation, update the compatibility statement in `docs/architecture.md`, keep the frozen discovery paths, and replace the duplicate compatibility statement in `.agents/skills/api-design/SKILL.md` with a reference to that policy. Keep ordinary API breaks explicit in proposals; do not infer exceptions for unrelated operations. The delta removes the old compatibility requirement while carrying its persisted-memory guarantee into the updated memory requirement.

**Alternatives rejected:** Keeping gigabyte conversion conflicts with the requested deletion. Removing only the field while defaulting omitted megabytes to zero silently erases capacity. Disabling discovery when a heartbeat fails prevents installed agents from updating themselves.

### 7. Delete only the legacy surface this change replaces

Register all system routes with the user-contract guard and access middleware. Replace each old route in the same implementation task that adds its contracted handler, before its HTTP tests run; remove the old disk route before checking its 404. Connect the telemetry module to the application runner and provide the local-receiver test option in the task that first tests report delivery. Stop starting the old telemetry loop at that point to avoid duplicate reporting. Later cleanup deletes the now-unused route registration, feature packages and custom loop.

Before deleting the backups bridge, migrate both seed-test calls. Have the seed fixture own a real backups module assembled from its existing dependencies, mount that instance's routes directly, and use the same fixture-owned instance to read the saved physical config and retain the `ForceFullRequestedAt` assertion. Preserve the event subscriptions and lifecycle supplied by the old bridge. Also replace the removed system-health symbols used by that fixture. These changes remove the affected dependencies without rewriting the unrelated seed helpers; their broader migration stays in 13c. Remove `CallOldAPI` after moving the notifiers caller to the generated database client. Direct database fixtures also create the mandatory healthcheck config, so copy events see the same invariant as an API-created database; this repairs the copy test exposed by package verification. Replace old version imports in app, command and agent e2e tests with configured versions or the existing command version input, without importing `app` into the agent.

Retain `background.Group` for the HTTP server unless a smaller existing supervised server lifecycle already covers it; deleting a now-unused telemetry loop does not authorize unsupervised goroutines. Update the corresponding server and console lifecycle assertions. Remove `setUpDependencies` and its once guard if telemetry is their only remaining work, but do not expand into unrelated global bridges.

In `frontend/src/entity/disk`, replace the handwritten request with a generated-client call and re-export the generated response type where consumers still need that slice. Keep disk presentation and access unchanged. Translate network errors through the existing generated-client error path.

Add `MainScreenComponent.test.tsx` using the DOM environment, `renderScreen` and contract-typed `mockApi`. Supply the response only for `/system/disk-usages`, along with the other responses the screen needs, and assert that the rendered disk summary contains the returned used space, total space and percentage. Keep the real disk client and formatting path; an old endpoint request or a response that never reaches the summary must fail this test.

**Alternatives rejected:** A legacy route alias defeats the agreed breaking cleanup. Keeping the old helper solely for one already-contract-covered test preserves a bypass. Deleting all remaining bridges, rewriting seed setup or enabling parallel API tests belongs to 13c.

### 8. Define the monitoring vocabulary alongside the contracts

Update `docs/glossary.md` in this change with the following short entries and names:

| Term | Code/API names | Meaning |
| --- | --- | --- |
| Instance health | `HealthReport`, `HealthProblem`, `getInstanceHealth` | Whether this Databasus instance's local dependencies, verification-agent freshness and periodic work meet their monitoring checks; distinct from a healthcheck of a user's database |
| Job snapshot | `background.JobSnapshot`, `Runner.GetSnapshots` | A copied observation of one periodic job's running state and most recent start and completion, readable without waiting for the job |
| Permitted pass duration | `MaxPassDuration` in the application job-health policy, `maxPassDurationSeconds` in diagnostics | The monitoring budget used to judge a pass and calculate its completion deadline; expiration affects instance health and does not cancel execution |

Keep the existing Healthcheck definition scoped to a user's database. Link architecture's monitoring explanation to these definitions rather than introduce a second use of that term. The budget is distinct from the receiver client's execution timeout.

**Alternatives rejected:** Reusing Healthcheck for both database availability and instance monitoring makes the existing module and API terms ambiguous. Calling the permitted duration a timeout implies cancellation that this design explicitly excludes. Leaving the snapshot unnamed in the glossary would give its new public contract no shared definition.

## Risks / Trade-offs

- **Monitoring budgets trade detection speed for workload tolerance.** The deletion budget can exceed an hour with production settings. Expose it, test boundaries and keep existing job behavior; shorter cancellation or incremental progress reporting is outside this change.
- **Older agents can temporarily become stale.** Keep independent unauthenticated discovery/download working and state the break in release-facing proposal text. Do not promise uninterrupted verification while an old process updates.
- **The health body changes for scripts.** Preserve its path, HTTP status meanings and CORS; mark the body change breaking and describe both response schemas in the contract.
- **Migrating tests can hide a telemetry behavior change.** Preserve existing cases for engine facts, activity boundaries, sizes, duplicates, caps, identity failures and schedules; build real module state through APIs and capture actual outgoing JSON at a local receiver. Pure mapping and schedule arithmetic can use unit tests; do not replace modules with fakes.
- **Time-moving verification tests can also stale unrelated jobs.** Change their assertions to inspect the verification-agent check when testing agent freshness, and test overall instance status in dedicated health scenarios with explicit job state. Do not disable production checks in the test application.

## Migration Plan

1. Add runner snapshots and their focused tests; existing schedules and callers remain valid.
2. Add backup batch reads, then implement the telemetry module together with its application runner registration and local-receiver test option before testing report delivery. Stop starting the old telemetry loop. Test applications keep reporting disabled unless a test explicitly supplies that receiver and enables it.
3. Move system operations one at a time: extend that operation's contract, regenerate, implement and mount its handler, remove its old route and run its HTTP tests in the same task. Migrate the frontend disk call and test its rendered summary against the new endpoint response.
4. Tighten the agent contract and heartbeat handling, regenerate again and update protocol and self-update coverage.
5. Migrate both seed-test backups-bridge calls and the other remaining affected callers, then delete the unused legacy packages and helpers. Reconcile architecture, glossary, compatibility wording and the milestone description. Do not mark milestone 13b complete until implementation and its checks pass.
6. Run the change's package checks, lint, formatting and required review before handoff. No deployment, commit or push is part of this change's tasks.

No database migration or identity-file migration is required. Rollback is not implemented as a compatibility layer: operators follow the repository's existing refusal to run an older binary against a newer schema. This change itself leaves the schema unchanged.
