# databasus-recovery Specification

## Purpose
Restores Databasus from an encrypted backup file and its separately retained master key through mounted startup files or confirmed UI file import, without upgrading the snapshot or starting competing work.

## Requirements

### Requirement: Recovery requires the exact source release

Both startup recovery and UI file import SHALL accept a Databasus backup file only when its authenticated source release exactly matches the running Databasus release, including the patch version, and its schema identity is compatible with the running build under the rules below. Both older and newer releases SHALL be refused before importing data, applying application migrations or replacing the destination key. The refusal SHALL name the source and running releases and instruct the owner to start the source release, recover there, and then upgrade normally. There SHALL be no override accepting a different release. Successfully consumed startup inputs SHALL be skipped on later normal starts as specified below; leaving them configured SHALL NOT turn an ordinary upgrade into a new recovery attempt.

Schema identity SHALL contain a digest of the ordered migration bundle and the effective applied migration versions captured in the snapshot. Compatibility SHALL require an identical bundle digest, every migration carried by the target recorded as applied in the snapshot, and no applied version newer than the target's newest migration. Additional historical applied versions absent from the current bundle SHALL be accepted when they are no newer than that newest version, as ordinary upgraded installations permit. Those historical entries SHALL be preserved. Recovery SHALL compare the imported migration history with the authenticated snapshot identity before publication; it SHALL NOT require the entire applied-version set to equal the current bundle's version set.

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
- **AND** successful recovery replaces its automatically generated key with the exact source key and opens the recovered instance with its work running

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

### Requirement: Startup recovery explains every skip or refusal in the log

When configured startup recovery is skipped or refused, the application SHALL log a clear operator-facing message stating that recovery did not run, the specific reason, the effect on the current installation and the next step. It SHALL distinguish normal skipping of an already consumed archive from a recovery refusal. Refusals SHALL identify the applicable cause, including existing accounts or other user state, changed persisted settings, unknown target state, incomplete or unreadable inputs, incompatible release/schema, invalid archive/key or inability to obtain exclusive access. The message SHALL preserve the existing refusal behavior and SHALL NOT imply successful recovery. It SHALL NOT expose keys, credentials, request bodies or user records. An ordinary start without configured recovery inputs SHALL NOT be reported as a recovery failure.

#### Scenario: Existing accounts prevent startup recovery

- **WHEN** configured startup recovery is refused because an account exists, including a sole root account
- **THEN** an error log explains that existing accounts prevent mounted-file recovery, that the current installation is unchanged and that the owner can use root-confirmed UI import or a separate fresh target

#### Scenario: Changed settings or other user data prevent startup recovery

- **WHEN** configured startup recovery is refused because persisted settings were changed or other user data exists
- **THEN** an error log identifies that category of state, confirms that the current installation is unchanged and directs the owner to confirmed UI import or a separate fresh target

#### Scenario: Configured inputs cannot be used

- **WHEN** startup recovery cannot proceed because an input is missing or unreadable, the key or archive is invalid, compatibility fails or eligibility/exclusive access cannot be established
- **THEN** an error log explains the specific cause and corrective action without exposing secret values or claiming recovery succeeded

#### Scenario: An already consumed archive is skipped

- **WHEN** startup skips an archive already applied successfully
- **THEN** an informational log explains that it was already recovered, the current installation starts normally and no action is needed

### Requirement: Completed startup recovery is deduplicated by archive content

Startup recovery SHALL identify an archive by SHA-256 of the complete encrypted file. Successful consumption SHALL survive replacement of snapshot state and SHALL be recorded only after the database, exact source key and the marker of the recovery that produced them are durably published. Restarting with the same successfully consumed archive SHALL start the current installation without importing it again, even when the file is renamed, an explicit UI import has since replaced the snapshot or the application is being upgraded normally. A different archive at the same configured path SHALL NOT be treated as consumed and SHALL be refused on a target containing user accounts, user data or user-changed settings. A failed attempt SHALL NOT be marked successfully consumed. An interrupted accepted recovery SHALL resume or finish its recorded publication safely rather than begin another import. A deliberate new UI import of identical bytes SHALL remain possible; retries of one accepted operation SHALL NOT publish it twice.

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

### Requirement: UI recovery imports an archive and text key independently of the current installation

The Databasus backups section SHALL accept an encrypted archive file selected from the owner's computer and the source master key pasted or typed as text. Both SHALL be required; UI import SHALL NOT require a master-key file. Import SHALL NOT require a matching history entry, configured destination, connection to the original storage or use of the target's current master key. An owner recovering into a newly initialized installation SHALL first authenticate as that installation's root admin. To recover a backup from current history, the owner SHALL download it and then upload it through this same import flow; no direct history-restore action SHALL be required.

Preparation SHALL authenticate the entire archive, validate release and schema compatibility and import into isolated staging before the active installation can be replaced. Preparation failure, cancellation or expiry SHALL preserve the active database and key and clean its private staged inputs. The owner SHALL be able to inspect preparation progress, authenticated snapshot time and source version, and a safe failure reason. Prepared imports SHALL expire after one hour without apply. There SHALL be at most one pending import preparation or application operation per instance. Applying SHALL be asynchronous and SHALL NOT depend on the initiating browser staying connected.

#### Scenario: Import into a new installation with another master key

- **WHEN** root admin uploads a valid archive and its source key to a newly initialized same-release installation with a different key, no backup history and no configured destinations
- **THEN** preparation succeeds without contacting the original storage or changing the current key
- **AND** confirmed application installs the snapshot and its supplied source key

#### Scenario: Owner supplies the wrong key or damaged file

- **WHEN** the supplied text key is wrong or the uploaded archive is truncated, modified or has an incompatible schema
- **THEN** preparation fails with a safe diagnostic and leaves the active installation usable with its original key

#### Scenario: Unconfirmed import expires

- **WHEN** a prepared import has not been applied within one hour
- **THEN** it can no longer replace the installation and its staged archive, key and imported staging state are released

### Requirement: UI replacement requires current root authorization and explicit confirmation

Upload, preparation status, cancellation and apply SHALL require the target installation's current root admin; other administrators and workspace members SHALL be refused. The owner SHALL explicitly confirm replacement after seeing the snapshot date, source version and warning that current users, settings and backup catalog will be replaced. While the instance requires two-factor sign-in, by either method, applying SHALL also require a single-use confirmation by root admin's second factor; while it does not, the session and the replacement confirmation SHALL be enough. When root admin has a connected authenticator app, the confirmation SHALL be a current app code or an unused recovery code, checked under the rules of the `authenticator-app` spec; no mail SHALL be sent, and requesting a confirmation SHALL tell the client that the app code is expected. Otherwise the confirmation SHALL be a single-use code emailed to root admin and bound to the current root account/session, that recovery operation and the immutable archive content. The account password SHALL NOT be accepted as a confirmation. A code SHALL be emailed only while the instance requires two-factor sign-in, root admin has no connected app, a system mail server is configured and root admin's address is usable; while the instance does not require two-factor sign-in, or the mail server or the address is missing for a root admin without an app, a code request SHALL be refused with 409 `identity_proof_unavailable`. Possession of the source key alone SHALL NOT authorize replacement. While a code is required, a missing, invalid, expired, reused or wrong-purpose code SHALL NOT apply an import. If the emailed code cannot be sent, the interface SHALL show the sending error; the owner can still recover from mounted files into a separate fresh target, and the interface SHALL NOT imply that mounted files can replace the current account-bearing target.

#### Scenario: Another administrator imports an archive

- **WHEN** an authenticated administrator other than root admin attempts upload, preparation inspection or application
- **THEN** the request is refused with 403 and no recovery data or capability is exposed

#### Scenario: Owner cancels replacement confirmation

- **WHEN** root admin cancels confirmation of a prepared import
- **THEN** the active installation and its key remain unchanged and no publication begins

#### Scenario: An export proof is submitted for recovery

- **WHEN** a caller supplies a master-key export proof or a proof for another prepared import
- **THEN** recovery apply is refused without replacing the active installation

#### Scenario: Two-factor sign-in is off

- **WHEN** the instance does not require two-factor sign-in and root admin confirms replacement of a prepared import
- **THEN** apply is accepted without a code or a password
- **AND** a request for an emailed code is refused with 409 `identity_proof_unavailable` without sending mail

#### Scenario: Two-factor sign-in is on

- **WHEN** the instance requires two-factor sign-in and root admin confirms replacement without a code or with a wrong one
- **THEN** apply is refused with 400 `identity_proof_invalid` and the prepared recovery stays ready

#### Scenario: Two-factor sign-in is on and root admin has an app

- **WHEN** the instance requires two-factor sign-in, root admin has a connected app, and root admin confirms replacement with a current app code
- **THEN** apply is accepted without sending mail, even when the instance has no mail server

#### Scenario: Requesting a confirmation when root admin has an app

- **WHEN** the instance requires two-factor sign-in, with or without a mail server, root admin has a connected app, and root admin requests a confirmation for a prepared recovery
- **THEN** the answer names the app method and carries no expiry time, no mail is sent, and the request is not refused with 409 `identity_proof_unavailable`

#### Scenario: Confirming replacement with a recovery code

- **WHEN** the instance requires the authenticator app method and has no mail server, and root admin with a connected app confirms replacement with an unused recovery code
- **THEN** apply is accepted and that recovery code stops working

#### Scenario: An app code already spent

- **WHEN** root admin confirms replacement with an app code that was already accepted in the same 30-second period, for example for a master-key export
- **THEN** apply is refused with 400 `identity_proof_invalid` and the prepared recovery stays ready

### Requirement: UI publication stops live work and returns a running application

Before replacing active state, UI recovery SHALL prevent new mutations and background execution, stop and drain active requests and operations, and close active database connections. A timeout or failure to establish exclusive access SHALL refuse publication and preserve the current database/key pair. Applying SHALL publish a validated, normalized snapshot and its source key consistently; a crash SHALL NOT expose a partial or mismatched installation. The application SHALL become available again without requiring a browser request, operator restart or configured container restart policy to complete the operation.

Successful application SHALL invalidate both the replaced installation's sessions and credentials copied from the snapshot. The interface SHALL tolerate the temporary interruption, require sign-in using accounts and credentials from the snapshot, and expose the completed recovery result after authentication. Lost connectivity alone SHALL NOT be shown as success. The published installation SHALL run its schedules and accept its actions at once, without an activation step. The recovered-runtime rules below SHALL apply to both startup and UI recovery, even when the UI import uses the same host or archive as an earlier recovery.

#### Scenario: Work cannot stop before replacement

- **WHEN** an accepted UI recovery cannot drain an operation or close a connection within its allowed stop period
- **THEN** it reports failure without publishing the staged snapshot or source key
- **AND** the previous installation can serve again with its original database and key

#### Scenario: Recovery completes after the browser disconnects

- **WHEN** root admin confirms a valid import and closes the browser after apply is accepted
- **THEN** the application completes recovery and becomes available with its work running, without an external restart
- **AND** the owner can sign in with restored credentials and inspect the durable result

#### Scenario: An apply response is lost and retried

- **WHEN** root admin repeats apply for the same already accepted recovery operation
- **THEN** it reports that operation's current state without scheduling or publishing another recovery

#### Scenario: A completed archive is deliberately imported again

- **WHEN** root admin uploads identical archive bytes as a new import and provides fresh replacement confirmation and identity proof
- **THEN** that explicit operation can restore the snapshot again despite earlier consumption of the same archive

### Requirement: Recovered runtime state does not claim work survived the lost server

Recovery SHALL end operations that were in progress at the snapshot as interrupted, invalidate sessions, temporary action/download credentials and verification-agent credentials, discard obsolete runtime claims and local work, and preserve the original schedules, which run from the first start of the recovered installation. It SHALL discard source-instance unfinished-write and deletion obligations rather than replay potentially stale cleanup against surviving storages, and SHALL inform root admin in the Databasus backups section that leftover files can require manual cleanup when it dropped any. Recovered storages, local ones included, SHALL be used as they were at the snapshot from the first start, without a connection test or any other validation; root admin is responsible for a local storage's folder existing on the new host. Physical backup creation SHALL start a new full chain rather than continuing a chain whose local WAL queue was lost. Previously completed backup history SHALL remain available, subject to its actual remote files and coverage at the snapshot; files created after the snapshot SHALL NOT be claimed as recovered history.

#### Scenario: Physical backup configuration survives recovery

- **WHEN** a physical backup of the recovered instance becomes due
- **THEN** it requests a new full backup and does not rely on the lost local WAL queue

#### Scenario: A remote logical backup existed at the snapshot

- **WHEN** the owner of the recovered instance restores a cataloged logical backup whose remote files still exist
- **THEN** its recovered credentials and encryption key permit restoration

#### Scenario: A completed physical restore set existed at the snapshot

- **WHEN** the owner downloads and restores a cataloged physical restore set after the recovery
- **THEN** its retained full backup, increments, WAL and timeline history support the recovery target that was available at the snapshot

#### Scenario: Recovered installation starts its work at once

- **WHEN** a recovery is published and its database has an enabled backup schedule, a healthcheck and a notifier
- **THEN** the schedule, the healthcheck and notifications run without any confirmation from root admin
- **AND** no operation answers that the instance is paused

#### Scenario: Recovered local storage is used at once

- **WHEN** a recovery is published and a Databasus backup or a database backup becomes due for a local storage whose folder exists on the new host
- **THEN** the backup writes its file to that storage without anyone testing the storage first
- **AND** no operation answers that the storage is unavailable because of the recovery

### Requirement: Recovery uploads remain private and bounded over any connection

Recovery uploads SHALL preserve the archive's encryption and the exact UTF-8 bytes of the entered master key without trimming or normalizing whitespace or line endings. The key and decrypted dump SHALL NOT appear in responses, URLs, logs, telemetry, browser persistent storage or cacheable content. Staging SHALL restrict file access to the runtime identity, keep memory use bounded and stage no plaintext dump. Archive uploads SHALL enforce a configurable finite byte limit; the default SHALL be 10 GiB. The UTF-8 text-key field SHALL be limited to 64 KiB and SHALL be write-only. Oversized uploads, insufficient disk space and disconnected uploads SHALL fail without replacing the active database or key and SHALL release partial private files. Private staged key copies SHALL be removed after failed, cancelled or expired preparation or completed publication, while inputs needed for interrupted accepted publication SHALL survive a restart. Recovery SHALL NOT delete the operator's original mounted input files or the installed master key as part of staging cleanup.

The interface SHALL upload the archive and the text key over the connection the page uses, HTTP included, without a separate warning or acknowledgment, and the server SHALL accept such an upload.

#### Scenario: Upload exceeds the configured limit without a declared length

- **WHEN** an archive upload without a usable declared length exceeds the configured byte limit
- **THEN** the server refuses it while streaming and releases its partial staging files without replacing active state

#### Scenario: Owner imports over plain HTTP

- **WHEN** root admin selects a recovery archive and enters the key on a page served over HTTP
- **THEN** the upload starts without a warning or an acknowledgment and the server accepts it
