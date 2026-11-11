## Purpose

Restores Databasus from an encrypted backup file and its separately retained master key through mounted startup files or confirmed UI file import, without upgrading the snapshot or starting competing work.

## ADDED Requirements

### Requirement: Recovery requires the exact source release

Both startup recovery and UI file import SHALL accept a Databasus backup file only when its authenticated source release exactly matches the running Databasus release, including the patch version, and its schema identity is compatible with the running build under the rules below. Both older and newer releases SHALL be refused before importing data, applying application migrations or replacing the destination key. The refusal SHALL name the source and running releases and instruct the owner to start the source release, recover there, and then upgrade normally. There SHALL be no override accepting a different release. Successfully consumed startup inputs SHALL be skipped on later normal starts as specified below; leaving them configured SHALL NOT turn an ordinary upgrade into a new recovery attempt.

Schema identity SHALL contain a digest of the ordered migration bundle and the effective applied migration versions captured in the snapshot. Compatibility SHALL require an identical bundle digest, every migration carried by the target recorded as applied in the snapshot, and no applied version newer than the target's newest migration. Additional historical applied versions absent from the current bundle SHALL be accepted when they are no newer than that newest version, as ordinary upgraded installations permit. Those historical entries SHALL be preserved. Recovery SHALL compare the imported migration history with the authenticated snapshot identity before activation; it SHALL NOT require the entire applied-version set to equal the current bundle's version set.

#### Scenario: Newer Databasus tries to restore an older backup

- **WHEN** an archive from release X is supplied to a different newer release Y
- **THEN** recovery stops without modifying the destination and tells the owner to recover on X before upgrading

#### Scenario: A patch release differs

- **WHEN** an archive and running instance differ only in patch version
- **THEN** recovery is refused with the same guidance

#### Scenario: UI import uses a different release

- **WHEN** root admin uploads a valid archive from a different release, including a patch difference
- **THEN** import is refused with the source and running releases and guidance to recover on the source release
- **AND** the current installation's data and master key remain unchanged

#### Scenario: Development builds share a release label

- **WHEN** source and target have the same development release label but different migration-bundle digests
- **THEN** recovery is refused before modifying the destination

#### Scenario: Same-release recovery retains retired historical migrations

- **WHEN** an upgraded source and target use the exact same release and migration bundle, and the source records all current migrations plus older applied entries absent from that bundle
- **THEN** recovery accepts the Databasus backup file and preserves those historical entries without rerunning or deleting them
- **AND** no application migration is applied as part of recovery

### Requirement: Startup recovery accepts mounted files on a fresh or automatically initialized target

The supplied Docker deployment SHALL support startup recovery from an archive file and a separate master-key file supplied through paths in a mounted volume, including the persistent data volume. Both paths SHALL be required together. A new startup recovery SHALL run before normal application initialization and SHALL accept either a fresh target or one whose state was created entirely by automatic initialization. Previous ordinary starts, existing application tables, recorded migrations, unmodified default settings and an automatically generated target master key SHALL NOT by themselves prevent recovery. Supplying both files and configuring both paths after an ordinary start SHALL permit recovery at the next start without deleting the volume, provided the target still meets these conditions. Adding files alone SHALL NOT trigger recovery in a running instance.

Startup recovery SHALL refuse to replace a target containing any user account, including a sole root account, other user data or user-changed persisted settings. Checking only account count SHALL NOT establish eligibility. Unknown or unverifiable state SHALL be refused, and exclusive publication SHALL recheck eligibility so concurrently created user state cannot be overwritten. The newer-schema refusal SHALL remain in force. An ineligible target SHALL remain unchanged and the refusal SHALL direct the owner to confirmed UI import or a separate fresh target.

Recovery SHALL authenticate and validate the complete archive before publishing recovered state. A wrong key, truncated or modified archive, incompatible schema or failed import SHALL preserve any existing target database and key and SHALL leave no apparently usable partial recovered installation. On success, the source key SHALL replace an automatically generated target key only as part of consistent publication.

#### Scenario: Recovery on a fresh target

- **WHEN** the owner supplies a valid archive and its correct key to a fresh target of the exact source release
- **THEN** startup recovers the snapshot and installs that key before opening the recovered application

#### Scenario: Owner adds recovery files after the first ordinary start

- **WHEN** the owner starts Databasus normally without recovery inputs, stops it without creating accounts, user data or changing persisted settings, then adds the archive and source key to the same mounted volume and restarts the exact source release with both paths configured
- **THEN** startup accepts the automatically initialized target without requiring deletion of its volume
- **AND** successful recovery replaces its automatically generated key with the exact source key and opens the recovered instance paused

#### Scenario: A root account exists without workspace data

- **WHEN** the target contains only a user-created root account in addition to automatically initialized state
- **THEN** startup recovery is refused without changing the account, database or key
- **AND** the owner is directed to confirmed UI import or a separate fresh target

#### Scenario: No account exists but persisted user state remains

- **WHEN** the target has no accounts but contains user data or user-changed persisted settings
- **THEN** startup recovery is refused without changing that state or its key

#### Scenario: Target state cannot be classified safely

- **WHEN** recovery cannot establish that the target contains only known automatic initialization state
- **THEN** it refuses recovery without changing the target database or key

#### Scenario: User state appears before publication

- **WHEN** a target passes initial eligibility checks but another process creates user state before exclusive publication
- **THEN** publication is refused and preserves the new user state and target key

#### Scenario: Automatically initialized target has a newer schema

- **WHEN** the target has only automatic initialization state but records migrations newer than the running image carries
- **THEN** startup recovery refuses without replacing the database or key and retains the newer-schema diagnostic

#### Scenario: Damaged archive or incorrect key

- **WHEN** the archive is corrupted or the supplied key does not decrypt it
- **THEN** recovery fails without activating partial recovered state or replacing an existing database or key, including an automatically initialized target's key

### Requirement: Completed startup recovery is deduplicated by archive content

Startup recovery SHALL identify an archive by SHA-256 of the complete encrypted file. Successful consumption SHALL survive replacement of snapshot state and SHALL be recorded only after the database, exact source key and recovery pause are durably published. Restarting with the same successfully consumed archive SHALL start the current installation without importing it again, even when the file is renamed, an explicit UI import has since replaced the snapshot or the application is being upgraded normally. A different archive at the same configured path SHALL NOT be treated as consumed and SHALL be refused on a target containing user accounts, user data or user-changed settings. A failed attempt SHALL NOT be marked successfully consumed. An interrupted accepted recovery SHALL resume or finish its recorded publication safely rather than begin another import. A deliberate new UI import of identical bytes SHALL remain possible; retries of one accepted operation SHALL NOT publish it twice.

#### Scenario: Restart with unchanged startup inputs

- **WHEN** a successfully recovered instance restarts with its archive and key files still present and their paths still configured
- **THEN** it starts from its current state without importing the archive again

#### Scenario: Consumed archive is renamed

- **WHEN** the configured startup archive has the same bytes as a successfully consumed file but a different filename
- **THEN** startup does not apply it again

#### Scenario: Another file replaces a consumed archive

- **WHEN** the archive at a configured startup path now has a different SHA-256 and the target contains an installation
- **THEN** startup refuses a new recovery and preserves that installation and its key

#### Scenario: Normal upgrade leaves startup inputs configured

- **WHEN** the owner starts a newer release after successful recovery with the consumed archive and both paths still configured
- **THEN** startup skips that archive and follows the ordinary migration path without refusing its older source release

#### Scenario: UI import follows startup recovery

- **WHEN** a UI import has replaced a previously startup-recovered installation and that installation restarts with the earlier consumed startup archive still configured
- **THEN** the UI-imported state remains current and the earlier archive is not imported again

#### Scenario: Recovery is interrupted during publication

- **WHEN** the process stops after accepting recovery but before publication completes
- **THEN** the next start resolves the recorded operation without serving a mismatched database and key or applying it twice

### Requirement: UI recovery imports files independently of the current installation

The Databasus backups section SHALL accept an encrypted archive file and a separate source master-key file uploaded from the owner's computer. Both files SHALL be required. Import SHALL NOT require a matching history entry, configured destination, connection to the original storage or use of the target's current master key. An owner recovering into a newly initialized installation SHALL first authenticate as that installation's root admin. To recover a backup from current history, the owner SHALL download it and then upload it through this same import flow; no direct history-restore action SHALL be required.

Preparation SHALL authenticate the entire archive, validate release and schema compatibility and import into isolated staging before the active installation can be replaced. Preparation failure, cancellation or expiry SHALL preserve the active database and key and clean its private staged inputs. The owner SHALL be able to inspect preparation progress, authenticated snapshot time and source version, and a safe failure reason. Prepared imports SHALL expire after one hour without apply. There SHALL be at most one pending import preparation or application operation per instance. Applying SHALL be asynchronous and SHALL NOT depend on the initiating browser staying connected.

#### Scenario: Import into a new installation with another master key

- **WHEN** root admin uploads a valid archive and its source key to a newly initialized same-release installation with a different key, no backup history and no configured destinations
- **THEN** preparation succeeds without contacting the original storage or changing the current key
- **AND** confirmed application installs the snapshot and its supplied source key

#### Scenario: Owner supplies the wrong key or damaged file

- **WHEN** uploaded files contain the wrong key, a truncated or modified archive or an incompatible schema
- **THEN** preparation fails with a safe diagnostic and leaves the active installation usable with its original key

#### Scenario: Unconfirmed import expires

- **WHEN** a prepared import has not been applied within one hour
- **THEN** it can no longer replace the installation and its staged archive, key and imported staging state are released

### Requirement: UI replacement requires current root authorization and explicit confirmation

Upload, preparation status, cancellation and apply SHALL require the target installation's current root admin; other administrators and workspace members SHALL be refused. Applying SHALL require a recent single-use password or email identity proof bound to the current root account/session, that recovery operation and the immutable archive content. The owner SHALL explicitly confirm replacement after seeing the snapshot date, source version and warning that current users, settings and backup catalog will be replaced. Possession of the source key alone SHALL NOT authorize replacement. Invalid, expired, reused or wrong-purpose proofs SHALL NOT apply an import. If root cannot provide password or email proof, the interface SHALL explain mounted-file startup recovery into a separate fresh target; it SHALL NOT imply that mounted files can replace the current account-bearing target.

#### Scenario: Another administrator imports an archive

- **WHEN** an authenticated administrator other than root admin attempts upload, preparation inspection or application
- **THEN** the request is refused with 403 and no recovery data or capability is exposed

#### Scenario: Owner cancels replacement confirmation

- **WHEN** root admin cancels confirmation of a prepared import
- **THEN** the active installation and its key remain unchanged and no publication begins

#### Scenario: An export proof is submitted for recovery

- **WHEN** a caller supplies a master-key export proof or a proof for another prepared import
- **THEN** recovery apply is refused without replacing the active installation

### Requirement: Uploaded recovery material remains private and bounded

Recovery uploads SHALL preserve the archive's encryption and the exact master-key bytes. The key and decrypted dump SHALL NOT appear in responses, URLs, logs, telemetry, browser persistent storage or cacheable content. Staging SHALL restrict file access to the runtime identity, keep memory use bounded and stage no plaintext dump. Archive uploads SHALL enforce a configurable finite byte limit; the default SHALL be 10 GiB. Uploaded key files SHALL be limited to 64 KiB. Oversized uploads, insufficient disk space and disconnected uploads SHALL fail without replacing the active database or key and SHALL release partial private files. Private staged key copies SHALL be removed after failed, cancelled or expired preparation or completed publication, while inputs needed for interrupted accepted publication SHALL survive a restart. Recovery SHALL NOT delete the operator's original mounted input files or the installed master key as part of staging cleanup.

If the page or upload connection uses unencrypted HTTP, the interface SHALL warn and require explicit acknowledgment before transferring either file. Cancelling the warning SHALL send neither archive nor key bytes and SHALL offer guidance for a protected connection or mounted-file recovery. The server SHALL require the acknowledgment on an insecure upload as well.

#### Scenario: Owner cancels HTTP import warning

- **WHEN** root admin selects recovery files over HTTP and cancels the warning
- **THEN** neither file is uploaded and the active installation is unchanged

#### Scenario: Upload exceeds the configured limit without a declared length

- **WHEN** an archive upload without a usable declared length exceeds the configured byte limit
- **THEN** the server refuses it while streaming and releases its partial staging files without replacing active state

### Requirement: UI publication stops live work and returns a paused application

Before replacing active state, UI recovery SHALL prevent new mutations and background execution, stop and drain active requests and operations, and close active database connections. A timeout or failure to establish exclusive access SHALL refuse publication and preserve the current database/key pair. Applying SHALL publish a validated, normalized snapshot and its source key consistently; a crash SHALL NOT expose a partial or mismatched installation. The application SHALL become available again without requiring a browser request, operator restart or configured container restart policy to complete the operation.

Successful application SHALL invalidate both the replaced installation's sessions and credentials copied from the snapshot. The interface SHALL tolerate the temporary interruption, require sign-in using accounts and credentials from the snapshot, and expose the durable paused recovery result after authentication. Lost connectivity alone SHALL NOT be shown as success. Root activation and recovered-runtime restrictions below SHALL apply to both startup and UI recovery, even when the UI import uses the same host or archive as an earlier recovery.

#### Scenario: Work cannot stop before replacement

- **WHEN** an accepted UI recovery cannot drain an operation or close a connection within its allowed stop period
- **THEN** it reports failure without publishing the staged snapshot or source key
- **AND** the previous installation can serve again with its original database and key

#### Scenario: Recovery completes after the browser disconnects

- **WHEN** root admin confirms a valid import and closes the browser after apply is accepted
- **THEN** the application completes recovery and becomes available paused without an external restart
- **AND** the owner can sign in with restored credentials and inspect the durable result

#### Scenario: An apply response is lost and retried

- **WHEN** root admin repeats apply for the same already accepted recovery operation
- **THEN** it reports that operation's current state without scheduling or publishing another recovery

#### Scenario: A completed archive is deliberately imported again

- **WHEN** root admin uploads identical archive bytes as a new import and provides fresh replacement confirmation and identity proof
- **THEN** that explicit operation can restore the snapshot again despite earlier consumption of the same archive

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

After activation, the existing saved connection-test action SHALL be the only file-access exception to the unavailable state. In addition to the ordinary temporary-storage check, it SHALL verify the actual backup directory by creating a unique probe, writing non-empty content, reading and verifying it, and removing the probe without changing existing backups. Every step SHALL succeed; writable temporary storage alone SHALL NOT establish backup-directory access. A successful probe by a user authorized to edit that storage SHALL make it available only if its saved settings have not changed since the probe began. Viewers SHALL retain permission to test saved storages with the same extended probe, but their results SHALL remain diagnostic only and SHALL NOT change availability. The interface SHALL explain when an editor's validation is still required. A failed probe SHALL attempt cleanup, retain the unavailable state and show a safe failure reason. Before instance activation the recovery pause SHALL still block a writing probe. Validation SHALL NOT claim that missing historical backup files were recovered.

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

#### Scenario: Backup directory is inaccessible despite writable temporary storage

- **WHEN** an authorized editor tests a recovered local storage after activation and its temporary directory is writable but the actual backup directory is inaccessible
- **THEN** validation fails and the storage remains `REQUIRES_VALIDATION`
- **AND** its settings, secrets and existing backup files stay unchanged and ordinary storage operations remain blocked

#### Scenario: Viewer tests connectivity without validating recovery

- **WHEN** a viewer successfully tests a recovered local storage after activation
- **THEN** the test reports connected but the storage remains unavailable pending an authorized editor's validation
