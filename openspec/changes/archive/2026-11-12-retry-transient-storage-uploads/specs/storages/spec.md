## ADDED Requirements

### Requirement: S3 and Azure Blob uploads outlast temporary provider refusals

Saving a file to an S3 or Azure Blob storage SHALL survive a temporary refusal of one request of that upload. A temporary refusal is a request timeout (`408`), too many requests (`429`), any server error (`5xx`) or a network failure before the provider answers. The upload SHALL send the refused request again with the same content, after a delay that starts at one second, doubles with each further refusal of the same request and never exceeds 30 seconds. It SHALL give up on that request after ten such repeats, and the save SHALL then fail. These repeats SHALL come in addition to any short retries the storage client library makes. Each request gets a fresh budget once the provider accepts it, so refusals spread across a long upload never add up to a failure.

For S3 storage this SHALL cover every request of an upload that is safe to send twice: each piece of a file split into pieces, the start of each multipart object, a file stored as one object and the manifest of a file split into pieces. The request that completes a multipart object is not repeated by the upload. For Azure Blob storage it SHALL cover every request of an upload.

A save that fails after the repeats SHALL report which request failed and how many attempts it took. It SHALL leave the storage as any other failed save does, with no file published under the requested name.

#### Scenario: A piece is refused for longer than the storage client library retries

- **WHEN** an S3 storage answers the first several uploads of one piece of a large backup with `503 Service Unavailable` and accepts it within the repeat budget
- **THEN** the backup is saved, and reading it back returns exactly the bytes that were written

#### Scenario: A small file is refused for a while

- **WHEN** an S3 storage answers a WAL segment that fits in one piece with `503` several times and then accepts it
- **THEN** the WAL segment is saved and reads back intact

#### Scenario: An Azure Blob piece is throttled

- **WHEN** an Azure Blob storage answers uploads of one piece of a large backup with `503 Server Busy` more times than the storage client library retries, and then accepts it within the repeat budget
- **THEN** the backup is saved and reads back intact

#### Scenario: Refusals spread over a long upload

- **WHEN** different pieces of one upload are each refused a few times, and the total exceeds ten
- **THEN** the upload succeeds, because each piece has its own budget

#### Scenario: The provider stays unavailable

- **WHEN** the storage keeps answering one piece with `503` after ten repeats
- **THEN** the save fails with an error that names the piece and the number of attempts, and no file is published under the requested name

### Requirement: Permanent refusals and cancellation end an upload at once

A refusal that is not temporary SHALL fail the save on its first occurrence, without a repeat by the upload. Examples are access denied, a missing bucket or container, and an invalid storage class. Cancelling a save SHALL end it promptly, both while a request is in flight and while the upload waits before a repeat.

#### Scenario: Access is denied

- **WHEN** an S3 storage answers the first piece of an upload with `403 AccessDenied`
- **THEN** the save fails after that single request, and nothing is left under the file's name

#### Scenario: Cancelled while waiting

- **WHEN** a backup is cancelled while its upload waits before repeating a refused piece
- **THEN** the save ends without waiting out the delay and reports the cancellation

### Requirement: Upload repeats are visible in the logs without exposing credentials

Every repeat after a temporary refusal SHALL produce a warning in the application log. The warning SHALL identify the stored file, the refused request and its piece number where there is one, the attempt number, the attempt limit and the provider's error. It SHALL contain no access key, secret key, connection string, account key, signature or request body.

#### Scenario: A refused piece is logged

- **WHEN** an upload repeats a piece after a `503`
- **THEN** the log has a warning with the object name, the piece number, the attempt and the limit, so an operator sees the struggling provider before any backup fails

#### Scenario: The warning carries no secrets

- **WHEN** an upload to an S3 storage with an access key and a secret key repeats a refused piece
- **THEN** neither key nor the request signature appears in the warning
