# master-key-export Specification

## Purpose
Helps the instance owner retain the existing master key separately and controls its deliberate export without exposing it through ordinary account or settings responses.

## Requirements

### Requirement: Exported key material is never an ordinary response field

The key SHALL leave the server only through the explicitly authorized file download. It SHALL NOT appear in ordinary settings, account or status responses, URLs, logs, telemetry, browser persistent storage or cacheable responses. Download authorization SHALL be short-lived, single-use and purpose-bound. The audit log SHALL record the actor and export event without key material or download credentials.

#### Scenario: A download credential is reused

- **WHEN** a caller reuses an already consumed key-download credential
- **THEN** the download is refused and the key is not returned again

### Requirement: Only root admin exports the master key, with an emailed code when the instance requires two-factor sign-in

Export SHALL require an active root-admin session. Ordinary instance administrators and workspace members SHALL receive no key or export capability.

When the instance does not require two-factor sign-in, the active root-admin session SHALL be sufficient, and the export SHALL ask for no further identity confirmation. Anyone holding root admin's session can then download the key; requiring two-factor sign-in is the protection against that.

When the instance requires two-factor sign-in, export SHALL additionally require a single-use confirmation code sent to root admin's address through the configured system mail server and bound to root admin's session and to the export purpose. The account password SHALL NOT be accepted as this confirmation. A missing, wrong, expired, reused or wrong-purpose code SHALL yield no key.

The downloaded file SHALL contain exactly the existing key; export SHALL NOT regenerate the key or change its bytes.

#### Scenario: Root admin downloads without two-factor sign-in

- **WHEN** the instance does not require two-factor sign-in and root admin obtains the key through the export
- **THEN** the file contains exactly the existing key and the installed key is unchanged
- **AND** no password or code was requested

#### Scenario: Root admin downloads with two-factor sign-in

- **WHEN** the instance requires two-factor sign-in and root admin requests a code, receives it and supplies it with the download
- **THEN** the file contains exactly the existing key

#### Scenario: Two-factor sign-in is required and no valid code is supplied

- **WHEN** the instance requires two-factor sign-in and root admin requests an export without a code, with a wrong, expired or already used code, with a code issued for a recovery, or with only the account password
- **THEN** the request is refused and no key is transferred

#### Scenario: Another administrator requests a download

- **WHEN** another administrator directly requests key export
- **THEN** the request is refused with 403 and no key is transferred

### Requirement: Onboarding blocks until root admin confirms saving the current key

On any authenticated visit, root admin SHALL see the key onboarding step instead of the application while the current key has no acknowledgment. This SHALL apply regardless of workspace or database setup and to existing installations. The step SHALL take the whole screen, like the sign-in page, and the rest of the interface SHALL NOT be shown until the acknowledgment is recorded. It SHALL offer no close, skip, defer or "remind later" action. No other master-key reminder, such as a floating notice, SHALL be shown.

The step SHALL explain that Databasus encrypts backups with the key and that the key is needed to decrypt the backups, including the backup of Databasus itself, if the server is lost.

The step SHALL obtain the key through the export: at once when the instance does not require two-factor sign-in, and after root admin supplies the emailed code when it does. It SHALL then offer two ways to save the key: copying it to the clipboard, next to a preview that shows only part of the key, and downloading it as a text file. The copy SHALL work when the page is served over plain HTTP. The "I've saved it in a safe place" action SHALL stay disabled until root admin copies or downloads the key in this step. That action SHALL record the acknowledgment for the current key and open the application. Obtaining, copying or downloading the key alone SHALL NOT record an acknowledgment.

The acknowledgment SHALL be recorded only for root admin, SHALL persist across browsers and sessions and SHALL belong to the current key, so a key replaced by recovery requires a new confirmation.

When the instance requires two-factor sign-in and the confirmation code cannot be sent, the step SHALL show the sending error. It SHALL also explain that the key can be copied from the server by hand and give the key's path inside the container. The step SHALL stay in place.

The master-key panel in the instance settings SHALL show whether the current key is saved and give instructions for copying the key from the server with the key's path inside the container. It SHALL obtain the key only after root admin asks for it, under the same confirmation rules as the onboarding step, and then offer the same copy and download.

#### Scenario: First visit of a new installation

- **WHEN** root admin signs in to an installation whose key has no acknowledgment
- **THEN** the onboarding step takes the whole screen instead of the interface and offers no way to skip or defer it
- **AND** no floating master-key notice appears

#### Scenario: Root admin copies the key

- **WHEN** the step shows the partial preview and root admin copies the key
- **THEN** the clipboard holds exactly the existing key and the "I've saved it in a safe place" action becomes enabled

#### Scenario: Root admin downloads the key

- **WHEN** root admin downloads the key from the onboarding step
- **THEN** the file `databasus-master-key.txt` holds exactly the existing key and the "I've saved it in a safe place" action becomes enabled

#### Scenario: The key was neither copied nor downloaded

- **WHEN** the step shows the key options and root admin has neither copied nor downloaded the key
- **THEN** the "I've saved it in a safe place" action is disabled and the key stays unacknowledged

#### Scenario: Root admin confirms saving

- **WHEN** root admin presses "I've saved it in a safe place" after copying or downloading the key
- **THEN** the application opens, and later visits in any browser do not show the step for that key

#### Scenario: Another administrator records an acknowledgment

- **WHEN** another administrator directly requests the acknowledgment
- **THEN** the request is refused with 403 and the key stays unacknowledged

#### Scenario: The key is replaced by recovery

- **WHEN** a recovery installs another key on an instance whose previous key was acknowledged
- **THEN** root admin is shown the onboarding step again

#### Scenario: The code cannot be sent

- **WHEN** the instance requires two-factor sign-in and sending the confirmation code fails
- **THEN** the step shows the sending error, the instructions for copying the key from the server and the key's path inside the container
- **AND** the step stays in place

#### Scenario: Root admin opens the master-key panel

- **WHEN** root admin opens the master-key panel in the instance settings
- **THEN** the panel shows whether the current key is saved and the instructions for copying the key from the server with its path inside the container
- **AND** the key is obtained only after root admin asks for it, with a code exactly when the instance requires two-factor sign-in, and then can be copied or downloaded
