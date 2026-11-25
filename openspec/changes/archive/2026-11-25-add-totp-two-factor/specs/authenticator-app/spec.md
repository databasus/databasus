## Purpose

Lets a user prove the second factor with codes from an authenticator app (TOTP, RFC 6238) and with single-use recovery codes, so that two-factor sign-in works on an instance without a mail server, and gives users, administrators and the host operator ways to replace or reset an app that is lost.

## ADDED Requirements

### Requirement: A user connects an authenticator app by confirming a code from it

Every active user SHALL be able to connect one authenticator app from the profile. Starting the setup SHALL produce a new random secret of at least 160 bits and SHALL show it both as a QR code of a standard `otpauth://totp/` address and as text for manual entry. The address SHALL name the issuer `Databasus` and the account's email address, and SHALL describe six-digit SHA-1 codes for a 30-second period, so every common authenticator app reads it without manual settings.

The app SHALL become connected only when the user submits a code generated from that secret. Until then the secret SHALL be accepted nowhere: not at sign-in, not for confirmations and not for profile changes. An account SHALL have at most one unconfirmed setup: starting a setup again, from the profile or at sign-in, SHALL replace the previous unconfirmed one, whose secret then stops being confirmable.

An unconfirmed setup SHALL stop being confirmable thirty minutes after it started, and after five wrong confirming codes. A wrong confirming code SHALL count only toward that setup's five, not toward the per-account limit on wrong codes, because whoever confirms a setup already sees its secret and gains nothing by guessing. A setup that stopped being confirmable SHALL leave the account's previous state unchanged; the user starts a new setup.

Connecting an app SHALL issue a new set of recovery codes, as the recovery code requirement describes, and SHALL be recorded in the audit log with the user as author.

#### Scenario: Connecting an app

- **WHEN** a signed-in user starts the setup, scans the QR code and submits the code the app shows
- **THEN** the app is connected, the answer carries a new set of recovery codes, and the next password sign-in asks for a code from the app
- **AND** the audit log holds a record, authored by that user, saying that an authenticator app was connected

#### Scenario: Entering the secret by hand

- **WHEN** a user types the shown secret into an authenticator app instead of scanning the QR code and submits the code the app shows
- **THEN** the app is connected exactly as after scanning

#### Scenario: A wrong code during setup

- **WHEN** a user submits a code that the setup secret did not generate
- **THEN** the app is not connected, no recovery codes are issued, sign-in keeps its previous method, and the per-account limit on wrong codes is unchanged

#### Scenario: Five wrong codes during setup

- **WHEN** a user submits five wrong codes for the same setup and then the correct one
- **THEN** the correct code is refused, the app is not connected, and a newly started setup can be confirmed

#### Scenario: A setup confirmed too late

- **WHEN** a user submits a correct code from the setup secret thirty-one minutes after the setup started
- **THEN** the code is refused and the app stays unconnected

#### Scenario: Abandoned setup

- **WHEN** a user starts the setup and never confirms it
- **THEN** sign-in keeps its previous method and the unconfirmed secret is never accepted as a second factor

#### Scenario: Starting the setup twice

- **WHEN** a user starts a setup, then starts another one, and submits a correct code generated from the first secret
- **THEN** the code is refused, and only a code from the second secret connects the app

### Requirement: The authenticator secret never leaves the server after setup

The secret SHALL be stored encrypted with the instance's master key, so that reading the internal database alone does not reveal it. It SHALL be returned to the client only in the answer of the operation that started its setup, and in no other answer: no later request SHALL return it again, whether the setup is confirmed or not. After confirmation no operation SHALL return the secret or its QR address. The secret, app codes and recovery codes SHALL NOT appear in logs, in audit records or in error messages.

#### Scenario: Reading the profile after setup

- **WHEN** a user with a connected app opens the profile
- **THEN** the profile states that an app is connected and when, and shows neither the secret nor its QR code

#### Scenario: Inspecting the stored data

- **WHEN** someone reads the stored authenticator data without the master key
- **THEN** the secret cannot be read from it

### Requirement: An app code is accepted once, within a small clock drift

The instance SHALL accept a six-digit app code generated for the current 30-second period or for the period just before or just after it, so that a phone whose clock is up to about thirty seconds off still works. Codes for any other period SHALL be refused.

Each accepted code SHALL spend its period for that account: a code for the same period or an earlier one SHALL be refused afterwards, wherever it is submitted, at sign-in, in the profile or in a confirmation. Two requests that carry the same correct code at the same time SHALL NOT both succeed.

#### Scenario: Phone clock slightly behind

- **WHEN** a user submits a code that the app generated for the previous 30-second period
- **THEN** the code is accepted

#### Scenario: An old code

- **WHEN** a user submits a code generated two minutes ago
- **THEN** the code is refused

#### Scenario: Replaying an observed code

- **WHEN** a code has completed a sign-in and someone submits the same code again within the same minute
- **THEN** the second attempt is refused

#### Scenario: The same code twice at once

- **WHEN** two sign-in requests carrying the same correct app code for the same account arrive at the same time
- **THEN** exactly one of them succeeds

### Requirement: Recovery codes replace the app once each

Connecting an app SHALL issue ten recovery codes, each with at least 50 bits of randomness, written in a form that is easy to copy by hand. They SHALL be shown in that answer only and SHALL be stored so that reading the stored form does not reveal them.

Wherever the instance asks for an app code, an unused recovery code SHALL be accepted in its place. A recovery code SHALL work once. Using one SHALL be recorded in the audit log with the account as author.

A signed-in user SHALL be able to see how many unused recovery codes remain, and SHALL be able to replace the whole set with a new one; the replacement SHALL invalidate every earlier code of that account and SHALL be shown once, like the first set.

Disconnecting or resetting the app SHALL delete its recovery codes.

#### Scenario: Signing in without the phone

- **WHEN** a user whose phone is lost submits an unused recovery code instead of an app code at sign-in
- **THEN** the sign-in completes, that recovery code stops working, and the profile shows one code fewer
- **AND** the audit log holds a record, authored by that user, saying that a recovery code was used

#### Scenario: Reusing a recovery code

- **WHEN** a recovery code that has already completed a sign-in is submitted again
- **THEN** it is refused

#### Scenario: Regenerating recovery codes

- **WHEN** a user replaces their recovery codes and later submits a code from the earlier set
- **THEN** the earlier code is refused and the codes from the new set work

### Requirement: Changing the app needs a current second factor

Disconnecting the app, replacing it with another one and regenerating recovery codes SHALL each require a current app code or an unused recovery code in the same request, and each SHALL be available only to the account itself. A session alone SHALL NOT be enough, so that someone holding a stolen session cannot move the second factor to a device of their own. A wrong code SHALL change nothing, SHALL be refused with 400 `second_factor_invalid` and SHALL count toward the per-account limit on wrong codes.

Replacing the app SHALL keep the connected one working until the new one is confirmed with its own code; only then SHALL the old secret and the old recovery codes stop working and the new recovery codes be issued.

While the instance requires the authenticator app method, disconnecting SHALL be refused with 409 `authenticator_app_required`; replacing SHALL remain possible.

#### Scenario: Disconnecting with a code

- **WHEN** a user on an instance that does not require the app method submits a current app code with a disconnect request
- **THEN** the app and its recovery codes are removed and the next password sign-in follows the instance requirement alone

#### Scenario: Disconnecting with only a session

- **WHEN** a request to disconnect, replace or regenerate carries no code or a wrong one
- **THEN** it is refused with 400 `second_factor_invalid` and the app, its secret and its recovery codes are unchanged

#### Scenario: Changing another user's app

- **WHEN** a signed-in user sends a disconnect, replacement or regeneration request for another account
- **THEN** it is refused with 403 `permission_denied` and nothing changes

#### Scenario: Moving to a new phone

- **WHEN** a user starts a replacement with a current code from the old phone and confirms it with a code from the new phone
- **THEN** only the new phone's codes work from then on, and a new set of recovery codes is shown

#### Scenario: Abandoning a replacement

- **WHEN** a user starts a replacement and never confirms it
- **THEN** the old app and its recovery codes keep working

#### Scenario: Disconnecting under the app requirement

- **WHEN** the instance requires the authenticator app method and a user submits a valid disconnect request
- **THEN** the request is refused with 409 `authenticator_app_required` and the app stays connected

### Requirement: Wrong app codes are limited per account

Beyond the five wrong codes a single pending sign-in tolerates, the instance SHALL refuse to check app or recovery codes for an account that has submitted ten wrong ones within the last hour, counted across sign-ins, profile changes and master-key or recovery confirmations. Wrong codes confirming a setup SHALL NOT count. A refused request SHALL answer 429 `too_many_wrong_codes` and SHALL check no code until the hour since the oldest counted wrong code has passed. This limit exists because app codes, unlike emailed ones, can be tried without sending mail: without it, someone who knows the password could start pending sign-ins one after another and keep guessing, and someone holding a session could keep guessing in the profile.

An account's existing sessions SHALL keep working while the limit holds.

#### Scenario: Guessing across many pending sign-ins

- **WHEN** someone who knows an account's password starts new pending sign-ins and submits wrong app codes until ten have failed within an hour
- **THEN** the next submission is refused with 429 `too_many_wrong_codes` without being checked, even if it carries the correct code

#### Scenario: The limit passes

- **WHEN** an hour has passed since the wrong codes that reached the limit
- **THEN** a correct app code completes a sign-in again

### Requirement: An administrator can reset another user's app

An instance administrator SHALL be able to reset another user's authenticator app, which removes the app and its recovery codes. The same rules SHALL decide who may reset whom as decide who may deactivate whom: nobody resets their own app this way, nobody resets root admin's app, and only root admin resets another instance administrator's app. A refused reset SHALL answer 403 `permission_denied` and change nothing. A reset of an account without an app SHALL succeed and change nothing.

A reset SHALL make every pending sign-in of that account that waits for an app code unusable. It SHALL NOT end the account's existing sessions. Under the authenticator app method the account sets up a new app at its next password sign-in. Every reset that removes an app SHALL be recorded in the audit log with the administrator as author and the account named.

#### Scenario: A user lost the phone and the recovery codes

- **WHEN** an administrator resets a member's app on an instance that requires the app method, and the member then signs in with the password
- **THEN** the member is taken through a new app setup before any session is issued
- **AND** the audit log holds a record, authored by that administrator, naming the member and saying that the app was reset

#### Scenario: A sign-in in flight and a session during a reset

- **WHEN** a member has a session in one browser and a pending sign-in waiting for an app code in another, and an administrator resets the member's app
- **THEN** submitting a correct code from the old app for that pending sign-in is refused, and the existing session keeps working

#### Scenario: Resetting one's own app

- **WHEN** an administrator sends the reset for their own account
- **THEN** the request is refused with 403 `permission_denied` and the app stays connected

#### Scenario: Resetting root admin

- **WHEN** another administrator tries to reset root admin's app
- **THEN** the request is refused with 403 `permission_denied` and the app stays connected

#### Scenario: An administrator resets another administrator

- **WHEN** an administrator who is not root admin tries to reset another administrator's app
- **THEN** the request is refused with 403 `permission_denied`

#### Scenario: Resetting an account without an app

- **WHEN** an administrator resets a member who has no app
- **THEN** the request succeeds, nothing changes, and no audit record is written

### Requirement: An operator can reset an account's app from the host

An operator with shell access to the running instance SHALL be able to reset one account's authenticator app and its recovery codes with a single command that names the account by its email address, so that an owner who has lost both the phone and the recovery codes can sign in again. The command SHALL have the same effect on pending sign-ins and sessions as an administrator's reset, SHALL report what it changed, SHALL succeed without complaint when the account has no app, and SHALL fail with a non-zero exit code and change nothing when no account has that address.

A run that removes an app SHALL be recorded in the audit log with no user as author, saying that the app was reset from the host and naming the account. The command SHALL require the same trust level as the other console commands: the ability to execute commands inside the running instance.

#### Scenario: The owner lost the phone

- **WHEN** an operator runs the reset command for root admin's address
- **THEN** the app and its recovery codes are removed, the command reports it, and the next password sign-in follows the instance requirement as for an account without an app
- **AND** the audit log holds a record without a user author, naming root admin's account and saying that its app was reset from the host

#### Scenario: No app connected

- **WHEN** the operator runs the reset command for an account without an app
- **THEN** the command reports that nothing changed and exits successfully

#### Scenario: Unknown address

- **WHEN** the operator runs the reset command with an address that belongs to no account
- **THEN** the command says so, exits with a non-zero code and changes nothing
