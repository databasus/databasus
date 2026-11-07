# api-http-server Specification

## Purpose
Defines how the backend's HTTP server answers a path it does not know and which connection limits protect it, without cutting off long backup downloads.

## Requirements

### Requirement: An unknown API path returns 404

A request to a path under `/api/` that matches no API route SHALL get the status `404` with the content type `application/problem+json` and a `Problem` body whose `code` is `not_found`. It SHALL NOT get the frontend page. A request to any other unknown path SHALL keep getting the frontend page, so the frontend's own routes keep working on a page reload.

#### Scenario: Mistyped API path

- **WHEN** a client sends `GET /api/v1/no-such-route`
- **THEN** the response status is `404`
- **AND** the content type is `application/problem+json` and the body's `code` is `not_found`

#### Scenario: Frontend route on reload

- **WHEN** a browser requests the frontend route `/auth/callback`
- **THEN** the response is the frontend page with the status `200`

### Requirement: Slow and idle connections are closed

The HTTP server SHALL close a connection whose client has not finished sending the request headers within 10 seconds. It SHALL close a keep-alive connection that stays idle for more than 120 seconds. These limits are a security measure: a client that opens many connections and sends headers slowly SHALL NOT hold server resources indefinitely.

#### Scenario: Headers sent too slowly

- **WHEN** a client opens a connection and does not finish its request headers within 10 seconds
- **THEN** the server closes the connection

#### Scenario: Idle keep-alive connection

- **WHEN** a keep-alive connection carries no request for more than 120 seconds
- **THEN** the server closes the connection

### Requirement: Long responses are not cut by a server deadline

The HTTP server SHALL NOT end a response because a fixed time has passed since the request arrived. A backup download or any other long response SHALL run until it completes, the client disconnects or the application shuts down.

#### Scenario: Large backup download

- **WHEN** a user downloads a backup whose transfer takes longer than 130 seconds
- **THEN** the download completes

### Requirement: The application serves no API documentation interface

The application SHALL NOT serve an API documentation interface (Swagger UI) or the API description that interface was built from, in any mode: neither with `ENV_MODE=production` nor with `ENV_MODE=development`. Requests to the former paths of the interface and of the description SHALL get the same `404` `Problem` response with `code` `not_found` as any unknown path under `/api/`. The description was built from code comments the compiler does not check, and it described neither errors nor authentication. A published map of every operation also makes weak spots easier to find and is of no use to the user of an installed instance.

#### Scenario: Documentation requested on an installed instance

- **WHEN** a client requests `GET /api/v1/docs/swagger/index.html` from the application with `ENV_MODE=production`
- **THEN** the response status is `404`, the content type is `application/problem+json` and the body's `code` is `not_found`

#### Scenario: API description requested on an installed instance

- **WHEN** a client requests `GET /api/v1/docs/swagger/doc.json` from the application with `ENV_MODE=production`
- **THEN** the response status is `404`, the content type is `application/problem+json` and the body's `code` is `not_found`

#### Scenario: Documentation requested during local development

- **WHEN** a developer opens `/api/v1/docs/swagger/index.html` on the application with `ENV_MODE=development`
- **THEN** the response status is `404`, the content type is `application/problem+json` and the body's `code` is `not_found`

### Requirement: Contract operations answer errors as Problem

Every error response of an operation described in the API contract SHALL have the content type `application/problem+json` and a body with the fields `type`, `title`, `status`, `detail` and a `code` that the contract's list of codes contains. The `status` field SHALL equal the HTTP status. A request the operation cannot decode SHALL get `400` with `code` `invalid_request`.

#### Scenario: A request breaks the contract

- **WHEN** a client sends a request a contract operation cannot decode, such as a malformed parameter
- **THEN** the response is `400` with the content type `application/problem+json` and `code` `invalid_request`

### Requirement: An internal error reveals no internal details

When a contract operation fails for a reason the client did not cause, the response SHALL be `500` with `code` `internal` and a fixed `detail` that is the same for every such failure. The text of the internal error, stack traces, SQL, file paths and secrets SHALL NOT appear in the response. The internal error SHALL be logged with the request's `request_id`, so an operator can find it from the response's `X-Request-Id`.

#### Scenario: A handler fails with a database error

- **WHEN** a contract operation fails because the database returned an error that names a table and a value
- **THEN** the client gets `500` with `code` `internal` and a `detail` that names neither
- **AND** the log holds the error under the same `request_id` as the response's `X-Request-Id`

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

### Requirement: Contract operations enforce their access mark

Every operation in the user API contract SHALL be checked against its access mark before its handler runs. An operation marked public SHALL run without a session. An operation without a mark SHALL require a valid session, and a request without one SHALL get `401` with `code` `unauthenticated`. An operation marked admin SHALL also require the instance admin role, and a signed-in user without it SHALL get `403` with `code` `permission_denied`. A session is invalid when its token cannot be verified, when its user no longer exists or is not active, or when the user's password changed after the token was issued. Operations of the agent API contract carry no access mark and accept no session: they authenticate the verification agent by its token, as the `backup-verification` capability requires.

#### Scenario: Signed-in operation without a session

- **WHEN** a client calls a contract operation without an access mark and sends no token
- **THEN** the response is `401` with `code` `unauthenticated`, and the operation does not run

#### Scenario: Token of a deactivated user

- **WHEN** a client calls a contract operation without an access mark with a token of a user who has since been deactivated
- **THEN** the response is `401` with `code` `unauthenticated`

#### Scenario: Token issued before a password change

- **WHEN** a client calls a contract operation without an access mark with a token issued before the user's password was changed
- **THEN** the response is `401` with `code` `unauthenticated`

#### Scenario: A user who is not an instance admin calls an admin operation

- **WHEN** a signed-in user who is not an instance admin calls a contract operation marked admin
- **THEN** the response is `403` with `code` `permission_denied`, and the operation does not run

#### Scenario: Public operation without a session

- **WHEN** a client calls a contract operation marked public without a token
- **THEN** the operation runs

#### Scenario: An agent's token on a user operation

- **WHEN** a client calls a user API operation without an access mark with a verification agent's token
- **THEN** the response is `401` with `code` `unauthenticated`, and the operation does not run

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

### Requirement: Binary downloads bypass HTTP gzip

Databasus SHALL NOT apply HTTP gzip to responses of the logical backup file download, physical restore-set download or verification-agent binary download operations, including when a client sends `Accept-Encoding: gzip`. Successful downloads SHALL have no `Content-Encoding` header and SHALL deliver the same file or archive bytes as a request with `Accept-Encoding: identity`. Backup files and restore-set artifacts SHALL retain their existing compression formats after decryption. A successful verification-agent binary download SHALL retain its exact `Content-Length` and declared download and version headers. A physical restore set SHALL stream without requiring its complete length in advance.

#### Scenario: Logical backup requested with gzip accepted

- **WHEN** a client with a valid download token requests a logical backup file with `Accept-Encoding: gzip`
- **THEN** the response is `200` without `Content-Encoding`
- **AND** its body is the decrypted backup file in its existing compression format

#### Scenario: Physical restore set requested with gzip accepted

- **WHEN** a client with a valid restore token requests a physical restore set with `Accept-Encoding: gzip`
- **THEN** the response is `200` without `Content-Encoding`
- **AND** its body is the restore-set tar archive with the expected artifacts and integrity manifest

#### Scenario: Verification-agent binary requested with gzip accepted

- **WHEN** a client requests an available verification-agent binary for a supported architecture with `Accept-Encoding: gzip`
- **THEN** the response is `200` without `Content-Encoding`
- **AND** its body is the binary for that architecture, its `Content-Length` equals the delivered byte count, and its declared download and version headers are present

### Requirement: Other responses retain compression negotiation

JSON responses outside the three excluded download operations and eligible text static assets SHALL remain gzip-compressed when the client sends `Accept-Encoding: gzip`. Such a response SHALL declare `Content-Encoding: gzip` and `Vary: Accept-Encoding`, and decompress to the original response content. A request with `Accept-Encoding: identity` SHALL receive a response without HTTP gzip. The existing static-asset exclusions for `.png`, `.gif`, `.jpeg`, `.jpg`, `.ico`, `.svg`, `.pdf` and `.mp4` SHALL remain excluded even when the client accepts gzip. Excluding the download operations SHALL NOT exclude neighboring JSON operations.

#### Scenario: JSON requested with gzip accepted

- **WHEN** a client requests a JSON response outside the excluded download operations with `Accept-Encoding: gzip`
- **THEN** the response declares gzip encoding and varies by `Accept-Encoding`
- **AND** decompressing the body yields the original JSON content

#### Scenario: Eligible static asset requested with gzip accepted

- **WHEN** a client requests an eligible JavaScript or CSS static asset with `Accept-Encoding: gzip`
- **THEN** the response declares gzip encoding and decompresses to the original asset

#### Scenario: Client requests identity encoding

- **WHEN** a client requests a JSON response or text static asset with `Accept-Encoding: identity`
- **THEN** the response has no `Content-Encoding` header and contains the original content

#### Scenario: Excluded static asset requested with gzip accepted

- **WHEN** a client requests a static asset with one of the excluded extensions and sends `Accept-Encoding: gzip`
- **THEN** the response has no `Content-Encoding` header and contains the original asset bytes

#### Scenario: Neighboring JSON operation requested with gzip accepted

- **WHEN** a client requests the logical backup list or the server version with `Accept-Encoding: gzip`
- **THEN** the response is gzip-compressed JSON rather than being excluded with the download operations

### Requirement: Request execution retains its own cancellation and metadata

Execution of every user and verification-agent API request SHALL use that request's own cancellation signal, deadline and metadata. Reusing transport resources for a subsequent request SHALL NOT replace the metadata or cancellation observed by work still completing for an earlier request. This SHALL apply to system operations as well as module operations, without introducing a fixed response deadline.

#### Scenario: Transport resources are reused for another request

- **WHEN** an API request captures its execution context and the transport subsequently reuses its request-processing resources for another request
- **THEN** the captured context still identifies the first request
- **AND** cancellation of the second request does not cancel the first request's captured context

#### Scenario: Request cancellation reaches execution

- **WHEN** a request is cancelled while its operation is running
- **THEN** work using the request's execution context observes that cancellation
- **AND** any deadline and request identifier supplied for that request remain available to the operation

#### Scenario: Verification-agent request retains its metadata

- **WHEN** an authenticated verification agent calls an operation and another request subsequently uses the same transport resources
- **THEN** the earlier operation retains its own request metadata and cancellation signal

### Requirement: Authenticated identity belongs to the current request

An operation that requires an authenticated user SHALL act only as the user authenticated for its current request. It SHALL NOT obtain an identity from an earlier or later request through reused transport state. A missing identity SHALL NOT cause the operation to run with an empty or another request's identity. Existing session rejection and workspace authorization responses SHALL remain unchanged.

#### Scenario: Requests authenticate as different users

- **WHEN** two requests authenticate as different users and their transport resources are reused
- **THEN** each operation receives only its own authenticated identity
- **AND** neither request can acquire the other's workspace access through that reuse

#### Scenario: A request has no valid session

- **WHEN** a client without a valid session calls an operation requiring one
- **THEN** the response remains 401 with code `unauthenticated`
- **AND** the operation does not run as another user
