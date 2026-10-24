## Purpose

Defines how the backend's HTTP server answers a path it does not know and which connection limits protect it, without cutting off long backup downloads.

## ADDED Requirements

### Requirement: An unknown API path returns 404

A request to a path under `/api/` that matches no API route SHALL get the status `404` with a JSON body that carries an error message. It SHALL NOT get the frontend page. A request to any other unknown path SHALL keep getting the frontend page, so the frontend's own routes keep working on a page reload.

#### Scenario: Mistyped API path

- **WHEN** a client sends `GET /api/v1/no-such-route`
- **THEN** the response status is `404`
- **AND** the body is JSON with an error message

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
