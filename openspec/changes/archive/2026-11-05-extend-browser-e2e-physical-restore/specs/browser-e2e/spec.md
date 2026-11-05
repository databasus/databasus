## MODIFIED Requirements

### Requirement: The critical path is verified from the first account to the restored rows

On an instance holding no account, a run SHALL register the first account through the interface and see that the interface says this account will administer the instance. A scenario SHALL then, through the interface alone: sign in as that instance admin, create a workspace and a local storage, add a PostgreSQL logical database through the database wizard with the connection test passing, wait until the backup the wizard starts is shown as completed, start a restore of that backup into an empty database of the same server and wait until the restore is shown as completed. The run SHALL write rows into the database it backs up and SHALL check after the restore that this database still holds them, so two empty databases cannot pass. It SHALL then read the restored database directly and compare its rows with the rows it wrote. A status shown on a screen SHALL NOT be enough to pass.

After the logical restore passes, the same scenario SHALL continue on the same running instance, account, workspace and storage. Through the interface, it SHALL add a physical PostgreSQL 18 connection configured for full backups, incremental backups and WAL streaming, see its connection test pass, and complete a full backup followed by an incremental backup on that full backup's chain. The run SHALL write distinguishable rows before the full backup, between the full and incremental backups, and in two separate stages after the incremental backup. The latter two stages SHALL finish different WAL segments, and the run SHALL wait until both segments and the WAL needed to replay through the final write are archived. Two rows in the backup list SHALL NOT substitute for evidence of two distinct archived segments.

The scenario SHALL restore the physical chain to its latest archived state twice, first through the Host tab and then through the Docker tab of the Script restore method. For each restore, it SHALL obtain a fresh command through the interface, fill a distinct non-default restore directory containing a space, fill the tab's PostgreSQL tools or image field, and execute the rendered recovery command unchanged. The scenario SHALL also execute the startup commands shown by that tab in the prepared recovery environment. It SHALL NOT reconstruct the commands from an API response or application code, replace their URLs or arguments, or repair a displayed command in the test. Each restored cluster SHALL finish recovery and contain exactly the recorded rows from all four write stages. The source SHALL still contain those rows after both restores.

#### Scenario: The path works

- **WHEN** the critical path scenario runs
- **THEN** it passes, and the restored database holds exactly the rows of the backed-up database
- **AND** both physical restore environments serve the recorded full, incremental and WAL-stage rows

#### Scenario: A restore loses rows

- **WHEN** the restore is shown as completed but the restored database does not hold the backed-up database's rows
- **THEN** the run fails and names the difference

#### Scenario: The first account takes the instance

- **WHEN** the run opens its empty instance
- **THEN** the sign-up form says the account will administer the instance, and after submitting it the account can open the instance settings

#### Scenario: A command shown by the interface is wrong

- **WHEN** a rendered recovery or startup command uses an incorrect path, image, tool directory or argument
- **THEN** the run fails at command execution or at the restored-data check instead of correcting the command

#### Scenario: WAL is missing from a restore

- **WHEN** the restored cluster lacks either of the two write stages made after the incremental backup
- **THEN** the run fails and identifies the missing rows, even if backup and restore statuses look successful

#### Scenario: Both tabs restore the same chain

- **WHEN** the Host restore finishes and the scenario starts the Docker restore
- **THEN** it obtains a new single-use restore command through the interface and writes to a different directory
- **AND** it keeps the same Databasus instance, account, workspace, storage and physical backup chain

### Requirement: Secrets stay out of a run's output

Each run SHALL generate its own account and database passwords, and SHALL pass them to what it starts through the environment of those processes and not through a file in the repository. A run SHALL record no browser trace, video or screenshot and SHALL retain no file of its own after it ends other than the ignored dependency directory. Docker's layer cache, pulled images and the package manager's cache are shared caches and stay. Everything a run prints, including its test output, the application's and containers' logs, and recovery command diagnostics, SHALL have the run's generated passwords, any bearer token and any physical restore token masked. A failed run SHALL print the end of the log of each of its containers, the application's and recovery environments' included. Failure output SHALL NOT include the values of local variables or reveal a restore token through a URL or an echoed command.

#### Scenario: A failed run

- **WHEN** a run fails after signing in
- **THEN** its printed output, including test output, the application's log and container logs, contains neither the run's passwords nor any bearer token or physical restore token, and no file of the run remains on disk other than the ignored dependency directory

#### Scenario: A recovery command fails

- **WHEN** a recovery command fails and its diagnostic includes the restore URL
- **THEN** the output identifies the failing recovery stage and exit status while masking the restore token

#### Scenario: Recovery log collection stops responding

- **WHEN** a failed run's private recovery engine stops responding during log collection
- **THEN** the run prints the available outer container logs and partial recovery output with secrets masked
- **AND** it identifies the log collection timeout and removes its resources

## ADDED Requirements

### Requirement: Physical recovery environments belong to the run

A run SHALL prepare the tools, empty directories and isolated container engine needed by both restore environments without requiring a running development application or installing PostgreSQL on the developer's host. Commands shown by the interface SHALL resolve the same application address from both recovery environments as from the browser. The run SHALL leave that address unchanged in the commands and SHALL publish no application or restored-database port on the host. Every recovery process, container, engine state and directory SHALL belong to that run and SHALL be removed after success, failure, interruption and recovery of a crashed run. Overlapping runs SHALL NOT share writable recovery data or a recovery engine.

#### Scenario: Two complete critical paths overlap

- **WHEN** two browser runs execute logical and physical recovery concurrently
- **THEN** each restores its own recorded rows, and cleaning up either run leaves the other one's recovery resources usable

#### Scenario: A run fails during physical recovery

- **WHEN** a run fails or is interrupted after its recovery engine has started a restore container
- **THEN** cleanup removes the recovery processes, containers, directories and engine state without changing the shared host engine or another run

#### Scenario: A physical recovery run is killed

- **WHEN** a run is killed while its recovery resources exist and a later run starts
- **THEN** the later run removes those resources before starting its own and leaves live runs untouched
