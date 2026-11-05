## Why

A backup fails at its very end when the dump tool has already exited successfully and the storage is still accepting the last of the data. The server waits 10 seconds (one minute for PostgreSQL logical tools) and reports a failed backup, so on a connection below about 13 Mbit/s a backup that runs for hours can fail on every attempt. This is a regression of the engine refactoring: v3.64.0 read the tool's whole output before it waited for the tool.

## What Changes

- A backup keeps waiting for the storage after the dump tool exits, until the storage has accepted the data. This covers logical PostgreSQL, MySQL, MariaDB and MongoDB dumps. Full and incremental physical PostgreSQL backups get the same fix but keep their stall limits, so they wait up to those limits: 60 seconds without progress once the storage has accepted data, and 10 minutes before the first accepted data.
- The time limit that follows a tool's exit applies only while nothing arrives from the tool's output. A process the tool left behind that holds the output open still fails the backup after that limit, as it does now.
- PostgreSQL logical tools, `pg_dump` and `pg_restore`, return to the 10-second limit the other tools use, because the one-minute limit existed only to work around this defect. For `pg_restore` that limit bounds standard input, so a restore's wait for its input after the tool exits drops from one minute to 10 seconds.

Out of scope:

- A new limit on a storage that stops accepting data. A logical tool that is running can already wait on a stalled storage until the context ends, so this change adds no new kind of hang.
- The physical backup's byte-stall limits of 60 seconds and of 10 minutes before the first accepted data, which keep applying during the tool's run and after its exit.
- How standard input reaches restore tools. Its mechanism does not change; only the PostgreSQL value above does.
- Retries of failed backups and any change to how failures are classified or reported.
- Error output of tools, which is written to memory and has no slow consumer.

Nothing here breaks callers, schemas, configs or APIs.

This change answers to `backend/AGENTS.md`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `backup-streaming`: adds the requirement that a backup does not fail while the storage is still accepting data after the dump tool has exited.

## Impact

- `backend/internal/lib/process`: how a tool's standard output is delivered to its writer, which every caller of the package shares.
- `backend/internal/engines/postgresql/logical/client_tool.go`, `backend/internal/engines/mysqlfamily/client_tool.go`, `backend/internal/engines/mongodb/client_tool.go`: the limit's value or its comment. The value in `postgresql/logical` also applies to `pg_restore`.
- `backend/internal/engines/postgresql/physical/base_backup.go` and `backend/internal/modules/backups/internal/physical/backuping/stream.go`: a stalled stream releases a write blocked on the storage and covers the last encrypted chunk, so the stall is reported.
- Tests of `lib/process`, of `pg_basebackup` and of the physical stream.
- `docs/architecture.md`: one sentence on external tools.
- No API, migration or frontend changes.
