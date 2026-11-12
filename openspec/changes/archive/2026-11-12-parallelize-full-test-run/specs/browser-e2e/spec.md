## MODIFIED Requirements

### Requirement: Overlapping runs do not disturb each other

Runs SHALL be able to overlap on one machine, from one worktree or several. Each run SHALL own its names, ports, data and temporary directories, and SHALL NOT take any machine-wide lock. Every network a run creates on the host's Docker engine SHALL have a subnet inside the test network range that the testing documentation names, which lies outside Docker's default address pools, so that overlapping runs and other suites do not use up those pools. A run SHALL NOT stop or change the shared test PostgreSQL server, the development services or another run's resources.

#### Scenario: Two runs overlap

- **WHEN** two runs execute concurrently
- **THEN** both pass, and their accounts, data, ports and temporary directories are different

#### Scenario: A nested test process finishes

- **WHEN** a browser-suite pytest process finishes while another process is using temporary files
- **THEN** it removes only its own temporary root and leaves the other process's files usable

#### Scenario: A run's networks stay in the test network range

- **WHEN** a run starts its application, browser and recovery environment
- **THEN** every network it creates on the host's Docker engine has a subnet inside the test network range that the testing documentation names

## ADDED Requirements

### Requirement: Crashed-run probes can run apart from the other probes

A crashed-run probe is a lifecycle probe that leaves a crashed run behind on purpose: it kills a run or the process recorded as a run's supervisor, or it prevents a run's cleanup, so that a later run is expected to remove what remains. Any run that starts in the meantime removes that crashed run first, so a crashed-run probe gives a reliable result only while its caller starts no other run. Every crashed-run probe SHALL register the unconditional teardown that the lifecycle guarantees require through the one shared mechanism for it, and a probe that registers teardown through that mechanism SHALL count as a crashed-run probe without further marking. The lifecycle command SHALL let its caller run crashed-run probes apart from the others: one selection SHALL run only crashed-run probes, and another SHALL run every other probe. Both selections SHALL combine with a group selection, and together they SHALL cover every applicable probe of the chosen group, or of the whole command, exactly once. The command SHALL also let its caller split a selection into a given number of parts, so that that many processes run it at once; together the parts SHALL cover the selection exactly once. Without either selection, the command SHALL run every applicable probe as it does today. Each selection SHALL keep the failure reporting, signal handling, cleanup and isolation guarantees of a complete run, and a selection with no collected probe SHALL fail rather than report an empty success. Runs that another worktree starts while crashed-run probes run can still remove their crashed runs first; the suite takes no machine-wide lock against that race.

#### Scenario: The two selections cover the suite

- **WHEN** the crashed-run probes and the other probes are collected for one group under the same image environment
- **THEN** together they match that group's complete selection, with no probe missing or repeated

#### Scenario: The parts of a selection cover it

- **WHEN** the selection without crashed-run probes is collected once whole and once in each of its parts
- **THEN** together the parts match the whole selection, with no probe missing or repeated

#### Scenario: Every probe that leaves a crashed run is a crashed-run probe

- **WHEN** a probe kills a run, kills a process recorded as a run's supervisor or prevents a run's cleanup, including a probe added later
- **THEN** it registers its teardown through the shared mechanism and runs only in the crashed-run selection

#### Scenario: The other probes overlap

- **WHEN** the selection without crashed-run probes runs for each group, or in each of its parts, at the same time
- **THEN** every probe passes, and none of them removes a resource another live run owns
