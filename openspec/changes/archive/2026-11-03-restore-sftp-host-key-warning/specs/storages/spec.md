## ADDED Requirements

### Requirement: Skipping SFTP host key verification has a visible security warning

Whenever the SFTP host key verification option is visible in the storage editor, the interface SHALL show adjacent explanatory text warning that skipping verification reduces security. The warning SHALL be visible without hovering, focusing or enabling the option, SHALL be available as the checkbox's accessible description and SHALL use the selected interface language in each of the six supported languages.

#### Scenario: Reviewing the option before enabling it

- **WHEN** a user opens the SFTP editor with host key verification enabled and expands the advanced settings
- **THEN** the unchecked option to skip verification has a visible warning that skipping verification reduces security
- **AND** assistive technology can read that warning as the checkbox's description

#### Scenario: Editing settings that already skip verification

- **WHEN** a user opens the SFTP editor with the option to skip host key verification already enabled
- **THEN** the advanced settings show the checked option and its security warning immediately

#### Scenario: Displaying the warning in each supported language

- **WHEN** a user views the SFTP host key verification option in any of the six supported interface languages
- **THEN** the visible warning and the checkbox's accessible description use that language
