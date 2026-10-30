## Purpose

Defines the anonymous usage report an instance sends, the information it excludes, its persistent identity, and when reporting is enabled or retried.

## ADDED Requirements

### Requirement: Telemetry reports only anonymous usage facts

The report SHALL contain the instance identifier, application version, operating system, architecture, installation date when available, a positive user count when there are users, and arrays of active databases, storage types, notifier types and verification-agent capacities. Each array SHALL contain at most 200 entries. Storage and notifier types SHALL be sorted and SHALL preserve duplicates.

A database entry SHALL contain its engine type, version and whether an SSH tunnel is enabled. It SHALL include the physical backup mode when applicable, positive raw and compressed sizes from its latest completed logical backup or latest completed physical full backup, and its enabled verification schedule when one exists. Positive sizes SHALL be rounded up to whole megabytes; zero or unavailable sizes SHALL be omitted. Verification schedules after a backup SHALL omit the interval; interval schedules SHALL include it. Agent capacities SHALL contain CPU, memory in megabytes, disk in gigabytes and maximum concurrent jobs, without agent identity.

#### Scenario: A report describes the configured instance

- **WHEN** an instance with users, active databases, duplicate storage and notifier types, and verification agents sends a report
- **THEN** the receiver gets the facts above, including duplicate type entries and agent capacities in their declared units
- **AND** an array with more than 200 entries contributes only its first 200 entries

#### Scenario: Database sizes come from the appropriate backup

- **WHEN** a logical database has a completed logical backup and a physical database has a completed full backup
- **THEN** each reported database uses its own backup's sizes, with a positive size below one megabyte reported as one
- **AND** a database without the appropriate completed backup has no size fields

### Requirement: Telemetry excludes identifying and secret configuration

The report SHALL NOT contain account emails or names, workspace names or identifiers, database names or identifiers, connection addresses, ports, credentials, storage paths or addresses, notifier destinations, agent names or tokens, backup content, or error text. Its persistent instance identifier SHALL be the only identifier used to correlate installations.

#### Scenario: Configuration contains sensitive values

- **WHEN** the instance sends a report for objects whose configuration contains names, hostnames, paths, passwords and tokens
- **THEN** none of those configuration values appears in the report
- **AND** the report contains only the declared anonymous facts

### Requirement: Only active databases contribute to telemetry

A database with an availability status SHALL be included only when that status is available. A database without an availability status SHALL be included only when it has a sufficiently recent backup. For a logical database, this means a successful backup created at or after seven days before collection. For a physical database, its recorded last backup time SHALL be later than seven days before collection. All comparisons SHALL use the same current time for the report.

#### Scenario: Availability decides inclusion

- **WHEN** two databases have availability statuses available and unavailable
- **THEN** the report includes the available database and excludes the unavailable database

#### Scenario: Backup recency decides inclusion without an availability status

- **WHEN** databases without an availability status have recent, old or no backups
- **THEN** only those meeting the seven-day rule are reported

### Requirement: An installation keeps its telemetry identity

The instance SHALL retain its existing telemetry identifier and installation date across upgrades and restarts. A missing identity SHALL be created once with a random identifier and the current UTC installation date, and its persistent file SHALL be readable and writable only by its owner. An existing identity that is unreadable, invalid or missing its identifier SHALL NOT be overwritten or replaced by a temporary identifier. Failure to read or persist identity SHALL skip reporting without preventing the application from serving requests.

#### Scenario: Restart preserves an existing installation

- **WHEN** an instance with a saved identity restarts and sends another report
- **THEN** it reports the same identifier and installation date

#### Scenario: The identity cannot be read or written

- **WHEN** the saved identity is corrupt or the instance cannot persist a new identity
- **THEN** no telemetry request is sent and the application keeps serving requests
- **AND** a corrupt existing identity remains unchanged

### Requirement: Operators can disable anonymous reporting

When `IS_DISABLE_ANONYMOUS_TELEMETRY` is enabled, the instance SHALL send no report and SHALL neither create nor read its telemetry identity. Disabling reporting SHALL NOT make instance health fail or prevent other background work.

#### Scenario: Reporting is disabled

- **WHEN** the instance starts with anonymous telemetry disabled and reporting opportunities pass
- **THEN** the receiver gets no request, no telemetry identity is created or read, and other application work continues

### Requirement: Reporting preserves warmup, daily jitter and retry backoff

An enabled instance SHALL wait at least one minute after reporting starts before its first attempt. After an HTTP success, the next report SHALL be due 24 hours after completion plus a random offset between minus and plus ten minutes. A failed attempt SHALL be retried after one minute, with the delay doubled after each consecutive failure up to 24 hours. Success SHALL reset the retry delay. No collection or send SHALL occur before its due time, and reporting SHALL NOT overlap itself. These deadlines SHALL be evaluated at reporting opportunities no more than one minute apart while the instance is running normally.

A report SHALL be an HTTP POST with JSON content and a user agent of `Databasus-Telemetry/<version>` to the configured receiver. Any 2xx status SHALL count as success; other statuses and transport failures SHALL count as failure. The HTTP send SHALL have a five-second timeout and SHALL stop when the application stops. A skipped report because identity is unavailable SHALL wait the ordinary daily interval before trying identity again, without creating a new identity.

#### Scenario: The first report waits for warmup

- **WHEN** less than one minute has elapsed since reporting started
- **THEN** no report is collected or sent
- **AND** the first reporting opportunity at or after that minute attempts delivery

#### Scenario: Failures increase the retry delay and success resets it

- **WHEN** consecutive attempts fail and a later attempt receives a 2xx response
- **THEN** retry delays grow from one minute by doubling up to 24 hours
- **AND** the successful attempt schedules the next daily report with jitter, and the next failure starts again with a one-minute retry

#### Scenario: Shutdown interrupts an in-flight send

- **WHEN** the application stops while the receiver has not answered
- **THEN** the send ends through cancellation and no further report starts
