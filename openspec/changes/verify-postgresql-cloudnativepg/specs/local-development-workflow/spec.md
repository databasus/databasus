## ADDED Requirements

### Requirement: The CloudNativePG suite finishes within ten minutes on the reference machine

The reference developer machine has at least 10 performance cores and 32 GB of memory free for tests, out of about 64 GB in total. On that machine, the complete CloudNativePG physical backup suite on PostgreSQL 17 and 18, run alone as a direct package run with Go's default parallel test limit, SHALL finish within ten minutes, with about five minutes as the target. The suite SHALL NOT limit how many of its scenarios run at once below the parallel test limit the test runner is given, so the runner's limit alone decides its concurrency locally and in CI. Running more scenarios at once than the machine has processor cores is acceptable; the memory free for tests is the bound. In these scenarios a running CloudNativePG cluster is one CloudNativePG `Cluster` resource whose PostgreSQL nodes are running; the Kubernetes clusters that host them are counted separately.

#### Scenario: The suite runs alone on the reference machine

- **WHEN** a developer runs the complete CloudNativePG suite as a direct package run with Go's default parallel test limit on the reference machine
- **THEN** it finishes within ten minutes
- **AND** every backup and fixture lifecycle scenario on both PostgreSQL majors runs and passes, with none skipped

#### Scenario: A runner limit above two is reached

- **WHEN** the test runner is given a parallel test limit N greater than two and at least N CloudNativePG scenarios are waiting to run
- **THEN** N CloudNativePG `Cluster` resources with running PostgreSQL nodes are observed at the same time

#### Scenario: The CI limit of two holds

- **WHEN** the test runner is given a parallel test limit of two, as CI uses
- **THEN** no more than two CloudNativePG `Cluster` resources of that run with running PostgreSQL nodes are observed at the same time

### Requirement: The CloudNativePG suite owns its Kubernetes clusters

Each run of the CloudNativePG suite SHALL create its own disposable Kubernetes clusters, with names, networks, volumes, credentials and files no other run uses. Their networks SHALL take subnets from the test network range that the testing documentation names. The suite SHALL NOT read or change the developer's Kubernetes configuration or selected context, and SHALL NOT connect to a Kubernetes cluster it did not create.

After success, a failure during startup or cancellation, the run SHALL remove the containers, networks, volumes and temporary files of its Kubernetes clusters, and only those. When the test process of `mise run test-full` is killed, the surviving script SHALL remove the run's labelled containers, networks and volumes when it exits. A killed direct package run carries no run label, so it leaves its resources and its temporary directories behind. The testing documentation SHALL name the fixture label that finds those resources and SHALL say that this label also matches resources of runs still active in other worktrees.

The suite needs a Docker daemon that runs privileged containers and makes container addresses reachable from the host, which is rootful Docker on Linux. A missing prerequisite, such as an unavailable Docker daemon, a refused privileged container, an unreachable container address or an image that cannot be obtained, SHALL fail the suite with a message that names it instead of skipping scenarios.

#### Scenario: Two worktrees run the suite at once

- **WHEN** two worktrees on one machine run the CloudNativePG suite at the same time
- **THEN** each run uses only its own Kubernetes clusters and both pass as they do alone
- **AND** neither run's cleanup removes a resource of the other

#### Scenario: The developer's Kubernetes configuration is left alone

- **WHEN** a developer with a configured Kubernetes context runs the suite
- **THEN** their Kubernetes configuration file and selected context are unchanged after the run
- **AND** no scenario connects to a Kubernetes cluster from that configuration

#### Scenario: Startup fails part way

- **WHEN** a Kubernetes cluster or the CloudNativePG operator fails to start after some of the run's resources were created
- **THEN** the suite fails with the cause and diagnostics
- **AND** every resource the run created is removed

#### Scenario: A scenario is cancelled

- **WHEN** a run's context is cancelled while scenarios hold their CloudNativePG clusters
- **THEN** the run's Kubernetes clusters, networks, volumes and temporary files are removed

#### Scenario: The test process of a full run is killed

- **WHEN** the test process of `mise run test-full` is killed while CloudNativePG scenarios run
- **THEN** after the script exits, no container, network or volume with that run's label remains

#### Scenario: A prerequisite is missing

- **WHEN** Docker is unavailable, a privileged container is refused, the container address of a Kubernetes cluster is unreachable from the host or a pinned image cannot be obtained
- **THEN** the suite fails with a message naming the missing prerequisite
- **AND** no scenario is reported as skipped or passed
