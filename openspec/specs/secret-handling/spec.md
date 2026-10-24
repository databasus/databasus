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

On a fresh installation, the backend SHALL generate the master encryption key once and SHALL use that one key for every secret it encrypts. This SHALL hold when several operations need the key at the same time before it exists.

#### Scenario: Concurrent first use

- **WHEN** several operations that encrypt secrets start at the same moment on an installation that has no master key yet
- **THEN** one key file exists afterwards
- **AND** every secret encrypted by those operations decrypts with that key

#### Scenario: Key already exists

- **WHEN** the backend runs with an existing master key file and no master key left in its database
- **THEN** it uses the key from that file and does not rewrite the file

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
