## MODIFIED Requirements

### Requirement: The interface renders in the selected language

The frontend SHALL render all user-facing copy in one of six languages: English, Russian, Spanish, Portuguese, Chinese and French. Copy includes labels, placeholders, button text, headings, table columns, empty states, confirmation dialogs, tooltips and validation messages.

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

## ADDED Requirements

### Requirement: Active notifier-form feedback follows language changes

Validation and operation errors still displayed in the notifier form SHALL re-render in the selected language without resetting the draft, expanded sections or pending operations. User-entered values and explicitly supported technical diagnostics SHALL remain literal text. Already displayed transient success notices MAY retain the language used when they were created until their bounded lifetime ends; later notices SHALL use the current language.

#### Scenario: A user changes language after validation fails

- **WHEN** the notifier form shows a validation error and the user selects another language
- **THEN** the error and field labels change language while field values and expanded sections remain unchanged

#### Scenario: A request fails after a language switch

- **WHEN** a settings test starts in one language, the user switches languages and the active request then fails
- **THEN** the resulting error uses the newly selected language
- **AND** the switch sends no additional test request

#### Scenario: A diagnostic contains markup-like or interpolation-like text

- **WHEN** a supported test diagnostic or user-entered value contains angle brackets or placeholder syntax
- **THEN** it is displayed literally within the translated message and creates no markup or substituted value

### Requirement: The language menu supports keyboard selection

The paired language control SHALL expose an accessible name and support opening, selection and dismissal from the keyboard, including when its visible presentation is compact. Existing pointer-hover and touch/click opening behavior and the six native language names SHALL remain available.

#### Scenario: A user changes language without a pointer

- **WHEN** a user focuses the language control, opens its menu and selects a language from the keyboard
- **THEN** the interface changes language and the selected language is indicated
- **AND** closing the menu leaves focus on the control
