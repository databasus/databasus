## MODIFIED Requirements

### Requirement: The master encryption key is generated once

On a fresh installation, the backend SHALL generate the master encryption key once and SHALL use that one key for every secret it encrypts. The backend SHALL load or create the key while it starts, before it serves requests or runs background work. This SHALL hold when several backend processes start at the same time before the key exists. A key that an older release left in the internal database SHALL be written into the key file before the key is loaded, replacing whatever the file held, and SHALL then be removed from the database.

#### Scenario: Concurrent first use

- **WHEN** several backend processes start at the same moment on an installation that has no master key yet
- **THEN** one key file exists afterwards
- **AND** every secret encrypted by those processes decrypts with that key

#### Scenario: Key already exists

- **WHEN** the backend runs with an existing master key file and no master key left in its database
- **THEN** it uses the key from that file and does not rewrite the file

#### Scenario: Key left in the database by an older release

- **WHEN** the backend starts on an internal database that still holds a master key
- **THEN** the key file contains that key
- **AND** the database no longer holds it
- **AND** secrets encrypted with that key before the upgrade decrypt

## ADDED Requirements

### Requirement: The backend refuses to start without a usable master key

When a legacy master key in the internal database cannot be written into the key file, or when, after any legacy key has been moved into the file, the key file cannot be read or created or holds fewer than 32 bytes, the backend SHALL stop during startup with an error that names the key file. It SHALL NOT serve requests or run background work in that state. Without a legacy key it SHALL NOT replace an existing key file that it cannot use, and a legacy key that could not be written SHALL stay in the database.

#### Scenario: Key file shorter than 32 bytes

- **WHEN** the backend starts and the master key file holds fewer than 32 bytes
- **THEN** the backend stops with an error that names the key file
- **AND** the key file is unchanged

#### Scenario: Key file cannot be read

- **WHEN** the backend starts and the master key file exists but cannot be read
- **THEN** the backend stops with an error that names the key file
- **AND** no new key file replaces it

#### Scenario: Key file cannot be created

- **WHEN** the backend starts without a master key file, and the directory of the key file does not let the backend create a file
- **THEN** the backend stops with an error that names the key file
- **AND** it serves no request and starts no background work

#### Scenario: Legacy key cannot be written into the key file

- **WHEN** the backend starts on an internal database that still holds a master key, and the key file cannot be written
- **THEN** the backend stops with an error that names the key file
- **AND** it serves no request
- **AND** the database still holds the key
