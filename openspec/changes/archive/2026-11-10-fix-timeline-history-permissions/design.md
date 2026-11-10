## Context

See `proposal.md` for motivation and scope. The restricted-role reproduction ran before production changes on PostgreSQL 17 and 18. Both versions logged `ERROR: permission denied for function pg_read_binary_file (SQLSTATE 42501)` from the history publisher and failed the wait for a published history row. The failure is reproduced, rather than inferred from a timeout. After replacing acquisition, both promotion suites pass, including startup of the restored database and a marker written after the replacement FULL.

Current integration points:

- `backend/internal/engines/postgresql/physical/timeline_history.go:8` reads history with `pg_read_binary_file` on an ordinary SQL connection.
- `backend/internal/modules/backups/internal/physical/walstreaming/wal_watch_dir.go:73` parses a local history filename, opens SQL inspection and calls the common publisher; it removes the local file only after success.
- `backend/internal/modules/backups/internal/physical/backuping/full_backup.go:55` calls the same publisher after a FULL stream completes. Lines 62-97 preserve a valid FULL when the additional history operation fails.
- `backend/internal/modules/backups/internal/physical/timelines/history_upload.go:59` returns an existing row before fetching history. Lines 157-189 validate the publication fence for new rows, spend write receipts and handle a concurrent unique insert.
- `backend/internal/modules/backups/internal/physical/timelines/timeline.go:145` skips validation when no history exists but can reject a FULL solely because its timeline has no row. This presence check must not override the intended independence of a valid FULL from separate history acquisition.
- `backend/internal/modules/backups/internal/physical/chains/restore_set_service.go:41` resolves reach before loading all history rows. `restore_coverage_service.go:45` uses the same reach calculation but has no history prerequisite.
- `backend/internal/engines/postgresql/physical/source.go:101` already opens physical replication connections using the shared credential and TLS configuration.
- `backend/internal/app/apptest/physicaltest/promotion.go:65` already exercises replacement FULL, download and actual startup on both supported versions, but uses administrator credentials and checks no marker written after the new FULL.

The implementation follows root `AGENTS.md`, `backend/AGENTS.md`, and `docs/architecture.md` sections "Engines", "Stored files", "Storage switches" and "Physical backups". PostgreSQL protocol and history parsing belong to the engine; publication, storage generations and recovery selection belong to backups. No engine code imports module tables, storage services or HTTP handlers.

## Goals / Non-Goals

**Goals:** Use one authenticated history acquisition method for both callers; bind cluster verification to the connection that obtains the bytes; preserve file ownership through retries and configuration changes; make recovery reporting agree with the delivered bundle.

**Non-Goals:** Treat local history contents as a second source, implement cross-timeline replay, redesign streaming probes, introduce new HTTP fields or monitoring reasons, or migrate the history catalog.

## Decisions

### 1. Acquire history through the physical replication connection

Move history acquisition from `Inspection` to `Source`. Use the existing `connectReplication` path and low-level `pgconn` simple-query execution. Pass a named request containing the expected system identifier and a numeric timeline ID. Return the validated filename and raw history bytes. Remove the SQL reader and update every caller; retain no SQL fallback.

Both the WAL watcher and FULL completion call the same publisher, which obtains history from `Source`. The watcher uses its local file only as a pending notification; FULL completion does not wait for that file or require continuous WAL reception.

```text
local history notification --+
                            +--> Source history request
FULL result ----------------+             |
                                          v
                               physical replication connection
                                          |
                                   IDENTIFY_SYSTEM
                                          |
                                verify expected system ID
                                          |
                                 TIMELINE_HISTORY N
                                          |
                                validate name and content
                                          |
                              common history publication
```

The alternatives are reading the local file and granting access to the SQL reader. Local-only acquisition would couple FULL completion to another process's queue and require durable provenance for retained files. Additional SQL privileges violate the accepted role constraints. A hybrid adds two mechanisms and a precedence rule without serving an additional caller.

### 2. Verify identity and response before storage writes

Require a nonempty expected cluster identifier for stored-backup history work. On one connection, issue `IDENTIFY_SYSTEM`, compare its system identifier with the stored identifier, and then issue `TIMELINE_HISTORY` for the numeric requested timeline. Do not verify identity on an earlier SQL connection and then trust a new connection.

Reject timeline 0 and timeline 1 as history requests, future timeline IDs, a missing or malformed protocol result, an unexpected filename, empty history and malformed ancestor entries. Timeline 1 needs no history and is skipped by callers. Preserve content bytes exactly; response fields are raw bytes despite their text labels. Parse history only for validation: ignore PostgreSQL-supported blank/comment lines, require at least one ancestor, require positive strictly increasing ancestor timeline IDs below the requested timeline, and require valid nondecreasing fork LSNs. A reason field is descriptive text, not a fixed string. Equal fork LSNs must remain valid for promotions without intervening writes.

A request for earlier history may succeed after another promotion; equality between requested and current timeline is not required. A history filename and parser do not establish cluster provenance. That comes from the checked replication session.

Separate identity checks leave a connection-routing race. Trusting only filenames accepts the same name from another cluster. Rejecting all older timelines would strand pending history after another promotion.

Use a 30-second child context for connection, identification and history retrieval, bounded by the caller's deadline. Close connections on success, error and cancellation with bounded cleanup. Do not hold one across the streaming session. Exhausted WAL senders are an acquisition failure and use the same later retry as other transient failures. This consumes one additional sender during acquisition but creates no replication slot and changes no source setting. An unbounded wait could stall WAL upload scanning; reconnecting until success inside one call would conceal that failure.

### 3. Retain publication and storage-switch ownership

Keep compression, optional encryption, artifact/metadata writes, per-attempt object identities, write receipts and discarded-file cleanup in the existing common publisher. Validate source provenance before reusing a catalog entry. Reuse an entry only under a transaction that validates the captured publication fence and confirms its storage matches the captured selection. The concurrent-insert recovery path performs the same validation and lookup rather than returning an unchecked winner or a nil success.

A storage switch deletes the old catalog and advances the generation. An old attempt cannot publish or acknowledge success against the replacement generation. Its failed artifact and metadata remain owned by cleanup. The current streamer can refetch history independently even if the old local queue was removed. A same-generation duplicate returns the winning entry, and cleanup removes only the losing attempt's pair.

Leaving the early return unchanged would bypass the generation check. Reusing deterministic object names would let cleanup from a failed attempt target a successful retry. New catalog columns or a separate history cache are unnecessary for this change.

### 4. Keep backup validity separate from archive replay readiness

Preserve an otherwise valid FULL as `COMPLETED` if its separate history attempt fails. Adjust the touched presence-only history validation so absence of a separately published row alone cannot mark that FULL `CHAIN_BROKEN`. Retain the existing live cluster/timeline compatibility checks and genuine timeline failures; do not broaden cross-timeline recovery.

For archive-based restore selection on a timeline above 1, require the selected timeline's verified history row under the current storage selection before extending reach through archived WAL. Apply this once in the shared reach computation used by restore selection and coverage reporting. Timeline 1 bypasses the prerequisite.

When history is absent, reach stops at the latest included completed backup, the WAL run is empty, and coverage contains only the completion points of completed backups in the selected chain. A latest database restore returns that backup chain without additional WAL or history. A target beyond its completion gets the existing `409 target_time_unreachable`. An explicitly selected completed backup keeps the existing per-backup path and needs no separately archived history. When history publishes later, existing WAL extends recovery on the next request without another FULL.

For archive replay, include the selected timeline's history rather than every history row of the database. Revalidate the same prerequisite when a token is redeemed; a targeted request that lost history fails before streaming starts. Do not invent a WAL gap or a new monitoring reason for pending history.

This is a conservative publication prerequisite, not a claim that every PostgreSQL startup intrinsically needs a separately archived copy: a base backup may contain enough information for its own recovery. It makes archive replay readiness explicit while retaining that backup. Marking the FULL failed discards useful recovery; reporting later WAL before required publication overstates readiness. Inferring history availability from the completed FULL status recreates the same ambiguity.

The public contract retains its response shapes and codes. Clarify descriptions of target reachability and coverage in `backend/api/openapi.yaml` and regenerate the affected generated artifacts using `mise run generate-api`; do not add an endpoint, enum value or frontend copy.

### 5. Extend existing real PostgreSQL fixtures and public API scenarios

Create the backup role through the public API using primary administrator settings, wait for the role and password change to replay to the standby, and use the issued credentials for all subsequent backup work. Source setup, marker writes and promotion may use the administrator; that connection must not supply backup or history acquisition credentials. Do not call role creation against an unpromoted standby.

Extend `TestPhysicalBackupAfterPromotion` in both `backend/internal/modules/backups/e2e/postgresqlphysical/pg17` and `pg18`. Reuse shared setup in `internal/app/apptest/physicaltest`, compare the decompressed published history with the source history read by the fixture administrator, check its metadata and sole catalog entry, then create a replacement FULL, write a later marker, archive its WAL, download the restore set and start PostgreSQL. Verify all markers in the running restored database. Retain a post-promotion FULL-only startup scenario to exercise acquisition without a receiver.

Before production changes, run the restricted-role scenario and record the SQL permission error causing the missing history. Check role flags, memberships and function privileges rather than assuming `rolsuper=false` proves the role is restricted. Assert no server-file function execution or broad role membership was added and keep a protected application table to prove SQL access is denied.

Use engine tests against real PostgreSQL for successful requests, identity mismatch, older requested timelines and connection cancellation. Use pure parser cases for malformed content and protocol-result validation, without mocking our modules. API scenarios cover concurrent publication, write failure, storage generation changes, missing history, token redemption and coverage. Existing recovery tests and helpers are reused; a cascading-standby variant of the engine fixture exercises two promotions. Catalog-only assertions and admin-only tests cannot establish the requested behavior.

## Risks / Trade-offs

- Additional WAL sender capacity is temporarily required. A source with every sender occupied may defer history publication and archive replay availability. Bound the attempt, close it promptly, retain pending work, and document the extra connection; do not stop the receiver or grant privileges to work around exhaustion.
- A short acquisition timeout must not become a machine-speed assertion. Tests wait for observable completion or cancellation with shared test waiting helpers.
- Historic catalog rows have no new provenance column. Existing rows continue to be trusted as cataloged artifacts; the stricter acquisition validates new publications. Correct old misattribution or corrupt storage is outside scope.
- Failed history acquisition may temporarily reduce WAL recovery coverage on an existing promoted timeline. Explicit backup restore remains available, and a later publication restores coverage without a new FULL.
- FULL completion and the receiver can race. Shared publication, generation validation and unique-entry handling must be verified together; a unique constraint alone is insufficient.

## Migration Plan

No schema or source-privilege migration is needed. Deploy through the normal release. Missing history is acquired by the receiver's retained notifications or the FULL path; there is no bulk source scan or new periodic job. Update `docs/architecture.md` for the acquisition flow, sender use, publication prerequisite and explicit-backup behavior. Search documentation for server-file privilege guidance and correct any affected statements without adding grants.

Rolling back the binary restores the old acquisition and reporting behavior, including the reproduced restricted-role failure. Stored artifacts retain their existing formats and encryption. A rollback is not a reason to retain the SQL reader in the implementation.

Protocol references: [PostgreSQL 17 streaming replication](https://www.postgresql.org/docs/17/protocol-replication.html), [PostgreSQL 18 streaming replication](https://www.postgresql.org/docs/18/protocol-replication.html), and [server-file SQL functions](https://www.postgresql.org/docs/18/functions-admin.html#FUNCTIONS-ADMIN-GENFILE).

## Implementation verification

The initial restricted-role promotion run failed on both PostgreSQL 17 and 18 with SQLSTATE `42501` from `pg_read_binary_file`. The replacement protocol reader passed the same promotion scenarios.

The following checks passed:

- `mise run test`: all backend groups and 194 frontend test files with 2,864 tests.
- The complete `backend/internal/engines/postgresql/physical/...` and `backend/internal/modules/databases` packages, the affected physical API suites (`backuping`, `walstreaming`, `restoring`, `streamstatistics`), and the internal `timelines` and `backuping` packages.
- Both complete physical recovery suites, `backend/internal/modules/backups/e2e/postgresqlphysical/pg17` and `pg18`, including restricted-role promotion, the post-FULL WAL marker, and FULL-only recovery after promotion.
- `TestGetTimelineHistoryCancellation` uses a TCP proxy over real PostgreSQL 17 and 18 to withhold each acquisition response, then verifies cancellation, client closure and WAL sender release without forwarding that response. Login and storage failure fixtures install the failure before promotion and wait for role replay.
- The final `TestGetTimelineHistory` run on both versions and `TestPhysicalFullHistoryFailure` after two promotions with an already published earlier history.
- `mise run backend:lint`, `pnpm --dir frontend lint`, `pnpm --dir frontend typecheck`, and `pnpm --dir frontend build`.
- The concurrency test observes history failure events from both callers after the FULL finishes and the receiver acknowledges publication. Temporarily disabling unique-conflict recovery made it fail at `both initial history publishers must succeed`; the publisher was restored exactly, and three normal runs with `-race` passed. The promotion and FULL-only recovery scenarios also passed again on PostgreSQL 17 and 18.
- Three consecutive runs with `-race` passed for `TestGetTimelineHistoryCancellation`, `TestPhysicalTimelineHistory` and `TestPhysicalTimelineHistoryConcurrency`; the updated backend passed lint with zero issues.
- `TestPhysicalTimelineHistory` now also refuses only history metadata uploads through an HTTP proxy over real MinIO. It verifies the artifact exists before the metadata failure, no history row publishes, the local notification remains, cleanup deletes the abandoned artifact before repair, and retry publishes a different artifact with matching metadata and exact source bytes. All four failure variants passed three consecutive runs with `-race`.
- `TestGetTimelineHistory` snapshots attributes, memberships (including grantor and admin/inherit/set options), file-reader/WAL-rotation function execution, and protected-table/column privileges for every role before acquisition. An unrelated login retains its pre-existing pg_monitor membership and SELECT grant. The entire snapshot is compared after success and after each identity, invalid/future timeline and pre-canceled request failure. `TestGetTimelineHistoryCancellation` also compares the full snapshot after interrupting each withheld `IDENTIFY_SYSTEM` and `TIMELINE_HISTORY` response on both versions. The complete test passed on PostgreSQL 17 and 18 with `go -C backend test -count=1 -timeout 20m -run '^TestGetTimelineHistory$' ./internal/engines/postgresql/physical`; backend lint reported zero issues.
- `TestHistoryReuseStorageGeneration` holds the old receiver's history query at the real PostgreSQL boundary before any reuse lookup. A peer switches A to B and back to A, then publishes current-generation history through FULL while that acquisition is held. After release, the old receiver must report failure, retain its notification and leave the winner unchanged. Removing only `Publication.Validate` from `findPublishedHistory` made the test fail at `an old generation must not acknowledge the current generation's history`; the publisher was restored exactly. This isolates generation validation from the separate storage-ID check and pre-insert fence.
- Three consecutive runs with `-race` passed for `TestGetTimelineHistory`, `TestGetTimelineHistoryCancellation`, `TestHistoryReuseStorageGeneration` and `TestLateHistoryStoragePublication` (engine package 50.512 seconds, WAL API package 68.156 seconds). This includes both PostgreSQL versions for active cancellation and both publication stages for storage switching.
- `openspec validate fix-timeline-history-permissions --strict` and `git diff --check`.

No source privileges, database schema, dependency pins or CI workflows changed. The working tree is left uncommitted for implementation review.
