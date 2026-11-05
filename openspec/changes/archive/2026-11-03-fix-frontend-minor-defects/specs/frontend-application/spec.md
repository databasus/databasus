## ADDED Requirements

### Requirement: The shell identifies the current section

The main navigation SHALL distinguish the selected section with a solid background. While the pointer is not over it, its icon and any visible label SHALL keep at least 4.5:1 contrast against that background, in the light and dark themes, on wide and narrow viewports. Assistive technology SHALL announce the selected section as the current page, and SHALL NOT read its icon separately from its name.

#### Scenario: A section is selected on a wide viewport

- **WHEN** a user opens a section in either theme on a wide viewport
- **THEN** its navigation button has the solid selected background and the light icon
- **AND** it is the only navigation button exposed as the current page

#### Scenario: A section is selected in the narrow menu

- **WHEN** a user opens the menu in either theme on a narrow viewport
- **THEN** the selected section has the solid background with a light icon and a light label
- **AND** it is the only menu entry exposed as the current page

#### Scenario: The user moves to another section

- **WHEN** a user activates another section
- **THEN** the selected presentation and the current-page state move to that section and leave the previous one

### Requirement: The new-version link explains itself

On wide viewports, where the shell shows the running version, the shell SHALL show a small status dot that links to the latest release when a newer release exists. The link SHALL explain that a new version is available in a translated tooltip on pointer hover and keyboard focus, and SHALL have the same text as its accessible name.

#### Scenario: A newer release exists

- **WHEN** the latest release is newer than the running version and the user hovers over or focuses the new-version link on a wide viewport
- **THEN** the translated new-version explanation appears and the link leads to the latest release

#### Scenario: No newer release exists

- **WHEN** the running version is the latest release, the application runs as a development build, or the release lookup fails
- **THEN** no new-version link is shown
