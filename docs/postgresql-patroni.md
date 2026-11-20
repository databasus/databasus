# PostgreSQL physical backups with Patroni

Databasus connects to the current primary through one stable PostgreSQL endpoint.
The endpoint must route SQL and replication connections to the same cluster and
current primary. A load balancer can use Patroni's `/primary` health check and
close existing sessions when a server loses that role. Databasus reconnects
through the saved endpoint; it does not call Patroni to choose a primary.

Enable data checksums and WAL summaries for PostgreSQL physical incremental
backups. Keep `fsync`, `full_page_writes` and `synchronous_commit` enabled.
For an asynchronous replica, a successful client commit alone does not prove
that the replica has received it. Data not replayed on the promoted replica
can be lost when the primary changes.

## Access and external slot ownership

Add the stable endpoint as a PostgreSQL physical database, create its limited
user through Databasus, then save that user's credentials for backups. The
replication-only user can perform physical backups and WAL reception without
becoming a superuser or reading ordinary application tables. Its privileges
replicate with the PostgreSQL catalog and remain restricted after promotion.
Keep the source's normal TLS and HBA policy. Test-network authentication
settings are not a deployment recommendation.

Databasus owns its physical replication slot. Before enabling WAL reception,
inspect the saved slot name in Databasus's internal database. The user API does
not expose it. Use the managed database's ID in this read-only query:

```sql
SELECT replication_slot_name
FROM databases_postgresql_physical
WHERE database_id = '<database-id>';
```

Configure Patroni's dynamic configuration to ignore that exact name:

```yaml
ignore_slots:
  - name: databasus_slot_REPLACE_WITH_THE_SAVED_NAME
    type: physical
```

Use the complete saved name rather than a prefix or regular expression. This
exemption lets an inactive slot survive Patroni's management cycles. It does
not copy the slot to another node or guarantee that a promoted node has it.
Without an exemption or an appropriate permanent-slot configuration, Patroni
can remove an inactive external slot.

If a slot is absent, Databasus follows its existing recovery policy or reports
`BROKEN` with `SLOT_MISSING`. A recreated slot protects future WAL; it cannot
recover segments already recycled before archival. Affected recovery targets
remain unavailable, including after a later full backup succeeds.

## Backup chains and recovery

A promotion starts a new PostgreSQL timeline. WAL reception reconnects,
archives the new timeline's history alongside its WAL and continues from the
fork point, even when Databasus had archived writes of the former primary that
the promoted replica never received. The next incremental backup continues the
existing chain on the new timeline, so a promotion costs no full backup.

That continuation needs `summarize_wal = on` on every node that can become
primary. Without WAL summaries on the promoted node, or when PostgreSQL refuses
the incremental backup across the timeline change, the incremental ends with
`CHAIN_BROKEN` / `TIMELINE_SWITCH_DETECTED` and Databasus requests one immediate
replacement full backup.

When a primary changes during an active backup and the failure recheck reaches
the new primary, the attempt ends with `ERROR` / `FAILOVER_DURING_BACKUP`. An
interrupted full backup requests one immediate replacement full backup. An
interrupted incremental does not: the archived WAL already covers the switch,
and the next incremental runs at its scheduled time. When the route is still
unavailable during the recheck, Databasus retains the original stream failure
and uses its existing scheduled retry behavior.

A restore to a target after the promotion replays the chain across the fork
point once the new timeline's history is published, also before any backup runs
on the new timeline. Commits only the former primary made after the fork point
are never restored, and a target that only they cover returns
`409 target_time_unreachable`.

PITR eligibility follows source transaction timestamps in the published WAL on
the chain's timeline line. Inclusive recovery needs a transaction strictly later
than the target to confirm coverage. WAL without that evidence does not extend
the time bound beyond a completed backup; latest recovery and explicit
completed-backup recovery remain available. [Physical backup
operations](../openspec/specs/physical-backups/operations/spec.md) define the
exact coverage and failure behavior.

The recovery script prepares an independent PostgreSQL instance. It removes
inherited primary connection settings, standby/recovery signals and competing
recovery targets from copied configuration, and disables source archiving.
Included configuration and authentication files copied inside the source's
data directory are prepared for the local target, including paths that differ
from PostgreSQL's defaults. Recovery uses only the selected backup and
downloaded WAL; the former primary need not be reachable.

Configuration stored outside the source's data directory is not in the physical
backup. The recovery script reports the missing files. Prepare an independent
`postgresql.conf`, `pg_hba.conf` and `pg_ident.conf` for the target as instructed,
then start it with the matching PostgreSQL major. Choose the target's own access
policy without changing the source configuration.

## Verified environment

The [recorded verification](../openspec/changes/archive/2026-11-11-verify-postgresql-patroni-wal/verification.md)
owns the exercised versions, architecture, scenarios and results. The
[fixture README](../backend/internal/lib/testcontainers/testdata/patroni/README.md)
owns its dependency records; [testing guidance](testing.md#patroni-physical-backup-tests)
provides the command and prerequisites.

Those results do not establish native arm64 scenario execution or the behavior
of Kubernetes, operators, multiple data centers, network partitions or every
Patroni configuration. CloudNativePG is verified separately, as
[PostgreSQL with CloudNativePG](postgresql-cloudnativepg.md) describes; other
Kubernetes operators remain unverified. Starting a client binary under
emulation does not establish recovery behavior on that architecture.
