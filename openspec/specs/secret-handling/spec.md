# secret-handling Specification

## Purpose
Defines how the backend treats the secrets it stores and uses: tokens stay out of the process output, the master encryption key exists once, and a secret that cannot be decrypted makes its operation fail visibly.

## Requirements

### Requirement: OAuth tokens never reach the process output

When the backend uses or refreshes an OAuth token of a storage, it SHALL NOT write the token or any part of it to standard output, standard error or the log. Events about the refresh SHALL go through the application log, which masks sensitive data.

#### Scenario: Google Drive token refresh

- **WHEN** a Google Drive storage operation fails with an authorization error and the backend refreshes the token
- **THEN** standard output and standard error contain no part of the old or the new access token or refresh token
- **AND** the application log records that the token was refreshed

#### Scenario: Refresh fails

- **WHEN** the token refresh of a Google Drive storage fails
- **THEN** the operation fails with an error that says the refresh failed
- **AND** no part of a token appears in the process output

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

### Requirement: A notifier secret that cannot be decrypted fails the notification visibly

When a webhook notifier sends a notification and its URL, its body template or one of its header values cannot be decrypted, the notification SHALL fail with an error that says decryption failed. The backend SHALL NOT send a request built from the encrypted value. The notifier SHALL still appear in the notifier list, so the user can edit it.

#### Scenario: URL cannot be decrypted

- **WHEN** a webhook notifier's stored URL cannot be decrypted with the current master key, and a notification is sent through it
- **THEN** the notification fails with an error that says the URL could not be decrypted
- **AND** no HTTP request is made

#### Scenario: Body template cannot be decrypted

- **WHEN** a webhook notifier's stored body template cannot be decrypted with the current master key, and a notification is sent through it
- **THEN** the notification fails with an error that says the body template could not be decrypted
- **AND** no HTTP request is made

#### Scenario: Header value cannot be decrypted

- **WHEN** one header value of a webhook notifier cannot be decrypted, and a notification is sent through it
- **THEN** the notification fails with an error that says a header could not be decrypted
- **AND** no HTTP request is made

#### Scenario: Notifier list with an undecryptable notifier

- **WHEN** a workspace has a webhook notifier whose secrets cannot be decrypted, and a user opens the notifier list
- **THEN** the list loads and includes that notifier
