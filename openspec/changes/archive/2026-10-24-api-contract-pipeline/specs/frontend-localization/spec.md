## ADDED Requirements

### Requirement: Every error code in the contract has a translation

For every error code the API contract declares, every interface dictionary SHALL hold a translated message. A contract code without a translation SHALL fail the frontend type check, which names the missing code, so a new code cannot reach users as raw text. A failed call to a contract operation SHALL reach the user through the same translation by code as every other backend error.

#### Scenario: A code is added to the contract

- **WHEN** a developer adds an error code to the contract, regenerates the frontend types and adds no translation
- **THEN** the frontend checks fail and name the missing code

#### Scenario: A contract operation fails

- **WHEN** a call to a contract operation answers `Problem` with `code` `internal`
- **THEN** the user sees the translated message for that code in the selected language
