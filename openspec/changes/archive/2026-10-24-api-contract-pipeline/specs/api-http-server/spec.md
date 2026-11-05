## MODIFIED Requirements

### Requirement: An unknown API path returns 404

A request to a path under `/api/` that matches no API route SHALL get the status `404` with the content type `application/problem+json` and a `Problem` body whose `code` is `not_found`. It SHALL NOT get the frontend page. A request to any other unknown path SHALL keep getting the frontend page, so the frontend's own routes keep working on a page reload.

#### Scenario: Mistyped API path

- **WHEN** a client sends `GET /api/v1/no-such-route`
- **THEN** the response status is `404`
- **AND** the content type is `application/problem+json` and the body's `code` is `not_found`

#### Scenario: Frontend route on reload

- **WHEN** a browser requests the frontend route `/auth/callback`
- **THEN** the response is the frontend page with the status `200`

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

## ADDED Requirements

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

A route mounted among the routes served from the contract, whose method and path the contract does not describe, SHALL answer every request with `500` and `code` `internal` and SHALL NOT run its handler. The failure SHALL be logged with the method and the path. Routes of code that has not moved into the target layout are not mounted among them and are not checked.

#### Scenario: An operation is mounted but missing from the contract

- **WHEN** a contract route is mounted for a method and path the contract does not describe, and a client calls it
- **THEN** the response is `500` with `code` `internal`, the handler does not run, and the log names the method and the path

### Requirement: The version operation keeps its path and body

`GET /api/v1/system/version` SHALL answer `200` with the content type `application/json` and the body `{"version": "<version>"}`, where the version is the one the running Databasus reports. It SHALL require no session or token. Its path and the `version` field SHALL NOT change in any release, because installed verification agents update themselves through it.

#### Scenario: An agent asks for the version without a token

- **WHEN** a client without a session calls `GET /api/v1/system/version`
- **THEN** the response is `200` and its JSON body has the field `version` with the running version
