# Verification evidence

## Comparable builds

Both server binaries were built with Go 1.26.10 on linux/amd64, `CGO_ENABLED=0`, empty `GOFLAGS` and no linker overrides, using `go -C backend build -o <temporary path> ./cmd/databasus`. The baseline was built before deleting the provider. Neither binary embeds frontend files; the frontend is served from the filesystem. Binaries and complete `go version -m` output are retained outside the repository in `/tmp/databasus-remove-rclone.ECDzYN`.

| Build | Bytes |
| --- | ---: |
| Before removal | 134,789,184 |
| After removal | 58,489,902 |

Both `go version -m` outputs report identical build settings. The measured reduction is 76,299,282 bytes (56.61%). This measures these builds only, not a promised threshold for future releases or a vulnerability reduction.

`go list -deps ./cmd/databasus ./cmd/verification-agent`, `go list -m all` and resulting module build information contain no rclone integration or adapter modules. Account OAuth remains a consumer of `golang.org/x/oauth2`; its version is unchanged. `go mod tidy` removed 154 unused module requirements without changing any retained module version. Newly explicit indirect requirements come from existing generator or dependency test graphs.

`mise run security:go` passed: no reachable vulnerabilities. The scanner also reported two vulnerabilities in imported packages and one in required modules whose affected symbols are not called. No suppressions were added.

## Website layout and claims

The website static export passed. Source and exported HTML, text and JavaScript contain no rclone, Dropbox or `70+` offer. A headless Chromium check opened the home, MySQL, MongoDB and storage pages in all seven languages at 320 and 1280 CSS pixels. All 56 page/viewport combinations passed; 140 storage summary checks found no clipping, and no page overflowed horizontally. The temporary server and browser were stopped in `finally`, and their container was removed.

## Released images

The rclone-only regression pins the published 3.67.1 multi-platform image index `sha256:663dd65f995d31802b41d4eb20399798285bff7367cc4f91dcb5617f6af83b80`, resolved with `docker buildx imagetools inspect databasus/databasus:v3.67.1`. The independent Google Drive and combined regressions retain the pinned 3.67.0 image. Snapshot inspection uses each scenario's own release image.

## Completed checks

- `mise run lint`: passed, including backend generation, architecture, Go lint, frontend types and dictionaries, shell and Python checks.
- `TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -count=1 -timeout 15m ./internal/app ./cmd/databasus ./internal/modules/storages ./migrations ./tools/archlint`: application, CLI, storage API and migration checks passed in the combined run. The architecture fixture was corrected and the complete architecture package passed separately. Application and storage refusal checks passed again after the final fixture and key-file assertions.
- `TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -count=1 -p=1 -parallel=1 -timeout 15m ./internal/modules/storages/internal/providers/...`: passed. Fedora's libedit shim was applied before backend checks.
- `go -C backend test -count=1 ./internal/lib/iox`: passed as part of the focused run.
- `pnpm --dir frontend test src/features/edit-storage src/pages/storages src/pages/databases/backup-config src/shared/i18n/dictionaries.test.ts`: 349 tests passed.
- `pnpm --dir frontend typecheck`: passed.
- `pnpm --dir frontend test:browser --project=ui`: 315 tests passed, including the exact six-provider creation list, remaining provider forms and details, read-only access, secrets and reflow. Its production build and harness checks passed without reference updates.
- `pnpm --dir website lint` and `pnpm --dir website build`: passed.
- `mise run backend:test-upgrade`: passed, including the existing general upgrade, independent Google Drive refusal, rclone-only 3.67.1 refusal/recovery and combined 3.67.0 refusal/recovery across two workspaces.
- `go -C backend build ./cmd/verification-agent`: passed with output redirected to the temporary build directory.
- `openspec validate remove-rclone --strict` and `git diff --check`: passed.

The initial combined backend command passed all application, CLI, storage API and migration tests; its new architecture fixture failed because an agent import went outside the allowed directories. The fixture was moved into `internal/lib`, and the complete architecture package passed afterward.

## Final verification

`mise run test` passed: all selected backend packages, including the long WAL streaming and storage scenarios, and all 194 frontend test files (2,863 tests). The final backend lint and standalone frontend production build also passed. No test was skipped to make the change pass, no security exception was added, and no visual reference was updated.

Remaining references outside OpenSpec are limited to historical SQL, focused historical/refusal/recovery tests, runtime eligibility/removal checks and executable dependency prohibitions. Regenerated API contracts and all current frontend, website and README content contain no retired variant or unsupported offers.

## Additional coverage

The production dependency tests use synthetic `github.com/rclone/Proton-API-Bridge` and `github.com/Files-com/files-sdk-go/v3/lib` packages and reject their `testing` and assertion-library imports. Both tests passed with the unconditional checker. In a separate temporary copy, restoring the original allowlist caused both tests to fail because those imports were accepted; no working-tree production source was changed for this check.

`TestSaveFileThroughput` starts four writers and stores exactly 24 pairs of a 16 MiB segment and 1 KiB metadata file. It applies and verifies 50 ms container egress delay for all five network providers and leaves local storage unshaped. The 12-second deadline covers the complete write workload. The test then reads every file and compares its contents after measurement. Its focused run passed for all six providers:

| Provider | Write duration |
| --- | ---: |
| Local | 0.073 s |
| SFTP | 5.264 s |
| NAS / SMB | 7.533 s |
| FTP | 9.554 s |
| Azure Blob | 5.390 s |
| S3 | 2.902 s |

The complete architecture test package and backend lint passed again. Blank lines separate the actor, settings, workspace, key-file and record-verification steps in the affected tests.

The full storage provider suite passed with `-p=1 -parallel=1`, including the new workload and existing streaming, latency, cancellation and secret tests. The workload also passed with ordinary Go test parallelism. The focused startup and retired-provider API tests passed again.

`DATABASUS_IMAGE=databasus-candidate:local mise run backend:test-upgrade` passed again after the test-step spacing changes. The candidate image was reused because production source did not change in this verification pass.
