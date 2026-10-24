## MODIFIED Requirements

### Requirement: One filesystem suite runs locally and in CI

The repository SHALL provide `mise run test-filesystem` as the single entry point for the Docker filesystem suite. It SHALL build or accept a local candidate image and run the same cases locally and in CI. The suite SHALL sort every case into exactly one case group: `permissions`, `network`, `upgrades` or `lifecycle`. It SHALL run every group when no group or case is selected, the cases of one group when a group is selected, and a single case when a case is selected. An unknown group SHALL fail the run before any case starts.

CI SHALL build the candidate once and run each case group as its own check from that same candidate. The candidate SHALL pass between CI jobs only as a short-lived workflow artifact and SHALL NOT be pushed to or pulled from an image registry.

Each positive case SHALL start the real container and invoke `databasus --test-storage`, which uses the public local-storage save and delete methods. Each negative case SHALL assert the startup failure and its actionable error. Every case SHALL clean its containers, volumes, networks, mounts, and temporary files.

#### Scenario: Developer runs the complete suite

- **WHEN** a developer runs `mise run test-filesystem` without an image reference, a group or a case
- **THEN** the task builds a local candidate and runs every case group against it

#### Scenario: Developer runs one group

- **WHEN** a developer runs `mise run test-filesystem` with the `network` group selected
- **THEN** only the cases of the `network` group run

#### Scenario: An unknown group is selected

- **WHEN** a developer selects a case group that does not exist
- **THEN** the run fails with an error that names the group, and no case starts

#### Scenario: CI tests its candidate

- **WHEN** CI tests a commit
- **THEN** one job builds the candidate, each case group runs as its own check against that candidate, and no candidate image is pushed to or pulled from a registry

#### Scenario: A case allocates Docker resources

- **WHEN** the case succeeds or fails after resource allocation
- **THEN** the harness removes every resource created by that case
