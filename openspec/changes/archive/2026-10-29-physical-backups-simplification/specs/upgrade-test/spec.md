## MODIFIED Requirements

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
