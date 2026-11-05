# SFTP credential coverage

| Surface | Existing coverage | Change coverage |
| --- | --- | --- |
| Secret lifecycle | `secrets_test.go`: encrypted, hidden and retained credentials for every provider | `TestSftpAuthenticationMetadata`, `TestSftpCredentials`: password/key/dual create/read/list/update metadata, unchanged encrypted bytes after reload, replacement and both switch directions |
| Unsaved tests | `unsaved_settings_test.go`: retained password, workspace/type/role/session checks | `TestSftpInvalidCredentials`, `TestSftpCredentialAccess`, `TestSftpSelectedConnection`: malformed or contradictory credentials, missing methods/credentials, session/role/workspace refusals and no writes |
| Provider connections | `internal/providers/provider_test.go`; `internal/providers/sftp/provider_test.go` | `TestSftpSelectedConnection`, `TestSftpConnectionIdentity`: password-only and key-only containers, excluded-password failure, retained key/password/dual tests, saved tests and a fresh key-authenticated session after pooling password authentication on the same endpoint |
| Editor | `EditStorageComponent.test.tsx`: retained S3 secrets and stale connection tests | `EditStorageComponent.test.tsx`: all three sets, secret-free rename, selected replacement, method-change invalidation, missing replacement, unsaved reversal, cleared acknowledged input, retained newer input, usable testing after overlapping save/test responses, required credentials after a saved removal and ignored stale saves after workspace/storage/session changes or unmount |
| Full storage screen | `StoragesComponent.test.tsx` | A held settings save followed by a held list refresh keeps the same password/key controls, preserves newer values and sends them in the next save; switching methods or changing the path and then renaming uses the acknowledged response; selecting another storage closes the previous editor |
| Detail | `StorageComponent.test.tsx` | `StorageComponent.test.tsx`: password/key/dual read-back without secret inference |
| Browser | UI production route suite | `sftpStorage.spec.ts`: saved key/dual settings, secret-free rename, explicit switch, keyboard focus, scoped axe and 320px field containment |

## Regression baseline

`TestSftpAuthenticationMetadata` reads a saved key-only storage through the API and expects `authMethods: [PRIVATE_KEY]` with neither secret present. The test is added before changing the contract. Baseline: `mise run backend:test-package ./internal/modules/storages` failed; the isolated `go test -race -count=1 -run TestSftpAuthenticationMetadata ./internal/modules/storages` confirmed the intended failure: expected `[PRIVATE_KEY]`, received no `authMethods`.

## Contract and upgrade notes

SFTP requests require `authMethods`; missing, duplicate, unsupported or empty selections and explicitly empty secrets are invalid. Omit unchanged credentials for selected methods. Explicitly excluding a method removes its credential only on successful save. Unsaved tests use the same resolution without writing settings. Existing rows derive metadata from credentials without a migration or upgrade rewrite.

Host-key verification remains a separate unresolved limitation: the current transport ignores the host key despite the exposed flag. This change does not repair host authenticity.

## Localization

All six dictionaries include the dual-method label, saved-password/key guidance and required-credential errors. The editor screen test switches English to Russian with a path draft and a missing private-key error, preserving both the draft and validation state.

## HTTP validation

The storages adapter validates requests against its generated embedded contract, following the existing healthcheck adapter pattern. This preserves the difference between omitted and explicitly empty credentials before provider resolution. Validation failures return a constant safe description without rendering the underlying schema error or request values; rejected create, update and unsaved-test responses are checked for both secrets. The SFTP provider owns selection, fallback and exclusion; the existing row lock and update/audit transaction remain intact.

## Verification

All required checks passed:

- `mise run backend:test-package ./internal/modules/storages ./internal/modules/storages/internal/providers ./internal/modules/storages/internal/providers/sftp ./internal/lib/testcontainers` passed all four packages with race checks. A final focused `go test -race -count=1 -run TestSftp ./internal/modules/storages` also passed.
- `pnpm --dir frontend test` passed 124 files and 1109 tests.
- `pnpm --dir frontend test:browser --project=ui` passed its harness probes and all 40 production-browser scenarios, including both new SFTP cases and the repaired Google Drive accessibility check.
- `mise run backend:lint`, `mise run frontend:lint`, `pnpm --dir frontend build`, `openspec validate fix-sftp-storage-credentials --strict` and `git diff --check` passed.

Only owned changed files were formatted. No full backend suite or migration was run. Browser-owned processes and containers stopped at completion.

The full browser run exposed a pre-existing Google Drive accessibility race: axe could sample intermediate modal/button colors during opening animations. The affected test now waits for the dialog subtree's active animations to finish before checking unchanged contrast assertions; no retries or reduced assertions were added.

## Additional regression baselines

The rejected-request regression failed before the safe validation message: missing `authMethods` caused both supplied credentials to appear in create, update and test error responses. The full-screen regression failed before silent list refresh: accepting the held save unmounted both credential controls while the list request was pending. Both cases are fixed at their owning boundaries without changing the specified behavior.

The full-screen storage scenario has a 15-second limit because it drives two settings tests, two saves and a held list refresh through the real screen. The default five-second limit expired during the concurrent full run despite the scenario passing in isolation. The language-menu keyboard/hover tests wait for the observable closed state and restored focus rather than assuming the dropdown commits synchronously after Escape. Their assertions remain unchanged.

The notifier-composition scenarios also have a 15-second limit. The transfer case exceeded the default five seconds during the full run while driving workspace selection, notifier creation, a held save, parent refresh, cancellation and reselection. All behavioral assertions remain intact.

Full-screen rename regressions also pin the separation between acknowledged settings and the open editor draft. Before updating the detail view from the save response, a later rename sent the original method set/path. The detail now keeps the server response while the open settings editor retains its own entry snapshot and newer input. Selecting another storage closes the previous editing session.

The overlapping save/test regressions independently hold a settings save and a newer draft's connection test. Before invalidating that test on save acknowledgment, both scenarios failed because the test button stayed disabled. Acknowledgment now resets the test state while preserving newer input; late success and network failure responses are ignored, and a fresh test permits saving that input.
