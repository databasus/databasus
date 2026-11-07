## Context

See `proposal.md` for motivation and scope. The root `AGENTS.md`, `backend/AGENTS.md` and the testing skill govern this change. Module boundaries and the purpose of the public contract are documented in `docs/architecture.md:57`; communication through module interfaces is required at line 91.

`archlint` currently combines structural checks in `backend/tools/archlint/archlint.go:37`. Its source loader parses files without type information at `backend/tools/archlint/source.go:33`. Production dependency checks already use `go/packages`, but load only import and file information from the binary entry points at `backend/tools/archlint/production_dependencies.go:66`. That view omits test-only consumers and does not identify references to individual declarations.

The existing fixture runner creates small independent source trees and invokes the real checker in `backend/tools/archlint/archlint_test.go:59`. The command already prints each violation and returns a failure status in `backend/tools/archlint/main.go:13`. `mise-tasks/backend/lint:15` already runs this command.

The accepted cleanup in commit `4d8435c6` removed 71 unused aliases and constants. It retained 94 with resolved references, including aliases needed only by public signatures: `LogicalBackupCompletedHandler` at `backend/internal/modules/backups/module.go:53` appears in the subscription signature at line 108.

## Goals / Non-Goals

**Goals:**

- Enforce consumer use for exported aliases and constants while keeping signature dependencies valid.
- Make reference identity, consumer boundaries and transitive dependencies explicit enough to test with source fixtures.
- Reuse the existing architecture-check command, diagnostic format and backend lint integration.

**Non-Goals:**

The exclusions in `proposal.md` apply. This is a source-reference check, not runtime reachability analysis. A reference in an external consumer counts even if a separate dead-code tool might consider that consumer unreachable.

## Decisions

### Check a bounded set of declarations

Candidates are exported `TypeSpec` aliases with an assignment token and exported names in constant declarations in `internal/modules/<name>/module.go`. Newly defined types, functions, methods, variables and fields are not reported by this rule. They can participate in the dependency graph so that an externally used interface, constructor or configuration structure keeps the aliases it requires.

Integrate `checkModuleContractUsage` with the other checks in `archlint.go`, returning ordinary violations under `module-contract-usage`. Keep its implementation in `module_contract_usage.go` with tests in the matching test file. If loading and declaration analysis need independent units, split them by concept rather than expanding the existing syntax loader's responsibilities.

Rejected alternatives: checking every exported member would require decisions about interface methods and implicit implementations beyond the agreed scope; limiting the check to a count of imports would not identify unused re-exports.

### Load resolved references once, including tests

Use the existing `golang.org/x/tools/go/packages` dependency to load the backend's packages with syntax and type-use information and test variants. Load the full package selection rather than only production entry points. Respect the host's current platform and the repository's default build without additional build tags. A contract file excluded from that build is outside this run's guarantee.

Candidate discovery can use the existing parsed sources. When no active candidates exist, avoid the additional type load. Otherwise perform one shared type-information load for all candidate modules. Preserve the existing syntax-based structural checks rather than converting every check to the new representation.

Use source declaration locations and package identity to reconcile production and test variants; raw type-object pointer identity is insufficient across variants. A load or type-resolution error fails analysis. Never convert an incomplete reference index into a list of unused declarations, and do not introduce per-symbol `gopls` subprocesses or a new dependency.

Rejected alternatives: text matching would misread shadowed names, comments and renamed imports; a production-only graph would lose test consumers; loading packages for every declaration would repeat expensive work.

### Define consumers by the owning module boundary

A source reference outside `internal/modules/<owner>/` counts as external, including references from other modules, commands, `internal/app`, `apptest` and test packages elsewhere in the backend. Tests inside the owning module stay inside that boundary even when their package name ends in `_test`. A reference from the module's own implementation or function bodies does not justify exporting an alias.

This follows the domain boundary in `docs/architecture.md:57` and the ownership rule in `backend/AGENTS.md`. The producer's own tests should not force an otherwise internal name into the public contract. They can use the owning internal package when testing behavior outside the HTTP surface is justified by the testing skill.

Rejected alternatives: counting only other production modules would reject names needed by composition and test support; counting every local use would allow constructor implementation details to preserve redundant public aliases.

### Follow dependencies from used declarations

Build a graph of declarations in each public contract. Seed it with declarations that have resolved references from outside their owning module. From each seed, follow references in declaration types, function and method signatures, type parameter constraints, interface members and embeddings, type definitions and declaration initializers. Do not traverse function bodies as public contract dependencies.

Walk the graph until no new declarations are reached. Report each candidate outside that reachable set. Local chains and cycles without an external seed do not preserve their members. References between different modules still count as external source references; this rule does not attempt whole-backend dead-code elimination.

Track each name separately in grouped declarations, and account for dependencies of implicit constant expressions. A used sibling does not exempt the entire group. The checker reports names without rewriting declarations or changing constant values.

```text
External use --> Module --> subscription signature --> callback alias

Unused local declaration --> another unused alias
```

The first alias is required; the second is rejected. This preserves the documented contract shape while preventing unused local declaration chains from passing the rule.

Rejected alternatives: treating every exported declaration as a seed would preserve unused public chains; traversing function bodies would confuse implementation use with contract need; exempting every alias used in any signature would keep aliases belonging to unused declarations.

### Verify the checker with source fixtures and preserve existing checks

Exercise the new rule through the real architecture checker using valid, isolated source fixtures. Assert rule, location and declaration name for rejected aliases and constants, and absence of this rule's violations for accepted cases. Cover renamed imports, shadowed names, external test consumers, own-module tests, signature and transitive dependencies, unused chains and cycles, mixed constant groups, body-only references, test variant deduplication and analysis errors.

Where existing tests intentionally contain structurally invalid programs, preserve their original diagnostic coverage. Keep new usage fixtures type-correct except when testing analysis failure; do not globally disable type errors or weaken unrelated assertions. A check with no candidate aliases or constants needs no type analysis.

Run all architecture-check tests, the checker on the real cleaned backend, backend formatting and lint, and the routine test command. Record architecture-check elapsed time before and after adding the type load, without introducing a timing-sensitive test or claiming a speed target before measuring it.

Rejected alternatives: mocking the reference graph would not exercise import resolution or package variants; snapshots of a deletion count would not establish the rule's behavior.

## Risks / Trade-offs

- Type information makes the check heavier -> load once for all modules, skip it when there are no candidates, and measure the real command during implementation.
- Package test variants duplicate objects and files -> canonicalize declaration identities and deduplicate diagnostics through the checker's existing stable output pipeline.
- A signature-only alias looks unused -> traverse declaration dependencies from externally used public declarations and cover the case with a fixture.
- Fixture compilation errors obscure structural findings -> keep usage fixtures valid and preserve existing checks without suppressing analysis failures.
- Host and build selection limit what the check can see -> state that the guarantee covers the default host build and its tests; a cross-platform matrix is outside this change.

## Migration Plan

Add the principle and enforcement description to the Modules section of `docs/architecture.md`. Keep one source of the rule there; existing agent instructions already link to it. The delta spec adds observable lint behavior to `backend-code-checks`.

The cleaned backend should pass without runtime or contract changes. If implementing the check finds a genuine unused alias or constant missed by the cleanup, verify its references before removing it within this rule's scope; never exempt it merely to make lint pass. No data migration, API regeneration, deployment step or compatibility layer is required.
