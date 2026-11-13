## Why

Databasus's physical backups are verified against PostgreSQL managed by Patroni, but not against CloudNativePG (CNPG), the Kubernetes operator many self-hosted users run. CNPG routes clients to the primary through a Kubernetes Service and has declarative roles and access rules, TLS by default, its own replication slot synchronization and a configuration layout that keeps certificates outside the data directory. None of that is covered by the Patroni suite. We need reproducible local evidence on PostgreSQL 17 and 18 before claiming that a CNPG source keeps restorable backups and WAL through switchovers and failovers.

## What Changes

- Add a Docker-based end-to-end suite that runs CNPG in disposable single-node Kubernetes clusters (k3s in a container). A package run shares its Kubernetes clusters between scenarios, and every scenario gets its own namespace with a CNPG cluster of two PostgreSQL nodes. Databasus reaches the primary through a NodePort Kubernetes Service that uses the same selector as CNPG's `-rw` Kubernetes Service, so the CNPG operator alone decides which PostgreSQL node serves the address.
- Configure each source only through its CNPG manifest: a declared limited replication role with the right to force a WAL segment switch, HBA rules for replication over TLS, a test certificate authority with server certificates verified by Databasus, and WAL summaries for INCREMENTAL backups. No administrative role, disabled certificate check or server-file access hides a problem.
- Reach full parity with the Patroni scenarios on PostgreSQL 17 and 18:
  - initial recovery with schema and sequences;
  - two consecutive switchovers;
  - abrupt primary loss;
  - independent recovery for target, latest, backup and external configuration, plus a CNPG equivalent of the referenced-authentication case;
  - four slot modes mapped to CNPG's slot synchronization;
  - switchovers during active FULL and INCREMENTAL transfers;
  - routing outages during the failure recheck.
- Make the HA scenarios in the physical test harness independent of the cluster manager: they drive a cluster through a small common interface implemented by the Patroni fixture and the new CNPG fixture. The Patroni suite keeps every check it has today.
- Teach the test proxy to terminate TLS on both legs with certificates from the test authority, so transfer interruption and routing outages work over verified TLS.
- Run the suite in a dedicated CI group `cnpg`, sharded and native on amd64 and arm64, and exclude it from `physical`. It joins the slow backend packages of `mise run test-full`, not `mise run test`.
- Document the local command and prerequisites, the CNPG source configuration for users, the verified versions and recovery limits. Raise the inotify instance and watch limits in the CNPG CI job and document the host requirement.
- Fix only defects these scenarios reproduce, each starting from a failing reproducer.

The planned versions are the latest age-eligible CNPG 1.30 patch release, CNPG's PostgreSQL 17 and 18 images and an age-eligible k3s release, all recorded with immutable references and platform digests for both architectures before the suite runs. These are planned versions; no CNPG scenario has run yet and no defect has been reproduced.

### Out of scope

- kind, other Kubernetes distributions, managed cloud Kubernetes, cloud disks, multi-node or multi-zone clusters and network partitions.
- CNPG's own backup integrations (Barman Cloud, volume snapshots, the plugin interface), replica clusters, poolers and every other CNPG configuration variant.
- Running Databasus inside Kubernetes or changing its Helm chart.
- PITR across timelines; a replacement FULL does not recover earlier targets between promotion and its completion.
- Custom tablespaces, which Databasus keeps refusing.
- Full corruption, archive-delay, encryption and non-default WAL segment size matrices.
- Synchronous replication guarantees; delivery of control transactions to the future primary is checked explicitly instead.
- New HTTP operations, UI, engine versions or client tool bundles.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `postgresql-physical-backup-failover`: define observable CNPG compatibility on PostgreSQL 17 and 18: recovery through an unchanged Kubernetes Service address, repeated switches and abrupt loss, interrupted transfers, Databasus's slot under CNPG slot synchronization, published history, independent recovery from a CNPG data directory and restricted TLS access with a role declared in the manifest.
- `ci-pipeline`: add the `cnpg` group with its shards on both architectures and keep the CNPG suite out of `physical`.
- `local-development-workflow`: bound the CNPG suite's duration on the reference machine, leave its concurrency to the runner and require it to own and remove its Kubernetes clusters without touching the developer's Kubernetes configuration.

## Impact

- Backend tests: a new package `backend/internal/modules/backups/e2e/postgresqlphysical/cnpg`; HA scenarios in `backend/internal/app/apptest/physicaltest` move from Patroni-specific files to manager-independent ones; a new k3s/CNPG fixture and TLS support in the test proxy under `backend/internal/lib/testcontainers`.
- Dependencies: no new Go module. New test images (k3s, its system images, the CNPG operator, CNPG PostgreSQL 17 and 18) in `backend/internal/lib/testimages/images.json`, and the CNPG release manifest recorded with its source and checksum in the fixture's test data.
- Test runner and CI: `backend/test-slow-packages.txt`, `backend/test-shard-weights.txt`, `mise-tasks/backend/test-group-packages`, `mise-tasks/backend/pull-testcontainers`, the `test-backend` matrix in `.github/workflows/ci-release.yml` and the script tests that check runner selection and the expected CI check names.
- Possible runtime repairs are confined to the PostgreSQL physical engine, physical backup, WAL, history and restore code and the recovery script, each only after a reproducer fails.
- Documentation: `docs/testing.md`, a new `docs/postgresql-cloudnativepg.md`, `docs/postgresql-patroni.md` (its statement that Kubernetes is unverified), `docs/machine-setup.md` and `docs/glossary.md`.
- Rules: root `AGENTS.md`, `backend/AGENTS.md`, the testing skill and `docs/architecture.md` constrain the work. No frontend or website change is planned.
- No breaking change to HTTP contracts, persisted schemas, source credentials or the Patroni suite's observable checks.
