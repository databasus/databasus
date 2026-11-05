## Why

No document says how code inside a package is split into files. `backend/AGENTS.md:24` asks for one role per file, and `frontend/AGENTS.md:205` asks for one type per file in `model/`. Neither rule covers the common case: a unit made of one function, one type or a few types and functions that belong together. No review hook checks the split either, so an agent can add new logic to whatever file is open. Then the file tree stops telling what a package does. The rewrite plan has moved users and the first part of workspaces into the new layout, and every later milestone moves more code. The rule and its check should be in place before that, so each move follows them from the start. The code that has already moved should be brought in line once.

## What Changes

- The root `AGENTS.md` gets the rule "Code is split into logical units". A unit holds one concept: a type with its methods, a function with its private helpers, or a few types and functions that only make sense together. What changes together sits together. A file holds one unit and is named after it in domain words, so the file tree of a package tells what the package does. The rule leaves the placement of code across packages, modules and layers to `docs/architecture.md`, and a file whose content `docs/architecture.md` or a module doc fixes, such as `module.go` or `errors.go`, follows that rule instead.
- `backend/AGENTS.md` keeps only the backend conventions on top of that rule: role file names and all errors of a package in `errors.go`. `frontend/AGENTS.md` keeps its one-type-per-file rule as the frontend form of the same rule.
- **BREAKING** for the review hooks: the quick check `naming` becomes `code-shape`. It keeps checking names and comments, and it also checks how the code the turn added is split into files and functions. The old gate name and its prompt file are deleted, with no alias. `.claude/settings.json` calls the new name.
- The prompts of `code-shape` and `architecture` draw one boundary. `architecture` judges packages, modules, layers and roles. `code-shape` judges files and functions inside a package.
- One pass over the code that is already in the new layout splits files that mix concepts. It covers `internal/app`, `internal/lib`, `internal/modules`, `cmd`, `tools/archlint` and `e2e/upgrade` in `backend/`, and the frontend files that milestones 3c and 4d changed. `internal/modules` includes `workspaces/module.go` from milestone 5a. Behavior does not change.
- The change lands as two commits. The rule and the gate go first, so the pass over the code runs under the new check. The pass and the archived change go second.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `agent-workflow`: a new requirement says what the quick code-shape check reviews in a finished turn: the split of new code into files and functions, and its names and comments.
- `proposal-review`: the requirement "Quick checks yield to the full proposal review" names the `code-shape` check instead of `naming`.

## Out of scope

- The old code in `backend/internal/features`. Each later milestone moves its part into the new layout, and the moved code then falls under the check. This includes the 55 call sites milestone 5a changed mechanically and the files milestone 5a changed in `features/workspaces`, including `services/role_actions.go`: `workspaces-module` deletes them when it moves the module.
- The plan check and the roles of the full proposal and implementation reviews. A plan rarely shows how code is split into files.
- New linters for file size or function length. The rule splits by concept, not by length.
- Renaming code beyond what a split needs.
- Any change to the product's behavior, the HTTP API or the schema.

## Impact

- Rules: root `AGENTS.md`, `backend/AGENTS.md`; `docs/architecture.md:101` keeps the role file names and may link to the new rule.
- Review hooks: `.agent-hooks/review_gate.py` (gate list, docstring, comments), `.agent-hooks/naming-prompt.md` replaced by `.agent-hooks/code-shape-prompt.md`, `.agent-hooks/architecture-prompt.md` (the boundary), `.agent-hooks/review_gate_test.py`, `.claude/settings.json`.
- Code, behavior-preserving: the packages listed in "What Changes"; their tests move with the code they test.
- Plan: `temp/refactor/plan.md` gets an item for this change between 5a and 5b.
- The change answers to the root `AGENTS.md`, `backend/AGENTS.md` and `frontend/AGENTS.md`.
