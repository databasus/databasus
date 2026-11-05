## MODIFIED Requirements

### Requirement: Changing a database keeps the secrets the request leaves out

Changing a database SHALL replace its name, settings and notifiers with those in the request, except a secret the request leaves out, which SHALL keep its stored value, unless the request changes the database's resource address as the `secret-handling` spec describes. A change SHALL first connect with the resulting settings. When the connection fails, the database SHALL keep its stored settings, and the response SHALL be `400` with `code` `database_connection_failed` and the server's reason in `detail`.

#### Scenario: Renaming a database without its password

- **WHEN** a member changes the name of a PostgreSQL database and sends its connection settings without the password
- **THEN** the database has the new name, and its next backup connects with the stored password

#### Scenario: A change that cannot connect

- **WHEN** a member changes the host of a database to one where no server answers and sends the password
- **THEN** the response is `400` with `code` `database_connection_failed`, and the database keeps its stored host

#### Scenario: A host change without the password

- **WHEN** a member changes the host of a database and leaves out the password
- **THEN** the response is `400` with `code` `invalid_request` and a field error for the password, no connection is attempted, and the database keeps its stored host

### Requirement: Unsaved settings can be tested before saving

Testing unsaved settings SHALL connect with the settings in the request and answer `200` with the outcome, as a test of a saved database does. When the request names a saved database of the same workspace, the stored secrets of that database SHALL fill the secrets the request leaves out, unless the request changes the database's resource address as the `secret-handling` spec describes. A named database of another workspace, or one that does not exist, SHALL get `404` with `code` `not_found`. Settings of another type than the named database SHALL get `400` with `code` `invalid_request`, and no stored secret SHALL be used. In both cases no connection SHALL be attempted.

#### Scenario: Testing a new database

- **WHEN** a member tests the settings of a database that is not saved yet
- **THEN** the response is `200` and reports the outcome of the connection

#### Scenario: Testing changed settings of a saved database

- **WHEN** a member tests the settings of a saved database with a new user name at the same address and without its password
- **THEN** the test connects with the new user name and the stored password

#### Scenario: Testing a saved database at another port

- **WHEN** a member tests the settings of a saved database with a new port and without its password
- **THEN** the response is `400` with `code` `invalid_request`, and no connection is attempted

#### Scenario: A database of another workspace

- **WHEN** a member of one workspace tests settings and names a database of another workspace
- **THEN** the response is `404` with `code` `not_found`, and no connection is attempted

#### Scenario: Settings of another type than the named database

- **WHEN** a member tests PostgreSQL settings and names a saved MySQL database of the workspace
- **THEN** the response is `400` with `code` `invalid_request`, and no connection is attempted

### Requirement: Only a user who may change databases sends stored secrets elsewhere

Testing unsaved settings, checking whether a limited user is advisable and creating a limited user SHALL require the right to change the workspace's content. Each of them connects to the host in the request and, with a saved database named, uses that database's stored secrets, unless the request changes the database's resource address as the `secret-handling` spec describes. A viewer SHALL get `403` with `code` `permission_denied` from each of them, and no connection SHALL be attempted.

#### Scenario: A viewer points a saved database at another host

- **WHEN** a viewer of a workspace checks whether a limited user is advisable, names a saved database, and sends a host of their own without the password
- **THEN** the response is `403` with `code` `permission_denied`, and no connection is made to that host

#### Scenario: A viewer tests unsaved settings

- **WHEN** a viewer of a workspace tests unsaved settings
- **THEN** the response is `403` with `code` `permission_denied`

#### Scenario: A member checks a limited user at another host

- **WHEN** a member of a workspace checks whether a limited user is advisable, names a saved database, and sends another host without the password
- **THEN** the response is `400` with `code` `invalid_request`, and no connection is made to that host

#### Scenario: A member creates a limited user at another host

- **WHEN** a member of a workspace creates a limited user, names a saved database, and sends another host without the password
- **THEN** the response is `400` with `code` `invalid_request`, and no connection is made to that host
