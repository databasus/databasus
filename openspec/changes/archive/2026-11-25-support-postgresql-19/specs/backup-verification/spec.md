# Spec Delta

## ADDED Requirements

### Requirement: PostgreSQL 19 backups are verified

The verification agent SHALL verify backups of PostgreSQL 19 like those of every other accepted PostgreSQL major, and its release SHALL name a PostgreSQL 19 minor. An extension that the PostgreSQL 19 verification image lacks SHALL count as an ignorable error of the verification, as for every other major.

#### Scenario: Verify a PostgreSQL 19 backup

- **WHEN** an agent whose release names PostgreSQL 19.0 verifies a completed backup of a PostgreSQL 19 database with the default image repository
- **THEN** the restore runs on a PostgreSQL 19.0 server and the verification completes

#### Scenario: A curated extension without a PostgreSQL 19 build

- **WHEN** a PostgreSQL 19 backup creates an extension the verification image carries for older majors but that has no PostgreSQL 19 build, and nothing in the backup depends on it
- **THEN** the verification treats the missing extension as an ignorable error and completes

### Requirement: An agent explains a PostgreSQL major its release cannot verify

When an agent's release names no minor for a backup's PostgreSQL major, the agent SHALL fail that attempt as an agent-side setup failure without starting a restore, so the attempt policy can hand the verification to another agent. When the attempts run out, the verification's short failure summary SHALL name the backup's major and the majors this agent release verifies, and say that a newer agent release may add that major.

#### Scenario: A backup of a major the agent release does not name

- **WHEN** an agent whose release names minors for PostgreSQL 12 to 19 is assigned a verification of a PostgreSQL 20 backup
- **THEN** the attempt fails as an agent-side setup failure without a restore container being started
- **AND** when the verification's attempts run out, its short failure summary says that this agent release cannot verify PostgreSQL 20 backups, lists 12 to 19 as the verified versions, and says that a newer agent release may add PostgreSQL 20

#### Scenario: A major the agent release covers

- **WHEN** the same agent is assigned a verification of a PostgreSQL 19 backup
- **THEN** the agent restores it into the PostgreSQL minor its release names for 19
