## 1. Extract verification execution steps

- [x] 1.1 Establish the current runner behavior before editing production code by running `go -C backend test -count=1 -failfast ./internal/agent/runner/...` with the existing scenarios unchanged.
- [x] 1.2 Extract private domain functions for container preparation, archive restoration and statistics collection, keeping lifecycle ownership in `executeJob` and reusing existing download and reporting functions. Preserve the design's cleanup order, cancellation guards, disk verdicts, TimescaleDB sequence, diagnostics and exit-code distinctions. Verify with `go -C backend test -count=1 -failfast ./internal/agent/runner/...` and `mise run agent:e2e`, without changing existing scenarios or assertions to accommodate the refactor.

## 2. Check the completed refactor

- [x] 2.1 Format only this change's Go files using the formatter configured by `mise run backend:format`, then run `mise run backend:lint` and `go -C backend build ./cmd/verification-agent`. Check the final diff for preserved report messages and fields, unchanged public contracts and no unrelated edits; update existing documentation references only if a moved or renamed unit is actually documented.

After these tasks pass, finish the turn to run the mandatory implementation-review hooks and resolve any blocking findings before handing over.

## Verification Evidence

- `mise run doctor`: passed with all eight Codex project hooks trusted and enabled.
- `go -C backend test -count=1 -failfast ./internal/agent/runner/...`: passed before and after extraction.
- `mise run agent:e2e`: passed unchanged in 295.606 seconds after creating the worktree's missing `.env` from `.env.example`. The initial attempt stopped at fixture setup because `TEST_DATABASE_DSN` was absent.
- `golangci-lint fmt internal/agent/runner/runner.go internal/agent/runner/job_execution.go`, from `backend/`: passed using the backend formatter configuration.
- `mise run backend:lint`: passed.
- `go -C backend build ./cmd/verification-agent`: passed; removed the generated binary after checking it.
- `gopls check internal/agent/runner/job_execution.go internal/agent/runner/runner.go`, from `backend/`: passed without diagnostics.
- Execution log and failure-message literals match the original implementation. Existing test files, API contracts and public interfaces are unchanged; no existing documentation references required an update.
