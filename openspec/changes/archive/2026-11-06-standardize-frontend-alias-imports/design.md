## Context

See [proposal.md](proposal.md) for motivation and scope. A design is needed because a bulk rewrite must preserve module identity and the lint rule must distinguish internal imports from resource and tooling paths.

- The alias already resolves to `src` in TypeScript (`frontend/tsconfig.app.json:12`), Vite (`frontend/vite.config.ts:8`) and Vitest (`frontend/vitest.config.ts:5`). No new alias is needed.
- Layer direction and public entry points are defined in `docs/architecture.md:323` and `docs/architecture.md:335`. Its file-layout explanation currently permits both spellings (`docs/architecture.md:341`).
- Source lint runs through Oxlint, followed by architecture and style checks, type checking and generated API verification (`frontend/lint/check.sh:4`). The source override currently configures localization alone (`frontend/oxlint.config.ts:32`).
- The architecture scanner already parses imports, re-exports, import-type expressions and dynamic imports. It retains only resolved script dependencies inside the source root (`frontend/lint/frontendDependencies.ts:45`). Alias enforcement must also inspect stylesheet references and unresolved targets, so dependency edges alone are insufficient.
- A source test imports the architecture checker outside `src` (`frontend/src/shared/lib/test/frontendArchitecture.test.ts:9`). Other tests store intentionally invalid module references inside fixture strings in that same file. Neither needs another alias or a blanket test exemption.
- Entry imports include fonts and the global stylesheet (`frontend/src/main.tsx:1`); import sorting is already configured (`frontend/oxfmt.config.ts:10`). Changing path categories can change formatting, so evaluation order needs explicit attention.

The implementation follows root `AGENTS.md`, `frontend/AGENTS.md`, their architecture conventions and the testing skill. It changes a contribution convention rather than product behavior.

## Goals / Non-Goals

**Goals:** Preserve the dependency graph and runtime behavior while establishing one spelling for internal references. Keep the bulk migration independently reviewable before enforcement is introduced.

**Non-Goals:** See the proposal's exclusions. In particular, do not add aliases for infrastructure modules outside `src` or introduce compatibility spellings. Infrastructure that imports source modules must resolve their internal source alias.

## Decisions

### 1. Scope the convention by both importer and target location

For a literal relative module reference in `src`, normalize its location against the importing file's directory. If that location is within `src`, require `@/`. Use path containment, not a raw string prefix, so a neighboring directory with a similar name is not mistaken for the source root. Perform this classification independently of whether a target exists or has a script extension.

Cover static imports, type-only imports and exports, import-type expressions, re-exports, literal dynamic imports and side-effect imports of CSS or other assets. Include no-substitution template literals in dynamic imports. Leave computed dynamic sources outside this change; do not introduce a new restriction on them.

References escaping the source root remain relative when needed by tests, including the existing checker import. Configurations, browser infrastructure and tools outside `src` remain outside this rule, even when they reference source files. Package imports, CSS `url(...)` and `new URL(..., import.meta.url)` retain their semantics.

**Alternatives rejected:** Banning every relative string would rewrite resource URLs and fixture content. Banning every relative import in every frontend file would require additional aliases and runtime configuration for standalone tools. Exempting all tests would leave a large part of the migration unenforced.

### 2. Reuse the existing syntax traversal for a separate lint finding

Extract a focused collection of actual module-reference occurrences from the existing TypeScript traversal and share it between dependency collection and alias enforcement. Keep the alias check in its own logical unit under `frontend/lint`, integrate its findings into the existing architecture-check invocation, and emit a distinct `project/source-alias` diagnostic with the source, location and offending reference. Preserve the existing dependency checker and its findings.

Inspect module-reference nodes directly before module resolution. The alias check therefore detects a relative CSS import and a relative reference to a missing module. It does not treat fixture source strings as imports of the containing test. Do not add a second whole-project parser, a tolerated-findings list or a new dependency.

**Alternatives rejected:** A syntax-only blacklist in the native `no-restricted-imports` rule cannot classify target containment from the importer and would need per-file exceptions for references to external tooling. Enforcing only resolved dependency edges misses stylesheets and unresolved references. A new lint plugin duplicates infrastructure already available in the source scanner.

### 3. Rewrite module specifiers without changing their targets

Use a temporary migration driven by the TypeScript syntax tree, replacing only actual relative module-specifier literals whose normalized target lies within `src`. Derive the alias path from the same target location. Preserve extensions, directory entry-point resolution, imported and exported names, and side-effect ordering. Do not introduce explicit extensions, explicit index filenames, barrel imports or directory moves as part of the rewrite.

Capture the resolved script dependency graph before migration and compare it afterward, retaining occurrence counts. Account separately for stylesheet and other asset references that the script resolver does not expose. Run the existing formatter and inspect resulting import movement, preserving evaluation order, especially around side-effect imports. The migration tool is temporary; the permanent rule prevents recurrence.

**Alternatives rejected:** A textual search-and-replace can alter embedded fixture programs and resource paths. Replacing imports with a slice's own public entry point can create cycles and changes the graph. Keeping both spellings requires contributors to continue choosing between them.

Migration checks exposed source consumers outside the main application configuration: the tooling TypeScript project, the isolated Vitest runner and the Playwright loader. Add the existing source alias to `tsconfig.node.json` and the generated isolated configuration, and select the tooling TypeScript configuration in Playwright. Their own imports remain relative. Existing harness lifecycle tests and browser-test discovery verify that these consumers can still load source modules; this repair is required before the mechanical phase passes.

### 4. Retain architectural checks and document the convention once

External consumers continue importing a slice's public API. References within the same slice continue naming private modules directly, now through `@/`. Preserve architecture fixtures proving that relative and alias spellings are equivalent for dependency checks; the alias rule adds a separate failure for the relative spelling.

Write the durable authoring rule, its scope and exclusions in `frontend/AGENTS.md`. In `docs/architecture.md`, replace the mixed-style statement with the reason for the single spelling and a link to that rule. Search repository-owned documentation for affected guidance and update only contradictory project guidance; retain vendored upstream methodology unchanged.

**Alternatives rejected:** Duplicating the convention across skills and architecture prose makes later updates inconsistent. Weakening dependency checks after standardizing spelling would let aliases bypass module boundaries.

### 5. Separate migration from enforcement

Finish and verify the mechanical phase before changing lint behavior, tests of the new rule or convention documentation. Keep enforcement, its regression coverage and documentation together in the second phase. This supports the user's requested first mechanical commit and second enforcement commit; actual Git operations are outside the task checklist and this planning request.

**Alternatives rejected:** A single combined diff makes the rule harder to review among hundreds of path changes. Enabling the rule first creates a deliberately failing intermediate source tree.

## Risks / Trade-offs

- Import regrouping can affect evaluation order -> Preserve side-effect ordering, compare dependency targets and occurrence counts, and run frontend tests and the production build after migration.
- A bulk rewrite can change fixture programs or module-relative resources -> Restrict replacement to parsed module-reference occurrences and inspect excluded path forms separately.
- A rule based only on resolved modules misses CSS and missing files -> Classify normalized reference locations before resolution and pin those cases in lint tests.
- Same-slice alias references could be mistaken for external public-API access -> Retain same-slice direct imports and verify the existing architecture suite accepts them while rejecting private access from another slice.
- Tooling exclusions could hide internal violations -> Decide scope per occurrence and prove that an allowed external-tool import does not exempt an internal relative reference in the same file.
- Two phases leave a short interval without enforcement -> Complete the second phase before declaring the change implemented.

## Migration Plan

1. Record script dependency targets and import-order constraints, migrate actual internal module references, format changed files, and verify the mechanical phase with frontend lint, frontend tests and the production build. Preserve this diff as the first reviewable phase.
2. Add the scoped alias diagnostic using shared syntax traversal, regression tests and convention documentation. Keep fixtures that intentionally exercise relative references.
3. Repeat targeted lint and architecture tests, run frontend lint and the production build, then run the change verification command. Finish at a reviewed, ready-to-commit working tree.

No database or deployed configuration migration is needed. If a path changes its target during migration, correct the occurrence before continuing. A later rollback can restore the two phases together through the normal Git workflow; no compatibility layer is needed.

## Verification Evidence

- Migrated 2,419 references in 708 source files. A syntax-aware before/after audit preserved all 3,866 resolved module references, the 17 remaining module references, asset targets and explicit side-effect import order. Non-module string literals were preserved apart from the intentional isolated-runner configuration repair.
- The mechanical phase passed all 2,560 existing frontend tests, frontend lint and the production build after configuring the external source consumers. Browser-test discovery loaded all 136 UI scenarios.
- The initial 16 alias-rejection and CLI cases failed against the checker before enforcement was added. After implementation, all 43 alias tests and 21 existing architecture tests passed.
- Final `mise run frontend:lint`, `pnpm --dir frontend build` and `python3 -m unittest discover -s scripts -p 'frontend_lint_test.py'` passed. The script suite checked successful completion and failure propagation through every frontend lint entry point.
- Final `mise run test` passed both backend groups and all 2,603 frontend tests. The first attempt lacked the worktree's `.env`; copying `.env.example` supplied the documented test configuration before the successful repeat.
- `openspec validate standardize-frontend-alias-imports --strict` and Git whitespace checks passed. The migration was preserved separately from enforcement for the requested commit split.
