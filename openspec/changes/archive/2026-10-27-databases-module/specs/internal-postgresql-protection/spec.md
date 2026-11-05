## MODIFIED Requirements

### Requirement: Direct connection tests require workspace authorization

The test of unsaved database settings SHALL require an authenticated user who can manage databases in the relevant workspace. The request SHALL name that workspace. A request that names a saved database SHALL be authorized against the workspace stored on that database before saved credentials are merged with request fields. The named database SHALL be checked first: one that does not exist or does not belong to the workspace in the request SHALL get `404` before the caller's role in that workspace is checked.

#### Scenario: Ad hoc request without a workspace

- **WHEN** an authenticated user submits unsaved database settings without a workspace
- **THEN** the response is `400` with `code` `invalid_request`
- **AND** no connection is attempted

#### Scenario: Ad hoc request for an unauthorized workspace

- **WHEN** a viewer of a workspace submits unsaved database settings for that workspace
- **THEN** the response is `403` with `code` `permission_denied`
- **AND** no connection is attempted

#### Scenario: Ad hoc request for another team's workspace

- **WHEN** an authenticated user submits unsaved database settings for a workspace they are not a member of
- **THEN** the response is `404` with `code` `not_found`
- **AND** no connection is attempted

#### Scenario: Saved database from another workspace

- **WHEN** a user names a saved database of a workspace other than the one in the request, or of a workspace they are not a member of, and supplies a different host
- **THEN** the response is `404` with `code` `not_found` before the saved password is applied
- **AND** the system makes no connection to the supplied host

#### Scenario: Viewer names a saved database of their own workspace

- **WHEN** a viewer of a workspace names a saved database of that workspace and supplies a different host
- **THEN** the response is `403` with `code` `permission_denied` before the saved password is applied
- **AND** the system makes no connection to the supplied host

#### Scenario: Authorized workspace member tests a saved database

- **WHEN** a workspace member who can manage databases tests the settings of a saved database in that workspace and leaves its password blank
- **THEN** the system uses the saved password and performs the connection test

### Requirement: Direct connection-test failures do not disclose protected records or internal errors

The test of unsaved database settings SHALL return the same `404` response for an unknown saved database and a saved database of a workspace the caller is not a member of. Failures while loading a saved target and failures while checking workspace permissions SHALL return `500` with `code` `internal` and SHALL NOT include the underlying error.

#### Scenario: Unknown saved database ID

- **WHEN** a caller names an ID that does not identify a saved database
- **THEN** the response is the same `404` body used for a saved database of another team's workspace, apart from the ID

#### Scenario: Saved target lookup fails internally

- **WHEN** the internal database fails while loading the named saved database
- **THEN** the response is `500` with `code` `internal`
- **AND** the database error is absent from the response

#### Scenario: Permission lookup fails internally

- **WHEN** the workspace permission lookup fails for an ad hoc request or one that names a saved database
- **THEN** the response is `500` with `code` `internal`
- **AND** the underlying membership error is absent from the response
