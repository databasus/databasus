## Context

See `proposal.md` for the motivation and scope. No Patroni scenario has been run for this change, and no runtime defect is assumed.

Relevant current behavior:

- `backend/internal/app/apptest/physicaltest/promotion.go:60` exercises promotion and recovery, while `source_promotion.go:14` invokes `pg_promote` directly. `promotion_source.go:49` changes the saved connection to the standby before promotion.
- `backend/internal/app/apptest/physicaltest/source_reconnect.go:132` already exercises a stable test-proxy address, but lines 160-163 explicitly route it to the standby and restart application modules. That does not establish unattended Patroni/HAProxy recovery.
- `backend/internal/lib/testcontainers/postgres_physical.go:120` builds fast fixtures with durability disabled. The HA crash fixture must not inherit those settings.
- `backend/internal/app/apptest/physicaltest/schema_recovery.go:18`, `sequence_recovery.go:20` and `restore_schema.go:24` provide reusable schema/sequence preparation and recovery assertions. `published_history.go:14` checks stored history but fixes the timeline to 2.
- `backend/internal/engines/postgresql/physical/timeline_history.go:22` identifies the source and requests history on one replication connection. `wal_session.go:27` verifies the actual WAL session before handing it to the client. Reuse these mechanisms.
- `backend/internal/modules/backups/internal/physical/chains/restore_set_service.go:98` selects WAL on the chosen FULL's timeline. Lines 108-123 require that timeline's published history in the FULL's storage before including later WAL.
- `backend/internal/modules/backups/internal/physical/backuping/executor.go:302` preserves the original stream failure if the identity recheck cannot connect. The scheduler chooses immediate replacements before scheduled work in `backuping/scheduler_decision.go:24`.
- `backend/internal/modules/backups/internal/physical/restorestream/recovery_script.sh:516` appends replay settings when WAL is shipped. The script has no general removal of inherited `primary_conninfo`, competing recovery targets or signal files. This is a test target, not evidence of a reproduced Patroni failure.
- `backend/internal/app/apptest/physicaltest/orphan_sweep.go:32` runs the physical cleaner and checks a restore token; it neither runs the deletion worker nor boots the downloaded recovery set.

Root `AGENTS.md`, `backend/AGENTS.md`, the testing skill and `docs/testing.md` apply. The placement follows `docs/architecture.md` sections Backend layout, Shared code, Physical backups and Stored files: infrastructure fixtures know no application tables; application composition stays in `apptest`; runtime repairs stay with their owning engine or module. No architecture exception is needed.

## Goals / Non-Goals

**Goals:** establish the same observable recovery guarantees on both PostgreSQL majors, with evidence from real Patroni elections, real HAProxy routing, real backup tools and booted recovered clusters. Keep the suite independently runnable and every scenario isolated.

**Non-Goals:** see the proposal's exclusions. In particular, the suite establishes one concrete two-node deployment, not general compatibility with Kubernetes or every Patroni slot mode. Fixture observations do not become new product endpoints or production-only test controls.

## Decisions

### 1. Run the environment from Go using ordinary Docker containers

The new package `backend/internal/modules/backups/e2e/postgresqlphysical/patroni` holds the test entry points. Infrastructure support belongs in `backend/internal/lib/testcontainers`, with fixture assets beside the fixtures. Scenario orchestration belongs in `backend/internal/app/apptest/physicaltest`, matching the existing E2E wrappers and allowing private chain/download/recovery helpers to remain private. Add focused scenario entry points instead of exporting the entire helper implementation.

```text
Go test --> apptest Databasus --> HAProxy --> PostgreSQL A + Patroni
                                   |
                                   +----> PostgreSQL B + Patroni

                                         Patroni A --> etcd
                                         Patroni B --> etcd

storage --> existing recovery script --> independent PostgreSQL
```

Each scenario gets a fresh application database, storage directory, source cluster, Docker network and restored targets. Databasus itself is composed in the Go process; its test database uses the existing harness. Test containers expose randomly allocated loopback host ports. The network uses the existing free-subnet mechanism. PostgreSQL data uses run-owned disk-backed storage, with `fsync=on`, `full_page_writes=on`, `synchronous_commit=on`, `summarize_wal=on`, and sufficient senders/slots on both nodes. Assert the live settings before crash tests. Initialize with data checksums and enable Patroni `use_pg_rewind` so the old primary can rejoin before the second switchover.

Independent scenario subtests run in parallel with a root-test-owned capacity limit of two active HA fixtures, avoiding an unbounded number of PostgreSQL pairs. Every scenario still owns its application and containers. No process-global dependency or shared source server is introduced.

Register cleanup as soon as each resource exists, including partial startup. Stop application loops and release blocked traffic before deleting containers and networks. Cleanup uses a bounded context independent of the canceled test context, removes only run-labeled resources, and fails with the resource identity when removal cannot be confirmed. Add fixture lifecycle coverage for successful completion and intentional setup failure; intentional failures run in isolated child tests so the outer test can assert both their nonzero exit and resource absence. Reuse the existing testcontainers cleanup rather than introduce a supervisor program. Mask credentials in bounded failure log tails.

Alternatives: kind/k3s add Kubernetes scheduling, storage and service routing outside the question being tested. Compose plus a separate supervisor duplicates test lifecycle management. Replacing the existing manual-promotion tests would lose their narrower engine regression coverage; retain them.

### 2. Use one explicit version and dependency matrix

| Component | Planned version or existing immutable reference |
| --- | --- |
| PostgreSQL 17 source and restore tools | `docker.io/library/postgres:17@sha256:ff87138949ad9a46f32ff23bd6852b52ce024a13dffe9b4087be98292cdd3b4f` |
| PostgreSQL 18 source and restore tools | `docker.io/library/postgres:18@sha256:e3d24f6906d0760bf4da0dae872415391ededefad23fd471c47dc53b0ddf3c42` |
| Patroni, identical for both majors | `4.1.5` |
| etcd, one member | `3.6.16`, using the etcd v3 API |
| HAProxy | `3.2.26` |

The PostgreSQL references come from `backend/internal/lib/testimages/images.json:316` and `:330`; the implementation records the actual server patch banners as evidence. Build a small first-party Patroni fixture image from each pinned PostgreSQL base, using a locked, hash-verified Python dependency set with the etcd3 client and PostgreSQL driver. Install from that lock without resolving dependencies during ordinary tests. Add third-party etcd/HAProxy images to the existing catalog with immutable index and platform digests for linux/amd64 and linux/arm64. Missing mandatory images or tooling fail; they do not produce a successful skip. Reuse existing client-tool bundles.

Version selection sources checked on 2026-11-11: [Patroni 4.1.5 release notes](https://patroni.readthedocs.io/en/latest/releases.html), released 2026-08-12; [etcd 3.6.16 release](https://github.com/etcd-io/etcd/releases/tag/v3.6.16), released 2026-10-22; [HAProxy 3.2 source releases](https://www.haproxy.org/download/3.2/src/), listing 3.2.26 on 2026-10-14. The releases clear the project's waiting periods. Exact new image publication dates, image digests and Patroni transitive pins are not yet recorded: the dependency task must verify those before adopting them. Release dates do not prove the publication age of a later rebuilt image. Use an age-eligible immutable image of the selected version or an age-eligible, checksum-verified upstream distribution in a first-party fixture build; do not silently relax age checks or select a different release line. This keeps dependency resolution a local implementation step, not a requirement to publish images or run CI.

Alternatives: a full HA distribution adds unrelated services and settings; floating images or an unpinned `pip install` make runs change without a code change. One PostgreSQL major would leave half the explicitly requested coverage unverified.

### 3. Let Patroni and HAProxy perform all role and route changes

Use Patroni REST switchover requests with the current leader and explicit candidate. Automatic failover is caused by abruptly killing the entire primary-node container, including Patroni, and leaving it stopped; do not gracefully stop PostgreSQL or let Patroni immediately restart it. Never call `pg_promote`, manually change routes, restart Databasus, or call `RestartModules` as a recovery step.

HAProxy forwards TCP connections to PostgreSQL and checks Patroni `/primary` on each node's REST port. Set `on-marked-down shutdown-sessions` so existing backup and WAL sessions do not remain attached to a demoted node. Inspect new SQL connections through that same saved endpoint after every switch and match `inet_server_addr()` to the node's known container address; verify `pg_is_in_recovery()=false`, the live WAL timeline and the unchanged system identifier. Observe the receiver on the new node through its application name/slot and actual newly archived WAL, not only an API status.

Use Patroni `loop_wait=2`, `retry_timeout=3`, `ttl=20`, `primary_start_timeout=0` and asynchronous replication with explicit replay barriers. These satisfy Patroni's documented timing constraint. Wait for roles, HAProxy health and rejoin state with conditions. For the second switchover, first wait until the former primary has rejoined as a streaming replica and replayed the next control transaction.

Before any primary loss or planned switch, commit a unique control row on the current primary, obtain a post-commit WAL position, and wait on the future primary for both replay through that position and row visibility. Direct administrative SQL belongs only to fixture setup and observations; Databasus keeps its provisioned limited credentials. Do not equate zero sampled lag with durable delivery of the control transaction.

Alternatives: direct promotion bypasses Patroni, a SQL-only health check can select the wrong role, and restarting Databasus hides reconnect defects. Synchronous replication would change the tested deployment; explicit replay barriers establish the required crash precondition without claiming delivery of arbitrary asynchronous writes.

### 4. Separate slot protection, absence and lost WAL

Create the database and limited user through the Databasus API before enabling backups. Read the exact generated WAL slot name for observation, configure that name and `type: physical` in Patroni dynamic `ignore_slots`, confirm the configuration is applied, then enable backups. Preserve that configuration throughout positive switch/recovery cases. Do not use a wildcard exemption that could hide unexpected slots.

The protected-slot scenario disconnects the receiver without deleting the database or disabling the backup configuration that owns the slot. Use test-controlled transport interruption while keeping independent admin observations available. Observe at least three completed Patroni HA cycles from fresh DCS member updates or cycle logs; advancing wall time or reading a cached endpoint three times does not count. Keep the consumer disconnected across those cycles, assert the inactive slot and a valid `restart_lsn`, then reconnect and verify slot progress and recovered control rows.

Use fresh fixtures for the negative modes:

- Absence: before the first promotion, prove the external slot is absent on the candidate. Observe the actual post-promotion policy, including `SLOT_MISSING` health if reception cannot yet resume. Allow successful recovery only when the required WAL remains available; no test-created substitute slot silently repairs the scenario.
- Unprotected removal: omit the external `ignore_slots` entry, disconnect the receiver and prove Patroni removes the inactive external slot during completed management cycles. Reconnection then exercises Databasus's existing response.
- Lost WAL: after that removal, keep reception disconnected, generate and checkpoint WAL until an identified required source segment is actually recycled, and verify it was never archived. Configure bounded source retention for this fixture; do not merely delete a file from storage. Reconnect and assert that the affected old-chain target is refused, even if a new slot/FULL is created. Any slot rebuild remains subject to the existing hourly cap and cluster-identity checks.

Patroni does not copy ignored external slots to a replica. Positive scenarios therefore permit the existing missing-slot recovery on the new primary; they prove recovery through a new FULL, not uninterrupted old-chain PITR. Permanent slots are a different mode and receive no success claim here. `ignore_slots` also does not alter handling of the short-lived base-backup slots; record any reproduced interference and repair only what the accepted scenarios require.

Alternatives: protecting a slot only while it is active cannot distinguish protection from Patroni failing to drop an in-use slot. A permanent slot would hide the missing-slot path and test a different guarantee. Artificially removing archived files would test storage corruption rather than Patroni slot ownership.

### 5. Observe transfer overlap and respect the existing recheck policy

FULL and INCREMENTAL interruption are separate fresh-fixture cases on each major. Prepare sufficient changed, poorly compressible data; for INCREMENTAL, wait for required WAL summaries before starting. Match the active `pg_basebackup` backend and a positive change in `pg_stat_progress_basebackup.backup_streamed`, then trigger the Patroni switchover. Correlate its stream/connection termination with the observed role change. A backup that completed before the role change is a failed overlap precondition, never a passed scenario.

When necessary, pace or gate bytes only at a test transport boundary so the transfer remains active until Patroni changes roles. Any gate must expose a reached signal, release on cleanup, leave Patroni/DCS traffic available and preserve source TLS/HBA. It must not inject a successful backup result or replace Patroni's election. The implementation task includes verification of the observation/gate itself and records the chosen fixture mechanism. A fixed delay and a larger payload alone are not proof of overlap.

For a stream failure whose recheck reaches the new primary, require the exact existing timeline reason and one replacement FULL. If the recheck hits the routing outage, require preservation of the original failure and the existing scheduled retry policy. Configure a short supported schedule through the API for this dedicated outage case, wait for its due pass, and do not manually force a FULL to disguise a missing automatic retry. Assert no extra immediate replacements appear for that generic failure. Existing identity-probe-failure regressions remain part of verification.

For the immediate-replacement assertions, use transport coordination to keep the original operation's failure pending until the new primary is reachable, and exercise the outage branch separately by holding the primary route unavailable through the recheck. If this requires a new test transport primitive, add only that primitive with lifecycle coverage. Do not change the runtime recheck policy just to make one timing outcome pass. Check failed attempts are terminal and unavailable for completed-backup download, then download and boot the successful replacement set and inspect its rows/schema.

Alternatives: waiting for `IN_PROGRESS` can hit preflight rather than transfer. Requiring a failover classification when the recheck could not connect contradicts the current policy. Allowing either result without proving its precondition would hide failures; each branch must have observed evidence.

### 6. Reuse history security and prove recovery reachability

Complete one successful same-timeline INCREMENTAL with the limited user before switching. After each of the two switches, request an INCREMENTAL against the old root to exercise timeline refusal and the existing immediate replacement policy. Use live application loops or explicit job passes, without restarting modules. Record the old terminal attempt, replacement FULL, timeline, storage and archive coverage independently for each switch. Wait for each replacement before starting the next switch.

Generalize the existing timeline-history assertion to the observed timeline. Compare decompressed stored bytes with the source history, parse ancestry/fork positions, and check filename, sidecar and catalog storage against the actual selected FULL. Administrative fixture reads can inspect source files; the Databasus connection must retain limited credentials. Check the limited role's memberships and denied file/table reads before and after promotion. Existing same-session identity refusal regressions supplement the positive same-cluster identity checks inside every HA scenario.

Close and archive WAL for an old-timeline target before switching; replica replay alone is insufficient. Save that target and its expected rows/schema. On each new timeline, archive rows written after the replacement FULL and choose a reachable later target. Also save a target after promotion but before replacement FULL completion and verify `409 target_time_unreachable` both before and after replacement completion. Keep same-timeline recovery during a newer FULL as an existing regression, so the new cross-timeline restriction does not remove valid targets.

Inspect each downloaded set's selected FULL, timeline and required history. Confirm actual history use from PostgreSQL archive-recovery logs or an equivalent observed successful archive retrieval of the expected `.history`; merely shipping the file or starting a FULL-only restore is insufficient. If the base copy already holds the same history, use an isolated recovery preparation that leaves the published archive as the available source for that history; do not alter the stored archive or claim cross-timeline replay. Verify recovery remains on the selected FULL's timeline.

Alternatives: checking only history presence misses wrong storage, invalid content and unused files. Checking only the newest chain misses damage to still-retained older recovery points. Expecting the new FULL to make earlier targets reachable would introduce unsupported cross-timeline PITR.

### 7. Boot independent recovered clusters and run both cleanup stages

Use `NewSchemaRecovery`, `NewSequenceRecovery`, `RequireStage`, `RequireDefinitions`, `RequireAllocations` and `RequireWrites`. Prepare schema/serial/sequence objects before the initial FULL and perform the existing post-FULL DDL and sequence stages before/after the relevant switches. Record semantic catalog/row observations per recovery target and reopen connections after a switch. Include added defaults, indexes, constraints, renames, table deletion and rolled-back DDL, as well as `CACHE 1`, larger caches, `nextval`, forward `setval` and `ALTER SEQUENCE`. New writes must avoid restored IDs without requiring gapless or adjacent values.

Before independent recovery, make the former primary unreachable and use a separate network/target container with no Patroni process or source volumes mounted. Test explicit-target recovery, latest recovery and an explicitly selected backup with no extra archived WAL on both majors. For copied stale settings, prepare disposable recovery inputs that contain `primary_conninfo`, an old `restore_command`, conflicting active `recovery_target_*` settings and recovery/standby signals. This is controlled configuration input, not archive corruption. Require the new recovery operation to override or remove these settings in the restored output, including when there is no additional WAL; do not edit them out in a startup wrapper that hides the recovery script's behavior.

If the source keeps configuration outside PGDATA, follow the existing recovery diagnostic to supply the independent configuration in the test environment. Copy or recreate only target-side configuration and credentials; do not disable source TLS, weaken source HBA or grant the backup role extra privileges. Source authentication remains the configured authentication throughout.

After two switchovers, keep at least the old target and newest target protected by retention. Run `backups_physical_cleaner` and `storage_file_deletion` until the relevant work is processed, download fresh sets for both targets and boot both. Check rows, semantic schema, sequence properties, history use and generated-ID writes. Repeat the gap refusal. Assert the jobs processed their passes; unrelated pending work in another test is outside the fixture's observations.

Alternatives: a `COMPLETED` status or issued token does not demonstrate recovery. Booting with access to the original primary could conceal inherited recovery settings. A separate SQL assertion implementation would duplicate the already available schema/sequence fixtures.

## Risks / Trade-offs

- Two PostgreSQL pairs plus recovered targets can exhaust local resources. The per-root capacity limit bounds active HA fixtures, and each scenario removes targets as soon as its checks finish.
- Patroni may attempt to restart a crashed primary or drop unprotected inactive slots. Kill the whole node for failover and make slot ownership an explicit scenario input.
- Timeline numbers need not be exactly 2 and 3. Assert advancement and observed ancestry, and use the actual timeline in every helper.
- Recheck timing and short transfers can make a test pass without exercising its claim. Require observed overlap and branch-specific connection evidence, with bounded, cleanup-safe transport control where needed.
- A fixture can preserve the control row yet lose old archive coverage. Require old-target archival before switching and restore it again after cleanup.
- Native sequence values can advance during failover and recovery. Compare semantic definitions and valid allocations, not equality of cached counters or adjacent IDs.
- New dependencies can be released or republished after planning. Verify publication age and immutable pins locally before use; report the exact runtime versions and platform actually tested. Passing one major or one architecture does not establish the other.

## Migration Plan

The WAL time repair adds nullable `last_transaction_at` to the WAL catalog and sidecars, plus matching `pg_waldump` binaries in the existing PostgreSQL 17.10/18.4 bundles on both architectures. It changes no HTTP request or response shape. Existing rows without source-time evidence do not extend the confirmed PITR time beyond their completed-backup anchor; explicit completed-backup recovery and latest archive replay remain available. No receipt-time fallback invents missing source evidence.

Implement the dependency/fixture slice first, then the baseline PostgreSQL 17 case, repeat that case on PostgreSQL 18, and add each remaining scenario to both versions before moving on. This orders debugging without reducing the final matrix.

Document the source configuration in `docs/postgresql-patroni.md`, link it from `docs/testing.md`, and add short glossary entries for primary, standby, promotion, switchover, failover, Patroni's DCS and point-in-time recovery. Record tested versions, slot modes and exclusions. If a reproducer requires a runtime repair, keep it within the affected ownership boundary and update these artifacts with the observed cause and verification. The change ends after implementation, documentation, verification and required review, ready for a later commit; no automatic apply, commit or push is part of planning.

Local full-matrix command, from the repository root:

```sh
cd backend
source ../scripts/fedora.sh
apply_libedit_shim
TESTCONTAINERS_RYUK_DISABLED=true go test -count=1 -timeout 40m ./internal/modules/backups/e2e/postgresqlphysical/patroni
```

This command must execute all added scenarios for both majors. A focused rerun diagnoses a failure but does not replace this run. Missing packages or mandatory tools fail. Changes to the shared harness require `mise run test`; targeted engine, backup and recovery-script regressions cover any runtime repair, and backend lint covers modified Go code and contracts. No task invokes `mise run test-full`.

## Implementation evidence

The initial FULL plus WAL recovery scenario failed on both PostgreSQL 17.11 and
18.6 because Patroni wrote absolute `hba_file` and `ident_file` paths into the
copied configuration. PostgreSQL attempted to open the source data directory
inside the independent target and refused to start. The recovery script now
removes source data-directory paths and uses the copied local authentication
files when present. Missing external configuration still requires operator
preparation under the existing diagnostic. The source access rules are unchanged.

Recovery preparation also clears inherited replication settings, competing
recovery targets and signal files in the restored configuration, including its
local include graph regardless of filename extension, before selecting the new
recovery operation. Assignments with and without `=` are removed. Included
paths resolve relative to their containing file; `include_dir` follows
PostgreSQL's `.conf` selection, and missing `include_if_exists` files are allowed.
This applies even when a selected backup ships no extra WAL.

Patroni 4.1.5 caches the slot inventory and does not discover every externally
created slot during an ordinary management cycle. The slot-ownership cases use
Patroni's public pause/resume configuration operation after disconnecting the
receiver, observe the paused cycle and then at least three resumed cycles.
Resume schedules Patroni's own slot inventory reconciliation. This makes the
protected and unprotected cases exercise actual slot matching and deletion;
the test does not drop or recreate the Databasus slot itself.

Transfer observation filters BASE_BACKUP CopyData for archive data messages
(`d` inside CopyData), rather than archive-name/progress metadata. The reported
byte count excludes both protocol headers, and the scenario also checks the
active server-side base-backup progress view before requesting switchover.

The outage retry cases wait for the failed backup worker to finish and release
its database reservation before restoring connectivity. Through the API they
record a concrete cron due time, process scheduler/request passes and observe
no later backup or forced request throughout the preceding window. The test
then identifies the exact FULL dispatched with reason `full cadence is due`,
checks its dispatch/creation times against that boundary, waits for completion
and boots its recovery set. These dedicated cases configure Patroni's permanent
physical slot on both nodes to exclude a slot-rebuild FULL as a competing
trigger. The separate absent-slot and lost-slot cases retain their ownership
modes and do not receive that configuration.

Expected-failure fixture children have parent-side cleanup registered before
launch. It removes only containers/networks with the child's unique fixture
label and files in the parent's assigned child directory, including when a
hard deadline kills the process group before child cleanup can run. A forced
parent-interruption case verifies this fallback; normal failure cases still
assert that the child cleaned its own resources before fallback runs.

The Patroni package is registered in `backend/test-slow-packages.txt`. The
standard runner gives slow packages 40 minutes and other packages 15 minutes;
runner tests cover both limits and selection of this package in the slow group.

The schedule-boundary reproducer exposed an existing UTC cron defect: the
PostgreSQL driver can return the previous run's timestamp in the host's local
offset. Cron evaluation used that offset as its default timezone and missed
the recorded UTC due time. `lib/schedule` now normalizes that timestamp to UTC
before computing the next occurrence, while explicit timezone prefixes keep
their configured zone. A pure schedule regression first failed for stored
UTC+3 and UTC-3 timestamps and then passed together with the existing schedule
tests, including the explicit-zone case. The Patroni outage cases exercise the
same repair through the physical scheduler and booted recovery.

The held-transfer proxy adds the PostgreSQL replication protocol option
[`MAX_RATE 2048`](https://www.postgresql.org/docs/18/protocol-replication.html)
to the pending gated `BASE_BACKUP` command. A gateway read pause alone does not
bound the data already buffered upstream, so a fast server can finish before
the test observes its progress. The rate cap gives the test time to observe the
active copy. The proxy forwards the real archive and manifest, records the
backend PID from `BackendKeyData` and matches that exact PID in
`pg_stat_progress_basebackup`. The next backup runs through the released gate
with its original options. Lifecycle cases still require completion after
release and cleanup after cancellation.

Absolute configuration includes below the source `data_directory` recorded in
the copied main, automatic or locally included configuration are rewritten as
relative paths inside the restored PGDATA before sanitization. A discovery pass
follows local includes in configuration order, then repeats with the discovered
source directory so absolute includes can precede its declaration. Recursive
include cycles are bounded, and repeated includes outside the active recursion
path retain their setting precedence. Nested relative paths still
resolve from their containing file. This covers direct includes, optional
includes and include directories without retaining a dependency on the source
filesystem or the host path used while preparing recovery.

Authentication-path preparation prefers a copied file referenced by `hba_file`
or `ident_file` and rewrites its source-local path under runtime PGDATA. Native
recovery uses the prepared absolute data-directory path; `--combine-image`
uses the selected image's PGDATA path, so host preparation paths are not retained
inside its runtime container. Authentication paths cannot use the relative
include strategy because PostgreSQL resolves them from its launch directory
([File locations](https://www.postgresql.org/docs/18/runtime-config-file-locations.html)).
It does not require default root-level authentication files when the referenced
files were copied in a subdirectory. The explicit-backup regression keeps the
source's active configuration external and stages local copied configuration
with `auth/hba.rules` and `auth/ident.rules`. Booted recovery checks both file
hashes when default root-level files are absent and when distinct defaults
coexist. In the latter case, the root HBA rejects all connections and the root
ident contains an unrelated mapping, so a fallback cannot satisfy the restored
connection and file-hash assertions. Source authentication remains unchanged.
Both the direct `pg_ctl -D` start and the
image entrypoint after `--combine-image` preparation are exercised on both majors.

A delayed-publication regression reproduced an invalid restore token: an old
timeline's WAL received after promotion extended its apparent time into the
replacement-FULL gap. `pg_waldump` now decodes complete transaction timestamps
from each queued segment before publication. The source timestamp is cataloged
and included in the sidecar; `received_at` keeps its archiving and retention
meaning. The resolver and coverage use only the selected timeline's contiguous
published run and completed-backup anchors. They neither merge timelines nor
infer the source time from a later FULL's start or history-publication time.

`pg_waldump` prints prepared-transaction identifiers verbatim, so their multiline
payload can imitate both descriptions and record headers. The transaction-time
reader accepts anchored Transaction headers and retains a trusted prefix until
PREPARE. After PREPARE, it treats subsequent descriptions as untrusted candidates
and verifies the last candidate through a separate `--start` / `--limit=1` decode.
Only that invocation's first line, at the requested LSN, can supply new evidence.
A failed verification preserves the trusted prefix and reports a diagnostic.
This needs at most one extra decoder invocation per segment, keeps bounded output
memory, and still recognizes genuine COMMIT/ABORT completion after PREPARE.

Inclusive PostgreSQL time recovery needs a transaction strictly later than the
requested target. The confirmed upper bound is therefore one microsecond before
the latest decoded COMMIT/ABORT timestamp, including prepared transactions.
An unconfirmed segment does not extend this bound. If a record continues past
the segment boundary, the uploader preserves the WAL, logs the diagnostic and
uses only complete records already decoded. Latest recovery without an explicit
target may replay the final confirming transaction as well.

An exact completed-backup target needs no later WAL when no transaction can
confirm time-based replay. The resolver omits extra WAL only for that exact
selected FULL/INCREMENTAL completion point and only without a later confirming
transaction. Required backup ancestors remain; latest recovery still ships
eligible WAL. Booted regressions on both majors take the advertised statistics
point and pass its microsecond precision through the recovery script. They
verify source WAL has no later transaction, successful startup and new writes.


### Maximum confirmed transaction time

WAL record order does not guarantee increasing transaction timestamps. The reader retains the maximum trusted timestamp rather than replacing it with the most recently decoded record. A verified completion after PREPARE extends this maximum only when newer. Keeping the last record timestamp was rejected because it can shrink coverage despite an already confirmed later source time.

The regression uses real PostgreSQL 17/18 WAL, changes an earlier COMMIT timestamp only in a copied segment and recalculates its CRC32C. The matching real decoder checks the resulting record. This makes decreasing timestamps deterministic without depending on a scheduling race or changing the source clock. Both ordinary later COMMIT and verified prepared-COMMIT paths must retain the earlier maximum.
