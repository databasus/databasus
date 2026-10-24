# Backend Code Checks Specification

## Purpose

Defines which rules of the backend architecture and code style a machine enforces, where those rules apply while the backend moves into its target layout, and what the production server binary may contain.

## Requirements

### Requirement: The backend lint enforces the documented rules on code in the target layout
The backend lint SHALL check these rules, taken from the architecture document and, for suppressions, from the repository's security rules, on production and test files alike, for all code in the target layout: the composition root, the domain modules, the engine drivers, the shared library, the verification agent's new home, the server and agent entry points in the target layout, and the backend's own tools:

- no fixed sleeps or eventual assertions, no printing to standard output, no process exit, no environment reads and no background contexts outside the places the architecture document allows;
- database access only in repository files;
- log calls with attributes only, context-aware methods, constant messages and `snake_case` keys;
- test libraries only in test code, and no HTTP framework or ORM in the verification agent;
- every field listed in a dependency struct literal;
- every value handled in a switch on an enum;
- no package initialization functions;
- no lint suppression without a written reason.

A violation SHALL fail the lint with the file, the line and the rule. Three rules of the architecture document are outside this requirement: fixed import aliases, the ban on package-level dependency variables, and the checks of the HTTP contract and of generated code.

#### Scenario: A sleep in a new test
- **WHEN** a test file of a module in the target layout calls a fixed sleep
- **THEN** the backend lint fails and names the file, the line and the rule against fixed pauses

#### Scenario: A log call with a formatted message
- **WHEN** code in the target layout logs a message built from a value instead of passing the value as an attribute
- **THEN** the backend lint fails and names the logging rule

#### Scenario: A dependency literal with a missing field
- **WHEN** code in the target layout builds a module's dependency struct without one of its fields
- **THEN** the backend lint fails and names the missing field

#### Scenario: A suppression without a reason
- **WHEN** code in the target layout suppresses a lint finding without a written reason
- **THEN** the backend lint fails

### Requirement: Code awaiting its move keeps the checks it had
For the rules of the previous requirement, code outside the target layout SHALL be checked exactly as before this change: its production files by the linters that checked them before, and its test files by none. The lint configuration SHALL list the target layout explicitly, so code added outside it falls under the old checks and code added inside it falls under every rule. Three rules of the architecture check also cover code outside the target layout: the production binary contains no test code, old backup code writes stored files only through the file store, and test helpers behind the test helper build tag exist only outside the target layout.

#### Scenario: An old package with an existing violation of a new rule
- **WHEN** a production file of a package outside the target layout calls a fixed sleep, as it did before this change
- **THEN** the backend lint passes

#### Scenario: An old package breaks a rule it was checked by before
- **WHEN** a production file outside the target layout ignores a returned error
- **THEN** the backend lint fails, as it did before this change

#### Scenario: A new directory in the target layout
- **WHEN** a developer adds a new domain module in the target layout
- **THEN** every rule applies to it without a change to the lint configuration

### Requirement: The architecture check enforces structure rules
The backend lint SHALL include an architecture check for the rules the architecture document assigns to it: dependencies point from the entry points through the composition root and the modules to the engines and the shared library, never back; another package reaches a module only through the module's root package, and that root package holds only the module's contract file; methods of a handler, a service or a repository live in files of that role; a package's errors are declared in its errors file; a bare goroutine start appears only in the shared library; the verification agent imports only what the architecture document allows. These rules SHALL apply to production files in the target layout. Test files SHALL be exempt, because a module's API tests build the whole application through the test harness above the modules and start goroutines of their own.

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
- **WHEN** a package of the verification agent in the target layout imports a domain module
- **THEN** the architecture check fails and names the import and the list of packages the agent may import

#### Scenario: A goroutine started in a module
- **WHEN** a service in a module starts a goroutine with a bare go statement
- **THEN** the architecture check fails and names the shared library as the only place where a bare go statement is allowed

### Requirement: Old backup code writes stored files only through the file store
The architecture check SHALL fail when backup code awaiting its move calls a storage provider's own save or delete method directly outside test files, so that every write and removal of a stored file goes through the file store that keeps a failed backup from leaving a file behind. In the target layout the providers live inside the storages module, so the module boundary rule already keeps backup code away from them.

#### Scenario: A direct provider delete
- **WHEN** backup code calls a storage provider's delete method directly
- **THEN** the architecture check fails and names the call and the file store to use instead

#### Scenario: A test calls the provider
- **WHEN** a backup test calls a storage provider's save method to prepare data
- **THEN** the architecture check passes

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

### Requirement: Test helpers behind the build tag stay out of the target layout
Files that only builds with the test helper build tag compile SHALL exist only outside the target layout. The architecture check SHALL fail for such a file in the target layout and name the file and the rule, because tests in the target layout keep their helpers in test files or in packages that only tests import.

#### Scenario: A tagged helper in a module
- **WHEN** a module in the target layout has a file with the test helper build tag
- **THEN** the architecture check fails and names the file and the rule

#### Scenario: A tagged helper in an old package
- **WHEN** a package awaiting its move has a file with the test helper build tag
- **THEN** the architecture check passes

### Requirement: Backend tests detect data races
Backend tests SHALL run with the race detector, both in CI and in the command that runs given test packages locally. A data race detected during a test SHALL fail that test.

#### Scenario: A race in a test run
- **WHEN** two goroutines write the same variable without synchronization during a backend test
- **THEN** the test fails with the race report

#### Scenario: A local package run
- **WHEN** a developer runs the given test packages with the backend's package test command
- **THEN** the packages run with the race detector
