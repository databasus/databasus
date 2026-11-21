## ADDED Requirements

### Requirement: An imported PostgreSQL string avoids transaction-mode poolers

A transaction-mode pooler may hand a client to another server process between transactions, while backups and restores depend on settings made once per session. When a PostgreSQL connection string is imported in any accepted format, URI, JDBC or key-value, the form SHALL therefore fill the address of a direct or session-mode connection to the same database instead of a known transaction-mode pooler, and SHALL show no message about the replacement.

- A Neon pooler host, whose first label ends with `-pooler` under `neon.tech`, SHALL be filled as the same host without the `-pooler` suffix of its first label. The port, user, password, database name and SSL mode SHALL be filled as the string gives them.
- A Supabase pooler host under `pooler.supabase.com` with port 6543 SHALL be filled with port 5432 on the same host. Every other field SHALL be filled as the string gives it.
- Every other host and port SHALL be filled unchanged. This includes a Supabase pooler host on port 5432, a host outside `neon.tech` whose first label ends with `-pooler`, and a host outside `pooler.supabase.com` on port 6543.
- A host or port the user types into the fields by hand SHALL be kept as typed.

#### Scenario: A Neon pooler string

- **WHEN** a user imports `postgresql://owner:secret@ep-young-dew-123456-pooler.c-8.us-east-2.aws.neon.tech/neondb?sslmode=require&channel_binding=require`
- **THEN** the host is `ep-young-dew-123456.c-8.us-east-2.aws.neon.tech`, the port 5432, the user `owner`, the database `neondb` and the SSL mode `require`
- **AND** the form shows no message about the host

#### Scenario: A Neon pooler string in the key-value format

- **WHEN** a user imports `host=ep-young-dew-123456-pooler.c-8.us-east-2.aws.neon.tech port=5432 dbname=neondb user=owner password=secret sslmode=require`
- **THEN** the host is `ep-young-dew-123456.c-8.us-east-2.aws.neon.tech` and every other field is filled as the string gives it

#### Scenario: A direct Neon string

- **WHEN** a user imports a Neon string whose host is `ep-young-dew-123456.c-8.us-east-2.aws.neon.tech`
- **THEN** the host is filled unchanged

#### Scenario: A pooler-named host outside Neon

- **WHEN** a user imports a string whose host is `db-pooler.example.com`
- **THEN** the host is filled unchanged

#### Scenario: A Supabase transaction pooler string

- **WHEN** a user imports `postgres://postgres.abcdef:secret@aws-0-eu-central-1.pooler.supabase.com:6543/postgres`
- **THEN** the host is `aws-0-eu-central-1.pooler.supabase.com` and the port 5432
- **AND** the user is `postgres.abcdef`

#### Scenario: A Supabase session pooler string

- **WHEN** a user imports a Supabase pooler string on port 5432
- **THEN** the host and port are filled unchanged

#### Scenario: Port 6543 outside Supabase

- **WHEN** a user imports a string whose host is `pooler.example.com` on port 6543
- **THEN** the host and port are filled unchanged

#### Scenario: A pooler host typed by hand

- **WHEN** a user types `ep-young-dew-123456-pooler.c-8.us-east-2.aws.neon.tech` into the host field
- **THEN** the field keeps the typed host
