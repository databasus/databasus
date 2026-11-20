## Why

The connection settings of a MySQL or MariaDB database come from the user, and today three of the places that pass them to software on the Databasus host let a crafted value become an option of that software:

- The database name is the last, bare argument of `mysqldump`, `mariadb-dump`, `mysql` and `mariadb`. A name such as `--result-file=/path` is read as an option, so a backup or a restore writes or truncates any file the Databasus process can write.
- The Go driver's connection string is assembled with `fmt.Sprintf` and no escaping. A database name such as `x?allowAllFiles=true&a` turns on the driver's `allowAllFiles`. A server the user controls can then ask the driver for any file the Databasus process can read, the master key included.
- The `.my.cnf` file the client tools read is written line by line, and the user name, the host and the password may contain a line break. Any text after the break becomes a client option, such as `result-file`, so the same file write is reachable through those fields too. A run of the bundled `mysqldump` confirmed it for the user name and the password.

Whoever may add a database or start a restore reaches all three, and starting a restore needs only the viewer role. Restore target settings come straight from the request and skip the checks a saved database gets. A workspace member must not be able to read or write files on the host: it holds the master key, the credentials of every workspace and the data of the internal database.

## What Changes

- The database name of a MySQL or MariaDB database or restore target is refused when it starts with `-`. Host, user name, password and database name are refused when they contain a line break or another control character. A saved database gets `400` `invalid_request` naming the setting, and a restore gets the same naming the target's setting. Validation lives in the engine, so creation, change, every operation on unsaved settings, a restore target and a backup of stored settings all go through it.
- The dump tools and the network compression probe that runs before each dump get `--` before the database name. The `mysql` and `mariadb` clients get the name as `--database=<name>`, because the bundled `mysql` 8.4, 9 and 26 keep parsing options after `--`.
- The Go driver is opened from a driver configuration structure instead of a connection string, so no field can add driver parameters. The process-wide TLS registration this removes goes with it.
- The `.my.cnf` writer refuses a value with a control character itself, so a stored value that predates validation fails the run and is never written into the file.
- **BREAKING**: a saved MySQL or MariaDB database whose name starts with `-`, or whose host, user name, password or database name holds a control character, can no longer be backed up or saved until the value is changed. The backup fails with an error naming the setting. No such value could have worked as intended. The old string-building code and the global TLS registration are deleted, not kept beside the new code.
- The contract's description of these fields states the rules.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `databases`: a new requirement that MySQL and MariaDB connection settings never become options of the host's tools or driver, and the refusals that follow from it.
- `restores`: a new requirement that a MySQL or MariaDB restore target is checked by the same rules before anything connects.

## Impact

- Engines: `backend/internal/engines/mysql`, `backend/internal/engines/mariadb`, `backend/internal/engines/mysqlfamily`: settings validation, the dump and restore argument builders, the driver connection and the `.my.cnf` writer.
- Modules: the MySQL and MariaDB providers of `databases` call the engine validation; the logical backup pipeline and `restore` of `backups` refuse an invalid target. The PostgreSQL TLS refusal becomes one case of a general invalid-target-setting refusal.
- Contract: descriptions in `backend/api/openapi.yaml` for the MySQL and MariaDB settings and restore targets, then `mise run generate-api`. Request shapes do not change.
- The verification agent does not use the MySQL family engines and is not affected.
- The change answers to the root `AGENTS.md` and `backend/AGENTS.md`.

## Out of scope

- PostgreSQL and MongoDB connection handling. PostgreSQL quotes every connection value. MongoDB puts its host into the connection URI without escaping, which needs its own look.
- The SQL that grants a limited user rights quotes the database name in backticks without escaping a backtick inside it. It runs on the user's own server with the user's own credentials and does not reach the host.
- Who may start a restore.
- An identifier-shaped rule for database names. MySQL allows almost any character in a quoted name, and the fix does not rely on the shape of the name.
- Frontend changes. The forms show a field refusal from the server as they do today.
