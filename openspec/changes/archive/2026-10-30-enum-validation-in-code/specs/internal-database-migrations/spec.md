## ADDED Requirements

### Requirement: An upgrade preserves provider-defined storage classes

An upgrade SHALL start successfully when the previous release accepted an S3-compatible storage class outside the application's suggested classes. It SHALL preserve that class exactly and SHALL NOT silently replace it with a default. Invalid application-owned settings SHALL still be validated or repaired according to their domain requirements.

#### Scenario: A class supplied by an S3-compatible service

- **WHEN** the previous release accepted an S3 storage configured with class COLD and the instance is upgraded
- **THEN** the instance starts successfully and reading the storage returns COLD

#### Scenario: An AWS class outside the suggestions

- **WHEN** the previous release accepted an S3 storage configured with class DEEP_ARCHIVE and the instance is upgraded
- **THEN** the instance starts successfully and reading the storage returns DEEP_ARCHIVE
