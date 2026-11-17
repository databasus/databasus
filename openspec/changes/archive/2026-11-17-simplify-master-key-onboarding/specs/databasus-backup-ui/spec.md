## MODIFIED Requirements

### Requirement: Initial backup reminders wait until the instance is in use

The initial floating setup reminder SHALL appear only to root admin, only when no destinations are selected, and only after at least one user database or storage exists anywhere in the instance. An empty workspace alone SHALL NOT trigger it. The reminder SHALL link to the backup section and SHALL offer no way to defer or dismiss it: it stays until a destination is selected. It SHALL be suppressed while that section is open and SHALL NOT appear while key onboarding is shown. Eligibility SHALL survive reload.

#### Scenario: Instance has only an empty workspace

- **WHEN** root admin has created a workspace but no user database or storage exists
- **THEN** no initial storage-setup floating reminder appears

#### Scenario: Data exists in another workspace

- **WHEN** a user database or storage exists outside the currently selected workspace and no backup destination is selected
- **THEN** root admin receives the setup reminder regardless of the selected workspace

#### Scenario: The reminder cannot be put off

- **WHEN** root admin sees the setup reminder
- **THEN** it offers only the way to the backup section, with no "remind me later" action and no close button, and it stays on later visits until a destination is selected
