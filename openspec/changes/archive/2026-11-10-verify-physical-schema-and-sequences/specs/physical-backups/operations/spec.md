## ADDED Requirements

### Requirement: Physical recovery preserves committed schema changes

On PostgreSQL 17 and 18, restoring to the latest available point on one timeline SHALL preserve the committed schema and rows represented by the selected backups and contiguous archived WAL. This SHALL hold both for a FULL followed only by WAL and for a FULL followed by incremental backups and WAL. Changes made after the last completed backup SHALL be recovered from the subsequent WAL. Rolled-back transactional DDL SHALL leave no changes in the recovered schema.

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

Physical recovery on PostgreSQL 17 and 18 SHALL preserve logged sequences used by `serial`, identity columns and explicit sequence defaults, including objects created after the FULL. It SHALL preserve their column defaults, identity generation mode, column ownership dependencies and sequence type, start, increment, minimum, maximum, cycle and cache settings at the selected recovery point.

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

### Requirement: PITR applies the transaction boundary to schema changes

For a reachable target after the end of the selected base backup on one timeline, PITR SHALL preserve the transactional schema changes committed before the target and SHALL exclude those committed after it, including commands executed before the target by a transaction that commits later. This SHALL cover object creation, alteration, rename and deletion, as well as transactional sequence definition changes. It SHALL NOT impose that transaction boundary on the effects of `nextval` or `setval` beyond PostgreSQL's native recovery semantics.

#### Scenario: The target precedes the schema transaction commit

- **WHEN** a transaction executes schema changes before the target time but commits after it, and WAL containing the later commit is available
- **THEN** restoring to that target preserves the schema as it was before the transaction, including objects that the transaction later renames or deletes
- **AND** objects created only by that transaction are absent

#### Scenario: The target follows the schema transaction commit

- **WHEN** a transaction commits schema and sequence definition changes before the target, a later transaction changes the schema after the target, and WAL containing both commits is available
- **THEN** restoring to the target includes the first transaction's definitions and rows and excludes the later transaction's changes
- **AND** generated-ID inserts succeed under the recovered definitions and native sequence semantics
