## Why

During a logical backup the server commits the stored size to its internal database after every stored megabyte, and the copy of the dump waits for each commit. A fast backup therefore makes many synchronous commits per backup, up to about 100 per second at 100 MB/s of stored output (an estimate, not a measurement), and any slowness of the internal database slows the dump. Physical backups report every 5 seconds, which users already find sufficient.

## What Changes

- While its dump streams to the storage, a logical backup writes its size and duration once every 5 seconds instead of after every stored megabyte. Before the stream and during the sidecar write and publication after it, the two values hold still, as they do today.
- These writes leave the goroutine that copies the dump, so the copy no longer waits for the internal database.
- The duration also advances while a streaming dump produces no output, because every interval writes even when the size has not changed.
- The size and duration a user sees for a running logical backup change in steps of about 5 seconds, starting from 0 until the first refresh. A fast backup updated them more often before.
- The API contract, the schema and the outcome of a finished backup do not change. Nothing here is breaking.

**Out of scope:**

- Computing a running logical backup's duration in the browser, as physical backups do.
- Physical backup progress, restore progress and any other periodic write.
- Changing the size of the connection pool or the durability settings of the internal database.
- Removing the size report made when the dump completes.

This change answers to the root `AGENTS.md` and to `backend/AGENTS.md`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `logical-backups`: adds a requirement on how often a running backup's size and duration refresh, and that the dump does not wait for those writes.

## Impact

- `backend/internal/modules/backups/internal/pipeline`: the progress reporting of a logical backup and its tests.
- `backend/internal/modules/backups/internal/logical`: the interval passed to the pipeline.
- No migration, no contract or generated-code change, no frontend change.
