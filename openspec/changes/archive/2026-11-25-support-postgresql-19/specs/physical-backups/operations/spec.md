# Spec Delta

## MODIFIED Requirements

### Requirement: Script recovery instructions preserve the selected restore settings

For both Host and Docker recovery, the Script method SHALL show a recovery command and startup instructions that consistently use the restore directory and PostgreSQL version selected in the form. A restore directory containing spaces SHALL remain one shell argument. Relative directories SHALL resolve from the execution directory, and absolute directories SHALL remain absolute.

For Host recovery, a supplied PostgreSQL tools directory SHALL apply to both recovery preparation and the displayed PostgreSQL startup command. For Docker recovery, the selected PostgreSQL image SHALL apply to both recovery preparation and the displayed startup command. The displayed container mount SHALL make PostgreSQL serve the cluster prepared in the selected restore directory, including the version-specific data layout of PostgreSQL 18 and later. Following these commands on a prepared recovery machine SHALL start the restored cluster without editing the displayed paths, image or arguments.

#### Scenario: Host recovery with a custom directory

- **WHEN** a user selects Host recovery, a relative restore directory containing a space and a PostgreSQL tools directory outside the default command search path
- **THEN** the recovery and startup commands use those directories, and following the commands starts the restored cluster

#### Scenario: Docker recovery with an absolute directory

- **WHEN** a user selects Docker recovery, an absolute restore directory containing a space and a matching PostgreSQL image identified by digest
- **THEN** both commands use the selected image, and startup mounts the selected directory without prepending the execution directory
- **AND** PostgreSQL serves the restored cluster rather than initializing an empty cluster

#### Scenario: Docker recovery of a PostgreSQL 19 cluster

- **WHEN** a user selects Docker recovery for a PostgreSQL 19 database
- **THEN** the startup command mounts the restore directory at the image's volume root, so that PostgreSQL 19 serves the cluster prepared under its version-specific data directory

### Requirement: Physical recovery preserves committed schema changes

On PostgreSQL 17, 18 and 19, restoring to the latest available point on one timeline SHALL preserve the committed schema and rows represented by the selected backups and contiguous archived WAL. This SHALL hold both for a FULL followed only by WAL and for a FULL followed by incremental backups and WAL. Changes made after the last completed backup SHALL be recovered from the subsequent WAL. Rolled-back transactional DDL SHALL leave no changes in the recovered schema.

The preserved schema SHALL include created schemas and tables, added and renamed columns, column types and defaults, indexes, constraints, and committed object renames and deletions. A column type conversion that rewrites stored rows SHALL preserve both the converted values and the resulting definition.

#### Scenario: Schema changes recovered only from WAL after a FULL

- **WHEN** a FULL completes, schema changes and corresponding rows are committed afterwards, and the following WAL is archived without taking an incremental backup
- **THEN** restoring to the latest available point starts a writable cluster with those rows and the committed object definitions

#### Scenario: Schema changes span incrementals and subsequent WAL

- **WHEN** a FULL and two successive incremental backups contain different stages of schema changes, further changes are committed after the last incremental, and contiguous WAL through those changes is archived
- **THEN** restoring to the latest available point preserves the rows and schema changes from every stage, including those present only in the later WAL

#### Scenario: Renamed and deleted objects retain their committed state

- **WHEN** columns or tables are renamed and other objects are deleted before the selected recovery point
- **THEN** the recovered schema contains the renamed objects under their new names and excludes their old names and the deleted objects

#### Scenario: Transactional DDL is rolled back

- **WHEN** a transaction creates, alters or drops schema objects and is rolled back before the selected recovery point
- **THEN** recovery preserves the previously committed definitions and excludes objects created only in that transaction

### Requirement: Physical recovery preserves sequence configuration and usable ID generation

Physical recovery on PostgreSQL 17, 18 and 19 SHALL preserve logged sequences used by `serial`, identity columns and explicit sequence defaults, including objects created after the FULL. It SHALL preserve their column defaults, identity generation mode, column ownership dependencies and sequence type, start, increment, minimum, maximum, cycle and cache settings at the selected recovery point.

Sequence values after recovery SHALL follow native PostgreSQL recovery semantics. Recovery SHALL NOT promise gapless IDs, an exact next value based on the largest restored ID, or transactional rollback of `nextval` and `setval`. For an ascending, noncycling sequence used consistently to generate IDs, with any explicit repositioning moving it forward, new generated IDs SHALL avoid the restored committed IDs and respect the recovered configuration. A forward `setval` represented by the recovered WAL SHALL not be silently lost even when losing it would not immediately cause a duplicate ID.

#### Scenario: Serial and identity columns remain usable

- **WHEN** a backup and subsequent WAL contain tables with `serial` and identity columns, including a table created after the FULL
- **THEN** recovery preserves the generation definitions and sequence dependencies
- **AND** the first insert that omits a generated ID succeeds and allocates an ID distinct from the restored committed IDs
- **AND** subsequent allocations also produce IDs distinct from the restored committed IDs

#### Scenario: An explicit sequence is repositioned and altered

- **WHEN** a logged sequence used by a column default is moved forward with `setval`, its increment and cache are changed, and those operations are included in the selected recovery point
- **THEN** recovery preserves the column default and the committed sequence parameters
- **AND** newly generated values respect the forward position and the recovered increment, allowing native recovery gaps

#### Scenario: Both cache settings survive recovery

- **WHEN** the recovered state includes a sequence with `CACHE 1` and another with a larger cache
- **THEN** both retain their cache settings and generate distinct usable IDs after recovery without requiring either sequence to resume at the largest restored ID plus one

#### Scenario: A sequence call occurs in a rolled-back transaction

- **WHEN** a transaction calls `nextval` or `setval` on an existing logged sequence, writes a row and then rolls back
- **THEN** the recovered table excludes that row
- **AND** the sequence remains usable according to native PostgreSQL recovery semantics without a guarantee that the call was undone

### Requirement: Physical recovery is independent of inherited replication settings

For PostgreSQL 17, 18 and 19, preparing a physical recovery set SHALL prevent inherited source replication settings, competing recovery targets and recovery signal files from overriding the newly selected recovery operation. An explicit target SHALL govern replay to that target; recovery without an explicit target SHALL NOT stop at a stale inherited target. Startup SHALL use the restored data directory and local archived WAL without depending on the former primary's connection settings or archive command. When local authentication files are copied, source-specific data-directory and authentication-file path overrides SHALL NOT prevent the independent target from using them. This SHALL apply to assignments with or without `=` and to locally included configuration regardless of filename extension, including nested `include`, `include_if_exists` and `include_dir` directives. This SHALL also hold when an explicitly selected completed backup ships no additional archived WAL.

If the source keeps configuration outside its data directory, the recovery script SHALL retain its existing diagnostic and operator-controlled configuration preparation. The test environment SHALL supply an independent recovery configuration without weakening the source's access rules or transport security.

Absolute includes below the source data directory recorded in copied configuration SHALL be rebased into the restored data directory. Their contents SHALL receive the same sanitization as relative includes, and the rewritten paths SHALL remain usable when the prepared directory is moved into its runtime container.

Source-directory discovery SHALL follow the local include graph before sanitization and honor configuration order. A source directory declared only in an included file SHALL support rebasing absolute includes, including those appearing before that declaration.

Source-local `hba_file` and `ident_file` paths to copied authentication files SHALL be rebased to their restored locations, including subdirectories without default root-level authentication files. A referenced copied file SHALL take precedence over an unrelated default root-level file.

Authentication paths SHALL target the prepared local PGDATA for native startup or the selected image's PGDATA for `--combine-image`. Copied authentication files at those paths SHALL NOT trigger the missing-configuration diagnostic merely because their default root-level names are absent.

#### Scenario: Absolute source-local configuration includes

- **WHEN** copied configuration records its source data directory and includes an absolute path below it, including a nested optional include or include directory
- **THEN** recovery preparation rewrites the source-local path to the copied file or directory, removes inherited recovery settings from its contents and starts PostgreSQL without the source filesystem

#### Scenario: Source directory is declared in a local settings file

- **WHEN** the main configuration includes a local file that records the source data directory and includes another file by an absolute source-local path
- **THEN** recovery preparation discovers the source directory through the include graph before removing its declaration
- **AND** PostgreSQL starts with the sanitized copied files without depending on the original source paths

#### Scenario: Targeted recovery from a Patroni source

- **WHEN** a downloaded recovery set from a promoted Patroni source contains inherited replication settings or older recovery targets and the former primary is unavailable
- **THEN** preparing and starting the recovery reaches the newly selected target using the local archive
- **AND** the recovered cluster contains transactions through that target, excludes later transactions and accepts new writes with generated IDs

#### Scenario: Copied authentication files reside in a subdirectory

- **WHEN** copied configuration references source-local HBA and ident files in a subdirectory and no default root-level authentication files were copied
- **THEN** recovery preparation rebases both paths and PostgreSQL starts using the copied authentication files
- **AND** their contents and the source's access rules remain unchanged

#### Scenario: Referenced authentication files take precedence over root defaults

- **WHEN** copied configuration references source-local HBA and ident files in a subdirectory and distinct default root-level authentication files were also copied
- **THEN** recovery preparation retains the referenced files and PostgreSQL starts using their unchanged contents
- **AND** both native startup and image-entrypoint startup use those files without changing source access rules

#### Scenario: Latest recovery ignores a stale target

- **WHEN** latest recovery is requested from a Patroni source whose copied configuration contains an older recovery target or a standby signal
- **THEN** recovery includes the available later archived transactions rather than stopping at the inherited target or remaining a standby
- **AND** the recovered cluster becomes writable without contacting the former primary

#### Scenario: Explicit backup recovery ships no additional WAL

- **WHEN** a completed physical backup from a Patroni source is selected by its identifier and its copied configuration contains source replication or recovery settings
- **THEN** recovery starts a writable independent cluster at the selected backup's consistency point without requiring an additional archive or the former primary

#### Scenario: External configuration is prepared for recovery

- **WHEN** the downloaded data directory lacks configuration stored externally by the source
- **THEN** the recovery diagnostic identifies the missing configuration and the required preparation
- **AND** supplying independent configuration according to that diagnostic allows the restored cluster to start while source access rules and transport security remain unchanged

### Requirement: Cleanup preserves recovery points around repeated Patroni switches

On PostgreSQL 17, 18 and 19, physical backup cleanup and storage file deletion SHALL preserve the files, manifests, WAL and timeline histories required by recovery points still protected by retention after repeated Patroni switches. This SHALL include a reachable target before a switch, a target between a switch and the next completed backup, and a target after that backup.

#### Scenario: Retained targets around switches survive both cleanup stages

- **WHEN** two Patroni switchovers have completed, retention still protects a reachable target before the first switch, a target one minute after the newest switch that lies before the next completed backup on the promoted timeline, and a target after that backup, and physical cleanup and storage deletion have processed their work
- **THEN** newly downloaded sets for all three targets start independent PostgreSQL clusters with their respective rows, schema definitions and sequence definitions
- **AND** new generated-ID writes succeed in every recovered cluster
- **AND** each archive-based set includes the published history of every timeline after its FULL's timeline on its line, in its FULL's storage

#### Scenario: Cleanup keeps the WAL that crosses a switch

- **WHEN** a completed FULL on the former timeline is retained, the promoted timeline's history is published in that FULL's storage, no backup has completed on the promoted timeline yet, a transaction later than a target after the switch is archived, and both cleanup stages have run
- **THEN** the WAL of the promoted timeline and its history remain archived
- **AND** that target receives a restore token, and the recovered cluster contains the rows committed up to it
