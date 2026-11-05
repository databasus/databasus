## MODIFIED Requirements

### Requirement: The interface renders in the selected language

The frontend SHALL render all user-facing copy in one of seven languages: English, Russian, Spanish, Portuguese, Chinese, French and Indonesian. Copy includes labels, placeholders, button text, headings, table columns, empty states, confirmation dialogs, tooltips and validation messages.

Technical strings are excluded and stay identical in every language: command lines and code blocks, connection strings, cron expressions, database engine and tool names, file paths, and duration readouts such as `2h 15m 3s` in full, and the unit words in size readouts such as the `GB` in `1.4 GB`. The digits in a size or a count are not excluded; see the number formatting requirement below.

A transient success notice already displayed when the user changes language MAY retain its creation language until its existing bounded lifetime ends. This exception SHALL NOT extend that lifetime or apply to persistent feedback, validation errors or operation failures. Notices created after the switch SHALL use the selected language.

#### Scenario: A user switches language

- **WHEN** a user selects Russian
- **THEN** every visible label, button and message re-renders in Russian without a page reload, except a transient success notice already displayed before the switch
- **AND** an already displayed transient success notice may retain its creation language only until its existing bounded lifetime ends
- **AND** command blocks, connection strings and engine names remain byte-identical to what English showed

#### Scenario: Switching language keeps what is on screen

- **WHEN** a user switches the language while a form on the page holds unsaved input, or from the language control inside the open narrow-viewport sidebar
- **THEN** the form keeps its input and the sidebar stays open, both now labelled in the new language

#### Scenario: A transient success notice spans a language switch

- **WHEN** a success notice is already visible and the user selects another language
- **THEN** that notice may retain its creation language until it expires without restarting its lifetime
- **AND** persistent feedback and failures use the new language immediately
- **AND** the next success notice is created in the new language

#### Scenario: A user selects Indonesian

- **WHEN** a user selects Indonesian
- **THEN** the interface renders current product copy in Indonesian
- **AND** technical strings and values exchanged with the backend remain unchanged

### Requirement: The language control sits beside the theme control

The language control SHALL be presented together with the light/dark theme control as a single paired element, and SHALL appear everywhere that element appears: the main navigation bar, the narrow-viewport sidebar, and the authentication screen. A user SHALL be able to choose a language before signing in.

The control SHALL show which language is currently active, and SHALL list all seven by their native names rather than by language code or flag. Indonesian SHALL appear as Bahasa Indonesia.

#### Scenario: Choosing a language before signing in

- **WHEN** an unauthenticated user opens the authentication screen
- **THEN** the language control is visible there
- **AND** selecting a language translates the authentication screen immediately and keeps that language after signing in

#### Scenario: Opening the language list

- **WHEN** a user moves the pointer onto the language control
- **THEN** the list of seven languages opens without requiring a click
- **AND** moving the pointer from the control to the list does not close it

#### Scenario: Narrow viewport

- **WHEN** the viewport is too narrow for the main navigation bar
- **THEN** the paired language and theme control remains reachable in the sidebar
