## ADDED Requirements

### Requirement: Contract operations enforce their access mark

Every operation in the API contract SHALL be checked against its access mark before its handler runs. An operation marked public SHALL run without a session. An operation without a mark SHALL require a valid session, and a request without one SHALL get `401` with `code` `unauthenticated`. An operation marked admin SHALL also require the instance admin role, and a signed-in user without it SHALL get `403` with `code` `permission_denied`. A session is invalid when its token cannot be verified, when its user no longer exists or is not active, or when the user's password changed after the token was issued.

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
