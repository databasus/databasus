## Why

After a PostgreSQL switchover or failover, Databasus cannot restore any moment between the promotion and the completion of a new FULL on the promoted timeline. That window lasts until the next scheduled INCREMENTAL fails with `CHAIN_BROKEN` plus the whole replacement FULL, often hours, and it falls exactly when the cluster runs without a standby. The WAL of that window is received, but restore selection ignores it and the orphan sweep deletes it. pgBackRest restores through promotions and keeps incremental chains going; Databasus has to match that recovery point and recovery time.

## What Changes

- WAL reception continues across a promotion without a hole, including a failover in which Databasus archived WAL the promoted standby never received. The part of the former timeline beyond the fork point is kept out of the recovery line.
- A chain owns WAL along its timeline line: the timelines from its root FULL's timeline to the current one, as the newest verified `.history` lists them, each up to the next fork point. Restore selection, restore coverage, the orphan sweep, cascade deletion and history deletion use this one definition.
- Restore selection follows the line of the newest timeline. A target after a promotion is reached from an earlier FULL on an ancestor timeline by replaying WAL across the fork points, with every required `.history` in the set. The recovery script pins `recovery_target_timeline`.
- An INCREMENTAL on a promoted timeline extends the chain whose backups lie on an ancestor timeline before the fork point. The replacement FULL becomes the fallback when PostgreSQL or Databasus refuses that INCREMENTAL.
- A verified `.history` that differs from the published one for the same database and timeline blocks crossing that fork point, for restore and for INCREMENTAL, instead of silently reusing the published entry.
- **BREAKING** (behavior): a target time on a former timeline after its fork point, covered only by WAL the promoted node never had, is no longer offered for PITR. Those transactions never became part of the cluster's history. No shim keeps the old selection.
- **BREAKING** (behavior): an INCREMENTAL started after a promotion no longer ends as `CHAIN_BROKEN` / `TIMELINE_SWITCH_DETECTED` when the chain can continue. That reason remains for chains that cannot continue.

## Out of scope

- Restoring the former primary's transactions that the promoted node never received.
- A restore set whose files sit in more than one storage. A line crossing a storage change is not crossed.
- Restoring a branch other than the line of the newest timeline by time, such as an abandoned sibling timeline.
- Enabling `summarize_wal` on standbys automatically. Documentation states the requirement.
- Changes to schedules, retention policies or the FULL cadence beyond the replacement FULL becoming a fallback.
- Backing up a cluster that reuses an already archived timeline number with another fork point, for example after the source was rebuilt from a backup. This change only refuses to cross such a fork point; making the new branch's own chain safe needs separate work.
- Logical backups and engines other than PostgreSQL.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `postgresql-physical-backup-failover`: INCREMENTAL continues across a promotion; the replacement FULL becomes a fallback; PITR crosses fork points; Patroni and CloudNativePG scenarios restore the window after a switch; WAL reception after a failover that lost the former primary's tail.
- `physical-backups/operations`: restore token and download selection along the timeline line, with every required history; cleanup after repeated Patroni switches keeps the switch window restorable; delayed former-timeline WAL still cannot extend reachability past the fork point.
- `physical-backups/wal-retention`: WAL belongs to a retained FULL on an ancestor timeline, not only on the same timeline.
- `physical-backups/background-load`: a chain's WAL lookup returns the chain's segments along its line.
- `physical-backups/wal-archiving`: history that conflicts with the published entry is refused and blocks the fork point.

## Impact

- Backend: `backend/internal/modules/backups/internal/physical/` (`chains`, `catalog`, `backuping`, `timelines`, `walstreaming`, `restorestream` with `recovery_script.sh`) and `backend/internal/engines/postgresql/physical/` (history ancestry, WAL receiver resume after a fork).
- No HTTP contract change. Error codes stay as they are; more targets become reachable.
- One additive migration: each published timeline history row records its parent timeline, its fork point's LSN and whether a conflicting history was seen, so restore selection and cleanup can compute the line from the catalog alone. A check constraint keeps the parent timeline and the fork point's LSN either both set or both empty. Rows published before the upgrade have no fork data, and their timelines are not crossed.
- Tests: the engine package, `physicalapitests/restoring`, `physicalapitests/catalog`, and the end-to-end suites `postgresqlphysical/pg17`, `pg18`, `patroni` and `cnpg`, with shared harness code in `backend/internal/app/apptest/physicaltest`.
- Documentation: `docs/architecture.md`, `docs/glossary.md`, `docs/postgresql-patroni.md`, `docs/postgresql-cloudnativepg.md`.
- The change answers to the root `AGENTS.md` and `backend/AGENTS.md`.
