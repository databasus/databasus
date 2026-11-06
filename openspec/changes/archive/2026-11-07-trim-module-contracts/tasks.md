## 1. Trim the smaller contracts

- [x] 1.1 Verify references from production code, tests and retained declarations before deleting the four user alias and constant candidates listed in `design.md`. Keep any candidate with a real reference and remove only comments or imports made redundant by the deletion. Verify test-package compilation with `go -C backend test -run '^$' ./internal/modules/users`.
- [x] 1.2 Apply the same reference check to the seven notifier provider constants and `CategoryAll`, preserving the types and constants needed by consumers and contract signatures. Verify test-package compilation with `go -C backend test -run '^$' ./internal/modules/notifiers`.
- [x] 1.3 Verify and remove the eight unused storage provider constants, preserving the module interface and its signature types. Verify test-package compilation with `go -C backend test -run '^$' ./internal/modules/storages`.
- [x] 1.4 Verify and remove the ten workspace alias and constant candidates, preserving the actions, workspace type and event declarations that consumers or signatures need. Verify test-package compilation with `go -C backend test -run '^$' ./internal/modules/workspaces`.

## 2. Trim the backups contract and unused methods

- [x] 2.1 Verify the 41 backup alias and constant candidates with resolved references, including test consumers and references inside the contract. Delete only declarations that no caller or retained declaration needs; preserve subscription handler aliases, return types and used status constants. Verify test-package compilation with `go -C backend test -run '^$' ./internal/modules/backups/...`.
- [x] 2.2 Confirm that the four implementation methods `RequestFullBackup`, `ClearFullBackupRequest`, `RequestIncrementalBackup` and `ClearIncrementalBackupRequest` still have no references, then delete them and the unused `time` import from `backups/internal/module/module.go`. Preserve the corresponding configuration service methods and all working module forwarding and orchestration code. Verify test-package compilation with `go -C backend test -run '^$' ./internal/modules/backups/...`.

## 3. Verify the complete cleanup

- [x] 3.1 Audit all remaining root module contracts for unused aliases and constants, including databases, verification, healthcheck and telemetry. Confirm the deletion set against production, test and retained-signature references, update glossary references to deleted aliases, and record any retained candidates in the implementation report. Verify all backend test packages compile with `go -C backend test -run '^$' ./...` and production packages build with `go -C backend build ./...`.
- [x] 3.2 Format only files changed by this implementation using the backend formatter configuration, then run `mise run backend:lint`, which includes `archlint`, and `mise run test` for existing behavioral coverage. Confirm that the diff preserves module interfaces, implementation packages, application wiring, domain service methods and HTTP contracts. Resolve blocking review findings before reporting the implementation complete.

## Verification results

- Resolved-reference audit: all 71 removed aliases/constants and four removed methods had no references. Each of the 94 retained aliases/constants has resolved references. All module interfaces match their pre-change declarations.
- Scoped compilation passed for users, notifiers, storages, workspaces and all backups packages. `go -C backend test -run '^$' ./...` also passed for all backend test packages, including the end-to-end packages; these checks compiled tests without executing their scenarios.
- `go -C backend build ./...` passed.
- `golangci-lint fmt` completed for the six changed Go files using the backend configuration. `mise run backend:lint` passed, including generated-code checks, contract lint, formatting, Go lint, `archlint` and module dependency checks.
- The final `mise run test` passed both backend groups and all 2,616 frontend tests in 189 files. The worktree has the standard local `.env` copied from `.env.example`; direct Fedora checks use the documented libedit shim.
- `docs/glossary.md` now refers to the owning internal types in the five affected entries. Repository search found no remaining documentation references to the removed qualified names.

## Change verification

```sh
mise run test
```
