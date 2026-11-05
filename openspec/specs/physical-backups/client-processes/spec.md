# Physical Backup Client Processes Specification

## Purpose

Defines how the client tools that physical backups and WAL streaming run on the Databasus host end, so that no tool keeps reading from the source or holding its replication slot after its backup, its stream or Databasus itself has stopped.

## Requirements

### Requirement: An ended physical backup leaves no client process behind

When a full or incremental backup ends before its tool exits, Databasus SHALL stop the tool and every process the tool started on the Databasus host. This covers a cancellation by a user, a stalled stream, a failed storage upload and a stop of Databasus. Databasus SHALL first ask the processes to stop with an interrupt, and SHALL kill every one still running 10 seconds later.

#### Scenario: A user cancels a running full backup

- **WHEN** a user cancels a full backup while its tool is streaming
- **THEN** the tool and every process it started have exited within 10 seconds
- **AND** the backup run has ended

#### Scenario: The tool ignores the interrupt

- **WHEN** a backup ends while its tool ignores the interrupt
- **THEN** the tool is killed 10 seconds after the interrupt
- **AND** the backup run ends only after the tool has exited

### Requirement: A stopped WAL receiver leaves no client process behind

When Databasus stops a WAL receiver, the receiver and every process it started on the Databasus host SHALL exit before Databasus starts the next receiver of that database. This covers a stop of the WAL stream, a restart Databasus asks for to relieve back pressure or to rebuild the slot, and a stop of Databasus. A receiver still running 10 seconds after the interrupt SHALL be killed. A restart for back pressure and a stop of Databasus SHALL keep the replication slot on the source, so the next receiver resumes from it. Whether other stops keep or drop the slot stays as the specs `physical-backups/wal-archiving` and `physical-backups/wal-slot-rebuild` say.

#### Scenario: Restart for back pressure

- **WHEN** the local WAL queue grows past its limit and Databasus restarts the receiver
- **THEN** the old receiver and every process it started have exited before the next receiver starts streaming
- **AND** the replication slot still exists on the source

### Requirement: The client tools of physical backups end with Databasus

When the Databasus process ends without stopping its work, including a crash or a kill, the operating system SHALL kill every backup tool and WAL receiver that Databasus started, so none of them keeps streaming from the source.

#### Scenario: Databasus is killed during a full backup and a WAL stream

- **WHEN** the Databasus process is killed with SIGKILL while a full backup and a WAL stream run
- **THEN** the backup tool and the WAL receiver are killed at once

### Requirement: The fail message of a physical backup names what the tool reported

When the backup tool itself fails, the fail message of the backup SHALL contain the tool's exit status and the end of its error output. It SHALL NOT contain the path of the tool on the Databasus host.

#### Scenario: The source refuses the replication connection

- **WHEN** a full backup fails because the source rejects the backup tool's connection
- **THEN** the backup's fail message contains the tool's exit status and the server's rejection from its error output
- **AND** the fail message does not contain the directory the client tools are installed in
