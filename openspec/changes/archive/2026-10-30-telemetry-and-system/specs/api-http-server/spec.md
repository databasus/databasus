## MODIFIED Requirements

### Requirement: A contract route without an operation answers 500

A route mounted among the API routes, whose method and path neither API contract describes, SHALL answer every request with `500` and code `internal` and SHALL NOT run its handler. The failure SHALL be logged with the method and path. No API controller SHALL bypass the contract and its access checks through a separate unguarded route group. An unknown API path with no mounted route SHALL keep answering 404 as specified separately.

#### Scenario: An operation is mounted but missing from the contract

- **WHEN** an API route is mounted for a method and path neither contract describes, and a client calls it
- **THEN** the response is 500 with code `internal`, the handler does not run, and the log names the method and path

### Requirement: The version operation keeps its path and body

`GET /api/v1/system/version` SHALL answer 200 with the content type `application/json` and the body `{"version": "<version>"}`, where the version is the one the running Databasus reports. It SHALL require no session or token. Its path and the `version` field SHALL NOT change in any release, because installed verification agents update themselves through it. Answering this operation SHALL NOT depend on disk inspection, internal-database connectivity, client tool checks, verification-agent heartbeats or periodic task health.

#### Scenario: An agent asks for the version without a token

- **WHEN** a client without a session calls `GET /api/v1/system/version`
- **THEN** the response is 200 and its JSON body has the field `version` with the running version

#### Scenario: Deep health fails while the server serves requests

- **WHEN** a dependency health check fails but the HTTP server can serve requests
- **THEN** the version operation still returns 200 with the running version

## ADDED Requirements

### Requirement: Installed agents can download the current binary without credentials

`GET /api/v1/system/verification-agent` SHALL require no token or session and SHALL retain that path and the `arch` query parameter across releases. `arch` SHALL be required and SHALL accept only `amd64` or `arm64`. An available binary SHALL be streamed as `application/octet-stream` with `Content-Disposition: attachment; filename=databasus-verification-agent` and `X-Databasus-Version` containing the running version. A missing or unsupported architecture SHALL get 400 with code `invalid_request`; an unavailable binary SHALL get 404 with code `not_found`. Downloading a binary SHALL NOT depend on a successful heartbeat.

#### Scenario: An installed agent downloads its architecture

- **WHEN** a client without credentials requests a binary for a supported architecture that is available
- **THEN** the response is 200 with the requested architecture's binary and the declared download and version headers

#### Scenario: The architecture is missing or invalid

- **WHEN** the query omits `arch` or supplies an architecture other than the two supported values
- **THEN** the response is 400 with code `invalid_request` and no binary body

#### Scenario: The binary is unavailable

- **WHEN** a supported architecture's binary is unavailable
- **THEN** the response is 404 with code `not_found`

### Requirement: Signed-in users can read the instance data-volume usage

`GET /api/v1/system/disk-usages` SHALL return 200 with total, used and free space in bytes for the volume holding the instance data. It SHALL require a valid user session and SHALL be available to ordinary signed-in users as well as admins. Without a valid session it SHALL answer 401 with code `unauthenticated`. An inspection failure SHALL return a 500 `Problem` with code `internal` and no internal details. The interface's existing disk summary SHALL use these values. The removed `/api/v1/disk/usage` route SHALL return the standard unknown-API-path 404 response.

#### Scenario: An ordinary user reads disk usage

- **WHEN** an ordinary signed-in user requests data-volume usage
- **THEN** the response is 200 with `totalSpaceBytes`, `usedSpaceBytes` and `freeSpaceBytes`
- **AND** the interface displays its disk summary using those values

#### Scenario: A client has no session

- **WHEN** a client without a valid session requests data-volume usage
- **THEN** the response is 401 with code `unauthenticated`

#### Scenario: The old route is called

- **WHEN** a client requests `/api/v1/disk/usage`
- **THEN** the response is 404 with code `not_found`
