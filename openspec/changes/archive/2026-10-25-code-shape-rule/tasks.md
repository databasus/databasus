The hook test command below runs from the repository root: `python3 -m unittest discover -s .agent-hooks -p '*_test.py'`.

## 1. The rule

- [x] 1.1 Add the section "Code is split into logical units" to the root `AGENTS.md` right after "Functions, methods and types", with the text of design.md decision 1. Verify with `rg -n '^## Code is split into logical units' AGENTS.md` showing one line after the `## Functions, methods and types` line.
- [x] 1.2 Shrink the file bullet of "Where code goes" in `backend/AGENTS.md` to the backend conventions of design.md decision 1 (role file names, errors in `errors.go`, a file only when it has content) with a link to the root rule, and leave `frontend/AGENTS.md` rule 5 and `docs/architecture.md:101` as they are. Verify with `rg -n 'logical units' backend/AGENTS.md` showing the link, and with `rg -n -i 'one concept|logical unit' AGENTS.md backend/AGENTS.md frontend/AGENTS.md docs` showing the rule stated only in the root `AGENTS.md`.

## 2. The code-shape check

- [x] 2.1 Rename the gate `naming` to `code-shape` in `.agent-hooks/review_gate.py` as design.md decision 2 lists: `GATES`, `GATES_YIELDING_TO_PROPOSAL_REVIEW`, `get_stop_subject`, the call of `is_code_changed_since_request`, the docstring and the comment above that function. In `.claude/settings.json` call `run.sh code-shape` with the status "Checking code shape". Rename the gate's tests in `review_gate_test.py` to the new name. Verify with the hook test command.
- [x] 2.2 Replace `.agent-hooks/naming-prompt.md` with `.agent-hooks/code-shape-prompt.md` holding the standard and the sections "Decomposition", "Names and comments" and the boundary of design.md decision 3. Add hook tests that the reviewer of a finished turn gets a prompt carrying: the section "Code is split into logical units" in its standard; the instruction to read every created source file in full; the rejection of a new file or function with more than one concept and of a concept added to a file that holds a different one; the instruction to leave package, module, layer and role placement to the architecture check; the instruction not to reject an addition to `module.go`, `errors.go`, an entity file with its enum or a role file; and the instruction not to judge concepts mixed before the turn. These tests cover the scenarios of the `agent-workflow` delta: "A new file holds two concepts", "A new function does two things", "An event added to the public contract of a module", "A new concept appended to an existing file", "An edit inside a file that was already mixed", "A unit in the wrong layer" and "A comment that restates the code". Verify with the hook test command.
- [x] 2.3 Add to `.agent-hooks/architecture-prompt.md` the sentence of design.md decision 3 that leaves the split inside a package to the code-shape check, and a hook test that the architecture prompt carries it. Keep the existing test that a turn with only change artifacts skips the quick checks, now with `code-shape` in its list; it covers the `proposal-review` scenario "A turn with only artifacts". Verify with the hook test command.
- [x] 2.4 Search the repository for the old gate name and bring every mention in line. Verify with `rg -n '\bnaming\b' .agent-hooks .claude AGENTS.md docs` printing only prose about naming rules and no gate name, and with `mise run doctor` exiting with status 0. The main spec `openspec/specs/proposal-review` keeps the old name until the archive syncs this change's delta.

## 3. Ready for the first commit

- [x] 3.1 Run `mise run lint` and the hook test command, and check that the work of groups 1 and 2 touches nothing under `backend/internal`, `backend/cmd` or `frontend/src`, so it can be committed apart from the pass as design.md decision 5 describes. Verify that both commands exit with status 0 and that `git status --short -- backend/internal backend/cmd frontend/src` shows no file this change edited.

## 4. The pass over the code

Each task below follows design.md decision 6: read every non-generated source file of the scope, split each file that mixes concepts into files named after them, move the tests with the code, and change no behavior. Before each task, skip any file that `git status` shows with someone else's uncommitted edits, and note it for the summary.

- [x] 4.1 `backend/internal/lib`. Verify with `mise run backend:test-package ./internal/lib/...`, `mise run backend:lint` and `GOFLAGS= go run ./tools/archlint` in `backend/` exiting with status 0.
- [x] 4.2 `backend/internal/app`, `backend/cmd` and `backend/tools/archlint`. Verify with `mise run backend:test-package ./internal/app/... ./cmd/... ./tools/archlint/...` and `mise run backend:lint` exiting with status 0.
- [x] 4.3 `backend/internal/modules`: `users` and `workspaces/module.go` from milestone 5a. Verify with `mise run backend:test-package ./internal/modules/...` and `mise run backend:lint` exiting with status 0.
- [x] 4.4 `backend/e2e/upgrade` and `upgradecheck`. The upgrade test itself runs only with release images, so verify with `go test -count=1 -run '^$' ./e2e/upgrade/` in `backend/` compiling the package, `mise run backend:test-package ./e2e/upgrade/upgradecheck/...` and `mise run backend:lint` exiting with status 0.
- [x] 4.5 The frontend files that `git show --name-only --format= b93d5e2d 3d92a5bb -- frontend/src` lists, without generated files and locale dictionaries. Re-export moved types from the slice `index.ts`. Verify with `pnpm --dir frontend test` and `mise run frontend:lint` exiting with status 0.

## 5. Plan and checks

- [x] 5.1 Add the item `code-shape-rule` to `temp/refactor/plan.md` between 5a and 5b with one line about the rule, the gate and the pass, and check it. Verify with `rg -n 'code-shape-rule' temp/refactor/plan.md` showing a checked line between the `5a` and `5b` lines.
- [x] 5.2 Run `mise run lint` and the hook test command, and fix what they report. Verify that both exit with status 0.

## Change verification

```sh
mise run backend:test-package ./internal/lib/... ./internal/app/... ./cmd/... ./tools/archlint/... ./internal/modules/... ./e2e/upgrade/upgradecheck/...
pnpm --dir frontend test
```
