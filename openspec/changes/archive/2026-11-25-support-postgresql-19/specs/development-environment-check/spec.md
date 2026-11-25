# Spec Delta

## MODIFIED Requirements

### Requirement: The check verifies that the client tool bundles find their libraries

The check SHALL fail when a client tool bundle of the machine's architecture holds a program that needs a shared library the system cannot provide, or a symbol that the installed library lacks. It SHALL look up libraries and symbols as the dynamic loader finds them in the caller's environment, without creating aliases for missing ones, and SHALL name each program with each missing library or symbol.

#### Scenario: A system library is missing

- **WHEN** a program of a client tool bundle needs a shared library that is not installed
- **THEN** the client tool bundles check fails and names the program and the missing library

#### Scenario: A system library is older than a program needs

- **WHEN** a program of a client tool bundle calls a function that the installed version of its shared library lacks, as the newest `psql` does with an older `libpq5`
- **THEN** the client tool bundles check fails and names the program and the missing symbol
