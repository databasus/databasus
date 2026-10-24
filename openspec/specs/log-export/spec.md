# Log Export Specification

## Purpose

Defines where Databasus sends its logs, so an operator can read them in the log system they already run and can still read them after the container or the internal database is lost.

## Requirements

### Requirement: Every log record goes to stdout, a log file and an optional collector

Databasus SHALL write every log record to stdout and to a rotating file in its data volume. When the operator configures an OpenTelemetry endpoint, Databasus SHALL also export every record there over OTLP. When the endpoint is unreachable, Databasus SHALL drop the records meant for it and SHALL NOT delay requests or backups.

#### Scenario: Collector configured

- **WHEN** the operator configures an OTLP endpoint and a backup runs
- **THEN** the backup's log records arrive at the endpoint and also appear in stdout and in the log file

#### Scenario: Collector down

- **WHEN** the configured endpoint stops answering
- **THEN** requests and backups continue at their usual speed
- **AND** stdout and the log file keep receiving every record

### Requirement: Audit events survive the loss of the internal database

Every audit event SHALL be written to the log file as well as to the internal database. In the log file it SHALL be marked as an audit record. The log level setting SHALL NOT suppress audit records.

#### Scenario: Internal database lost

- **WHEN** the internal database is wiped after users performed actions
- **THEN** those actions can still be read from the log file

### Requirement: Records of one request share an identifier

Databasus SHALL give every HTTP request a new identifier and return it in the `X-Request-Id` response header. It SHALL attach the identifier to every log record written while handling the request. It SHALL ignore an `X-Request-Id` sent by the client.

#### Scenario: Error during a request

- **WHEN** a request fails
- **THEN** its access log record, its audit event and the error record carry the same identifier, which the client received in `X-Request-Id`

#### Scenario: Client sends its own identifier

- **WHEN** a client sends `X-Request-Id: abc`
- **THEN** the request is logged under a new identifier, not `abc`

### Requirement: Secrets never reach a log sink

Databasus SHALL mask passwords, tokens, keys, authorization headers, cookies and email addresses in log records before any sink receives them.

#### Scenario: Error that contains a connection password

- **WHEN** an error message includes a database password
- **THEN** stdout, the log file and the collector receive the record with the password masked
