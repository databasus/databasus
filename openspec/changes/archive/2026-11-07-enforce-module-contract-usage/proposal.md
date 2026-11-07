## Why

The module contract cleanup removed 71 unused aliases and constants, but the architecture check still allows the same declarations to return. Enforcing their actual use keeps contracts focused on what consumers need and removes the need for repeated manual audits.

## What Changes

- Add a `module-contract-usage` rule to `archlint` for exported type aliases and constants declared in root module contracts.
- Accept a declaration when a package outside its owning domain module references it, or when it is required by an externally used public declaration through its signature, type definition or declaration dependencies.
- Include application composition, commands, test support and tests outside the owning module as consumers. References confined to the owning module, including its own tests, do not establish an external need.
- Resolve references by declaration identity and follow dependencies transitively, so import aliases, signature-only types and unused declaration chains receive the right result.
- Report each unused declaration with its location, name and rule, through the existing backend lint command. Keep the check read-only.
- Document the rule in `docs/architecture.md` and cover it with architecture-check fixture tests.

### Out of Scope

- Checking whether individual interface methods are called or concrete implementation methods are needed.
- Checking exported functions, variables, newly defined types or struct fields as unused exports; these declarations can still be dependency roots or connectors for checked aliases and constants.
- Replacing module interfaces, removing `internal/module`, changing application wiring, or changing HTTP and database contracts.
- A repository-wide dead-code detector, automatic deletion, an exception baseline, or a build-platform matrix.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `backend-code-checks`: Reject unused exported aliases and constants in domain module contracts while preserving declarations needed by external consumers and their public contract dependencies.

## Impact

- `backend/tools/archlint` gains reference analysis and a new diagnostic. The existing `mise run backend:lint` command already invokes it; no new command is needed.
- `golang.org/x/tools/go/packages` is already a backend dependency. The check will need type information and test consumers in addition to the existing syntax-only analysis.
- `docs/architecture.md` describes the rule; `backend/AGENTS.md` already directs readers there for module boundaries, so it needs no duplicate rule.
- The change follows the root `AGENTS.md`, `backend/AGENTS.md` and the testing skill. New checks are tested against source fixtures, without application services or databases.
- The previously accepted module cleanup is the baseline and should pass unchanged. No runtime behavior, existing caller, schema or API changes are planned. Future unused contract aliases and constants will fail lint.
