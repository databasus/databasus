## MODIFIED Requirements

### Requirement: The backend lint enforces the documented rules on code in the target layout
The backend lint SHALL check these rules, taken from the architecture document and, for suppressions, from the repository's security rules, on production and test files alike, for all code in the target layout: the composition root, the domain modules, the engine drivers, the shared library, the verification agent's new home, the server and agent entry points in the target layout, and the backend's own tools:

- no fixed sleeps or eventual assertions, no printing to standard output, no process exit, no environment reads and no background contexts outside the places the architecture document allows;
- database access only in repository files;
- log calls with attributes only, context-aware methods, constant messages and `snake_case` keys;
- test libraries only in test code, and no HTTP framework or ORM in the verification agent or in the engine drivers;
- every field listed in a dependency struct literal;
- every value handled in a switch on an enum;
- no package initialization functions;
- no lint suppression without a written reason.

On production files in the target layout, the backend lint SHALL also check that external processes start only through the shared library's process helper, which ties a process to its operation and stops it with that operation. The helper itself is the only production code that starts a process directly. Test files are outside this rule, because they start the binary under test or the test binary itself rather than an external tool.

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

#### Scenario: An engine driver starts a client tool directly
- **WHEN** production code of an engine driver starts an external program without the shared library's process helper
- **THEN** the backend lint fails and names the process helper as the only way to start an external program

#### Scenario: A test starts the binary under test
- **WHEN** a test file in the target layout builds and starts the server binary directly
- **THEN** the backend lint passes

#### Scenario: An engine driver imports the ORM
- **WHEN** a file of an engine driver imports the ORM or the HTTP framework
- **THEN** the backend lint fails and names the rule that engine drivers know no tables and no HTTP
