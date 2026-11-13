# PostgreSQL physical backups with CloudNativePG

CloudNativePG (CNPG) is a Kubernetes operator that runs a PostgreSQL cluster as
pods and decides which PostgreSQL node is the primary. Databasus connects to the
current primary through one Kubernetes Service address and reconnects through
that address after a switchover or failover. It does not call the CNPG operator
or the Kubernetes API to choose a primary.

## The address Databasus connects to

The address must route SQL and replication connections to the current primary.
CNPG's `<cluster>-rw` Kubernetes Service does that: it selects the pods labelled
`cnpg.io/cluster=<cluster>` and `cnpg.io/instanceRole=primary`, and the CNPG
operator moves that label when the primary changes.

- **Databasus inside the Kubernetes cluster.** Use the `-rw` Kubernetes Service,
  for example `<cluster>-rw.<namespace>.svc` on port 5432.
- **Databasus outside the Kubernetes cluster.** Expose a Kubernetes Service with
  the same selector. The verified setup uses a NodePort Kubernetes Service, and
  Databasus connects to a node address and the allocated node port:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: <cluster>-databasus
  namespace: <namespace>
spec:
  type: NodePort
  selector:
    cnpg.io/cluster: <cluster>
    cnpg.io/instanceRole: primary
  ports:
    - name: postgres
      port: 5432
      targetPort: 5432
      protocol: TCP
```

Never point Databasus at a single pod or at the `-r` or `-ro` Kubernetes
Services: they do not follow the primary.

## The backup role

Declare Databasus's role in the `Cluster` manifest instead of creating it
through Databasus. CNPG's superuser access can stay disabled, because Databasus
needs no administrative role. The role logs in with replication rights and
belongs to a `NOLOGIN` group that may force a WAL segment switch:

```yaml
apiVersion: postgresql.cnpg.io/v1
kind: Cluster
metadata:
  name: <cluster>
spec:
  instances: 2
  bootstrap:
    initdb:
      database: <database>
      owner: <owner>
      dataChecksums: true
      postInitSQL:
        - CREATE ROLE databasus_backup_group NOLOGIN
        - GRANT EXECUTE ON FUNCTION pg_switch_wal() TO databasus_backup_group
      postInitApplicationSQL:
        - GRANT EXECUTE ON FUNCTION pg_switch_wal() TO databasus_backup_group
  managed:
    roles:
      - name: databasus_backup
        ensure: present
        login: true
        replication: true
        inRoles:
          - databasus_backup_group
        passwordSecret:
          name: databasus-backup
  postgresql:
    parameters:
      summarize_wal: "on"
    pg_hba:
      - hostnossl all databasus_backup all reject
      - hostnossl replication databasus_backup all reject
      - hostssl replication databasus_backup all scram-sha-256
      - hostssl all databasus_backup all scram-sha-256
  certificates:
    serverCASecret: <server-ca-secret>
    serverTLSSecret: <server-tls-secret>
  replicationSlots:
    synchronizeReplicas:
      enabled: true
```

The secret named in `passwordSecret` is a `kubernetes.io/basic-auth` secret with
the role's `username` and `password`. The grant goes to the group because the
bootstrap SQL runs before CNPG creates the managed role. A function
grant belongs to one database: `postInitSQL` runs in the `postgres` database and
`postInitApplicationSQL` in the application database, so the example grants
`pg_switch_wal()` in both. Make the grant at least in the database Databasus
connects to. CNPG runs these statements only when it creates the cluster; on an
existing cluster, run them once yourself.

The HBA rules reject the role on any connection without TLS and accept it over
TLS with SCRAM authentication. The role cannot read server files or ordinary
application tables, and its privileges replicate with the PostgreSQL catalog,
so they stay the same after a promotion. Databasus's readiness check reports the
missing `pg_switch_wal()` grant if the role lacks it.

Add the address as a PostgreSQL physical database in Databasus with that role's
credentials.

## TLS

Every connection from Databasus to the source uses TLS with the server
certificate verified. Set the database's SSL mode to `verify-full` and give
Databasus the certificate authority that issued the server certificate.

With `verify-full`, the server certificate must name the address Databasus
connects to: the `-rw` Kubernetes Service name for a client inside the
Kubernetes cluster, or the node address or host name for a client outside it.
CNPG's generated server certificates name only the in-cluster Kubernetes Service
names. The verified setup supplies its own certificate authority and server
certificate through `certificates.serverCASecret` and
`certificates.serverTLSSecret`, and the certificate names both the in-cluster
names and the address Databasus uses.

Databasus refuses a certificate from another authority and a certificate that
does not name the address. The connection test then reports the database as not
connected, a full backup ends with `ERROR` and WAL reception archives nothing.

## Incremental backups and durability

Physical incremental backups need data checksums and WAL summaries. Set
`bootstrap.initdb.dataChecksums: true` when the cluster is created, and
`summarize_wal: "on"` in `postgresql.parameters`. Keep `fsync`,
`full_page_writes` and `synchronous_commit` at their enabled defaults.

CNPG replicates asynchronously unless you configure synchronous replication. A
successful client commit alone does not prove that the standby has received
it, and data not replayed on the promoted standby can be lost when the primary
changes.

## Databasus's replication slot

Databasus owns a physical replication slot on the primary, named
`databasus_slot_<uuid>`. [PostgreSQL with
Patroni](postgresql-patroni.md#access-and-external-slot-ownership) shows how to
read the saved name from Databasus's internal database.

CNPG can copy user-defined physical slots to its standbys with
`.spec.replicationSlots.synchronizeReplicas`. A standby's copy lags behind the
primary by up to `.spec.replicationSlots.updateInterval`. The setting decides
what Databasus finds on a promoted standby:

- **Synchronization enabled, slot not excluded.** The standby holds a copy of
  the slot. WAL reception resumes on the promoted node without losing WAL only
  if the copy's position still covered the WAL Databasus had not archived at
  the switch. A slot with the right name on the new primary does not prove that
  by itself.
- **Slot name matches `excludePatterns`.** The standby has no copy. After
  promotion, Databasus recovers the slot under its existing slot recovery policy
  or reports the WAL stream as `BROKEN` with `SLOT_MISSING`.
- **Synchronization turned off.** CNPG removes the copies it created on the
  standby. After promotion, Databasus behaves as in the excluded case and does
  not report uninterrupted archiving without the WAL to support it.

Keep synchronization enabled and make sure no exclusion pattern matches the
saved slot name.

`wal_keep_size` makes every PostgreSQL node keep at least that much WAL
regardless of slots, which leaves more WAL available to a recreated slot after
promotion. CNPG's default is 512 MB. It does not replace the slot.

A recreated slot protects future WAL but cannot recover segments recycled before
Databasus archived them. Recovery targets inside such a gap stay unavailable,
even after a new slot or a later full backup.

## Backup chains and recovery

A switchover or failover starts a new PostgreSQL timeline, and Databasus treats
it as it does under Patroni. An incremental backup on the old timeline ends with
`CHAIN_BROKEN` / `TIMELINE_SWITCH_DETECTED`, a full backup interrupted by the
switch ends with `ERROR` / `FAILOVER_DURING_BACKUP`, and Databasus requests one
replacement full backup. These results need the failure recheck to reach the new
primary. When the address is still unavailable during the recheck, Databasus
keeps the original failure and waits for the next scheduled backup. WAL
reception reconnects through the same address and archives the new timeline's
history.

A target between the promotion and the completion of the replacement full backup
is unavailable: restore-token creation returns `409 target_time_unreachable`.
Recovery does not replay across timelines. [Physical backup
operations](../openspec/specs/physical-backups/operations/spec.md) define the
exact coverage and failure behavior.

## Recovery outside Kubernetes

The recovery script prepares an independent PostgreSQL server outside
Kubernetes from the selected backup and downloaded WAL. The source cluster, its
Kubernetes Service and the former archive need not be reachable. The script
removes inherited primary connection settings, restore commands, competing
recovery targets and standby or recovery signal files.

Start the restored data directory with PostgreSQL of the same major version,
for example the official `postgres:17` or `postgres:18` image. CNPG's images
have no entrypoint that starts PostgreSQL without CNPG's instance manager. The verified setup uses
CNPG's `standard-trixie` images and boots recovery with official images built on
the same Debian release, so both use the same C library collations.

CNPG keeps `postgresql.conf`, `pg_hba.conf` and `pg_ident.conf` inside the data
directory. The recovery script prepares them for the target location, and the
restored server uses the copied authentication rules.

CNPG also keeps the server certificates, the Unix socket directory and the logs
outside the data directory, under `/controller`, so the backup does not contain
them. The recovery script lists every such setting, for example:

```text
ssl_cert_file = '/controller/certificates/server.crt'
ssl_key_file = '/controller/certificates/server.key'
ssl_ca_file = '/controller/certificates/client-ca.crt'
unix_socket_directories = '/controller/run'
log_directory = '/controller/log'
```

The restored server does not start until each one works on the target host.
Give each a value for that host in `postgresql.auto.conf` inside the restored
data directory, which overrides the copied files:

```text
ssl = off
unix_socket_directories = '/var/run/postgresql'
log_directory = 'log'
```

Instead of `ssl = off`, you can point the TLS settings at the target host's own
certificates. With TLS off, the copied `hostssl` rules match no connection, so
review the restored `pg_hba.conf` and choose the target's own access policy.
None of this changes the source's TLS or access rules.

## Verified environment

The CloudNativePG suite runs each CNPG cluster in k3s, a single-node Kubernetes
distribution, inside a privileged container on rootful Docker on Linux. It
reaches the primary through a NodePort Kubernetes Service. These versions are
pinned:

| Component | Version |
| --- | --- |
| CloudNativePG | 1.30.1 |
| k3s | v1.36.5+k3s1 |
| CNPG PostgreSQL 17 image | 17.11, `standard-trixie` |
| CNPG PostgreSQL 18 image | 18.6, `standard-trixie` |

The [fixture README](../backend/internal/lib/testcontainers/testdata/cnpg/README.md)
owns the dependency records, and [testing guidance](testing.md#cloudnativepg-physical-backup-tests)
provides the command and prerequisites.

These results do not establish the behavior of other Kubernetes distributions,
managed Kubernetes services, multi-node or multi-zone clusters, network
partitions, CNPG's own backup integrations, replica clusters, poolers or other
CNPG configuration variants.
