# Frontend API Requests Specification

## Purpose

Give frontend callers one predictable HTTP failure model while preserving successful empty operations, cancellation and explicit domain requests.

## Requirements

### Requirement: HTTP failures have stable frontend classifications

A request that cannot reach the server SHALL be classified as `network_unreachable`. A response that cannot provide required data, has unreadable JSON, or refuses an operation without a usable Problem SHALL be classified as `request_failed`, retaining its HTTP status when available. A valid Problem SHALL retain its code and status. These failures SHALL reach the interface through translated error handling. An operation whose success has no body SHALL complete successfully without inventing response data. Intentional lifecycle cancellation SHALL NOT be presented as a network or server failure.

#### Scenario: The connection fails

- **WHEN** a declared request fails before receiving a response
- **THEN** its caller receives the network failure classification and the interface can display its translated network message

#### Scenario: A proxy returns an unusable error response

- **WHEN** an operation receives an error status with an empty, non-JSON or structurally invalid Problem body
- **THEN** its caller receives the request failure classification with that status
- **AND** the interface does not display the response text as its error message

#### Scenario: Required data is missing or unreadable

- **WHEN** an operation requiring a JSON body receives an empty response or unreadable JSON
- **THEN** its caller receives the request failure classification rather than a successful empty value

#### Scenario: A bodyless success completes

- **WHEN** an operation whose successful response has no body returns that success status
- **THEN** its caller observes success

#### Scenario: A request is intentionally cancelled

- **WHEN** a request is cancelled because its screen or list lifecycle has ended
- **THEN** it does not produce a network-failure notification

#### Scenario: An accepted transfer omits required operation data

- **WHEN** a database transfer returns status 202 without its required progress data
- **THEN** its caller receives the request failure classification with status 202 rather than treating the transfer as completed

### Requirement: Request handling does not replay mutations implicitly

Failure normalization and complete-list traversal MUST NOT automatically repeat a mutation after a refusal, network failure or unreadable response. A new mutation request SHALL require a new explicit invocation by its caller.

#### Scenario: The outcome of a mutation is unknown

- **WHEN** a mutation fails while awaiting its response
- **THEN** only the original request was sent
- **AND** no automatic retry follows

### Requirement: Importing the shared API starts no request machinery

Loading shared API declarations SHALL NOT start timers, queues, HTTP requests or retry loops. Request work SHALL start only when a caller invokes an operation.

#### Scenario: A screen loads the API declarations

- **WHEN** shared API declarations are imported before any operation is invoked
- **THEN** no interval, queued request or HTTP request starts as a consequence of that import
