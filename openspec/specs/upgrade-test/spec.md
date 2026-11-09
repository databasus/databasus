# Upgrade Test Specification

## Purpose

Defines the test that proves an instance of the newest release keeps its data and its backups after an upgrade to the current code.

## Requirements

### Requirement: The upgrade starts from the newest release

The upgrade test SHALL start from the image of the newest published Databasus release, chosen by version number among the repository's release tags, whether or not the checked-out branch contains that tag. The current code SHALL run as an image built from the checkout, or as the image a developer or CI names explicitly. The test output SHALL name both images.

#### Scenario: Branch behind the newest release

- **WHEN** the newest release tag is not reachable from the checked-out commit
- **THEN** the test still upgrades from that newest release

#### Scenario: CI supplies the image

- **WHEN** CI runs the test with the image it built for the commit
- **THEN** the test upgrades to that image and does not build another one

### Requirement: Data written before the upgrade is there after it

Before the upgrade, the test SHALL write through the API of the old release, the way a user does: an account, a workspace, one storage of every currently supported type with every setting filled in, one notifier of every type with every setting filled in, a PostgreSQL database with a password that uses all of those notifiers, with a schema list that names the schema of the seeded rows and table and table data exclusion lists that name only a table without rows, a MySQL, a MariaDB and a MongoDB database each with a password and an exclusion list that names only an object without rows, the PostgreSQL database's backup config with backups turned on, backup encryption turned on, the local storage, a weekly schedule on a day and at a time other than the default's, grandfather-father-son retention with every period set, retries of failed backups with a count other than the default's, and only the failure notification trigger, so that every value of the config differs from the logical default, and one completed, encrypted logical backup. It SHALL then stop the old instance and start the current code on the same data volume. Through the API of the current code, the test SHALL check that the account signs in with its password, that every item above is listed with the values written before the upgrade, apart from secrets, which the API does not return, and that the PostgreSQL database still uses every notifier. The notifier, storage and database secrets, which the API never returns, are covered by the requirements "An upgrade keeps every notifier's secrets" of the `notifiers` spec, "An upgrade keeps every storage's secrets" of the `storages` spec and "An upgrade keeps every database" of the `databases` spec instead.

#### Scenario: Upgrade keeps every item

- **WHEN** the current code starts on the volume of the old release
- **THEN** the account signs in, and the workspace, storages, notifiers, databases, backup config and backup are listed with their values
- **AND** the backup config still has encryption turned on

#### Scenario: Upgrade keeps every backup config setting

- **WHEN** the current code starts on the volume of an old release whose PostgreSQL database's backup config has backups turned on and runs weekly with grandfather-father-son retention, retries and only the failure notification trigger, each value different from the logical default
- **THEN** reading that config returns backups turned on, the same schedule, every retention period, the retry count, only the failure notification trigger, the encryption and the local storage

#### Scenario: Upgrade keeps every notifier

- **WHEN** the current code starts on the volume of an old release that has an email, Telegram, Slack, Discord, Teams, Mattermost and webhook notifier
- **THEN** each of them is listed with its type, its name and its settings that are not secrets, the webhook header names included, and the database still uses all seven

#### Scenario: Upgrade keeps every storage

- **WHEN** the current code starts on the volume of an old release that has a local, S3, NAS, Azure Blob, FTP and SFTP storage
- **THEN** each of them is listed with its type, its name and its settings that are not secrets, and the backup config still uses the local storage

#### Scenario: Upgrade keeps every logical database type

- **WHEN** the current code starts on the volume of an old release that has a PostgreSQL, a MySQL, a MariaDB and a MongoDB database
- **THEN** each of them is listed with its type, its name, its settings that are not secrets and every entry of its exclusion lists

#### Scenario: A migration loses a value

- **WHEN** a migration of the current code drops or empties a column that holds one of the written values that the API returns
- **THEN** the upgrade test fails and names the item and the value that differs

### Requirement: A backup made before the upgrade restores after it

After the upgrade, the test SHALL restore the logical backup made by the old release into a new database through the API of the current code. The restored database SHALL hold the same rows as the source had when the backup was made. Before the restore, the test SHALL test the connection of the database written before the upgrade through the API of the current code, which uses the stored database password. Together with the restore of the encrypted backup, this proves that secrets and the master key survive the upgrade.

#### Scenario: Restore after upgrade

- **WHEN** the test restores the pre-upgrade backup with the current code
- **THEN** the restore completes and the restored database holds the rows the source had at backup time

### Requirement: The upgrade test cleans up after itself

The containers, network and volume the upgrade test creates SHALL carry unique names. When the run that started the test sets a run ID, all of them, the volume included, SHALL carry that run's label. They SHALL be removed when the test ends, whether it passes or fails. When the test fails, its output SHALL include the end of the log of the instance that failed.

#### Scenario: The current image fails to start

- **WHEN** the current code exits during start-up on the old volume
- **THEN** the test fails, its output shows the end of that instance's log, and no container or volume of the test remains

### Requirement: A physical backup made before the upgrade streams after it

Before the upgrade, the test SHALL write through the API of the old release a physical PostgreSQL database with WAL streaming turned on, on the same PostgreSQL 17 source server that holds the logical source database, its physical backup config with backups turned on and the local storage, one completed full backup and one completed incremental backup built on it. It SHALL wait until the old release lists at least one archived WAL segment after the incremental backup. After the upgrade, through the API of the current code, the test SHALL check that the full backup and the incremental backup are listed with their status, that the incremental backup names the full backup as its chain's full backup and has no other incremental as its parent, and that the WAL segments listed before the upgrade are still listed. It SHALL then issue a restore token for the database and read the restore set it opens to the end. The restore set SHALL hold the full backup, the incremental backup and at least one of the WAL segments listed before the upgrade, which proves that the stored physical backup records still name readable files and that the chain survived the upgrade.

#### Scenario: Physical backups after the upgrade

- **WHEN** the current code starts on the volume of an old release that has a physical database with a completed full backup, a completed incremental backup built on it and archived WAL segments
- **THEN** both backups and the WAL segments are listed, the incremental backup belongs to the full backup's chain, a restore token is issued, and the restore set streams to its end with the full backup, the incremental backup and at least one of those WAL segments

#### Scenario: A migration loses physical backup records

- **WHEN** a migration of the current code deletes rows of a physical backup table, or empties a column the list shows, while the table stays readable
- **THEN** the upgrade test fails and names the full backup, the incremental backup or the WAL segments that are missing or differ

#### Scenario: A migration breaks a chain

- **WHEN** a migration of the current code changes the chain's full backup of the incremental backup, or gives the incremental backup an incremental parent
- **THEN** the upgrade test fails and names the incremental backup

#### Scenario: A migration breaks a physical backup table

- **WHEN** a migration of the current code drops or renames a physical backup table that the current code still reads
- **THEN** the upgrade test fails on the first request that reads the table, or because the current code fails to start

### Requirement: Verification data made before the upgrade is there after it

Before the upgrade, the test SHALL write through the API of the old release, with its account that is the instance admin and the workspace's owner, a verification agent and a verification config of the logical PostgreSQL database. The config SHALL have automatic verification turned on with the interval schedule type, a weekly schedule on a day and at a time other than the default's, and only the success notification, so every value apart from the weekly interval type differs from the default config. It SHALL then start a manual verification of the completed logical backup, which stays pending because no agent runs. After the upgrade, through the API of the current code, the test SHALL check that the agent is listed with its ID and name, that the config is returned with every value written before the upgrade, the schedule type included, and that the verification is returned with its backup, its trigger and a status that is still pending.

#### Scenario: Verification data after the upgrade

- **WHEN** the current code starts on the volume of an old release that has a verification agent, a verification config with automatic verification on a weekly interval schedule and only the success notification, and a pending manual verification
- **THEN** the agent is listed, the config is returned with the same values, and the verification is returned pending, for the same backup, with the manual trigger

#### Scenario: A migration loses verification data

- **WHEN** a migration of the current code deletes verification rows, or empties or rewrites a column the test reads, while the tables stay readable
- **THEN** the upgrade test fails and names the agent, the config or the verification that is missing or differs

#### Scenario: A migration breaks a verification table

- **WHEN** a migration of the current code drops or renames a verification table or column that the current code still reads
- **THEN** the upgrade test fails on the first request that reads it, or because the current code fails to start

### Requirement: A healthcheck config made before the upgrade is there after it

Before the upgrade, the test SHALL save through the API of the old release a healthcheck config of the logical PostgreSQL database in which every value differs from the default: checks switched off, notifications switched off, an interval, a number of failed attempts and a number of retention days other than the default's. After the upgrade, through the API of the current code, the test SHALL check that the config is returned with every value written before the upgrade.

#### Scenario: Healthcheck config after the upgrade

- **WHEN** the current code starts on the volume of an old release that has a healthcheck config with non-default values
- **THEN** the config is returned with the same values

#### Scenario: A migration loses the healthcheck config

- **WHEN** a migration of the current code resets the config to the default, or empties or rewrites a column the test reads, while the table stays readable
- **THEN** the upgrade test fails and names the config value that differs

### Requirement: A released 3.64.0 verification agent recovers through background self-update

Upgrade coverage SHALL run the actual verification agent distributed with release 3.64.0 against the current server. It SHALL establish that the old heartbeat is rejected before allowing the background self-update to finish. After the agent downloads and restarts into the current server's agent version, the server SHALL accept its heartbeat and the agent SHALL complete verification of a real backup, including reporting statistics that match the backup's known contents. The agent SHALL retain its registration, token and resource settings across the update.

This coverage SHALL remain anchored to release 3.64.0 when the newest release used by the general server upgrade test changes. The existing checks that verification records survive server upgrade SHALL complete before an agent starts consuming their pending work.

#### Scenario: Background update after rejection of the old heartbeat

- **WHEN** the released 3.64.0 agent runs against the current server and its heartbeat is rejected because it does not report memory in megabytes
- **THEN** its independent background update downloads the current agent and restarts it
- **AND** the current server accepts a heartbeat from the restarted agent using the original registration and resource settings
- **AND** a verification of a real backup completes and reports the known tables and row counts

#### Scenario: The newest server release changes

- **WHEN** the general upgrade test starts using a release newer than 3.64.0
- **THEN** the released-agent regression still starts with the actual 3.64.0 binary

### Requirement: Failed agent replacement has a tested manual recovery path

Upgrade coverage SHALL exercise a released 3.64.0 agent that can run and use its working files but cannot write to the directory holding its executable. It SHALL observe a failed background update and a rejected heartbeat while the executable remains version 3.64.0. It SHALL then stop the agent, replace its executable with the current binary using an identity permitted to do so, and restart it using the existing configuration without resupplying initial credentials or resource flags. A missing saved configuration SHALL cause restart to fail rather than recreating settings from the initial setup. After that recovery, a heartbeat SHALL be accepted and verification of a real backup SHALL complete. This scenario SHALL NOT require assignments to be issued to an agent whose heartbeat is rejected.

The repository documentation SHALL explain the permissions needed for self-update, how to recognize failure and how to perform this manual recovery. Instructions SHALL cover both the agent's own daemon and an agent managed by a service or container, preserve its credentials and settings, and direct the operator to check server contact and a completed verification after restart.

#### Scenario: The executable directory is not writable

- **WHEN** a running released 3.64.0 agent encounters a current server but cannot write its replacement executable
- **THEN** the background update reports a failure, the executable remains version 3.64.0 and its old heartbeat remains rejected
- **AND** replacing the executable and restarting with the saved configuration restores accepted heartbeats and successful backup verification

#### Scenario: An operator follows the recovery instructions

- **WHEN** an operator reads the repository recovery instructions after an agent fails to update
- **THEN** the instructions explain how to stop the applicable process or service, download the correct architecture from the connected server to a temporary file, install the replacement, and restart using the existing settings
- **AND** the instructions explain how to confirm server contact and a completed backup verification without using or exposing a new token

### Requirement: A 3.67.0 Google Drive installation has a tested refusal and recovery path

Upgrade coverage SHALL use the actual Databasus 3.67.0 image to create an account, a workspace, an unused Google Drive storage and a supported storage through its public API. It SHALL attempt startup of the current image on the same volume and observe a non-zero exit with the retirement diagnostic and both recovery options. It SHALL verify that no pending migration was recorded and that the stored application records and encrypted credentials remain unchanged.

The test SHALL then start 3.67.0 again on that volume, sign in and verify that the records remain available. It SHALL delete the unused Google Drive storage through that release's API and retry the current image. The upgrade SHALL succeed, preserve the supported records and expose no Google Drive storage option. An unused storage SHALL suffice; the regression SHALL require no Google account, real token exchange or external Drive access. This regression SHALL remain anchored to 3.67.0 when the newest release used by general upgrade coverage changes.

#### Scenario: Refusal and rollback on the original volume

- **WHEN** the current image attempts to upgrade a 3.67.0 installation containing a Google Drive storage
- **THEN** it exits non-zero before recording a pending migration and its diagnostic identifies 3.67.0 as the last supporting release
- **AND** restarting 3.67.0 on the same volume permits sign-in and lists the unchanged records

#### Scenario: Removal followed by a successful retry

- **WHEN** the regression removes the unused Google Drive storage through the restarted 3.67.0 API and retries the current image
- **THEN** startup succeeds and the supported records remain available

#### Scenario: The newest release no longer supports Google Drive

- **WHEN** the newest release used by the general upgrade test advances beyond 3.67.0
- **THEN** general upgrade coverage creates only currently supported storage types and the retirement regression still starts from 3.67.0

### Requirement: A 3.67.1 rclone installation has a tested refusal and recovery path

Upgrade coverage SHALL use the actual Databasus 3.67.1 image, with its resolved release identity pinned, to create an account, a workspace, an unused rclone storage with synthetic configuration and a supported storage through its public API. It SHALL attempt startup of the candidate on the same volume and observe a non-zero exit with rclone-only recovery guidance naming 3.67.1 and both recovery options. It SHALL compare persisted records, encrypted configuration and migration history before and after refusal and prove they are unchanged without disclosing credentials.

The regression SHALL restart 3.67.1 on that same volume, sign in, check the records, delete the unused rclone storage through its API and retry the candidate directly. Startup SHALL succeed, supported records and encrypted credentials SHALL remain available, and obsolete rclone settings schema SHALL be absent. The regression SHALL require no real cloud account, valid remote credentials or external storage access. It SHALL remain anchored to 3.67.1 when the newest release used for general upgrade coverage changes.

#### Scenario: Refusal leaves 3.67.1 usable

- **WHEN** the candidate attempts to upgrade the 3.67.1 volume containing rclone
- **THEN** it exits non-zero without advancing migrations or changing stored records
- **AND** 3.67.1 can restart, sign in and list the unchanged records

#### Scenario: Removal permits the direct retry

- **WHEN** the test removes the unused rclone storage through the restarted 3.67.1 API and starts the candidate directly
- **THEN** the upgrade succeeds, supported records and encrypted credentials survive, and the obsolete settings schema is absent

#### Scenario: The newest release advances

- **WHEN** the newest published release no longer accepts rclone
- **THEN** general upgrade coverage still seeds only supported storage types and this regression still starts from 3.67.1

### Requirement: Combined blockers have a released-image recovery regression

Upgrade coverage SHALL use the actual Databasus 3.67.0 image with pinned release identity to create an account, workspace, unused Google Drive and rclone storages with synthetic credentials, and a supported storage through its public API. Starting the candidate directly on that volume SHALL exit non-zero before any pending migration. One diagnostic SHALL name both blocking types and select 3.67.0 to remove both, or remain on 3.67.0. Persisted records, encrypted credentials and migration history SHALL remain unchanged, and output SHALL disclose none of those credentials.

The test SHALL restart 3.67.0 on the original volume, sign in, verify the records and remove both unused storages through its API. Starting the candidate directly SHALL then succeed and preserve supported records and encrypted credentials. Both obsolete providers' settings schemas SHALL be absent. The regression SHALL NOT start an intermediate 3.67.1 instance. It SHALL remain anchored to 3.67.0 independently of newest-release selection and SHALL retain the existing Google Drive-only regression as independent coverage.

#### Scenario: Direct jump from a release supporting both types

- **WHEN** the candidate starts on a 3.67.0 volume containing both blocking types
- **THEN** it reports both types together and identifies 3.67.0 as the common recovery version
- **AND** records and migration history are unchanged and 3.67.0 can reopen the volume

#### Scenario: Both types are removed on 3.67.0

- **WHEN** the regression deletes both unused storages through the restarted 3.67.0 API and retries the candidate
- **THEN** startup succeeds without running 3.67.1 and supported records and credentials survive
