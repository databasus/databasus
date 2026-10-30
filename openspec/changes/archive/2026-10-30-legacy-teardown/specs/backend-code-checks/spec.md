## RENAMED Requirements

- FROM: `### Requirement: The backend lint enforces the documented rules on code in the target layout`
- TO: `### Requirement: The backend lint enforces the documented rules throughout the backend`

## MODIFIED Requirements

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

## REMOVED Requirements

### Requirement: Code awaiting its move keeps the checks it had

**Reason**: The backend migration is complete and every package is subject to the documented checks.

**Migration**: Fix violations in remaining commands and shared support, then remove migration-period lint exclusions.

### Requirement: Test helpers behind the build tag stay out of the target layout

**Reason**: The temporary tagged helpers and their runtime bridges are removed.

**Migration**: Keep shared test support reachable only from tests and retain production dependency-graph checks; ordinary test commands need no helper build tag.
