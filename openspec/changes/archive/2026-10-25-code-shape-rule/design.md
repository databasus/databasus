## Context

See proposal.md for the motivation. The current state that shapes the approach:

- The split of code into files is written in three places, each for one module. `backend/AGENTS.md:24` asks for one role per file. `docs/architecture.md:101` names the role files and puts all errors of a package in `errors.go`. `frontend/AGENTS.md:205` asks for one domain type per file in `model/`.
- The quick check `naming` reviews names and comments the diff added or changed (`.agent-hooks/naming-prompt.md:12`). Names include files, but nothing in the prompt asks what a file holds. The hook starts it only when a `.go`, `.ts` or `.tsx` file under the code directories changed since the request (`.agent-hooks/review_gate.py:686`, `:713`, `:826`). It is wired in `.claude/settings.json:68-69` with the status "Checking names", and the main spec `proposal-review` names it in "Quick checks yield to the full proposal review".
- `archlint` already keeps the methods of a handler, a service and a repository in a file of that role (`backend/tools/archlint/file_roles.go:13`). What a machine cannot tell is whether a file holds one concept, so that part goes to a reviewer.
- The quick architecture check already reviews "files sit in the roles the documents name" (`.agent-hooks/architecture-prompt.md:18`) and leaves names to other reviewers (`:26`).
- A quick check answers only `ok` and `reason` (`.agent-hooks/review-result.schema.json`). It has no `minor` category, so every finding it reports blocks the turn.
- The code in the new layout is about 180 Go files in `backend/internal/app`, `internal/lib`, `internal/modules`, `cmd`, plus `tools/archlint` and `e2e/upgrade`. Most of `internal/lib` already has one concept per file, for example `httpserver/client_ip.go`, `request_id.go` and `access_mark.go`. Milestone 5a added `internal/modules/workspaces/module.go` and the old-code file `internal/features/workspaces/services/role_actions.go`, which `openspec/changes/workspaces-module/tasks.md` task 5.1 deletes along with the rest of `features/workspaces`.

The approach is constrained by the root `AGENTS.md` ("Documentation consistency": one rule in one place; "Refactoring within a feature"; "Other agents' changes in the working tree"), by `backend/AGENTS.md` "Where code goes" and by `frontend/AGENTS.md` FSD rule 5.

## Goals / Non-Goals

**Goals:**

- One rule for every module that says how code is split into units and files, with the module docs adding only their stack's conventions.
- A quick check that blocks a turn when the code it adds mixes concepts in a file or a function.
- The code already in the new layout follows the rule when the change is packed.

**Non-Goals:**

- A machine check by file size or function length.
- Splitting old code in `internal/features`, including the files milestone 5a changed there.

## Decisions

### 1. The rule lives in the root `AGENTS.md`, after "Functions, methods and types"

The rule applies to Go and TypeScript alike, so it belongs in the root document. It sits next to "Functions, methods and types", because that section already says that a unit whose honest name needs "and" is split. The new section extends the same test from functions to files. The text:

```markdown
## Code is split into logical units

Code is split into units that each hold one concept: a type with its methods,
a function with its private helpers, or a few types and functions that only
make sense together. What changes together sits together; what changes for a
different reason lives apart. A file holds one such unit, so the file tree of
a package tells what the package does without opening a file.

- A part that can be reused, tested or changed on its own is a unit of its own
  and gets its own file. A private helper serving one unit stays with it.
- If the honest name of a file or a function needs "and", it holds two units
  and is split.
- Split by concept, not by length: one long concept stays whole, two short
  unrelated concepts are split.
- A file is named after its unit in domain words, never `utils`, `helpers`,
  `common`, `types` or `misc`. A test file mirrors the file it tests.

Where a unit belongs across packages, modules and layers is decided by
`docs/architecture.md`. A file whose content `docs/architecture.md` or a module
doc fixes follows that rule instead: `module.go` holds the whole public
contract of a module, `errors.go` holds every error of its package, an enum
sits next to the entity whose field it types, and a role file holds its role.
The module docs add the other conventions of their stack.
```

`backend/AGENTS.md:24` shrinks to the backend conventions and links to the root rule: role file names (`service.go`, `*_repository.go`, `<entity>.go`), all errors of a package in `errors.go`, and creating a file only when it has content. `frontend/AGENTS.md` rule 5 stays as it is; it is the frontend form of the rule, and its exception for a tiny helper type matches "a private helper serving one unit stays with it". `docs/architecture.md:101` keeps the role file names, because the architecture check reads them there.

Rejected: a rule in each module doc. It would retell one rule three times, and the copies would drift. Rejected: a rule of one type per file everywhere. A Go type with its constructor and an unexported helper is one concept, and three files for it make the tree harder to read, not easier.

### 2. The quick check `naming` becomes `code-shape`

The check now reviews how code is split as well as how it is named, and the old name hides half of its job. `code-shape` pairs with `architecture`: architecture is the shape of the system, code-shape is the shape of the code inside a package. The status in Claude Code reads "Checking code shape".

The rename touches `GATES` and `GATES_YIELDING_TO_PROPOSAL_REVIEW` (`review_gate.py:75-76`), the branch in `get_stop_subject` and the call of `is_code_changed_since_request` (`:686`, `:713`), the docstring (`:17-19`, `:29`), the comment at `:826`, the prompt file, `.claude/settings.json:68-69`, the gate's tests in `review_gate_test.py` and the main spec `proposal-review` through this change's delta. The gate keeps its trigger: a changed `.go`, `.ts` or `.tsx` file under the code directories. The state file of the old gate name stays unused in the session state directory, which is harmless; no migration code is written.

Rejected: `readability`. Splitting code is also about how easily it changes, not only how it reads. Rejected: `structure`. It is easy to confuse with the architecture check. Rejected: keeping the name `naming` and only extending its prompt. It saves the rename edits, but the user asked for the rename, and a gate called `naming` that blocks a turn for a mixed file misleads whoever reads the status line or the hook list. Rejected: keeping `naming` and adding a separate `decomposition` gate. Both checks read the same diff and the same files, and the problems are linked: a function cannot be named honestly because it does two things. One reviewer reports that once; two report it twice and cost a second reviewer run on every turn with code.

### 3. The prompt has two sections

`.agent-hooks/code-shape-prompt.md` replaces `naming-prompt.md` and keeps its standard and its scope rules. It reads:

- **Standard:** the root `AGENTS.md` sections "Naming", "Functions, methods and types", "Code is split into logical units", "Comments are a last resort", "No issue numbers in production code" and the comment rule of "No "how it was" comments", plus the naming, comment and file rules of the module doc for each changed area.
- **Decomposition:** read every source file the turn created in full, and the added parts of every changed file. Reject a new file or a new function that holds more than one concept, and a concept the turn added to an existing file that holds a different one. In the reason name the file or the unit, the rule, and the files or functions the split should produce.
- **Names and comments:** the current text of `naming-prompt.md`, unchanged in substance.
- **Boundary:** do not judge which package, module, layer or role a unit belongs to; the architecture check does that. Do not reject an addition to a file whose content `docs/architecture.md` or a module doc fixes: an event or a contract type in `module.go`, an error in `errors.go`, an enum next to its entity, a method in its role file. Do not judge concepts that were already mixed in a file before the turn.

`architecture-prompt.md` gains one sentence: it judges which package, module, layer and role a unit belongs to, and leaves the split inside a package to the code-shape check.

### 4. Concepts mixed before the turn are not reported

A quick check has no `minor` category: every finding blocks (`review-result.schema.json`). If the check reported every mixed file a turn touched, any one-line fix in such a file would force a split, would grow the diff and could reach into lines another agent is editing. So the check reports only what the turn added. Splitting a mixed file the task works with stays the agent's duty under "Refactoring within a feature", and the pass of decision 6 removes the mixed files that exist now.

Rejected: a `minor` category for quick checks. It would change the result schema and the hook's handling of every quick check for one rule.

### 5. Two commits in one change

The rule and the gate are committed first, once groups 1 to 3 of the tasks are checked: the agent hands the work over at that point, and the commit is made when the user agrees, outside the task list. The pass over the code then runs under the new check, so every split it makes is reviewed by the rule it applies. The second commit comes from packing the change: the split code, the synced main specs and the archived change.

Rejected: one commit. The pass would run under the old `naming` gate, which does not check the split. Rejected: two changes. The pass has no meaning without the rule, and a second proposal for one pass adds a review cycle with nothing to decide.

### 6. The pass over the code

The pass covers:

- `backend/internal/app`, `internal/lib`, `internal/modules` (`users` and `workspaces/module.go` from milestone 5a), `cmd`, `tools/archlint` and `e2e/upgrade`;
- the frontend files that milestones 3c and 4d changed, as `git show --name-only --format= b93d5e2d 3d92a5bb -- frontend/src` lists them, without generated files and locale dictionaries.

For each package or slice the agent reads every non-generated source file and applies the rule. A file that mixes concepts is split into files named after each concept, and its tests move with the code into the matching test files. Code is moved, not rewritten: no signature, behavior or exported name changes unless the split needs a file-level rename. A unit that is in the wrong package or layer is out of this pass and goes into the final summary for the user.

These stay whole on purpose: generated files (`*.gen.go`, `schema.gen.ts`), `errors.go` of each package, an enum next to the entity whose field it types, the locale dictionaries, and `module.go`, which the architecture rules make the single public face of a module.

Rejected: the whole backend. Old code under `internal/features` moves into the new layout milestone by milestone and comes under the check then; splitting it now means doing that work twice. Rejected: the files milestone 5a changed in `features/workspaces`, `role_actions.go` included. `workspaces-module` deletes them when it moves the module, so a split would be thrown away. From milestone 5a only `internal/modules/workspaces/module.go` is in the pass, as part of `internal/modules`, and `docs/architecture.md` already fixes what it holds. Rejected for the same reason: splitting `frontend/src/features/workspaces/ui/WorkspaceMembershipComponent.tsx`, which milestone 4d touched. `workspaces-module` task 6.2 renames and reworks it for the generated client, so a split now would be redone there.

## Risks / Trade-offs

- [The reviewer judges "one concept" by taste and blocks good code] → The prompt requires a reason that names the concepts and the rule, as the common prompt already demands, and the agent rejects a false finding with its reason, as the root `AGENTS.md` allows.
- [The pass touches files that `workspaces-module` or `audit-log-views` is editing at the same time] → Before the pass the agent checks `git status`. A file with someone else's uncommitted edits is skipped and named in the summary, per "Other agents' changes in the working tree".
- [Moving code between files breaks unexported references or test helpers] → Every package is verified with its own tests and `mise run backend:lint` or `mise run frontend:lint` right after its split, and `GOFLAGS= go run ./tools/archlint` checks the import graph.
- [Rename leaves a stale reference to `naming`] → Task 2.4 searches the repository for the old name.

## Migration Plan

A session started before the first commit may still hold the old hook list and call `run.sh naming`. The hook's argument parser refuses the unknown action, and `run.sh` stops the agent with that error, as the spec `review-hook-launch` requires for a hook that fails. Restarting the session picks up the new name. Rollback is a revert of the first commit; the second commit only moves code.
