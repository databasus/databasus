## Context

See `proposal.md` for scope. The dependency-maintenance guide, native locks and offline check from `add-dependency-upgrade-verification` are already available. Use `docs/how-to-upgrade-deps.md` for version selection, release-age review, frozen installation and reporting.

Before this change, the frontend pins Vite 6, Vitest 3, TypeScript 5.8 and React plugin 4 (`frontend/package.json:45`, `:56`, `:57`, `:58`), Router 7 and i18next/react-i18next 26/17 (`:31`, `:25`, `:30`), and MSW 2 (`:49`). Repository linters directly call the classic TypeScript compiler API (`frontend/lint/styleChecks.ts:286`, `frontend/lint/frontendModuleReferences.ts:21`). The HTTP test harness depends on MSW handler and lifecycle behavior (`frontend/src/shared/lib/test/mockApiLifecycle.ts:1`).

The backend imports MongoDB driver v1 and JWT v4 (`backend/go.mod:32`, `:14`). MongoDB 4.2 remains accepted (`backend/internal/engines/mongodb/version.go:78`). Session expiry deliberately uses the injected clock (`backend/internal/modules/users/internal/signin/session.go:72`). Log export uses the pre-stable SDK and two exporters (`backend/internal/lib/logging/otlp_sink.go:12`), and gRPC is a development build (`backend/go.mod:123`).

Toolchains and packaging tools are pinned at `mise.toml:143` through `:167`. Browser tooling repeats Node/pnpm pins and archive hashes (`frontend/browser/environment.json:1`). The website uses npm and a committed npm lock even though its package-manager annotation names pnpm (`website/package.json:26`, `.github/workflows/deploy-website.yml:53`). CI uses an archived release-creation action (`.github/workflows/ci-release.yml:1522`).

Follow the backend, frontend and website `AGENTS.md` documents and the boundaries in `docs/architecture.md`. This is dependency maintenance within those boundaries, not a reason to move domain logic between modules, adopt server-rendered routing or remove compiler-based architecture checks. Main specifications remain unchanged, so this change declares `skip_specs: true`.

## Goals / Non-Goals

**Goals:** Complete the accepted compatible migrations, retain the five lines explicitly approved by the user, remove superseded direct integrations and demonstrate preserved behavior with the existing real application and engine tests.

**Non-Goals:** Adopt every newest major regardless of compatibility, drop supported servers, redesign session claims, redesign the UI or rewrite the frontend compiler tooling for the native compiler API.

## Decisions

### 1. Select exact eligible releases at apply time, inside explicit target lines

The table is the final bounded migration set. The user accepted completion with Node 24, Python 3.14, frontend TypeScript 5.9, MSW 2 and i18next/react-i18next 26/17 retained; those newer major migrations are excluded rather than claimed as implemented. Entries name target lines rather than unconditionally selecting the newest registry tag. At apply time the agent checks official release and migration information, peer/runtime requirements and the native lock diff. Record a concise selection summary with exact versions, source links, compatibility constraints and any deferrals or release-age exceptions. Follow the guide when choosing an older eligible patch or reporting that a target is unavailable or too fresh; do not sleep waiting for eligibility or mark an unfinished migration complete. The offline check validates its known local declarations, not release age or upstream safety. No candidate-graph acceptance fixture, machine-readable evidence store or new registry checker is required.

| Migration | Target | Coordination |
| --- | --- | --- |
| Vite 6 / React plugin 4 / Vitest 3 | Vite 8 / compatible plugin 6 / Vitest 5 | Preserve shared aliases, environment selection, discovery, isolation, time limits and production browser serving |
| MSW 2 | Retain 2.15.0 | Preserve explicit finite HTTP handlers, unexpected-request failures and teardown; MSW 3 is excluded after its trial failed the full test suite |
| Router 7 | Router 8 | Keep declarative SPA routing; adopt compatible React/DOM patches and verify URL/history/session behavior |
| i18next 26 / react-i18next 17 | Retain 26/17 | New majors did not meet release-age policy; retain live language switching, typed keys and escaped interpolation |
| TypeScript 5 | Website 6.0; frontend 5.9.3 | One classic compiler per project; retain the frontend tooling graph rather than install a second compiler |
| MongoDB Go driver v1 | Compatible v2 release, initially v2.8.2 | Preserve MongoDB 4.2 through the currently supported maximum; validate real connections, provisioning, backup and restore |
| JWT v4 | JWT v5 | Retain accepted HS256 algorithm, master-key signing, claims, password-version revocation and injected-clock expiry |
| Pre-stable log API/SDK | Stable coordinated log API/SDK line | Align bridge/exporters and other telemetry modules; select a stable gRPC graph |
| pnpm 9 | pnpm 12 | Frontend lock, mise/native sidecars, browser runtime, install-script allowlist and CI |
| Helm 3 | Helm 4 | Preserve chart rendering, values and OCI package/push interface without publishing |
| Website ESLint 9 | ESLint 10 | Use a compatible eligible Next.js 16 lint configuration; keep the website on npm and correct its package-manager declaration |
| Go 1.26 | Go 1.27 | Module directive, mise, builder images, native locks, linter and language tooling compatibility |
| Node 24 | Retain 24.21.0 | Keep builder digest and browser native archives/hashes; runtime types follow the retained Node line |
| Python 3.14 | Retain 3.14.7 | Keep runtime, type/lint targets and uv lock compatible with the pinned Semgrep installation |
| Older external workflow actions | Eligible maintained successor major per action | Resolve full commit and release identity; preserve permissions, gates and artifact formats |
| Archived release-creation action | Pinned GitHub CLI | Use the existing tool; preserve tag, title, generated notes and release status |

No migration changes a user's MongoDB server, embedded database major or accepted database identity. Same-major dependencies move only when required by these migrations, their peers or a defect exposed by verification. Native pre-stable dependencies may still be required transitively; do not replace them merely because their version starts with zero.

Reject repository-wide `--latest` and treating the target table as proof that a release is available or eligible. Exact selections remain the implementing agent's responsibility under the existing guide.

### 2. Preserve the supported server range when selecting MongoDB driver v2

The vendor's [upgrade guide](https://www.mongodb.com/docs/drivers/go/current/reference/upgrade/) says driver 2.9 drops MongoDB 4.2. The module proxy currently lists v2.8.2 as the latest stable 2.8 patch. Use the newest eligible v2 release that still supports the complete accepted server range, starting the evaluation at v2.8.2. Check current reachable vulnerabilities and the vendor compatibility statement before locking it. If no v2 candidate satisfies both security acceptance and the supported range, keep this migration incomplete and report the concrete conflict rather than drop support or retain v1 as a parallel adapter.

Update all production, test-support, logical-backup E2E and upgrade-suite imports together. Reuse the existing real-server matrices, adding the minimum accepted 4.2 case only if it is missing; version parsing alone does not prove driver connectivity or a restored backup.

Reject the latest v2 tag as the only target because it silently changes supported product behavior. Reject removing the 4.2 identity or adding an old-driver fallback: neither is authorized by this maintenance change.

### 3. Keep one classic TypeScript compiler per project and preserve lint behavior

Move the website to TypeScript 6.0.3 and retain one frontend TypeScript 5.9.3 compiler. Frontend architecture and generation tooling still resolves TypeScript 5 dependencies under 6; the user accepted keeping that migration outside this change. Preserve type-check strictness, source aliases and every architecture/style rule. The repository's direct `createProgram` and `createSourceFile` uses need the classic API. The vendor describes the [native API transition](https://github.com/microsoft/TypeScript/wiki/Using-the-Compiler-API); a TypeScript 7 migration would require a separate compiler-tooling design.

Reject deleting checks, broad suppressions or a hidden TypeScript 5/6 compiler installed alongside TypeScript 7. A single classic compiler satisfies the migrations without that extra maintenance surface. TypeScript 7 is explicitly deferred for its API redesign, not claimed as upgraded.

### 4. Change foundational tooling before its consumers

Start from a passing baseline. Review the candidate releases and installation scripts, update native locks with their owning tools and verify frozen installation. Review direct-download checksums and both supported architecture records through the existing procedure. Migrate Go and package management with their dependent lock/configuration surfaces first, followed by website TypeScript and frontend build/test tooling, then the application router. Retain the accepted Node, Python, frontend compiler, HTTP test harness and localization lines. Update pnpm build-script controls intentionally: preserve the current restriction on unreviewed dependency lifecycle scripts and authorize only scripts actually needed by accepted dependencies.

Keep the website's npm lock and `npm ci`. Correct its misleading package-manager annotation to the exact npm version provided by the selected Node toolchain instead of converting the website to pnpm. Update ESLint and its compatible Next.js 16 configuration together, preserving static export, language routes and generated metadata.

Reject a monolithic dependency bump with one final test run: it makes a failed lifecycle, navigation or compile check difficult to attribute. Reject keeping an old tool installation as a compatibility fallback; each completed slice leaves one coherent accepted set of pins.

### 5. Preserve authentication, logging and navigation contracts during library adaptation

For JWT v5, keep the wire claims and injected time semantics. Existing user-operation tests cover malformed token text and invalid password versions (`backend/internal/modules/users/access_test.go:33`, `:93`), plus revocation after a password change (`backend/internal/modules/users/profile_test.go:135`). They do not cover signature tampering, disallowed signing algorithms, expiry boundaries or invalid subjects. These are concrete migration gaps: add focused cases through the existing user HTTP operations, including acceptance just before expiry and rejection at and after expiry using the injected clock. Also add acceptance of an independently signed token using the explicit pre-migration claim schema without retaining a v4 runtime dependency. Reuse the existing test setup rather than introducing a token-validation harness. Existing sessions remain valid because the token contract is unchanged; this compatibility requires no old-library adapter.

For telemetry, update the stable log API/SDK, bridge, exporters and necessary protocol/gRPC modules as one compatible graph. Preserve endpoint paths, header redaction, protocol selection, bounded shutdown and error propagation. Do not infer that a stable SDK implies the exporters use the same version numbering.

For Router 8, follow the [v7 upgrade guide](https://reactrouter.com/upgrading/v7) only for APIs used by the existing declarative router. Do not adopt framework mode or SSR. Keep MSW 2 with its existing imports and lifecycle integration. Preserve failures on deliberately unexpected HTTP requests and cleanup of pending requests. For the retained localization pair, reuse the existing live-switching and interpolation tests. Add a case only when changed integration behavior is not already covered.

Reject changing application rules to match a library's new defaults. Adapt the integration to the existing main specs and retain security assertions.

### 6. Migrate release tools without publishing to test them

Inventory every external action, inspect the official successor release and runtime requirements, and update eligible successor majors with full commit SHAs and exact release comments. Pin related action instances consistently. Replace `actions/create-release` with the already pinned `gh` command, passing generated notes through a file, binding tag creation to the verified workflow commit with `--target "$GITHUB_SHA"` and keeping least-privilege job permissions. Do not create a real release to test it.

Check Helm 4 with its native local lint, render and package commands in a temporary directory. The agent reviews rendered resources, package metadata and documented OCI arguments; no Helm wrapper or command-capture fixture is needed. For the website, use the existing lint and static build and inspect representative output for preserved routes and metadata without adding an export-check script. For workflows, run the existing checks, adapting assertions affected by the migration, and review release gates, notes-file handling, permissions, cache/artifact interfaces and architecture ownership. Native hosted-runner execution remains a release check; local checks do not claim to prove successful publication.

Reject another third-party release action when the existing pinned CLI covers the operation, and reject new test harnesses that merely restate its arguments. Remote publication or CI runs are not local task acceptance criteria.

### 7. Reuse existing checks and add tests only for a concrete gap

This is a dependency migration, not an expansion of verification infrastructure. The agent selects versions, reviews migration notes and diffs, runs native frozen installations and uses the existing application, engine, workflow and script suites. Keep current assertions and adapt affected tests to changed APIs. New tests belong only to an uncovered compatibility requirement or a defect found during migration, such as missing MongoDB 4.2 coverage or previously issued session tokens. Do not add generic selection checks, duplicate compiler/runtime fixtures or standalone website/Helm/release test harnesses for pin changes.

Each slice runs the checks relevant to its changes. Reuse a passing result while its inputs remain unchanged; the final verification runs the integrated suite after all accepted migrations. Run lifecycle/interruption suites only if their startup or cleanup code changes, following `docs/testing.md#checking-work`.

Reject mandatory new tests for every bumped package: they add maintenance without necessarily observing new behavior. Reject skipping existing security and compatibility checks: agent judgment complements those checks and does not replace them.

## Apply selections — 2026-11-10

The baseline `mise run verify-deps` passed. The following selections use official release metadata; a selected version is not a claim that its implementation or verification is complete. Task checkboxes track completion.

| Group | Exact selection / status | Source and constraint |
| --- | --- | --- |
| Go | 1.27.3; golangci-lint 2.15.0 | [Go release history](https://go.dev/doc/devel/release) dates 1.27.3 to 2026-10-19; [linter release](https://github.com/golangci/golangci-lint/releases/tag/v2.15.0) is from 2026-10-16 and reads the new compiler export format. Native scanning rejected 1.27.2 because of reachable [GO-2026-6809](https://pkg.go.dev/vuln/GO-2026-6809). The selected builder index `sha256:e519d3b0a5b285e5aa53ab9eef02bc6bbb9cb04608389b743a136a9ce75f0d1e` was pushed on 2026-11-05: accept this explicit image-age exception to retain the certificate-security fix instead of introducing a vulnerable compiler. Both native compiler archives have official checksums. |
| Node | Retain 24.21.0; Node 26 excluded | [26.13.0 LTS](https://github.com/nodejs/node/releases/tag/v26.13.0) was released on 2026-10-28. Fourteen days complete on 2026-11-11; the current Alpine image digest was last pushed on 2026-11-05 and needs separate age review. Do not adopt an unreviewed digest or call a pre-LTS binary an LTS release. |
| Python | Retain 3.14.7; Python 3.15 excluded | [3.15.2](https://www.python.org/downloads/release/python-3152/) is old enough, but actual locked installation fails because Semgrep 1.182.1 requires `ruamel.yaml.clib==0.2.15`, which has no CPython 3.15 wheel. The latest Semgrep metadata retains that pin. The unsuccessful Python slice was restored; no second interpreter or build-policy workaround was added. |
| pnpm / Helm | 12.17.0 / 4.3.1 | [pnpm release](https://github.com/pnpm/pnpm/releases/tag/v12.17.0), published 2026-10-30; [Helm release](https://github.com/helm/helm/releases/tag/v4.3.1), published 2026-10-15. pnpm uses its official GitHub binary through mise; Docker's frozen npm bootstrap explicitly runs only the reviewed pnpm installer to link the locked native binary. Move frontend overrides and script decisions to `pnpm-workspace.yaml`. The unused MSW 2.15.0 worker-copy postinstall is explicitly denied; no unreviewed lifecycle scripts are enabled. |
| TypeScript | Website 6.0.3; retain one frontend 5.9.3 compiler | [TypeScript registry](https://registry.npmjs.org/typescript) dates 6.0.3 to 2026-04-16. `openapi-typescript@7.13.0` and `tsconfck@3.1.6` declare TypeScript 5 peers; the architecture tooling also installs a TypeScript 5 dependency under 6. Frontend 5.9.3 keeps the resolved classic-tooling graph on one compiler. The user accepted this split as the final scope. |
| Vite / React plugin / Vitest / MSW | 8.3.5 / 6.2.1 / 5.1.0; retain MSW 2.15.0; MSW 3 excluded | [Vite migration](https://vite.dev/guide/migration), [Vitest migration](https://vitest.dev/guide/migration/), [MSW migration](https://mswjs.io/docs/migrations/2.x-to-3.x/); npm publication dates are 2026-10-21 / 2026-11-02 / 2026-11-02 / 2026-10-10. Keep newer patches inside the waiting period deferred. |
| Router / React | 8.5.1 / React and React DOM 19.2.7 | [Router registry](https://registry.npmjs.org/react-router) dates 8.5.1 to 2026-10-27 and requires React/DOM >=19.2.7; select the compatible patch from 2026-06-01 instead of an unrelated minor update. |
| Localization | Retain 26/17; 27/18 excluded | [i18next registry](https://registry.npmjs.org/i18next) and [react-i18next registry](https://registry.npmjs.org/react-i18next) date the first 27/18 stable pair to 2026-11-07. Earliest seven-day eligibility is 2026-11-14 after the publication times; reselect patches then. |
| MongoDB / JWT | Driver 2.8.2 / JWT 5.3.1 | [MongoDB upgrade guide](https://www.mongodb.com/docs/drivers/go/current/reference/upgrade/) preserves 4.2 below 2.9; module-proxy dates are 2026-08-27 / 2026-01-28. [JWT migration](https://github.com/golang-jwt/jwt/blob/v5.3.1/MIGRATION_GUIDE.md) retains our explicit HS256 and injected-clock validation. |
| Log export / gRPC | Log API/SDK and core telemetry 1.47.0; bridge 0.21.0; exporters 0.23.0; gRPC 1.85.0 | [Telemetry release](https://github.com/open-telemetry/opentelemetry-go/releases/tag/v1.47.0) and corresponding module metadata date the coordinated telemetry graph to 2026-10-02; [gRPC release](https://github.com/grpc/grpc-go/releases/tag/v1.85.0) is from 2026-10-27. Exporter/bridge version numbering remains pre-stable. |
| Website ESLint | 10.14.0 with eslint-config-next 16.4.4 | npm publication dates are 2026-10-30 / 2026-11-02. Next's supplied `fixupPluginRules` adapts the React/import plugins to ESLint 10; their peer metadata still excludes 10. Native `npm ci` warns but succeeds without force/legacy-peer flags, and website lint passes. Retain those warnings and verify the static build. Node declarations follow the retained Node 24 line: 24.19.2; npm annotation is 11.19.0. |
| Workflow actions | checkout 7.0.1; build-push 7.4.0; login 4.7.0; dependency-review 5.0.0; configure-pages 6.0.0; deploy-pages 5.0.1; upload-pages-artifact 5.0.0 | Official GitHub release/tag metadata supplies the dates and full commit identities. Already-current upload/download-artifact 7/8, mise 5, CodeQL 4 and Namespace 1 stay at their existing pins; no unrelated patch refresh. New JavaScript actions require runner >=2.327.1; checkout credential access from Docker actions requires >=2.329.0. Existing jobs retain permissions, named architecture artifacts and release gates. GitHub CLI remains 2.103.3 and receives release notes through a file. Hosted execution remains unverified locally. |

New frontend transitive releases were reviewed separately from unchanged baseline pins. Native pnpm resolution selected `ansi-regex@6.5.0` (2026-10-29), deferring its newer patch inside the seven-day period; the attempted MSW 3 cookie dependency was removed with that slice. Website npm resolution used its native `--before` cutoff of 2026-11-02T22:00:00Z. No permanent registry checker or additional release-age policy was introduced.

The user accepted closing the change with the five retained lines above. Their newer major migrations are outside this change and do not block its completion. Retaining them costs those newer features and fixes; it avoids an interpreter installation workaround, a second frontend compiler, a larger HTTP test-harness rewrite and adoption before the release waiting periods finish. Any later migration follows a separate request and the normal dependency guide.

The MSW 3 trial observed cross-origin interception failures and request/teardown hangs in happy-dom. A limited fetch-tracking/Undici adaptation passed targeted probes, but the full suite stopped making progress and timed out audit-screen cases. The exact root cause was not established, so this is not a confirmed upstream MSW defect. The entire unaccepted slice, including its imports, transport and added probe, was restored. Vitest 5's test platform supplies the missing browser `alert` method so existing negative assertions can use its stricter `spyOn` API. Browser shard fixtures keep the real native dependency graph and own their copied installation, because pnpm 12 validates dependencies before running a script and can relink them.

A read-only consultation through the active Codex review client advised distinguishing declared peer ranges from tested integration behavior, checking Next's supplied compatibility handling and avoiding a hidden older TypeScript compiler. These decisions retain native frozen installation, integrity checks and all existing lint/security rules.

## Risks / Trade-offs

- A target line is unavailable or too fresh at apply time -> Follow the guide to select a suitable patch or report the deferral or justified exception. The offline checker does not decide release eligibility.
- The newest MongoDB driver has a higher server minimum -> Use an eligible compatible v2 patch and prove the minimum server through real-engine tests; report an unsatisfied security/compatibility combination as a blocker.
- New test-tool defaults can make a broken harness appear green -> Run and retain the existing isolated expected-failure and cleanup probes, adapting them only where APIs change or a concrete gap appears, without retries or weakened assertions.
- A new toolchain can require newer linters or missing architecture wheels -> Resolve its required compatible tool graph first and verify both platform records; do not silently skip ARM or lower checks.
- Browser output can change under new build tooling -> Run the production browser suite with existing references; investigate any difference before using the explicit reference-update path.
- Action runtime behavior depends on hosted runners -> Run existing local workflow checks, review documented runner requirements and report the limits of local verification; do not add a simulated hosted runner.

## Migration Plan

1. Confirm the completed dependency-maintenance baseline, then select exact targets and record a concise migration summary under the existing guide.
2. Migrate the accepted Go/package-management, website TypeScript, frontend build/test, router, backend and release-tool slices in that dependency order. Run each slice's targeted checks before moving on.
3. Run the full routine suite, production browser UI, browser E2E, affected backup matrices, agent E2E and upgrade tests, plus the dependency verifier, linters and builds. No task invokes the aggregate full-test command or waits for a push.
4. Update the guide's inventory and affected architectural/testing descriptions. Finish with no superseded direct imports, compatibility aliases, accidental parallel compiler installations or unrecorded artifact changes.

No product schema migration is expected. If a migration fails, repair the integration or restore that entire unaccepted dependency slice before proceeding; never mix its new lock with old configuration or silently mark it complete.
