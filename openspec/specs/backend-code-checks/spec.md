# Backend Code Checks Specification

## Purpose

Defines which rules of the backend architecture and code style a machine enforces, how those rules apply throughout the backend, and what the production server binary may contain.

## Requirements

### Requirement: The backend lint enforces the documented rules throughout the backend
The backend lint SHALL check these rules, taken from the architecture document and, for suppressions, from the repository's security rules, on production and test files alike, throughout the backend, including every command, the composition root, domain modules, engine drivers, shared support, the verification agent and the architecture tools:

- no fixed sleeps or eventual assertions, no printing to standard output, no process exit, no environment reads and no background contexts outside the places the architecture document allows;
- database access only in repository files;
- log calls with attributes only, context-aware methods, constant messages and `snake_case` keys;
- test libraries only in test code, and no HTTP framework or ORM in the verification agent or in the engine drivers;
- every field listed in a dependency struct literal;
- every value handled in a switch on an enum;
- no package initialization functions or package-level application dependencies;
- no global variables without a documented, specific justification accepted by the lint;
- consistent import aliases;
- no lint suppression without a written reason.

On production files in the backend, the backend lint SHALL also check that external processes start only through the shared library's process helper, which ties a process to its operation and stops it with that operation. The helper itself is the only production code that starts a process directly. Test files are outside this rule, because they start the binary under test or the test binary itself rather than an external tool.

A violation SHALL fail the lint with the file, the line and the rule. The architecture document's permissions for particular operations in particular roles SHALL remain applicable. A package's location outside an enumerated set of migrated directories SHALL NOT exempt it from checks. Contract validation and generated-code freshness remain separate checks of the backend lint command.

#### Scenario: A sleep in a new test
- **WHEN** a test file of a module in the backend calls a fixed sleep
- **THEN** the backend lint fails and names the file, the line and the rule against fixed pauses

#### Scenario: A log call with a formatted message
- **WHEN** code in the backend logs a message built from a value instead of passing the value as an attribute
- **THEN** the backend lint fails and names the logging rule

#### Scenario: A dependency literal with a missing field
- **WHEN** code in the backend builds a module's dependency struct without one of its fields
- **THEN** the backend lint fails and names the missing field

#### Scenario: A suppression without a reason
- **WHEN** code in the backend suppresses a lint finding without a written reason
- **THEN** the backend lint fails

#### Scenario: An engine driver starts a client tool directly
- **WHEN** production code of an engine driver starts an external program without the shared library's process helper
- **THEN** the backend lint fails and names the process helper as the only way to start an external program

#### Scenario: A test starts the binary under test
- **WHEN** a test file in the backend builds and starts the server binary directly
- **THEN** the backend lint passes

#### Scenario: An engine driver imports the ORM
- **WHEN** a file of an engine driver imports the ORM or the HTTP framework
- **THEN** the backend lint fails and names the rule that engine drivers know no tables and no HTTP

#### Scenario: A global application dependency
- **WHEN** a backend package stores an application dependency in a package-level variable
- **THEN** the backend lint fails and identifies the variable

#### Scenario: A command breaks a backend rule
- **WHEN** a backend command other than the server or verification agent violates a documented rule that applies to commands
- **THEN** the backend lint fails under the same rule

#### Scenario: A justified fixed lookup table
- **WHEN** code uses a fixed shared lookup table with a specific lint suppression that explains why it is shared and is never mutated
- **THEN** the global-variable check accepts that declaration without exempting its package

### Requirement: The architecture check enforces structure rules
The backend lint SHALL include an architecture check for the rules the architecture document assigns to it: dependencies point from the entry points through the composition root and the modules to the engines and the shared library, never back; another package reaches a module only through the module's root package, and that root package holds only the module's contract file; methods of a handler, a service or a repository live in files of that role; a package's errors are declared in its errors file; a bare goroutine start appears only in the shared library; the verification agent imports only what the architecture document allows. These rules SHALL apply to production files throughout the backend, including every backend command. There SHALL be no migration-period directory exemption. Test files SHALL be exempt, because a module's API tests build the whole application through the test harness above the modules and start goroutines of their own.

#### Scenario: A module imports another module's internals
- **WHEN** a module imports a package inside another module's internal directory
- **THEN** the architecture check fails and names both packages and the boundary rule

#### Scenario: The shared library imports a module
- **WHEN** a package of the shared library imports a domain module
- **THEN** the architecture check fails and names the layer rule

#### Scenario: An error declared outside the errors file
- **WHEN** a module declares a sentinel error in its service file
- **THEN** the architecture check fails and names the file

#### Scenario: A second file in a module's root package
- **WHEN** a module's root package holds a file other than its contract file
- **THEN** the architecture check fails and names the file and the rule that the root package holds only the contract

#### Scenario: A service method in a handler file
- **WHEN** a method of a module's service type is declared in the module's handler file
- **THEN** the architecture check fails and names the method, the file and the file of the service role

#### Scenario: The agent imports a module
- **WHEN** a package of the verification agent imports a domain module
- **THEN** the architecture check fails and names the import and the list of packages the agent may import

#### Scenario: A goroutine started in a module
- **WHEN** a service in a module starts a goroutine with a bare go statement
- **THEN** the architecture check fails and names the shared library as the only place where a bare go statement is allowed

#### Scenario: A command imports the agent implementation
- **WHEN** a backend command other than the verification agent imports the agent implementation
- **THEN** the architecture check fails and identifies the forbidden dependency

#### Scenario: A package outside the former checked directories
- **WHEN** a backend production package outside the former migration directory list declares a sentinel error outside its errors file
- **THEN** the architecture check fails and identifies the declaration

#### Scenario: An unclassified package imports the agent
- **WHEN** a backend production package outside the documented layers imports the verification agent implementation
- **THEN** the architecture check fails and identifies that only the agent may import its implementation

#### Scenario: An unclassified package imports module internals
- **WHEN** a backend production package outside the documented layers imports another module's internals
- **THEN** the architecture check fails and identifies the module boundary

### Requirement: Migrations add no cascading foreign key to another module's table
The architecture check SHALL fail when a migration added after this change declares a foreign key with a cascading delete from one module's table to a table of another owner. A table has an owner when its name is a module name from the architecture document, or starts with a module name followed by an underscore, or starts with `lib_<name>_`; a shared library table counts as an owner of its own. A foreign key that involves a table without an owner SHALL NOT be checked, because tables that predate the module prefixes do not follow the scheme yet. Migrations that existed before this change SHALL NOT be checked either.

#### Scenario: A cascade across modules
- **WHEN** a new migration adds a foreign key with a cascading delete from a backups table to a storages table
- **THEN** the architecture check fails and names the migration, the constraint and the rule that cross-module keys restrict deletes

#### Scenario: A cascade to a shared library table
- **WHEN** a new migration adds a foreign key with a cascading delete from a backups table to the shared audit table
- **THEN** the architecture check fails, because the shared library table has another owner

#### Scenario: A cascade inside one shared library owner
- **WHEN** a new migration adds a foreign key with a cascading delete between two tables of the same `lib_<name>` owner
- **THEN** the architecture check passes

#### Scenario: A cascade from a table that predates the prefixes
- **WHEN** a new migration adds an engine table named after the engine that cascades to the databases table
- **THEN** the architecture check passes, because the engine table has no module owner

#### Scenario: A cascade inside one module
- **WHEN** a new migration adds a foreign key with a cascading delete between two storages tables
- **THEN** the architecture check passes

### Requirement: The production server binary contains no test code
No package that the server binary in the image links SHALL import the standard testing package, a test assertion or mocking library, or the test container library. The architecture check SHALL fail when a package in the dependency graph of the server's entry point imports any of them, and name the production package that pulls it in. Third-party packages that the server links for a storage backend and that import a test library in production code SHALL be accepted only when the architecture check lists them by name with the reason; any other importer SHALL fail the check.

#### Scenario: A production helper imports an assertion library
- **WHEN** a production file of any package reachable from the server's entry point imports an assertion library
- **THEN** the architecture check fails and names that package and the import

#### Scenario: A new third-party dependency imports a test library
- **WHEN** a dependency the server starts to link imports the standard testing package in a production file, and the architecture check does not list it
- **THEN** the architecture check fails and names that package and the import

#### Scenario: An accepted third-party importer
- **WHEN** a storage backend's dependency that the architecture check lists by name imports the standard testing package
- **THEN** the architecture check passes

#### Scenario: Shared test helpers
- **WHEN** a helper used by tests of several packages lives in a package that only test files import
- **THEN** the architecture check passes

### Requirement: Backend tests detect data races
Backend tests SHALL run with the race detector in CI. On a developer's machine every repository test command, including those OpenSpec change verification names, SHALL run them without it. A developer SHALL be able to run any backend test with the race detector on demand. A data race detected during a run with the race detector SHALL fail that test.

#### Scenario: A race in a test run
- **WHEN** two goroutines write the same variable without synchronization during a backend test that runs with the race detector
- **THEN** the test fails with the race report

#### Scenario: A local package run
- **WHEN** a developer runs `mise run test`
- **THEN** the backend packages run without the race detector

#### Scenario: OpenSpec change verification
- **WHEN** an agent runs the verification commands of an OpenSpec change
- **THEN** the backend packages run without the race detector

#### Scenario: A race reported by CI is reproduced locally
- **WHEN** a developer runs the test CI reported with the Go test command and its race flag
- **THEN** the test runs with the race detector

### Requirement: The verification agent binary links no server frameworks

No package that the verification agent binary links SHALL import the ORM, the HTTP framework or the migration tool of the backend server. The architecture check SHALL fail when a package in the dependency graph of the agent's entry point imports any of them, and name the production package that pulls it in. The agent restores into a throwaway database and talks to the backend as an HTTP client, so it has no tables, no migrations and no HTTP server, and a dependency on any of them only grows the binary and its attack surface.

#### Scenario: The agent imports an engine package that links the ORM

- **WHEN** a package of the verification agent imports an engine package whose dependencies include the ORM
- **THEN** the architecture check fails and names the package that imports the ORM and the import

#### Scenario: The agent imports the shared restore package

- **WHEN** a package of the verification agent imports the engine package that builds the restore command, which depends only on the standard library
- **THEN** the architecture check passes

#### Scenario: The server links the ORM

- **WHEN** the server's entry point links the ORM, the HTTP framework and the migration tool
- **THEN** the architecture check passes, because the rule covers only the agent's entry point

### Requirement: Independent subtests run in parallel
A backend test that runs in parallel SHALL run its subtests in parallel too, unless they depend on each other or on shared state. The backend lint SHALL fail, unless the test states a reason, in three cases: a parallel test none of whose direct subtests run in parallel, a test with parallel subtests that does not run in parallel itself, and a test with parallel subtests that releases resources through `defer` and calls no test cleanup. A stated reason SHALL name the shared state or the dependency that keeps the test as it is. Review, not the lint, checks the content of a reason. Review also covers what the lint cannot see: a test that mixes parallel and sequential direct subtests, nested subtests, a `defer` beside a test cleanup, and subtests the linter cannot resolve statically.

#### Scenario: Sequential subtests without a reason
- **WHEN** none of the direct subtests of a parallel backend test run in parallel and the test carries no stated reason
- **THEN** the backend lint fails and names the test

#### Scenario: Parallel subtests under a sequential test
- **WHEN** the subtests of a backend test run in parallel, the test itself does not, and it carries no stated reason
- **THEN** the backend lint fails and names the test

#### Scenario: A deferred release beside parallel subtests
- **WHEN** a backend test with parallel subtests releases a resource through `defer`, calls no test cleanup and carries no stated reason
- **THEN** the backend lint fails and names the test, because the release would run before the subtests do

#### Scenario: A test that mixes parallel and sequential subtests
- **WHEN** one direct subtest of a parallel test runs in parallel and another does not
- **THEN** the backend lint passes, and review decides whether the sequential subtest has a reason

#### Scenario: Subtests that share a server
- **WHEN** the subtests of a parallel test share one server and the test states that reason
- **THEN** the backend lint passes and the subtests keep running one after another

#### Scenario: Independent subtests in one test
- **WHEN** the subtests of one test each create their own application and database
- **THEN** they run at the same time within the limit of parallel tests
