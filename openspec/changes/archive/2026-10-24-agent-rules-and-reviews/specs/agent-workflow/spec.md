## ADDED Requirements

### Requirement: The quick architecture check covers the HTTP contract
The quick architecture check SHALL check a plan that adds or changes an HTTP operation, and a finished turn that edits an HTTP contract file under `backend/api/`, against the repository's API design skill, in addition to `docs/architecture.md`. It SHALL reject the plan or the turn for each broken rule and name the rule of the API design skill.

#### Scenario: A contract edit that breaks an API rule
- **WHEN** a turn adds an operation to `backend/api/openapi.yaml` whose error responses do not use the error format the API design skill requires
- **THEN** the quick architecture check rejects the turn and names the rule of the API design skill

#### Scenario: A plan with a new operation
- **WHEN** a plan proposes an HTTP operation that lists a collection without the pagination the API design skill requires
- **THEN** the quick architecture check rejects the plan and names the pagination rule

#### Scenario: A turn without contract edits
- **WHEN** a turn changes only Go files outside `backend/api/` and adds no HTTP operation
- **THEN** the quick architecture check judges the turn against `docs/architecture.md` and the module docs only
