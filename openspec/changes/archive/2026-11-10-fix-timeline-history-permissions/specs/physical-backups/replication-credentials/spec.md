## ADDED Requirements

### Requirement: Timeline history requires no additional source privileges

Databasus SHALL obtain timeline history with the replication credentials it provisions, without granting administrative membership, server-file read access, broad table access or execution of SQL server-file reading functions. Obtaining history SHALL NOT change any source role's privileges.

#### Scenario: History after promotion with provisioned credentials

- **WHEN** credentials created by Databasus are used for physical backups and the source standby is promoted
- **THEN** Databasus obtains the new timeline's history using those credentials
- **AND** the role still cannot read server files through SQL, read application tables, create roles or databases, or act as an administrator
- **AND** no other role gains privileges

#### Scenario: History request fails

- **WHEN** obtaining timeline history fails
- **THEN** Databasus leaves the source privileges unchanged
- **AND** it does not propose server-file read privileges as a required workaround
