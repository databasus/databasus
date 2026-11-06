## 1. Mechanical migration

- [x] 1.1 Record the existing script dependency targets and occurrence counts, then migrate the small `widgets/app-shell` slice with a temporary syntax-aware rewrite. Preserve direct internal targets, extensions, fixture strings and resource URLs. Format only changed files and verify with `pnpm --dir frontend test src/widgets/app-shell` and `mise run frontend:lint` before expanding the migration.
- [x] 1.2 Apply the same rewrite to the remaining source modules, tests, public re-exports, import-type expressions, literal dynamic imports and side-effect stylesheet imports. Preserve references outside the source root and runtime evaluation order; compare the script dependency graph and audit asset references against the recorded targets. Give the tooling TypeScript project, isolated test runner and Playwright loader the existing source alias so they can consume migrated source; preserve their own relative imports. Format only owned changes, verify with `pnpm --dir frontend test`, `mise run frontend:lint`, `pnpm --dir frontend build` and `pnpm --dir frontend exec playwright test --list --project=ui`, and retain this as a separate migration diff before starting enforcement.

## 2. Alias enforcement

- [x] 2.1 Add lint-runner regression cases in `frontend/tooling/sourceAlias.test.ts` for sibling and parent imports, type-only imports and exports, import-type expressions, re-exports, dynamic string and no-substitution template imports, stylesheet imports and missing internal targets. Run `pnpm --dir frontend test tooling/sourceAlias.test.ts` against the existing checker and confirm that the relative-reference rejection cases fail because alias enforcement is absent.
- [x] 2.2 Share actual module-reference occurrences from the existing syntax traversal, implement the per-reference containment check in its own lint unit, and integrate `project/source-alias` diagnostics into the existing architecture runner. Preserve dependency collection and avoid new dependencies, duplicate project parsing or recorded exceptions. Verify with `pnpm --dir frontend test tooling/sourceAlias.test.ts src/shared/lib/test/frontendArchitecture.test.ts` and `mise run frontend:lint`.
- [x] 2.3 Complete acceptance and scope tests for permitted aliases, same-slice private targets, packages, source-test references to external tooling, files outside the source root, similarly named neighboring directories, CSS resource URLs, module-relative resource URLs, computed dynamic imports and embedded fixture source. Prove that one allowed reference does not exempt another internal relative reference in the same file, and that forbidden dependencies still fail for both spellings. Verify the diagnostic and nonzero CLI exit as well as clean-code success with `pnpm --dir frontend test tooling/sourceAlias.test.ts src/shared/lib/test/frontendArchitecture.test.ts`.

## 3. Convention documentation and integration

- [x] 3.1 Add the authoring convention and its scope to `frontend/AGENTS.md`, update the alias explanation in `docs/architecture.md` to link to it, and reconcile affected repository-owned guidance without editing vendored methodology. Keep these changes with enforcement. Verify that the documented convention matches executable lint cases with `pnpm --dir frontend test tooling/sourceAlias.test.ts` and `mise run frontend:lint`.
- [x] 3.2 Format owned frontend changes, then run `mise run frontend:lint`, `pnpm --dir frontend build`, `python3 -m unittest discover -s scripts -p 'frontend_lint_test.py'` and the change verification below. The script regression tests must verify that every lint entry point returns a failure when any frontend check fails, covering the scenario "Another frontend check fails". Confirm that the first phase remains mechanically separable, record validation evidence, and resolve mandatory review findings before marking the working tree ready to commit.

## Change verification

```sh
mise run test
```
