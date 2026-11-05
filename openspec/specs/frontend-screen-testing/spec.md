# Frontend Screen Testing Specification

## Purpose

Make frontend screen tests fail on undeclared network activity and run with realistic, isolated language, theme and routing context.

## Requirements

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

### Requirement: Browser UI checks exercise the production build with finite fixtures

The browser UI suite SHALL open the production application build and replace only its HTTP boundary with finite, operation-typed fixtures. The suite SHALL not require a live backend, test-only production routes or substituted application services. Unexpected application requests SHALL fail the scenario even if application error handling catches the failure. Its entry point SHALL return a nonzero exit status when a scenario fails and SHALL clean up only the servers, containers and temporary state it owns on success, failure or interruption. This suite SHALL be distinguished from real-backend end-to-end tests.

#### Scenario: A production-build form is exercised

- **WHEN** the browser UI suite opens notifier editing
- **THEN** it uses the built application's actual controls, providers and domain requests against the declared fixtures

#### Scenario: An undeclared request occurs

- **WHEN** the application makes an undeclared HTTP request during a browser UI scenario
- **THEN** the suite fails with the request method and address, even if the application handles the rejection

#### Scenario: A scenario fails or the run is interrupted

- **WHEN** the browser run fails or is interrupted after starting its own resources
- **THEN** it returns failure and removes its own processes and temporary resources without stopping shared services

### Requirement: Browser UI coverage measures layout and accessible interaction

Browser UI checks SHALL cover the notifier pilot and shared controls in light and dark themes, a narrow viewport, long existing translations and Chinese fallback. They SHALL verify keyboard activation, visible focus, dialog focus containment and restoration, field labels, language/theme changes without losing input, and a graph with nonzero rendered geometry. Automated accessibility scanning SHALL supplement explicit keyboard assertions. Enlarged-text checks SHALL measure reflow; a higher pixel density alone SHALL not count as enlarged text.

#### Scenario: A chart is rendered

- **WHEN** a fixture supplies enough monitoring points to draw a graph
- **THEN** the browser check verifies that the actual plot has positive width and height

#### Scenario: A user switches language while editing

- **WHEN** a browser scenario changes language with an unsaved SMTP draft and visible validation
- **THEN** the draft and focusable actions remain available while the validation and labels update

### Requirement: Visual baselines require an explicit update

Visual comparisons SHALL use a small declared matrix of themes, languages, viewports and stable data in a pinned browser, operating-system and font environment shared by local runs and CI. Ordinary runs SHALL fail on missing or changed reference images without accepting replacements. Updating references SHALL require an explicit separate action. A run SHALL wait for fonts and relevant rendering readiness, control time-dependent content and animations, and preserve enough failure evidence to review the difference without real credentials.

#### Scenario: A reference image changes

- **WHEN** an ordinary run produces a screenshot different from the committed reference
- **THEN** the run fails and leaves reviewable evidence without overwriting the reference

#### Scenario: A reference image is missing

- **WHEN** an ordinary run needs a reference image that is not committed
- **THEN** it fails rather than treating the newly produced image as an accepted baseline
