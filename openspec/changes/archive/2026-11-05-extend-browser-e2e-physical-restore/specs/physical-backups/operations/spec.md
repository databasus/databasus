## ADDED Requirements

### Requirement: Script recovery instructions preserve the selected restore settings

For both Host and Docker recovery, the Script method SHALL show a recovery command and startup instructions that consistently use the restore directory and PostgreSQL version selected in the form. A restore directory containing spaces SHALL remain one shell argument. Relative directories SHALL resolve from the execution directory, and absolute directories SHALL remain absolute.

For Host recovery, a supplied PostgreSQL tools directory SHALL apply to both recovery preparation and the displayed PostgreSQL startup command. For Docker recovery, the selected PostgreSQL image SHALL apply to both recovery preparation and the displayed startup command. The displayed container mount SHALL make PostgreSQL serve the cluster prepared in the selected restore directory, including PostgreSQL 18's version-specific data layout. Following these commands on a prepared recovery machine SHALL start the restored cluster without editing the displayed paths, image or arguments.

#### Scenario: Host recovery with a custom directory

- **WHEN** a user selects Host recovery, a relative restore directory containing a space and a PostgreSQL tools directory outside the default command search path
- **THEN** the recovery and startup commands use those directories, and following the commands starts the restored cluster

#### Scenario: Docker recovery with an absolute directory

- **WHEN** a user selects Docker recovery, an absolute restore directory containing a space and a matching PostgreSQL image identified by digest
- **THEN** both commands use the selected image, and startup mounts the selected directory without prepending the execution directory
- **AND** PostgreSQL serves the restored cluster rather than initializing an empty cluster
