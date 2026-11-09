## MODIFIED Requirements

### Requirement: The production server binary contains no test code
No package that the server binary in the image links SHALL import the standard testing package, a test assertion or mocking library, or the test container library. The architecture check SHALL fail when a package in the dependency graph of the server's entry point imports any of them, and name the production package that pulls it in. The same rule SHALL apply to third-party dependencies without an importer allowlist.

#### Scenario: A production helper imports an assertion library
- **WHEN** a production file of any package reachable from the server's entry point imports an assertion library
- **THEN** the architecture check fails and names that package and the import

#### Scenario: A new third-party dependency imports a test library
- **WHEN** a dependency the server starts to link imports the standard testing package in a production file
- **THEN** the architecture check fails and names that package and the import

#### Scenario: An accepted third-party importer
- **WHEN** an otherwise accepted third-party storage dependency imports a test library in production code
- **THEN** the architecture check fails and names that dependency and import

#### Scenario: Shared test helpers
- **WHEN** a helper used by tests of several packages lives in a package that only test files import
- **THEN** the architecture check passes

## ADDED Requirements

### Requirement: Production binaries exclude the rclone integration

The published server and verification-agent binaries SHALL contain no rclone storage integration or bundled rclone providers. The production dependency check SHALL fail if either entry point depends on that integration, directly or transitively, and SHALL name the offending dependency. A comment, a removal notice or the absence of an interface option SHALL NOT substitute for this check.

#### Scenario: A direct production dependency brings back rclone

- **WHEN** the server or verification agent directly depends on the rclone integration
- **THEN** the production dependency check fails and names the dependency

#### Scenario: A transitive dependency brings back rclone

- **WHEN** a dependency of either production binary depends on the rclone integration
- **THEN** the production dependency check fails even though the entry point has no direct reference to it

#### Scenario: Only supported providers are linked

- **WHEN** the production binaries contain supported providers without rclone
- **THEN** the rclone dependency check passes while other production dependency restrictions remain enforced
