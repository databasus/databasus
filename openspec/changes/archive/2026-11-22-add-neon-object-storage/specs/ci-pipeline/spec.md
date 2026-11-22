## ADDED Requirements

### Requirement: The storage-providers group requires the Neon Object Storage test

The `storage-providers` check SHALL run the Neon Object Storage test on both architectures with the Neon Object Storage settings from repository secrets and with the test required, so that a missing secret fails the check instead of skipping the test. Because the `storage-providers` check gates release, a missing secret, an unreachable bucket or a failed Neon Object Storage cycle SHALL prevent version selection and every publication operation, as any failed required check does. No other check SHALL receive these secrets.

#### Scenario: The secrets are configured

- **WHEN** CI runs the `storage-providers` check on `main` with the Neon Object Storage secrets set and the bucket reachable
- **THEN** the Neon Object Storage test runs and its result counts toward the check on each architecture

#### Scenario: A secret is missing

- **WHEN** CI runs the `storage-providers` check without one of the Neon Object Storage secrets
- **THEN** the check fails with a message naming the missing variable
- **AND** the commit is not eligible for release

#### Scenario: Another group runs

- **WHEN** CI runs any check other than `storage-providers`
- **THEN** that check's environment holds no Neon Object Storage setting

### Requirement: The Neon Object Storage secrets are exposed only to the step that needs them

The Neon Object Storage endpoint, region, bucket, access key and secret key SHALL reach CI only as repository secrets. They SHALL be set in the environment of the step that runs the `storage-providers` tests and in no earlier step, no other job and no workflow output or artifact. CI masks the exact secret values in logs; the test itself keeps the keys out of its output, as the test's own requirement says.

#### Scenario: The workflow is read

- **WHEN** a reader inspects the workflow definition
- **THEN** each Neon Object Storage secret appears only in the environment of the backend test step, and only for the `storage-providers` group
