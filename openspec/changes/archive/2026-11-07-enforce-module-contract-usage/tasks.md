## 1. Resolve direct contract consumers

- [x] 1.1 Add the `module-contract-usage` check for exported aliases and constants, integrate it with the architecture-check pipeline, and load resolved references once for the default host build with tests. Add valid source fixtures for unused aliases and constants, direct references from another module, renamed imports, and matching names in comments, strings or unrelated declarations. Assert diagnostic rule, name and location; verify with `go -C backend test -count=1 ./tools/archlint`.
- [x] 1.2 Add fixtures for references from application composition, commands, external test-support packages and external tests outside the production entry-point graph. Reject references confined to the owning module's implementation or its own tests, including a separate test package in the owner's directory. Verify with `go -C backend test -count=1 ./tools/archlint`.

## 2. Preserve required declaration dependencies

- [x] 2.1 Build transitive reachability from externally used public declarations across signatures, interface members and embeddings, type definitions, generic constraints and declaration initializers. Cover callback aliases named only in subscription signatures, indirect alias dependencies and a constant needed by an externally used declaration. Preserve those declarations without traversing local function bodies. Verify the source fixtures with `go -C backend test -count=1 ./tools/archlint`.
- [x] 2.2 Reject aliases and constants supported only by unused local declaration chains, a valid pointer-mediated type dependency cycle, or function bodies inside the owning module. Add mixed constant-group fixtures, including implicit expressions, to show that a used sibling does not exempt unused names. Verify with `go -C backend test -count=1 ./tools/archlint`.

## 3. Verify analysis failures and integration

- [x] 3.1 Add fixtures that confirm package-resolution errors fail analysis, production and test variants do not duplicate findings, and multiple findings have stable order. Preserve the existing structural fixture coverage without suppressing type errors or weakening unrelated assertions. Verify the entire architecture-check suite with `go -C backend test -count=1 ./tools/archlint`.
- [x] 3.2 Document the declaration-use rule and its host-build scope in the Modules section of `docs/architecture.md`, keeping existing agent instructions as links to that rule. Verify the cleaned backend passes without an exception baseline through `mise run backend:lint`; if a genuine unused alias or constant is discovered, confirm its references before removing it and rerun this check. Record the architecture-check elapsed time before and after type analysis while running the existing lint command.
- [x] 3.3 Format only the changed Go files using the backend formatter configuration, then run `go -C backend test -count=1 ./tools/archlint`, `mise run backend:lint` and `mise run test`. Record the results and resolve blocking review findings before reporting the implementation complete.

## Verification results

- The initial unused-alias, unused-constant and misleading-name fixtures failed against the original checker because it returned no violations.
- `go -C backend test -count=1 ./tools/archlint` passed, including the existing structural fixtures and the new usage, dependency, analysis-error, test-variant and build-selection scenarios.
- The changed Go files were formatted with the backend's `golangci-lint fmt` configuration. `mise run backend:lint` passed, including the real contract inventory, without exceptions or further contract removals.
- `/usr/bin/time -f '%e' mise exec -- go -C backend run ./tools/archlint` measured 0.46 seconds before the implementation and 1.25 seconds after it on this host. These observations include `go run` startup and are not a timing guarantee.
- `mise run test` passed: both backend groups succeeded, and the frontend reported 2,616 passing tests across 189 files.
- `openspec validate enforce-module-contract-usage --strict` and `git diff --check` passed.

## Change verification

```sh
mise run test
```
