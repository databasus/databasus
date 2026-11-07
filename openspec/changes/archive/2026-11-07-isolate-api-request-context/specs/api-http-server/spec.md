## ADDED Requirements

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
