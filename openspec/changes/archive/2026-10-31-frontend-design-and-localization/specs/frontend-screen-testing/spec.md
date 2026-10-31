## ADDED Requirements

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
