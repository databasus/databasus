## Why

Existing physical backup tests exercise PostgreSQL promotion and a manually routed proxy, but do not establish recovery when Patroni elects the primary, HAProxy changes the destination, and Patroni manages replication slots. We need reproducible local evidence for PostgreSQL 17 and 18, including retained recovery points after repeated switches, before claiming compatibility with that deployment.

## What Changes

- Add one Docker-based integration suite with the same scenarios on PostgreSQL 17 and 18. Each scenario owns two PostgreSQL/Patroni nodes, one etcd and one HAProxy; Databasus runs through the existing application test harness.
- Cover an initial FULL and WAL restore, two consecutive Patroni switchovers, an automatic failover after an abrupt primary-node shutdown, and separate switchovers during active FULL and INCREMENTAL transfers.
- Keep Databasus's connection address unchanged and require automatic reconnection without restarting its modules. Verify the serving node changes, the timeline advances, and the system identifier remains the original cluster's identifier.
- Use the provisioned limited user for all Databasus backup and WAL connections. Verify published timeline history, obsolete-chain refusal, the existing replacement-FULL policy, schema recovery and usable sequence-generated IDs.
- Test exact-name `ignore_slots` protection independently from an absent slot on the promoted primary and removal of an unprotected external slot. Preserve existing slot recovery limits and expose a broken archive when required WAL is unavailable.
- Restore retained targets before a switch and after its replacement FULL, reject targets in the intervening recovery gap, and repeat recovery after the physical cleaner and storage deletion worker have run.
- Prove recovery is independent of the former primary and inherited replication/recovery settings. Reuse current helpers and fix only defects reproduced by these scenarios in this change.
- Bound PITR targets by source transaction timestamps from the selected timeline's published WAL, rather than WAL processing time. Add `pg_waldump` to the existing PostgreSQL 17/18 bundles and a nullable catalog field for this evidence; keep receipt times for archiving measurements and retention.
- Document the local command, the supported Patroni source configuration and its recovery limits. Add glossary definitions for the HA terms used by the documentation and specifications.

The planned versions are the repository's digest-pinned PostgreSQL 17 and 18 images, Patroni 4.1.5, etcd 3.6.16 and HAProxy 3.2.26. Implementation records immutable image references and locked Patroni dependencies for both supported Linux architectures before running the suite. These are planned versions, not a statement that the scenarios have already passed.

### Out of scope

- Permanent-slot synchronization and its guarantees; `ignore_slots` does not transfer slot state.
- Kubernetes, kind, k3s, operators, other load balancers or other Patroni configurations.
- DCS failures, multiple data centers, full split-brain analysis and a deliberately lagging replica's data-loss matrix.
- PITR across timelines; a replacement FULL does not recover earlier targets between promotion and its completion.
- Full corruption, clock-skew, conflicting-WAL and Databasus crash/restart matrices.
- A standalone environment-management program, new UI, new HTTP operations, new engine versions or replacement of existing client binaries, deployment, CI dispatch or publishing images.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `postgresql-physical-backup-failover`: define observable Patroni compatibility on PostgreSQL 17 and 18, repeated switches, interrupted transfers, external-slot modes, restricted history access and the permanent gap around a promotion.
- `physical-backups/operations`: require independent recovery without inherited replication or competing recovery targets, and preserve retained recovery sets through cleanup after multiple timeline changes.
- `physical-backups/wal-stream-monitoring`: report confirmed PITR bounds from source transaction timestamps and keep them consistent with token issuance and downloads under delayed WAL publication.

## Impact

- Backend: new tests under `backend/internal/modules/backups/e2e/postgresqlphysical/patroni`, reusable scenarios in `backend/internal/app/apptest/physicaltest`, and infrastructure-only fixtures under `backend/internal/lib/testcontainers`. Extend the existing history assertion beyond timeline 2 instead of copying it.
- Dependencies: fixture-only Patroni with locked Python dependencies, etcd and HAProxy; existing PostgreSQL images and client binaries remain in use. Extend `backend/internal/lib/testimages/images.json` for third-party images and their platform digests.
- Possible runtime repairs are confined to the exercised PostgreSQL engine, physical backup/WAL/history/restore code and recovery script, plus the shared UTC cron evaluator when the scheduled-retry reproducer exposes an offset-dependent due time. Start each repair with a reproducer; no defect is assumed from this plan alone.
- Documentation: `docs/testing.md`, new `docs/postgresql-patroni.md`, `docs/glossary.md`, and the two delta specs. No architecture change is planned.
- Rules: root `AGENTS.md`, `backend/AGENTS.md`, the testing skill and `docs/architecture.md` constrain the implementation. No frontend or website changes are planned.
- No breaking change to HTTP contracts, persisted schemas or source credentials is planned. Existing timeline, slot-rebuild and retention policies remain in force.
