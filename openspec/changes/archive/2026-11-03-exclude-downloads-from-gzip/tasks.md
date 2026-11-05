Run direct Go commands below from `backend/` after exporting the root `.env`, with the Fedora libedit shim and `TESTCONTAINERS_RYUK_DISABLED=true` prepared as described in `docs/testing.md`, "Commands". Use the existing application, engine and storage fixtures; do not mock project modules.

## 1. Document the response contract

- [x] 1.1 Update `backend/api/openapi.yaml` descriptions for the three download operations: no HTTP gzip, exact logical and agent-binary lengths, and no required physical restore-set length. Keep request shapes, access marks, media types and existing headers. Run `mise run generate-api`; verify contract and generated-code consistency with `mise run backend:lint`.

## 2. Reproduce the defects before changing production code

- [x] 2.1 Add the scenarios "does not compress the backup stream when the client accepts gzip" to `TestDownloadLogicalBackup` and "does not compress the restore stream when the client accepts gzip" to `TestDownloadPhysicalRestoreSet` in the physical restoring API tests. Extend `TestSystemAgentDownload` for the same behavior and exact binary length. Explicitly set `Accept-Encoding: gzip` through the contract client's request editor, assert the response has not been transparently decompressed, and check original file or archive contents and existing download headers. Verify these scenarios fail on the current server because it sends HTTP gzip: `go test -count=1 -run 'TestDownloadLogicalBackup|TestDownloadPhysicalRestoreSet|TestSystemAgentDownload' ./internal/modules/backups ./internal/modules/backups/physicalapitests/restoring ./internal/app/apptest`.
- [x] 2.2 Add encrypted and unencrypted logical download scenarios that explicitly request identity encoding and compare `Content-Length` with the delivered byte count. Use real completed backup files larger than 2 KiB and not an exact multiple of a megabyte so automatic small-response length inference cannot hide the defect. Verify the new scenarios fail for the missing length before the fix: `go test -count=1 -run TestDownloadLogicalBackup ./internal/modules/backups`.

## 3. Exclude binary operations from HTTP gzip

- [x] 3.1 Pass a slice of excluded path regexes into `httpserver.Compress`, retaining the existing extension exclusions. Configure the three anchored expressions from `internal/app/server.go`; leave the physical stream writer, agent binary handler and agent client unchanged. Verify the logical gzip scenario with `go test -count=1 -run '^TestDownloadLogicalBackup$/does_not_compress' ./internal/modules/backups`, then verify physical and agent downloads with `go test -count=1 -run '^(TestDownloadPhysicalRestoreSet|TestSystemAgentDownload)$' ./internal/modules/backups/physicalapitests/restoring ./internal/app/apptest`. Leave the identity-length regression scenarios pending until task 4.1.
- [x] 3.2 Add `TestCompress` in `lib/httpserver` covering JSON and text-static gzip with successful decompression, identity and absent gzip acceptance, all existing excluded extensions, configured exact exclusions, query parameters and neighboring paths. Use generic paths in shared middleware tests. Add an application API assertion that neighboring JSON still receives gzip, observing the default transport's `HTTPResponse.Uncompressed` when using the contract client. Verify with `go test -count=1 -run 'TestCompress|TestSystemAgentDownload' ./internal/lib/httpserver ./internal/app/apptest`.

## 4. Restore the exact logical download length

- [x] 4.1 Add `SizeBytes int64` to the logical download service result, derive it from the loaded backup's pre-encryption `BackupSizeMb`, and map it to the generated response's `ContentLength`. Extend the explicit-gzip logical download scenario from task 2.1 to assert both absent `Content-Encoding` and `Content-Length` equal to the delivered byte count on the same response, for encrypted and unencrypted backups. Extend the existing shared engine download-size assertion to request identity encoding and check the announced byte length, retaining its checks for the recorded size. Verify the gzip, identity and fractional-megabyte cases with `go test -count=1 -run TestDownloadLogicalBackup ./internal/modules/backups`, then verify the four engines through their existing API coverage with `go test -count=1 -run '^TestBackupAndRestore(Mysql|Mariadb|Mongodb|Postgresql)$' ./internal/modules/backups/e2e/mysql ./internal/modules/backups/e2e/mariadb ./internal/modules/backups/e2e/mongodb ./internal/modules/backups/e2e/postgresql`.

## 5. Check the complete change

- [x] 5.1 Run `mise run backend:format` and `mise run backend:lint`; run the command in "Change verification" and record its result.

After these checks, run the mandatory implementation review and resolve blocking findings before handing over the change.

## Verification evidence

- Before the fix, the download regression tests failed because logical backups, physical restore sets and both agent architectures received `Content-Encoding: gzip`. The separate identity-encoding tests for encrypted and unencrypted logical backups failed because `Content-Length` was absent and the client reported `ContentLength: -1` for a 21063-byte file.
- After the compression change, both task 3.1 commands passed. The task 3.2 middleware and system API tests passed, covering JSON and static gzip, excluded extensions and paths, query parameters, neighboring responses and the binary headers.
- After restoring logical lengths, task 4.1's logical API tests and all four engine API test packages passed. The logical scenarios check gzip and identity requests, encrypted and unencrypted storage, and exact lengths for files larger than 2 KiB whose sizes are not whole megabytes.
- `mise run generate-api`, `mise run backend:format`, `mise run backend:lint`, `mise run frontend:lint`, `pnpm --dir frontend build` and the changed production files' `gopls check` passed.
- The first full run exposed the existing damaged-file test: restoring `Content-Length` makes its truncated download return `io.ErrUnexpectedEOF`. The updated test checks the error, delivered bytes, recorded length and unchanged completed status. `go test -count=1 -run '^TestVerificationHelpers$' ./internal/app/apptest` passed, followed by backend formatting and linting.
- Review identified a weak agent-length fixture: both architectures now use 6 KiB binary contents so automatic small-response length inference cannot supply the tested header. `go test -count=1 -run '^TestSystemAgentDownload$' ./internal/app/apptest`, backend formatting and backend linting passed after the fixture change.
- The final `mise run test` after the fixture change passed every backend package, all 183 frontend test files (2363 tests), and the upgrade test from `v3.64.0` to the current build.

## Change verification

```sh
mise run test
```
