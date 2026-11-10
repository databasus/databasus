## 1. Baseline and target selection

- [x] 1.1 Follow `docs/how-to-upgrade-deps.md` to select exact versions within the target lines. The agent reviews official migration notes, release age, peer/runtime requirements and native lock diffs, then records a concise selection summary, source links and any deferrals or exceptions in this design. Do not build selection scripts or candidate-graph acceptance fixtures. Verify the existing baseline with `mise run verify-deps`.

## 2. Toolchains and package management

- [x] 2.1 Migrate Go to the eligible 1.27 line, updating the module directive, mise/native locks, builder digests and required compatible linter/language tools together. Verify with `mise run backend:lint`, `go -C backend build ./cmd/verification-agent`, `go -C backend test -count=1 ./internal/lib/...` and `mise run e2e:browser` for the Dockerfile change.
- [x] 2.2 Retain Node 24.21.0 with its existing builder digest and both browser archive hashes, as accepted by the user after reviewing Node 26 release age. Keep Node type declarations on the 24 line. Verify with `mise run verify-deps`, `pnpm --dir frontend build`, `npm --prefix website run build` and `mise run e2e:browser`.
- [x] 2.3 Retain Python 3.14.7 and its existing lint/type targets and uv lock, as accepted by the user after the Python 3.15 Semgrep installation failed. Add no second interpreter or source-build workaround. Verify with `mise run python:lint`, `python3 -m unittest discover -s scripts -p '*_test.py'` and `python3 -m unittest discover -s .agent-hooks -p '*_test.py'`.
- [x] 2.4 Migrate pnpm to an eligible 12 release and regenerate its frontend/tool native locks, sidecars and coordinated browser/CI pins. Review required dependency build scripts and verify frozen installation with `pnpm --dir frontend install --frozen-lockfile --ignore-scripts`. Preserve native lock ownership rather than adding a lock parser. Verify with `mise run verify-deps`, `pnpm --dir frontend test` and `pnpm --dir frontend build`.

## 3. Frontend compiler and build/test stack

- [x] 3.1 Move website TypeScript to 6.0.3 and retain one frontend TypeScript 5.9.3 compiler, as accepted by the user after reviewing the frontend tooling graph. Preserve every compiler-based lint rule without a second compiler or compatibility shim. Verify with `mise run generate-api`, `mise run frontend:lint`, `pnpm --dir frontend typecheck`, `pnpm --dir frontend test`, `pnpm --dir frontend build` and `npm --prefix website run build`.
- [x] 3.2 Migrate Vite, its React plugin and Vitest to the compatible 8/6/5 group. Preserve aliases, environments, discovery, isolation, shared time limits and production browser serving; adapt existing failure fixtures only for changed interfaces. Verify with `mise run frontend:lint`, `pnpm --dir frontend test`, `pnpm --dir frontend build` and `pnpm --dir frontend test:browser --project=ui`.
- [x] 3.3 Retain MSW 2.15.0 and restore the entire unsuccessful MSW 3 trial, as accepted by the user. Keep the existing unexpected-request, pending/aborted-request, teardown and listener-isolation assertions without a new transport or weakened checks. Verify with `pnpm --dir frontend test`.

## 4. Application routing and localization

- [x] 4.1 Migrate Router to 8 with compatible React/DOM patches and adapt the declarative router. Preserve deep links, reloads, history, authorization, missing-resource and session-change behavior through existing tests. Verify with `pnpm --dir frontend test`, `pnpm --dir frontend test:browser --project=ui` and `mise run e2e:browser`.
- [x] 4.2 Retain i18next/react-i18next 26/17, as accepted by the user after reviewing the 27/18 release age. Preserve typed keys, live language changes, locale formatting and escaping. Verify with `mise run frontend:lint`, `pnpm --dir frontend test` and `pnpm --dir frontend test:browser --project=ui`.

## 5. Backend driver, authentication and log export

- [x] 5.1 Migrate production, test-support, backup E2E and upgrade-suite MongoDB imports to an eligible compatible v2 release, evaluating v2.8.2 first and retaining the 4.2 minimum. Reuse the real-server matrices, adding a 4.2 connection/provisioning and backup/restore case only if missing. Verify with `mise run security:go`, `go -C backend test -count=1 ./internal/engines/mongodb`, `go -C backend test -count=1 ./internal/modules/backups/e2e/mongodb` and `mise run backend:test-upgrade`, using the documented Fedora setup. A candidate failing security or server compatibility does not complete this task.
- [x] 5.2 Migrate JWT to v5 and remove v4 imports. Retain existing malformed-token, password-version and password-change revocation tests. Close the confirmed gaps with focused user-operation cases for signature tampering, disallowed algorithms, invalid subjects, injected-clock expiry just before/at/after the boundary, and acceptance of an independently signed token with the explicit pre-migration claim schema. Reuse existing test setup without retaining a v4 runtime dependency or adding a validation harness. Verify with `go -C backend test -count=1 ./internal/modules/users` and `mise run security:go`.
- [x] 5.3 Migrate the log API/SDK, bridge and exporters to a coordinated stable line and replace the development gRPC build with a stable graph. Retain existing protocol, endpoint-path, redaction, shutdown and failure checks. Verify with `go -C backend test -count=1 ./internal/lib/logging`, `mise run backend:lint`, `go -C backend build ./cmd/verification-agent` and `mise run security:go`.

## 6. Website and release tools

- [x] 6.1 Migrate website ESLint to 10 with a compatible eligible Next.js 16 lint configuration and correct its package-manager declaration to the selected toolchain's exact npm version. Keep npm's frozen lock and static export. Verify with `npm --prefix website ci --ignore-scripts --no-audit --no-fund`, `npm --prefix website run lint` and `npm --prefix website run build`. The agent inspects representative language routes, canonical/hreflang metadata and the shared 404 in the generated output; do not add a website-export test script for this tooling update.
- [x] 6.2 Migrate Helm to an eligible 4 release. Verify with `helm lint deploy/helm`, `helm template databasus deploy/helm` and `helm package deploy/helm --destination "$package_directory"`, using a fresh temporary directory removed afterwards. The agent reviews rendered resources, package metadata and documented OCI arguments. Do not add a Helm wrapper or command-capture harness, and do not publish.
- [x] 6.3 Upgrade eligible external action successor majors across workflows, keeping full commit pins and exact release comments. Review runner requirements, permissions, release gates, architecture ownership, cache and artifact interfaces. Adapt affected existing workflow tests rather than adding another harness. Verify with `mise run verify-deps`, `python3 -m unittest discover -s scripts -p '*_test.py'` and `mise run lint-workflows`.
- [x] 6.4 Replace the archived release-creation action with the pinned GitHub CLI, passing generated notes through a file, binding a new tag to the checked commit with `--target "$GITHUB_SHA"` and retaining failure propagation, gates and permissions. The agent reviews the documented CLI arguments and existing workflow assertions; add a focused case only for uncovered behavior in repository-owned logic. Verify with `python3 -m unittest discover -s scripts -p '*_test.py'` and `mise run lint-workflows`, without creating a release or building a generic release-command fixture.

## 7. Documentation and integrated verification

- [x] 7.1 Update affected dependency-guide, toolchain, build and testing references, recording completed migrations and the TypeScript 7 deferral. Remove superseded imports and tool declarations without shims. Verify with `mise run verify-deps`, `mise run lint`, `pnpm --dir frontend build`, `npm --prefix website run lint`, `npm --prefix website run build` and a repository search for replaced direct integrations and pins.
- [x] 7.2 Verify the integrated result with `mise install --locked --include-task-tools`, `go -C backend mod tidy -diff`, `pnpm --dir frontend install --frozen-lockfile --ignore-scripts`, `npm --prefix website ci --ignore-scripts --no-audit --no-fund`, `uv sync --locked --project e2e/browser`, `mise run verify-deps`, `mise run agent:e2e` and the change-verification commands below. Reuse passing targeted results while their inputs remain unchanged; repeat task 5.1's MongoDB/upgrade checks and the script/hook checks if later changes affect them. Run lifecycle/interruption suites only if startup or cleanup code changes, as `docs/testing.md#checking-work` requires. Format this change's files and resolve verification failures. Mandatory review remains a separate pre-archive and pre-commit gate.

## Apply status — 2026-11-10

The user accepted completion with Node 24.21.0, Python 3.14.7, frontend TypeScript 5.9.3, MSW 2.15.0 and i18next/react-i18next 26/17 retained. Tasks 2.2, 2.3, 3.1, 3.3 and 4.2 record those final decisions rather than claim the excluded migrations happened. All implementation and verification tasks are complete. Mandatory implementation review must pass before archive and commit.

Verification of the implemented slices:

- Native locked mise installation, Go module tidy diff, frozen pnpm/npm installation, locked browser Python synchronization and `mise run verify-deps` passed. A fresh pnpm installation also passed with its normal lifecycle-script policy and the explicit denial of the unused MSW worker-copy script.
- `mise run lint`, frontend type checking and production build, website lint and static build passed. The exported website retains all 161 localized routes, their canonical/hreflang metadata and the shared 404. Native Helm lint, rendering and temporary packaging passed without publishing.
- `mise run test` passed all three processes (backend slow packages, remaining backend packages and frontend). Frontend tests passed all 2,863 cases. Browser UI passed all 315 cases, including its existing harness probes and complete/disjoint-shard checks. Browser E2E passed six cases; five intentionally disabled lifecycle/probe cases remained skipped in that command.
- MongoDB engine and backup/restore matrices passed, including MongoDB 4.2 and TLS. User-operation and logging tests passed. The Go security scan reported no reachable vulnerabilities. Agent E2E and the upgrade from v3.67.2 passed, including the older-agent scenarios. Affected PostgreSQL logical and physical backup E2E packages passed.
- All 193 script tests and all 282 review-hook tests passed. Hosted action execution, publication and native ARM runtime behavior have not been established locally.

An integrated run exposed a server-start test timeout. Twenty isolated repetitions did not reproduce it. The tests released a selected port before the application bound it, so another socket could take it; they also hid early startup errors behind HTTP polling. Startup fixtures now let the application bind `127.0.0.1:0`, obtain its actual address from the startup log and report an early exit immediately. Affected server and telemetry-shutdown scenarios passed after the repair, and the repeated integrated run passed. `go -C backend test -count=1 ./internal/lib/...` also passed. The final Go image catalog refresh passed its package tests, the affected native-release/verifier script tests and the dependency verifier. No production startup behavior or timeout was changed.

The pre-migration session fixture uses the fixed `sub`, `exp`, `iat`, `role` and `passwordVersion` keys, taking only the account ID and stored password version from its fixture. Renaming the production subject key in both issuance and validation made the previous fixture pass; the strengthened compatibility case failed with 401 under the same mutation. Production keys were restored and the user package passed. The release-target workflow assertion failed before the CLI fix and passed afterwards.

Remaining non-failing diagnostics are the website plugins' ESLint 10 peer-range warnings, the linter's deprecated configuration warnings and a happy-dom abort diagnostic during Vitest environment teardown. None was suppressed or used to weaken existing assertions.

## Change verification

```sh
mise run test
pnpm --dir frontend test:browser --project=ui
mise run e2e:browser
```
