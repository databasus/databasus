## Context

See `proposal.md` for the motivation and scope. When this design was written, no CloudNativePG scenario had run and no runtime defect was assumed; defects reproduced during apply are recorded in [Defects reproduced during apply](#defects-reproduced-during-apply).

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
- Keep the entire local run within the 32 GB test memory budget on the 32-vCPU, 64-GB reference host, and size CI shards from measured runtime and memory.

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

Found during apply:

- The image declares `/var/lib/kubelet`, `/var/lib/cni` and `/var/log` as volumes. Without a mount, Docker creates anonymous volumes that carry no label and survive a killed run, so the fixture mounts tmpfs over them.
- testcontainers' Docker discovery falls back to the default socket when `DOCKER_HOST` names an unreachable daemon. The fixture therefore pings the daemon that `DOCKER_HOST` or the default socket names before anything else, so an unreachable daemon is reported instead of hidden behind another one.
- A TCP connection alone does not prove reachability: on the development host, an unrouted address accepted connections. The fixture completes a TLS handshake with the Kubernetes API verified against the k3s server authority. The lifecycle test of an unreachable address uses an unassigned address of the fixture's own `/24`.

### 2. A pool of shared Kubernetes clusters per package run, one namespace per scenario

Scenarios lease a place on a shared k3s container from a pool owned by the package run. The pool starts its first k3s container on the first lease and starts another one when every running k3s already holds its capacity of CNPG clusters. Each new k3s installs the CNPG operator and imports the images. The capacity is a measured constant: how many CNPG clusters one k3s runs without its API server or kubelet stretching scenario waits. Task 1.4 sets a provisional value and task 7.2 confirms it; the constant states the measurement beside it, as `docs/testing.md` asks for a resource limit.

Every scenario creates its own namespace with a CNPG `Cluster` of two PostgreSQL nodes, its own NodePort Kubernetes Service, certificate secrets and role secret. Deleting the namespace ends the lease. The package's `TestMain` stops the pool's k3s containers after `m.Run()` returns. The pool uses process-level synchronization with a documented lint suppression, as `backend/AGENTS.md` allows.

The pool limits how many CNPG clusters share one k3s, not how many scenarios run. A direct package run keeps Go's default `-parallel`, and the ten-minute bound is measured that way. CI parallelism and runner size follow the measured shard load, within 16 vCPUs and 32 GB RAM. Fixture-control tests can create several CNPG clusters within one test; their cluster count is measured separately from scenario parallelism.

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

Found during apply:

- k3s pulls its system images from the registry as soon as it starts. The fixture configures a mirror without endpoints for every registry and starts k3s with `--disable-default-registry-endpoint`, so a missing imported image fails a pod instead of downloading an unpinned one. System pods still waiting for an image after the import are recreated, so they do not sit out the kubelet's pull back-off.
- The local-path provisioner creates volume directories with a busybox helper pod, so `rancher/mirrored-library-busybox` is pinned and imported with the other system images.
- The import starts after the Kubernetes node is Ready. Before that, k3s may still be bringing containerd up, and an import then failed mid-stream with a refused containerd socket.
- The release manifest sets the operator's `imagePullPolicy: Always`. The fixture replaces that one line with `IfNotPresent` when it applies the manifest and checks the manifest's SHA-256 first.
- The k3s image has no `k3s ctr` or `k3s kubectl` subcommand in this release; the fixture runs the `ctr` and `kubectl` links to the same binary.

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

Found during apply: the scenarios write control rows, change the schema, force WAL switches and checkpoints, and read progress views and source files. The harness does this as the database owner that `bootstrap.initdb` creates, connected through the route over verified TLS. `postInitApplicationSQL` grants that owner `pg_checkpoint`, `pg_monitor` and `EXECUTE` on `pg_switch_wal()`, `pg_read_file(text)` and `pg_read_binary_file(text)`. These are privileges of the test harness's application role, not of Databasus: the backup role keeps only what the manifest declares, and the scenarios keep checking its restrictions. `synchronizeReplicas` starts disabled and each source enables it with its slot mode once Databasus has named its slot.

Found during apply: Databasus refuses to save a source whose certificate it cannot verify; creating the database answers `400` with `database_connection_failed` and the TLS verification error. The refused-certificate scenarios therefore save the source while it presents the run's trusted certificate, then rotate the `server-tls` and `server-ca` secrets to the refused certificate and wait until the routed primary presents it. The harness verifies its own control connections against the certificate's real issuer and a name it holds, so only what Databasus verifies differs.

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

Found during apply: recovery, switchover, primary loss, interrupted transfers and routing outages run through one implementation in `failover_<scenario>.go`. The slot and independent-recovery families differ in what they exercise, not only in how a manager is driven. Patroni's protected slot is never switched, while the CNPG case requires a switch onto the standby's synchronized copy. Patroni's unprotected slot is dropped on the primary, while CNPG removes only the standby's copy. CNPG's independent recovery hibernates the whole source and finds its certificates outside the data directory. These two families therefore keep a scenario per manager, `patroni_slots.go`, `patroni_independent.go`, `cnpg_slots.go` and `cnpg_independent.go`, on shared helpers, and the `FailoverCluster` interface holds only what the shared scenarios use.

### 8. Switch, fail over and isolate through CNPG's own mechanisms

- **Switchover.** The fixture patches the `Cluster` status subresource with the target primary and the switchover phase, the same patch the pinned CNPG `kubectl cnpg promote` plugin applies. The source reference is recorded beside the fixture code. It then waits until the route serves the target PostgreSQL node with a newer timeline.
- **Failover.** After the replay barrier confirms that the future primary has replayed the control transaction, the fixture resolves the primary pod's `postgres` container ID and sends `SIGKILL` to every process in that container with `ctr -n k8s.io tasks kill --all --signal SIGKILL`. It requires that command to succeed, then force-deletes the pod with a zero grace period to prevent a same-node restart from satisfying failover. It then requires a different serving PostgreSQL node, a newer timeline and an unchanged `system_identifier`. A restart of the same PostgreSQL node fails the scenario; it is not retried.
- **Former primary unavailable during independent recovery.** The fixture hibernates the `Cluster` declaratively with `cnpg.io/hibernation=on` and confirms that no pod of it runs. The restored cluster on the host cannot resolve in-cluster names, and the scenario also checks that the route refuses connections.

Alternatives considered:

- **The plugin binary.** It is one more per-architecture download to pin, for one status patch.
- **Killing postmaster alone.** CNPG's instance manager can restart PostgreSQL in the same pod. Killing the whole container and deleting the pod prevents that restart from satisfying the different-node check.
- **Force-deleting the pod alone.** Kubelet can still grant a minimum termination grace period, allowing clean PostgreSQL shutdown; it does not guarantee abrupt loss.
- **Stopping the k3s container.** It stops the whole Kubernetes cluster, including the PostgreSQL node that should take over.

### 9. Map slot modes to CNPG slot synchronization

| Mode | Used by | Patroni | CNPG |
|---|---|---|---|
| protected | slots, recovery | exact-name `ignore_slots` | `synchronizeReplicas.enabled: true`; before the switch the standby's copy must exist with `restart_lsn` no later than the first WAL Databasus has not archived |
| kept on every node | routing outages | `KeepSlotOnAllNodes`, a permanent slot | synchronization enabled; before the switch, observe completed management cycles and require the candidate slot to retain at least the primary slot's current `restart_lsn`, with a current segment still unarchived |
| absent | slots | no slot on the candidate | `excludePatterns` with the exact escaped slot name; absence on the standby is observed before the switch |
| unprotected | slots | Patroni drops an inactive external slot | synchronization is turned off after the copy existed; CNPG's removal of the standby copy is observed before the switch |
| lost | slots | slot loss with recycled WAL | as absent; an identified required segment is proven recycled on the promoted PostgreSQL node and not archived |

"Waiting for management" waits for several observed `updateInterval` periods in which the standby's slot state is re-read, as Patroni waits for its cycles. Reception stays blocked through the proxy (decision 10) while the slot is inactive.

Found during apply: a cycle is observed, not timed. The fixture advances a probe slot on the primary and waits until the standby's copy reaches that position, three times, then drops the probe so it retains no WAL.

### 10. Terminate TLS in the test proxy for interception scenarios

`PostgresProxy` gains an optional TLS mode. It answers the client's `SSLRequest` and completes the handshake with a certificate from the run's test authority for its own address. Upstream it sends `SSLRequest` to the NodePort and verifies the server certificate against the same authority with the expected name. Databasus keeps `verify-full` toward the proxy, and the proxy keeps it toward the source, so verification is never disabled. Protocol parsing, `BASE_BACKUP` gating, replication blocking and the outage switch work unchanged on the decrypted stream. Scenarios without interception connect to the Kubernetes Service directly.

The routing-outage scenarios keep using the proxy's outage switch. Its events prove that the failure recheck met the outage, which is what the requirement checks.

Alternatives considered:

- **Disabling TLS in those scenarios.** The task forbids it.
- **A byte-level TCP proxy with server-side observation through `pg_stat_progress_basebackup`.** It cannot tell replication connections from SQL connections and cannot hold a transfer at a known command, so the switch would land at an uncontrolled moment.
- **An outage through an empty Kubernetes Service selector.** kube-proxy updates its rules asynchronously, so the outage window is not deterministic, and nothing records that the recheck reached it.

Found during apply: SCRAM channel binding hashes the certificate the client sees into the authentication, so a source refuses every login through a proxy that presents a different certificate. The proxy therefore presents the source's own server certificate, which also names the proxy's loopback address, and verifies the source against the same authority. Both legs keep `verify-full`.

### 11. Boot recovered clusters with the official image of the same major

Recovery stays outside Kubernetes through the existing host recovery helpers, booted from `postgres:17` or `postgres:18` (`physicaltest/cluster_directory.go:17-37`). Users restore on hosts without CNPG's instance manager, and CNPG's images have no standalone entrypoint. The CNPG PostgreSQL images are pinned to the same Debian release as the official images, so the C library collations match. Every boot checks that no database reports a collation version mismatch.

Alternative considered: **booting the CNPG image directly.** It needs a custom entrypoint and does not reflect how users recover.

### 12. Placement and names in local runs and CI

- Code names use the abbreviation `cnpg` in one form: the package `cnpg`, the CI group `cnpg` and test names such as `TestCNPGRecoveryPostgres17`. Prose writes "CloudNativePG (CNPG)". The glossary entry records the abbreviation.
- The package joins `backend/test-slow-packages.txt`. `mise run test` excludes it like every end-to-end package. `mise run test-full` runs it in the slow group with the runner's limits. The direct package command uses Go's defaults and a 15-minute timeout.
- Scenarios are top-level tests per family and major, mirroring Patroni, with weights in `backend/test-shard-weights.txt` measured from complete local runs and refreshed after the review repairs.
- `mise-tasks/backend/test-group-packages` gains `cnpg` and excludes it from `physical`. The workflow gains five `cnpg` shards per architecture, initially on 8-vCPU / 16-GB runners with Go's default scenario parallelism; measured race-enabled shard runs validate this selection. Before tests, the job raises `fs.inotify.max_user_instances` and `max_user_watches`, because k3s's kubelet and containerd use inotify instances that the runner's default of 128 may not cover. Runner size and parallelism are chosen from measured shard duration and peak memory (task 7.2), with up to 16 vCPUs and 32 GB RAM available. Small jobs retain small runners to conserve the paid runner vCPU concurrency quota. The script tests that pin CI check names gain the `cnpg` shards.

Found during apply (task 7.2 measurements, local amd64 host with 32 threads and 64 GB, no race detector; complete package):

- Go's default limit, host at `fs.inotify.max_user_instances = 512`: two runs took 491 s and 343 s. Every scenario passed, but two `TestCNPGFixtureCleanup` cases failed in each run because k3s ran out of inotify instances (`error creating fsnotify watcher: too many open files`) at about 15 k3s containers. The documented 1024 instances and 1048576 watches were then set on the host, and both default runs are repeated under them.
- `-parallel=4`: 1683 s, at most 4 running CNPG clusters on 4 k3s containers, memory at most 4.4 GiB above the 13.5 GiB the host used before the run.
- `-parallel=2`: 3433 s, memory at most 3.4 GiB above the host's 13.4 GiB. At most 3 CNPG clusters ran at once. Cluster count can exceed scenario parallelism because fixture-control tests create several clusters; there is no acceptance limit of two clusters.
- Five shards at `-parallel=2` give about 690 s of tests per shard before CI's race detector, over the ten-minute limit. The selected configuration therefore removes the two-test limit and raises the CNPG runner size, retaining five shards. The 15-minute timeout applies to the documented default-parallelism command; the slower comparison runs used 90m and 150m timeouts.

Final runner selection after raising inotify limits:

- Two complete default-limit runs on the 32-vCPU host passed without skips in 586.317 s and 549.716 s. Peak host-memory increases were 14.78 GiB and 17.33 GiB, below the 32 GB test budget. They meet the ten-minute limit with little margin, but not the five-minute target.
- An additional complete run after repairing the startup-publication race passed in 489.705 s. Its sampler included every k3s fixture, including lifecycle children with their own run IDs, and observed 16 k3s and 30 running CNPG clusters, with a 14.87-GiB increase in host memory.
- CNPG keeps five shards per architecture on 8-vCPU / 16-GB runners, with Go's default test parallelism instead of `-parallel=2`. Race-enabled local checks at `GOMAXPROCS=8` passed shards 1, 2 and 3 in 326.337 s, 484.562 s and 329.252 s. In that preceding assignment, shard 3 had the largest weight and shard 2 included the two-k3s capacity check. Shards 2 and 3 overlapped and together increased host memory by about 10.23 GiB. These are local concurrency measurements, not a native CI timing or ARM execution claim.
- The complete runs confirmed four CNPG clusters per pooled k3s. Fixture controls deliberately hold several clusters in one scenario, so cluster count is separate from scenario parallelism. The two-k3s capacity check retains both leases until its scenario ends.
- The initial weights were the rounded-up means of summed leaf durations from the two successful default-limit runs; later complete runs refreshed them after the review repairs. Exact commands, resource-counter limits and cleanup evidence are in `verification.md`.

The first post-review default-limit run passed all 62 scenarios in 481.764 s
(484 s including startup), with no skips. Global fixture sampling observed
20 k3s and 32 running CNPG clusters; host memory increased by 16.31 GiB,
within the 32 GB local test budget. The weights then used this run's summed
leaf durations. With that assignment, race-enabled shards 1
(cleanup, slots and interrupted backups) and 5 (controls and recovery on both
majors) passed at `GOMAXPROCS=8` in 316.887 s and 429.771 s. They overlapped
and together increased host memory by 11.18 GiB, supporting the selected
8-vCPU / 16-GB runners. `verification.md` records the exact commands,
selections and measurement limits.

The latest complete default-limit run passed all 65 scenario leaves, the
no-op child entry and all 19 top-level tests in 407.613 s (409 s including
startup). Host memory increased by 17.55 GiB; global fixture sampling observed
18 k3s and 32 running CNPG clusters. Fixture switchover, failover and slot
controls have separate test files. Every current weight uses this run's
summed leaf durations. The five shard weights are 1985, 1955, 1976, 1972 and
2001 s. Current shard 1 and largest-weight shard 5 passed with `-race` at
`GOMAXPROCS=8` in 267.890 s and 306.329 s. They ran concurrently and increased
aggregate host memory by 10.95 GiB, within one selected runner's 16-GB budget.
The final child/root ownership metadata was verified by the cleanup family
and these CI checks after the full run. Exact selections, measurement limits
and the preceding runs are in `verification.md`.

## Risks / Trade-offs

- [CNPG rejects `summarize_wal` or another required parameter] → Task 1.3 checks the live settings first. If CNPG forbids a parameter that INCREMENTAL needs, the scenario cannot be built as specified; stop and report rather than weaken it.
- [k3s differs from kubeadm-built Kubernetes] → The scenarios exercise upstream mechanisms that k3s runs unchanged; the user documentation names k3s as the verified environment and does not claim other distributions.
- [k3s in a privileged container fails on the host's cgroup v2 or SELinux setup] → The data directory is on a volume. Startup failures print k3s logs and name the prerequisite, as decision 1 describes.
- [The capacity of one k3s is misjudged] → Task 1.4 measures a provisional value, and task 7.2 confirms it with the complete package under Go's default limit and the selected CI configuration.
- [Kubelet grants termination grace after force-deletion] → The fixture first sends SIGKILL to all primary-container processes through containerd. It then deletes the pod and requires a different serving node, a newer timeline and the old pod UID's absence before recovery.
- [Synchronized slot positions lag by `updateInterval`] → A short interval in the manifest, and waits on the observed `restart_lsn` instead of on time.
- [The TLS-terminating proxy differs from a direct connection] → Only interception scenarios use it. Initial recovery, switchovers, failover and access scenarios connect straight to the Kubernetes Service.
- [Collations of the CNPG image and the boot image differ] → Pin the same Debian release and fail on a reported collation version mismatch.
- [The scenario refactor changes Patroni behavior] → The Patroni suite runs complete after the refactor and before CNPG scenarios are added.
- [Disk use grows with many PostgreSQL nodes on k3s volumes] → The tasks sample Docker storage and `/tmp` during the measured runs, as the Patroni parallelization did, and record the peak.
- [A killed direct package run leaves resources] → A direct run has no run label, so its resources carry only the fixture label, which other worktrees' active runs share. `docs/testing.md` names that label with this warning. `mise run test-full` removes its own run's resources by the run label.
- [Terms collide with the glossary] → In `docs/glossary.md` an operator is the person who runs a console command, "Service" means the backend's service layer, and "instance" means a Databasus instance, as in "Instance admin" and "Instance settings". The specs, the user document and the glossary write "CNPG operator", "Kubernetes Service" and "PostgreSQL node".

## Defects reproduced during apply

### The recovery script called a CloudNativePG copy ready although it could not start

- Reproducer: the initial recovery scenario, `TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -count=1 -timeout 15m -run '^TestCNPGRecoveryPostgres1[78]$/restores_initial' ./internal/modules/backups/e2e/postgresqlphysical/cnpg`. On the unfixed script the restored server stopped at startup because its server certificate file, `/controller/certificates/server.crt`, did not exist, while the script had printed its usual next steps without any action required.
- Cause: CloudNativePG keeps its server certificate, key and CA, the socket directory and the log directory under `/controller`, outside the data directory, and names them in the copied `custom.conf`. A base backup copies only the data directory. The script reported only configuration files missing from the copy, not settings that point to files outside it.
- Final behavior: after sanitizing the copied configuration, the script lists every `ssl_cert_file`, `ssl_key_file`, `ssl_ca_file`, `ssl_crl_file`, `ssl_crl_dir`, `ssl_dh_params_file`, `krb_server_keyfile`, `log_directory` and `unix_socket_directories` setting whose absolute path lies outside the source data directory, with the file that sets it. It prints `ACTION REQUIRED: the restored configuration points outside the data directory` with the values to override in `postgresql.auto.conf`, and its summary says the copy will not start until then. Paths under the source data directory travel with the copy and are not reported. The CNPG scenarios follow those instructions (`ssl = off`, a local socket directory, `log_directory = 'log'`) and keep the copied `pg_hba.conf` and `pg_ident.conf`.
- Verification: the reproducer above passed on PostgreSQL 17 and 18 after the fix, `shellcheck backend/internal/modules/backups/internal/physical/restorestream/recovery_script.sh` reports nothing, and both `TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -count=1 -timeout 15m ./internal/modules/backups/physicalapitests/restoring`, whose package serves the script, and the complete Patroni suite, `go -C backend test -count=1 -timeout 90m ./internal/modules/backups/e2e/postgresqlphysical/patroni` with Ryuk enabled, passed. `verification.md` records the complete runs.

### The connection test hid why a replication connection failed

- Reproducer: `TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -count=1 -timeout 20m -run '^TestCNPGAccessPostgres1[78]$/refuses' ./internal/modules/backups/e2e/postgresqlphysical/cnpg`, and the engine test `TestTestReplicationConnection/*/names_the_certificate_verification_failure_when_another_authority_issued_the_server_certificate` in `backend/internal/engines/postgresql/physical`. On the unfixed code both reported the error as the bare `CONNECTION_FAILED`.
- Cause: `classifyReplicationConnectError` in `backend/internal/engines/postgresql/physical/replication_readiness.go` mapped every unclassified replication connection failure to `CONNECTION_FAILED` with an empty message, so the connection test returned the code as its error text and dropped the driver's TLS verification error. Creating the same database already returned the full text.
- Final behavior: an unclassified connection-attempt failure keeps `CONNECTION_FAILED` as its reason and the driver's host and TLS diagnostic. A configuration parse failure instead returns `invalid PostgreSQL connection configuration`, without the embedded connection string. A certificate from another authority reports `x509: certificate signed by unknown authority`, a certificate without the address reports `x509: certificate is valid for 127.0.0.1, not <address>`, and the frontend shows that text under the classified failure.
- Verification: both reproducers pass after the fix, with `TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -count=1 -timeout 15m -run 'TestTestReplicationConnection' ./internal/engines/postgresql/physical` for the whole engine test.

### Concurrent pool startup read its error before publication

- Reproducer: CNPG shard 1 with `GOMAXPROCS=8`, `-race`, and the recorded shard selection. The race detector reported a write to `pooledK3s.err` after `StartK3s` against a read while another independent-recovery scenario selected a pool member. The original trace is retained in `/tmp/cnpg-measure/ci-1-before-fix.jsonl`.
- Cause: the pool mutex protected selection and lease counts, but startup wrote the error outside that mutex. Selection read the error even while startup was still running.
- Final behavior: selection reads a startup error only after the member's `ready` channel is closed. That channel publishes the completed startup state; an unfinished startup can still accept leases up to its capacity. Kubernetes startup stays outside the pool mutex, so independent k3s containers can start together.
- Verification: rerun the same race-enabled shard and the fixture-heavy shards, recording the results in `verification.md`.

### Cancelling an attached Docker exec could block fixture cleanup

- Reproducer: `go -C backend test -count=1 -timeout 2m -run '^TestDockerExec$' ./internal/lib/testcontainers`. Both real-container cases waited until a command had attached, then cancelled its context. Before the fix, each failed because exec remained blocked, with and without a streaming stdin. Evidence: `/tmp/cnpg-measure/docker-exec-before.log`.
- Cause: the Docker client's hijacked connection does not follow context cancellation after attachment. Reading output and waiting for the stdin copy could therefore block indefinitely.
- Final behavior: cancellation closes the attached stream and its closable stdin; exec joins the input goroutine and reports the context error. An output-read error also closes stdin before joining it. Normal output completion lets the input copy finish before closing its underlying archive. The callback is joined before returning.
- Verification: both reproductions passed after the fix; the full suites exercise image imports and kubectl through the same path.

### Killing a lifecycle parent left a child outside the enclosing run's cleanup

- Reproducer: `TestCNPGFixtureCleanup/stops_the_fixture_child_after_its_parent_is_killed` starts a parent and its child, waits for a live CNPG cluster, then kills the parent. Before the watchdog fix, the child remained alive until the probe failed. Evidence: `/tmp/cnpg-measure/orphan-before.jsonl`.
- Cause: children occupied separate process groups and replaced the enclosing run ID. Killing the parent neither stopped its children nor preserved ownership for the surviving script's run-label cleanup.
- Final behavior: lifecycle children inherit the enclosing run ID by default. A child with an explicit different ID retains the enclosing ownership in `databasus.test-root-run`; the shell runner cleans the union of its own and root labels using its exact run ID. A private inherited pipe ties each child to its parent; EOF stops the child's process group, including after SIGKILL. A nested-parent probe verifies termination and removal of the recorded resources while an unrelated network survives. Cancellation is also tested after CNPG readiness.
- Verification: record the post-fix lifecycle and full-suite results in `verification.md`.

### Recovery and refusal checks needed observable processing evidence

The initial recovery now builds a completed FULL → INCREMENTAL chain, verifies that the downloaded bundle contains both manifests, and restores distinct data written before FULL, before INCREMENTAL and after INCREMENTAL through WAL. Certificate-refusal cases wait for the WAL worker's certificate connection error before asserting no archived WAL, no completed backup and non-healthy status. The plaintext refusal requests physical replication with `replication=true`. CI execution tests fail each inotify sysctl in turn and verify preparation order and failure propagation.

### TLS proxy cancellation raced with startup negotiation

- Reproducer: the complete CNPG package with `-race`, recorded in `/tmp/cnpg-measure/repairs-cnpg.jsonl`, reported a read in the connection cancellation callback concurrent with replacing the raw connection by the negotiated TLS connection.
- Cause: the callback captured the mutable `client` variable. Failed negotiation could also replace it with nil while cancellation read it.
- Final behavior: the callback captures the original raw connection, whose identity never changes; closing it also interrupts the TLS wrapper. A proxy test cancels after the SSL response while the TLS handshake is still pending.
- Verification: the proxy and Docker exec cancellation tests run with `-race`; final suite results are recorded in `verification.md`.

The slot probes also distinguish resumed reception of new WAL from continuity of an earlier interval. After the worker has processed a resumed stream, a recovery token for the earlier target must either boot the recorded state or remain refused as `target_time_unreachable`. A new active slot and newly archived segments alone cannot establish the target's availability.

### Startup diagnostics could prevent cleanup when the Kubernetes API stalled

- Reproducer: `TestCNPGFixtureCleanup/removes_owned_resources_after_cancellation_with_a_stalled_Kubernetes_API` starts a real k3s API, stops the server process beneath its init process and verifies that an API request times out before cancelling startup. Without the diagnostics deadline, kubectl made repeated TLS attempts and startup took 104.89 s; the regression failed because the bounded context error was absent. The regression now also requires completion within one minute after the API stalls. Preparation failures in the earlier `stalled-*` probes are not evidence of this defect.
- Cause: failed startup collected diagnostics using `context.WithoutCancel` before termination, without a new deadline. A stalled API could keep the attached kubectl command open indefinitely and prevent resource removal.
- Final behavior: all startup diagnostics share a ten-second deadline. Expiry interrupts the attached exec, then startup terminates its owned container, volume, network and temporary directory. The regression also preserves an unrelated network.
- Verification: the bounded implementation passed with `-race` in 16.913 s (`stalled-threads`). The causal before/after and complete runs are recorded in `verification.md`.

### The missing-WAL refusal needed a same-timeline control

The promotion probe still verifies lost-slot behavior across a switchover. A separate case now drops the slot without changing timelines, recycles an unreceived segment, then archives transactions beyond the target on that same timeline. An earlier target remains available through an eligible FULL and its contiguous prefix. The target inside the missing interval must report `WAL gap before target`, both before and after a later FULL completes. This distinguishes the gap from timeline eligibility and from a target that simply has no backup before it. Both PostgreSQL majors passed with `-race` (`gap-fixed`).

### Cancelling upstream SSL negotiation could leave proxy cleanup waiting

- Reproducer: `TestPostgresProxy/returns_after_cancellation_while_the_upstream_withholds_its_SSL_response` completes client TLS startup, observes the proxy's SSLRequest on a real TCP peer and withholds the response. Before the fix, cancellation left forwarding blocked until the five-second probe failed (`upstream-before.log`). The peer is closed on failure so the probe itself still joins its goroutines.
- Cause: upstream cancellation was registered only after TLS negotiation returned. The initial SSL response read did not observe context cancellation.
- Final behavior: cancellation closes the immutable raw upstream connection as soon as dialing succeeds, before SSL negotiation. The client leg already closes its immutable raw connection. Normal forwarding retains its TLS cleanup.
- Verification: the new regression, all existing proxy cases and Docker exec cancellation passed with `-race` in 2.203 s (`upstream-after.log`). The regression also requires EOF on the upstream peer after cancellation. Both managers' routing-outage families on both PostgreSQL majors and the routine command are repeated in `verification.md`.


### Access, transfer gating and overlapping cleanup require observed effects

The declared backup role now attempts both text and binary reads of
`postgresql.conf` and `pg_hba.conf`, requiring SQLSTATE `42501` before and
after promotion. The TLS proxy probe consumes initial backup data, observes
a blocked read while the gate is held and observes resumed data after
release. Removing only the gate's wait makes that probe fail. The cleanup
probe requires both differently labelled child fixtures to be ready before
releasing one, then observes removal of that child's resources while the
other remains live.

### Distinct child run IDs must retain enclosing cleanup ownership

Inspecting live fixture labels exposed that inheriting only the enclosing
run ID did not exercise distinct child IDs. Children now keep their requested
`databasus.test-run` and preserve the enclosing ID in
`databasus.test-root-run`. The shell runner exports the root ID for every
descendant and cleans resources matching either exact ownership label,
deduplicating IDs. Its interruption regression failed before root-label
cleanup was added and passed afterward. Real child-process checks inspect
both labels on every container, network and volume; unrelated resources
survive cleanup. The complete cleanup family and the current race-enabled
CI candidates passed with this ownership model.


### FTP fixture authentication exceeded the connection deadline under load

- Reproducer: the final routine command failed `TestTestStorageSettings/connects_to_the_entered_FTP_server_and_creates_no_storage_when_settings_are_tested_before_saving`; the connection-test request exhausted its ten-second deadline. The assertion printed a string pointer rather than the returned error, so it now compares the full response to expose any future diagnostic. Five isolated repeats passed, confirming sensitivity to the full-run conditions rather than invalid credentials.
- Investigation: the fixture uses real pure-ftpd authentication. [Its password-calibration code](https://github.com/jedisct1/pure-ftpd/blob/1.0.50/src/pure-pw.c#L254-L302) defaults to eight concurrent logins. A controlled real-container probe calibrated the hash before applying the same 0.2-CPU limit to each server. The default hash timed out at 10.012 s; `pure-pw -C 32` authenticated successfully in 4.705 s. Both used the same account, server image and protocol. An earlier 0.25-CPU comparison took 9.182 s and 2.605 s respectively. This reproduces the CPU-sensitive authentication deadline; the original failing response text is unavailable.
- Final behavior: the test fixture supplies `-C 32` when generating its password hash, matching concurrent fixture authentication on the reference host. Authentication still verifies the password against a real calibrated hash. Application connection deadlines, refusal checks and runner concurrency are unchanged.
- Verification: the existing storage API tests exercise successful and refused authentication. Their race-enabled package result and the repeated routine command are recorded in `verification.md`, alongside the causal probe and the original failure.


### WAL publication tests removed storage before draining their applications

- Reproducer: the post-FTP-repair routine command failed `TestLateWalStoragePublication/queues_late_files_without_publishing_into_the_replaced_storage` during application cleanup. Its backup-run drain reached the 30-second deadline, followed by the notification wait using that expired context. The test body itself had passed.
- Cause: both MinIO namespaces were created after their applications registered cleanup. LIFO cleanup could therefore remove a storage container while a replacement FULL still used it. This was sensitive to whether the backup finished before the assertions did.
- Final behavior: the existing application shutdown operation is exposed as `App.Stop`. Each concurrent publication case registers both applications' shutdown after creating the target storage, so both drain while the storage is still available. The ordinary cleanup later calls the same operation again. Existing cancellation, drain deadlines and test assertions remain in force.
- Verification: both late-publication cases passed with `-race` in 41.962 s (`wal-cleanup-fixed.log`); the complete routine command is repeated afterward. These changes affect test shutdown order, not backup behavior.


### Force-deleting a pod did not guarantee an unclean primary loss

Review found that Kubernetes can grant a minimum shutdown window even after
`--grace-period=0 --force`. Different-pod and promotion checks therefore did
not establish the required crash. The failover control now sends SIGKILL to
all processes in the identified primary PostgreSQL container, asserts command
success and records the container ID before deleting the pod. It preserves
the confirmed-replay barrier, different serving node, newer timeline,
unchanged system identifier and removed-pod checks. Both fixture failover
and booted recovery after primary loss are repeated on both PostgreSQL majors
with `-race`; exact results are recorded in `verification.md`.


### A driver configuration error could reveal an escaped password suffix

- Reproducer: `TestTestReplicationConnectionConfiguration` calls the real source readiness operation with an invalid CA and passwords containing an apostrophe, a backslash or both. Before the fix, pinned pgx's `ParseConfigError` partially masked the conninfo password but left the synthetic `PRIVATE_SUFFIX` visible in the apostrophe cases. The backslash case still exposed the connection string. Evidence is `escaped-password-before.log`.
- Cause: generic connection-error preservation included `pgconn.ParseConfigError`, which embeds the original conninfo string and relies on incomplete password masking. The connection-test response must never include that string.
- Final behavior: the classifier recognizes wrapped `*pgconn.ParseConfigError` through `errors.AsType` and returns `CONNECTION_FAILED` with `invalid PostgreSQL connection configuration`. It never formats the parser error or returns its chain. Actual connection failures retain their useful certificate and host diagnostics, including both CNPG certificate-refusal cases.
- Verification: all three real-parser regressions passed with `-race` after the fix. The complete routine command and both majors' CNPG certificate-refusal cases are repeated in `verification.md`.

The node-command contract and execution now live in `cnpg_node_command.go`;
SQL query types and methods remain in `cnpg_node_sql.go`. Independent target
recovery on both majors verifies the command callers after this file split.


### Routing-outage scenarios must observe the synchronized slot before promotion

Review found that `NewCNPGManager` discarded `ShouldKeepSlotOnEveryNode`.
Synchronization was enabled, but the routing-outage cases never observed
Databasus's slot on the candidate before switching. A later recreation could
therefore hide its earlier absence.

The requested mode now wraps the source's switch operation with an explicit
retention check. After observed management cycles, it reads the primary's
current external-slot position and the candidate's copy. The copy must exist
with a nonempty `restart_lsn` no later than the primary's position, so it
retains every byte the source slot still protects for Databasus. The check
also observes a current segment absent from the archive. It records both
positions and the segment before calling the switch, then requires that
candidate to serve the route. It never creates Databasus's slot itself.

WAL reception stays active during this observation. Blocking every
replication connection would also close the held `BASE_BACKUP`, ending the
interrupted-transfer experiment before the planned promotion. The separate
protected-slot scenario still blocks reception to prove continuity of its
controlled earlier interval. Routing-outage and protected-slot cases are
repeated with `-race` on both majors; results are in `verification.md`.

## Migration Plan

There is no runtime migration. The Patroni suite is refactored first and must pass unchanged. The CNPG fixture, scenarios, CI group and documentation follow. Rollback reverts the change; the refactored Patroni scenarios keep working because they do not depend on the CNPG code.

Documentation:

- `docs/testing.md`: a CloudNativePG section with the command, the prerequisites (rootful Docker on Linux, privileged containers, inotify limits) and the fixture label that finds a killed direct run's resources, with a warning that the label also matches other worktrees' active runs; the resource section records the 32-vCPU local reference and its total 32 GB test budget, and CI runners up to 16 vCPUs / 32 GB with smaller runners for small jobs.
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

None. The recorded local measurements select 8-vCPU / 16-GB CNPG runners; native CI results remain the workflow's responsibility.
