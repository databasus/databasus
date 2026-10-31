# Frontend Localization Specification

## Purpose

Lets a user read the Databasus interface in their own language, and gives the project a way to keep every language complete as the product changes.

## Requirements

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

### Requirement: The language control sits beside the theme control

The language control SHALL be presented together with the light/dark theme control as a single paired element, and SHALL appear everywhere that element appears: the main navigation bar, the narrow-viewport sidebar, and the authentication screen. A user SHALL be able to choose a language before signing in.

The control SHALL show which language is currently active, and SHALL list all six by their native names rather than by language code or flag.

#### Scenario: Choosing a language before signing in

- **WHEN** an unauthenticated user opens the authentication screen
- **THEN** the language control is visible there
- **AND** selecting a language translates the authentication screen immediately and keeps that language after signing in

#### Scenario: Opening the language list

- **WHEN** a user moves the pointer onto the language control
- **THEN** the list of six languages opens without requiring a click
- **AND** moving the pointer from the control to the list does not close it

#### Scenario: Narrow viewport

- **WHEN** the viewport is too narrow for the main navigation bar
- **THEN** the paired language and theme control remains reachable in the sidebar

### Requirement: The language choice defaults to the browser's preference and persists

On a first visit with no stored choice, the frontend SHALL take the browser's ordered list of preferred languages and use the first entry that matches a supported language, ignoring any region subtag, and English when none matches. Once a user makes a choice, that choice SHALL be stored in the browser and applied on subsequent visits.

#### Scenario: First visit with Spanish preferred

- **WHEN** a user whose browser reports Spanish as its preferred language opens the application for the first time
- **THEN** the interface renders in Spanish

#### Scenario: The first preference is unsupported

- **WHEN** a user whose browser reports German first and Russian second opens the application for the first time
- **THEN** the interface renders in Russian rather than English, because Russian is the highest-ranked supported preference

#### Scenario: No preference is supported

- **WHEN** a user whose browser reports no supported language opens the application for the first time
- **THEN** the interface renders in English

#### Scenario: A regional variant with no dictionary of its own

- **WHEN** a user whose browser reports Taiwanese Chinese or European Portuguese opens the application for the first time
- **THEN** the interface renders in the single Chinese or Portuguese variant the product ships, rather than falling through to English

#### Scenario: An explicit choice outranks the browser preference

- **WHEN** a user whose browser prefers Spanish selects English
- **THEN** the interface renders in English on this and every later visit from the same browser

#### Scenario: The choice survives a reload

- **WHEN** a user has selected a language and then reloads the page or returns in a new session on the same browser
- **THEN** the interface renders in the previously selected language
- **AND** the first paint is already in that language, with no visible flash of English

### Requirement: Library and framework text follows the selected language

Text produced by the component library and by date formatting SHALL follow the selected language, not the browser default and not English.

#### Scenario: Component library strings

- **WHEN** the interface renders in a non-English language
- **THEN** table pagination, empty-table placeholders, date pickers, select dropdowns and confirmation popovers display their built-in text in that language

#### Scenario: Dates and times

- **WHEN** the interface renders in a non-English language
- **THEN** month names, weekday names and relative times render in that language

### Requirement: Numbers are grouped by the selected language

Numeric values displayed to the user SHALL be grouped and separated according to the selected language rather than the browser's own locale, using each language's standard digit grouping. Unit words attached to a number stay English. Abbreviated forms that read as prose, such as the Chinese 万 form the website uses for millions in running text, are not used in the interface: counts and sizes appear in full.

#### Scenario: A browser locale that differs from the selected language

- **WHEN** a user whose browser is English selects French, and a screen shows a row count of one million eight hundred thousand
- **THEN** the number is grouped in the French convention rather than the English one
- **AND** it does not disagree with the dates on the same screen, which already follow the selected language

#### Scenario: Clock and calendar conventions are regional, not linguistic

- **WHEN** a user selects a language whose speakers commonly use a 24-hour clock, while their browser reports a region that uses a 12-hour clock
- **THEN** times continue to display in the 12-hour form their browser reports
- **AND** the words around those times, including month and weekday names, are still in the selected language

#### Scenario: A size readout

- **WHEN** a backup size is displayed in a language that groups digits differently from English
- **THEN** the digits are grouped for that language
- **AND** the unit remains `MB` or `GB`

### Requirement: The document reports the selected language

The page SHALL declare the selected language to the browser, and its title SHALL be translated.

#### Scenario: Assistive technology and browser translation

- **WHEN** the interface renders in a given language
- **THEN** the document's language attribute reports that language
- **AND** the browser does not offer to translate a page that is already in the reader's language

#### Scenario: The browser tab

- **WHEN** the interface renders in a non-English language
- **THEN** the browser tab and any bookmark made from it show a title in that language

### Requirement: Website links open in the selected language

When the interface links to a page on the product website, the link SHALL open that page in the selected language if the website publishes it in that language, and the English page otherwise. A link to a section of a page SHALL land on that section in either case.

#### Scenario: The website has the page in the selected language

- **WHEN** a user with Russian selected follows a link to the installation guide
- **THEN** the Russian version of the installation guide opens

#### Scenario: The website has the page only in English

- **WHEN** a user with Russian selected follows a link to a page the website publishes only in English
- **THEN** the English page opens rather than a missing page

#### Scenario: A link to a section

- **WHEN** a user with Chinese selected follows a link to a section of a translated page
- **THEN** the Chinese page opens scrolled to that section

### Requirement: Status labels are translated while wire values stay stable

Status values exchanged with the backend SHALL remain unchanged English identifiers. Their displayed labels SHALL be translated.

#### Scenario: A backup status is displayed

- **WHEN** the backend reports a backup status of `COMPLETED` and the interface renders in Spanish
- **THEN** the user sees the Spanish label for a completed backup
- **AND** the value sent back to the backend in any subsequent request is still `COMPLETED`

#### Scenario: A new status value is introduced

- **WHEN** a new value is added to a status enum and no label is supplied for it
- **THEN** the type check fails

### Requirement: Dictionary completeness is enforced by the type checker

Each non-English dictionary SHALL be checked against the English dictionary at build time. English is the source of truth for the set of keys; the wording of each language is written independently and is not a literal rendering of the English text.

#### Scenario: A key is missing from a translation

- **WHEN** a key exists in the English dictionary and is absent from any other language
- **THEN** the type check fails and names the missing key

#### Scenario: A key is left over in a translation

- **WHEN** a key is removed from the English dictionary but remains in another language
- **THEN** the type check fails and names the stray key

#### Scenario: A lookup uses an unknown key

- **WHEN** code requests a key that does not exist in the English dictionary
- **THEN** the type check fails

### Requirement: New code cannot add hardcoded strings

The lint job SHALL reject a user-facing string literal added anywhere in the frontend source. Technical strings and test files are exempt.

#### Scenario: A hardcoded string is added

- **WHEN** a contributor adds a literal label, placeholder or message to any frontend source file
- **THEN** the lint job fails and points at the literal

#### Scenario: A technical string is added

- **WHEN** a contributor adds a command block, connection string or engine name to a frontend source file
- **THEN** the lint job accepts it, either through the configured exclusions or through an inline suppression carrying a reason

### Requirement: Translated text and inserted values render as plain text

Every translated message SHALL reach the page as text, never as markup. A value inserted into a translated sentence, such as a workspace, user or database name, SHALL be shown as text and SHALL never become markup, even when it contains characters that look like HTML. The only markup in a translated sentence is the formatting the interface itself supplies, such as code spans, bold text and links.

#### Scenario: A name that looks like markup

- **WHEN** a workspace is named `<img src=x onerror=alert(1)>` or `<strong>bold</strong>` and a translated sentence mentions it, including a sentence that carries the interface's own formatting
- **THEN** the sentence shows that name literally, as text
- **AND** no element is created from it and no script runs

#### Scenario: A translation that contains markup-like text

- **WHEN** a translated message contains angle brackets that are not part of the interface's own formatting
- **THEN** they are shown as characters rather than interpreted as elements

### Requirement: Backend errors are translated by code without a free-text fallback

The frontend SHALL translate an error response by its stable error code when that code has a translation. When the code is absent or unknown, it SHALL show a general error message in the selected language. It MUST NOT use the response's message, title, detail, raw code or request address as fallback copy. This change does not add backend codes or change the HTTP contract.

#### Scenario: The backend supplies a known error code

- **WHEN** a request fails with an error code the frontend has a translation for
- **THEN** the user sees the translated message in the selected language

#### Scenario: The backend supplies no code

- **WHEN** a request fails with only a free-text message
- **THEN** the user sees a general translated error rather than that message

#### Scenario: The backend supplies an unrecognized code

- **WHEN** a request fails with an error code the frontend has no translation for and a nonempty message
- **THEN** the user sees a general translated error rather than the message or raw code

#### Scenario: The backend supplies an unrecognized code and no message

- **WHEN** a request fails with an error code the frontend has no translation for and no message
- **THEN** the user sees a general error message in the selected language rather than the raw code

#### Scenario: The request fails without an answer from the backend

- **WHEN** the server cannot be reached, or answers with an error status but no usable error description
- **THEN** the user sees a message in the selected language rather than a request address or a status line

### Requirement: Every error code in the contract has a translation

For every error code the API contract declares, every interface dictionary SHALL hold a translated message. A contract code without a translation SHALL fail the frontend type check, which names the missing code, so a new code cannot reach users as raw text. A failed call to a contract operation SHALL reach the user through the same translation by code as every other backend error.

#### Scenario: A code is added to the contract

- **WHEN** a developer adds an error code to the contract, regenerates the frontend types and adds no translation
- **THEN** the frontend checks fail and name the missing code

#### Scenario: A contract operation fails

- **WHEN** a call to a contract operation answers `Problem` with `code` `internal`
- **THEN** the user sees the translated message for that code in the selected language

### Requirement: Technical error diagnostics require an explicitly supported code

An error's technical detail SHALL be displayed only when its known code explicitly allows diagnostic text under the API contract. Connection failure reasons for `database_connection_failed` and required/free disk bytes for `insufficient_disk_space` SHALL remain available as plain text with translated surrounding copy. Unrecognized errors MUST NOT expose technical detail through that mechanism.

#### Scenario: Connection diagnostics are useful

- **WHEN** a request fails with `database_connection_failed` and a connection reason
- **THEN** the user sees the translated connection-failure message and that reason rendered as text

#### Scenario: Disk diagnostics are useful

- **WHEN** a request fails with `insufficient_disk_space` and the required and free byte counts
- **THEN** the user sees the translated capacity message and those diagnostic counts

#### Scenario: An unknown code supplies diagnostic-looking text

- **WHEN** an unknown code includes a nonempty detail, including markup-like text
- **THEN** the user sees only the general translated error and no part of that detail

#### Scenario: A code resembles an inherited property

- **WHEN** an unrecognized code matches a built-in object-property name
- **THEN** it still selects the general translated error and exposes no server text

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
