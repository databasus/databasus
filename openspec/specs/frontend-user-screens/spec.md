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

- **WHEN** the password step returns a pending sign-in
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
