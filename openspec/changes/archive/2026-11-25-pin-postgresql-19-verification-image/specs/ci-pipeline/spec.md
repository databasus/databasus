# Spec Delta

## ADDED Requirements

### Requirement: Release image publication waits out a registry rate limit

CI SHALL run at most eight verification image pushes of one release at once. When the image registry refuses a push or a manifest update of a release because of its rate limit, CI SHALL wait and try that operation again, up to four attempts in all with growing waits of about seven minutes in total. Any other refusal SHALL fail the operation at once. The release SHALL go out without manual action when the rate limit clears before the attempts run out.

#### Scenario: A push hits the registry rate limit once

- **WHEN** the registry answers one image push of a release with `429 Too Many Requests` and accepts the next attempt
- **THEN** CI waits, pushes again, and the release completes without a rerun

#### Scenario: A manifest update hits the registry rate limit once

- **WHEN** the registry answers a manifest update of a release with `429 Too Many Requests` and accepts the next attempt
- **THEN** CI waits, creates the manifest again, and the release completes without a rerun

#### Scenario: The rate limit outlasts every attempt

- **WHEN** the registry refuses all four attempts of a push with `429 Too Many Requests`
- **THEN** the push fails after the fourth attempt, and CI creates no release

#### Scenario: A push fails for another reason

- **WHEN** an image push of a release fails with an error other than the registry's rate limit
- **THEN** the push fails at once without another attempt

#### Scenario: Many verification images are pushed

- **WHEN** a release pushes the verification image of every pinned PostgreSQL major on both architectures
- **THEN** no more than eight of those pushes run at the same time
