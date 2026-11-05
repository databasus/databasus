## MODIFIED Requirements

### Requirement: Backend tests detect data races
Backend tests SHALL run with the race detector in CI. On a developer's machine every repository test command, including those OpenSpec change verification names, SHALL run them without it. A developer SHALL be able to run any backend test with the race detector on demand. A data race detected during a run with the race detector SHALL fail that test.

#### Scenario: A race in a test run
- **WHEN** two goroutines write the same variable without synchronization during a backend test that runs with the race detector
- **THEN** the test fails with the race report

#### Scenario: A local package run
- **WHEN** a developer runs `mise run test`
- **THEN** the backend packages run without the race detector

#### Scenario: OpenSpec change verification
- **WHEN** an agent runs the verification commands of an OpenSpec change
- **THEN** the backend packages run without the race detector

#### Scenario: A race reported by CI is reproduced locally
- **WHEN** a developer runs the test CI reported with the Go test command and its race flag
- **THEN** the test runs with the race detector

## ADDED Requirements

### Requirement: Independent subtests run in parallel
A backend test that runs in parallel SHALL run its subtests in parallel too, unless they depend on each other or on shared state. The backend lint SHALL fail, unless the test states a reason, in three cases: a parallel test none of whose direct subtests run in parallel, a test with parallel subtests that does not run in parallel itself, and a test with parallel subtests that releases resources through `defer` and calls no test cleanup. A stated reason SHALL name the shared state or the dependency that keeps the test as it is. Review, not the lint, checks the content of a reason. Review also covers what the lint cannot see: a test that mixes parallel and sequential direct subtests, nested subtests, a `defer` beside a test cleanup, and subtests the linter cannot resolve statically.

#### Scenario: Sequential subtests without a reason
- **WHEN** none of the direct subtests of a parallel backend test run in parallel and the test carries no stated reason
- **THEN** the backend lint fails and names the test

#### Scenario: Parallel subtests under a sequential test
- **WHEN** the subtests of a backend test run in parallel, the test itself does not, and it carries no stated reason
- **THEN** the backend lint fails and names the test

#### Scenario: A deferred release beside parallel subtests
- **WHEN** a backend test with parallel subtests releases a resource through `defer`, calls no test cleanup and carries no stated reason
- **THEN** the backend lint fails and names the test, because the release would run before the subtests do

#### Scenario: A test that mixes parallel and sequential subtests
- **WHEN** one direct subtest of a parallel test runs in parallel and another does not
- **THEN** the backend lint passes, and review decides whether the sequential subtest has a reason

#### Scenario: Subtests that share a server
- **WHEN** the subtests of a parallel test share one server and the test states that reason
- **THEN** the backend lint passes and the subtests keep running one after another

#### Scenario: Independent subtests in one test
- **WHEN** the subtests of one test each create their own application and database
- **THEN** they run at the same time within the limit of parallel tests
