# postgresql-logical-backup-scope Specification

## Purpose
Defines which tables a PostgreSQL logical backup leaves out, either completely or by their rows only, what a restore of such a backup contains, which privileges the backup still requires and how the estimated database size accounts for the tables left out.

## Requirements

### Requirement: A whole-excluded table is missing from the backup and the restore

A PostgreSQL logical database SHALL have a list of whole-excluded tables. A table matched by an entry of this list SHALL be left out of the backup together with its structure, indexes, constraints and rows, so a restore of that backup SHALL NOT create the table. An entry of the form `schema.table` SHALL match by schema and table name. An entry with no schema part SHALL match only tables that the connecting role sees by their bare name under its default search path, which is usually only `public`; a same-named table in another schema SHALL NOT be matched by it. Either part MAY be a glob pattern, such as `logs_*`, and a double-quoted part SHALL match that name literally. Tables not matched by any entry SHALL be backed up and restored with their rows.

#### Scenario: Whole-excluded table by name

- **WHEN** a database lists `audit_tmp` as a whole-excluded table, a backup completes and that backup is restored into an empty database
- **THEN** the restored database has no `public.audit_tmp` table, and every other table is restored with its rows

#### Scenario: Unqualified name leaves other schemas alone

- **WHEN** a database lists `audit_tmp` as a whole-excluded table, the role's default search path is `"$user", public`, and the tables `public.audit_tmp` and `archive.audit_tmp` exist
- **THEN** a restore of its backup has no `public.audit_tmp` table and has `archive.audit_tmp` with its rows

#### Scenario: Whole-excluded tables by pattern

- **WHEN** a database lists `public.old_*` as a whole-excluded table and has the tables `public.old_orders` and `reporting.old_orders`
- **THEN** a restore of its backup has no `public.old_orders` table and has `reporting.old_orders` with its rows

### Requirement: A table whose data is excluded is restored empty

A PostgreSQL logical database SHALL have a second list, of tables whose data is excluded. Its entries SHALL match tables by the same rules as the list of whole-excluded tables. A table matched by an entry of this list SHALL be backed up without its rows, while its structure, column defaults, indexes, constraints and triggers SHALL stay in the backup. A restore of that backup SHALL create the table with all of these and with no rows. When the database also limits the backup to certain schemas, the list SHALL apply only within those schemas. An entry SHALL match only the tables whose own names it matches: naming a partitioned table SHALL NOT exclude the rows of its partitions, which need entries or a pattern of their own. Tables matched by neither list SHALL be backed up and restored with their rows.

#### Scenario: Naming a partitioned table keeps its partitions' rows

- **WHEN** a database has the partitioned table `public.events` with the partitions `public.events_2025` and `public.events_2026`, both holding rows, and lists only `public.events` as a table whose data is excluded
- **THEN** a restore of its backup has `public.events_2025` and `public.events_2026` with their rows

#### Scenario: Partitioned table needs its partitions matched

- **WHEN** a database has the partitioned table `public.events` with the partitions `public.events_2025` and `public.events_2026`, and lists `public.events_*` as tables whose data is excluded
- **THEN** a restore of its backup has `public.events` and both partitions with no rows

#### Scenario: Data-excluded table restored empty

- **WHEN** a database lists `public.logs` as a table whose data is excluded, `public.logs` holds rows and has a primary key and a secondary index, a backup completes and that backup is restored into an empty database
- **THEN** the restored database has the `public.logs` table with the same columns, primary key and secondary index and with no rows, and every other table is restored with its rows

#### Scenario: Data-excluded tables by pattern

- **WHEN** a database lists `*.history_*` as tables whose data is excluded and has the tables `public.history_2024` and `archive.history_2025`
- **THEN** a restore of its backup has both tables with no rows

#### Scenario: Data exclusion together with included schemas

- **WHEN** a database limits the backup to the schema `public` and lists `logs` as a table whose data is excluded, and the tables `public.logs` and `reporting.logs` exist
- **THEN** a restore of its backup has `public.logs` with no rows and has no `reporting` schema

#### Scenario: Existing databases after the upgrade

- **WHEN** Databasus is upgraded and a PostgreSQL logical database created before the upgrade is backed up
- **THEN** the database has an empty list of tables whose data is excluded, and its backup contains the same tables and rows as before the upgrade

### Requirement: A table in both lists is excluded entirely

When a table is matched by an entry of the whole-excluded list and by an entry of the data-excluded list, the backup SHALL leave out the whole table, as if only the whole-excluded entry existed. Saving a database whose lists overlap SHALL succeed.

#### Scenario: Same table in both lists

- **WHEN** a database lists `public.logs` both as a whole-excluded table and as a table whose data is excluded, and its backup is restored
- **THEN** the save succeeds, and the restored database has no `public.logs` table

### Requirement: Excluding a table's data does not lift the read-privilege check

When a PostgreSQL logical database is created, updated or has its connection tested, Databasus SHALL refuse it if its connecting role cannot read every table and sequence the backup will contain. A table whose data is excluded SHALL count as a table the backup contains, because the backup still has to lock it, so the role SHALL still need read access to it. Only a whole-excluded table SHALL be skipped by this check, matched by the same rules the backup uses, so a same-named table in a schema an unqualified entry does not reach SHALL still be checked. The error returned when the check fails SHALL name the tables and schemas the role cannot read, SHALL suggest whole-table exclusion as a way to narrow the backup, and SHALL state that excluding only a table's data still requires read access to that table.

#### Scenario: Role cannot read a data-excluded table

- **WHEN** a database lists `public.logs` as a table whose data is excluded and its connecting role has no read access to `public.logs`
- **THEN** testing the connection fails with an error that names `public.logs` and says that excluding only its data still requires read access

#### Scenario: Role cannot read a same-named table in another schema

- **WHEN** a database lists `logs` as a whole-excluded table, the role's default search path is `"$user", public`, and the role cannot read `archive.logs`
- **THEN** testing the connection fails with an error that names `archive.logs`

#### Scenario: Role cannot read a whole-excluded table

- **WHEN** a database lists `public.logs` as a whole-excluded table, its connecting role has no read access to `public.logs` and can read every other table and sequence in the backup
- **THEN** testing the connection succeeds

### Requirement: The estimated database size leaves out excluded rows

The estimated size of a PostgreSQL logical database SHALL leave out the size of whole-excluded tables and the size of tables whose data is excluded, including the size of their indexes. Tables SHALL be matched by the same rules the backup uses, so a same-named table that an unqualified entry does not reach SHALL still be counted.

#### Scenario: Large data-excluded table

- **WHEN** a database lists a table whose data is excluded, and that table has rows and a secondary index
- **THEN** the estimated database size is smaller than the estimate without any exclusion by at least that table's total size, its indexes included

#### Scenario: Same-named table in another schema is still counted

- **WHEN** a database lists `logs` as a table whose data is excluded, the role's default search path is `"$user", public`, and the tables `public.logs` and `archive.logs` hold rows
- **THEN** the estimated database size leaves out `public.logs` and includes `archive.logs`

### Requirement: Both lists are saved, shown and copied

Both lists SHALL be accepted when a PostgreSQL logical database is created or updated, SHALL be returned when the database is read and SHALL be carried over when the database is copied. Each list SHALL accept entries pasted as one value separated by commas or new lines, SHALL drop surrounding whitespace and empty entries, and SHALL keep each entry once. Updating a database without the list of tables whose data is excluded SHALL clear that list, as it does for the whole-excluded list.

#### Scenario: Saving and reading the lists

- **WHEN** the user saves a database with `audit_tmp` as a whole-excluded table and with `logs, history_*` pasted as tables whose data is excluded
- **THEN** reading the database returns `audit_tmp` in the whole-excluded list and `logs` and `history_*` as two entries in the data-excluded list

#### Scenario: Copying a database

- **WHEN** the user copies a database whose data-excluded list holds `logs`
- **THEN** the copy has `logs` in its data-excluded list

### Requirement: The form tells the two exclusions apart

The backup settings of a PostgreSQL logical database SHALL show the two lists as separate fields, one directly below the other. The whole-excluded list SHALL be labeled "Exclude whole tables" and its tooltip SHALL say that the table will be missing after a restore. The data-excluded list SHALL be labeled "Exclude table data" and its tooltip SHALL say that the table is restored empty with its structure kept, that a table in both lists is excluded entirely, that the partitions of a partitioned table must be matched by their own names or a pattern, and that a restore fails to recreate a foreign key when another backed-up table references rows of an emptied table. Both tooltips SHALL describe the accepted name and pattern forms, including that a name without a schema matches only tables on the default search path, usually `public`, and that `*.name` matches it in every schema. The database page SHALL show each list under the same label when it is not empty. The section holding these fields SHALL open on its own when either list is not empty. The labels and tooltips SHALL be available in every interface language. The forms of other database engines SHALL keep their current labels.

#### Scenario: Editing a database with data-excluded tables

- **WHEN** the user opens the edit form of a PostgreSQL logical database whose data-excluded list holds `logs` and whose other advanced settings are empty
- **THEN** the advanced settings are open, and the form shows "Exclude whole tables" followed by "Exclude table data" with `logs` in it

#### Scenario: Database page

- **WHEN** the user opens the page of a PostgreSQL logical database with `audit_tmp` as a whole-excluded table and `logs` as a table whose data is excluded
- **THEN** the page shows `audit_tmp` under "Exclude whole tables" and `logs` under "Exclude table data"

#### Scenario: MySQL form keeps its label

- **WHEN** the user opens the edit form of a MySQL database
- **THEN** its table exclusion field is still labeled "Exclude tables"

### Requirement: A backup records the sizes of its largest indexes without excluded rows

A PostgreSQL logical backup SHALL record, in megabytes, the sizes of the eight largest indexes of the tables whose rows it dumps, largest first, or of all of them when there are fewer. Indexes behind primary keys, unique constraints and exclusion constraints SHALL count as indexes. Indexes of whole-excluded tables and of tables whose data is excluded SHALL NOT count, and tables SHALL be matched by the same rules the backup uses. Each size SHALL be the size of the index on the source server when the backup starts. A backup of a database whose dumped tables have no indexes SHALL record an empty list, which is distinct from recording no sizes.

An index of a partitioned table SHALL count through the indexes of its partitions, each partition's index once, and not again as an index of the partitioned table. Partitions SHALL be matched by their own names, so excluding the rows of a partitioned table SHALL NOT drop its partitions' indexes from the record.

When the index sizes cannot be read, the backup SHALL still complete with its estimated database size, and it SHALL record no index sizes. When the estimated database size cannot be read, the backup SHALL record no index sizes either. Such a backup's verification is planned as for a backup without recorded index sizes.

#### Scenario: Largest indexes of the dumped tables

- **WHEN** a database has ten tables, each with one index, and a backup of it completes
- **THEN** the backup records the sizes of the eight largest of those indexes in megabytes, largest first

#### Scenario: A primary key counts as an index

- **WHEN** the largest index of a database backs the primary key of its largest table, and a backup of it completes
- **THEN** the first size the backup records is the size of that primary key's index

#### Scenario: A data-excluded table with the largest index

- **WHEN** a database lists a table whose data is excluded, that table has the largest index of the database, and a backup of it completes
- **THEN** the sizes the backup records leave out that index

#### Scenario: Same-named table in another schema is still counted

- **WHEN** a database lists `logs` as a whole-excluded table, the role's default search path is `"$user", public`, `public.logs` has the largest index of the database and `archive.logs` has the second largest
- **THEN** the sizes the backup records leave out the index of `public.logs` and start with the index of `archive.logs`

#### Scenario: Partitions of a data-excluded partitioned table

- **WHEN** a database has the partitioned table `public.events` with an index and the partitions `public.events_2025` and `public.events_2026`, both holding rows, lists only `public.events` as a table whose data is excluded, and a backup of it completes
- **THEN** the backup records the size of the index of `public.events_2025` and the size of the index of `public.events_2026`, each once, and no separate size for `public.events`

#### Scenario: Dumped tables without indexes

- **WHEN** a database whose tables have no indexes is backed up
- **THEN** the backup completes and records an empty list of index sizes

#### Scenario: The index sizes cannot be read

- **WHEN** reading the index sizes fails while the estimated database size and the dump succeed
- **THEN** the backup completes with its estimated database size and records no index sizes
