## ADDED Requirements

### Requirement: The live physical jobs of a test application include the backup request job

When a test asks its application to run the physical backup scheduler live, the backup request job SHALL run live with it, stop with it and restart with it when the test restarts the modules. A test SHALL be able to run one pass of that job by its name, `backups_physical_request_scheduler`, with or without the live jobs, and with the live jobs that pass SHALL run in the live job. A test SHALL be able to read the job snapshots of the live jobs. A pass by name SHALL return once it has started the backups it decided on, without waiting for them to end. Without the live jobs, the job SHALL NOT run on its own. One pass of `backups_physical_scheduler` SHALL still act on pending backup requests, so a test that runs only that job sees them served.

#### Scenario: One pass by name without live jobs

- **WHEN** a test without live jobs requests a full backup and runs one pass of `backups_physical_request_scheduler`
- **THEN** right after the pass returns, the list holds a full backup of that database that is in progress or completed

#### Scenario: One pass by name with live jobs

- **WHEN** a test with live jobs runs one pass of `backups_physical_request_scheduler`
- **THEN** the pass returns without failing the test, and the job snapshot of the live job holds a completed pass

#### Scenario: A pending request and a scheduler pass

- **WHEN** a test without live jobs requests a full backup and runs one pass of `backups_physical_scheduler`
- **THEN** right after the pass returns, the list holds a full backup of that database that is in progress or completed

#### Scenario: A pending request without live jobs and without a pass

- **WHEN** a test without live jobs requests a full backup and runs no pass
- **THEN** the list stays empty and the request stays pending

#### Scenario: The live job runs and restarts with the application

- **WHEN** a test with live jobs waits until the job snapshot of `backups_physical_request_scheduler` holds a completed pass, and then restarts the modules
- **THEN** after the restart the job snapshot of that job again holds a completed pass
- **AND** a full backup requested after the restart appears in the list without an explicit pass
