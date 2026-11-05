# Internal PostgreSQL Protection Specification

## Purpose

Prevents workspace-controlled PostgreSQL connections and backups from reading the Databasus metadata database or sending another workspace's saved credentials to a caller-controlled server.

## Requirements

### Requirement: Logical backups cannot target the embedded metadata database

The system SHALL reject a logical PostgreSQL source when its database name is `databasus` and any effective host points back to the Databasus container. Local targets include filesystem and abstract Unix sockets, loopback and unspecified IP addresses, supported container-local aliases, libc-compatible numeric IPv4 forms, and an empty entry in a comma-separated host list.

The system SHALL inspect every host in a multi-host value. A remote entry SHALL NOT make a later local entry acceptable.

#### Scenario: Filesystem Unix socket

- **WHEN** a user configures a logical PostgreSQL source with database `databasus` and a filesystem Unix-socket directory as its host
- **THEN** the system rejects the source before opening the connection

#### Scenario: Abstract Unix socket

- **WHEN** a user configures database `databasus` with an abstract Unix-socket host
- **THEN** the system rejects the source before opening the connection

#### Scenario: Loopback address in a host list

- **WHEN** a PostgreSQL host list contains a remote hostname followed by a loopback address and the database is `databasus`
- **THEN** the system rejects the whole source

#### Scenario: Numeric loopback spelling

- **WHEN** a user specifies a loopback address in an abbreviated, integer, octal, or hexadecimal form accepted by the operating-system resolver
- **THEN** the system treats it as local and rejects database `databasus`

#### Scenario: Local application database

- **WHEN** a user configures a local PostgreSQL source whose database name is not `databasus`
- **THEN** the embedded-target rule allows the source

#### Scenario: Remote database with the same name

- **WHEN** a PostgreSQL source is remote and its database name is `databasus`
- **THEN** the embedded-target rule allows the source

### Requirement: Physical backups cannot target the embedded PostgreSQL cluster

The system SHALL reject a physical PostgreSQL source when an effective local host uses the embedded cluster's port. This rule SHALL apply without relying on a database name because a physical backup addresses the cluster rather than one logical database.

#### Scenario: Embedded cluster through a Unix socket

- **WHEN** a user configures a physical backup against a local Unix socket and the embedded PostgreSQL port
- **THEN** the system rejects the source before any physical backup connection is opened

#### Scenario: Different local PostgreSQL instance

- **WHEN** a physical source uses a local address on a port other than the embedded PostgreSQL port
- **THEN** the embedded-target rule allows the source

### Requirement: Remote SSH targets keep their remote meaning

When a database is reached through an SSH bastion on another host, a loopback database address SHALL be interpreted relative to that remote host. A local SSH bastion SHALL NOT relax the embedded-target rules.

#### Scenario: Database bound to loopback on a remote bastion

- **WHEN** SSH tunneling is enabled through a non-local bastion and the PostgreSQL host is a loopback address
- **THEN** the embedded-target rule allows the source

#### Scenario: Local bastion points back to the container

- **WHEN** SSH tunneling names a local bastion and the PostgreSQL target matches the embedded instance
- **THEN** the system rejects the source

### Requirement: Persisted configurations are checked at connection time

The system SHALL apply embedded-target protection immediately before a PostgreSQL connection is opened, including connections created from rows saved before this policy existed. Model validation alone SHALL NOT be the only enforcement point.

#### Scenario: Existing logical row contains a socket target

- **WHEN** a previously saved logical source names the embedded database through a local socket
- **THEN** a backup, connection test, or healthcheck refuses to open that connection

#### Scenario: Existing physical row contains the embedded cluster

- **WHEN** a previously saved physical source names the embedded PostgreSQL cluster
- **THEN** physical connection and backup paths refuse to open that connection

### Requirement: PostgreSQL connection values remain literal

Every user-controlled string used in a PostgreSQL conninfo value SHALL be interpreted as one value. Whitespace, quotes, backslashes, equal signs, and text resembling another conninfo parameter SHALL NOT create or replace connection parameters.

The database name passed to a logical dump SHALL follow the same rule even when the dump tool accepts a conninfo string in place of a plain database name.

#### Scenario: Database name contains a second database parameter

- **WHEN** a database name contains `dbname=databasus` after another value
- **THEN** PostgreSQL receives the complete input as one literal database name
- **AND** the text cannot redirect the connection to the embedded database

#### Scenario: Other credential fields contain conninfo syntax

- **WHEN** a host, username, password, TLS mode, or certificate path contains whitespace or conninfo metacharacters
- **THEN** the connection parser returns the original field value without creating another parameter

### Requirement: Direct connection tests require workspace authorization

The test of unsaved database settings SHALL require an authenticated user who can manage databases in the relevant workspace. The request SHALL name that workspace. A request that names a saved database SHALL be authorized against the workspace stored on that database before saved credentials are merged with request fields. The named database SHALL be checked first: one that does not exist or does not belong to the workspace in the request SHALL get `404` before the caller's role in that workspace is checked.

#### Scenario: Ad hoc request without a workspace

- **WHEN** an authenticated user submits unsaved database settings without a workspace
- **THEN** the response is `400` with `code` `invalid_request`
- **AND** no connection is attempted

#### Scenario: Ad hoc request for an unauthorized workspace

- **WHEN** a viewer of a workspace submits unsaved database settings for that workspace
- **THEN** the response is `403` with `code` `permission_denied`
- **AND** no connection is attempted

#### Scenario: Ad hoc request for another team's workspace

- **WHEN** an authenticated user submits unsaved database settings for a workspace they are not a member of
- **THEN** the response is `404` with `code` `not_found`
- **AND** no connection is attempted

#### Scenario: Saved database from another workspace

- **WHEN** a user names a saved database of a workspace other than the one in the request, or of a workspace they are not a member of, and supplies a different host
- **THEN** the response is `404` with `code` `not_found` before the saved password is applied
- **AND** the system makes no connection to the supplied host

#### Scenario: Viewer names a saved database of their own workspace

- **WHEN** a viewer of a workspace names a saved database of that workspace and supplies a different host
- **THEN** the response is `403` with `code` `permission_denied` before the saved password is applied
- **AND** the system makes no connection to the supplied host

#### Scenario: Authorized workspace member tests a saved database

- **WHEN** a workspace member who can manage databases tests the settings of a saved database in that workspace and leaves its password blank
- **THEN** the system uses the saved password and performs the connection test

### Requirement: Direct connection-test failures do not disclose protected records or internal errors

The test of unsaved database settings SHALL return the same `404` response for an unknown saved database and a saved database of a workspace the caller is not a member of. Failures while loading a saved target and failures while checking workspace permissions SHALL return `500` with `code` `internal` and SHALL NOT include the underlying error.

#### Scenario: Unknown saved database ID

- **WHEN** a caller names an ID that does not identify a saved database
- **THEN** the response is the same `404` body used for a saved database of another team's workspace, apart from the ID

#### Scenario: Saved target lookup fails internally

- **WHEN** the internal database fails while loading the named saved database
- **THEN** the response is `500` with `code` `internal`
- **AND** the database error is absent from the response

#### Scenario: Permission lookup fails internally

- **WHEN** the workspace permission lookup fails for an ad hoc request or one that names a saved database
- **THEN** the response is `500` with `code` `internal`
- **AND** the underlying membership error is absent from the response

### Requirement: Healthchecks use a trusted internal connection path

Scheduled healthchecks SHALL test a database already loaded by the system without pretending to be an end user. The user-facing direct endpoint SHALL NOT expose that trusted path or bypass its workspace checks.

#### Scenario: Scheduled healthcheck

- **WHEN** the healthcheck service loads a saved database and probes it
- **THEN** the probe runs without an end-user principal
- **AND** embedded-target protection still runs before the connection opens

#### Scenario: User-facing direct test

- **WHEN** an HTTP caller requests a direct connection test
- **THEN** the system uses the workspace-authorized path rather than the trusted healthcheck path

### Requirement: The embedded PostgreSQL instance accepts only authenticated loopback TCP

The embedded PostgreSQL server SHALL run under the same non-root operating-system account as the Databasus application. The server SHALL create no Unix socket, so no process in the container can open a socket connection to it. Loopback TCP connections SHALL require SCRAM authentication with the generated credential from the first connection the server accepts, and replication connections to the embedded cluster SHALL be rejected.

The embedded PostgreSQL port SHALL remain unpublished by the image.

#### Scenario: A process looks for a local socket

- **WHEN** any process in a running container attempts a Unix-socket connection to the embedded PostgreSQL server
- **THEN** the connection fails because the server exposes no socket

#### Scenario: Loopback TCP connection without the generated password

- **WHEN** any process connects to the embedded PostgreSQL server over loopback TCP without the current generated password
- **THEN** SCRAM authentication rejects the connection

#### Scenario: Application uses the generated password

- **WHEN** Databasus connects to the embedded PostgreSQL server over loopback with the generated password
- **THEN** PostgreSQL accepts the connection

#### Scenario: Replication connection to the embedded cluster

- **WHEN** a client requests a replication connection to the embedded PostgreSQL server
- **THEN** PostgreSQL rejects it

### Requirement: Startup administration of the embedded cluster opens no connection path

The container SHALL set the generated password on the internal role before the embedded PostgreSQL server accepts any connection. That step SHALL run while the cluster has no network listener and no Unix socket. No startup step SHALL grant a password-less or identity-mapped login to the embedded cluster.

#### Scenario: First connection after startup administration

- **WHEN** the embedded PostgreSQL server accepts its first connection after a container start
- **THEN** the internal role already carries the newly generated password
- **AND** the connection must authenticate with that password

#### Scenario: Password from the previous run after a restart

- **WHEN** the container restarts and a process logs in over loopback with the password published by the previous run
- **THEN** SCRAM authentication rejects the connection
- **AND** a loopback login with the newly published password succeeds

### Requirement: The generated password stays out of process arguments and server logs

Startup SHALL NOT pass the generated password to any process through its command-line arguments. When the statement that sets the password fails, the container SHALL exit with an error before the Databasus application starts, and the container log SHALL NOT contain that statement or the password.

#### Scenario: Process list during startup

- **WHEN** a process in the container reads the command-line arguments of the startup processes
- **THEN** none of them contains the generated password

#### Scenario: Password statement fails

- **WHEN** the statement that sets the generated password fails
- **THEN** the container exits before starting the Databasus application
- **AND** the container log contains neither the statement nor the generated password

### Requirement: Internal PostgreSQL credentials are generated at startup

When Databasus uses its embedded metadata database, the container SHALL generate a cryptographically random PostgreSQL password at startup, set it on the internal role, and supply it to the application without logging it. A container restart SHALL replace the previous password. An explicitly supplied external metadata-database connection string SHALL remain unchanged.

The generated connection string SHALL also be readable by processes that start later in the same container, so that a command run against a running instance authenticates with the same credential as the application. That value SHALL exist only for the lifetime of the running container: it SHALL NOT be written to storage that survives a container stop, and it SHALL NOT be part of the distributed image.

When the operator supplies a metadata-database connection string, startup SHALL neither generate nor publish one, and every process in the container SHALL use the supplied value. When the operator supplies none, every process SHALL use the generated one. No process SHALL fall back to a connection string shipped as a default inside the image.

An operator-supplied connection string that points back to the embedded database with the previously published fixed password is incompatible with rotation and SHALL NOT be treated as a supported embedded-database configuration.

#### Scenario: Fresh container startup

- **WHEN** a container starts without an explicit metadata-database connection string
- **THEN** it configures the embedded role and the application with the same generated password
- **AND** the application becomes healthy without printing that password

#### Scenario: Container restart

- **WHEN** the same container restarts
- **THEN** the embedded role receives a different generated password
- **AND** the application reconnects with the new password
- **AND** a command started after the restart authenticates with the new password rather than the previous one

#### Scenario: Previously published fixed password

- **WHEN** a client attempts loopback TCP authentication with the old fixed image password
- **THEN** PostgreSQL rejects the connection

#### Scenario: External metadata database

- **WHEN** the operator explicitly supplies a connection string for an external metadata database
- **THEN** startup preserves that value rather than replacing it with the embedded database connection
- **AND** startup publishes no generated connection string for other processes to pick up

#### Scenario: Explicit connection string points to the embedded database

- **WHEN** the operator explicitly supplies the previously published fixed connection string for the embedded database
- **THEN** the application cannot authenticate after startup rotates the internal role's password
- **AND** the operator must remove that explicit value so startup can supply the generated credential

### Requirement: Console commands reach the metadata database of a running container

A console command executed inside an already-running container SHALL resolve the metadata-database connection the same way the application did, and SHALL connect successfully whenever the application itself is connected. This SHALL hold for a container using the embedded database and for one pointed at an external metadata database.

An operator-supplied connection string SHALL take precedence over the one startup generated. A console command SHALL NOT need any argument, file or environment value beyond what the operator already supplied to the container.

#### Scenario: Password reset on a fresh instance

- **WHEN** an operator runs the documented password-reset command inside a running container that uses the embedded database
- **THEN** the command connects to the metadata database and sets the new password
- **AND** the operator can sign in with it

#### Scenario: Console command against an external metadata database

- **WHEN** the operator supplied a connection string for an external metadata database and runs a console command
- **THEN** the command connects to that external database

#### Scenario: Console command while the application is running

- **WHEN** a console command runs while the application holds its own connection to the embedded database
- **THEN** both connect with the same credential and neither disturbs the other

### Requirement: Missing metadata-database configuration is reported as such

When no metadata-database connection string is available from any source, a starting process SHALL report that the configuration is missing and SHALL NOT attempt a connection with a default credential. A rejected credential and an absent configuration SHALL produce distinguishable messages.

A process that never opens the metadata database SHALL be exempt from this requirement, because container startup runs the storage probe before the embedded database exists.

#### Scenario: No connection string anywhere

- **WHEN** a process starts in a container where startup published no connection string and the operator supplied none
- **THEN** it reports the metadata-database configuration as missing
- **AND** the message does not describe an authentication failure

#### Scenario: Storage probe before the database exists

- **WHEN** container startup runs the storage probe, before any metadata-database credential exists
- **THEN** the probe completes on its own merits rather than being refused for a missing connection string

#### Scenario: Wrong credential supplied

- **WHEN** a process starts with a connection string whose password the database rejects
- **THEN** it reports an authentication failure rather than a missing configuration

### Requirement: The shared internal credential stays inside the container

The published connection string SHALL be readable only by the account that runs Databasus and by the container's privileged account. Any other account inside the container SHALL be refused. The value SHALL NOT appear in logs, in the image, or on any volume the operator mounts.

#### Scenario: Another account in the container

- **WHEN** a process running under an operating-system account other than the Databasus runtime account attempts to read the published connection string
- **THEN** the read is refused

#### Scenario: Operator inspects the mounted data volume

- **WHEN** the operator inspects the mounted data directory after the container has started
- **THEN** the generated password is absent from it

#### Scenario: Operator inspects a stopped container's image

- **WHEN** the operator inspects the distributed image
- **THEN** it contains no metadata-database connection string usable against the embedded database
