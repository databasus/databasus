## Purpose

Make frontend screen tests fail on undeclared network activity and run with realistic, isolated language, theme and routing context.

## ADDED Requirements

### Requirement: Screen tests declare finite network dependencies

A screen test SHALL declare the HTTP method and finite pathname pattern of each allowed request. Named path parameters and an origin wildcard SHALL be permitted; all-method handlers and pathname catch-alls SHALL be rejected, including handlers that never respond. Responses SHALL be checked against the operation's declared status and body types, with explicit facilities for empty successes, errors, network failures, malformed external responses and controlled delays. Tests SHALL be able to inspect outgoing path parameters, query parameters, headers and bodies.

#### Scenario: A test declares an empty success

- **WHEN** a declared operation succeeds without a body
- **THEN** the mock returns its successful status with no body
- **AND** assigning an incompatible response body fails type checking

#### Scenario: A test needs to inspect a mutation

- **WHEN** a screen submits a declared mutation
- **THEN** the test can inspect the request and assert its parameters and body without replacing the domain API

#### Scenario: A blanket pending handler is registered

- **WHEN** a test registers an all-method handler or a pathname catch-all, including one that leaves every request pending
- **THEN** the test fails with a diagnostic identifying the invalid declaration

#### Scenario: A delayed request is intentional

- **WHEN** a test declares a specific request and controls when its response is released
- **THEN** that request can remain pending while the test exercises an overlapping action
- **AND** teardown releases or cancels the pending work

### Requirement: Unexpected network activity fails independently of application error handling

An undeclared request SHALL fail its screen test even if the screen catches the resulting network rejection or a response handler would otherwise hide the request. Failure reporting SHALL identify the request's method and address. Unexpected-request accounting SHALL be independent of the assertions made by the screen test itself.

#### Scenario: The screen catches an undeclared request's rejection

- **WHEN** a screen makes an undeclared request and catches its rejection before rendering an ordinary fallback
- **THEN** the test still fails and identifies the undeclared request

#### Scenario: The screen makes only its declared requests

- **WHEN** a screen completes the scenario using only its finite declared dependencies
- **THEN** network accounting does not fail the test

### Requirement: The renderer supplies configurable real screen context

Screen tests SHALL run with the actual language and theme providers, library configuration, notification context and router needed by the screen. A test SHALL be able to select its initial language, theme and router entry without running unrelated full-application startup work. Pure logic tests SHALL remain runnable without a document environment.

#### Scenario: A screen runs in a selected context

- **WHEN** a test selects a non-default supported language, a dark theme and a router entry
- **THEN** the screen, built-in controls and notifications observe that language and theme
- **AND** navigation observes that router entry

#### Scenario: Language changes with an unsaved form

- **WHEN** a screen test changes the language after entering unsaved input
- **THEN** visible copy updates and the input remains

### Requirement: Screen-test state does not leak into the next test

Teardown SHALL unmount the screen, remove the test's handlers and recorded requests, settle or cancel its pending work, and restore the storage, document state, global overrides, locale state, timers and subscriptions it changed. A failed scenario SHALL receive the same cleanup. Harness verification SHALL exercise real test-runner success and failure outcomes, rather than only asserting that a teardown callback was called.

#### Scenario: A scenario changes providers and leaves a request pending

- **WHEN** one test changes the language and theme, installs a handler and leaves a declared request pending before it finishes
- **THEN** its screen and pending work are cleaned up
- **AND** the next test starts with its own configured context, no previous handlers and no previous request records

#### Scenario: A failing harness fixture is verified

- **WHEN** a fixture deliberately makes a caught undeclared request or registers a blanket handler
- **THEN** its test-runner process exits unsuccessfully for that specific reason
- **AND** a valid control fixture succeeds
