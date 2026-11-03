## ADDED Requirements

### Requirement: A finished physical backup that lost its storage is not published

A physical backup whose execution succeeds after its storage assignment changed SHALL NOT be recorded as completed, and its database's last backup time SHALL NOT move. Its files SHALL be handed to the deletion of stored files, and it SHALL NOT send a success or a failure notification, whatever state its record is in. A record the storage change has already removed SHALL stay removed, and a record it has already ended SHALL keep its status and fail message. A record still running SHALL end as cancelled when the storage change's stop reaches it, as for a backup the change stopped earlier.

#### Scenario: Execution finishes just before the stop reaches the backup

- **WHEN** a physical backup still recorded as running finishes its execution after the database's storage was cleared and before the stop is recorded
- **THEN** the backup is not recorded as completed, its files are requested for deletion, the last backup time does not move and no notification is sent
- **AND** once the stop is recorded the backup is listed as cancelled

#### Scenario: The storage change already removed the backup

- **WHEN** a physical backup finishes its execution after a storage switch removed its record
- **THEN** no record of the backup returns to the list, its files are requested for deletion and no notification is sent

#### Scenario: The storage change already ended the backup

- **WHEN** a physical backup finishes its execution after the storage change recorded it as cancelled
- **THEN** it stays cancelled with its fail message unchanged, its files are requested for deletion and no notification is sent
