## RENAMED Requirements

- FROM: `Backend errors are translated by code, with the message as fallback`
- TO: `Backend errors are translated by code without a free-text fallback`

## MODIFIED Requirements

### Requirement: Backend errors are translated by code without a free-text fallback

The frontend SHALL translate an error response by its stable error code when that code has a translation. When the code is absent or unknown, it SHALL show a general error message in the selected language. It MUST NOT use the response's message, title, detail, raw code or request address as fallback copy. This change does not add backend codes or change the HTTP contract.

#### Scenario: The backend supplies a known error code

- **WHEN** a request fails with an error code the frontend has a translation for
- **THEN** the user sees the translated message in the selected language

#### Scenario: The backend supplies no code

- **WHEN** a request fails with only a free-text message
- **THEN** the user sees a general translated error rather than that message

#### Scenario: The backend supplies an unrecognized code

- **WHEN** a request fails with an error code the frontend has no translation for and a nonempty message
- **THEN** the user sees a general translated error rather than the message or raw code

#### Scenario: The backend supplies an unrecognized code and no message

- **WHEN** a request fails with an error code the frontend has no translation for and no message
- **THEN** the user sees a general error message in the selected language rather than the raw code

#### Scenario: The request fails without an answer from the backend

- **WHEN** the server cannot be reached, or answers with an error status but no usable error description
- **THEN** the user sees a message in the selected language rather than a request address or a status line

## ADDED Requirements

### Requirement: Technical error diagnostics require an explicitly supported code

An error's technical detail SHALL be displayed only when its known code explicitly allows diagnostic text under the API contract. Connection failure reasons for `database_connection_failed` and required/free disk bytes for `insufficient_disk_space` SHALL remain available as plain text with translated surrounding copy. Unrecognized errors MUST NOT expose technical detail through that mechanism.

#### Scenario: Connection diagnostics are useful

- **WHEN** a request fails with `database_connection_failed` and a connection reason
- **THEN** the user sees the translated connection-failure message and that reason rendered as text

#### Scenario: Disk diagnostics are useful

- **WHEN** a request fails with `insufficient_disk_space` and the required and free byte counts
- **THEN** the user sees the translated capacity message and those diagnostic counts

#### Scenario: An unknown code supplies diagnostic-looking text

- **WHEN** an unknown code includes a nonempty detail, including markup-like text
- **THEN** the user sees only the general translated error and no part of that detail

#### Scenario: A code resembles an inherited property

- **WHEN** an unrecognized code matches a built-in object-property name
- **THEN** it still selects the general translated error and exposes no server text
