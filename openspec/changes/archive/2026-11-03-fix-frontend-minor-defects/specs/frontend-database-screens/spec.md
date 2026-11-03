## ADDED Requirements

### Requirement: A language change keeps database drafts and restore commands

Changing the interface language SHALL keep the unsaved input of the physical backup config and the verification config, and SHALL keep the restore commands shown on the physical restore screen. It SHALL NOT start a new restore token request. A repeated request would replace the command on screen after the user may have copied it, would leave an extra live token unused until it expires, and would be refused while the user's restore stream runs, which would replace the commands with an error. Labels around the kept values SHALL follow the selected language.

#### Scenario: Language changes with a physical backup config draft

- **WHEN** a user changes the language while the physical backup config holds an unsaved edit
- **THEN** the edit remains, no save request is sent, and the form labels use the selected language

#### Scenario: Language changes with a verification config draft

- **WHEN** a user changes the language while the verification config holds an unsaved edit
- **THEN** the edit remains, no save request is sent, and the form labels use the selected language

#### Scenario: Language changes with restore commands on screen

- **WHEN** a user changes the language while the physical restore screen shows restore commands
- **THEN** the commands and their token stay the same, no new token is requested and the titles around them use the selected language
