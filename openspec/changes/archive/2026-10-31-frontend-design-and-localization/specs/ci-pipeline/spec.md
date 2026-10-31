## ADDED Requirements

### Requirement: Browser UI verification gates frontend release

CI SHALL run the browser UI suite as its own check named `test-frontend-browser-ui`, using the same entry point, pinned toolchain and visual environment as local verification. Dependency installation SHALL preserve the committed lockfile. Release version selection and subsequent release publication SHALL require that check to pass alongside their existing requirements. The check SHALL not require the shared development application or a real-backend test service.

#### Scenario: Browser UI verification fails on main

- **WHEN** ordinary frontend tests pass but browser UI verification fails for a commit on the release branch
- **THEN** release version selection and release publication do not proceed for that commit

#### Scenario: CI starts the browser UI check

- **WHEN** CI prepares browser UI verification
- **THEN** it runs the documented local entry point with the committed dependency pins and the same screenshot environment
- **AND** it requires no manually started application
