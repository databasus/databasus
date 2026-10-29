## MODIFIED Requirements

### Requirement: The contract is the single source of the API shape

The repository SHALL hold two OpenAPI documents. The user API contract describes the API that the frontend and users' scripts call. The agent API contract describes the operations the verification agent calls, apart from the version and agent download operations, whose paths never change. Both have paths relative to `/api/v1`, and no operation SHALL appear in both. One command, `mise run generate-api`, SHALL generate the backend's server interface, the backend's test client and the frontend's API types from the user API contract, and the backend's agent server interface, the agent's client and the agent test client from the agent API contract. Generated files SHALL NOT be edited by hand. Every operation in either document SHALL have a tag, an operation ID and a default error response with the `Problem` schema.

#### Scenario: Developer changes the contract

- **WHEN** a developer adds a field to the version response in the contract and runs `mise run generate-api`
- **THEN** the backend server interface, the backend test client and the frontend types all carry the new field

#### Scenario: Developer changes the agent contract

- **WHEN** a developer adds a field to the heartbeat response in the agent API contract and runs `mise run generate-api`
- **THEN** the backend's agent server interface, the agent's client and the agent test client all carry the new field

### Requirement: Stale generated code fails the lint

`mise run lint` SHALL fail when any generated backend or frontend file differs from what `mise run generate-api` would produce from the current contracts, and SHALL name the stale file. `mise run lint` SHALL also fail when either contract breaks a rule of the API design rules, such as an operation without a default `Problem` response, and SHALL name the rule and the place. An exception to a rule SHALL be written next to the rule with its reason, and SHALL cover only the operations it names.

#### Scenario: Contract edited without regeneration

- **WHEN** a developer edits the contract and runs `mise run lint` without running `mise run generate-api`
- **THEN** the lint fails and names each generated file that is out of date

#### Scenario: Agent contract edited without regeneration

- **WHEN** a developer edits the agent API contract and runs `mise run lint` without running `mise run generate-api`
- **THEN** the lint fails and names the generated agent server and agent client files that are out of date

#### Scenario: Operation without an error response

- **WHEN** the contract has an operation without a default `Problem` response
- **THEN** the lint fails and names the operation and the rule

#### Scenario: Agent operation without an error response

- **WHEN** the agent API contract has an operation without a default `Problem` response
- **THEN** the lint fails and names the operation and the rule

#### Scenario: An exception covers only the paths it names

- **WHEN** the user API contract gets a path with a singular segment, such as `/verification/{verificationId}/report`, and the developer runs `mise run lint`
- **THEN** the lint fails and names `databasus-path-plural-nouns` and the path, while the frozen paths of the agent API contract pass

### Requirement: Backend tests hold every response to the contract

A backend test that calls the API through a contract-checking test client SHALL fail when any response it receives differs from the contract in content type, body shape, a required field, a field type or a field format. The user API test client SHALL check against the user API contract, and the agent test client SHALL check against the agent API contract. A status the operation declares SHALL be checked against that status's body. Any other status falls under the operation's `default` response, so its body SHALL be a `Problem`. The failure SHALL name the operation, the status and the difference.

#### Scenario: Handler returns an undeclared field type

- **WHEN** the version handler returns the version as a number instead of a string, and a test calls it through the test client
- **THEN** the test fails and the message names the operation and the field

#### Scenario: Error without a Problem body

- **WHEN** an operation answers an error status with a JSON body that is not a `Problem`
- **THEN** the test calling it fails and names the status

#### Scenario: Undeclared success status

- **WHEN** the version operation answers `201` with the version body, a status the contract does not declare for it
- **THEN** the test calling it fails, because the body is checked as a `Problem` under `default`

#### Scenario: Agent operation answers an error without a Problem body

- **WHEN** an agent operation answers `410` with a JSON body that is not a `Problem`, and a test calls it through the agent test client
- **THEN** the test fails and names the operation and the status
