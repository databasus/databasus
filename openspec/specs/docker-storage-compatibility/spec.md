# Docker Storage Compatibility Specification

## Purpose

Defines the supported Docker account settings and the real-container filesystem tests that prevent storage permission changes from breaking common installation layouts.

## Requirements

### Requirement: Docker services share one configurable runtime identity

The Docker image SHALL run Databasus and embedded PostgreSQL under one non-root operating-system account named `databasus`. Operators MAY set its numeric identity with `PUID` and `PGID`. Each supplied value SHALL be a non-zero decimal Linux ID within the supported system range.

When an override is absent, startup SHALL select that ID independently from the existing PostgreSQL data owner, then the mounted backup or data-root owner and finally `999`. Root ownership SHALL NOT be selected automatically.

#### Scenario: Fresh installation uses the fallback identity

- **WHEN** an operator starts the image without identity overrides or existing non-root ownership
- **THEN** all bundled services run as user ID `999` and group ID `999` under the account name `databasus`

#### Scenario: Existing data determines the identity

- **WHEN** an existing PostgreSQL data directory has a non-root numeric owner and `PUID` or `PGID` is absent
- **THEN** startup uses the corresponding owner ID before considering other mounts or the fallback

#### Scenario: Operator supplies identity overrides

- **WHEN** an operator supplies valid `PUID` and `PGID` values
- **THEN** all bundled services run with those numeric IDs

#### Scenario: Operator overrides one identity value

- **WHEN** an operator supplies either `PUID` or `PGID` and mounted data provides the other numeric ID
- **THEN** startup combines the explicit value with the automatically selected counterpart

#### Scenario: Operator supplies an invalid identity

- **WHEN** `PUID` or `PGID` is empty, non-numeric, zero or outside the supported range
- **THEN** the container exits before starting its services
- **AND** the log identifies the invalid variable

### Requirement: Startup validates storage through required operations

Startup SHALL attempt to prepare the data root, PostgreSQL data, temporary and backup paths with the selected identity and required modes. A failed ownership or mode change SHALL NOT stop startup when the selected account can still perform every required operation.

Startup SHALL run the main Databasus binary with `--test-storage` under the selected account before starting PostgreSQL or the Databasus application. The storage probe SHALL save a unique file through the configured local storage and remove that file through the same storage path. Startup SHALL also verify the operations required for PostgreSQL data and Databasus control files. A failed check SHALL stop startup before PostgreSQL or the Databasus application runs. Startup SHALL NOT start or wait for a separate cache service.

#### Scenario: Filesystem denies metadata changes but permits operations

- **WHEN** a mounted filesystem rejects ownership or mode changes but the selected account can perform all required operations
- **THEN** the container starts successfully

#### Scenario: Selected identity creates missing service directories

- **WHEN** container root cannot create a required directory but the selected account can create it
- **THEN** startup creates the directory as the selected account and continues

#### Scenario: Backup publication is not permitted

- **WHEN** the selected account cannot publish a file from the temporary path to the backup path
- **THEN** the container exits before starting the application
- **AND** the English error names the affected path, required operation, effective user and group IDs and `https://databasus.com/advanced-config/#docker-storage-permissions`

#### Scenario: Mounted path is read-only

- **WHEN** any required mounted path is read-only
- **THEN** the container exits before starting PostgreSQL or the Databasus application
- **AND** the log provides the same identity and documentation details

#### Scenario: Data root cannot create control files

- **WHEN** temporary, backup and PostgreSQL data paths are writable but the data root cannot create a control file
- **THEN** the container exits before starting PostgreSQL or the Databasus application
- **AND** the English error identifies the data root and required operation

#### Scenario: Container starts its bundled services

- **WHEN** storage validation succeeds
- **THEN** startup proceeds directly to embedded PostgreSQL and the Databasus application without a cache-service readiness step

### Requirement: Existing Docker data remains usable after an image update

The current image SHALL start with persistent data created by versions `v3.54.0`, `v3.55.0`, `v3.55.1`, `v3.56.0` and `v3.60.0` when the mounted filesystem permits the selected account's required operations. Startup SHALL preserve the metadata database, secret key, application log, existing backup files, resumable nested WAL queues, and the existing PostgreSQL cluster. It SHALL prepare an existing WAL queue recursively and verify directory creation plus non-mutating file opens under the selected account before PostgreSQL or the Databasus application starts.

#### Scenario: Existing installation is updated

- **WHEN** an operator replaces one of the covered earlier images with the current image without adding identity settings
- **THEN** the current container becomes healthy
- **AND** the metadata database, secret key, application log, existing backup, nested WAL queue, and PostgreSQL data remain available

### Requirement: Container preserves incompatible-data startup guards

The container SHALL refuse startup when it detects data at the deprecated Postgresus mount location or a metadata database containing a PostgreSQL configuration with backup type `WAL_V1`.

#### Scenario: Postgresus data directory is mounted

- **WHEN** the deprecated Postgresus mount location exists and contains data
- **THEN** the container exits before starting its services
- **AND** its log identifies the unsupported directory

#### Scenario: WAL_V1 configuration exists

- **WHEN** the metadata database contains a PostgreSQL configuration whose backup type is `WAL_V1`
- **THEN** the container exits before starting the Databasus application
- **AND** its log identifies the unsupported backup type

### Requirement: One filesystem suite runs locally and in CI

The repository SHALL provide `mise run test-filesystem` as the single entry point for the Docker filesystem suite. It SHALL build or accept a local candidate image and run the same cases locally and in CI. The suite SHALL sort every case into exactly one case group: `permissions`, `network`, `upgrades` or `lifecycle`. It SHALL run every group when no group or case is selected, the cases of one group when a group is selected, and a single case when a case is selected. An unknown group SHALL fail the run before any case starts.

CI SHALL build the candidate once and run each case group as its own check from that same candidate. The candidate SHALL pass between CI jobs only as a short-lived workflow artifact and SHALL NOT be pushed to or pulled from an image registry.

Each positive case SHALL start the real container and invoke `databasus --test-storage`, which uses the public local-storage save and delete methods. Each negative case SHALL assert the startup failure and its actionable error. Every case SHALL clean its containers, volumes, networks, mounts, and temporary files.

#### Scenario: Developer runs the complete suite

- **WHEN** a developer runs `mise run test-filesystem` without an image reference, a group or a case
- **THEN** the task builds a local candidate and runs every case group against it

#### Scenario: Developer runs one group

- **WHEN** a developer runs `mise run test-filesystem` with the `network` group selected
- **THEN** only the cases of the `network` group run

#### Scenario: An unknown group is selected

- **WHEN** a developer selects a case group that does not exist
- **THEN** the run fails with an error that names the group, and no case starts

#### Scenario: CI tests its candidate

- **WHEN** CI tests a commit
- **THEN** one job builds the candidate, each case group runs as its own check against that candidate, and no candidate image is pushed to or pulled from a registry

#### Scenario: A case allocates Docker resources

- **WHEN** the case succeeds or fails after resource allocation
- **THEN** the harness removes every resource created by that case

### Requirement: The matrix covers supported mount behavior

The suite SHALL cover an ext4 bind-mounted data root, a Docker named volume, separate temporary, backup, and PostgreSQL mounts, automatic and explicit runtime IDs, rejected backup and data-root permission layouts, CIFS numeric ownership and forced file modes, writable NFS with root squash, a container that cannot create Unix sockets, and upgrades from the covered earlier images. It SHALL verify one non-root numeric identity for all bundled services, runtime PostgreSQL authentication, the absence of a PostgreSQL Unix socket, and restrictive modes for generated credential files.

#### Scenario: Local Docker storage layouts

- **WHEN** the candidate uses an ext4 bind mount, named volume, or separate mounts
- **THEN** the container becomes healthy and the storage command saves and deletes its probe

#### Scenario: Temporary and backup paths use different filesystems

- **WHEN** the temporary and backup paths have different device IDs
- **THEN** `LocalStorage.SaveFile` succeeds through its supported cross-filesystem path and `LocalStorage.DeleteFile` removes the probe

#### Scenario: CIFS maps numeric ownership

- **WHEN** a real CIFS fixture presents root ownership, numeric group ownership `999`, directory mode `0770`, and mount-enforced file modes
- **THEN** the container becomes healthy and completes the storage command
- **AND** startup skips the root user ID and writes through group `999`

#### Scenario: NFS data storage denies ownership changes to container root

- **WHEN** a root-squashed NFS data root has no backups directory, rejects ownership and mode changes, remains writable by the selected identity, and local mounts provide PostgreSQL and temporary storage
- **THEN** the selected identity creates the backups directory and the container becomes healthy
- **AND** the storage command succeeds without requiring ownership changes

#### Scenario: Container cannot create Unix sockets

- **WHEN** the container runs with Unix-socket creation denied for every process
- **THEN** the container becomes healthy and completes the storage command

#### Scenario: Mounted path denies required access

- **WHEN** a mounted path does not permit an operation required by the selected identity
- **THEN** startup fails with the documented actionable error

### Requirement: Startup does not depend on Unix-socket support in the data mount

The container SHALL start when the mounted data root cannot hold Unix sockets, provided the selected account can perform every other required operation there. Startup SHALL NOT create, prepare or verify a PostgreSQL socket directory. A socket directory left by an earlier image SHALL be ignored.

#### Scenario: No Unix socket can be created in the container

- **WHEN** no process in the container can create a Unix socket, in the data root or anywhere else, as happens on a host folder that Docker Desktop on macOS shares as the data root
- **THEN** the embedded PostgreSQL server starts and the container becomes healthy
- **AND** the container lists no PostgreSQL Unix socket

#### Scenario: Socket directory from an earlier image

- **WHEN** the data root contains a PostgreSQL socket directory created by an earlier image
- **THEN** startup leaves the directory in place and ignores it
- **AND** the container becomes healthy

### Requirement: A write-ahead log reset is limited to a cluster that cannot be opened

Startup SHALL reset the embedded cluster's write-ahead log only when the cluster cannot be opened for startup administration, which includes replaying the existing log. After a reset, startup SHALL try to open the cluster once more, and SHALL exit with an error that reports the failed recovery when the reset or that attempt fails. When administration succeeds and the regular server then fails to start, the container SHALL exit with an error and SHALL NOT reset the write-ahead log.

#### Scenario: Interrupted shutdown

- **WHEN** the previous container stopped without a clean PostgreSQL shutdown and the write-ahead log is intact
- **THEN** startup replays the log and keeps every committed transaction
- **AND** the container becomes healthy without a write-ahead log reset

#### Scenario: Missing checkpoint record

- **WHEN** the cluster cannot be opened because its write-ahead log lacks a valid checkpoint record
- **THEN** the log reports a write-ahead log reset
- **AND** the container becomes healthy

#### Scenario: Reset does not make the cluster open

- **WHEN** the write-ahead log reset fails, or the cluster still cannot be opened after the reset
- **THEN** the container exits with an error before the Databasus application starts
- **AND** the log reports the failed recovery

#### Scenario: Server start fails after administration

- **WHEN** startup administration succeeds but the regular server cannot start
- **THEN** the container exits with an error
- **AND** the write-ahead log is not reset
