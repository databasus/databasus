## ADDED Requirements

### Requirement: MySQL and MariaDB connection settings never act as options on the Databasus host

The host, user name, password and database name of a MySQL or MariaDB database SHALL reach the client tools and the database driver on the Databasus host only as those values, never as an option or a parameter of that software. A value SHALL NOT be able to make Databasus read, create, change or truncate a file on its host.

Every operation that takes MySQL or MariaDB settings from the request SHALL refuse them with `400`, `code` `invalid_request` and the setting named, change nothing and attempt no connection when:
- the database name starts with `-`;
- the host, the user name, the password or the database name contains a line break or another control character.

These operations are creating a database, changing a database, testing unsaved settings, checking whether a limited user can be created and creating a limited user.

Any other character SHALL stay allowed. A database name that holds characters such as `?`, `&`, `=` or `/` SHALL be used whole as the name of the database, and a password that holds `@`, `/`, `?` or `:` SHALL be used whole as the password.

A saved database whose values break these rules because they were stored before the rules existed SHALL fail its backup with an error that names the setting, and SHALL NOT create or change any file on the Databasus host.

#### Scenario: A database name that starts with a dash

- **WHEN** a member creates a MySQL database with the database name `--result-file=/tmp/out.sql`
- **THEN** the response is `400` with `code` `invalid_request` naming the database name, no connection is attempted, the workspace has no new database, and `/tmp/out.sql` does not exist

#### Scenario: A line break in the user name

- **WHEN** a member creates a MariaDB database whose user name contains a line break
- **THEN** the response is `400` with `code` `invalid_request` naming the user name, no connection is attempted, and the workspace has no new database

#### Scenario: Changing a saved database to a dash-led name

- **WHEN** a member changes the database name of a saved MySQL database to `--tee=/tmp/out.log`
- **THEN** the response is `400` with `code` `invalid_request` naming the database name, no connection is attempted, and the database keeps its stored name

#### Scenario: Testing unsaved settings with a dash-led database name

- **WHEN** a member tests unsaved MySQL settings whose database name starts with `-`
- **THEN** the response is `400` with `code` `invalid_request` naming the database name, and no connection is attempted

#### Scenario: Creating a limited user with a line break in the host

- **WHEN** a member creates a limited user for unsaved MariaDB settings whose host contains a line break
- **THEN** the response is `400` with `code` `invalid_request` naming the host, and no connection is attempted

#### Scenario: A database name carrying driver parameters

- **WHEN** a member tests MySQL settings whose database name is `app?allowAllFiles=true&x` against a reachable server that has no database of that name
- **THEN** the response reports a failed connection whose reason names the unknown database `app?allowAllFiles=true&x`, so the whole value was used as the name and none of it as a driver parameter

#### Scenario: A password with URL separators

- **WHEN** a member tests MariaDB settings whose password is `p@ss/w?rd:1` against a server that accepts that password
- **THEN** the response reports a successful connection

#### Scenario: A stored dash-led database name

- **WHEN** a backup starts for a MySQL database saved earlier with the database name `--result-file=/tmp/out.sql`
- **THEN** the backup fails with an error naming the database name, and `/tmp/out.sql` does not exist

#### Scenario: A stored line break in the user name

- **WHEN** a backup starts for a MariaDB database saved earlier with the user name `x`, a line break and `result-file=/tmp/out.sql`
- **THEN** the backup fails with an error naming the user name, and `/tmp/out.sql` does not exist
