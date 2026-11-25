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

When the instance does not require two-factor sign-in, the active root-admin session SHALL be sufficient, and the export SHALL ask for no further identity confirmation, whether or not root admin has a connected authenticator app. Anyone holding root admin's session can then download the key; requiring two-factor sign-in is the protection against that. A confirmation request SHALL then be refused with 409 `identity_proof_unavailable`.

When the instance requires two-factor sign-in, by either method, export SHALL additionally require a single-use confirmation by root admin's second factor:

- When root admin has a connected authenticator app, the confirmation SHALL be a current app code or an unused recovery code, checked under the rules of the `authenticator-app` spec. Requesting a confirmation SHALL send no mail, SHALL need no mail server, and SHALL answer that the app code is expected, with no expiry time.
- Otherwise the confirmation SHALL be a code sent to root admin's address through the configured system mail server and bound to root admin's session and to the export purpose. When the instance has no mail server or root admin's address is not usable, a confirmation request SHALL be refused with 409 `identity_proof_unavailable`. This holds under the authenticator app method too, for a root admin whose app was reset or who signs in only through an external identity provider.

The account password SHALL NOT be accepted as this confirmation. A missing, wrong, expired, reused or wrong-purpose code SHALL yield no key.

The downloaded file SHALL contain exactly the existing key; export SHALL NOT regenerate the key or change its bytes.

#### Scenario: Root admin downloads without two-factor sign-in

- **WHEN** the instance does not require two-factor sign-in and root admin obtains the key through the export
- **THEN** the file contains exactly the existing key and the installed key is unchanged
- **AND** no password or code was requested

#### Scenario: Root admin with an app on an instance without the requirement

- **WHEN** the instance does not require two-factor sign-in, root admin has a connected app, and root admin requests a confirmation
- **THEN** the request is refused with 409 `identity_proof_unavailable` and the export succeeds without a code

#### Scenario: Root admin downloads with two-factor sign-in

- **WHEN** the instance requires two-factor sign-in, root admin has no connected app, and root admin requests a code, receives it and supplies it with the download
- **THEN** the file contains exactly the existing key

#### Scenario: Root admin confirms with the app

- **WHEN** the instance requires two-factor sign-in, root admin has a connected app, and root admin requests a confirmation and then supplies a current app code with the download
- **THEN** the confirmation answer names the app method and carries no expiry time, no mail is sent, and the file contains exactly the existing key

#### Scenario: Root admin confirms with a recovery code on an instance without a mail server

- **WHEN** the instance requires the authenticator app method and has no mail server, and root admin with a connected app supplies a recovery code with the download
- **THEN** the file contains exactly the existing key and that recovery code stops working

#### Scenario: Root admin without an app under the app method and without mail

- **WHEN** the instance requires the authenticator app method and has no mail server, root admin's app was reset or disconnected, and root admin requests a confirmation
- **THEN** the request is refused with 409 `identity_proof_unavailable` and no key is transferred

#### Scenario: Two-factor sign-in is required and no valid code is supplied

- **WHEN** the instance requires two-factor sign-in and root admin requests an export without a code, with a wrong, expired or already used emailed code, with a code issued for a recovery, with a wrong app code, with an app code already used in the same 30-second period, or with only the account password
- **THEN** the request is refused and no key is transferred

#### Scenario: Another administrator requests a download

- **WHEN** another administrator directly requests key export
- **THEN** the request is refused with 403 and no key is transferred

### Requirement: Onboarding blocks until root admin confirms saving the current key

On any authenticated visit, root admin SHALL see the key onboarding step instead of the application while the current key has no acknowledgment. This SHALL apply regardless of workspace or database setup and to existing installations. The step SHALL take the whole screen, like the sign-in page, and the rest of the interface SHALL NOT be shown until the acknowledgment is recorded. It SHALL offer no close, skip, defer or "remind later" action. No other master-key reminder, such as a floating notice, SHALL be shown.

The step SHALL explain that Databasus encrypts backups with the key and that the key is needed to decrypt the backups, including the backup of Databasus itself, if the server is lost.

The step SHALL obtain the key through the export: at once when the instance does not require two-factor sign-in, and after root admin supplies the confirmation when it does. When the instance answers that the app code is expected, the step SHALL ask for a current app code or a recovery code and SHALL offer no send action; otherwise it SHALL ask for the emailed code as before. It SHALL then offer two ways to save the key: copying it to the clipboard, next to a preview that shows only part of the key, and downloading it as a text file. The copy SHALL work when the page is served over plain HTTP. The "I've saved it in a safe place" action SHALL stay disabled until root admin copies or downloads the key in this step and 15 seconds have passed since the step appeared; while the time runs, the action SHALL show the seconds left. That action SHALL record the acknowledgment for the current key and open the application. Obtaining, copying or downloading the key alone SHALL NOT record an acknowledgment.

The acknowledgment SHALL be recorded only for root admin, SHALL persist across browsers and sessions and SHALL belong to the current key, so a key replaced by recovery requires a new confirmation.

When the instance requires two-factor sign-in, root admin has no connected app and the emailed confirmation code cannot be sent, the step SHALL show the sending error. It SHALL also explain that the key can be copied from the server by hand from the file `databasus-data/secret.key` in the folder where Databasus is installed. It SHALL NOT show an absolute path or a copy command, because the host path of that folder depends on the installation. The step SHALL stay in place.

The instance settings SHALL offer a Download master key action in the Databasus backups section that opens a master-key dialog. The dialog SHALL explain why the key must be kept outside this server and give instructions for copying the key from the server that name the file `databasus-data/secret.key` in the folder where Databasus is installed, without an absolute path or a copy command. It SHALL NOT show whether the current key is saved, because onboarding already blocks the application while it is not. These instructions SHALL be reachable inside the dialog through at most one keyboard-operable disclosure. Pressing Download master key is the request for the key: the dialog SHALL obtain the key on opening, under the same confirmation rules as the onboarding step, and offer the same partial preview, copy and download in the same layout. Nothing else in the section SHALL obtain the key. Closing the dialog SHALL discard the obtained key from the page.

#### Scenario: First visit of a new installation

- **WHEN** root admin signs in to an installation whose key has no acknowledgment
- **THEN** the onboarding step takes the whole screen instead of the interface and offers no way to skip or defer it
- **AND** no floating master-key notice appears

#### Scenario: Root admin copies the key

- **WHEN** the step shows the partial preview and root admin copies the key
- **THEN** the clipboard holds exactly the existing key and the "I've saved it in a safe place" action becomes enabled once 15 seconds have passed since the step appeared

#### Scenario: Root admin downloads the key

- **WHEN** root admin downloads the key from the onboarding step
- **THEN** the file `databasus-master-key.txt` holds exactly the existing key and the "I've saved it in a safe place" action becomes enabled once 15 seconds have passed since the step appeared

#### Scenario: The key was neither copied nor downloaded

- **WHEN** the step shows the key options and root admin has neither copied nor downloaded the key
- **THEN** the "I've saved it in a safe place" action is disabled and the key stays unacknowledged

#### Scenario: Root admin copies the key right away

- **WHEN** root admin copies the key 5 seconds after the step appeared
- **THEN** the "I've saved it in a safe place" action stays disabled and shows the seconds left until 15 seconds have passed

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

- **WHEN** the instance requires two-factor sign-in, root admin has no connected app, and sending the confirmation code fails
- **THEN** the step shows the sending error and the instructions for copying the key from the server with the file `databasus-data/secret.key` in the folder where Databasus is installed
- **AND** the step stays in place

#### Scenario: Root admin opens the master-key panel

- **WHEN** root admin opens the master-key dialog from the Databasus backups section
- **THEN** the dialog offers the instructions for copying the key from the server with the file `databasus-data/secret.key` in the folder where Databasus is installed
- **AND** it shows no saved-state line, absolute path or `docker cp` command
- **AND** no key export is requested before root admin presses Download master key
- **AND** after that the key is obtained with a code exactly when the instance requires two-factor sign-in, an app code when root admin has a connected app and an emailed code otherwise, and then can be copied or downloaded

#### Scenario: Root admin closes the master-key dialog after obtaining the key

- **WHEN** root admin obtains the key in the dialog and closes it
- **THEN** the key leaves the page, and reopening the dialog obtains it again

#### Scenario: Onboarding with a connected app

- **WHEN** root admin with a connected app reaches the onboarding step on an instance that requires two-factor sign-in
- **THEN** the step asks for an app code or a recovery code, offers no send action, sends no mail, and obtains the key after a correct code
