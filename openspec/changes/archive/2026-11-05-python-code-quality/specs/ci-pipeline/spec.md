## ADDED Requirements

### Requirement: CI checks Python with the local quality command

Release and dev-image workflows SHALL run a dedicated Python quality check using the same entry point, configuration and repository-pinned tools as local Python lint. The check SHALL cover all repository-owned Python, including scripts, agent hooks, tests and browser scenarios. It SHALL install its Python dependencies from the committed dependency lockfile without updating it, and SHALL require no application image, running application, Docker service or browser installation. A formatting violation, lint violation, missing annotation, type error or setup failure SHALL fail the check. The Python check SHALL run independently of application builds and test results; its failure SHALL NOT cancel unrelated running checks. Existing workflow events and branch selection SHALL remain unchanged.

#### Scenario: A hook fails type checking in release CI

- **WHEN** an agent hook has a type error in a release workflow run
- **THEN** the Python quality check fails even when application lint and tests pass

#### Scenario: The dev workflow checks a browser fixture

- **WHEN** a browser test fixture is missing an annotation in a dev-image workflow run
- **THEN** the Python quality check fails without building an image or starting the fixture

#### Scenario: A quality tool pin changes

- **WHEN** a repository pin changes for the Python linter or type checker
- **THEN** both workflows use the updated version through the local quality command

#### Scenario: Python lint fails while application tests run

- **WHEN** the Python quality check fails while other test jobs are running
- **THEN** those jobs continue and report their own results

### Requirement: Python quality gates release and dev-image publication

Release version selection and all subsequent publication SHALL require a successful Python quality check for the same commit, in addition to their existing requirements. Every dev-image publishing job SHALL require that commit's Python quality check to succeed before publishing any architecture image or manifest. A failed, cancelled, skipped or timed-out Python check SHALL prevent these dependent operations. These requirements SHALL be verified locally through automated workflow-dependency checks without needing a push or a hosted pipeline run.

#### Scenario: Python quality fails on the release branch

- **WHEN** all other release prerequisites pass but the Python quality check fails
- **THEN** release version selection does not run and no release artifact is published

#### Scenario: Python quality fails on the development branch

- **WHEN** the Python quality check fails in the dev-image workflow
- **THEN** no architecture image or combined manifest for that commit is published

#### Scenario: Python quality is cancelled or skipped

- **WHEN** the Python quality check is cancelled, skipped or times out
- **THEN** neither release publication nor dev-image publication proceeds in that workflow run

#### Scenario: Python quality succeeds

- **WHEN** Python quality and every existing prerequisite succeed
- **THEN** the workflow remains eligible to select a release version or publish the dev image according to its existing branch and release rules
