## ADDED Requirements

### Requirement: Physical recovery is independent of inherited replication settings

For PostgreSQL 17 and 18, preparing a physical recovery set SHALL prevent inherited source replication settings, competing recovery targets and recovery signal files from overriding the newly selected recovery operation. An explicit target SHALL govern replay to that target; recovery without an explicit target SHALL NOT stop at a stale inherited target. Startup SHALL use the restored data directory and local archived WAL without depending on the former primary's connection settings or archive command. When local authentication files are copied, source-specific data-directory and authentication-file path overrides SHALL NOT prevent the independent target from using them. This SHALL apply to assignments with or without `=` and to locally included configuration regardless of filename extension, including nested `include`, `include_if_exists` and `include_dir` directives. This SHALL also hold when an explicitly selected completed backup ships no additional archived WAL.

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

### Requirement: Cleanup preserves retained recovery points after repeated Patroni switches

On PostgreSQL 17 and 18, physical backup cleanup and storage file deletion SHALL preserve the files, manifests, WAL and timeline histories required by recovery points still protected by retention after repeated Patroni switches. This SHALL include a reachable target on an older timeline as well as a reachable target after a replacement FULL on the newest timeline. Cleanup SHALL NOT make a target between promotion and completion of its replacement FULL reachable.

#### Scenario: Retained old and new targets survive both cleanup stages

- **WHEN** two Patroni switchovers have completed, retention still protects a reachable target before the first switch and another after the newest replacement FULL, and physical cleanup and storage deletion have processed their work
- **THEN** newly downloaded sets for both targets start independent PostgreSQL clusters with their respective rows, schema definitions and sequence definitions
- **AND** new generated-ID writes succeed in both recovered clusters
- **AND** each archive-based set includes the published history required for its own timeline in its FULL's storage

#### Scenario: Cleanup does not hide the promotion recovery gap

- **WHEN** both cleanup stages have run after the replacement FULL and a target between promotion and that FULL's completion still requires replay across timelines
- **THEN** requesting that target returns `409` with code `target_time_unreachable` and issues no restore token

### Requirement: Delayed WAL publication does not extend a source timeline's PITR time

An explicit PITR target SHALL require source-time evidence from complete transaction records in the selected timeline's contiguous published WAL, or a completed-backup consistency point. Receipt, claim and upload timestamps SHALL NOT establish source-time reachability. With inclusive recovery, a confirming WAL transaction SHALL be strictly later than the target. Old WAL without source-time evidence SHALL NOT use receipt time as a substitute. Latest recovery and explicit completed-backup recovery SHALL remain available without claiming an unconfirmed explicit target.

User-controlled descriptive payload, including multiline prepared-transaction identifiers, SHALL NOT be interpreted as COMMIT/ABORT records or extend source-time evidence. Genuine transaction completion after PREPARE SHALL remain usable as evidence.

Confirmed source time SHALL retain the maximum timestamp among trusted complete transaction records. A later record with an older timestamp, including an independently verified completion after PREPARE, SHALL NOT reduce already confirmed coverage.

When an explicit target equals the selected completed backup's consistency point and contiguous WAL contains no later confirming transaction, recovery SHALL use the completed backup and its required ancestors without additional archived WAL. Supplying that advertised target to the recovery script SHALL start a writable cluster without requiring a later transaction. Latest recovery SHALL retain eligible WAL, and targets beyond the confirmed bound SHALL remain refused.

#### Scenario: A completed backup is the only confirmed time

- **WHEN** a completed FULL or INCREMENTAL has subsequent archived WAL but no transaction later than its completion, and the owner requests the completion point advertised by statistics
- **THEN** the downloaded set contains the selected backup and its required ancestors without additional archived WAL
- **AND** starting recovery with that explicit target succeeds and permits new writes
- **AND** latest recovery retains eligible WAL without advertising an unconfirmed later PITR target

#### Scenario: Transaction timestamps decrease in WAL order

- **WHEN** a later complete transaction record has a timestamp older than an already confirmed record
- **THEN** statistics and target validation retain the greatest confirmed source timestamp as their time evidence
- **AND** an independently verified completion after PREPARE cannot reduce an earlier trusted bound

#### Scenario: A prepared identifier imitates transaction output

- **WHEN** a prepared-transaction identifier contains a multiline COMMIT description or a fabricated WAL record header with a future timestamp
- **THEN** decoding retains only time confirmed by actual completed WAL records, and the fabricated time cannot extend recovery coverage or authorize a target
- **AND** a subsequent genuine COMMIT or ABORT remains observable

#### Scenario: Old timeline WAL is published after promotion

- **WHEN** old-timeline WAL is published after promotion and its receipt timestamp is later than a requested target in the replacement-FULL gap
- **THEN** token issuance and download validation refuse that target with `409 target_time_unreachable`, both before and after the replacement FULL completes
- **AND** a pre-promotion target confirmed by that timeline's archived transactions remains recoverable

#### Scenario: A final transaction is not a later confirmation

- **WHEN** a requested target equals the last decoded transaction timestamp and no later transaction is published
- **THEN** token issuance refuses that explicit target
- **AND** a target one microsecond earlier remains eligible when its WAL is contiguous

#### Scenario: Confirming WAL disappears after token issuance

- **WHEN** a valid target token was issued while confirming WAL was available, that WAL is removed, and remaining late-received WAL has insufficient source-time evidence
- **THEN** downloading with that token returns `409 target_time_unreachable` before archive streaming begins
