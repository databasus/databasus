## Why

A large backup to S3 or Azure Blob storage is uploaded in 16 MiB pieces, one after another, while the database dump is still running. When the provider answers one piece with a temporary refusal, such as `503 Service Unavailable` from a busy Backblaze B2 node, the upload library repeats the request for only about five to ten seconds before it gives up. The whole backup then fails and is discarded. A 110 GB logical backup streams for about 12 hours and sends about 6,500 pieces, so one refusal burst longer than ten seconds throws away half a day of work. Users of S3-compatible providers that document `503` as an ordinary "try again later" answer see most of their large backups fail this way.

The data of the refused piece is still in memory, and both providers accept the same piece again, so the system can safely keep trying for a few minutes before failing the backup. Restores already do this for dropped reads.

## What Changes

- An upload to S3 storage repeats a piece that the provider refused with a temporary error. Temporary errors are a request timeout, too many requests, a server error or a network failure. The repeats come on top of the client library's own short retries and use delays that grow from one second up to 30 seconds, for up to ten repeats per piece.
- The same applies to every other S3 request of an upload that is safe to send twice: starting a multipart upload, writing a small file as one object (WAL segments, sidecars, small dumps) and writing the manifest of a chunked file.
- An upload to Azure Blob storage gets the same repeats with the same budget. Its client library currently retries for about nine seconds. Staging a block, uploading an empty file and committing the block list are all safe to send twice, so every request of an Azure upload is covered.
- A permanent refusal, such as access denied or a missing bucket, still fails the upload at once, with no repeats.
- Every repeat is logged as a warning that names the file, the piece and the attempt, so a struggling provider is visible before it fails a backup.
- Cancelling the backup stops the upload at once, also while it waits before a repeat.
- The restore-side retry policy and the S3 classification of temporary errors are shared with the upload side under names that fit both directions. This is an internal rename with no change to restore behavior.

No API, setting, schema or stored data changes. Nothing breaks for callers.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `storages`: adds requirements for how S3 and Azure Blob uploads handle temporary and permanent provider refusals, and what a repeat logs.

## Out of scope

- FTP, SFTP, NAS (SMB) and local storages. They write each file as one continuous stream over one connection, so there is no refused piece to send again. A broken connection fails the file. Surviving that would need protocol-specific resume and buffering of unacknowledged data, which is a separate feature.
- Repeating the request that completes an S3 multipart object. If the provider completed the object but the answer was lost, a second request fails with "no such upload", so a safe repeat would first have to check whether the object exists. This request is sent once per 1,000 pieces, about seven times for a 110 GB backup, and the client library already retries it for a few seconds.
- Honoring a `Retry-After` header on S3 uploads. The S3 client library does not expose it, and the growing delay covers the same need.
- Any user-visible setting for the retry budget.
- Restore behavior. Reads already resume after dropped connections, and this change keeps that behavior as it is.
- Retrying a whole failed backup, which the backup schedule already covers.

## Impact

- Backend only, in the storages module: the S3 and Azure Blob providers and the shared retry policy in `backend/internal/lib/iox`. Test support gains a proxy that refuses chosen requests with a chosen status, in the storages provider test helpers.
- While an upload waits before a repeat, it stops reading the dump, so the database dump tool blocks for up to about four minutes for one piece. Slow uploads already cause pauses of this kind.
- In the worst case, one piece is sent about 100 times before the backup fails: ten repeats, each with up to ten attempts inside the client library.
- The change answers to the root `AGENTS.md` and to `backend/AGENTS.md`.
