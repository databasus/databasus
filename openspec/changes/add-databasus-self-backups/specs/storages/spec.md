## MODIFIED Requirements

### Requirement: A storage that backups use cannot be deleted

After ordinary authorization, deletion SHALL check an active Databasus backup destination before database-backup references or outstanding cleanup. Deleting a selected Databasus backup destination SHALL get 409 with code `databasus_backup_in_use` and SHALL change nothing, including when it is also used by database backups. Its interface SHALL explain that root admin must first remove it from the Databasus backups section; root admin SHALL get a direct route to that section. If no Databasus backup destination selects the storage, a database backup config or retained database backup reference SHALL instead cause 409 with code `in_use`. This ordering SHALL hold against concurrent selection and deletion. Removing a Databasus backup selection SHALL not delete existing archives; Databasus backup history alone SHALL not prevent later deletion of an otherwise unused storage and SHALL become unavailable history when that storage is deleted. Deleting an otherwise unused storage SHALL follow the pending-cleanup acknowledgement rules of `storage-file-cleanup`: queued files require explicit abandonment, and successful deletion answers 204. Abandonment SHALL NOT override retained references or an active Databasus backup selection.

#### Scenario: Storage used by a backup config

- **WHEN** a member deletes a storage that the backup config of a database of the workspace uses and that is not a selected Databasus backup destination
- **THEN** the response is `409` with `code` `in_use`, and the storage is still listed

#### Scenario: Unused storage

- **WHEN** a member deletes a storage that nothing references and that has no pending file deletion
- **THEN** the response is `204`, and the storage is no longer listed

#### Scenario: Selected Databasus backup destination

- **WHEN** an authorized storage editor deletes a selected Databasus backup destination
- **THEN** deletion gets 409 with code `databasus_backup_in_use` and the interface explains that root admin must first deselect it in global settings

#### Scenario: Storage has both database and Databasus backup uses

- **WHEN** an authorized editor deletes a selected Databasus backup destination that also has a database backup config, retained database backups and pending cleanup
- **THEN** the response is 409 with code `databasus_backup_in_use` regardless of cleanup abandonment and nothing changes
- **AND** after deselection, another deletion attempt still gets `in_use` while the database-backup references remain

#### Scenario: Deletion races destination selection

- **WHEN** selecting a storage for Databasus backups overlaps its deletion
- **THEN** either selection succeeds and deletion is refused, or deletion succeeds and selection is refused without changing the previous configuration

#### Scenario: Databasus backup history remains after deselection

- **WHEN** root admin deselects a storage and an authorized editor later deletes it with no other blocking references or cleanup
- **THEN** deletion succeeds, existing Databasus backup objects are not automatically erased and the Databasus backup history no longer offers that storage as accessible

### Requirement: A connection test reports its outcome and changes nothing

Testing a saved storage SHALL run its provider's connection check with the stored settings, and SHALL answer `200` with the outcome: connected, or not connected together with the error the storage gave. Each provider keeps the check it runs today: it connects with the settings and reaches the configured location, and the S3 and Azure Blob providers also write a probe object there and remove it. A viewer of the workspace SHALL be allowed to test a saved storage. Except for the recovery-validation transition below, a test SHALL change nothing in the saved storage, its secrets included.

After recovery activation, a successful saved connection test of a recovered local storage SHALL change its `recoveryValidationStatus` from `REQUIRES_VALIDATION` to `AVAILABLE` only when the caller is authorized to edit that storage and the stored settings have not changed since the probe began. This exception SHALL change no settings or secrets. A viewer's test SHALL remain diagnostic only, even when successful, and SHALL NOT validate the recovered storage. Failed tests and tests of unsaved settings SHALL NOT make it available. The recovery pause SHALL block writing probes before activation. The interface SHALL distinguish successful connectivity from recovery validation and explain that an editor must validate a storage that remains unavailable.

#### Scenario: Test of a working storage

- **WHEN** a viewer tests a saved local storage whose folders are writable and which does not require recovery validation
- **THEN** the response is `200` and reports the storage as connected without changing the saved storage

#### Scenario: Test of a storage that refuses the write

- **WHEN** a member tests a saved S3 storage whose bucket does not exist
- **THEN** the response is `200` and reports the storage as not connected with an error that names the missing bucket

#### Scenario: Viewer checks an unvalidated recovered storage

- **WHEN** a viewer successfully tests a recovered local storage after activation
- **THEN** the response reports connected, but the storage remains `REQUIRES_VALIDATION` and unavailable for ordinary operations
- **AND** the interface explains that an authorized editor must validate it

#### Scenario: Editor validates an unvalidated recovered storage

- **WHEN** a user authorized to edit a recovered local storage successfully tests its saved settings after activation and those settings remain unchanged throughout the probe
- **THEN** the response reports connected and the storage becomes `AVAILABLE` without changing its settings or secrets

#### Scenario: Editor's validation probe fails

- **WHEN** an authorized editor tests a recovered local storage after activation and the probe fails
- **THEN** the response reports not connected with a safe diagnostic and the storage remains `REQUIRES_VALIDATION`

#### Scenario: Settings change during a validation probe

- **WHEN** a recovered local storage's settings change while an editor's successful connection-test probe is running
- **THEN** that result does not validate the changed settings and the storage remains `REQUIRES_VALIDATION`

### Requirement: Stored secrets stay encrypted

Every secret of a storage SHALL be stored encrypted with the instance's master key, whether it was entered by the user or produced by the backend. No operation SHALL write a storage's secret to the database in clear text.

Connection tests SHALL preserve all saved settings and secrets, including their encrypted representations. The only permitted storage-column change SHALL be the authorized `recoveryValidationStatus` transition from `REQUIRES_VALIDATION` to `AVAILABLE` under the requirement "A connection test reports its outcome and changes nothing". It SHALL NOT persist credentials changed in memory during a probe or rewrite other storage or provider-settings columns.

#### Scenario: Connection test writes nothing

- **WHEN** a member tests a saved S3 storage and then a saved local storage that does not qualify for the authorized recovery-validation transition
- **THEN** no column of either storage or of its provider settings changes, so no test can store a credential that the backend changed in memory

#### Scenario: Recovery validation preserves settings and encrypted secrets

- **WHEN** an authorized editor successfully tests a recovered local storage after activation, its status is `REQUIRES_VALIDATION` and its saved settings remain unchanged throughout the probe
- **THEN** only its `recoveryValidationStatus` changes to `AVAILABLE`, and all other storage and provider-settings columns, including encrypted secret bytes, remain unchanged

## ADDED Requirements

### Requirement: Databasus backup selection does not elevate storage edit or transfer restrictions

Selecting a storage for Databasus backups SHALL NOT change the ordinary permissions or validation for editing its name, address or credentials, or transferring it between workspaces. Selection SHALL remain a reference to the same storage and SHALL follow a permitted transfer. Only root admin SHALL change the Databasus backup selection; a workspace editor SHALL NOT gain access to the full Databasus backup file or key through the Databasus API. The choice of shared workspace membership and the external storage account SHALL remain the owner's responsibility.

#### Scenario: Workspace editor changes a selected storage

- **WHEN** an ordinary authorized workspace editor submits a valid settings change for a selected Databasus backup storage
- **THEN** the ordinary update succeeds and the next Databasus backup uses its current settings
