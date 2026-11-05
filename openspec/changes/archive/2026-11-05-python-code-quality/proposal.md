## Why

Python runs the repository's review hooks, infrastructure checks and browser end-to-end suite, but the shared lint command and commit hook do not check its formatting, lint rules or types. Add mandatory checks and bring all existing Python code up to those checks so defects in this infrastructure receive the same early feedback as application code.

## What Changes

- Add repository-pinned Ruff and ty, shared Python configuration, and hidden `python:lint` and `python:format` tasks. Include them in the existing `lint` and `format` commands.
- Format and lint every repository-owned Python file, including scripts, review hooks, tests, fixtures and browser scenarios. Discover new Python files automatically, including files in hidden directories.
- Fully annotate named functions and methods, including tests, callbacks and fixtures, and give collections, process results and structured data meaningful types. Correct defects exposed by the checks rather than hiding them behind broad ignores, untyped signatures or placeholder `Any` annotations.
- Make the commit hook run the complete Python check for Python source or relevant configuration changes, without fixing, staging or stashing files.
- Add the same Python check to release and dev-image workflows. Its success becomes a prerequisite for release version selection and publication, and for dev-image publication.
- Add behavioral coverage for file discovery, annotation enforcement, type failures, formatting, hook selection and publication dependencies. Document the commands and the Python annotation convention.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `local-development-workflow`: Extend lint, format, pinned tools and commit checks to fully annotated Python throughout the repository.
- `ci-pipeline`: Run the local Python quality check in CI and require success before release or dev-image publication.

## Impact

- Configuration: `mise.toml`, `hk.pkl`, a root `pyproject.toml`, Python cache exclusions and, where needed, the existing browser suite's tool configuration.
- Code: all repository-owned Python, currently 75 files across `scripts/`, `.agent-hooks/` and `e2e/browser/`. Implementation must rediscover the actual inventory rather than rely on this count.
- CI: `.github/workflows/ci-release.yml`, `.github/workflows/deploy-dev-image.yml` and a shared callable Python quality workflow.
- Documentation: `docs/testing.md` and a short Python convention in root `AGENTS.md`. Root `AGENTS.md` governs every planned edit; no module-specific `AGENTS.md` applies to these directories. If a necessary repair reaches a module, its rules must be read before changing it.
- Compatibility: existing script arguments, hook payloads, output formats, exit behavior and e2e entry points remain stable. More commits and CI runs can fail because Python quality is now mandatory. No application API, database schema or user configuration changes are planned.
- Delivery: this is one OpenSpec change. The work may be divided into a Python cleanup slice and an enforcement/integration slice, but completion requires both. This proposal neither creates commits nor authorizes publication.

### Out of Scope

- Replacing the existing test frameworks, rebuilding the Python directory layout or turning scripts into an installable application package.
- Migrating the browser suite's dependencies to a new root environment or adding Python tooling to published application images.
- Changing workflow events, branch policy, security checks or the set of tests run by everyday commands.
- Implementing unrelated application features, broad refactors or additional lint stacks alongside Ruff and ty.
