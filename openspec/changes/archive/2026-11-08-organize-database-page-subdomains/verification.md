## Implementation

The database page now collects its code in the nine areas described in `design.md`. The page's public export, root page/list/tab composition, entity contracts and runtime behavior are preserved. No compatibility forwarding files or new architecture exceptions were added.

The page has 267 script files, down from 307. The WAL model has 19 files, down from 40. All 20 separate label-map files were merged into their corresponding enum units. Three focused generated-schema alias contracts cover WAL responses, logical backups and verification; the single physical backup row alias retains its own file. Tiny view and draft declarations were placed with their owners as described in the design.

The style check's existing chart-color owner and its existing test fixture now reference the relocated `wal-monitoring/lib/chartColors.ts`; the ownership policy and assertions are unchanged.

## Verification evidence

- `mise run doctor` passed before applying the change.
- The baseline `mise run frontend:lint`, `pnpm --dir frontend typecheck` and `pnpm --dir frontend test src/pages/databases` passed. The page suite contained 69 files and 1,031 tests.
- Each relocation task's focused area tests and explicit typecheck passed. After all relocations, frontend lint, explicit typecheck and all 1,031 page tests passed again before declaration consolidation.
- WAL's 182 tests and explicit typecheck passed after its consolidation. After the remaining consolidations, frontend lint, explicit typecheck and all 1,031 page tests passed.
- `pnpm --dir frontend test tooling/styleChecks.test.ts` passed all 125 existing style-check tests after updating the chart owner path.
- Targeted `pnpm --dir frontend exec oxfmt` runs formatted only the affected page and style-check files. The final `mise run frontend:lint` passed formatting, source, architecture, style, type and generated API checks.
- The final explicit `pnpm --dir frontend typecheck` and `pnpm --dir frontend build` passed.
- A temporary TypeScript AST audit compared the page against the saved pre-refactor snapshot: all 1,128 non-import/re-export top-level declarations and statements were preserved, including the bodies of all 69 test files. The same audit found no unresolved internal static imports or runtime import cycles. This audit is additional refactor evidence, not a new repository test suite.
- Repository searches found no remaining references to the obsolete module locations outside historical change artifacts. `git diff --check` and `openspec validate organize-database-page-subdomains --strict` passed.
- `mise run test` passed with exit code 0: both backend groups passed, and the complete frontend suite passed all 193 files and 2,796 tests.

## Review

Mandatory implementation review follows completion of the checks. No commit, merge or push has been performed.
