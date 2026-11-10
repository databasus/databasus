## Purpose

Restores a lost Databasus installation from an encrypted Databasus backup file and its separately retained key without allowing recovery to upgrade the snapshot or start competing work.

## ADDED Requirements

### Requirement: Recovery requires the exact source release

Startup recovery SHALL accept a Databasus backup file only when its authenticated source release exactly matches the running Databasus release, including the patch version, and its schema identity is compatible with the running build under the rules below. Both older and newer releases SHALL be refused before importing data, applying application migrations or replacing the destination key. The refusal SHALL name the source and running releases and instruct the owner to start the source release, recover there, and then upgrade normally. There SHALL be no override accepting a different release.

Schema identity SHALL contain a digest of the ordered migration bundle and the effective applied migration versions captured in the snapshot. Compatibility SHALL require an identical bundle digest, every migration carried by the target recorded as applied in the snapshot, and no applied version newer than the target's newest migration. Additional historical applied versions absent from the current bundle SHALL be accepted when they are no newer than that newest version, as ordinary upgraded installations permit. Those historical entries SHALL be preserved. Recovery SHALL compare the imported migration history with the authenticated snapshot identity before activation; it SHALL NOT require the entire applied-version set to equal the current bundle's version set.

#### Scenario: Newer Databasus tries to restore an older backup

- **WHEN** an archive from release X is supplied to a different newer release Y
- **THEN** recovery stops without modifying the destination and tells the owner to recover on X before upgrading

#### Scenario: A patch release differs

- **WHEN** an archive and running instance differ only in patch version
- **THEN** recovery is refused with the same guidance

#### Scenario: Development builds share a release label

- **WHEN** source and target have the same development release label but different migration-bundle digests
- **THEN** recovery is refused before modifying the destination

#### Scenario: Same-release recovery retains retired historical migrations

- **WHEN** an upgraded source and target use the exact same release and migration bundle, and the source records all current migrations plus older applied entries absent from that bundle
- **THEN** recovery accepts the Databasus backup file and preserves those historical entries without rerunning or deleting them
- **AND** no application migration is applied as part of recovery

### Requirement: Startup recovery accepts an archive and a separate key only on an unused target

The supplied deployment SHALL support startup recovery from an archive file and a separate key file. Recovery SHALL run before normal application initialization, SHALL require an unused target, and SHALL NOT overwrite an existing installation. It SHALL authenticate and validate the complete archive before publishing recovered state. A wrong key, truncated or modified archive, incompatible schema or failed import SHALL leave no apparently usable partial installation. A completed recovery SHALL not repeat merely because recovery inputs remain configured across a restart.

#### Scenario: Recovery on a fresh target

- **WHEN** the owner supplies a valid archive and its correct key to an unused target of the exact source release
- **THEN** startup recovers the snapshot and installs that key before opening the recovered application

#### Scenario: Recovery on an existing installation

- **WHEN** the target already contains an installation
- **THEN** recovery is refused without changing its accounts, data or key

#### Scenario: Damaged archive or incorrect key

- **WHEN** the archive is corrupted or the supplied key does not decrypt it
- **THEN** recovery fails without activating partial recovered state or replacing an existing key

#### Scenario: Restart after successful recovery

- **WHEN** a successfully recovered instance restarts with the same recovery inputs still configured
- **THEN** it starts from the recovered state without importing the archive again

### Requirement: Recovery starts in a durable pause

A recovered instance SHALL remain paused across restarts until root admin explicitly confirms that the old instance is stopped. While paused, it SHALL permit authentication, inspection and the activation action, but SHALL start no backup, restore, verification or healthcheck execution, send no automatic notifications and perform no remote write or deletion, including through manual actions or previously issued download/action tokens with side effects. The interface SHALL explain the pause, require explicit root confirmation before activation, and keep the pause visible during a pending or failed activation. Only successful server acknowledgment SHALL clear it. Other administrators SHALL NOT activate it. Restarting or upgrading SHALL NOT clear the pause.

#### Scenario: Owner inspects a recovered instance

- **WHEN** root admin signs in after recovery and has not confirmed activation
- **THEN** the interface explains the pause and the need to stop the old instance
- **AND** no background or manual operation changes a remote storage or source database

#### Scenario: Another administrator attempts activation

- **WHEN** an administrator other than root admin requests activation
- **THEN** the request is refused and the instance stays paused

#### Scenario: Activation fails after confirmation

- **WHEN** root admin confirms that the old instance is stopped but the activation request fails
- **THEN** the interface keeps the pause explanation and shows an actionable error without claiming that execution has resumed

#### Scenario: Activation succeeds after confirmation

- **WHEN** root admin explicitly confirms and the server acknowledges successful activation
- **THEN** the interface clears the instance-wide pause while independently retaining any local-storage validation restriction

### Requirement: Recovered runtime state does not claim work survived the lost server

Recovery SHALL end operations that were in progress at the snapshot as interrupted, invalidate sessions, temporary action/download credentials and verification-agent credentials, discard obsolete runtime claims and local work, and preserve the original schedules without executing them until activation. It SHALL discard source-instance unfinished-write and deletion obligations rather than replay potentially stale cleanup against surviving storages, and SHALL inform root admin that leftover files can require manual cleanup. Local storage SHALL be unavailable until validated on the new host as the requirement below specifies. After activation, physical backup creation SHALL start a new full chain rather than continuing a chain whose local WAL queue was lost. Previously completed backup history SHALL remain available, subject to its actual remote files and coverage at the snapshot; files created after the snapshot SHALL NOT be claimed as recovered history.

#### Scenario: Physical backup configuration survives recovery

- **WHEN** the recovered instance is activated and a physical backup becomes due
- **THEN** it requests a new full backup and does not rely on the lost local WAL queue

#### Scenario: A remote logical backup existed at the snapshot

- **WHEN** the recovered instance is activated and the owner restores a cataloged logical backup whose remote files still exist
- **THEN** its recovered credentials and encryption key permit restoration

#### Scenario: A completed physical restore set existed at the snapshot

- **WHEN** the owner downloads and restores a cataloged physical restore set after activation
- **THEN** its retained full backup, increments, WAL and timeline history support the recovery target that was available at the snapshot

### Requirement: Recovered local storages remain unavailable until validated on the new host

Recovered local storages SHALL expose a requires-validation state and an explanation that the original host files were not restored by the Databasus backup. Activation alone SHALL NOT make them available. Until a successful validation, automated and manual backup writes, backup reads/downloads/restores and file cleanup against them SHALL be blocked with an explicit storage-unavailable outcome. Other validated destinations SHALL continue to operate independently.

After activation, the existing saved connection-test action SHALL be the only file-access exception to the unavailable state. It SHALL check the configured local location and its read/write access using a probe. A successful probe by a user authorized to edit that storage SHALL make it available only if its saved settings have not changed since the probe began. Viewers SHALL retain permission to test saved storages, but their results SHALL remain diagnostic only and SHALL NOT change availability. The interface SHALL explain when an editor's validation is still required. A failed probe SHALL retain the unavailable state and show a safe failure reason. Before instance activation the recovery pause SHALL still block a writing probe. Validation SHALL NOT claim that missing historical backup files were recovered.

#### Scenario: Activation leaves local storage unvalidated

- **WHEN** root admin activates the recovered instance without validating its local storage
- **THEN** its interface still reports requires-validation and backup, restore/download and cleanup operations against it remain blocked

#### Scenario: Local storage validation succeeds

- **WHEN** an authorized editor validates a local storage after activation and its configured location passes the read/write probe
- **THEN** the storage becomes available for ordinary operations
- **AND** historical files not present on the new host remain unavailable rather than being reported as recovered

#### Scenario: Local storage validation fails

- **WHEN** validation cannot read or write the configured local location
- **THEN** the storage remains unavailable with a safe diagnostic and its backup, restore/download and cleanup operations stay blocked

#### Scenario: Viewer tests connectivity without validating recovery

- **WHEN** a viewer successfully tests a recovered local storage after activation
- **THEN** the test reports connected but the storage remains unavailable pending an authorized editor's validation
