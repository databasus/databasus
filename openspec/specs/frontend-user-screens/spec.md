# frontend-user-screens Specification

## Purpose

Preserve existing account workflows while keeping editable values, user-management results and audit views consistent when requests overlap or presentation changes.

## Requirements

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

### Requirement: Saving profile or instance settings acknowledges only submitted values

Profile information and editable instance settings SHALL use the last acknowledged values to determine whether changes can be saved. Returning values to that baseline SHALL remove save eligibility. A pending save SHALL prevent duplicate submission while allowing newer input to remain distinct. Success SHALL acknowledge the submitted values, incorporating server normalization without replacing newer edits. Another save SHALL be available only when the resulting draft differs from the acknowledged response. Failure SHALL preserve the draft for correction. Accepted changes SHALL update relevant account-wide labels and capabilities. Obsolete completions after navigation or session replacement SHALL not publish feedback or completion effects.

#### Scenario: A draft changes while saving

- **WHEN** a profile or instance-settings save is pending and the user makes a newer edit
- **THEN** successful completion preserves that edit and acknowledges only the earlier submission
- **AND** the newer draft remains eligible for a subsequent save only if it differs from the acknowledged response

#### Scenario: Newer input matches the acknowledged response

- **WHEN** a profile edit made during saving equals the server-normalized response, or a settings toggle returns to its submitted value and that value is acknowledged
- **THEN** successful completion leaves the resulting draft equal to the acknowledged baseline
- **AND** the form is clean and does not offer another save

#### Scenario: A save fails

- **WHEN** the current profile or settings save is refused
- **THEN** saving ends, translated feedback is shown and the draft remains available for correction

#### Scenario: A user restores acknowledged values

- **WHEN** all editable values equal the most recently acknowledged baseline
- **THEN** the screen does not offer a redundant save

#### Scenario: The saved profile changes its displayed name

- **WHEN** the current profile update succeeds
- **THEN** account-wide labels use the acknowledged name without resetting newer form input

#### Scenario: Instance policy changes successfully

- **WHEN** an administrator saves an instance setting affecting available actions
- **THEN** the application uses the acknowledged policy for those actions without requiring a reload

### Requirement: User-management responses belong to the active query and action

User management SHALL retain its existing search, cursor pagination, per-user pending actions, role/status operations and audit drawer. A new query SHALL invalidate previous rows, cursor and in-flight work. Superseded responses SHALL not change current rows, loading indicators or feedback. Paging SHALL retain the current query and page size and SHALL not append duplicate users. Mutation pending state SHALL prevent duplicate activation for the same user and release correctly on success or failure. Failed actions SHALL not imply a successful role or status change.

#### Scenario: An earlier search finishes last

- **WHEN** an older user search finishes after the current query has loaded
- **THEN** the current query's rows and cursor remain unchanged and no obsolete error appears

#### Scenario: Scrolling overlaps a new search

- **WHEN** a new search is pending and the user scrolls
- **THEN** no next-page request uses the previous query or cursor

#### Scenario: Loading users fails

- **WHEN** an initial or filtered user-list request fails
- **THEN** translated feedback offers an explicit retry without displaying a successful empty result
- **AND** retry reloads the first page of the same query while preserving the search input

#### Scenario: A role or status action is refused

- **WHEN** a user's role or status operation fails
- **THEN** its pending state ends, its prior displayed value remains and translated feedback permits a new explicit attempt

### Requirement: Shared audit presentation preserves each view's meaning

Instance, user and workspace audit views SHALL retain their existing access checks, filters, newest-first paging, page sizes, author/workspace visibility, date/time conventions and distinct empty messages. Event content and captured author/workspace names SHALL be displayed literally. Changing the selected user or workspace, or leaving the view, SHALL invalidate its old loads. A common presentation SHALL not broaden which events the caller may fetch. Workspace audit SHALL retain author visibility, no redundant workspace column, short timestamps, its ten-event page size and its own scroll container.

#### Scenario: An administrator opens a user's audit drawer

- **WHEN** an administrator opens one user's audit history and then another's
- **THEN** each view requests that user's events and a late response from the first cannot replace the second
- **AND** paging, column selection and the user-specific empty message retain their meaning

#### Scenario: An event contains technical or user-entered characters

- **WHEN** an event message or captured label contains angle brackets or interpolation-like text
- **THEN** it is displayed literally without executing markup or translating the event's contents

#### Scenario: A user opens a workspace's audit view

- **WHEN** a user opens audit history for the current workspace
- **THEN** only that workspace's authorized events load in the existing order and page size
- **AND** desktop rows and mobile cards retain author visibility, short timestamps and workspace-specific empty copy

#### Scenario: Workspace changes while audit is loading

- **WHEN** an earlier workspace's audit request completes after a workspace switch
- **THEN** its events, errors and paging state cannot change the current view

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
