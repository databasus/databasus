## ADDED Requirements

### Requirement: Application controls expose names and keyboard interactions

Every action and field on the database screens, the storage authentication choice, the verification agent name field and the logo links of the application and the sign-in page SHALL have an accessible name, keyboard activation and visible focus. On the database screens this SHALL cover icon-only actions, edit icons, tabs, the selection cards of the creation wizard and fields without a connected visible label. Tabs SHALL expose which one is selected, selection cards SHALL expose which one is chosen, and a disclosure SHALL expose whether its content is expanded. Decorative images SHALL be hidden from assistive technology, and an image that carries meaning SHALL have a text alternative.

#### Scenario: A keyboard user changes the database tab

- **WHEN** a user moves focus to the tabs of a database and activates another tab from the keyboard
- **THEN** that tab becomes selected, its content appears and the shown database stays the same

#### Scenario: A screen reader reaches the database actions

- **WHEN** assistive technology reads the transfer, copy and delete actions of a database and the close action of its name editor
- **THEN** each action announces its name

#### Scenario: A keyboard user sees focus on an edit icon

- **WHEN** a user tabs to the edit icon of a settings section
- **THEN** a focus indicator is visible, the icon announces its name, and the keyboard opens that section

#### Scenario: A keyboard user chooses a database engine

- **WHEN** a user activates an engine card in the creation wizard from the keyboard
- **THEN** the card is chosen, exposes that state and announces the engine name once

#### Scenario: A keyboard user chooses a backup type

- **WHEN** a user activates a backup type card in the creation wizard from the keyboard
- **THEN** the card is chosen and exposes that state

#### Scenario: A field has no connected label

- **WHEN** assistive technology looks for a filter field, such as the backup status selection
- **THEN** it finds the field by a translated name

#### Scenario: A logo stands beside its name

- **WHEN** assistive technology reads a notifier name with its logo beside it
- **THEN** it announces the name once and does not announce the logo

#### Scenario: A logo links to the website

- **WHEN** assistive technology reaches the logo link of the application or the sign-in page
- **THEN** the link announces a translated name

#### Scenario: A disclosure expands

- **WHEN** a user activates the privileges disclosure of the limited user form from the keyboard
- **THEN** its full content appears and it exposes that it is expanded

#### Scenario: The language changes while a name is being edited

- **WHEN** the language changes while the database name editor holds unsaved text
- **THEN** the transfer, copy, delete and close actions announce names in the selected language and the typed name stays
