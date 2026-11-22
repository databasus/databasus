# Managed Object Storage Tests Specification

## Purpose

Checks Databasus S3 storage against a real managed object storage service, Neon Object Storage, so that S3 behavior that a MinIO container cannot reproduce stays verified.

## Requirements

### Requirement: A Neon Object Storage file cycle runs through the S3 storage

With Neon Object Storage settings available, a storage provider test SHALL use the Databasus S3 storage, configured as the storage form configures a Neon Object Storage storage, against a real Neon bucket. The test SHALL:

- pass the storage's connection test;
- save a file small enough to be stored as one object and read it back byte for byte;
- save a file large enough to be split into several multipart pieces, read it back and find the same SHA-256 digest as the saved content;
- delete both files and then find no object and no incomplete multipart upload left under the run's key prefix;
- cancel a save of a large file after its first pieces were uploaded and then find no object and no incomplete multipart upload left for that file.

The same scenario SHALL also run against a MinIO server in a container of the test harness, so that the scenario's own setup, assertions and cleanup are verified on every run without Neon. Both SHALL belong to the storage provider tests, so that `mise run test` runs them.

#### Scenario: The cycle passes

- **WHEN** the test runs with a reachable Neon bucket and valid credentials
- **THEN** every save, read and delete succeeds, both files read back intact, and nothing of the run remains in the bucket

#### Scenario: A file reads back different

- **WHEN** the content read back differs from the content saved
- **THEN** the test fails naming the file and the size or digest that differs

#### Scenario: Something of a file remains

- **WHEN** an object or an incomplete multipart upload remains under the run's prefix after a delete or after the cancelled save
- **THEN** the test fails naming each remaining key and, for an upload, its upload identifier

#### Scenario: The cycle runs without Neon

- **WHEN** `mise run test` runs without Neon Object Storage settings
- **THEN** the scenario still runs against MinIO and passes

### Requirement: Concurrent runs share one Neon bucket without interference

Several local worktrees and both CI architectures SHALL be able to run the test at once against one Neon bucket. Each run SHALL write only under its own key prefix, which carries the test's marker, a run-unique part and its creation time. When its test ends, whether it passed or failed, a run SHALL remove every object and incomplete multipart upload under its prefix, and SHALL fail naming each key it could not remove. A run killed before its cleanup leaves its objects to later runs: before its own work, a run SHALL remove the objects and incomplete multipart uploads under marked prefixes whose creation time is older than one hour, reading the time from the prefix, and SHALL leave younger ones alone. A sweep that cannot remove an object or upload of an old prefix SHALL log the key and let the run continue, because another run may be sweeping the same prefix at the same moment; only a failure of the run's own cleanup fails the test. The test SHALL NOT change or remove any object outside a marked prefix.

#### Scenario: Two runs overlap

- **WHEN** two runs of the test execute at the same time against one Neon bucket
- **THEN** both pass and neither reads or removes the other's objects

#### Scenario: A killed run left objects behind

- **WHEN** a run starts while the bucket holds objects under a marked prefix created two hours earlier and under another created five minutes earlier
- **THEN** the run removes the older prefix's objects and keeps the younger ones

#### Scenario: The sweep cannot remove an old object

- **WHEN** a run's sweep fails to remove an object of a marked prefix created two hours earlier
- **THEN** the run logs that key and goes on with its own work, and its result does not depend on the sweep

#### Scenario: Objects without the marker

- **WHEN** a run starts while the bucket holds an object created long ago outside any marked prefix
- **THEN** the run leaves it unchanged

#### Scenario: Cleanup cannot remove an object

- **WHEN** a run cannot remove one of its objects at the end
- **THEN** the test fails with a message naming that key

### Requirement: Missing Neon Object Storage settings skip locally and fail where they are required

The test SHALL read the Neon Object Storage settings from the environment variables `TEST_NEON_S3_ENDPOINT`, `TEST_NEON_S3_REGION`, `TEST_NEON_S3_BUCKET`, `TEST_NEON_S3_ACCESS_KEY` and `TEST_NEON_S3_SECRET_KEY`; `mise run test` and `mise run test-full` SHALL provide them from `.env` like the other test settings. When all five are empty or missing, the Neon test SHALL skip with a message that names them. When only some are set, the test SHALL fail naming the missing ones, so a mistyped name never turns the test off silently. `TEST_IS_NEON_S3_REQUIRED` SHALL accept `true`, `false` or an empty value; any other value SHALL fail the test with a message naming the variable. When it is `true`, missing settings SHALL fail the test instead of skipping it. An unreachable endpoint, a missing bucket or refused credentials SHALL fail the test whether or not it is required; they SHALL NOT be reported as a skip.

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

### Requirement: The Neon Object Storage secret key never reaches test output

Neither the secret key nor the access key SHALL appear in test output, failure messages, test logs or files the run keeps. The endpoint, the region, the bucket and object keys MAY appear, because a reader needs them to tell which bucket failed. The MinIO twin SHALL check this on every run with a deliberately wrong secret key. The testing documentation SHALL say that the Neon tests use a dedicated Neon project that holds no production data.

#### Scenario: Credentials are refused

- **WHEN** the scenario connects with a secret key that the storage refuses, Neon or its MinIO twin
- **THEN** its failure message contains neither the secret key nor the access key, and names the endpoint and the bucket
