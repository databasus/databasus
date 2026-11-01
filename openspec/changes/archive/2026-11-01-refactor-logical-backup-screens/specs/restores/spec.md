## ADDED Requirements

### Requirement: The backup's restore dialog stays open when its backup leaves the list

The backup's restore dialog SHALL hold the backup the user opened it for. When that backup leaves the backup list, for example because retention removed it, the dialog SHALL stay open with its restores and with any entered target. A start for a backup that no longer exists SHALL show the translated refusal and keep the form open with every entered value, as any refused start does.

#### Scenario: Retention removes the backup while the dialog is open

- **WHEN** the backup's restore dialog is open and a list refresh no longer contains that backup
- **THEN** the dialog stays open and shows the same restores

#### Scenario: A target is being entered when the backup goes away

- **WHEN** a member is entering a restore target and the backup leaves the list
- **THEN** the entered values stay in the form

#### Scenario: A start after the backup was removed

- **WHEN** a member submits the target form after the backup was removed and the server refuses the start as not found
- **THEN** the translated reason is shown, the form stays open and every entered value is kept

### Requirement: Restore list failures are recoverable and never repeat

The first failed load of a backup's restores in the dialog SHALL show the translated reason with a retry and SHALL not appear as an empty list. Until a load succeeds, the dialog SHALL offer no way to start a restore and periodic refreshes SHALL wait. After that, a failed refresh SHALL keep the shown restores and show no blocking alert or message.

#### Scenario: The first load fails

- **WHEN** a member opens the backup's restore dialog and its first request for restores fails
- **THEN** the dialog shows the translated reason with a retry instead of the empty list text, offers no target selection and sends no refresh request

#### Scenario: A retry succeeds

- **WHEN** the first load failed and the member retries successfully
- **THEN** the restores and the target selection appear and periodic refreshes begin

#### Scenario: The server becomes unreachable

- **WHEN** the dialog shows restores and its periodic refreshes keep failing
- **THEN** the restores stay on the screen and no blocking alert or message appears

### Requirement: A failed restore cancel is reported once

When a confirmed cancel of a running restore fails, the dialog SHALL show the translated reason once and SHALL keep the restore in the list.

#### Scenario: A running restore cannot be cancelled

- **WHEN** a member confirms cancelling a running restore and the request fails
- **THEN** the translated reason is shown once and the restore stays in the list
