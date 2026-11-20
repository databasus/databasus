## ADDED Requirements

### Requirement: A connection test only reports its outcome

Testing a saved storage SHALL run its provider's connection check with the stored settings, and SHALL answer `200` with the outcome: connected, or not connected together with the error the storage gave. Each provider SHALL keep its existing check; the S3 and Azure Blob providers also write a probe object and remove it. A viewer of the workspace SHALL be allowed to test a saved storage. A test SHALL change nothing in the saved storage, its secrets included. A storage recovered from a Databasus backup SHALL be tested like any other storage, and the test SHALL NOT be a precondition for using it.

#### Scenario: Test of a working storage

- **WHEN** a viewer tests a saved local storage whose folders are writable
- **THEN** the response is `200` and reports the storage as connected without changing the saved storage

#### Scenario: Test of a storage that refuses the write

- **WHEN** a member tests a saved S3 storage whose bucket does not exist
- **THEN** the response is `200` and reports the storage as not connected with an error that names the missing bucket

#### Scenario: Test of a recovered local storage

- **WHEN** an editor tests a local storage right after a Databasus recovery
- **THEN** the response reports the outcome of the ordinary check and no column of the storage changes

### Requirement: Storage secrets stay encrypted and untouched by tests

Every secret of a storage SHALL be stored encrypted with the instance's master key, whether it was entered by the user or produced by the backend. No operation SHALL write a storage's secret to the database in clear text.

Connection tests SHALL preserve all saved settings and secrets, including their encrypted representations. They SHALL NOT persist credentials changed in memory during a probe or rewrite any storage or provider-settings column.

#### Scenario: Connection test writes nothing

- **WHEN** a member tests a saved S3 storage and then a saved local storage
- **THEN** no column of either storage or of its provider settings changes, so no test can store a credential that the backend changed in memory

## REMOVED Requirements

### Requirement: A connection test reports its outcome and changes nothing

**Reason**: Replaced by "A connection test only reports its outcome": a recovered local storage no longer needs validation, so the test loses its validation transition and its extended probe.

**Migration**: Nothing to do. A storage still marked as requiring validation becomes usable after the upgrade.

### Requirement: Stored secrets stay encrypted

**Reason**: Replaced by "Storage secrets stay encrypted and untouched by tests", which drops the validation-status exception.

**Migration**: Nothing to do.
