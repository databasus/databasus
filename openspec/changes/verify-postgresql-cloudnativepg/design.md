## Context

See `proposal.md` for the motivation and scope. No CloudNativePG scenario has run for this change, and no runtime defect is assumed.

Terms: "PostgreSQL node" is a PostgreSQL server of a cluster, as `docs/glossary.md` defines primary and standby; CNPG calls it an instance and runs it as a pod. "Kubernetes Service" is the Kubernetes routing object, never the backend's service layer. "CNPG operator" is the program, never a human operator. "k3s container" is the Docker container that runs one Kubernetes cluster.

Relevant current state:

- The Patroni suite already holds the full scenario matrix this change needs (`backend/internal/modules/backups/e2e/postgresqlphysical/patroni/backups_test.go:18-197`). Its families are recovery, independent recovery, slots, interrupted backups and routing outages, each split into one top-level test per PostgreSQL major for CI shards.
- The scenario logic lives in `backend/internal/app/apptest/physicaltest/patroni_*.go`, about 1,500 lines. The coupling to Patroni sits in `patroniSource.Fixture *containers.Patroni` (`physicaltest/patroni_source.go:18`).
  - The scenarios call `Endpoint`, `GetIdentity`, `GetNode`, `GetReplica`, `Connect`, `Switchover`, `KillPrimary`, `KillNode` and `ReconcileSlots`.
  - The replay barrier `RequireReplay` runs inside `Switchover` and `KillPrimary` (`backend/internal/lib/testcontainers/patroni_switchover.go:20`, `patroni_failover.go:20`).
  - Slot ownership is chosen once, when the source is built: `newPatroniSource` calls `IgnoreSlot` and `KeepSlotOnAllNodes` (`physicaltest/patroni_source.go:76,80`). The routing-outage scenarios ask for the second one (`physicaltest/patroni_interruption.go:36`).
- Patroni sources use no TLS: the harness saves the database with `SslModeDisable` (`physicaltest/team.go:41`). The limited user is created by Databasus through `CreateLimitedUser` with the administrator's credentials (`physicaltest/team.go:69-95`).
- Databasus already accepts a user created outside it. Its readiness check asks for `GRANT EXECUTE ON FUNCTION pg_switch_wal()` when the grant is missing (`backend/internal/engines/postgresql/physical/replication_readiness.go:116`). The replication session itself is identified with `IDENTIFY_SYSTEM` (`wal_session.go:40`, `timeline_history.go:43`).
- The connection test reports `isConnected` with a failure class and the server's error text (`backend/api/openapi.yaml:4955-4980`). WAL stream health reports reasons such as `SOURCE_UNREACHABLE` and `SLOT_MISSING` (`backend/api/openapi.yaml:6343-6358`).
- Interruption, slot and outage scenarios route Databasus through `containers.PostgresProxy`, which parses the PostgreSQL protocol with `pgproto3` (`backend/internal/lib/testcontainers/postgres_proxy.go:15,62,183`). It gates a held `BASE_BACKUP`, blocks replication and simulates an outage (`physicaltest/patroni_interruption.go:122`, `patroni_slots.go:48`). It cannot read encrypted traffic.
- The recovery script already diagnoses settings that point outside the data directory, TLS files included (`backend/internal/modules/backups/internal/physical/restorestream/recovery_script.sh:312`). It removes inherited replication and recovery settings (`:726`).
- Recovered clusters boot from an official `postgres:<major>` image, and the image tag selects the data directory layout (`physicaltest/cluster_directory.go:17-37`).
- Fixtures label their resources with the run ID (`backend/internal/lib/testcontainers/containers.go:203`). When `scripts/test.sh` exits, including after its test process was killed, it removes containers, networks and volumes with its own run's label (`scripts/test.sh:32-66`). Networks take a free `/24` from the test range through `WithFreeSubnet` (`network_subnet.go:23`).
- `GetPinnedImage` skips a test only for an image listed as unavailable in the shared policy and fails for any other selection error (`backend/internal/lib/testcontainers/image.go:11-27`).
- CI places every physical end-to-end package except Patroni in `physical` (`mise-tasks/backend/test-group-packages:15-17`). Its shards run on 4-vCPU, 8 GB runners with `-parallel=2` (`.github/workflows/ci-release.yml:652-669,761-793`). A new `cnpg` package would join those shards silently. Script tests also pin the expected CI check names (`scripts/native_release_test.py:20`, `scripts/release_gate_test.py:153-157`, `scripts/backend_ci_execution_test.py`).
- `scripts/test.sh:197-198` runs slow packages, including end-to-end packages in a full run, with `-p=8 -parallel=8 -timeout 15m`. Its comment at `:189-195` records that Go's default limit started dozens of Patroni clusters at once and failed.

Constraints: root `AGENTS.md` (naming, logical units, blank lines, no unrequested compatibility, documentation consistency), `backend/AGENTS.md` (test libraries only in test files and test-support packages, `errors.go`, documented lint suppressions for process-level fixture synchronization), the testing skill (waits through `testwait`, no fixed sleeps, no fakes for our own code) and `docs/architecture.md:172` (test support may live in `internal/lib`, production binaries never depend on it). The change adds no production dependency and does not change the architecture.

## Goals / Non-Goals

**Goals:**

- Run every Patroni scenario family against CloudNativePG on PostgreSQL 17 and 18 with the same checks and shared assertions.
- Keep one implementation of each scenario, driven through a manager-independent cluster interface.
- Configure sources only through the CNPG manifest, as the user documentation recommends. The deliberate test-only differences are a small `wal_keep_size`, a short slot `updateInterval`, server certificates that name the k3s container address, and the foreign or mismatched certificates of the access scenarios.
- Make the suite cheap enough to run beside the other slow packages and in five CI shards per architecture.

**Non-Goals:**

- Testing Kubernetes itself, CNPG's backup integrations or a Databasus deployment inside Kubernetes.
- Changing Patroni scenario behavior. Patroni code changes only as far as the shared interface requires.
- A general Kubernetes test library for other engines.

## Decisions

### 1. Run Kubernetes as k3s in testcontainers containers

The fixture starts the pinned `rancher/k3s` image through `testcontainers.GenericContainer` as a privileged container on a network from `WithFreeSubnet`. It runs `k3s server` with Traefik, the service load balancer and metrics-server disabled, and with a kubelet pod limit raised above the capacity from decision 2. `/var/lib/rancher/k3s` lives on a labelled Docker volume, so containerd's snapshots and the PostgreSQL volumes sit on a real filesystem rather than on the container's overlay. The container, network and volume carry the fixture label and the run label, so `scripts/test.sh` removes them after a killed test process like any other fixture.

Right after start, the fixture checks its prerequisites and fails with a message that names the missing one:

- Docker is reachable.
- The container runs privileged. Docker's refusal, or k3s exiting with its cgroup or mount permission errors, is reported as "k3s requires a privileged container".
- The container's address accepts a TCP connection from the host on the Kubernetes API port. A failure is reported as "rootful Docker on Linux is required: the k3s container address is unreachable from the host".

Why k3s rather than kind: the task needs a real Kubernetes with the real CNPG operator, and both tools provide one, so the choice does not change what is verified. No requirement names kind; it was only a way to get a local Kubernetes without a cloud account. k3s fits the fixture conventions the repository already has, while kind would need its own code for each of them:

- **Cleanup.** The k3s container, network and volume are created by our code and carry the run label through `withTestRunLabel` (`backend/internal/lib/testcontainers/containers.go:203`), so `scripts/test.sh:32-66` removes them after a killed test process. kind creates its node containers itself and gives them only its own labels, so a killed run would need a separate search by cluster name.
- **Network isolation.** `WithFreeSubnet` gives the k3s network a random `/24` from the test range, as `docs/testing.md` requires. kind puts every cluster on its shared `kind` network unless the process environment variable `KIND_EXPERIMENTAL_DOCKER_NETWORK` redirects it. Parallel tests may not change the process environment, so cluster creation would have to move into `TestMain`.
- **Dependencies.** k3s is one more image in `images.json`, pinned by digest like every other test image. kind needs either the `sigs.k8s.io/kind` Go module or a binary pinned in mise on every machine and CI job.
- **Kubernetes configuration.** k3s is driven through `k3s kubectl` inside its container, with no kubeconfig on the host and no published API port. kind's library writes its context to the user's `~/.kube/config` unless every call passes an explicit path.
- **Footprint.** k3s runs as one process and keeps its state in SQLite instead of etcd. It usually starts faster and uses less memory than kind; task 7.2 measures the actual numbers.

The cost: k3s is not the Kubernetes that kubeadm assembles. It bundles its components into one binary and stores state in SQLite, but it is a certified Kubernetes distribution. The mechanisms these scenarios exercise run the same upstream code in it: Service selector to endpoints to kube-proxy, pod deletion, local-path volumes and the CNPG operator. Neither k3s nor kind would test etcd or kubeadm behavior. The user documentation states that the suite was verified on k3s.

Alternatives considered:

- **kind.** Rejected for the reasons above: it needs its own cleanup and network handling and adds a Go module or a pinned tool, while it would verify the same mechanisms.
- **The testcontainers-go `k3s` module.** Its v0.42.0 `go.mod` requires `k8s.io/client-go`, `k8s.io/api` and `k8s.io/apimachinery` 0.29.2. The few lines it saves do not justify that dependency tree.
- **An existing or long-lived cluster.** It breaks isolation between runs and worktrees and can touch a developer's real cluster.

### 2. A pool of shared Kubernetes clusters per package run, one namespace per scenario

Scenarios lease a place on a shared k3s container from a pool owned by the package run. The pool starts its first k3s container on the first lease and starts another one when every running k3s already holds its capacity of CNPG clusters. Each new k3s installs the CNPG operator and imports the images. The capacity is a measured constant: how many CNPG clusters one k3s runs without its API server or kubelet stretching scenario waits. Task 1.4 sets a provisional value and task 7.2 confirms it; the constant states the measurement beside it, as `docs/testing.md` asks for a resource limit.

Every scenario creates its own namespace with a CNPG `Cluster` of two PostgreSQL nodes, its own NodePort Kubernetes Service, certificate secrets and role secret. Deleting the namespace ends the lease. The package's `TestMain` stops the pool's k3s containers after `m.Run()` returns. The pool uses process-level synchronization with a documented lint suppression, as `backend/AGENTS.md` allows.

The pool limits how many CNPG clusters share one k3s, not how many scenarios run. A direct package run keeps Go's default `-parallel`, and the ten-minute bound is measured that way. With `-parallel=2` in CI one k3s serves the shard.

Lifecycle tests that check startup failures and cleanup start their own k3s containers outside the pool, because a partial startup cannot share a healthy cluster. A child process stopped by its parent leaves resources with its fixture label; the parent registers a fallback cleanup by that label before it starts the child, as `patroni/fixture_child_*_test.go` do.

Alternatives considered:

- **One k3s for the whole run.** With Go's default limit, 16 or more CNPG clusters would share one API server and kubelet, the failure the Patroni runner comment records at `scripts/test.sh:189-195`.
- **A k3s per scenario.** Each scenario would start Kubernetes, install the operator and import the PostgreSQL images again. With 32 scenarios that is 32 image imports to disk.
- **An explicit `-parallel` for the direct command.** The project rules leave concurrency to the runner, and a constant in the command would not follow the machine.

The operator is a resource shared within one k3s. A scenario only changes objects in its own namespace. Failover removes a pod in that namespace and never touches the operator or the k3s container.

### 3. Control Kubernetes through `k3s kubectl` inside the container

The fixture runs `k3s kubectl` through the Docker exec API, the same way the Patroni fixture talks to its containers. The Kubernetes API port is not published and no kubeconfig is written on the host. The fixture and the package never refer to `KUBECONFIG` or `~/.kube`, and task 1.5 checks this with a repository search. That search is how the "configuration left alone" scenario is verified: the code has no path that could read or write the developer's Kubernetes configuration. Manifests go in through stdin.

Alternatives considered:

- **client-go.** It is a heavy dependency for a test fixture and needs a published API port and a kubeconfig.
- **A host `kubectl` pinned in mise.** It adds a tool to every developer machine and CI job, and a kubeconfig file on the host.

### 4. Pin every image and the operator manifest, import images from the host

`backend/internal/lib/testimages/images.json` records immutable references and both platform digests for these images:

- k3s and the k3s system images that stay enabled: pause, CoreDNS and the local-path provisioner;
- the CNPG operator;
- CNPG's PostgreSQL 17 and 18 images.

When a k3s container starts, the fixture pulls the images through Docker and streams `docker save` output into `k3s ctr images import -` through Docker exec. Kubernetes pulls nothing from a registry, and no image archive is written on the host. `mise run backend:pull-testcontainers` prefetches the images for CI.

The CNPG release manifest for the pinned version lives in the fixture's test data. A README beside it records the release URL, the SHA-256 checksum and the publication date, as the Patroni fixture README does. The fixture applies it with server-side apply and waits for the operator Deployment and the CRDs to become ready.

Versions follow `docs/how-to-upgrade-deps.md`:

- the newest age-eligible CNPG 1.30 patch;
- a k3s release that CNPG 1.30 supports;
- CNPG PostgreSQL images on the same Debian release as the official `postgres:17` and `postgres:18` images used for booting (decision 11).

Alternatives considered:

- **Registry pulls inside k3s.** Every k3s start would download about a gigabyte, rate limits would make CI flaky, and the images would not be checked against the catalog digests.
- **An archive mounted into k3s's import directory.** It needs a host file that a killed run leaves behind, and streaming through exec does the same without it.
- **Downloading the manifest at test time.** The run would need network access to GitHub, and a manifest change would not show up as a reviewable diff.

### 5. Reach the primary through a NodePort Kubernetes Service with CNPG's `-rw` selector

Each scenario creates a NodePort Kubernetes Service whose selector equals the selector of CNPG's `-rw` Kubernetes Service, `cnpg.io/cluster=<name>` and `cnpg.io/instanceRole=primary`. Databasus connects to the k3s container's IP and the allocated node port. kube-proxy forwards to whichever pod CNPG labels primary, so a switch is observed exactly as an in-cluster client of `-rw` observes it.

After every switch the fixture checks route identity. Through the route, with the declared backup role, it reads `inet_server_addr()` and `pg_is_in_recovery()` on an SQL connection, and the timeline and system identifier with `IDENTIFY_SYSTEM` on a replication connection. It requires the answering pod IP to belong to the `Cluster`'s `.status.currentPrimary`. The test harness is not Databasus, so using the backup role for these probes does not hide a privilege Databasus would lack.

Alternatives considered:

- **Published host ports.** Docker publishes ports only when it creates a container, so a shared k3s would need a pool reserved in advance for scenarios that do not exist yet.
- **`kubectl port-forward` to a pod.** It bypasses the Kubernetes Service, so the test would no longer check that the route follows the primary.
- **Running Databasus inside k3s.** The test application runs in the test process and cannot connect to the in-cluster `-rw` Kubernetes Service without the routes above.
- **Also reading EndpointSlices.** The route identity already proves where the Kubernetes Service forwards, so the extra read observes nothing new.

The container IP is reachable from the host only with rootful Docker on Linux. That covers the Fedora development host and the Linux CI runners. Decision 1 turns any other engine into a named prerequisite failure.

### 6. Configure each source only through its CNPG manifest

The `Cluster` manifest declares:

- `instances: 2`, `bootstrap.initdb.dataChecksums: true`, and PostgreSQL defaults for `fsync`, `full_page_writes` and `synchronous_commit`.
- `postgresql.parameters` with `summarize_wal: "on"` and a small `wal_keep_size`. CNPG's default of 512 MB would hide slot loss within the scenario's WAL volume.
- A managed role, `managed.roles`, with `login` and `replication` and a `passwordSecret`, which is Databasus's backup role. It joins a NOLOGIN group role. `bootstrap.initdb.postInitSQL` creates that group and grants it `EXECUTE` on `pg_switch_wal()`, so the grant needs no role that exists only after bootstrap.
- `postgresql.pg_hba` rules that allow the backup role `hostssl` replication with SCRAM and reject it on non-TLS connections.
- `certificates.serverCASecret` and `serverTLSSecret`. Normally they come from a test certificate authority generated once per run, and the server certificate names the k3s container IP and the in-cluster Kubernetes Service names. Databasus gets that authority and connects with `verify-full`.
- `replicationSlots.synchronizeReplicas` and a short `replicationSlots.updateInterval`, set per slot mode (decision 9).

The two negative certificate cases of the access scenarios change only their own source, so the refusal comes from the certificate the source presents:

- **Another authority.** The scenario generates a second authority and gives its source `serverCASecret` and `serverTLSSecret` from it, so the CNPG cluster stays consistent internally. Databasus keeps the run's authority.
- **Name mismatch.** The scenario gives its source a server certificate from the run's authority that names only the in-cluster Kubernetes Service names, not the k3s container IP Databasus connects to.

The fixture observes and controls PostgreSQL nodes through `kubectl exec` into the pod's `postgres` container with the local `postgres` user over the Unix socket. That is how it reads replay positions and slots and plants inherited configuration. CNPG's network superuser access stays disabled, and the manifest declares no role besides the backup role and its group.

Alternatives considered:

- **Databasus creates the user.** That needs CNPG's superuser access to be enabled, and the task forbids using an administrative role for Databasus.
- **Enabling superuser access for the fixture.** It changes the documented source configuration only for test convenience, while local exec gives the same control.
- **CNPG's generated certificate authority.** Its server certificates do not name the container IP, so either the address or the verification would have to change.
- **Pointing Databasus at another authority for the first negative case.** The refusal would then come from Databasus's settings rather than from what the source presents.

### 7. Share the HA scenarios through a `FailoverCluster` interface

`physicaltest` defines the interface `FailoverCluster`, the role a cluster plays in these scenarios. Its operations are:

- the stable endpoint and route identity: PostgreSQL node, timeline and system identifier;
- a planned switch and an abrupt primary loss, each with the replay barrier inside, as in Patroni today;
- making the former primary unavailable;
- per-node admin SQL for observations and inherited configuration;
- waiting for the manager to finish its slot management.

Slot ownership stays a property of the source, chosen when the source is built, as `newPatroniSource` does today. The Patroni fixture and the new CNPG fixture implement the interface.

The scenario files move from `patroni_<scenario>.go` to `failover_<scenario>.go`, for example `failover_switchover.go`, `failover_slots.go`, `failover_interruption.go` and `failover_independent.go`. The source construction stays manager-specific in `patroni_source.go` and a new `cnpg_source.go`. Each suite passes its own source constructor. The Patroni suite keeps its test names and assertions; its routing stays HAProxy with plaintext, as its spec states.

Alternatives considered:

- **Copying the Patroni scenarios.** About 1,500 duplicated lines would drift apart as soon as one suite is fixed.
- **CNPG-only scenarios.** They would repeat assertions that are already proven on Patroni and lose the parity the user asked for.

### 8. Switch, fail over and isolate through CNPG's own mechanisms

- **Switchover.** The fixture patches the `Cluster` status subresource with the target primary and the switchover phase, the same patch the pinned CNPG `kubectl cnpg promote` plugin applies. The source reference is recorded beside the fixture code. It then waits until the route serves the target PostgreSQL node with a newer timeline.
- **Failover.** After the replay barrier confirms that the future primary has replayed the control transaction, the fixture force-deletes the primary pod with a zero grace period. It then requires a different serving PostgreSQL node, a newer timeline and an unchanged `system_identifier`. A restart of the same PostgreSQL node fails the scenario; it is not retried.
- **Former primary unavailable during independent recovery.** The fixture hibernates the `Cluster` declaratively with `cnpg.io/hibernation=on` and confirms that no pod of it runs. The restored cluster on the host cannot resolve in-cluster names, and the scenario also checks that the route refuses connections.

Alternatives considered:

- **The plugin binary.** It is one more per-architecture download to pin, for one status patch.
- **Killing postmaster.** CNPG's instance manager restarts PostgreSQL in the same pod, so no failover happens.
- **Stopping the k3s container.** It stops the whole Kubernetes cluster, including the PostgreSQL node that should take over.

### 9. Map slot modes to CNPG slot synchronization

| Mode | Used by | Patroni | CNPG |
|---|---|---|---|
| protected | slots, recovery | exact-name `ignore_slots` | `synchronizeReplicas.enabled: true`; before the switch the standby's copy must exist with `restart_lsn` no later than the first WAL Databasus has not archived |
| kept on every node | routing outages | `KeepSlotOnAllNodes`, a permanent slot | the same synchronization and pre-switch check as protected, so the outage scenarios meet an existing slot on the new primary |
| absent | slots | no slot on the candidate | `excludePatterns` with the exact escaped slot name; absence on the standby is observed before the switch |
| unprotected | slots | Patroni drops an inactive external slot | synchronization is turned off after the copy existed; CNPG's removal of the standby copy is observed before the switch |
| lost | slots | slot loss with recycled WAL | as absent; an identified required segment is proven recycled on the promoted PostgreSQL node and not archived |

"Waiting for management" waits for several observed `updateInterval` periods in which the standby's slot state is re-read, as Patroni waits for its cycles. Reception stays blocked through the proxy (decision 10) while the slot is inactive.

### 10. Terminate TLS in the test proxy for interception scenarios

`PostgresProxy` gains an optional TLS mode. It answers the client's `SSLRequest` and completes the handshake with a certificate from the run's test authority for its own address. Upstream it sends `SSLRequest` to the NodePort and verifies the server certificate against the same authority with the expected name. Databasus keeps `verify-full` toward the proxy, and the proxy keeps it toward the source, so verification is never disabled. Protocol parsing, `BASE_BACKUP` gating, replication blocking and the outage switch work unchanged on the decrypted stream. Scenarios without interception connect to the Kubernetes Service directly.

The routing-outage scenarios keep using the proxy's outage switch. Its events prove that the failure recheck met the outage, which is what the requirement checks.

Alternatives considered:

- **Disabling TLS in those scenarios.** The task forbids it.
- **A byte-level TCP proxy with server-side observation through `pg_stat_progress_basebackup`.** It cannot tell replication connections from SQL connections and cannot hold a transfer at a known command, so the switch would land at an uncontrolled moment.
- **An outage through an empty Kubernetes Service selector.** kube-proxy updates its rules asynchronously, so the outage window is not deterministic, and nothing records that the recheck reached it.

### 11. Boot recovered clusters with the official image of the same major

Recovery stays outside Kubernetes through the existing host recovery helpers, booted from `postgres:17` or `postgres:18` (`physicaltest/cluster_directory.go:17-37`). Users restore on hosts without CNPG's instance manager, and CNPG's images have no standalone entrypoint. The CNPG PostgreSQL images are pinned to the same Debian release as the official images, so the C library collations match. Every boot checks that no database reports a collation version mismatch.

Alternative considered: **booting the CNPG image directly.** It needs a custom entrypoint and does not reflect how users recover.

### 12. Placement and names in local runs and CI

- Code names use the abbreviation `cnpg` in one form: the package `cnpg`, the CI group `cnpg` and test names such as `TestCNPGRecoveryPostgres17`. Prose writes "CloudNativePG (CNPG)". The glossary entry records the abbreviation.
- The package joins `backend/test-slow-packages.txt`. `mise run test` excludes it like every end-to-end package. `mise run test-full` runs it in the slow group with the runner's limits. The direct package command uses Go's defaults and a 15-minute timeout.
- Scenarios are top-level tests per family and major, mirroring Patroni, with weights in `backend/test-shard-weights.txt` measured from two complete local runs.
- `mise-tasks/backend/test-group-packages` gains `cnpg` and excludes it from `physical`. The workflow gains five `cnpg` shards per architecture. Before tests, the job raises `fs.inotify.max_user_instances` and `max_user_watches`, because k3s's kubelet and containerd use inotify instances that the runner's default of 128 may not cover. The runner size is chosen from the measured memory peak of one shard at `-parallel=2` (task 7.2). The script tests that pin CI check names gain the `cnpg` shards.

## Risks / Trade-offs

- [CNPG rejects `summarize_wal` or another required parameter] → Task 1.3 checks the live settings first. If CNPG forbids a parameter that INCREMENTAL needs, the scenario cannot be built as specified; stop and report rather than weaken it.
- [k3s differs from kubeadm-built Kubernetes] → The scenarios exercise upstream mechanisms that k3s runs unchanged; the user documentation names k3s as the verified environment and does not claim other distributions.
- [k3s in a privileged container fails on the host's cgroup v2 or SELinux setup] → The data directory is on a volume. Startup failures print k3s logs and name the prerequisite, as decision 1 describes.
- [The capacity of one k3s is misjudged] → Task 1.4 measures a provisional value, and task 7.2 confirms it with the complete package under Go's default limit and under `-parallel=2`.
- [A force-deleted primary container keeps running briefly] → The scenario requires a different serving PostgreSQL node and a newer timeline through the route, and it checks the former pod's absence before recovery.
- [Synchronized slot positions lag by `updateInterval`] → A short interval in the manifest, and waits on the observed `restart_lsn` instead of on time.
- [The TLS-terminating proxy differs from a direct connection] → Only interception scenarios use it. Initial recovery, switchovers, failover and access scenarios connect straight to the Kubernetes Service.
- [Collations of the CNPG image and the boot image differ] → Pin the same Debian release and fail on a reported collation version mismatch.
- [The scenario refactor changes Patroni behavior] → The Patroni suite runs complete after the refactor and before CNPG scenarios are added.
- [Disk use grows with many PostgreSQL nodes on k3s volumes] → The tasks sample Docker storage and `/tmp` during the measured runs, as the Patroni parallelization did, and record the peak.
- [A killed direct package run leaves resources] → A direct run has no run label, so its resources carry only the fixture label, which other worktrees' active runs share. `docs/testing.md` names that label with this warning. `mise run test-full` removes its own run's resources by the run label.
- [Terms collide with the glossary] → "Operator" already means a human (`docs/glossary.md:150`), "Service" means the backend's service layer (`:133`), and "instance" means a Databasus instance (`:30-37`). The specs, the user document and the glossary write "CNPG operator", "Kubernetes Service" and "PostgreSQL node".

## Migration Plan

There is no runtime migration. The Patroni suite is refactored first and must pass unchanged. The CNPG fixture, scenarios, CI group and documentation follow. Rollback reverts the change; the refactored Patroni scenarios keep working because they do not depend on the CNPG code.

Documentation:

- `docs/testing.md`: a CloudNativePG section with the command, the prerequisites (rootful Docker on Linux, privileged containers, inotify limits) and the fixture label that finds a killed direct run's resources, with a warning that the label also matches other worktrees' active runs.
- `docs/postgresql-cloudnativepg.md` for users:
  - the Kubernetes Service address to use inside and outside the Kubernetes cluster;
  - the declared role, its grant and HBA;
  - TLS certificates whose names match the address Databasus uses;
  - `summarize_wal` and data checksums;
  - slot synchronization modes and `wal_keep_size`;
  - recovery limits and the configuration outside the data directory;
  - the verified environment: k3s in a privileged container on rootful Docker on Linux, with no claim for other Kubernetes distributions;
  - the verified versions.
- `docs/postgresql-patroni.md`: its statement that Kubernetes and operators are unverified now links to the CNPG document for CNPG.
- `docs/machine-setup.md`: the inotify limits.
- `docs/glossary.md`: entries for CloudNativePG (CNPG), with `cnpg` as the code abbreviation, and Kubernetes Service, each set apart from the existing "Service" and "operator" entries.

## Open Questions

- The CI runner size for `cnpg` shards. It follows from the measured peak in task 7.2 and changes neither the specs nor the task breakdown.
