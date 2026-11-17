## Context

See proposal.md, Why, for the three injection paths. This section records where the values travel today.

- **Arguments.** The database name is the last bare argument in `backend/internal/engines/mysql/dump.go:88`, `backend/internal/engines/mysql/restore.go:40`, `backend/internal/engines/mariadb/dump.go:85` and `backend/internal/engines/mariadb/restore.go:63`. The network compression probe that runs before every dump passes it bare as well, followed by a table name (`backend/internal/engines/mysqlfamily/network_compression.go:68`). Host, port and user travel as `--host=`, `--port=` and `--user=`, one argument each, so a leading dash in them is harmless. `--ignore-table=<database>.<table>` is one argument as well.
- **Driver.** `buildDSN` in `backend/internal/engines/mysql/source.go:157` and `backend/internal/engines/mariadb/source.go:151` pass user name, password, host and database name through `fmt.Sprintf` with no escaping. The driver's parser (go-sql-driver v1.10.0, `ParseDSN`) splits at the last `/` and then at the first `?`, so a `?` in the database name starts the parameter list. To encrypt without verifying the certificate, `mysqldriver.RegisterTLSConfig` registers a process-wide named configuration in each engine: `mysql-skip-verify` at `backend/internal/engines/mysql/source.go:165` and `mariadb-skip-verify` at `backend/internal/engines/mariadb/source.go:158`.
- **Option file.** `writeClientConfigFile` in `backend/internal/engines/mysqlfamily/client_config.go:44` writes `user=` and `host=` unquoted, and the password quoted with only `\` and `"` escaped. A line break in any of them starts a new option line in `[client]`. The dump and restore commands also pass `--host=` and `--user=` in their arguments, but the compression probe does not: it takes host and user from this file alone. With the bundled `mysqldump` 5.7, 8.4 and 26, a user name `x\nresult-file=<path>` created `<path>`. A password with a line break created a file as well, named with a trailing `"`.
- **Validation.** The MySQL provider of `databases` checks only that host, port, user name and password are present (`backend/internal/modules/databases/internal/providers/mysql/provider.go:51`). MariaDB has the same check. Creation, change and every operation on unsaved settings call it: `backend/internal/modules/databases/internal/userdatabase/service.go:171` and `:451`, and `ResolveSettingsTarget` in `settings_target_service.go:52`, which serves the connection test and the limited user check and creation (`backend/internal/modules/databases/internal/limiteduser/service.go:57`, `:90`). Backups run from stored settings and call no validation. A restore target bypasses it: `StartRestore` authorizes with `ActionView` (`backend/internal/modules/backups/internal/restore/service.go:126`) and passes the request's settings to `OpenRestoreTarget`, which opens the tunnel and connects (`backend/internal/modules/backups/internal/pipeline/mysql_engine.go:43`). The one existing target check is PostgreSQL's TLS check (`backend/internal/modules/backups/internal/pipeline/postgresql_engine.go:55`), refused as `InvalidTargetTLSError` (`backend/internal/modules/backups/internal/pipeline/errors.go:40`) and mapped to a field problem on `target.sslMode` (`backend/internal/modules/backups/internal/restore/errors.go:40`).

A run of the bundled clients against a closed port (assets/tools/x64) showed how each one treats `--`:

| Client | `-- --result-file=<path>` / `-- --tee=<path>` | `--database=--tee=<path>` |
|---|---|---|
| `mysqldump` 5.7, 8.0, 8.4, 9, 26 | treated as a name, no file | not applicable |
| `mariadb-dump` 10.6, 12.3, 13.0 | treated as a name, no file | not applicable |
| `mysql` 5.7, 8.0 | treated as a name, no file | treated as a name, no file |
| `mysql` 8.4, 9, 26 | still parsed as an option: `--foo` fails as an unknown option, `--tee=` as an unknown variable | treated as a name, no file |
| `mariadb` client | not run: it needs the libedit shim on the host | not run |

The constraints come from the root `AGENTS.md` (errors through `apierror`, naming, one unit per file, no backward-compatibility shims) and `backend/AGENTS.md`: engine knowledge stays in `internal/engines/<engine>`, `mysqlfamily` holds what MySQL and MariaDB share, and every error of a package sits in its `errors.go`. `docs/architecture.md`, Engines, says a restore target has settings but no database row, which is why its check cannot come from the `databases` provider registry.

## Goals / Non-Goals

**Goals:**

- Each place that hands a value to host software is safe by construction, whatever validation lets through.
- One validation, owned by the engines, gives a clear field refusal on every entry: create, change, every operation on unsaved settings and the restore target. A backup of stored settings gets an error that names the setting.

**Non-Goals:**

- Constraining the shape of a database name beyond what the injection paths need.
- Changing who may start a restore.

## Decisions

### Each sink is safe on its own; validation adds a clear error

There are three kinds of sink. Dump tools and the compression probe get `--` before the name. Clients get `--database=<name>`. The driver gets a configuration structure. The option file writer refuses a value with a control character. Validation sits on top and turns a bad value into a field refusal before any connection. The engine also runs it on its own entries, so a stored value fails with an error that names the setting.

- *Alternative: validation alone, as the report suggested.* Rejected. A later relaxation of the rule, or a missed call on some new path, would reopen the file write at once.
- *Alternative: sinks alone.* Rejected. A dash-led name would then fail only when the tool runs, with a tool error instead of a refusal that names the field.

### The `mysql` and `mariadb` clients take the name as `--database=`; the dump tools take `--`

The run above shows that `--` does not stop option parsing in `mysql` 8.4 and later. A value glued to its flag in one argument is never parsed as an option, by any version. The dump tools have no flag of the same meaning: `--databases` changes the dump to add `CREATE DATABASE` and `USE`, which would break a restore into a target of another name. For the dump tools `--` works in every shipped version. The probe calls `mariadb-dump` or `mysqldump` with a database and a table name, and `-- <database> <table>` keeps both positional.

- *Alternative: `--` everywhere.* Rejected by the run: `mysql` 8.4, 9 and 26 still parse `--tee=` after it. Today the client refuses it, but that only holds until some option it does accept is used.
- *Alternative: `--database=` for the dump tools too.* Rejected because no such flag exists for `mysqldump` and `mariadb-dump`.

### The driver is opened from `mysqldriver.Config` through `NewConnector`

`buildDSN` in both engines is replaced by one shared `mysqlfamily.OpenSingleSessionDB`, which takes a `DriverConnection` struct and builds a `*mysqldriver.Config` from `mysqldriver.NewConfig()`: `User`, `Passwd`, `Net: "tcp"`, `Addr` from `net.JoinHostPort`, `DBName`, `Timeout`, `ParseTime` and the `utf8mb4` charset through `Apply(mysqldriver.Charset(...))`. MySQL keeps `AllowCleartextPasswords` with encryption, as today, through the struct's `IsCleartextPasswordAllowed`. Encryption without verification sets the `TLS` field to a `*tls.Config` with `MinVersion: TLS12` and `InsecureSkipVerify`, which removes the `RegisterTLSConfig` call and its process-wide name. It opens the pool with `sql.OpenDB(connector)` and the one-session limits both engines' `openDB` set before, so the two copies become one. `AllowAllFiles` stays `false`, and no string is parsed on the way to the driver.

- *Alternative: escape the database name with `url.PathEscape` in the format string.* Rejected. The user name, password and host would stay unescaped, and every new field would need its own escaping. The structure has no separators to break.
- *Alternative: `cfg.FormatDSN()` and `sql.Open`.* Rejected. The connector skips a format and parse round trip that brings nothing.

### Validation is an engine method, shared through `mysqlfamily`

`mysqlfamily` gets a `ConnectionValues` struct with `Host`, `Username`, `Password` and `Database` fields and a `Validate()` method on it, so four strings never travel as adjacent parameters. It returns an `InvalidSettingError` carrying the setting's name (`host`, `username`, `password`, `database`, the contract's field names) and the reason. `mysql.Settings` and `mariadb.Settings` get a `Validate()` method that builds `ConnectionValues` and calls it. Three kinds of caller run it:

- the `databases` providers, after their presence checks, mapping the error to `providers.NewInvalidSettingError` with the same name;
- the pipeline's MySQL and MariaDB `OpenRestoreTarget`, before opening the tunnel;
- the engine's own entries that reach host software: opening the driver connection, `Dump` and `Restore`. A stored value from before this change therefore fails the backup with an error naming the setting, before the driver or a tool starts.

Rules: the database name must not start with `-`; none of the four values may contain a rune below `0x20` or `0x7f`. Port is an integer and SSH tunnel settings never reach the client tools or the driver, so neither is checked here.

- *Alternative: validate in the `databases` providers only.* Rejected, because a restore target never passes through them, and a stored value would fail with an unknown-database error from the server instead of naming the setting.
- *Alternative: an identifier regex such as `^[A-Za-z0-9_$][A-Za-z0-9_$-]*$`.* Rejected. MySQL accepts almost any character in a quoted name, so a regex would refuse real databases, and the sinks no longer depend on the shape of the name.
- *Alternative: escape control characters in the option file instead of refusing them.* Rejected. Escaping would depend on how every shipped client version parses option file values, and a host, user name or database name with a line break is not a real value. A password with a control character is the one real loss, and it is accepted.

### The option file writer refuses control characters itself

`writeClientConfigFile` returns `ErrControlCharacterInClientConfig` when the host, user name or password holds a control character, before it writes anything. The engine entries validate first, so this is the second layer, kept next to the sink in case a new caller skips validation.

- *Alternative: rely on validation.* Rejected for the same reason as validation alone.
- *Alternative: drop the `user=` and `host=` lines, since dump and restore pass `--user=` and `--host=` in their arguments.* Rejected. The compression probe passes neither and takes host and user from the file, so the lines are needed. Passing them to the probe as arguments too would leave the password line, which still needs the check.

### One target-setting refusal replaces the PostgreSQL TLS refusal

`InvalidTargetTLSError` in the pipeline becomes `InvalidTargetSettingError{Setting, Err}`. PostgreSQL's TLS check reports `Setting: "sslMode"`, MySQL and MariaDB report the setting their validation names. `restore` maps it through `newInvalidTargetSettingError` to a field problem on `target.<setting>`. The old type and `newInvalidTargetTLSError` are deleted.

- *Alternative: a second error type next to the TLS one.* Rejected. Two types would do one job, and `restore` would grow a branch for each engine.

### The contract documents the rules in field descriptions

The descriptions of `host`, `username`, `password` and `database` in the MySQL and MariaDB settings and restore target schemas of `backend/api/openapi.yaml` state the rules. No `pattern` is added. A pattern for control characters is hard to read in the contract, it would duplicate the engine rule, and the request validator would refuse before the engine with a less precise message.

## Risks / Trade-offs

- [A saved database with a dash-led name or a control character stops backing up] → No such value could work as the user meant: the tool read the name as an option, and a line break broke the option file. The engine's own validation fails the backup with an error naming the setting, and the user fixes the value in the form, where validation now explains it.
- [`mariadb` client behavior after the change was not run on this host] → The real-binary test runs every shipped MariaDB client under the libedit shim that `docs/testing.md` prescribes, so the apply phase proves `--database=` for it.
- [A future client version parses `--database=<value>` differently] → The real-binary test covers every shipped client version and runs again whenever a bundle is added or refreshed.
- [Dropping the named TLS registration changes how encryption is configured] → No engine test connects with encryption on today. The change adds one per engine against a real server that offers TLS: MySQL 8 generates its own certificate, and the MariaDB test uses a version that does the same.

## Migration Plan

No data migration. Stored values stay as they are, and invalid ones fail as described above. Rollback is a revert of the change.
