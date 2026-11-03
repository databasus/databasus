## ADDED Requirements

### Requirement: One failed removal does not stop retention for the database

When retention cannot remove one expired logical backup, it SHALL report that failure in the log and SHALL still remove the database's other expired backups in the same check. The backup that could not be removed SHALL stay listed and SHALL be tried again on a later check.

#### Scenario: One of several expired backups cannot be removed

- **WHEN** a database has three expired backups and removing one of them fails
- **THEN** the other two are removed in that check and the third stays listed
- **AND** the log names the backup that could not be removed

#### Scenario: The failure has passed

- **WHEN** the removal that failed can succeed on the next check
- **THEN** that check removes the remaining expired backup
