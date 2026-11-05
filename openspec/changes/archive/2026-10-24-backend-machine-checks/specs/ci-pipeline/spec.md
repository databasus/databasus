## ADDED Requirements

### Requirement: The lint jobs check Go dependencies
The CI lint jobs for the backend and for the verification agent SHALL verify that the downloaded Go modules match the recorded checksums and SHALL scan the module for known vulnerabilities in code it actually calls. A checksum mismatch or a reachable known vulnerability SHALL fail the job. The vulnerability scanner SHALL be pinned to a fixed version, so a run does not change its behavior when the scanner releases.

#### Scenario: A reachable vulnerability
- **WHEN** the backend calls a function of a dependency version with a published vulnerability
- **THEN** the backend lint job fails and names the vulnerability and the call path

#### Scenario: A vulnerability in code that is never called
- **WHEN** a dependency has a published vulnerability only in a function the backend never calls
- **THEN** the backend lint job passes

#### Scenario: A vulnerability in the verification agent
- **WHEN** the verification agent calls a function of a dependency version with a published vulnerability
- **THEN** the verification agent lint job fails and names the vulnerability and the call path

#### Scenario: A tampered module
- **WHEN** a downloaded module's content does not match its recorded checksum in either module
- **THEN** the lint job fails before the linters run

### Requirement: The backend lint job runs the architecture check
The CI backend lint job SHALL run the backend's architecture check next to the linters, so that the structure rules, the production-binary rule and the rule against direct provider writes in old backup code fail a pull request, not only a local run.

#### Scenario: A production package imports an assertion library
- **WHEN** a pull request adds an assertion library import to a production package reachable from the server's entry point
- **THEN** the backend lint job fails and names the package

#### Scenario: Old backup code writes to a provider
- **WHEN** a pull request makes old backup code call a storage provider's delete method directly
- **THEN** the backend lint job fails

### Requirement: CI backend tests run with the race detector
Every CI backend test group SHALL run with the race detector. A data race SHALL fail the group like any other test failure, under the same rules for failed groups as before.

#### Scenario: A race in one shard
- **WHEN** a test in one shard of the physical backup group triggers a data race
- **THEN** that shard fails, and the other groups keep running and report their own results
