## Purpose

Give users consistent, readable controls across themes and languages, and establish accessible shared interactions before the remaining interface areas are migrated.

## ADDED Requirements

### Requirement: Shared presentation follows the active theme

Shared controls and the notifier form SHALL use consistent roles for surfaces, text, borders, focus and operation status in light and dark modes. System mode SHALL follow the operating system preference. Scrollbars SHALL have defined track, thumb and hover colors in both modes. A status SHALL remain understandable through text or another non-color cue.

#### Scenario: A theme changes while editing

- **WHEN** a user changes between light, dark and system modes with an unsaved notifier form open
- **THEN** the form and its shared controls adopt the resolved theme without losing input or restarting an operation
- **AND** visible status messages remain understandable without distinguishing colors

#### Scenario: A scrollable surface is displayed

- **WHEN** a shared scrollable surface is shown in either theme
- **THEN** its scrollbar uses defined colors appropriate to that surface

### Requirement: Shared controls expose names and keyboard interactions

Shared fields SHALL associate visible labels with their controls and associate applicable help and error text with the field. Shared disclosure and language/theme controls SHALL have accessible names, keyboard activation and visible focus. Disclosures SHALL expose whether their content is expanded. Shared confirmation dialogs SHALL identify the action, keep keyboard focus inside while open and return focus to an available invoking control when closed. Destructive styling SHALL belong to the destructive action rather than its ordinary cancellation.

#### Scenario: A user opens additional settings from the keyboard

- **WHEN** a user focuses a shared disclosure and activates it with the keyboard
- **THEN** its content opens, its expanded state is exposed and its focus remains visible

#### Scenario: A field has an error

- **WHEN** the notifier form displays a field error
- **THEN** the field can be found by its visible label and its error is associated with it

#### Scenario: A user cancels a destructive confirmation

- **WHEN** a user opens a shared destructive confirmation, navigates its controls and presses Escape
- **THEN** focus stays in the dialog until it closes and then returns to the available invoking control
- **AND** the ordinary cancel action is not presented as destructive

### Requirement: The notifier form supports narrow layouts and text enlargement

The notifier form SHALL keep its fields, help, errors and actions reachable at a 320 CSS-pixel viewport width and with text enlarged to 200 percent. Long translated labels and actions SHALL wrap without covering controls. The application SHALL allow browser zoom. The pilot SHALL have an explicit owner of vertical scrolling, and technical text requiring horizontal scrolling SHALL remain inside its own region.

#### Scenario: A long translation is displayed on a narrow screen

- **WHEN** a user opens the notifier form at a 320 CSS-pixel width with a long translated label or error
- **THEN** the user can read the text, reach every action and scroll through the form without page-wide horizontal overflow

#### Scenario: A user enlarges text

- **WHEN** notifier-form text is enlarged to 200 percent
- **THEN** labels, help and actions remain available without clipping or overlap

### Requirement: Interface typography works without an external font service

The interface SHALL supply its selected Latin and Cyrillic typeface with the application and SHALL render Chinese through an explicit fallback. A running installation SHALL require no external font stylesheet or font-service request. Commands and inline technical text SHALL retain their technical characters and readable monospace presentation.

#### Scenario: External font access is unavailable

- **WHEN** an installation is opened with external font services unavailable
- **THEN** local interface fonts load and all six interface languages remain readable
- **AND** no request is sent to an external font service

### Requirement: Pilot feedback follows the application context

Notifier-form feedback SHALL follow the active theme. Persistent feedback SHALL follow the selected interface language, and transient success notices SHALL follow the language-switching lifetime policy. Field errors SHALL remain beside their fields; failed operations SHALL leave the draft available for correction. Success messages SHALL have a bounded lifetime, and closing or reopening the editor SHALL not leave feedback from a previous editing session attached to the new one.

#### Scenario: A test operation fails

- **WHEN** testing the active notifier settings fails
- **THEN** the editor shows themed feedback and preserves the draft for correction
- **AND** the user can continue without dismissing a browser-native alert
