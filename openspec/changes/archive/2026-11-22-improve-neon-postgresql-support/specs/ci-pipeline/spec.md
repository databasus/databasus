## ADDED Requirements

### Requirement: The logical group requires the Neon backup test

The `logical` check SHALL run the Neon backup end-to-end test on both architectures with the Neon connection string from a repository secret and with the test required, so that a missing secret fails the check instead of skipping the test. Because the `logical` check gates release, a missing secret, an unreachable Neon project or a failed Neon cycle SHALL prevent version selection and every publication operation, as any failed required check does. No other check SHALL receive the secret.

#### Scenario: The secret is configured

- **WHEN** CI runs the `logical` check on `main` with the Neon secret set and the Neon project reachable
- **THEN** the Neon test runs and its result counts toward the check on each architecture

#### Scenario: The secret is missing

- **WHEN** CI runs the `logical` check without the Neon secret
- **THEN** the check fails with a message naming `TEST_NEON_DATABASE_URL`
- **AND** the commit is not eligible for release

#### Scenario: Another group runs

- **WHEN** CI runs any check other than `logical`
- **THEN** that check's environment holds no Neon connection string

### Requirement: The Neon secret is exposed only to the step that needs it

The Neon connection string SHALL reach CI only as a repository secret. It SHALL be set in the environment of the step that runs the `logical` tests and in no earlier step, no other job and no workflow output or artifact. CI masks the exact secret value in logs; the password alone is kept out of the logs by the test itself, as the Neon test's own requirement says.

#### Scenario: The workflow is read

- **WHEN** a reader inspects the workflow definition
- **THEN** the secret appears only in the environment of the backend test step, and only for the `logical` group
