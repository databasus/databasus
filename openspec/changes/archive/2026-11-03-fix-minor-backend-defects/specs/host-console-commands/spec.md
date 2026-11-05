## ADDED Requirements

### Requirement: Asking for help succeeds

Starting the program with `-h`, `-help` or `--help` SHALL print the usage text and exit with code 0, without reading settings or opening the internal database. An unknown flag SHALL still print the usage text and exit with code 2.

#### Scenario: Help is requested

- **WHEN** an operator runs the program with `--help`, where the connection string is neither set nor published
- **THEN** the usage text is printed and the program exits with code 0
- **AND** the output has no message about a missing connection string

#### Scenario: An unknown flag

- **WHEN** an operator runs the program with a flag it does not know
- **THEN** the usage text is printed and the program exits with code 2
