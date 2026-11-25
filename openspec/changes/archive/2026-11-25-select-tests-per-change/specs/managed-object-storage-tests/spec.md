## MODIFIED Requirements

### Requirement: Missing Neon Object Storage settings skip locally and fail where they are required

The test SHALL read the Neon Object Storage settings from the environment variables `TEST_NEON_S3_ENDPOINT`, `TEST_NEON_S3_REGION`, `TEST_NEON_S3_BUCKET`, `TEST_NEON_S3_ACCESS_KEY` and `TEST_NEON_S3_SECRET_KEY`; `mise run test` and `mise run test-change` SHALL provide them from `.env` like the other test settings. When all five are empty or missing, the Neon test SHALL skip with a message that names them. When only some are set, the test SHALL fail naming the missing ones, so a mistyped name never turns the test off silently. `TEST_IS_NEON_S3_REQUIRED` SHALL accept `true`, `false` or an empty value; any other value SHALL fail the test with a message naming the variable. When it is `true`, missing settings SHALL fail the test instead of skipping it. An unreachable endpoint, a missing bucket or refused credentials SHALL fail the test whether or not it is required; they SHALL NOT be reported as a skip.

#### Scenario: A local run without the settings

- **WHEN** a developer runs `mise run test` without the Neon Object Storage variables in `.env`
- **THEN** the Neon test is reported as skipped with a message naming the variables
- **AND** the MinIO twin and the other tests are unaffected

#### Scenario: Settings are partly filled

- **WHEN** the test runs with the endpoint, region and bucket set but without the secret key
- **THEN** the test fails with a message naming `TEST_NEON_S3_SECRET_KEY`

#### Scenario: A required run without the settings

- **WHEN** the test runs with `TEST_IS_NEON_S3_REQUIRED=true` and no Neon Object Storage settings
- **THEN** the test fails with a message naming the missing variables

#### Scenario: An invalid required flag

- **WHEN** the test runs with `TEST_IS_NEON_S3_REQUIRED=yes`
- **THEN** the test fails with a message naming `TEST_IS_NEON_S3_REQUIRED`

#### Scenario: Wrong credentials

- **WHEN** the settings hold a secret key that the storage refuses
- **THEN** the test fails rather than skips
