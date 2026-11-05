## ADDED Requirements

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
