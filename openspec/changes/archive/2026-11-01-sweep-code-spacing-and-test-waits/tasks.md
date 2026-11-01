## 1. Preparation

- [x] 1.1 Build the frozen file list outside the repository from `git log --since=2026-10-18` on `develop`, keeping tracked `.go`, `.ts`, `.tsx` and `.py` files and dropping migrations, `testdata`, generated code, `.d.ts`, locale dictionaries, translated website pages, `frontend/tooling/runBrowserE2E.test.ts` and `frontend/vitest.browser-e2e.config.ts`. From it build the candidate list with the thresholds in `design.md`, and verify with `wc -l` that the lists hold about 2335 and about 942 paths.
- [x] 1.2 Write the throwaway checker outside the repository as `design.md` describes, and verify on temporary copies that it accepts an added blank line between statements and rejects a blank line inside a raw or template string, a blank line in template text after a `${...}` substitution, and a changed non-blank line, for Go, TypeScript and Python as each applies.

## 2. Browser lifecycle suite waits

- [x] 2.1 Set the shared 180-second test limit in `frontend/vitest.browser-e2e.config.ts` and remove the per-test limits from `frontend/tooling/runBrowserE2E.test.ts`, and verify with `rg -n '^\s*(\}, )?[0-9_]{4,},?\)?;?$' frontend/tooling/runBrowserE2E.test.ts` that none remain and with `pnpm --dir frontend typecheck` that the suite compiles apart from errors in the other agent's unfinished files.
- [x] 2.2 Replace the limited `vi.waitFor` in `frontend/tooling/runBrowserE2E.test.ts` with `waitForCondition`, and make the stopped-gateway check assert `ECONNREFUSED` without an abort signal, apply the wall criterion to both waits files, and verify with `rg -n 'AbortSignal\.timeout|vi\.waitFor' frontend/tooling/runBrowserE2E.test.ts` finding nothing, `rg -n ECONNREFUSED frontend/tooling/runBrowserE2E.test.ts` finding the check, and `pnpm --dir frontend exec vitest run --config vitest.browser-e2e.config.ts` passing.
- [x] 2.3 Name the suite's own test limit in `docs/testing.md` if the other agent has committed that file, otherwise record it for the report, and verify with `rg -n '180' docs/testing.md` or with the entry in the report.

## 3. Blank-line sweep

Each task takes the candidate files of its area, splits them into batches of up to about 10,000 lines by module and removes files with someone else's uncommitted changes before each batch. Agents add empty lines only where the wall criterion in `design.md` applies. A batch that the checker rejects goes back to an agent.

- [x] 3.1 Sweep the Python files in `.agent-hooks/` and `scripts/` as the pilot that calibrates the criterion, and verify with the checker, `python3 -m unittest discover -s .agent-hooks -p '*_test.py'` and `python3 -m unittest discover -s scripts -p '*_test.py'`.
- [x] 3.2 Sweep the listed `website/` files, and verify with the checker and `pnpm --dir website lint`.
- [x] 3.3 Sweep the listed non-test files in `frontend/`, and verify with the checker and `mise run frontend:lint`.
- [x] 3.4 Sweep the listed test files in `frontend/`, and verify with the checker and `pnpm --dir frontend test`.
- [x] 3.5 Sweep the listed non-test files in `backend/`, and verify with the checker and `mise run backend:lint`.
- [x] 3.6 Sweep the listed test files in `backend/`, and verify with the checker and `mise run backend:lint`.
- [x] 3.7 Sweep the files set aside earlier that are now clean, and verify with the checker. Files still dirty stay untouched and are named in the report.

## 4. Final checks

- [x] 4.1 Run the checker over every swept file in one pass, leaving out the two waits files that are not on the list, and verify that it rejects none and that `mise run lint` passes.

## Change verification

```sh
pnpm --dir frontend test
```
