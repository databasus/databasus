## Context

See `proposal.md` for motivation and scope. The following observations describe the checkout inspected on 2026-11-10; no contamination or recovery reproduction has run yet.

| Current behavior | Evidence |
| --- | --- |
| Receiver configuration contains no expected system identifier; the native receiver starts with `--no-loop` and `--synchronous`. | `backend/internal/engines/postgresql/physical/wal_receiver.go:30`, `:47`, `:72` |
| Process startup immediately marks the receiver as running, before server authentication or streaming is established. | `backend/internal/engines/postgresql/physical/wal_receiver.go:97`; `backend/internal/modules/backups/internal/physical/walstreaming/receivewal_process.go:27` |
| Startup verifies or creates a slot, removes partials, recovers queued files, then starts receiver supervision and companion loops. | `backend/internal/modules/backups/internal/physical/walstreaming/streamer.go:230` |
| Reconnection is supervised by Databasus; repeated recycled-WAL failures can request a slot rebuild. | `backend/internal/modules/backups/internal/physical/walstreaming/streamer.go:291` |
| Slot verification, rebuild and inspection open SQL connections without matching their cluster identity. | `backend/internal/modules/backups/internal/physical/walstreaming/deleted_database_slot.go:18`; `slot_rebuild.go:19`; `slot_lsn_watcher.go:92`; `backend/internal/engines/postgresql/physical/replication_slot.go:19` |
| A fatal streamer exit marks its claim failed and triggers existing chain notifications. Queue deletion is separately conditional. | `backend/internal/modules/backups/internal/physical/walstreaming/manager.go:351`, `:591` |
| Streamer rows persist status and failure start time, but not the failure reason; the first periodic sample can clear failure start time. | `backend/internal/modules/backups/internal/physical/catalog/wal_streamer.go:15`; `wal_streamer_repository.go:71`; `backend/internal/modules/backups/internal/physical/walstreaming/lag_monitor.go:249` |
| WAL statistics have a closed reason enum without identity mismatch, although connection testing already names that condition. | `backend/api/openapi.yaml:4977`, `:6342` |
| Timeline history already verifies identity and obtains content on the same replication connection. | `backend/internal/engines/postgresql/physical/timeline_history.go:22` |
| Existing promotion recovery tests create a real pair but change the saved endpoint to standby before promotion. | `backend/internal/app/apptest/physicaltest/promotion_source.go:24`; `promotion.go:60` |

The vendored client inventory names PostgreSQL 17.10 and 18.4 (`tools/dependencies/clients.json:337`, `:383`). In both versions, `StreamLog()` reads the current connection's system identifier and supplies it as the receiver's expected value. `ReceiveXlogStream()` compares it with another response on that connection. This does not compare the server with Databasus's saved identity. Sources: [PostgreSQL 17.10 receiver](https://github.com/postgres/postgres/blob/REL_17_10/src/bin/pg_basebackup/pg_receivewal.c#L499), [PostgreSQL 18.4 receiver](https://github.com/postgres/postgres/blob/REL_18_4/src/bin/pg_basebackup/pg_receivewal.c#L499), [PostgreSQL 18.4 stream implementation](https://github.com/postgres/postgres/blob/REL_18_4/src/bin/pg_basebackup/receivelog.c#L491). Both connection retry loops obey `--no-loop`; following a timeline on an existing session is not a new network connection.

The [replication protocol](https://www.postgresql.org/docs/18/protocol-replication.html) returns the source's system identifier and timeline through `IDENTIFY_SYSTEM`. The comparison must bind to the session later carrying WAL.

## Goals / Non-Goals

**Goals:** keep the native receiver's disk format, synchronous writes, resume behavior and timeline handling, while making source identity a condition of every receiving session and every WAL slot mutation. Reuse existing connection security and ownership fences. Make refusal and subsequent recovery observable without waiting for a successful periodic sample.

**Non-Goals:** the exclusions in `proposal.md` apply. This does not turn cluster identity into proof of compatible history for same-identifier divergent clones, validate queued file contents in production, or repair old contamination. Backup-stream identity enforcement outside continuous WAL reception is not expanded by this change.

## Decisions

### 1. Keep the native receiver behind a private session bridge

The PostgreSQL physical engine opens the upstream connection using its existing physical-replication configuration, credentials and tunnel route. It sets the receiver application name on that real upstream session, issues `IDENTIFY_SYSTEM`, and checks the saved identifier before handing any session to the child process. The engine rejects missing expected identity rather than treating the connected server as authoritative.

After a match, a one-use local bridge lets the unchanged native receiver speak to that already authenticated session. Use a Unix socket in an owner-only temporary directory, never a publicly reachable listener. The bridge completes only the local startup exchange, carrying the upstream session's required parameter statuses into the local response, and then forwards PostgreSQL traffic bidirectionally. It must never send a fresh upstream startup message or open another upstream connection after verification. Preserve the native client's server-version checks and setup commands.

Use the existing `pgconn` and `pgproto3` dependency. Synchronize any buffered protocol data before taking direct ownership of the upstream connection; do not read the raw socket concurrently with the protocol client. The bridge observes the transition into `CopyBothResponse` to signal that verified reception actually started, then forwards the copy stream without interpreting WAL payloads. Startup forwarding must preserve authentication outcomes, server errors and EOF correctly.

```text
saved identifier A
       |
       v
engine opens upstream replication session S
       |
       v
IDENTIFY_SYSTEM on S --> mismatch --> close S, return typed refusal
       |
      match
       |
       v
private Unix socket <--> session S <--> original source connection
       ^
       |
pg_receivewal --no-loop --synchronous
       |
       v
existing local queue --> existing uploader --> existing storage publication
```

The child connects only to the private socket. Local socket traffic does not use TLS; upstream TLS/mTLS, certificate verification and authentication remain enabled exactly as configured by the physical engine. Never replace upstream verification settings with the local socket's connection settings. Keep credential material out of bridge diagnostics and arguments.

Each receiver attempt owns one upstream session and one local connection. Reject additional local connections. A process retry cannot reuse an unchecked connection or fall back to the original host. Keep `--no-loop`; a network disconnect returns to Databasus supervision, which creates and verifies the next upstream session. This also bounds behavior if the utility attempts another local connection unexpectedly: it cannot obtain unverified upstream access.

**Alternatives rejected:** a SQL preflight leaves a connection-routing race; pinning the resolved IP does not identify a server behind a proxy; a custom receiver would take over WAL file writing and timeline behavior; patching the bundled utility would create a maintained binary fork across versions and architectures.

**Ownership:** protocol handling belongs to `internal/engines/postgresql/physical`; the backups module supplies the expected identity and decides lifecycle outcomes. This follows `docs/architecture.md:51` and `:153` and `backend/AGENTS.md` (Where code goes; Engines and providers). Document the bridge in the physical-backup section of `docs/architecture.md` during implementation.

### 2. Tie bridge, child process and refusal to one attempt

Continue starting the native child through `lib/process`. Run forwarding work through the context-bound background helpers. A single 30-second setup deadline covers connecting, authentication, identity response, local startup and all subsequent native setup commands until the first upstream `CopyBothResponse`. That response disarms the deadline for ongoing streaming. Expiry cancels the attempt, closes the sockets and stops the child under the existing interrupt/kill policy; the result is retryable while the caller remains active. Closing or canceling an attempt closes both sockets, stops the child, waits for forwarding work, and removes its private socket directory before a subsequent receiver starts. Reuse the existing interrupt/kill policy rather than introducing another process lifecycle.

Move the current process-start callback's meaning to verified streaming readiness. An authenticated session that has not entered replication is not a running receiver. A disconnect resets the live flag through the existing receiver supervision cleanup. Preserve application-name-based slot ownership and existing restart counters.

Represent identity mismatch as a typed engine error carrying expected and actual identifiers. Classify it independently from retryable transport errors, recycled-WAL resume mismatch and generic fatal stderr. Do not demote an observed identity refusal when a later bastion probe happens to fail. Return it to the manager without entering resume repair or rebuild escalation. On every later manager reclaim, connection identity is verified again.

**Alternatives rejected:** stderr substring matching can confuse utility errors and transport outcomes; signaling readiness when the process starts reports an unverified source as active; letting a child outlive its bridge can keep a slot held or overlap the next attempt. Existing `docs/architecture.md:207` and `backend/AGENTS.md` (Long-running work) require context-bound work and centralized process ownership.

### 3. Guard SQL observations and WAL slot mutations on their own sessions

Add a focused engine operation for opening a physical-source inspection with an expected identity, reusing the cluster-identity query on that very SQL connection. Streamer slot probes must use it before interpreting slot state or recording source-side metrics. A probe that reaches B must not report B's slot as A's lost or stolen slot.

WAL slot creation, drop/recreate during rebuild, and termination of a purportedly owned slot backend must use verified inspections. Retain each verified inspection through its corresponding read and mutation. When an operation opens another connection, verify that connection independently. Do not authorize a later connection using a prior probe result. Keep deletion-race cleanup on the same guarded path when startup discovers that its Databasus database was deleted.

Keep this API focused on continuous-WAL slot operations; existing backup-slot cleanup and unrelated deletion services need not be redesigned. Do not introduce optional expected-identity arguments that silently bypass the guard in a WAL caller. Replace the streamer's unguarded calls outright.

A typed mismatch from any companion loop must stop the owned run and reach the manager as the identity cause. Do not merely log it, map it to an unavailable source, or lose it when internal cancellation stops the receiver. Preserve the cause separately from ordinary parent shutdown so the manager records a failure, not a successful deliberate stop. A concurrent rebuild still checks identity before every mutation, even if another loop has already requested cancellation.

**Alternatives rejected:** only checking the WAL connection leaves SQL-side slot damage possible; one startup check cannot authorize later rebuild connections; using a global boolean that a source was once verified does not bind identity to an operation.

### 4. Persist refusal while retaining the original queue and retry policy

Add a nullable `error_reason` column to the WAL streamer row, initially null for existing rows. Store `SYSTEM_IDENTIFIER_MISMATCH` with the claim-fenced failure transition and existing `failing_since`. Reclaim, heartbeat, normal shutdown and an unsuccessful later connection must not erase an unresolved mismatch. A successful old-queue upload or first periodic sample must not clear it.

Disabling streaming cancels its work while retaining the claim row. Its terminal identity refusal can therefore still be persisted after companion work drains. This avoids making configuration requests wait for shutdown and prevents a concurrent disable from deleting an as-yet-unrecorded refusal. The backups module still removes streamer state transactionally during database-deletion cleanup, before the database's restrictive foreign key permits deletion.

Clear the identity reason only when the currently owned receiver attempt reaches verified streaming readiness. Fence updates by the current claim and attempt lifecycle so an obsolete callback cannot clear a newer failure. Preserve the existing treatment of generic failures when no identity refusal is pending. Stream health reads this persisted reason before its collecting-data grace period or two-sample rules. Existing public running states stay unchanged; an owned startup attempt is distinct from a live receiver.

An identity refusal does not set the manager's queue-removal flag. Stop the receiving run safely and retain completed A segments in both the ordinary queue and `pending-upload`; already committed A uploads remain usable. Preserve the current treatment of disposable partial files, whose presence is not proof of a completed valid segment. Recovery of legitimate A segments may occur before the next receiving attempt, but it cannot establish identity for that attempt.

Use the existing manager reconciliation interval to retry. After A returns, verify the next session and resume the existing slot/resume path. Do not request a new FULL merely because B was refused. If existing backup execution separately recorded an unresolved identity mismatch, its current manual-FULL recovery requirement still applies; stream recovery does not erase that backup outcome.

**Alternatives rejected:** an in-memory reason disappears on process exit; generic `STREAMING_STOPPED` cannot identify the actionable cause; automatic queue deletion can destroy the original recovery point; forcing a manual action for WAL retry would add a new product workflow. Normal retention and explicit deletion remain independent of identity refusal.

### 5. Extend existing diagnostics without adding an operation or screen

Add `SYSTEM_IDENTIFIER_MISMATCH` to `WalStreamReasonCode` in `backend/api/openapi.yaml`, regenerate the server, test client and frontend schema, and expose it as health `BROKEN` through the existing statistics operation. Keep existing live-state fields, streamer enums and request shapes. No new sample-state enum is needed: an identity-refused inspection contributes no trusted source metrics, and the persisted reason carries the diagnosis.

Use existing chain-failure notifications and throttling, including the stable code and guidance to restore routing to the original cluster. Expected and actual identifiers may appear in structured logs; credentials, full DSNs and backend error text must not become interface copy. Display the new reason and recovery guidance through the existing monitoring reason tables, source-stage mapping and all seven locale dictionaries. Keep error text translated at render time so changing language updates an active mismatch.

**Alternatives rejected:** pretending B is unreachable hides a known identity failure; exposing a new endpoint or state is unnecessary; surfacing only backup failures misses a WAL-only refusal. The enum addition is the only public API expansion because existing WAL reason values cannot express this result. Strict clients with exhaustive response-enum handling need to recognize the new value. No compatibility alias or alternative endpoint is added.

Apply the API-design skill and `backend/AGENTS.md` (Handlers, services, repositories; Errors), and follow `frontend/AGENTS.md`, `docs/frontend-localization.md` and `docs/frontend-design.md` for existing UI mapping and translated guidance.

### 6. Use a switchable real endpoint and restore actual bundles

Create one reusable test-support TCP proxy for engine, API and physical recovery scenarios. It routes each new connection based on its startup replication parameter, records its chosen destination and connection identity, and can close selected old streams without changing the listening address. Plaintext proxy fixtures can inspect startup and replication commands; production TLS/mTLS remains covered by real secured engine/API tests. Add a scripted per-connection route for slot tests that must change destination between a probe and a later mutation.

Create independent real clusters A and B with different identifiers, working credentials, matching timeline numbers, overlapping available WAL and the required same-named slot. Observe successful replication authentication and the `IDENTIFY_SYSTEM` result from B on the refused connection, not just a TCP accept. Record absence of `START_REPLICATION` on that connection after the authoritative refusal. In split-routing tests SQL continues to reach A while replication reaches B, so the SQL slot guard cannot mask a missing WAL-session guard. Run this at initial reception and after cutting a real existing A stream.

Begin with an API reproduction that compiles against current contracts and fails because foreign WAL is accepted, before adding fields or enum values needed by the fix. After the fix, temporarily replace the actual-session guard with an independent SQL precheck and rerun the split-routing cases; they must fail for the foreign-source behavior. Restore the guard and rerun the complete affected packages. Do not add a production bypass setting or mock the engine to obtain that evidence.

For queue recovery, make storage unavailable after valid A WAL is received, wait for completed segments in both queue locations as applicable, and stop Databasus normally. Keep its database, queue root, storage and configured identity while routing the endpoint to B for restart. Verify refusal without queue reset. Then return A, restore storage, wait for verified reception and queued uploads, and recover a selected pre-refusal A point from a newly downloaded bundle. Check control rows, preserved archived A content and the absence of B control rows. Compare queue/storage snapshots by content where filenames collide; unchanged catalog counts alone are insufficient. Allow only accounted-for legitimate A publications during refusal.

For promotion, use `StartPhysicalPrimaryWithStandby`, wait for control rows and the restricted backup role to replay, and compare primary/standby identifiers before switching. Promote the real standby and reroute the unchanged endpoint without updating database settings. Verify WAL reception on the new timeline, rejection of an incremental against the old root, the existing replacement FULL behavior, and recovery from that new FULL with the shared real download/reconstruction helpers. No cross-timeline PITR claim follows from this test.

Run recovery scenarios for PostgreSQL 17 and 18 through their existing end-to-end entry packages. Use `physicalapitests/walstreaming/team_test.go` for API coverage and `internal/app/apptest/physicaltest/` for actual recovery, extracting only shared fixture pieces that these callers require. Wait for protocol events and application conditions through `testwait`, not sleeps; clean up only the test's own sockets, processes, objects and containers.

**Alternatives rejected:** authentication or missing-slot failures never exercise identity enforcement; independent clusters with different timelines can fail for the wrong reason; reconfiguring the endpoint bypasses the reconnect scenario; asserting catalog rows without downloading and recovering does not prove archive usability. This follows the testing skill and `docs/testing.md` (Layers; Checking work).

## Risks / Trade-offs

- **Local protocol adaptation could change utility behavior.** Preserve upstream startup parameters, use existing protocol codecs, and test real native receiver setup, server rejection, EOF, cancellation, back pressure, synchronous feedback and timeline following on both supported versions.
- **Taking a raw connection while buffered data remains can corrupt protocol forwarding.** Synchronize before handoff, transfer ownership once, and exercise fragmented/withheld responses and cancellation against real PostgreSQL.
- **A bridge could weaken upstream security or consume an extra WAL sender.** Keep configured authentication/TLS on the one upstream session and verify real mTLS, SSH-tunnel and constrained-WAL-sender scenarios; do not keep an extra identity connection open.
- **Concurrent probes, rebuilds or late callbacks could hide refusal.** Propagate a typed terminal cause, guard each mutation on its own session, and keep failure updates claim-fenced until a current verified stream starts.
- **Queue uploads can legitimately change catalog/storage snapshots during refusal.** Account for original A segments by content and prove B's session never enters replication; do not require the whole catalog to freeze.
- **Same identifiers do not establish safe merging of divergent timelines.** Retain the existing incremental timeline barrier and explicitly keep split-brain/history compatibility outside this change.

## Migration Plan

1. Add the nullable streamer reason column through the normal startup migration. Existing streamer rows remain valid without synthetic historical failures; do not change any configured source identifier or archive content.
2. Regenerate API types and ship backend and frontend together. Document the new response reason for clients with exhaustive enum handling.
3. New and reclaimed receiver attempts use the guarded path. Ordinary deployments already stop their managed children before restarting; no data migration or receiver bundle replacement is required.
4. An ordinary downgrade to an older binary is unsupported: `docs/architecture.md:146` and `backend/internal/lib/database/migrate.go:22` require startup refusal when the database's newest migration exceeds the binary's newest migration. If an implementation rollback is necessary, build a rollback release that retains this change's migration while reverting its runtime implementation. That build can leave the additive column unused, but it removes source-session protection. Preserve the migrated schema and all original backup/queue data; do not reset identifiers or delete archives as part of rollback.

Implementation updates `docs/architecture.md` with the bridge's security boundary, slot guards, persisted diagnostics and recovery procedure. Main specs are synced through the OpenSpec lifecycle. No agent-instruction files need a new rule.
