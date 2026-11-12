## Context

See proposal.md, "Why", for the failure. This section covers only the current mechanics that shape the fix.

**S3 upload.** `bucket.saveFile` (`backend/internal/modules/storages/internal/providers/s3/bucket.go:65`) reads the stream in 16 MiB pieces. A stream that fits in one piece goes to `uploadSingleObject` (`bucket.go:473`) as one `PutObject`. A larger stream goes to `uploadChunked` (`bucket.go:490`). That function starts a multipart object per 1,000 pieces (`bucket.go:529`), sends each piece through `putInnerPart` → `PutObjectPart` (`bucket.go:622`, `bucket.go:637`), completes the object (`bucket.go:586`) and finally writes the manifest (`chunked_manifest.go:69`). Any error calls `abandon`, which aborts the multipart upload and deletes completed objects (`bucket.go:542-594`).

The only retry on these requests is inside minio-go v7.2.1 (`api.go`, `executeMethod`). It makes at most 10 attempts, and each pause is a random value between zero and `min(1s, 200ms * 2^n)`. The pauses add up to at most about 8.4 seconds. The cap is the package variable `minio.DefaultRetryCap`, which no client option overrides. minio-go treats `408`, `429`, `499`, `5xx` and `520` as retryable and rewinds a seekable body between attempts.

**Azure Blob upload.** `blobContainer.saveFile` (`backend/internal/modules/storages/internal/providers/azureblob/blob_container.go:58`) stages 16 MiB blocks with `StageBlock` (`blob_container.go:106`), uploads an empty blob with `UploadStream` when there are no blocks (`blob_container.go:130`), and commits the block list (`blob_container.go:149`). The client is built with `policy.RetryOptions{MaxRetries: 0}` (`blob_container.go:411`). In azcore v1.22.0 zero means the default of three retries, with delays of about 0.8, 2.4 and 5.6 seconds, so the budget is about nine seconds. The SDK honors `Retry-After` but gives up at once when that header asks for more than its maximum delay. It writes its retry events only to its own logger, which the application does not wire up.

**Restore side.** Reads already repeat after transient failures. S3 uses `reassemblingReader.resumeAfterFailure` (`s3/reassembling_reader.go:293`) with the classifier `isRetryableReadError` (`s3/reassembling_reader.go:349`). Azure Blob uses `iox.ResumingReader` with `isRetryableAzureError` (`azureblob/blob_container.go:193`). Both readers drive `iox.ReadRetryPolicy` (`backend/internal/lib/iox/read_retry_policy.go`). That policy counts attempts since the last progress, waits from 1 second doubling up to 30 seconds and allows 10 attempts. A completed piece resets the count.

**Other storages.** FTP (`ftp/ftp_server.go:59`), SFTP (`sftp/sftp_server.go:59`) and NAS (`nas/nas_share.go:70`) each write a file as one continuous stream over one connection. A broken connection there ends the whole transfer, and nothing in memory could be resent, so they do not have this failure mode. The local storage writes to the host disk.

Constraints come from `backend/AGENTS.md`: code with no domain knowledge goes into `internal/lib`, every error of a package lives in its `errors.go`, a function with four or more parameters takes a named struct, and logging goes through the context logger with `snake_case` attributes. `docs/architecture.md`, "Backup data flow", requires bounded memory and backpressure, and states that a slow storage only slows the backup. A wait before a repeat is a slow storage in that sense, so the architecture does not change.

## Goals / Non-Goals

**Goals:**

- One retry rule for uploads, shared by S3 and Azure Blob, with the same budget, classification shape and warning format.
- One retry policy type for both reads and writes, so the backoff numbers live in one place.
- Tests that reproduce the refusal against a real MinIO and a real Azurite, with no unit-test fakes of the providers.

**Non-Goals:**

- Changing how reads resume, or their budget.
- Changing the piece size, the number of pieces per object, or the order of requests.

## Decisions

### 1. Repeat each upload request in our own loop, on top of the client library's retries

Each upload request that is safe to send twice runs inside a loop. The loop calls the library and classifies the error. A temporary error is registered with the retry policy, logged as a warning and followed by a wait, after which the loop sends the request again. Any other error is returned at once. The piece is already in a byte slice, so every attempt sends a fresh reader over the same bytes.

Rejected alternatives:

- **Raise minio-go's `Options.MaxRetries`.** The pause cap is the package variable `minio.DefaultRetryCap` (1 s). Changing it would affect every minio client in the process, including reads and connection tests. Without changing it, 300 attempts mean 300 near-immediate resends of 16 MiB to a node that just said it is busy.
- **Raise the Azure SDK's `RetryOptions` for upload clients only.** This is the smallest change for Azure, and the SDK already honors `Retry-After`. It was rejected for three reasons. The SDK does not let us log each repeat, so the logging requirement in the delta spec would need a custom per-retry policy. The SDK stops retrying when `Retry-After` exceeds its cap, so a throttled account could still fail fast. And S3 would end up with a different mechanism, a different delay curve and different logs for the same rule.
- **Retry in an `http.RoundTripper`.** The transport would have to replay a signed body and know which S3 and Azure requests are safe to repeat, which only the provider code knows. A transport also cannot repeat a `CompleteMultipartUpload` safely, so it would need the same per-request knowledge anyway.
- **Retry the whole `SaveFile` above the provider.** The dump stream cannot be rewound, so a whole-file retry is only possible by re-running the backup, which the schedule already does.

### 2. Share the policy as `iox.TransferRetryPolicy`

`iox.ReadRetryPolicy` already has the right budget and the "attempts since progress" counter. For a write, progress means the provider accepted the request. The change renames it to a direction-neutral name:

- `ReadRetryPolicy` → `TransferRetryPolicy`, and `ReadRetryPolicySpec` → `TransferRetryPolicySpec`. The file is renamed to `transfer_retry_policy.go` and its test to match.
- `RegisterFailedAttempt(hasDeliveredBytes bool)` → `RegisterFailedAttempt(hasMadeProgress bool)`.
- `NewReadRetryPolicy` → `NewTransferRetryPolicy`.
- `ResetAttemptsAfterCompletedRead()` → `ResetAttempts()`.
- The constants `defaultMaxReadAttemptsWithoutProgress`, `defaultReadRetryBaseDelay` and `readRetryMaxDelay` → `defaultMaxAttemptsWithoutProgress`, `defaultRetryBaseDelay` and `retryMaxDelay`.
- `ErrReadRetryBudgetExhausted` → `ErrRetryBudgetExhausted`, with the message "transfer retries exhausted".

Both readers and their tests move to the new names in the same change. No old name remains. Every user of these names is inside the backend, so nothing outside breaks.

Rejected alternatives:

- **Use `ReadRetryPolicy` for writes as it is.** A write path that calls `RegisterFailedAttempt(false)` on a "read" policy hides its meaning, and the repository's naming rule requires names to state intent.
- **A separate write backoff.** Two copies of the same delay curve and budget would drift apart.

### 3. A shared loop in `iox`: `RetryRequest`

A new unit `iox.RetryRequest(ctx, RequestRetrySpec) error` in `backend/internal/lib/iox/request_retry.go` owns the loop. Its spec carries the policy spec, the logger, the `IsRetryableError func(error) bool` classifier, a `RequestDescription` string such as `part 368 of backups/x.part000005` or `block 12 of backups/x`, the log attributes that identify the request, and the `Send func(ctx) error` call. The classifier field has the same name as in `ResumingReaderSpec` (`iox/resuming_reader.go:39`), so reads and writes use one word for one concept. On exhaustion the loop returns an error that wraps `ErrRetryBudgetExhausted` and the last provider error, and whose text names `RequestDescription` and the attempt count. A failed save therefore reports which request failed, as the delta spec requires. Cancellation is checked after each failure and during the wait, and is returned as the context's error.

A fresh policy is created for each call, so every request gets its own budget. This is how the spec's "refusals spread over a long upload never add up" is met.

Rejected alternatives:

- **One loop per provider.** The two loops would be the same code with a different classifier.
- **A new `internal/lib/retry` package.** The policy already lives in `iox`, and the loop is its only new caller. Splitting them would move the policy without a gain.

### 4. Classify temporary errors per provider, and share the classifier with reads

S3 gets `isRetryableS3Error` in its own file in the `s3` package. It returns true for a `minio.ErrorResponse` with status `408`, `429` or `>= 500`, and true for any other non-nil error, which means a network failure before an answer. `isRetryableReadError` keeps its read-only terminal checks (`ErrRangeNotHonoured`, `ErrChunkSizeMismatch`, `ErrChunkChecksumMismatch`) and then delegates to it. Writes use `isRetryableS3Error` directly, so write code does not depend on read-only error kinds.

Azure Blob reuses `isRetryableAzureError` (`blob_container.go:193`) unchanged. It has no read-specific checks, and its name fits both directions. Code uses one word for this classification, "retryable", matching `IsRetryableError` and `isRetryableAzureError`.

Context cancellation is never treated as temporary: the loop checks `ctx.Err()` before it asks the classifier.

Rejected alternative: **rename `isRetryableReadError` to a transfer name and use it for both.** That works, because the read-only error kinds never occur on a write, but it puts read-only knowledge into the write path.

### 5. Which requests are repeated

| Request | Repeated | Why |
|---|---|---|
| S3 piece (`PutObjectPart`) | yes | The same part number with the same bytes replaces the earlier attempt. This covers about 6,500 requests of a 110 GB backup. |
| S3 start of multipart (`NewMultipartUpload`) | yes | A lost success leaves an unused upload ID. `abandon` and the delete path's `abortDerivedMultipartUploads` already clean those up. |
| S3 single object (`uploadSingleObject`) | yes | Rewriting a key with the same bytes is idempotent. WAL segments take this path constantly. |
| S3 manifest (`writeManifest`) | yes | Same as the single object. |
| S3 completion (`CompleteMultipartUpload`) | no | After a lost success, a repeat answers `NoSuchUpload`. A safe repeat needs an existence check first, which is out of scope. |
| Azure `StageBlock` | yes | Staging the same block ID again replaces the uncommitted block. |
| Azure empty blob (`UploadStream`) | yes | Idempotent overwrite with empty content. |
| Azure `CommitBlockList` | yes | The SDK commits with the "latest" block list, so a repeat after a lost success commits the same blocks again and succeeds. |

### 6. Budget values and test override

Production uses the policy defaults: 10 repeats, a first delay of 1 second, doubling, and a 30-second cap. That gives about 181 seconds of waiting per request (1 + 2 + 4 + 8 + 16 + 5 × 30), plus up to about 8 seconds of library retries per attempt. Each provider gets an unexported `uploadRetry iox.TransferRetryPolicySpec` field whose zero value takes these defaults. Tests set a 1 ms base delay, the same way S3 tests already shrink `partLayout` (`s3/provider.go:20`). No setting, environment variable or `Deps` field is added, because nobody needs to tune it in production.

Rejected alternative: **an unlimited or user-configured budget.** An unlimited budget can hold the dump transaction open indefinitely. A setting adds a form field and a migration for a value nobody has asked to change.

### 7. Logging

The loop logs `"storage request refused, retrying"` at Warn with `attempt`, `max_attempts` and `error`, plus attributes from the caller: `object_key` and `part_number` for S3 pieces, `file_name` and `block_number` for Azure blocks, matching the `file_name` key of the Azure provider's other upload logs, and `request` naming the operation, such as `put_object_part` or `stage_block`. The provider error text is the S3 or Azure error message. Neither library puts credentials or signatures in it. Redaction of anything else stays in the log handler, as `backend/AGENTS.md` requires. `uploadChunked`, `putInnerPart`, `uploadSingleObject` and `writeManifest` start taking the logger, because each of them now sends requests through the loop.

### 8. Tests

A new test helper, `providertest.RefusingProxy`, fronts a real MinIO or Azurite with `httputil.ReverseProxy`. It answers the first N requests that match a predicate with a given status and body, and passes everything else through. It counts the matching requests. It lives next to the existing `S3ClassProxy` in `providertest`. The existing proxy rejects every write with one fixed answer, while this one needs a count and a predicate. The two do not merge cleanly, so they stay separate.

The S3 scenarios run in `s3/provider_test.go` against MinIO:

- `503` on more piece uploads than minio-go retries, so the save only succeeds through the new loop, and the file then reads back byte for byte;
- `503` on a single-object write;
- `403` on the first piece, which gives exactly one request and leaves no objects;
- a refusal that outlasts the budget, which gives `ErrRetryBudgetExhausted` and leaves no objects;
- cancellation during the wait.

The Azure scenarios run in `azureblob/provider_test.go` against Azurite: refusals on staged blocks beyond the SDK's three retries, and `403` without a repeat. `iox` gets unit tests for the loop's budget, the fresh budget per call, classification, the exhaustion error text, the Warn record and cancellation, because it is a pure function of its inputs. The S3 scenarios also include several pieces each refused a few times, more than ten refusals in total, and a captured logger that checks the warning's attributes and the absence of the access key, the secret key and the signature.

Tests that refuse more often than minio-go retries also wait out minio-go's own jittered pauses, up to about 8 seconds per test. That is acceptable. The alternative would be changing the package variable `minio.DefaultRetryUnit` in tests, which is global state and unsafe for parallel tests.

## Risks / Trade-offs

- [The source stalls while the upload waits, up to about three to four minutes per refused request.] → `pg_dump`, `mysqldump`, `mongodump` and `pg_basebackup` block on a full pipe, as they already do on a slow storage. The budget is bounded, so the wait cannot hold a dump transaction open indefinitely. A source-side idle timeout shorter than the wait would fail the backup the same way it fails on a slow link today.
- [Resending 16 MiB up to about 100 times per piece in the worst case wastes bandwidth against an unhealthy provider.] → Delays grow to 30 seconds, and the budget ends the attempt. The cost occurs only while the provider keeps refusing.
- [A request that the provider accepted but whose answer was lost is sent again.] → Only idempotent requests are repeated (Decision 5). Multipart completion is excluded.
- [The rename touches restore code.] → The rename does not change behavior. The existing reader tests stay and must pass with the new names.
- [Tests take longer because of minio-go's real pauses.] → A few seconds per scenario. Only the scenarios that must outlast the library's own retries use `503`; the rest use a server error the library does not retry itself.
- [An S3 endpoint that cannot be reached at all now fails an upload only after about four minutes instead of a few seconds, because a refused connection is a network failure before an answer.] → This is the behavior the delta spec asks for: a node restart or a dropped connection looks the same. Application tests that need a storage which fails at once, such as the WAL queue and storage-failure scenarios, use an S3 stub that denies every request with `403`, which stays a permanent failure.

## Migration Plan

No data or configuration migration. The behavior takes effect on the next upload after deployment. Rolling back restores the previous short retry budget.
