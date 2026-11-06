## Context

See [proposal.md](proposal.md) for the problem and agreed scope. This design is required because the change spans local tooling, commit hooks, two publishing workflows and three Python code areas.

- `mise.toml:17` runs backend, frontend, shell and workflow lint, while `mise.toml:21` formats only backend and frontend. Python and uv are already pinned at `mise.toml:111`.
- `hk.pkl:6` selects backend and frontend checks. Its pre-commit configuration disables fixes and stashing at `hk.pkl:19`.
- `e2e/browser/pyproject.toml:1` defines the existing non-package Python project, requires Python 3.14 and pins pytest and Playwright. `mise.toml:42` starts its commands through locked uv execution with the system Python selected.
- Python includes sibling imports and dynamic data. For example, `.agent-hooks/review_client.py:24` has an unannotated function and `.agent-hooks/review_client.py:40` has an unparameterized dictionary. `e2e/browser/scenarios/application_screens.py:84` returns decoded JSON through a bare dictionary annotation after a browser evaluation.
- `.github/workflows/ci-release.yml:684` lists release prerequisites before version selection. `scripts/release_gate_test.py:24` verifies that list transitively. `.github/workflows/deploy-dev-image.yml:18` publishes each architecture without a quality prerequisite.
- `scripts/test.sh:150` already runs script tests, review hook tests and browser suite checks in full mode. `docs/testing.md`, especially Checking work, determines the targeted verification owed by changes to browser scenarios and lifecycle code.

Root `AGENTS.md` governs the planned files. None is beneath backend, frontend or website module rules. If a discovered defect needs a module edit, read that module's rules and extend verification before editing it. The architecture remains unchanged: under `docs/architecture.md`, End-to-end suites, browser tests use the application image from outside and import no backend or frontend implementation. Python quality must preserve that boundary. The Automatic checks section leaves tool configuration beside the code; it does not require a new architecture layer or an inventory of tools in architecture documentation.

## Goals / Non-Goals

**Goals:** Make the same complete Python checks enforceable locally, before a commit and before publication; fully type and format the existing code; retain working script and hook contracts.

**Non-Goals:** Do not move runtime dependencies into a root Python application, introduce a new test framework, type every local literal redundantly, or replace runtime input validation with static annotations. The proposal lists the remaining scope exclusions.

## Decisions

### 1. Use pinned Ruff and ty with one root configuration

Pin exact tool versions in `mise.toml`. Put `[tool.ruff]` and `[tool.ty]` in a root `pyproject.toml` that contains tool configuration only, without declaring an installable root project. Match the pinned Python 3.14 runtime. Use Ruff formatting with a line length of 100 and double quotes.

Start with Ruff rule groups `E4`, `E7`, `E9`, `F`, `I`, `UP`, `B` and `ANN`, plus `RUF100`, `PGH003` and `PGH004` to reject stale or blanket lint suppressions. Keep preview and unsafe fixes disabled. Keep annotation exemptions disabled, including exemptions for functions returning only `None` and untyped variadic arguments. Conventional `self` and `cls` receivers need no explicit annotation. `ANN401` prevents explicit `Any` parameters; it is only one part of preventing dynamic types from spreading.

Run ty with warnings treated as failures. Retain its default correctness diagnostics, reject unused suppressions where supported by the pinned release, and fix existing findings. Do not establish a baseline or turn off rules across directories to complete migration.

Enable ty's `blanket-ignore-comment` rule: Ruff's blanket-ignore checks cover `type: ignore` and `noqa`, while ty's own rule also rejects unrestricted `ty: ignore` directives.

Alternatives: Black plus isort and Flake8 would require several tools for Ruff's responsibilities. Adding mypy or Pyright alongside ty would duplicate type checking. A lint-only rollout would leave the user's requested full typing incomplete. Enabling every Ruff rule would add unrelated style constraints and formatter conflicts.

### 2. Share file discovery and keep the current dependency environment

Add a small, typed standard-library runner in `scripts/python_quality.py` with `lint` and `format` modes. It resolves the Git worktree root and obtains tracked and eligible untracked Python paths using NUL-delimited Git output, filtering deleted paths. Both modes use that single selection, including hidden directories and new top-level directories. Pass arguments as arrays, preserving spaces in paths. Pass the root Ruff configuration explicitly so nested project metadata cannot silently change rules. Pass the root project explicitly to ty. Exclude dependency environments and generated files through the existing Git ignore mechanism; add tool cache patterns to `.gitignore`.

The runner's lint mode attempts `ruff check`, `ruff format --check` and ty checking even if an earlier stage reports violations, collects their statuses and fails when any stage fails. If dependency preparation fails, report that type checking could not run and fail; still report independent Ruff results. No lint invocation applies fixes. Format mode applies only Ruff's safe fixes and formatting, performs no type-directed rewrites, and returns failure for remaining lint errors rather than concealing them.

Resolve pytest and Playwright through the existing `e2e/browser` uv environment and committed lockfile. Invoke ty inside locked `uv run --project e2e/browser` execution with the root passed as its type-checking project. Keep `UV_PYTHON_PREFERENCE=only-system` and `PYTHONDONTWRITEBYTECODE=1`. Model existing sibling-import search roots in type-checker configuration; do not rearrange working runtime imports just to satisfy discovery. The runtime commands for hooks and scripts remain dependency-free, and no browser binaries or Docker resources are installed by lint.

Alternatives: separate hardcoded directory lists drift when a new Python area appears. Relying on separate default discovery in each tool can give different coverage for hidden directories and nested projects. A root dependency workspace would change how working hooks and e2e commands start. Ambient virtual environments make local and CI type resolution differ. The small runner follows the Git-based discovery already used by `mise-tasks/lint-shell:8` and also provides consistent status aggregation.

### 3. Fully annotate existing Python using domain shapes

Annotate every named function and method, including tests, nested functions, fixtures and callbacks. Use explicit generator/iterator return types for yielding fixtures, concrete pytest and Playwright types for their objects, parameterized collections and subprocess results, and precise callable signatures. Let local inference stand where it establishes the type; annotate initially empty or heterogeneous state when inference is insufficient.

Use `TypedDict` for fixed JSON-shaped records, dataclasses for owned state and small protocols for actual structural interfaces. Keep each declaration with the concept that owns it or in its own domain-named file when independently reusable. Do not create a generic `types.py` or a parallel model hierarchy. An externally dynamic result can enter as `object`, but validate or narrow it before exposing a typed structure. Contain unavoidable dynamic SDK results at that boundary. Do not use broad `Any`, return-type `Any`, unchecked casts, fake stubs for repository code, relaxed annotation rules or blanket ignores to make checking pass. A cast or rule-specific suppression is acceptable only for a demonstrated external typing limitation with an adjacent reason and preserved runtime validation.

Fix defects discovered by checks and add a reproducer before a functional repair, following the testing skill. Preserve hook approval, child-process restrictions, secret masking, timeouts and resource cleanup. Annotation-only edits must preserve CLI arguments, serialization, exit codes and test behavior. Refactor only code that stands in the way of this task, within root naming and unit-boundary rules.

Alternatives: inferred types alone leave function boundaries insufficiently specified. Annotating everything as `dict[str, object]` or `Any` avoids modeling the structures the code relies on. Forcing every record into a dataclass would unnecessarily change JSON contracts and inflate the migration.

### 4. Use hidden mise tasks and a check-only commit step

`python:lint` and `python:format` run the runner from the worktree root and remain hidden. Add Python lint to the root command's continue-on-error aggregation and Python formatting to the root format dependencies. All eight everyday commands remain unchanged.

Add one hk Python step. Select it for any repository Python file, root or nested Python configuration, the browser dependency lockfile, `mise.toml`, `hk.pkl` and Git ignore configuration. Include additions, deletions and renames supported by hk's selected-file input. The check always invokes full `mise run python:lint`, never a formatter and never a selected-file-only type check. Preserve `fix = false` and `stash = "none"`. Dependency installation and ignored caches are allowed; tracked source, the index and the stash are unchanged.

The pinned hk version automatically selects existing paths only, using the destination of a rename. Explicit deleted paths still select a step. Cover explicit deletion input and Python-to-Python renames in integration tests, document the full-check command for deletion-only and extension-changing renames, and retain unconditional full checking in CI. Updating or wrapping hk's own Git selection is outside this change.

Alternatives: checking only selected files misses incompatible callers. Automatically fixing on commit violates the repository's shared-working-tree policy. Running all application lint for every Python change adds unrelated work.

### 5. Reuse one CI workflow and gate both publishing paths

Create `.github/workflows/lint-python.yml` with `workflow_call`, read-only top-level permissions, a bounded timeout and the existing pinned checkout and mise actions. Install only Python, uv, Ruff and ty at the versions from mise, and call `mise run python:lint`. Do not add tokens, Docker setup, application image dependencies or independent tool version pins. Python quality adds no paid review invocation.

Call it as `lint-python` from both `ci-release.yml` and `deploy-dev-image.yml`. Add the release job to the version-selection prerequisites. Add it as a prerequisite of `push-dev-image`, so neither architecture can publish before quality passes; the existing manifest dependency then covers the combined image. Do not add `always()` or another condition that allows publishing after a failed, skipped or cancelled quality check. Retain existing workflow events, branch filters, unrelated checks and permissions.

Extend `scripts/release_gate_test.py` and the workflow tests to cover these transitive dependencies, the shared entry point, pinned tools and absence of bypass conditions. These checks are local and require no push.

The existing `lint-shell` job also runs `scripts/` tests (`.github/workflows/ci-release.yml:82`). Once those tests exercise the real Python tools and hk, extend that job's mise installation list with uv, Ruff, ty and hk as needed. Keep its existing tools and test command; do not skip the new tests in CI or move the whole script suite into the lightweight Python lint workflow.

Alternatives: duplicating the job in both workflows creates two tool setup lists to maintain. A release-only gate leaves the dev-image publishing path unprotected. A check after publishing reports the defect too late.

### 6. Verify enforcement through real command boundaries

Add tests under `scripts/` that create isolated temporary Git repositories and run the actual quality runner against typed valid code and deliberately invalid snippets. Verify missing annotations, explicit dynamic parameter annotations, type mismatches, formatting differences, hidden and untracked files, paths with spaces, deleted and ignored files, stage aggregation, source/index/stash preservation and format idempotence. Reuse the locked dependency environment in fixtures rather than download dependencies for every case. Use hk's actual plan output for positive and negative file-selection cases, including configuration-only changes, and its actual Python step for execution. Validate root command aggregation through its entry point, isolating unrelated module checks in a fixture rather than running all application checks for every negative case.

Run every existing Python test collection affected by the migration. In particular, `mise run e2e:unit`, `mise run e2e:lifecycle`, the lifecycle interruption suite and `mise run e2e:browser` are required because all their Python code is in scope. Annotation work on lifecycle fixtures must still prove cleanup and interruption behavior. Keep the backend/frontend routine run out of this tooling-only change unless a necessary repair touches those modules; never run `mise run test-full` without a direct user request. Finish with root lint and the Python hk check.

Alternatives: configuration-string assertions alone do not prove that hidden files are checked or that annotation failures block a commit. A hosted CI run cannot be a completion prerequisite that the implementing agent cannot perform locally. Running the entire full suite is unnecessary when the affected Python suites have direct commands.

### 7. Record the new convention without duplicating implementation detail

Add a short Python typing convention to root `AGENTS.md`: annotate named callables, use meaningful value types and do not bypass checking with blanket suppressions. This is a new lasting convention explicitly requested by the user. Keep tool inventories, rule lists, runner signatures and invocation details in configuration and `docs/testing.md`, linking rather than copying. No architecture document change is required because the dependency and image boundaries are preserved.

Alternatives: leaving the convention only in this archived change makes it hard to find later. Copying all selected rules and command internals into agent instructions creates multiple sources that drift.

## Risks / Trade-offs

- Broad Python edits can change hook behavior or resource cleanup accidentally. Keep formatting and annotation changes distinguishable from defect repairs, rerun the existing suites and add regression cases for each functional repair.
- The browser suite has dynamic fixtures and SDK results, and ty may expose external typing limitations. Model real inputs first; document only narrow, evidenced exceptions and do not weaken global checking.
- The first commit check on a fresh checkout may install the locked pytest/Playwright packages. Reuse uv's cache; install neither browser binaries nor application images. Setup failure blocks checking visibly.
- A new nested project could change discovery or import resolution. The single Git inventory, explicit root configuration and integration fixtures prevent silent loss of coverage.
- Python configuration is now a prerequisite for dev-image publication. A broken check intentionally blocks publication; workflow-dependency tests verify there is no condition that bypasses it.
- Code snippets stored as strings in infrastructure tests are fixture data, not independently executable repository files. Keep deliberately malformed snippets when they test failure handling; type the surrounding tests and runners.
- The existing CI paragraph in `docs/testing.md` describes pull-request runs while workflow events select main/manual and develop runs. This discrepancy was reported during exploration. Preserve workflow events; keep documentation edits focused on the new checks and their actual placement rather than changing unrelated pipeline policy.

## Migration Plan

1. Add pinned tooling, shared configuration and hidden commands without introducing a tolerated-error baseline. Use the standalone checks to guide the cleanup.
2. Format and fully type scripts, then review hooks, then browser infrastructure and scenarios. Run each area's direct checks while progressing and repair discovered defects with regression tests.
3. After all Python passes, enable root lint/format integration, hk and the shared CI workflow, then finish enforcement tests and documentation.
4. Run the selected local verification, leave task completion evidence ready for implementation review, and stop at the pre-commit boundary. One OpenSpec change covers both the cleanup and enforcement; no commit or push is part of its tasks.

No data migration or deployment step is needed. If the implementation has to be withdrawn, revert its code, configuration and workflow integration together; do not leave an enabled gate referring to removed commands.
