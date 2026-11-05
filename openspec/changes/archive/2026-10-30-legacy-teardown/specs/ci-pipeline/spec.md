## MODIFIED Requirements

### Requirement: The backend lint job runs the architecture check
The CI backend lint job SHALL run the backend's architecture check next to the linters, so that the backend-wide structure rules and the production-binary rules fail a pull request, not only a local run.

#### Scenario: A production package imports an assertion library
- **WHEN** a pull request adds an assertion library import to a production package reachable from the server's entry point
- **THEN** the backend lint job fails and names the package

#### Scenario: Old backup code writes to a provider
- **WHEN** a pull request makes backup production code reach a storage provider directly across the storage module boundary
- **THEN** the backend lint job fails under the module-boundary rule

#### Scenario: A backend command violates an architecture rule
- **WHEN** a pull request makes a backend command import a dependency forbidden for that command
- **THEN** the backend lint job fails and identifies the command and the rule

### Requirement: Selected backend CI shards can run together locally

`mise run backend:test-group <shard>...` SHALL run the named CI selections concurrently in separate test processes, with race checking, `-failfast` and a 40-minute timeout per binary, without a helper build tag. It SHALL resolve selections through the existing group package and test selectors. It SHALL validate every shard before starting any test and reject duplicate names, unsharded groups, `other`, unknown groups and indices outside their count. It SHALL wait for every selected shard and return failure when any shard fails. On interruption it SHALL stop only its own test processes and remove its temporary logs.

#### Scenario: Every physical API selection runs

- **WHEN** all three `physical-api` shard names are passed
- **THEN** they run concurrently, and together they execute every top-level physical API test exactly once

#### Scenario: One selected shard fails

- **WHEN** one shard fails while another selected shard still runs
- **THEN** the task waits for the remaining shard, reports the selected shard names with their output and exits unsuccessfully

#### Scenario: A selection is invalid

- **WHEN** any requested shard is invalid or repeated
- **THEN** the task rejects the selection before any test starts

#### Scenario: Ordinary builds select the tests

- **WHEN** a developer runs a valid backend shard selection without a helper build tag in the process environment
- **THEN** its package and test selectors find the selection and its test processes start without adding such a tag
