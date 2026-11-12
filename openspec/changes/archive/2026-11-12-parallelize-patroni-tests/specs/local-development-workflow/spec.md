## ADDED Requirements

### Requirement: The Patroni suite finishes within ten minutes on the reference machine

The reference developer machine has at least 10 performance cores and 32 GB of memory free for tests, out of about 64 GB in total. On that machine, the complete Patroni physical backup suite run alone as a direct package run SHALL finish within ten minutes, with about five minutes as the target. The suite SHALL NOT limit how many of its scenarios run at once below the parallel test limit the test runner is given, so the runner's limit alone decides its concurrency locally and in CI. Running more scenarios at once than the machine has processor cores is acceptable; the memory free for tests is the bound. Every scenario SHALL keep checking what it checks today, including its real-time waits.

#### Scenario: The suite runs alone on the reference machine

- **WHEN** a developer runs the complete Patroni suite as a direct package run on the reference machine
- **THEN** it finishes within ten minutes
- **AND** every backup and fixture lifecycle scenario runs and passes, with none skipped

#### Scenario: A runner limit above two is reached

- **WHEN** the test runner is given a parallel test limit N greater than two and at least N Patroni scenarios are waiting to run
- **THEN** N Patroni clusters are observed running at the same time

#### Scenario: The CI limit of two holds

- **WHEN** the test runner is given a parallel test limit of two, as CI uses
- **THEN** no more than two Patroni clusters are observed running at the same time
