## ADDED Requirements

### Requirement: S3 storage classes are provider-defined settings

Creating, reading, changing and testing an S3-compatible storage SHALL accept a storage class supplied by the user without restricting it to the application's suggested classes. The application SHALL preserve and pass that value to the storage service exactly. An omitted or empty class SHALL select the bucket's default. The edit form SHALL allow a custom class and SHALL preserve an existing custom class when another field changes.

A remote service's rejection of an unsupported class SHALL be reported through the normal connection-test or backup failure outcome; it SHALL NOT be silently retried with a different class.

#### Scenario: Saving a custom class

- **WHEN** a member creates an otherwise valid S3 storage with class COLD
- **THEN** creation succeeds and reading the storage returns COLD

#### Scenario: Editing another field

- **WHEN** a member renames a storage whose class is DEEP_ARCHIVE without changing that class
- **THEN** the saved storage still uses DEEP_ARCHIVE

#### Scenario: Default class

- **WHEN** a member saves an S3 storage without a class or with an empty class
- **THEN** writes leave the class choice to the bucket

#### Scenario: Provider rejects the class

- **WHEN** the storage service rejects the configured class during a connection test
- **THEN** the test reports the provider failure and the saved class stays unchanged
