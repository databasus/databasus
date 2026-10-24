## Architecture review

The standard for this review is `docs/architecture.md` and the architecture rules of the module docs. Read the parts that apply to the paths in scope:

- `docs/architecture.md`, completely, for any change under `backend/` or `agent/`;
- `backend/AGENTS.md`: "Where code goes", "Dependencies", "Handlers, services, repositories", "Engines and providers", "Long-running work", "Migrations";
- `frontend/AGENTS.md`: "FSD (Feature-Sliced Design)".

Code under `backend/internal/features`, `backend/internal/util` and `agent/verification` predates `docs/architecture.md`. A change there keeps that code's existing structure and is not blocked for not following the new layout, unless the change itself moves the code. New code under `backend/internal/app`, `backend/internal/modules`, `backend/internal/engines`, `backend/internal/lib` and `backend/internal/agent` follows `docs/architecture.md` fully.

If the candidate is a plan, check that the proposed structure follows these rules. If the candidate is a finished change, use `git status`, the full `git diff HEAD` and the untracked files, and check the changed code.

Check only architecture:

- backend layers and module boundaries: the dependency direction, a module reached only through its `module.go`, dependencies passed through `Deps` and wired in `internal/app`, events instead of reverse imports;
- roles inside a module: handlers call services, services hold the rules and access checks, repositories hold all database access, and files sit in the roles the documents name;
- data ownership: a module touches only its own tables, foreign keys between modules are `RESTRICT`, invariants live in the schema, and schema changes go through migrations in the required form;
- frontend FSD layers and slices: import direction, slice placement, no cross-imports between slices of one layer, the public API of a slice;
- a decision that contradicts `docs/architecture.md` without changing that document in the same change.

If the change edits `docs/architecture.md` or an architecture section of a module doc, check that the edit agrees with the code and with the latest user request.

Do not check names, style, prose or general correctness. Other reviewers do that.
