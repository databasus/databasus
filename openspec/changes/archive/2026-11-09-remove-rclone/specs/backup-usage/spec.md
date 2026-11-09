## MODIFIED Requirements

### Requirement: Saved-storage details show backup usage beneath the name

Every saved storage's detail page SHALL display a localized Total usage line directly beneath its name, before Storage settings. Its value SHALL use the compact total format and visually secondary text. This SHALL apply to local, S3-compatible, NAS, Azure Blob, FTP and SFTP storages, including for users who can only view them. Usage loading and refresh SHALL NOT reset a storage name or settings draft or its connection-test state.

An initial usage failure SHALL show localized feedback with an explicit retry without claiming zero or preventing the storage details from being used. A failed background usage refresh SHALL retain the last value with a visible translated indication that it could not be refreshed and an explicit retry. A current authorization failure SHALL hide that value. A successful empty usage read SHALL show `0 MB`.

#### Scenario: A saved storage is opened

- **WHEN** a user opens a saved storage with 104 MB of eligible backups in English
- **THEN** `Total usage: 104 MB` appears directly under the name and above Storage settings

#### Scenario: A refresh during editing

- **WHEN** usage changes while a user is editing the storage's name or settings
- **THEN** its total updates without changing the draft or resetting a successful connection test

#### Scenario: Usage cannot initially load

- **WHEN** a saved storage loads successfully but its usage read fails
- **THEN** the storage details remain usable and localized usage feedback offers retry instead of a numeric value
