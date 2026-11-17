## ADDED Requirements

### Requirement: A MySQL or MariaDB restore target obeys the database connection rules

The settings of a MySQL or MariaDB restore target SHALL obey the rules the `databases` spec sets for the connection settings of those engines: they reach the client tools and the driver on the Databasus host only as values, never as options or parameters.

Starting a restore into a MySQL or MariaDB target SHALL get `400` with `code` `invalid_request`, name the target's setting, start nothing and attempt no connection when the target's database name starts with `-`, or when its host, user name, password or database name contains a line break or another control character. This SHALL hold for every role that may start a restore, a viewer included.

#### Scenario: A dash-led target database name

- **WHEN** a viewer restores a completed MySQL backup into a target whose database name is `--tee=/tmp/restore.log`
- **THEN** the response is `400` with `code` `invalid_request` naming the target's database name, the backup has no new restore, no connection is attempted, and `/tmp/restore.log` does not exist

#### Scenario: A line break in the target's password

- **WHEN** a member restores a completed MariaDB backup into a target whose password contains a line break
- **THEN** the response is `400` with `code` `invalid_request` naming the target's password, the backup has no new restore, and no connection is attempted

#### Scenario: A target database name with a dash inside it

- **WHEN** a member restores a completed MySQL backup into a reachable target whose database name is `orders-archive`
- **THEN** the response is `202` with a restore in `IN_PROGRESS`, reading it later shows `COMPLETED`, and the backed-up data is in `orders-archive` on the target
