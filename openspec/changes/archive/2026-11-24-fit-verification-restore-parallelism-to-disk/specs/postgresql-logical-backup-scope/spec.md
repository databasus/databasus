## ADDED Requirements

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
