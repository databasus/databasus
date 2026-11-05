## ADDED Requirements

### Requirement: A failed refresh gives no feedback

A failed first-page refresh of a polled list SHALL keep the shown rows, their cursor and their loading state, and SHALL show no blocking alert, message or error. This SHALL also hold for a refresh that follows a successful action, such as a cancel, because the action already succeeded and the next refresh updates the row. The next refresh SHALL run on its own schedule. A failed first page or next page SHALL still be reported, once per failure, and the report SHALL identify which of the two failed. A failed first page SHALL stay available to the screen until an explicit reload clears it.

#### Scenario: A refresh fails

- **WHEN** a polled list is shown, possibly with a next page pending, and its refresh fails
- **THEN** the rows stay unchanged, no error is reported, the cursor still loads the next page, a pending next page keeps its loading indicator and the next refresh still runs

#### Scenario: A refresh after a successful action fails

- **WHEN** a cancel succeeds and the refresh that follows it fails
- **THEN** no error is reported and the next refresh still runs

#### Scenario: The first page fails

- **WHEN** the first page of a list fails
- **THEN** loading ends, the list has no rows, the failure is reported once as a first-page failure and stays available to the screen, and an explicit reload clears it and starts a fresh request

#### Scenario: A next page fails

- **WHEN** the next page of a list fails
- **THEN** the failure is reported once as a next-page failure, the loaded rows and the cursor stay, and a later scroll requests that page again
