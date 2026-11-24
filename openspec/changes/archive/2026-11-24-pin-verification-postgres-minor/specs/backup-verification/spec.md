## ADDED Requirements

### Requirement: A verification restores into the PostgreSQL minor its agent release names

Each release of the verification agent SHALL name one exact PostgreSQL minor release for every PostgreSQL major version it verifies. For a PostgreSQL backup, the agent SHALL request the restore image by the tag of the minor its own release names for the backup's major version, from the Databasus verification image repository, from an image repository the agent's owner configured, or from the official PostgreSQL repository it falls back to. An image the agent obtained earlier under the major version alone or under another minor SHALL NOT be used instead. When a newer agent release names a newer minor, the agent SHALL obtain the image of that minor on its next verification without anyone removing the old image from its host.

When the image of the named minor cannot be obtained from the Databasus or the configured repository, the agent SHALL fall back to the official PostgreSQL image of the same minor. When the configured repository is the official PostgreSQL repository itself, there is no fallback and the job fails as a setup failure.

The published Databasus verification image for a minor SHALL contain a server of exactly that minor. The content of an image in a repository the agent's owner configured is that owner's responsibility; the agent only requests the tag of the named minor from it.

This requirement does not apply to TimescaleDB backups, which keep using the image of their extension version.

#### Scenario: A backup is restored into the named minor

- **WHEN** an agent whose release names PostgreSQL 18.7 for major version 18 verifies a PostgreSQL 18 backup with the default image repository
- **THEN** the restore runs on a PostgreSQL 18.7 server

#### Scenario: An older image on the host is not reused

- **WHEN** the agent's host holds `databasus/verification-postgres:18` or `databasus/verification-postgres:18.6`, both with a PostgreSQL 18.6 server, and the agent's release names 18.7
- **THEN** the agent obtains `databasus/verification-postgres:18.7` and restores into it

#### Scenario: An agent upgrade moves verification to a newer minor

- **WHEN** an agent that verified PostgreSQL 18 backups on 18.6 upgrades to a release that names 18.7
- **THEN** its next PostgreSQL 18 verification restores into 18.7 without any manual action on the host

#### Scenario: The Databasus image is unavailable

- **WHEN** `databasus/verification-postgres:18.7` cannot be obtained and the agent's release names 18.7
- **THEN** the agent restores into the official `postgres:18.7` image, not into an image of the major version alone

#### Scenario: A configured repository is asked for the named minor

- **WHEN** the agent's owner configured another image repository and it publishes the tag `18.7`
- **THEN** the agent requests and uses the `18.7` image from that repository

#### Scenario: A configured repository has no tag for the named minor

- **WHEN** the agent's owner configured an image repository that publishes only the tag `18` and no tag `18.7`
- **THEN** the agent falls back to the official `postgres:18.7` image and restores into it

#### Scenario: The official repository is configured and its minor is unavailable

- **WHEN** the agent's owner configured the official PostgreSQL repository and `postgres:18.7` cannot be obtained
- **THEN** the job fails as a setup failure without trying another image

#### Scenario: A verification image with another server minor is not published

- **WHEN** a release builds the Databasus verification image for a minor and the server installed in it is a different minor
- **THEN** the build fails and no image is published under that minor

### Requirement: The agent removes only its own superseded verification images

At startup, the verification agent SHALL remove every local image tag of its verification image repository that is not a version its release names: the Databasus verification repository, or the repository its owner configured. It SHALL NOT remove any image when the configured repository is the official PostgreSQL repository, and SHALL NOT remove images of any other repository, including official PostgreSQL images it fell back to and TimescaleDB images. It SHALL NOT force a removal: an image that a container uses SHALL stay, and an image that carries another tag SHALL keep that tag. A removal that fails, or a cleanup that cannot run, SHALL be logged as a warning and SHALL NOT stop the agent or fail a verification.

#### Scenario: Superseded images are removed after an upgrade

- **WHEN** an agent whose release names 18.7 starts on a host that holds `databasus/verification-postgres:18`, `databasus/verification-postgres:18.6` and `databasus/verification-postgres:18.7`
- **THEN** the agent removes the `18` and `18.6` tags and keeps `18.7`

#### Scenario: Images of other repositories are kept

- **WHEN** the agent starts on a host that holds `postgres:18.6`, `timescale/timescaledb:2.17.0-pg17` and an image of an unrelated repository
- **THEN** the agent removes none of them

#### Scenario: The official repository is configured

- **WHEN** the agent's owner configured the official PostgreSQL repository and the host holds `postgres:18.6`
- **THEN** the agent removes no image

#### Scenario: An image in use stays

- **WHEN** a superseded verification image is used by a container at agent startup
- **THEN** the image stays, the agent logs a warning and starts normally

#### Scenario: A shared image keeps its other tag

- **WHEN** a superseded verification tag points to an image that also carries another tag
- **THEN** only the superseded tag is removed

#### Scenario: A configured repository is cleaned by the same rule

- **WHEN** the agent's owner configured `example/pg`, the agent's release names 18.7, and the host holds `example/pg:18`, `example/pg:18.7` and `databasus/verification-postgres:18`
- **THEN** the agent removes only `example/pg:18`

#### Scenario: The images cannot be listed

- **WHEN** the agent cannot list the images of its verification image repository at startup
- **THEN** it logs a warning, removes nothing and starts normally
