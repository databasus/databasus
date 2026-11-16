Every `go test` command below runs from `backend/` after the Fedora preparation in `docs/testing.md#commands` (`source ../scripts/fedora.sh`, `apply_libedit_shim`) and with `TESTCONTAINERS_RYUK_DISABLED=true`.

## 1. Native prototype

- [x] 1.1 Add the engine test `TestTimelineCrossing` in `backend/internal/engines/postgresql/physical/`. It takes a FULL and an INCREMENTAL on timeline 1 from the isolated primary with `summarize_wal` on for both nodes, writes a marker, promotes the standby, writes a marker on timeline 2, and replays the backups with the archived WAL to a timeline 2 target using `restore_command` and `recovery_target_timeline`. Record which files PostgreSQL requests and which copy of the fork segment it reads. Verify with `go test -count=1 -run 'TestTimelineCrossing' ./internal/engines/postgresql/physical/` that both markers are present.
- [x] 1.2 Extend `TestTimelineCrossing` with a standby stopped before promotion while the primary keeps committing. Capture the server's refusal for a start LSN beyond the fork point and the fork segment `pg_receivewal` writes after restarting from the segment holding the fork point's LSN. Verify with the command in 1.1 that the refusal is recognized and the realigned segment replays.
- [x] 1.3 Extend `TestTimelineCrossing` with `pg_basebackup --incremental` on the promoted standby against the timeline 1 manifest, with and without `summarize_wal` on the standby, and `pg_combinebackup` over the result. Verify with the command in 1.1 that the combined cluster starts with all markers and that the refusal without summaries is recognized.
- [x] 1.4 Record in `design.md` any finding that contradicts D3, D6 or D8 and adjust the affected tasks. Verify with `openspec validate physical-restore-across-timelines --strict`.

## 2. Fork points in the catalog

- [x] 2.1 Add a goose migration on `backups_physical_wal_history_files` with nullable `parent_timeline_id` and `fork_lsn`, the constraint `CHECK (num_nonnulls(parent_timeline_id, fork_lsn) IN (0, 2))` and `is_conflicting BOOLEAN NOT NULL DEFAULT false`, with up and down. Verify with `mise run backend:lint` and a catalog test in `physicalapitests/catalog` that inserting only one of the two paired columns fails, run by `go test -count=1 ./internal/modules/backups/physicalapitests/catalog`.
- [x] 2.2 Make `validateTimelineHistory` return the last entry it parses as `TimelineFork{ParentTimelineID, ForkLSN}`, without a second parser. Verify with new cases in `TestValidateTimelineHistory` asserting the returned parent and fork point's LSN, run by `go test -count=1 -run 'TestValidateTimelineHistory' ./internal/engines/postgresql/physical/`.
- [x] 2.3 Store the parent timeline and fork point's LSN when `UploadHistoryFile` publishes a row. When a row exists: reuse it on a match; on a difference, set `is_conflicting` with one `UPDATE`, publish nothing and log a warning with the database and timeline only; never clear the flag; reuse a row without a fork point unchanged and unflagged. Verify with API tests in `physicalapitests/walstreaming`, where history publication runs against a real promoted source, for match, conflict, a later matching report after a conflict, and a legacy row, run by `go test -count=1 -run 'TestTimelineHistoryForkPublication' ./internal/modules/backups/physicalapitests/walstreaming`.

## 3. Timeline line and restore selection

- [x] 3.1 Add `TimelineLine`, `TimelineLineRange` and `BuildTimelineLine` to the `chains` package. They compute the newest line from the database's history rows and the line of a given FULL, including an off-line FULL. Verify with `TestBuildTimelineLine` covering one fork point, two fork points, a fork point on a segment boundary, a higher-numbered sibling that does not descend from the FULL, a conflicting link, a legacy link, a link in another storage and a FULL started after the fork point. Run `go test -count=1 -run 'TestBuildTimelineLine' ./internal/modules/backups/internal/physical/chains/`.
- [x] 3.2 Replace `FindByChainSpan` and `GetLatestReceivedAtInChainSpan` with `FindByTimelineLine` and `GetLatestReceivedAtOnTimelineLine`, and make `GetChainEndTimestamp` use the line. Verify with catalog tests in `physicalapitests/catalog`: deleting a chain that crosses one promotion leaves the parent segment that starts after the fork point's LSN alone, and a timeline 2 segment received last sets a closed chain's end time; the restore tests from 3.3 check that no such parent segment reaches a restore set. Run `go test -count=1 ./internal/modules/backups/physicalapitests/catalog`.
- [x] 3.3 Make `resolveRestoreReach` use the line: contiguity across fork points, all line histories in the set, only FULLs on the newest line for latest and time targets, and no time evidence from a parent segment that holds a fork point's LSN. Add the newest timeline to the restore set. Verify with tests in `physicalapitests/restoring` for:
  - a latest restore after a promotion whose set holds the parent WAL ending with the fork segment, the child history and child WAL starting with that segment;
  - a latest restore on a timeline 1 chain while timeline 2 history is pending;
  - a target one minute after a promotion;
  - a target after a completed replacement FULL that uses the earlier chain;
  - a target covered only by the former primary's WAL (409);
  - unpublished, conflicting, legacy and other-storage links (409);
  - an INCREMENTAL on timeline 2 whose set contains no timeline 1 WAL.

  Run `go test -count=1 ./internal/modules/backups/physicalapitests/restoring`.
- [x] 3.4 Update the existing tests that pin the promotion gap, such as `refuses the promotion gap when older timeline WAL publishes late` in `restore_timeline_time_test.go`, to the behavior in the operations delta spec. Verify with `go test -count=1 ./internal/modules/backups/physicalapitests/restoring`.
- [x] 3.5 Make the restore stream carry the newest timeline and the recovery script write `recovery_target_timeline` whenever it ships WAL and nothing otherwise. Verify with restore bundle assertions for both cases in `physicalapitests/restoring`, run by `go test -count=1 ./internal/modules/backups/physicalapitests/restoring`; the script's setting is asserted by the end-to-end scenario in 7.1.

## 4. Retention along the line

- [x] 4.1 Limit the `FindOrphans` SQL prefilter to timelines with no usable child link, check all other segments against the lines of the completed FULLs, and protect pending-history segments. Keep the running-FULL guard and the cleanup lock. Verify with tests in `physicalapitests/catalog/timeline_line_test.go`:
  - timeline 2 WAL before any timeline 2 backup survives a sweep with no running FULL, both with published history and while history is pending;
  - timeline 2 WAL behind a conflicting or legacy link is swept;
  - the former primary's WAL past the fork point is swept, and is kept when a FULL on timeline 1 started after the fork point;
  - after both cleanup stages with no timeline 2 backup, a target on timeline 2 still receives a restore token.

  Run `go test -count=1 ./internal/modules/backups/physicalapitests/catalog`.
- [x] 4.2 Use the line for cascade deletion and history deletion. The deletion preview this task named no longer exists in the code, so there is nothing to align. Verify with tests in `physicalapitests/catalog/timeline_line_test.go`. Deleting a timeline 2 FULL while a timeline 1 FULL remains keeps timeline 2 WAL and history. Deleting the earliest FULL whose line crosses a fork point removes only its ownership span. Deleting the last FULL on a promoted timeline's line removes that timeline's history together with its leftover WAL, which would otherwise stay protected as pending history. Run `go test -count=1 ./internal/modules/backups/physicalapitests/catalog`.

## 5. WAL reception after a lost tail

- [x] 5.1 Recognize in the engine the server's refusal for a start LSN outside its history and report `WalReceiverForkRealignment` with the fork point's LSN from the verified history read. Verify with a test in `wal_receiver_test.go` using the refusal text captured in 1.2, run by `go test -count=1 -run 'TestWalReceiver' ./internal/engines/postgresql/physical/`.
- [x] 5.2 Handle that disposition in the streamer: drop queued local segments of the former timeline that start at or after the segment holding the fork point's LSN, and restart reception from that segment, without a slot rebuild, a WAL gap notification or a replacement FULL. Verify with a test in `walstreaming` covering the queue cleanup and restart, run by `go test -count=1 ./internal/modules/backups/internal/physical/walstreaming/...`.

## 6. INCREMENTAL across a promotion

- [x] 6.1 Add `TimelineContinueAcrossPromotion`. Compare the live timeline with the timeline of the chain's newest included backup, publish the live timeline's history before deciding and refuse when publication fails or the link is unusable. Accept when the newest line reaches the live timeline and that backup stopped at or before the first fork point's LSN; otherwise record `CHAIN_BROKEN` / `TIMELINE_SWITCH_DETECTED`. Verify with tests in `backuping` for continuation, a backup past the fork point, a non-descending timeline, a conflicting history, a failed publication and two promotions, run by `go test -count=1 ./internal/modules/backups/internal/physical/backuping/...`.
- [x] 6.2 Map the summarizer's `DecisionFullNewChain` and `pg_basebackup` refusals tied to the manifest timeline or WAL summaries to `CHAIN_BROKEN` / `TIMELINE_SWITCH_DETECTED` when the live timeline differs from the newest backup's, so the existing immediate replacement FULL runs. Verify with tests in `backuping` for both refusals, run by the command in 6.1.
- [x] 6.3 Make the INCREMENTAL stream-failure recheck compare against the timeline captured at preflight. Classify a newer timeline as `ERROR` / `FAILOVER_DURING_BACKUP` with no immediate replacement FULL. Verify with scheduler and executor tests in `backuping` run by the command in 6.1, including a stream failure without promotion on a chain whose newest INCREMENTAL is already on timeline 2.

## 7. End-to-end scenarios

- [x] 7.1 Rewrite the shared plain PostgreSQL promotion scenario `RunPromotionReanchorsChain` in `backend/internal/app/apptest/physicaltest/promotion.go`, with `summarize_wal` on for the standby:
  - a FULL and an INCREMENTAL on timeline 1 and a marker, then the promotion;
  - a restore to a target one minute after promotion, before any timeline 2 backup, checking both markers with the schema and sequence checks from `schema_recovery.go`, `sequence_recovery.go` and `restore_schema.go`;
  - `backups_physical_cleaner` and `storage_file_deletion`, then a fresh token and restore for the same target;
  - an INCREMENTAL on timeline 2 without a replacement FULL, a marker and a restore after it;
  - both cleanup stages again, then both targets restored without access to the source.

  Verify with `go test -count=1 -timeout 40m ./internal/modules/backups/e2e/postgresqlphysical/pg17 ./internal/modules/backups/e2e/postgresqlphysical/pg18`.
- [x] 7.2 Add a lost-tail scenario to the same plain PostgreSQL suites. Stop the standby, commit and archive on the primary, stop the primary and promote the standby. Reception resumes on timeline 2 without a WAL gap notification, a target after the promotion restores without the former primary's rows, and a target only those rows cover gets `409 target_time_unreachable`. Verify with the command in 7.1.
- [x] 7.3 Add a fallback scenario to the same plain PostgreSQL suites that turns `summarize_wal` off on the standby before the promotion; the database settings check refuses a source without WAL summaries, so the standby starts with them. The next INCREMENTAL ends as `CHAIN_BROKEN` / `TIMELINE_SWITCH_DETECTED` and exactly one replacement FULL follows. A restore from that FULL succeeds, and a target between the promotion and that FULL's completion restores from the earlier chain. Verify with the command in 7.1.
- [x] 7.4 Update `RunStableEndpointPromotion` in `source_reconnect.go`, which still waits for a re-anchored FULL: after the switch, the next INCREMENTAL extends the chain and a restore to a target on the promoted timeline succeeds. Verify with the command in 7.1.
- [x] 7.5 Update the shared Patroni and CNPG switchover scenario `RunFailoverSwitchovers` in `failover_switchover.go` and the interruption scenarios in `failover_interruption.go` and `failover_replacement.go`. After each switch, the INCREMENTAL extends the chain and a target one minute after the switch restores before it. An interrupted INCREMENTAL ends as `ERROR` / `FAILOVER_DURING_BACKUP` with no immediate FULL. The switch window stays restorable after both cleanup stages. Verify with `go test -count=1 -timeout 40m ./internal/modules/backups/e2e/postgresqlphysical/patroni`.
- [x] 7.6 Update `RunFailoverPrimaryLoss` in `failover_crash.go` and `RunPatroniIndependentRecovery` in `patroni_independent.go`, which wait through `waitForReanchoredFull` or `requireSingleReplacement`. `RunFailoverPrimaryLoss` restores a target after the failover from the chain that crosses it, with the control transaction, schema and generated IDs. `RunPatroniIndependentRecovery` checks recovery inputs planted in the promoted node's data directory, which only a FULL taken on that node carries, so it requests that FULL explicitly instead of waiting for a replacement. Verify with the command in 7.5.
- [x] 7.7 Update `RunCNPGIndependentRecovery` in `cnpg_independent.go` (which, like `RunPatroniIndependentRecovery`, requests an explicit FULL on the promoted node), `RunCNPGSynchronizedSlot` in `cnpg_slots.go` and `RunCNPGRestrictedAccessAcrossSwitch` in `cnpg_role_access.go`, together with the CNPG runs of the shared scenarios from 7.5 and 7.6. Assertions that still expect a replacement FULL, `CHAIN_BROKEN` or `409` in the switch window follow the delta spec. Verify with `go test -count=1 -timeout 40m ./internal/modules/backups/e2e/postgresqlphysical/cnpg`.

## 8. Documentation

- [x] 8.1 Update `docs/architecture.md` "Physical backups" (the "Restore" and "Timeline history" bullets) for the timeline line and the reason fork points live in the catalog, and add "Timeline line" and "Fork point" to `docs/glossary.md`. Verify with `rg -n "selected timeline's" docs/` returning no stale statement and `mise run lint`.
- [x] 8.2 Replace the promotion-gap paragraphs in `docs/postgresql-patroni.md` and `docs/postgresql-cloudnativepg.md` with the new restore and INCREMENTAL behavior, the `summarize_wal` requirement on every node and the replacement-FULL fallback. Verify with `rg -n "does not fill that gap|does not replay across timelines" docs/` returning nothing and `mise run lint`.

## 9. Formatting and lint

- [x] 9.1 Run `mise run backend:format` and `mise run backend:lint` on the changed backend code and fix every finding.

## Change verification

```sh
mise run test
```
