## ADDED Requirements

### Requirement: Bounded dependency checks gate release and development-image publication

CI SHALL run the same bounded local dependency check available to maintainers against the candidate commit. Release version selection, release publication and both supported architectures' development-image publication routes SHALL require its success for that commit. Failed, cancelled, skipped, missing or timed-out required results SHALL block those operations. The check SHALL inspect the recorded inputs for both supported architectures without downloading or executing their artifacts. Existing native frozen-installation, integrity, test and security checks SHALL remain required on their existing publication routes. CI SHALL NOT require an independent publication-age proof service or trusted observation history.

#### Scenario: A known dependency pin is invalid

- **WHEN** a candidate has an unpinned third-party reference, conflicting known versions or a changed recorded client file
- **THEN** the bounded dependency check fails and publication remains blocked

#### Scenario: The required check does not succeed

- **WHEN** other checks pass but the required dependency result is missing, skipped, cancelled or timed out
- **THEN** release version selection and publication remain blocked

#### Scenario: One architecture's required record is missing

- **WHEN** a known artifact supports both architectures but its records cover only one
- **THEN** the bounded check fails even if it runs on the covered architecture

#### Scenario: Offline pin checks pass but a native installation fails

- **WHEN** local pins are valid but a native frozen installation finds a stale graph or corrupted package
- **THEN** the native job fails and its existing publication gate remains closed
