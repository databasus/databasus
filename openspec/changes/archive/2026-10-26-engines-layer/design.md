## Context

Why the change is needed is in proposal.md, section Why. This document covers what shapes the approach. Paths are under `backend/internal/` unless they start with another top-level directory. `DDB` stands for `features/databases/databases`.

**Engine models.** Five GORM models carry the engine knowledge:
- `DDB/postgresql/logical/model.go` (1002 lines);
- `DDB/postgresql/physical/model.go` (1163);
- `DDB/mysql/model.go` (621);
- `DDB/mariadb/model.go` (634);
- `DDB/mongodb/model.go` (679).

Each one mixes three kinds of code:
- **Persistence:** gorm tags, `TableName`, the `BeforeSave`/`AfterFind` hooks that turn list fields into CSV columns through `util/namelist`, and the `BeforeCreate` of the physical model that mints `ReplicationSlotName` (`physical/model.go:141`).
- **Old application rules:** `Validate`, `ValidateUpdate`, `Update`, `CopyForNewDatabase`, `HideSensitiveData`, `EncryptSensitiveFields`.
- **Engine behavior:**
  - `TestConnection`, `GetRawDbSizeMb` and `GetClusterSizeMb`;
  - `PopulateVersion` and `PopulateDbData`, which connect and write the detected version, privileges, system identifier and WAL segment size into the model;
  - the `ShouldSuggest*` and `Create*User` methods;
  - for MongoDB, `BuildConnectionURI` and `BuildRestoreURI`;
  - for physical PostgreSQL, `TestReplicationConnection`, `ProbeLiveness`, `VerifyWalSlot`, `DropWalSlot`, `DropWalSlotForRemoval` and `OpenInspectionConn` (`physical/model.go:306-855`).

The helper files beside the models are engine behavior already: `privileges.go`, `dumpscope.go`, `timescaledb.go`, `connection.go`, `size.go`, `forced_wal_rotation.go`, `mysqlfamily/{grants,privileges,reader}.go` and `postgresql/shared/{credentials,embedded_target,ssl,connection_error}.go`.

**Secrets.** Every connecting method takes `encryption.FieldEncryptor` and decrypts the password, the TLS client certificate, key and root certificate, and the SSH secrets itself. A nil encryptor means the values are in clear text, which is how a restore target arrives (`postgresql_shared.DecryptFieldIfNeeded`, `shared/credentials.go:223`; `decryptPasswordIfNeeded` in `mysql/model.go:613`, `mariadb/model.go:626` and `mongodb/model.go:671`).

**User provisioning.** These methods are long:

| Engine | `ShouldSuggest*` | `Create*User` |
|---|---|---|
| logical PostgreSQL | 293-443 (151 lines) | 445-893 (449) |
| physical PostgreSQL | 604-756 (153) | 760-851 (92) |
| MySQL | 347-429 (83) | 431-513 (83) |
| MariaDB | 332-415 (84) | 417-499 (83) |
| MongoDB | 295-479 (185) | 481-545 (65) |

Each `Create*User` generates the name `databasus-<8 hex>` (MariaDB `pgs-<8 hex>`, because MariaDB 5.5 allows 16 characters) and a password from `encryption.GenerateComplexPassword` (`util/encryption/password_generator.go:14`). It retries three times. Logical PostgreSQL retries when the error text is not empty (`logical/model.go:526`), MySQL, MariaDB and MongoDB retry on any error, and only physical PostgreSQL retries on "already exists" (`physical/model.go:801`). Each attempt defers its rollback inside the loop. Error text is compared at `logical/model.go:552`, `physical/model.go:801`, `:1080-1105` and in `classifyReplicationConnectError` (`physical/model.go:1141-1163`).

**Contexts.** `TestConnection`, `PopulateVersion` and `PopulateDbData` of every engine start from `context.Background()` (for example `logical/model.go:130,271`, `mysql/model.go:83,269,318`).

**Tunnels.** Each engine has an `OpenTunnel(ctx, OpenTunnelSpec{Database, Logger, Encryptor})` (for example `physical/tunnel.go:27-67`). It refuses the embedded PostgreSQL target first, opens `sshtunnel.Open`, and returns a shallow copy of the model that points at the local port. PostgreSQL keeps `Host` for TLS and `.pgpass` and sets `LocalTunnelEndpoint`, which becomes `PGHOSTADDR`. MySQL and MariaDB rewrite host and port. MongoDB forces a direct connection. `CopyDiscoveredMetadataToOriginal` copies the detected values back to the original model. `features/databases/tunneled_database.go` wraps the per-engine tunnels behind a switch on the database type.

**`util/tools`.** The package holds the client tool lookup and the startup check:
- `AssetsToolsDir` is a `sync.OnceValue` that walks up from the working directory to `assets/tools/<x64|arm>` and panics (`paths.go:32`);
- `executionVerdicts` is a package `sync.Map` with the `--version` verdict per bin directory (`common.go:32`);
- `runClients` starts each client with a bare `exec.CommandContext` (`common.go:120`);
- `LogAndExitIfClientToolsBroken` exits through `logger.ExitAfterFlush(1)` (`common.go:197`) and is called from `App.Serve` (`app/server.go:35`);
- `ClientToolsHealthError` is called from `features/system/healthcheck/service.go:46`.

Its version enums (`PostgresqlVersion`, `MysqlVersion`, `MariadbVersion`, `MongodbVersion`), the executable lookups, the downgrade comparisons and the password escaping are imported by 10 files of `features/databases`, 8 of `features/backups`, 5 of `features/restores`, `internal/app` and 39 test files. `util/namelist` is imported by the engine packages and the four logical backup use cases.

**`features/sshtunnel`.** It holds the stored `Config` with gorm tags and `Validate`, `Update`, `HideSensitiveData` and `EncryptSensitiveFields`. The rest is the connection code: `Open`, `Forwarder`, the authentication, the keepalive and the reachability probe. The connection code decrypts with `util/encryption` and falls back to `logger.GetLogger()` when no logger is given (`forwarder.go:85`).

**Process groups.** Nothing in the backend starts a tool in its own process group. `pg_basebackup.go:119-123` and `receivewal_process.go:240-263` send SIGINT to the single process on cancel and set `WaitDelay`. The logical dump and restore use cases rely on `exec.CommandContext` and `WaitDelay`.

**Tests.** The engine tests start their own containers through `util/testing/containers` and name every object with a UUID suffix. The exception is `DDB/postgresql/physical/model_test.go`, which reaches the shared compose servers through `config.GetEnv()` (lines 40, 46, 61, 86, 902, 1467) and creates slots `databasus_test_<suffix>` that `cmd/cleanup_test_db` does not remove.

**Rules in force.**
- `docs/architecture.md`, "Engines and providers" and "Backend layout": engines know nothing of modules, tables or HTTP, and the agent imports only `lib`, `engines` and its client.
- "Long-running work": external tools run through `lib/process`.
- `backend/.golangci.yml:148` already lists `internal/engines` in the target layout, and `tools/archlint/layers.go:12-38` already orders `lib < engines < modules`.
- `depguard` keeps Gin and GORM out of the agent only (`.golangci.yml:107-109`).
- The change answers to the root `AGENTS.md` and `backend/AGENTS.md`.

## Goals / Non-Goals

**Goals:**

- Engine behavior lives in `internal/engines` and passes every rule of the target layout. `databases-module` and the verification agent can call it without the old models.
- The old code keeps its behavior and its callers, so this diff is reviewable on its own.
- `lib/process` exists with one real user and with the semantics the dump, restore and WAL tools will need.

**Non-Goals:**

- Removing the old engine models. `databases-module` replaces them with provider rows of the module.
- Changing what any engine operation does, apart from the retry and the error matching of decision 5.
- Moving dump and restore into engines.

## Decisions

### 1. The packages

```
internal/engines/
  postgresql/
    shared/       settings, TLS, credential files, the embedded-target guard, the client tools
                  of PostgreSQL, connection errors (imported as pgshared)
    logical/      logical source: connect, version, size, TimescaleDB, dump scope privileges,
                  read-only user (pglogical)
    physical/     physical source: replication connection, cluster identity, size, liveness,
                  replication slots, inspection connection, forced WAL rotation, replication
                  user (pgphysical)
  mysqlfamily/    grants, privilege reading and checks shared by MySQL and MariaDB
  mysql/          MySQL source and client tools
  mariadb/        MariaDB source and client tools
  mongodb/        MongoDB source, connection URIs and client tools
  clienttools/    bundle descriptions, release lines, the bundle root and the checker
internal/lib/process/    starting external tools
internal/lib/sshtunnel/  SSH tunnels
```

`importas` fixes the aliases `pgshared`, `pglogical` and `pgphysical`, because three packages named `shared`, `logical` and `physical` would otherwise clash with each other and with the old packages in the same files.

`clienttools` is not an engine. It holds what the four engines' client tool sets have in common, the same way `mysqlfamily` holds what MySQL and MariaDB share. Task 7.1 adds it to the engine list in `docs/architecture.md`.

*Rejected: put the client tool check in `internal/app`.* The engines need the bundle descriptions and the release-line comparison to pick an executable, so the check would split one concept between two layers.

### 2. An engine takes plain settings and returns what it found

Each source kind has a settings struct with plain fields and no tags:
- `pglogical.Settings` holds host, port, user name, password, database, `pgshared.TLS` and an optional `*sshtunnel.Settings`;
- `pgphysical.Settings` holds the same connection fields plus the replication slot name;
- `mysql.Settings`, `mariadb.Settings` and `mongodb.Settings` hold their own connection fields.

Every secret in them is in clear text. The behavior hangs off a source value built from the settings, for example `pglogical.NewSource(settings)`. Its methods take the caller's `ctx` and a `*slog.Logger`.

Detection returns values instead of writing into a model:
- `DetectVersion(ctx, logger) (pgshared.Version, error)`;
- `pgphysical.Source.InspectCluster(ctx, logger) (ClusterIdentity, error)`, which returns the version, the system identifier and the WAL segment size;
- `mysql.Source.Inspect(ctx, logger) (Inspection, error)`, which returns the version and the privileges, and the same for MariaDB.

The old model assigns what it gets. Version enums move with their engine: `pgshared.Version`, `mysql.Version`, `mariadb.Version` and `mariadb.ClientVersion`, `mongodb.Version`, with the same string values, so the columns and the JSON do not change.

Options a dump or a restore needs are not settings. `IsExcludeExtensions`, `IsRestoreOwnership` and `IsRestorePrivileges` stay on the old model, which still serves as the restore request, and the dump scope (`IncludeSchemas`, `ExcludeTables`, `ExcludeTableData`) is a parameter of the privilege check that needs it.

*Rejected: keep the GORM structs and move them into `engines`.* `engines` would know tables, and `databases-module` could not give each provider its own table.

*Rejected: let an engine take the `FieldEncryptor`.* The verification agent has no master key, so the engine would need a second path for clear text, as the nil encryptor is today. Decrypting at the caller also keeps the rule of `docs/architecture.md` "Data" that a secret is decrypted where it is used: the old model decrypts immediately before the call.

### 3. The old models delegate

The old models in `DDB/*` keep their fields, tags, hooks, JSON, validation, encryption and copying. They gain `decryptSettings(encryptor)`, which decrypts the secrets (a nil encryptor passes them through, as today) and builds the engine settings. Each engine method of the old model becomes a call into the engine, and each detection writes the returned values into the model. The old method signatures stay, apart from a `ctx` added to `TestConnection`, `PopulateVersion` and `PopulateDbData`, which start from `context.Background()` today. The old tunnel wrappers keep `TunneledDatabase` and `CopyDiscoveredMetadataToOriginal` and open the tunnel through the engine.

The callers in backups, restores, health checks, verification, telemetry and the old tests keep calling the old models. They change in two ways:
- where they import a moved helper: `postgresql_shared` credential files, `tools.Get*Executable`, the version enums and comparisons, the password escaping and `namelist.NormalizeUniqueNames`;
- where they call one of the three methods that gain `ctx`, and pass their own: the old service, `restores/service.go:340`, `restores/restoring/restorer.go:143` and `backups/backups/usecases/physical/postgresql/testing.go:242`.

*Rejected: switch every caller to the engines now.* The callers hold `*databases.Database` and would build engine settings from it in some 40 places, and `databases-module` changes the same places again when the old models go. Delegation keeps this change to the engines and lets the module change the callers once.

### 4. The client tools

`clienttools` declares:

```go
// Bundle is one client tool set an engine ships: where it lives, what it must hold and how its
// clients report their version.
type Bundle struct {
	Engine           string
	Version          string
	BinDir           string
	RequiredCommands []string
	ReleaseLine      string
	ParseVersion     func(output string) (string, error)
	IsFatal          bool
}

// Checker runs every client of every bundle once and keeps the verdict per bin directory.
type Checker struct { /* bundles, process runner, verdicts under a mutex */ }

func FindBundleRoot(startDir string) (string, error)
func NewChecker(bundles []Bundle) *Checker
func (c *Checker) Check(ctx context.Context) []Result
func (c *Checker) GetHealthError(ctx context.Context) error
```

Each engine offers `GetClientToolBundles(root string) []clienttools.Bundle` and a lookup of its executables under `root`. For PostgreSQL that is `pgshared.ClientTools{Root}.GetExecutable(version, executable)`, and the other engines follow the same shape. `FindBundleRoot` is today's walk up from the working directory, and it returns an error instead of panicking. The architecture key (`x64`, `arm`) is part of it.

`internal/app` calls `FindBundleRoot` with the working directory while it builds the environment. When the root is missing, `Serve` refuses to start with the reason, while the console commands, which run no client tool, still work, as they did when `util/tools` resolved the root lazily. It stores the root in the old configuration as `ClientToolsDir`, so the old code reads it through `config.GetEnv()`, as the old code reads every setting today; the test loader of the old configuration does the same. `app.New` builds one `Checker` over the bundles of the four engines. `Serve` runs `Check`, logs each result as `LogAndExitIfClientToolsBroken` does, and returns an error when a fatal bundle failed, so `cmd/databasus` exits non-zero instead of `util/tools` exiting. The old system health check receives the checker through an install function, the way the bridges receive modules, and calls `GetHealthError`. That call reads the files again and reuses the verdicts, as the `client-tool-bundles` requirement "A client that cannot run is reported at startup" asks.

The verdicts move from the package `sync.Map` into the checker, so every test builds its own.

### 5. User provisioning is split into steps

`CreateReadOnlyUser` of logical PostgreSQL becomes named steps inside `pglogical`:
1. check that the connecting user may create roles;
2. refuse when row-level security is enabled on a table the new role would read, with the message that asks for a role with `BYPASSRLS`;
3. create the role with its password;
4. revoke `CREATE` on schema `public` from `PUBLIC`, and log the warning when that is not permitted;
5. grant connect on the database and revoke `TEMP`;
6. grant usage on the schemas in scope and revoke `CREATE` on them;
7. grant read access to existing tables and sequences;
8. set the default privileges for future objects, the role's own and, best effort, those of every other owner in scope;
9. verify that the new role exists before the commit.

The code before this change has no TimescaleDB grants in this method and verifies the role with a `pg_roles` lookup rather than a login, so the steps keep exactly that.

Each attempt is a function of its own that opens its transaction and rolls it back with its own `defer`, as each attempt opens its own transaction today (`logical/model.go:505`, `physical/model.go:782`). A failed `CREATE ROLE` aborts the PostgreSQL transaction it ran in, so a retry needs a fresh one. The other engines' `Create*User` and every `ShouldSuggest*` get the same treatment where a method holds more than one step. The behavior of each step and the privileges it grants do not change.

A user name is generated by one function per engine, as today. Creation retries up to three times only when the server says the name is taken:
- PostgreSQL: SQLSTATE `42710` (`duplicate_object`), from `pgconn.PgError`;
- MySQL and MariaDB: error 1396 (`ER_CANNOT_USER`), from `mysql.MySQLError`;
- MongoDB: error code 51003 (`UserAlreadyExists`), from `mongo.CommandError`.

Any other error ends the creation on the first attempt.

Errors are matched by code where the driver has one. The messages that share a code, such as the several causes of `42501` in `classifyReplicationConnectError` and the "permission denied" and "must have" checks of the replication grants, are matched by text in one function per engine, `classifyServerError`, next to that engine's errors in `errors.go`. MySQL and MariaDB create the user with the same statements, so the attempt loop, the retry on 1396 and their `classifyServerError` live once in `mysqlfamily`, and each engine keeps only its name generator and its log line. `ConnectionTestError` and its codes move from `connection_error.go` into `pgshared/errors.go`, as the errors file rule asks.

*Rejected: keep retrying on any error.* A refused permission or an unreachable host fails the same way three times and triples the time the user waits for the answer. A taken name is the only failure a new name fixes.

### 6. `lib/process`

```go
type Spec struct {
	Path   string
	Args   []string
	Env    []string
	Dir    string
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
	// StopSignal is sent to the process group when ctx ends; nil sends SIGKILL at once.
	StopSignal os.Signal
	// KillAfter is how long the group may take to exit after StopSignal before SIGKILL.
	KillAfter time.Duration
}

// Running is a started tool and its process group.
type Running struct { /* the command and its group */ }

func Start(ctx context.Context, spec Spec) (*Running, error)
func (r *Running) Wait() error
func Run(ctx context.Context, spec Spec) error
func CombinedOutput(ctx context.Context, spec Spec) ([]byte, error)
```

`Start` sets `Setpgid` so the tool and its children form their own group. It sets `Pdeathsig` to `SIGKILL`, so the leader dies with the backend. Its `Cancel` signals the negative process ID, which reaches the whole group. `WaitDelay` is `KillAfter`, after which `exec` closes the pipes and the helper sends `SIGKILL` to the group. `Wait` returns the context's error when the context ended the process, so a caller can tell a cancellation from a tool failure.

The client tool checker runs each `--version` through `CombinedOutput` with a 10-second deadline, as `clientVersionTimeout` does today. The dump and WAL tools move to the helper in milestones 9 and 10. They need exactly what the spec offers: SIGINT with a grace period for `pg_basebackup` and `pg_receivewal`, and closed pipes after a delay for the dumps.

`forbidigo` gets the patterns `^os/exec\.(Command|CommandContext)$`, with the message "start external tools through lib/process". Two exclusions apply:
- `internal/lib/process`, the helper itself;
- `_test.go` files. They start the binary under test (`cmd/databasus/binary_test.go:43,278`) or re-execute the test binary (`lib/testdb/testdb_test.go:164`), not an external tool that a cancelled operation could strand.

The rule applies to the target layout only, so the old dump and restore code keeps its calls until it moves.

*Rejected: the `exec.CommandContext` default, which kills only the leader.* `pg_dump -j`, `mysqldump` behind a pipe and `mongodump` with several collections start children or share pipes with them, and a killed leader leaves them running with the database connection open.

### 7. `lib/sshtunnel`

`Open`, `Forwarder`, the authentication, the keepalive and the reachability probe move to `lib/sshtunnel` without behavior changes. `OpenSpec` takes `Settings` with the private key, its passphrase and the password in clear text, and a required logger. The `logger.GetLogger()` fallback goes, and so does the `util/encryption` import. The keepalive and the accept loop keep their bare `go` statements, which `lib` may hold.

`features/sshtunnel` keeps the stored `Config` with its gorm tags and its four methods, plus `DecryptSettings(encryptor) (sshtunnel.Settings, error)`. `internal/util/testing/bastion` and the old test helpers switch to the new package. `databases-module` deletes what is left.

### 8a. The list helpers

`util/namelist` moves to `lib/namelist` instead of into the engines that use it. Logical PostgreSQL, `mysqlfamily` and MongoDB all normalize the lists of object names a user enters and shorten long name lists in error messages, so a copy per engine would hold the same code three times. The helpers know nothing of any engine, which is what `lib` is for.

### 8. The password generator

`GenerateComplexPassword` moves to `lib/crypto` as `GeneratePassword`, with the same length and character classes. The callers are the five provisioning steps, which now live in the engines.

### 9. Engine tests

The engine tests move from `DDB/*` into the engine packages and test the source values instead of the old models. They start their containers through `util/testing/containers`, as the target-layout tests of `storages` already do. The physical PostgreSQL tests stop reading `config.GetEnv()` and start their servers through `containers.StartPhysicalPostgres`. A test in the target layout builds without the `testhelpers` tag, so the old configuration is not loaded there. Every role, database and slot keeps a UUID suffix, so the tests run in parallel with each other and with other worktrees.

The tests of the old models that remain in `features/databases` keep checking what stays there: validation, copying, hiding secrets, the tunnel wrapper and the delegation, with one connecting test per engine that shows the old model reaches its engine.

New tests cover what this change adds:
- `lib/process`: a cancelled context kills a child the tool started, a stop signal with a grace period lets the tool exit on its own, and `Wait` reports the cancellation;
- `clienttools`: a bundle root found from a nested directory, a missing root reported as an error, a verdict reused by the health check;
- the retry of decision 5: a name collision is retried and a refused permission is not.

### 10. Lint

- `depguard` gets a rule `engines` that denies `gorm.io` and `github.com/gin-gonic/gin` in `internal/engines/**`, beside the agent's rule.
- `forbidigo` gets the process patterns of decision 6, for production files only.
- `docs/architecture.md` "Automatic checks" names both, and `backend/AGENTS.md` keeps the `lib/process` rule with one sentence that the linter enforces it.
- `docs/glossary.md` gets "Client tool bundle" (`clienttools.Bundle`: the vendored client tools of one engine version, their directory and the commands they must hold) and "Engine source" (`Source` of an engine package: an engine's handle on one user database, built from plain connection settings) in its Code section.

## Risks / Trade-offs

- [The old models and the engines describe the same connection twice until `databases-module`] → The old model builds the engine settings in one function per engine, and the tests of decision 9 show each old model still reaches its engine.
- [A read-only user that failed for a transient reason was retried before] → A transient failure such as a dropped connection now reaches the user on the first attempt, and the user retries by clicking again. A taken name, the failure the retry exists for, is still retried.
- [The physical engine tests start containers instead of using the shared servers] → They run longer, by the start of one container per version and package. They no longer leave `databasus_test_*` slots on the shared servers.
- [About 60 files change imports] → The changes are mechanical and the compiler finds every one of them.
- [`Pdeathsig` is per thread on Linux] → It fires when the thread that started the process exits, not only when the process does. `Start` locks the OS thread for the fork only, which Go's `exec` already does, and the backend never ends the threads that start tools. The group kill on cancel does not depend on it.

## Migration Plan

Nothing to migrate: no schema, contract or configuration changes. The startup check and the health check behave as before. A PostgreSQL bundle that fails the startup check still stops the process with exit status 1, now through `Serve`'s error.
