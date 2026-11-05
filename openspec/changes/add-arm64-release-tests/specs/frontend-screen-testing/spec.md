## MODIFIED Requirements

### Requirement: Visual baselines require an explicit update

Visual comparisons SHALL use a small declared matrix of themes, languages, viewports and stable data in a pinned browser, operating-system and font environment reproducible locally and in CI for each supported processor architecture. The browser UI command SHALL run natively on amd64 and arm64, selecting the matching pinned runtime and browser environment without processor emulation. An unsupported processor architecture SHALL produce an explicit failure before scenarios run.

Ordinary runs SHALL fail on missing or changed reference images without accepting replacements. Updating references SHALL require an explicit separate action. A run SHALL wait for fonts and relevant rendering readiness, control time-dependent content and animations, and preserve enough failure evidence to review the difference without real credentials. Architecture-specific references, if necessary for native rendering differences, SHALL be selected explicitly and SHALL follow the same update rules and comparison thresholds; a missing reference SHALL NOT fall back to the other architecture or disable its comparison.

#### Scenario: A reference image changes

- **WHEN** an ordinary run on either architecture produces a screenshot different from its committed reference
- **THEN** the run fails and leaves reviewable evidence without overwriting the reference

#### Scenario: A reference image is missing

- **WHEN** an ordinary run needs a reference image that is not committed for its selected environment
- **THEN** it fails rather than accepting its screenshot or borrowing an incompatible reference

#### Scenario: Native ARM browser verification starts

- **WHEN** the browser UI command runs on arm64
- **THEN** its browser and runtime execute natively at the committed versions
- **AND** the suite retains the same layout, accessibility and visual assertions as amd64

#### Scenario: The browser command runs on an unsupported architecture

- **WHEN** the browser UI command runs on a processor architecture outside amd64 and arm64
- **THEN** it fails with an architecture diagnostic before running scenarios
