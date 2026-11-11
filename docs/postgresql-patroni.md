# PostgreSQL physical backups with Patroni

Databasus connects to the current primary through one stable PostgreSQL endpoint.
The endpoint must route SQL and replication connections to the same cluster and
current primary. A load balancer can use Patroni's `/primary` health check and
close existing sessions when a server loses that role. Databasus reconnects
through the saved endpoint; it does not call Patroni to choose a primary.

Enable data checksums and WAL summaries for PostgreSQL physical incremental
backups. Keep `fsync`, `full_page_writes` and `synchronous_commit` enabled.
For an asynchronous replica, a successful client commit alone does not prove
that the replica has received it. The test fixture checks replay of a committed
marker and its post-commit flush LSN before switching or killing the primary.

## Access and external slot ownership

Add the stable endpoint as a PostgreSQL physical database, create its limited
user through Databasus, then save that user's credentials for backups. The
replication-only user can perform physical backups and WAL reception without
becoming a superuser or reading ordinary application tables. Its privileges
replicate with the PostgreSQL catalog and remain restricted after promotion.
Keep the source's normal TLS and HBA policy; the fixture's isolated, loopback
published endpoints use SCRAM authentication without TLS.

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

A promotion starts a new PostgreSQL timeline. An incremental backup cannot
continue a chain rooted on the old timeline. Databasus rejects that continuation
and requests a replacement full backup. WAL reception reconnects and archives
the new timeline's history alongside its WAL.

When a primary changes during an active full backup and the failure recheck
reaches the new primary, the attempt ends with `ERROR` /
`FAILOVER_DURING_BACKUP`. An interrupted incremental ends with `CHAIN_BROKEN` /
`TIMELINE_SWITCH_DETECTED`. Each requests one immediate replacement full backup.
When the route is still unavailable during the recheck, Databasus retains the
original stream failure and uses its existing scheduled retry behavior.

An archived target on the old timeline remains recoverable while retention
protects its files. A target between promotion and completion of the replacement
full backup is unavailable: restore-token creation returns
`409 target_time_unreachable`. A later full backup does not fill that gap.

PITR eligibility follows source transaction timestamps in the selected
timeline's contiguous published WAL. A delayed upload of old WAL cannot extend
that timeline into the promotion gap. Inclusive recovery needs a transaction
strictly later than the target, so the reported bound is one microsecond before
the latest confirming transaction. WAL catalog rows without timestamp evidence
do not extend that bound beyond a completed backup; latest recovery and explicit
completed-backup recovery remain available.

The recovery script prepares an independent PostgreSQL instance. It removes
inherited primary connection settings, standby/recovery signals and competing
recovery targets from copied configuration, and disables source archiving.
This covers assignments with or without `=` and follows local `include`,
`include_if_exists` and `include_dir` directives. Directly included files are
processed regardless of filename extension.
Absolute includes below the source `data_directory` recorded in the copied
configuration are rebased to relative paths inside the restored directory.
The script finds that setting through local includes before sanitization,
including when an absolute include appears before its declaration.
Source-local HBA and ident paths are rebased to their copied files, including
files in subdirectories when the default root-level files are absent. Copied
default files remain usable for source-specific authentication overrides.
A referenced copied authentication file takes precedence over an unrelated
root-level default.
Authentication paths point under the prepared local PGDATA or, with
`--combine-image`, the selected image's PGDATA. Recovery uses only the selected
backup and downloaded WAL; the former primary need not be reachable.

Configuration stored outside the source's data directory is not in the physical
backup. The recovery script reports the missing files. Prepare an independent
`postgresql.conf`, `pg_hba.conf` and `pg_ident.conf` for the target as instructed,
then start it with the matching PostgreSQL major. Choose the target's own access
policy without changing the source configuration.

## Verified environment

Local verification on 2026-11-11 ran PostgreSQL 17.11 and 18.6 on Linux amd64.
The fixture uses two real PostgreSQL nodes managed by Patroni 4.1.5,
etcd 3.6.16 and HAProxy 3.2.26. Dependency references, publication dates and
the hash-locked Python dependency selection are recorded in the
[fixture dependency manifest](../backend/internal/lib/testcontainers/testdata/patroni/README.md).

The matrix covers PostgreSQL 17 and 18: initial recovery, two planned
switchovers, abrupt primary loss, interrupted full and incremental transfers,
routing outages, external slot survival and loss, independent recovery and
retention. It checks rows, schema definitions, sequences and generated-ID writes
after actually starting the restored server. The two-switchover cases also
require PostgreSQL to read the published timeline history during recovery.

See [the test command and prerequisites](testing.md#patroni-physical-backup-tests).
No native arm64 scenario execution is claimed. The added arm64 `pg_waldump`
binaries were started through QEMU with the Debian bookworm runtime ABI.
These container tests do not establish Kubernetes, operator, multi-datacenter,
network-partition or every Patroni configuration's behavior.
