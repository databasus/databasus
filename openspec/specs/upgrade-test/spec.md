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

Before the upgrade, the test SHALL write through the API of the old release, the way a user does: an account, a workspace, a local storage, a webhook notifier, a PostgreSQL database with a password, its backup config with backup encryption turned on, and one completed, encrypted logical backup. It SHALL then stop the old instance and start the current code on the same data volume. Through the API of the current code, the test SHALL check that the account signs in with its password and that every item above is listed with the values written before the upgrade.

#### Scenario: Upgrade keeps every item

- **WHEN** the current code starts on the volume of the old release
- **THEN** the account signs in, and the workspace, storage, notifier, database, backup config and backup are listed with their values
- **AND** the backup config still has encryption turned on

#### Scenario: A migration loses a value

- **WHEN** a migration of the current code drops or empties a column that holds one of the written values
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
