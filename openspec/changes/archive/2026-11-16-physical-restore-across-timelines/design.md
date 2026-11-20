## Context

See `proposal.md` for the motivation and `specs/` for the required behavior. This section covers only the current mechanics the design has to change.

- **Chain WAL is one timeline.** `ResolveRestoreSet` reads WAL with `FindByChainSpan(..., chain.RootFull.TimelineID, ...)` (`backend/internal/modules/backups/internal/physical/chains/restore_set_service.go:99`). It attaches only the root FULL timeline's history (`restore_set_service.go:109`). `whereInChainSpan` filters on one `timeline_id` (`backend/internal/modules/backups/internal/physical/catalog/wal_segment_repository.go:104`). The chain end time used by retention ordering and the grace check comes from `GetLatestReceivedAtInChainSpan` (`wal_segment_repository.go:77`, called from `chains/view_service.go:72`) over the same single-timeline predicate. Restore coverage and the monitoring statistics call the same `resolveRestoreReach`, so they already agree with restore selection (`chains/restore_coverage_service.go`).
- **Orphan WAL is "no completed FULL on the same timeline".** `FindOrphans` (`wal_segment_repository.go:136`) drops every promoted-timeline segment that precedes the first FULL on that timeline. `DeleteOrphanWalSegmentsInSpan` only holds off while a FULL runs (`chains/service.go:90`). History deletion looks only at the FULL's own timeline (`chains/service.go:501`).
- **History arrives separately from WAL.** The streamer publishes `.history` from its watch directory on a ticker and retries on failure (`walstreaming/wal_watch_dir.go:58`), independently of segment uploads. Promoted-timeline segments can reach the catalog before their history row exists.
- **INCREMENTAL compares the live timeline with the root FULL's.** The preflight calls `CheckIncrementalTimelineCompatibility` with `rootFull.TimelineID` (`backuping/incremental_backup.go:67`). The stream-failure recheck compares against the same root timeline (`backuping/executor.go:353`). `timelineRefusalResult` maps a newer live timeline to `CHAIN_BROKEN` / `TIMELINE_SWITCH_DETECTED` (`backuping/executor.go:253`), and the scheduler turns that into an immediate replacement FULL (`backuping/scheduler.go:225`). The FULL path already compares a stream failure with the timeline captured at its preflight (`classifyFullStreamFailureAfterIdentity`, `backuping/executor.go:326`). A summarizer refusal maps to `CHAIN_BROKEN` with its own reason and gets no immediate FULL (`backuping/summarizer_precheck.go:46`).
- **History publication reuses any existing row.** `UploadHistoryFile` returns the published row for `(database, timeline)` without comparing content (`timelines/history_upload.go:73`). `validateTimelineHistory` already walks every entry and ends with the last ancestor timeline and position in local variables (`backend/internal/engines/postgresql/physical/timeline_history_validation.go:11`).
- **The receiver realigns only recycled WAL.** `isResumeMismatchError` matches "has already been removed" (`backend/internal/engines/postgresql/physical/wal_receiver.go:243`). A start LSN beyond the promoted node's fork point fails another way and is retried unchanged.
- **The recovery script sets no target timeline.** It writes `restore_command` and `recovery_target_time` only (`physical/restorestream/recovery_script.sh:876`), so PostgreSQL uses its default `latest`.
- **Restore selection reads only the catalog.** `ResolveRestoreSet` performs no storage I/O, and `docs/architecture.md` ("Physical backups", "Restore") keeps that property for tokens, downloads and coverage.

Constraints: `backend/AGENTS.md` places code that knows PostgreSQL but no tables in `internal/engines/postgresql`, and domain rules with their data in the backups module. Repositories change one column per `UPDATE` and run through `database.Conn`. Migrations follow "Data" and "Migrations" in `docs/architecture.md`, including invariants enforced by the schema. Names follow the root `AGENTS.md`: the glossary term "fork point" is used in prose and code, and its WAL position is the fork point's LSN (`ForkLSN`, column `fork_lsn`).

## Goals / Non-Goals

**Goals:**
- One definition of a chain's timeline line, computed from the catalog, shared by restore selection, coverage, the chain end time, the orphan sweep, cascade deletion and history deletion.
- PostgreSQL does the replay across fork points and the block-level combination across timelines; Databasus only selects files and checks preconditions.
- A native prototype settles the PostgreSQL facts the design relies on before product code changes.

**Non-Goals:**
- Decoding the fork segment's transaction timestamps more finely than per segment.
- Backfilling fork points for history rows published before the upgrade.

## Decisions

### D1. The fork point is stored in the catalog when history is published

Each `backups_physical_wal_history_files` row gains `parent_timeline_id` and `fork_lsn`, both nullable, with `CHECK (num_nonnulls(parent_timeline_id, fork_lsn) IN (0, 2))`, and `is_conflicting BOOLEAN NOT NULL DEFAULT false`. `validateTimelineHistory` is changed to return the last entry it already parses as a `TimelineFork{ParentTimelineID, ForkLSN}`, so the engine has one parser for the format. Publication stores both values. A chain's line is then a walk over catalog rows: from the newest usable timeline back through parents.

Alternatives:
- *Read the `.history` artifacts from storage during selection.* It breaks the "catalog only" property of restore selection, adds decryption and storage latency to every coverage request, and makes coverage fail when the storage is slow.
- *Store the full ancestry on each row.* It duplicates the parent rows. Parent links already give the full line, and the conflict check (D7) keeps each link trustworthy.
- *A second parser beside the validator.* It would duplicate the walk over the same format, and the two could drift.

Rows published before the migration keep both columns empty, and their fork point is unusable, which is the current behavior. A backfill would have to download and decrypt every stored history. The windows those rows could open were mostly deleted by the current orphan sweep, so the backfill buys little.

### D2. The newest line and a FULL's line

A link is *usable* when its row has a fork point, is not marked conflicting and lies in the storage of the FULL being considered. The *newest line* of a database, for a given storage, starts at the highest timeline whose row is usable, or at timeline 1, and follows usable parent links down to the first timeline whose link is missing or unusable. It is a list of `TimelineLineRange{TimelineID, StartLSN, EndLSN}`: each timeline the line leaves ends at the fork point's LSN of its child, and the newest timeline is open-ended.

A FULL's line is the newest line from the FULL's timeline and start LSN onward, provided the FULL's timeline is on the newest line and its start LSN is before the fork point's LSN where the line leaves that timeline. Otherwise the FULL is *off the line*: a FULL taken on a former timeline after the fork point, or on an abandoned branch. Its own line is its timeline from its start, open-ended. Restore by time and latest restore use only FULLs on the newest line. An off-line FULL stays restorable by its identifier, and retention keeps its own timeline's WAL.

The type `TimelineLine` and its builder `BuildTimelineLine` are pure functions in the backups module's `chains` package, next to the restore set. They take the database's history rows, which are few, and need no storage access.

Alternatives:
- *Follow the live timeline reported by the source.* Restore selection would depend on a source connection and stop working while the source is down, which is when restores matter.
- *Let the user choose a branch.* This adds an API field and a UI for a case that pgBackRest also leaves to `latest`. It stays out of scope.
- *Let an off-line FULL's line cross into the newest timeline.* Its data includes the former primary's lost writes, so replay onto the promoted timeline would mix branches.

### D3. WAL lookups take the line's ranges

`FindByChainSpan` and `GetLatestReceivedAtInChainSpan` are replaced by `FindByTimelineLine` and `GetLatestReceivedAtOnTimelineLine`, which take the chain's span cut to its line and run the existing indexed predicate once per range. On a timeline the line leaves, a range keeps only segments that start before the fork point's LSN, so the last one is the segment holding that LSN when it falls inside a segment. On the following timeline, the range starts at the segment whose range contains the LSN. Both copies of a fork segment that splits inside it are therefore selected. The chain end time (`GetChainEndTimestamp`) uses the line too, so WAL received on the promoted timeline moves a closed chain's end time and grace period.

`contiguousWalRun` walks the ranges in order. Contiguity across a fork point holds when the parent range reaches the start of the segment holding the fork point's LSN and the child range starts at that segment. The parent range need not hold the parent copy of the fork segment: a receiver that follows the promotion leaves that copy as `.partial`, Databasus archives only complete segments, and replay reads the child copy, which carries the parent's WAL up to the fork point (D10).

Alternatives:
- *One SQL query with a `VALUES` list of ranges.* Fewer round trips, but a line rarely has more than two or three timelines, and one query per range reuses the indexed predicate without a new SQL shape.
- *Ship only the child copy of a fork segment.* PostgreSQL most likely reads the fork segment from the child timeline, which makes the parent copy unused. The prototype (D10) checks this. Shipping both costs one segment and keeps replay working if the check fails on some version.

### D4. Reach time ignores the parent copy of a fork segment

A segment's `last_transaction_at` is its newest commit. The parent copy of a fork segment can include the former primary's lost commits after the fork point. The reach computation therefore takes no time evidence from a parent-timeline segment that holds a fork point's LSN. The child copy of the same segment carries the parent's commits up to the fork point, so no real evidence is lost.

Alternative: *decode commits up to the fork point's LSN.* It is more precise only inside one segment and needs a second decoding path. Rejected.

### D5. Retention asks the line, not the timeline

The SQL prefilter, `FindOrphanCandidates`, leaves out a segment anchored by a completed FULL on its own timeline unless it starts at or after a usable child link's fork point's LSN. A segment before every such fork point that a same-timeline FULL anchors is always on that FULL's line. Every remaining segment is checked in Go against the lines of the database's completed FULLs (D2), both when the sweep lists candidates and again under the cleanup lock before deletion. The former primary's WAL past a fork point therefore reaches the line check and is deleted unless an off-line FULL keeps it.

A segment on a timeline above 1 whose timeline has no history row and no completed FULL is *pending history* and is never an orphan. This covers the ticker gap between segment and history arrival. Once a FULL on that timeline completes, its own path has attempted history publication, and segments before it fall back to the current rule. Rows that are conflicting or lack a fork point do not protect their timeline's WAL, which is today's behavior.

Cascade deletion uses the same ownership span along the line. History deletion keeps a row while any surviving completed FULL has that timeline on its line. When it removes a timeline's history, it also removes that timeline's WAL that no surviving line holds, such as the segment holding the deleted FULL's start. Without its history that WAL would count as pending history and never be swept. All of this runs under the existing `AcquireBackupAndOrphanCleanupLock`, so a chain cannot start against history being removed, as "Scheduling and cleanup agree on chain state" in `docs/architecture.md` requires.

Alternatives:
- *Keep the SQL prefilter for every timeline.* It treats the former primary's WAL past a fork point as anchored by any earlier FULL on that timeline, so the WAL would never be swept.
- *A recursive SQL CTE over history rows.* It duplicates the line logic next to the Go version used by restore selection, and the two would drift.
- *Protect every promoted-timeline segment until its history arrives, without a bound.* A history that never publishes would keep WAL forever. Bounding the protection by the first completed FULL on that timeline matches the point where the FULL path has tried to publish.

### D6. INCREMENTAL continues when the line reaches the live timeline

`TimelineDecision` gains the kind `TimelineContinueAcrossPromotion`. The preflight compares the live timeline with the timeline of the chain's newest included backup, not with the root FULL's. When the live timeline is newer, the executor acquires and publishes the live timeline's history before deciding, and refuses with `CHAIN_BROKEN` / `TIMELINE_SWITCH_DETECTED` when publication fails or yields a conflicting or legacy link. This differs from `uploadHistoryForTimelineSwitch` in `backuping/full_backup.go:24`, which publishes after the stream and only logs a failure, so the INCREMENTAL path gets its own call. It then accepts the decision when the newest line from the chain's newest included backup reaches the live timeline and that backup's stop LSN is at or before the first fork point's LSN on the way.

The stream-failure recheck compares the live timeline with the timeline captured at preflight, as the FULL path already does with `preflightTimelineID`. A failure on a chain that crossed a promotion earlier is therefore not mistaken for a new promotion. A newer timeline than the preflight's yields `ERROR` / `FAILOVER_DURING_BACKUP`. The scheduler does not replace that INCREMENTAL immediately: the archived WAL already covers the window, so only restore time is affected, and the next INCREMENTAL runs at its configured time.

The summarizer precheck runs unchanged. Its `DecisionFullNewChain`, and any `pg_basebackup` refusal tied to the manifest's timeline or WAL summaries, are recorded as `CHAIN_BROKEN` / `TIMELINE_SWITCH_DETECTED` when the live timeline differs from the newest backup's. That keeps the existing immediate replacement FULL (`scheduler.go:225`) as the only fallback path.

Alternatives:
- *Keep a replacement FULL after every promotion and only add restore across fork points.* Simpler, but every switchover then costs a full copy of the cluster, and CNPG switches on every rolling update.
- *Retry the interrupted INCREMENTAL immediately.* It shortens replay for restores right after the switch, but needs new scheduler state for a case where recovery already works.
- *Let `SUMMARIES_EXPIRED` trigger the immediate FULL in general.* It changes the policy for ordinary summary expiry, which is out of scope.

### D7. Conflicting history is detected at publication

When `UploadHistoryFile` finds a published row, it compares the fresh history's parent timeline and fork point's LSN with the row's. On a match it reuses the row. On a difference it publishes nothing, sets `is_conflicting` on the row with one `UPDATE`, and logs a warning with the database and timeline only. The flag is never cleared. A row without a fork point is reused unchanged and not flagged, because there is nothing to compare; its link stays unusable.

Alternative: *compare full bodies through a stored hash.* The parent timeline and fork point's LSN are what selection uses; bodies that differ only in comments or in earlier entries already covered by parent rows do not change replay. The spec states the same rule.

### D8. The receiver realigns to the fork point

The engine recognizes PostgreSQL's refusal for a start LSN outside the server's history and reports the disposition `WalReceiverForkRealignment`, carrying the fork point's LSN from the server's history. The history is read through the same verified replication session as other history reads. The backups module's streamer then drops queued local segments of the former timeline that start at or after the segment holding the fork point's LSN, and restarts reception from that segment. PostgreSQL then sends the former timeline up to the fork point and continues on the new one. No slot rebuild or replacement FULL is requested. Former-timeline segments past the fork point that were already archived stay in the catalog off the line until the orphan sweep removes them (D5).

If the parent copy of the fork segment was already archived with lost commits, upload deduplication by `(database, timeline, start LSN)` keeps that archived copy. D4 takes no time evidence from it. Replay correctness depends on PostgreSQL reading the child copy for that segment, which the prototype checks (D10, step 1). If PostgreSQL reads the parent copy, the restore set must leave out the archived parent copy of the fork segment whenever the child copy exists, and D3 changes accordingly.

Alternative: *rebuild the slot and start from the server's current position.* It loses the promoted timeline's WAL from the fork point to the restart, which is the window this change exists to keep.

### D9. The recovery script pins the target timeline

The restore set records the newest timeline it contains. The script writes `recovery_target_timeline = '<n>'` whenever it ships WAL, and nothing when it ships none. The existing sanitizer already strips inherited `recovery_target*` settings (`recovery_script.sh:746`).

Alternative: *rely on `latest`.* It picks the highest `.history` the script happens to ship, so a stray file would change the replay path. An explicit value is deterministic and easy to assert in tests.

### D10. A native prototype comes first and stays as an engine test

Before product code changes, the engine test `TestTimelineCrossing` in `backend/internal/engines/postgresql/physical/` uses the isolated primary and standby from `backend/internal/lib/testcontainers/postgres_physical.go` and the pinned PostgreSQL client tools. It checks:
1. which files PostgreSQL requests through `restore_command` when it replays from a timeline 1 FULL to a timeline 2 target, and which copy of the fork segment it reads;
2. the server's refusal text for a start LSN beyond the fork point, and the content `pg_receivewal` writes for the fork segment after realignment;
3. `pg_basebackup --incremental` on the promoted standby against a timeline 1 manifest: acceptance with summaries, the refusal without them, and `pg_combinebackup` over the result.

Findings that contradict D3, D6 or D8 update this design before the dependent tasks start.

Findings, recorded on PostgreSQL 17 and 18. None contradicts D3, D6 or D8; three refine how they are built.

- **Replay reads the child copy of the fork segment.** Replaying a timeline 1 FULL and INCREMENTAL with `recovery_target_timeline = '2'` requested `00000002.history`, timeline 1 segments up to the one before the fork segment, then the timeline 2 copy of the fork segment and later timeline 2 segments. The parent copy of the fork segment, which held a commit the former primary made after the fork point, was never requested, and that commit was absent after recovery. D3 keeps shipping both copies when both are archived; D8's fallback of leaving out the parent copy is not needed.
- **The parent copy of the fork segment is often missing.** A receiver that follows the promotion stops the former timeline at the fork point, so its copy of the fork segment stays `.partial` and is never archived. Contiguity across a fork point therefore only requires the parent range to reach the start of the fork segment (D3).
- **The refusal names the requested timeline.** A start beyond the fork point fails with `requested starting point 0/7000000 on timeline 1 is not in this server's history`. The fork point's LSN comes from the entry for that timeline in the verified history of the server's live timeline, which is the last entry only when the server is one promotion ahead.
- **Reception restarts from a placeholder.** `pg_receivewal` resumes at the start of the newest `.partial` segment in its directory. After the queued former-timeline segments from the fork segment onward are dropped, the queue may hold no earlier segment, because uploaded segments are removed locally. The streamer therefore writes a zero-filled `.partial` copy of the fork segment on the former timeline after its usual removal of partial files. The prototype restarted reception this way; it wrote the former timeline up to the fork point and a complete timeline 2 copy of the fork segment, and replay through that copy recovered the promoted node's rows without the former primary's later commits.
- **INCREMENTAL across a promotion.** `pg_basebackup --incremental` on the promoted node against a timeline 1 manifest succeeded when the node had summarized WAL before the promotion, and `pg_combinebackup` over the FULL, the timeline 1 INCREMENTAL and the timeline 2 INCREMENTAL replayed to all markers. With `summarize_wal` off the node refuses with `incremental backups cannot be taken unless WAL summarization is enabled`, which the summarizer precheck reports first as `DecisionFullNewChain`. Turning summarization on after the promotion made the same INCREMENTAL succeed, because the node still kept the WAL since the FULL and summarized it. The refusal `WAL summaries are required on timeline` therefore appears only when that WAL is gone; D6 maps both refusals.

## Risks / Trade-offs

- [PostgreSQL rejects an incremental backup across a promotion in a case the prototype missed] → The refusal maps to `TIMELINE_SWITCH_DETECTED` and the existing replacement FULL runs. Restores to the window still work through WAL.
- [Standbys without `summarize_wal` make every switchover fall back to a FULL] → Patroni and CNPG documentation states the requirement. The fallback keeps backups going and the window restorable.
- [While a promoted timeline's history is not yet published, a latest restore still ends on the former timeline and can include the former primary's lost commits] → This is the current behavior, and the history usually publishes within one watch tick. Once it publishes, the line cuts the former timeline at the fork point.
- [Line computation in Go for every orphan sweep] → History rows per database are few, and the SQL prefilter still handles timelines without usable children.
- [Legacy history rows block crossing] → Matches today's behavior. New promotions publish fork points, so the limitation fades as old chains expire.
- [A cluster rebuilt from a backup reuses a timeline number] → Crossing into the conflicting timeline stops (D7). Backing up the new branch safely is out of scope and is called out in the proposal.
- [Replay from an old FULL across a long promoted period raises restore time] → The INCREMENTAL continuation (D6) bounds replay to the time since the newest backup, as before.

## Migration Plan

1. Add a goose migration on `backups_physical_wal_history_files` with nullable `parent_timeline_id INT` and `fork_lsn PG_LSN`, the constraint `CHECK (num_nonnulls(parent_timeline_id, fork_lsn) IN (0, 2))`, and `is_conflicting BOOLEAN NOT NULL DEFAULT false`. Existing rows remain valid and behave as today.
2. Deploy. New publications fill the columns. No backfill runs.
3. Rollback: the down migration drops the constraint and the columns. Restore sets built by the new version stay valid recovery sets for PostgreSQL. An older Databasus compares the live timeline with the root FULL's, so it refuses to extend a chain that already crossed a promotion and starts a new chain, which is the old behavior.

## Documentation

- `docs/architecture.md`, "Physical backups": the "Restore" and "Timeline history" bullets change from "the selected timeline's history" to the timeline line, with the reason for storing fork points in the catalog (D1).
- `docs/glossary.md`: add "Timeline line" and "Fork point".
- `docs/postgresql-patroni.md` and `docs/postgresql-cloudnativepg.md`: replace the promotion-gap paragraphs with the new behavior, and state the `summarize_wal` requirement on every node and the replacement-FULL fallback.
