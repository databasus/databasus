## MODIFIED Requirements

### Requirement: Account screens preserve existing authentication policy

The interface SHALL retain first registration, ordinary registration, password sign-in, emailed sign-in verification, reset-code requests and password-reset completion under the server's existing policy. Validation and refusal SHALL preserve correction paths and SHALL not duplicate mutations. A successful password change SHALL end the session and require complete ordinary sign-in as specified by two-factor-authentication. Loading failures SHALL provide a translated recovery path rather than an indefinite spinner or browser-native alert.

#### Scenario: The instance has no claimed account

- **WHEN** sign-in options indicate that the instance has no user
- **THEN** the interface opens the existing first-registration experience and successful registration enters the application

#### Scenario: A sign-in requires an emailed code

- **WHEN** the password step returns a pending sign-in that waits for an emailed code
- **THEN** the interface activates no session until verification succeeds and preserves resend, expired-attempt and return-to-password behavior

#### Scenario: A user resets a password

- **WHEN** a user requests a reset code and completes password reset successfully
- **THEN** the interface returns to ordinary sign-in without silently creating a session
- **AND** invalid input or refusal leaves a way to correct or request a new code

#### Scenario: Reset-code delivery succeeds before navigation

- **WHEN** reset-code delivery succeeds and the transition to password completion is pending
- **THEN** button activation and Enter cannot start another delivery
- **AND** completion opens for the email submitted by the successful request

#### Scenario: Password reset succeeds before navigation

- **WHEN** password reset succeeds and the transition to ordinary sign-in is pending
- **THEN** further activation cannot submit another password-reset completion
- **AND** the successful transition occurs once without activating a session

## ADDED Requirements

### Requirement: Sign-in shows the step the password answer names

After the password step, the sign-in screen SHALL show the step the instance named: the emailed code step, the app code step or the app setup step. No step SHALL activate a session before the instance issues one.

The app code step SHALL ask for the six-digit code from the authenticator app and SHALL offer a switch to entering a recovery code instead. It SHALL offer no resend. A wrong code SHALL keep the step open with a message and the entry cleared. When the pending sign-in has expired, was destroyed by wrong codes or became unusable, the step SHALL say so and return the user to the password step. When the per-account limit on wrong codes holds, the step SHALL say that too many wrong codes were entered and that signing in will be possible again later.

The app setup step SHALL explain that the instance requires an authenticator app, SHALL show the QR code and the secret as text with a copy action, and SHALL ask for the code the app generates. A wrong code SHALL keep the step open with a message, the same QR code and the entry cleared. When the setup has expired, was destroyed by five wrong codes or became unusable, the step SHALL say so and return the user to the password step, where a new password step starts a new setup. After a correct code it SHALL show the recovery codes with copy and download actions and SHALL enter the application only when the user confirms having saved them. It SHALL offer no way to skip the setup.

#### Scenario: A user with an app signs in

- **WHEN** the password step names the app code step and the user enters the app's code
- **THEN** the application opens with the issued session

#### Scenario: Signing in with a recovery code

- **WHEN** a user on the app code step switches to the recovery code entry and submits an unused recovery code
- **THEN** the application opens with the issued session

#### Scenario: The app code step offers no resend

- **WHEN** the app code step is shown
- **THEN** it has no action that sends a code

#### Scenario: The app code step can no longer be completed

- **WHEN** a submitted app code is refused because the pending sign-in expired or was destroyed by wrong codes
- **THEN** the step says that the sign-in must start again and shows the password step

#### Scenario: Too many wrong app codes

- **WHEN** a submitted app code is refused because the account reached the limit on wrong codes
- **THEN** the step says that too many wrong codes were entered and that signing in will be possible again later, and no session is activated

#### Scenario: First sign-in under the app requirement

- **WHEN** the password step names the app setup step and the user confirms the setup with a correct code
- **THEN** the screen shows the recovery codes, and the application opens only after the user confirms having saved them

#### Scenario: A wrong code during the sign-in setup

- **WHEN** the user submits a wrong code on the app setup step
- **THEN** the step stays open with the same QR code, shows that the code is wrong and clears the entry

#### Scenario: The sign-in setup can no longer be completed

- **WHEN** a submitted setup code is refused because the setup expired or was destroyed by five wrong codes
- **THEN** the step says that the setup must start again and shows the password step

### Requirement: The profile manages the user's authenticator app

The profile SHALL show whether an authenticator app is connected, since when, and how many unused recovery codes remain. Without an app it SHALL offer connecting one through the same QR code, secret and confirming code as the sign-in setup, followed by the recovery codes with copy and download actions.

With an app it SHALL offer replacing the app, regenerating the recovery codes and disconnecting the app. Each of these SHALL ask for a current app code or a recovery code before the request is sent. A refused code SHALL keep the dialog open with a message and SHALL change nothing shown in the profile. Disconnecting SHALL be absent, with an explanation, while the instance requires the authenticator app method. Recovery codes SHALL be shown only in the answer that issued them and SHALL be discarded from the page when the dialog closes.

#### Scenario: Connecting an app from the profile

- **WHEN** a user connects an app from the profile and confirms it with a code
- **THEN** the profile shows the recovery codes once, and afterwards states that an app is connected with ten recovery codes left

#### Scenario: Replacing the app from the profile

- **WHEN** a user with an app enters a current code, scans the new QR code and confirms with a code from the new app
- **THEN** the profile shows the new recovery codes once and states the new connection date

#### Scenario: Regenerating recovery codes from the profile

- **WHEN** a user with an app enters a current code and asks for new recovery codes
- **THEN** the profile shows ten new codes once, and afterwards states that ten codes are left

#### Scenario: A wrong code in a profile dialog

- **WHEN** a user enters a wrong code to replace, regenerate or disconnect
- **THEN** the dialog stays open with a message and the profile still shows the same app and the same remaining count

#### Scenario: Closing the recovery codes dialog

- **WHEN** the user closes the dialog that shows recovery codes and opens the profile section again
- **THEN** the codes are no longer on the page, and only the remaining count is shown

#### Scenario: Disconnecting is not offered under the app requirement

- **WHEN** the instance requires the authenticator app method and a user with an app opens the profile
- **THEN** replacing the app and regenerating recovery codes are offered, and disconnecting is replaced by an explanation that the instance requires an app

### Requirement: Instance settings offer the method of the required second factor

The instance settings SHALL show the switch that requires a second factor and, while it is on, the choice between the emailed code and the authenticator app. While the switch is off, the method choice SHALL be hidden and saving SHALL submit the requirement as off. When an administrator turns the switch on, the emailed code method SHALL be preselected if the instance says its mail server is configured, and the authenticator app method otherwise. When the screen opens with the requirement on, it SHALL show the required method.

Both methods SHALL always be selectable. While the emailed code method is chosen and the instance says its mail server is not configured, the screen SHALL explain that the method needs a mail server, with a link that moves to the mail server section of the same screen, and SHALL NOT let the settings be saved unless the instance already requires the emailed code. The authenticator app method SHALL name a few common apps as examples. Saving SHALL submit the switch and the method together. When saving requires the authenticator app method and the signed-in administrator has no connected app, the interface SHALL open their profile with the app connection already started.

#### Scenario: An instance without a mail server

- **WHEN** an administrator of an instance without a mail server turns the switch on
- **THEN** the authenticator app method is preselected, no mail server explanation is shown, and saving succeeds

#### Scenario: The administrator who requires the app has none

- **WHEN** an administrator without a connected app saves the authenticator app method
- **THEN** the profile opens with the dialog that connects an app

#### Scenario: The administrator who requires the app already has one

- **WHEN** an administrator with a connected app saves the authenticator app method
- **THEN** the instance settings stay open

#### Scenario: Choosing the emailed code without a mail server

- **WHEN** an administrator of an instance without a mail server chooses the emailed code method
- **THEN** the screen explains that the method needs a mail server, links to the mail server section, and does not let the settings be saved until the authenticator app method is chosen again

#### Scenario: A saved emailed code requirement after the mail server is gone

- **WHEN** the instance requires the emailed code, its mail server is gone, and an administrator changes another setting
- **THEN** the screen explains that the method needs a mail server and still saves the settings

#### Scenario: An instance with a mail server

- **WHEN** an administrator of an instance with a mail server turns the switch on
- **THEN** the emailed code method is preselected, and the authenticator app method can be chosen instead

#### Scenario: Turning the switch off

- **WHEN** an administrator turns the switch off and saves
- **THEN** the method choice is hidden and the instance answers that no second factor is required

### Requirement: User administration offers resetting a user's authenticator app

The user administration screen SHALL show whether each user has a connected authenticator app. For each user with an app it SHALL offer resetting the app, after a confirmation, to every administrator who sees the list, as it offers deactivation. When the instance refuses the reset, the screen SHALL show the translated refusal and leave the user's state unchanged.

#### Scenario: Resetting a member's app

- **WHEN** an administrator resets a member's app from the user administration screen and confirms
- **THEN** the screen shows that the member has no connected app

#### Scenario: A refused reset

- **WHEN** an administrator who is not root admin resets another administrator's app and the instance refuses
- **THEN** the screen shows the refusal and still shows that the administrator has an app

### Requirement: Confirmations ask for the app code when root admin has an app

The master-key export and the recovery apply confirmations SHALL ask root admin for a current app code or a recovery code, without a send action, when the instance answers that the app code is expected. They SHALL keep the emailed code flow, with its send action and sending error, otherwise.

#### Scenario: Master-key onboarding with an app

- **WHEN** root admin with a connected app reaches master-key onboarding on an instance that requires two-factor sign-in
- **THEN** the step asks for the app code, sends no mail, and obtains the key after a correct code

#### Scenario: Recovery apply with an app

- **WHEN** root admin with a connected app confirms replacing the installation with a prepared recovery on an instance that requires two-factor sign-in
- **THEN** the confirmation asks for an app code or a recovery code without a send action, and apply starts after a correct code
