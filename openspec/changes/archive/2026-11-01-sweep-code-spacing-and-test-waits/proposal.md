## Why

The rule "Blank lines separate logical steps" reached `AGENTS.md` and the review hooks only recently, and the review checks only the code a diff changes. Code that nobody touches keeps its long unbroken runs of statements, where a reader cannot see the steps of a function. Separately, the frontend testing rule forbids a time limit on one wait or one test, and the browser lifecycle suite still sets several.

## What Changes

- Cover all 2335 hand-written source files changed on `develop` since 2026-10-18, and add blank lines where code has collapsed into a wall of statements that mix distinct logical steps. A parser finds the 942 files that hold a block able to meet that definition, and only those are read.
- Make no micro-fixes: a short block, or a long block that forms one step, stays as it is. The sweep adds empty lines only.
- In the browser lifecycle suite, remove the per-wait and per-test limits and give the suite one shared test limit in its config.
- Make the suite's check that a stopped gateway is gone assert a refused connection. Today a request that hangs until its one-second limit also passes.

No program behavior changes, and nothing breaks for callers.

Out of scope:

- Files outside the frozen list: files not changed since 2026-10-18, generated code, migrations, `testdata`, `.d.ts` files, locale dictionaries and translated website pages.
- Any other code-shape fix found while reading, such as names, comments, file splits or misplaced logic.
- Removing blank lines the rule would call misplaced.
- A linter rule for blank lines, or a change to the review hooks.
- Waits in backend tests, including the bare `<-time.After(...)` sites.
- Protective deadlines, service limits and shared limits in the frontend test tooling.
- The frontend test waits another agent is changing in the working tree.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

None. This change refactors code and test tooling without changing behavior, so `.openspec.yaml` sets `skip_specs: true`.

## Impact

- Code: blank lines across `backend/`, `frontend/`, `website/`, `.agent-hooks/` and `scripts/`, and `frontend/tooling/runBrowserE2E.test.ts` with `frontend/vitest.browser-e2e.config.ts` for the waits.
- Rules the change answers to: the root `AGENTS.md`, `backend/AGENTS.md`, `frontend/AGENTS.md` and `website/AGENTS.md`.
- History: `git blame` of existing lines stays as it is, because the sweep only adds empty lines. Open branches may see merge conflicts around them.
- Documentation: `docs/testing.md` names the browser lifecycle suite's own test limit. The other rules are already written.
