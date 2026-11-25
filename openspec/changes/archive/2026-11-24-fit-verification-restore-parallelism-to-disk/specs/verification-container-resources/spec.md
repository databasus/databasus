## MODIFIED Requirements

### Requirement: Verification runs on hosts that cannot enforce a CPU limit

The verification agent SHALL create restore containers without a CPU limit when the Docker host reports at agent startup that it cannot enforce one. Verification on such a host SHALL proceed instead of failing at container creation.

#### Scenario: Host lacks CPU limit support

- **WHEN** the Docker host reports at agent startup that it cannot enforce a CPU limit
- **AND** the agent claims a verification job
- **THEN** the agent creates the restore container without a CPU limit
- **AND** the restore and verification proceed to their normal terminal status

#### Scenario: CPU budget still sizes the restore

- **WHEN** the Docker host cannot enforce a CPU limit
- **AND** the agent runs with a CPU total of 4 and 2 concurrent jobs
- **THEN** the agent's claim reports a CPU total of 4 and 2 concurrent jobs
- **AND** the assignment it receives names at most 2 parallel jobs, the per-job CPU budget
- **AND** the restore tool runs with exactly the number of parallel jobs the assignment names
