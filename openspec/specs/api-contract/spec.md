# API Contract Specification

## Purpose

Defines how the HTTP API contract describes every operation, how servers, test clients and frontend types are generated from it and checked against it, and how errors reach the user.

## Requirements

### Requirement: The contract is the single source of the API shape

The repository SHALL hold one OpenAPI document for the API that the frontend and users' scripts call, with paths relative to `/api/v1`. The backend's server interface, the backend's test client and the frontend's API types SHALL be generated from it by one command, `mise run generate-api`, and SHALL NOT be edited by hand. Every operation in the document SHALL have a tag, an operation ID and a default error response with the `Problem` schema.

#### Scenario: Developer changes the contract

- **WHEN** a developer adds a field to the version response in the contract and runs `mise run generate-api`
- **THEN** the backend server interface, the backend test client and the frontend types all carry the new field

### Requirement: Stale generated code fails the lint

`mise run lint` SHALL fail when any generated backend or frontend file differs from what `mise run generate-api` would produce from the current contract, and SHALL name the stale file. `mise run lint` SHALL also fail when the contract breaks a rule of the API design rules, such as an operation without a default `Problem` response, and SHALL name the rule and the place.

#### Scenario: Contract edited without regeneration

- **WHEN** a developer edits the contract and runs `mise run lint` without running `mise run generate-api`
- **THEN** the lint fails and names each generated file that is out of date

#### Scenario: Operation without an error response

- **WHEN** the contract has an operation without a default `Problem` response
- **THEN** the lint fails and names the operation and the rule

### Requirement: Backend tests hold every response to the contract

A backend test that calls the API through the contract-checking test client SHALL fail when any response it receives differs from the contract in content type, body shape, a required field, a field type or a field format. A status the operation declares SHALL be checked against that status's body. Any other status falls under the operation's `default` response, so its body SHALL be a `Problem`. The failure SHALL name the operation, the status and the difference.

#### Scenario: Handler returns an undeclared field type

- **WHEN** the version handler returns the version as a number instead of a string, and a test calls it through the test client
- **THEN** the test fails and the message names the operation and the field

#### Scenario: Error without a Problem body

- **WHEN** an operation answers an error status with a JSON body that is not a `Problem`
- **THEN** the test calling it fails and names the status

#### Scenario: Undeclared success status

- **WHEN** the version operation answers `201` with the version body, a status the contract does not declare for it
- **THEN** the test calling it fails, because the body is checked as a `Problem` under `default`

### Requirement: Backend API tests run against the real application

A backend API test SHALL call an application built by the same code that builds the server, with the same routes and middleware, its own internal database and no background loops running on their own. Data one test writes SHALL NOT be visible to another test. While code outside the target layout is wired through process-wide state, the API tests of one package SHALL run one at a time: a second test application started in a process while one is alive SHALL fail that test with a message saying so. Test packages SHALL still run in parallel with each other.

#### Scenario: A test reaches the server's router

- **WHEN** a test builds a test application and calls `GET /api/v1/system/version` through it
- **THEN** the response passes through the same middleware as on the server, so it carries an `X-Request-Id` header

#### Scenario: Two tests do not share data

- **WHEN** one test creates a row in its test application's internal database and a later test of the same package builds its own test application
- **THEN** the later test's database does not hold that row

#### Scenario: Two test applications at once

- **WHEN** a test builds a test application while another one is alive in the same process
- **THEN** that test fails with a message that API tests of one package run one at a time

### Requirement: Frontend tests mock the network by the contract

A frontend test that mocks an API response SHALL be able to give only a body the contract allows for the mocked status: the declared body for a declared status, and a `Problem` for any other status, which falls under `default`. A mock with a body of the wrong shape SHALL fail the frontend type check.

#### Scenario: Mock with a wrong body

- **WHEN** a frontend test mocks the version operation with a body without the `version` field
- **THEN** the frontend type check fails on that mock
