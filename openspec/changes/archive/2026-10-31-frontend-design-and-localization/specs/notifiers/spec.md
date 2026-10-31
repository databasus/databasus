## ADDED Requirements

### Requirement: Notifier editing measures changes against its baseline

The notifier editor SHALL derive whether a draft has changes from its initial or last acknowledged saved values. Returning all editable values to that baseline SHALL remove the save action. Initialization, selecting another notifier and reopening a cancelled editor SHALL establish the appropriate baseline without carrying over the previous editing session. A new notifier SHALL still require valid settings and an explicit successful test before creation.

#### Scenario: A user reverses an edit

- **WHEN** a user changes a saved notifier field and then returns it to its baseline value
- **THEN** the editor no longer offers to save an unchanged draft

#### Scenario: A user reopens an editor

- **WHEN** a user cancels a notifier draft and reopens the saved notifier
- **THEN** the editor starts from the saved values without the cancelled draft or its test result

### Requirement: A notifier settings test authorizes only the active settings

The editor SHALL allow a changed draft to be saved only after an explicitly requested successful test of its active settings in its current workspace and notifier context. Changing delivery settings SHALL invalidate previous test success and any in-flight result for the previous settings. Changing only the display name, interface language or theme SHALL not invalidate an already successful settings test. A late success or failure from an obsolete request SHALL not enable saving or overwrite current feedback. Testing SHALL never be retried automatically.

#### Scenario: Settings change before a successful test returns

- **WHEN** a user starts a settings test, edits the delivery settings and then receives the earlier success
- **THEN** saving remains unavailable until the current settings pass a new explicit test

#### Scenario: A display name changes after a successful test

- **WHEN** the active settings have passed their test and the user changes only the display name
- **THEN** the editor can save the valid changed draft without another test message

#### Scenario: An obsolete request fails

- **WHEN** a test of a previous draft fails after the user switches notifier or editing session
- **THEN** the failure does not alter the current editor or show feedback belonging to the previous session

### Requirement: Save completion preserves newer notifier edits

The editor SHALL acknowledge the values submitted by a successful save while preserving any newer changes entered during that request. After a successful creation, it SHALL retain the returned notifier ID and treat later saves as updates of that notifier, including when newer edits keep the original creation editor open. A completion for an obsolete editing session SHALL not close or replace the active editor or assign its saved ID to that editor. Failed saves SHALL retain the current draft and leave save or retest available as appropriate. Repeated activation while a save is pending SHALL send at most one save request.

The editor SHALL prevent changing provider type while creation is pending, while allowing changes to the name and settings of the submitted type. After successful creation the provider type SHALL remain immutable, including in the still-open editor. If creation fails without a saved notifier ID, the unsaved draft SHALL permit type selection again.

#### Scenario: A user edits while saving

- **WHEN** a user submits a tested draft, edits another value while it is saving and then receives a successful response
- **THEN** the newer edit remains visible and is not marked as saved by that response
- **AND** the editor stays open with the appropriate save or test action for the newer draft

#### Scenario: A creation succeeds while newer edits remain

- **WHEN** a user submits a new tested notifier, edits its name or settings of the submitted type before creation succeeds and then receives the created notifier's ID
- **THEN** the editor retains that ID and the newer edits without treating them as saved or closing
- **AND** after any required explicit test, the next save updates that same ID rather than creating another notifier
- **AND** the sequence creates exactly one notifier

#### Scenario: Creation locks type until the editor becomes saved

- **WHEN** a user submits tested SMTP settings and the creation response is still pending
- **THEN** the editor prevents choosing another provider type while allowing edits to the name and SMTP fields
- **AND** after success with newer edits, the editor retains the returned ID and presents SMTP as an immutable type even though the original creation editor remains open
- **AND** subsequent explicit tests and updates use that ID and SMTP settings, without another creation

#### Scenario: Failed creation unlocks the unsaved type

- **WHEN** the pending SMTP creation fails without a saved notifier ID
- **THEN** the unsaved draft keeps its current values and permits another provider type to be selected
- **AND** changing that type invalidates the previous settings test

#### Scenario: A user cancels newer edits after creation succeeds

- **WHEN** creation succeeds with newer edits still open and the user cancels those newer edits
- **THEN** the created notifier remains available with the values acknowledged by creation
- **AND** cancellation neither deletes it nor creates another notifier

#### Scenario: A save fails

- **WHEN** the active save fails
- **THEN** the current values remain in the editor with localized feedback and the applicable next action

### Requirement: Notifier editing preserves stored-secret omission

The form pilot SHALL preserve the existing distinction between an omitted stored secret and an explicitly supplied replacement. Loading, renaming, changing language/theme or testing a saved notifier SHALL not manufacture a secret value or send a display placeholder as a credential. Diagnostic and browser-test artifacts SHALL contain no real credentials.

#### Scenario: A saved SMTP notifier keeps its password

- **WHEN** a user tests and saves changes to an existing SMTP notifier without entering a replacement password
- **THEN** both requests omit the password and retain the existing credential through the established API behavior

#### Scenario: A user replaces a secret

- **WHEN** a user explicitly supplies a replacement secret and tests those settings
- **THEN** the test and subsequent save use that replacement without substituting a hidden-value placeholder
