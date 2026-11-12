## 1. Shared retry policy

- [x] 1.1 Rename `iox.ReadRetryPolicy` to `iox.TransferRetryPolicy` as design.md Decision 2 describes: the type, its spec, its constructor, its constants, `RegisterFailedAttempt(hasMadeProgress)`, `ResetAttempts()`, `ErrRetryBudgetExhausted` and the file names. Move `iox.ResumingReader`, the S3 `reassemblingReader` and their tests to the new names, with no old name left (`rg -n "ReadRetry|ReadAttempts|AfterCompletedRead" backend` finds nothing). Verify with `go -C backend test -count=1 ./internal/lib/iox/ ./internal/modules/storages/internal/providers/s3/ ./internal/modules/storages/internal/providers/azureblob/`.
- [x] 1.2 Add `iox.RetryRequest` with `RequestRetrySpec` in `backend/internal/lib/iox/request_retry.go`, per design.md Decisions 3 and 7: a fresh policy per call, a Warn record per repeat with the caller's attributes, an exhaustion error that wraps `ErrRetryBudgetExhausted` and the last error and names `RequestDescription` and the attempt count, and an immediate return on a permanent error or on cancellation, including during the wait. Cover these in `request_retry_test.go`: success after refusals; exhaustion, asserting the error names the request and the attempt count; a permanent error with a single call; separate calls that are each refused a few times, more than ten in total, and all succeed; the Warn record's `attempt` and `max_attempts` and the caller's attributes, read from a captured logger; and cancellation during the wait. Verify with `go -C backend test -count=1 ./internal/lib/iox/`.

## 2. Refusing test proxy

- [x] 2.1 Add `providertest.RefusingProxy` per design.md Decision 8: a reverse proxy in front of a target URL that answers the first N requests matching a predicate with a given status and body, passes the rest through and counts the matched requests. Verify with a test in `providertest` that sends matching and non-matching requests to an `httptest` target and checks the answers and the count: `go -C backend test -count=1 ./internal/modules/storages/internal/providers/providertest/`.

## 3. S3 uploads

- [x] 3.1 Extract `isRetryableS3Error` in the `s3` package and make `isRetryableReadError` delegate to it after its read-only checks, per design.md Decision 4. Verify that the existing reader tests pass: `go -C backend test -count=1 -run 'TestReassemblingReader' ./internal/modules/storages/internal/providers/s3/`.
- [x] 3.2 Add the `uploadRetry` field to the S3 `Provider` and route `PutObjectPart`, `NewMultipartUpload`, `uploadSingleObject` and `writeManifest` through `iox.RetryRequest`, leaving `CompleteMultipartUpload` as it is (design.md Decisions 5 and 6). Pass the logger into `uploadChunked`, `putInnerPart`, `uploadSingleObject` and `writeManifest`, and give each call a `RequestDescription` that names the request, the piece number where there is one, and the object key. Each attempt sends a fresh reader over the same bytes, and the object hash and the part list change only after a piece is accepted. Verify with `go -C backend build ./...`.
- [x] 3.3 Add the S3 scenarios from design.md Decision 8 to `TestSaveFile` in `s3/provider_test.go`, using `RefusingProxy` in front of MinIO and a 1 ms retry delay: more piece refusals than minio-go retries, then a byte-exact read-back; a refused single-object write; `403` on the first piece with one request and no objects left; a budget-exhausting refusal that gives `ErrRetryBudgetExhausted`, names the refused piece and the attempt count in the error, and leaves no objects; several pieces each refused a few times, more than ten refusals in total, with a successful save; a storage with an access key and a secret key whose refused piece produces a Warn record, read from a captured logger, with `object_key`, `part_number`, `attempt` and `max_attempts`, and without either key or the request signature; and cancellation during the wait. Verify with `go -C backend test -count=1 -run 'TestSaveFile' ./internal/modules/storages/internal/providers/s3/`.

## 4. Azure Blob uploads

- [x] 4.1 Add the `uploadRetry` field to the Azure Blob `Provider` and route `StageBlock`, the empty-blob `UploadStream` and `CommitBlockList` of `saveFile` through `iox.RetryRequest` with `isRetryableAzureError`, logging `file_name` and `block_number`. The block buffer is read through a fresh reader on each attempt. Verify with `go -C backend build ./...`.
- [x] 4.2 Add Azure scenarios to `azureblob/provider_test.go`, using `RefusingProxy` in front of Azurite and a 1 ms retry delay: `503` on staged blocks more times than the SDK retries, then a byte-exact read-back; and `403` on a staged block, which fails after one proxied request. Verify with `go -C backend test -count=1 ./internal/modules/storages/internal/providers/azureblob/`.

## 5. Finish

- [x] 5.1 Run `mise run backend:format` and `mise run backend:lint`, and fix every finding in the files this change touches.
- [x] 5.2 Point the application tests that need a storage which fails at once at `apptest.StartDenyingS3` instead of the unreachable `http://127.0.0.1:1`, because an unreachable endpoint is now retried for minutes: `StorageNamespace.MakeUnavailable`, the WAL streaming `useUnavailableStorage` and the physical backup storage-failure scenario. Verify with `go -C backend test -count=1 ./internal/modules/backups/physicalapitests/walstreaming/ ./internal/modules/backups/physicalapitests/backuping/ ./internal/modules/storages/`.

## Change verification

```sh
mise run test
```
