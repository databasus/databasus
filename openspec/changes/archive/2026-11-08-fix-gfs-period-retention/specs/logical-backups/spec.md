## MODIFIED Requirements

### Requirement: Retention removes the backups the policy no longer keeps

Databasus SHALL remove, on a regular check, the logical backups of each database that its config's retention no longer keeps, and SHALL request the deletion of their files and metadata files. A backup that the policy drops but that is still running, or that a running restore reads, SHALL stay until a check after that work has ended. A backup younger than one hour SHALL always be kept. The time period policy SHALL remove every backup older than the period. The count policy SHALL keep the newest completed backups up to the count. The grandfather-father-son policy SHALL select completed logical backups by the independent calendar rules in `backup-retention/gfs`, including completed weekly, monthly and yearly periods and protected current candidates. The count and grandfather-father-son policies SHALL keep failed and cancelled backups.

#### Scenario: Keeping the last five

- **WHEN** a database has seven completed backups older than one hour, and its config keeps the last five
- **THEN** after the next check its two oldest backups are no longer listed, and the deletion of their files is requested

#### Scenario: A restore reads an expired backup

- **WHEN** a backup is older than its config's time period, and a restore of it is running at the next check
- **THEN** the backup is still listed after that check
- **AND** after the first check once the restore has ended, it is no longer listed

#### Scenario: Two daily periods preserve older GFS points through repeated cleanup

- **WHEN** a database completes one logical backup daily at 12:00 UTC from October 12 through November 5, 2026, its GFS settings are two daily periods, one completed week and one completed month with other levels disabled, and cleanup runs two hours after each completion
- **THEN** after November 5 cleanup, listing its backups returns only November 5, November 4, November 1 and October 31
- **AND** the retained backups' files and metadata remain available

#### Scenario: Failed and cancelled backups do not change GFS candidates

- **WHEN** a failed or cancelled logical backup is newer than the successful backup that currently serves as a GFS candidate
- **THEN** cleanup keeps the failed or cancelled backup and the successful candidate
- **AND** the failed or cancelled backup does not advance the GFS reference instant
