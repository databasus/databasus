# Databasus — Agent Rules and Guidelines

This document contains project-wide coding standards and best practices for Databasus.
This is NOT a strict set of rules — it is a set of recommendations to help write better, more consistent code.

How the system is built is described in `docs/`:

- [`docs/architecture.md`](docs/architecture.md) — the layout, the module boundaries, data, the API contract, long-running work, and the reasons behind each rule
- [`docs/testing.md`](docs/testing.md) — the kinds of tests, isolation, test selection and runs
- [`docs/glossary.md`](docs/glossary.md) — one term per concept, in prose and in code

Product behavior is specified in `openspec/specs`.

Per-folder rules live next to the code they govern:

- [`backend/AGENTS.md`](backend/AGENTS.md) — backend and verification agent code rules, Go + Gin + GORM + PostgreSQL
- [`frontend/AGENTS.md`](frontend/AGENTS.md) — frontend code rules, React + TypeScript + Vite + Ant Design + Tailwind
- [`website/AGENTS.md`](website/AGENTS.md) — marketing site rules and its translations, Next.js
- [`assets/readme/AGENTS.md`](assets/readme/AGENTS.md) — translated copies of the root `README.md`

This root file holds the rules for every module.

---

## Mandatory communication skill

Read the [how-to-communicate skill](.agents/skills/humanizer/SKILL.md) completely once per context, before the first message or action: at the start of a session, after the context is compacted, and at the start of each subagent. Load it explicitly regardless of automatic skill selection. The skill then stays in context, so it is not reread on later turns. This applies to all agents working in this repository, including Claude Code and its subagents.

Apply the skill in embedded mode to all agent-authored prose, including chat responses, progress updates, plans, documentation, OpenSpec artifacts, review findings, commit messages and pull request text. Prioritize the reader's understanding, factual accuracy and necessary context before style and brevity. Run its draft, audit and final pass internally, then emit only the final text. Preserve exact code, commands, paths, identifiers, schemas, required templates, quotations and user-provided text unless the user asks to edit them. Select relevant information for original answers without hiding material limitations; preserve substantive claims when editing supplied text unless summarization is requested. Editing must not change facts, behavior, scope or technical meaning.

Reply to the user in the language they write in, and keep everything committed to the repository in English, with only the exceptions [Language in code](#language-in-code) lists. Build sentences a reader parses on the first pass, and match an answer's size to the question's: a one-line question needs neither headings nor a list.

---

## Development environment

Work happens on a Fedora host; any other host distro is on its own. Tool versions are pinned in [`mise.toml`](mise.toml): run `mise install` once and activate mise in your shell, so every command below resolves to the pinned version. In a fresh worktree, copy `.env.example` to `.env`.

The repository's skills live in `.agents/skills/`; `.claude/skills` is a symlink to that directory, so each skill has one source file.

| Command | Purpose |
|---|---|
| `just run` | Start the app locally: dev database, test fixtures, WAL backup source with its load generator, backend and frontend; seeds a dev account with databases and restarts the backend when its code changes |
| `just test` | Run every test |
| `just test-package <packages>`, run in `backend/` | Run the given backend test packages |
| `just test-filesystem` | Run the Docker filesystem suite, mostly left to CI |
| `just pull-testcontainers` | Pull the database images the backend tests use |
| `just lint` | Check every module without changing files |
| `just format` | Format every module |
| `just doctor` | Check that the review hooks find the tools they need |
| `goose -dir backend/migrations create <name> sql` | Create a migration |
| `goose -dir backend/migrations up` (or `down`), with `.env` exported | Apply or roll back migrations on the dev database; goose reads `GOOSE_DRIVER` and `GOOSE_DBSTRING` from the environment. The server also applies migrations itself at startup |
| `just generate-api` | Regenerate the servers, the test clients and the frontend types from `backend/api/*.yaml` |

### Code search and navigation

Search text with `rg`. For definitions, references, callers, implementations and type errors, use the LSP tool: the project enables the gopls and typescript-language-server plugins in `.claude/settings.json`, and `mise.toml` pins both servers. The TypeScript server takes TypeScript from `frontend/node_modules`, so it needs `pnpm install` in `frontend/`. It answers "who calls this" and "what breaks if I rename this" more reliably than a text search, and reports compile errors right after an edit. Check shell scripts with `shellcheck`.

### Running commands

Run tests, linters and builds as ordinary foreground commands, so the user sees which command runs and that the agent is working. A backend, frontend or other long-lived process the agent started to check its work is stopped before the task ends. Long-lived processes the user wants running are started by the user.

### Several agents, one machine

Several agents may work on this repo at once, in different worktrees, sharing one Docker daemon and one test PostgreSQL server. Every test takes its own database and gives its containers unique names, as [`docs/testing.md`](docs/testing.md) describes, so runs from different worktrees do not disturb each other.

- Lint and format whenever you want: they touch nothing shared.
- While working, run the packages and tests you need right now. Before handing the work over, run the tests of the packages you touched and of the packages that test the same feature, as [`docs/testing.md`](docs/testing.md#choosing-the-tests-to-run) says.
- Run the full `just test` only when the user asks for it.
- Never stop the shared test PostgreSQL server or prune Docker, because other runs use them.

### Until the migration finishes

The layout and the tooling described in `docs/` arrive milestone by milestone. Until they do, these substitutes apply:

- While backend tests still share per-worker metadata databases, as set up in `backend/internal/config/config.go`, an agent never runs `just test`, and `just test` takes no machine-wide lock yet. A full run resets the containers and databases every other agent uses, so until then the full suite is for the developer and CI.
- Until `just generate-api` exists, regenerate the API docs with `swag init -d . -g cmd/main.go -o swagger` in `backend/`.
- Go code under `backend/` and `agent/` outside the layout block in "Backend layout" of [`docs/architecture.md`](docs/architecture.md#backend-layout) keeps its existing structure until the milestone that moves it, as the first section of that document says.
- Packages outside that layout keep their cross-package test helpers in files behind the `testhelpers` build tag, so the production binary does not link them. `just test-package` passes the tag itself; a `go test` that runs such a package directly passes it too: `go test -race -tags testhelpers -count=1 ./internal/features/storages/...` in `backend/`. Every `go` command in a Claude Code session sees the tag, so the production graph is checked with `go run ./tools/archlint` or with `GOFLAGS=` in front of `go list`.

Temporary docs and plan for migration is here - @temp/refactor

---

## Tests

When you write, change, run or review tests, load the [testing skill](.agents/skills/testing/SKILL.md) first. It says how to write a test; [`docs/testing.md`](docs/testing.md) says which kinds of tests exist and how they are isolated.

When you design or change an HTTP operation, load the [api-design skill](.agents/skills/api-design/SKILL.md) first.

---

## Checking the app in a browser

When you start the app or look at its pages, load the [run-app skill](.agents/skills/run-app/SKILL.md) first. It says when to reuse, start and stop the app and how to sign in.

---

## Language in code

**English only in code, comments, identifiers, log messages, API strings, test assertions, and commit messages.** No other language inside `backend/` or `agent/` at all, and none inside `frontend/src/` outside the interface dictionaries named below — not even for user-facing fallback copy or error messages.

Translated end-user content is allowed in five places and nowhere else: the website page copies (`website/app/[lang]/*/content/<locale>.tsx`), the README translations (`assets/readme/README.<locale>.md`), the six language-switcher labels in the root `README.md`, the frontend interface dictionaries (`frontend/src/shared/i18n/locales/<locale>.ts`), and the quoted example pairs in the [how-to-communicate skill](.agents/skills/humanizer/SKILL.md), which model conversation output rather than documentation prose. Even there, file names (locale codes aside), identifiers, dictionary keys and commit messages stay English.

The dictionaries are the only target-language text anywhere under `frontend/`, `index.html` included. Every other file there stays English, fallback copy and error messages included: a component renders a translation by looking up a key, never by holding the translated string itself.

A temporary working document the agent writes for the user outside code, such as a plan, an analysis or draft notes, may be in the language the user speaks while it stays uncommitted. It is translated to English before it is committed.

---

## Review hooks

Every change answers to this document **and** the module doc for each area it touches. The module docs add stack-specific rules on top of this one; they never replace it, so a change under `backend/` answers to both.

Which review hooks exist, when they run and which model reviews are described in the docstring and comments of [`.agent-hooks/review_gate.py`](.agent-hooks/review_gate.py), the reviewer prompts in `.agent-hooks/*-prompt.md`, and the specs `openspec/specs/proposal-review` and `openspec/specs/implementation-review`.

What the agent does with findings:

- A blocking finding is resolved by the agent itself, without asking the user, and then the agent finishes the turn again. It fixes the code, the documentation or the change's artifacts; when the implementation is right, it fixes the delta spec. A false finding is rejected with the reason in the response.
- A `minor` finding is optional.
- A `scope` finding proposes cutting the feature. The agent does not apply it; the summary shows the user what they lose and what they save, and the user decides.
- After a full review blocks the turn, whether a reviewer found problems or a check before the reviewers failed, the agent finishes the turn again with one short line about its fixes that names every rejected finding with the reason. The repeat review reads that reply, and the summary for the user waits until the review accepts the result. After a quick check blocks a finished change, the next reply is that summary; after it blocks a plan, the agent revises the plan and shows it again.
- The summary for the user describes the final state as if there had been no review: what was done, which decisions were made, which checks passed, what remains and the next step. It leaves out review rounds, the findings and the fixes made for them.
- Run the linter and formatter for each directory the diff touches before finishing.

## The best code is unwritten

Before doing a task, check whether it is needed in this form. If the feature can be skipped or done much more simply, offer that to the user before implementing: what they lose and what they save.

- Product: an existing feature, a manual process or an external service already covers the need; a narrower option covers the same need; a setting, screen or scenario is rarely needed and costs more than it gives.
- Code: what is needed already exists in the repository or its dependencies; a small refactoring lets existing code be reused instead of writing new code; the standard library or a ready tool is enough.

The user decides; the task is never cut without their consent. Simplify code within the task yourself, and agree a large rework first per [Refactoring within a feature](#refactoring-within-a-feature).

## Refactoring within a feature

The system improves gradually, along with ordinary tasks. If code the task touches stands in the way of the task or breaks the rules of the module docs or `docs/architecture.md`, the refactoring is part of the same task; no separate task for later. The refactoring is limited to the code the task works with and does not touch other agents' changes in the working tree. The agent refactors the touched code while implementing, without asking the user, and lists no refactoring suggestions for it. Its summary names only a refactoring that goes beyond the change, together with its cost, because such a rework needs the user's agreement first.

## Other agents' changes in the working tree

Several agents sometimes work in one branch at once. A file or change the agent did not create within the current request belongs to someone else. Never delete, revert, format, stage or commit someone else's changes. If they must be moved aside with `git stash` before a commit, ask the user first.

## Agent memory

The agent writes no rules, preferences or facts to its local memory outside the repository, such as `~/.claude/projects/*/memory`. Other agents and people cannot read that memory, and it drifts from the repository rules. A rule or preference that must outlive the session goes, with the user's consent, into an `AGENTS.md` file or a repository skill.

## Naming

Names state intent, not mechanism. The rules below go beyond Go and TypeScript idiom, so the reviewers hold code to them:

- Booleans take an `is`, `can`, `has` or `should` prefix: `isSidebarOpen`, not `sidebarOpen`.
- State that holds the entity currently being acted on names that entity: `selectedDatabaseId`, not `selectedId`.
- Test variables are named after the domain, never `got`, `want` or `expected`: `persistedAgent`, not `got`.
- A loop variable takes the domain word: `for _, storage := range listedStorages`, not `for _, s := range listed`. Single letters stay only inside trivial one-line lambdas.
- Client code keeps the domain words of the API: if the API says `agent`, the client does not call it `worker`.

Short names stay where the convention is well established: Go receivers (`s *NotifierService`), `ctx`, and the JS/TS error catch `catch (e)`.

## Functions, methods and types

- Getters take a `Get` prefix and name the entity, even though Go idiom omits it: `GetWorkspace`, not `Workspace`. Constructors stay `New...`, and a getter that returns a bool is a predicate.
- A predicate reads as a question: `IsAborted(id)`, not `AbortedContains(id)`.
- The name states everything the unit does, or the unit is split: `repairAndSaveConfigForBackupType`, not `saveConfigForBackupType` for a method that also repairs the row.
- Four or more parameters, or two adjacent parameters of one type, become a named struct: `SpawnSpec`, not `spawn(name, image string, ...)`.
- A type name is neither generic nor stuttering: not `Helper` or `Processor`, not `container.ContainerManager`. When the only honest name stutters, the variable carries the noun: `containerManager := container.NewManager(...)`.

## Comments are a last resort

The default is **no comment**. Before writing one, reach for a clearer name or a smaller function first — self-explanatory code beats commented code every time. A comment is justified only when it carries something the code cannot: a *why* (business rule, hidden cross-system constraint, non-obvious algorithm or optimisation, a reference to `docs/architecture.md` or a spec). Never write a comment that restates *what* the code does or narrates the obvious — if a `// Foo does X` comment sits above the code, that's a naming bug: rename until the comment is redundant, then delete it. This applies everywhere, tests included — a well-named test needs no header comment explaining what it checks.

## No issue numbers in production code

Fixing an issue does not put its number in the code. Production files, comments and error messages state the constraint itself — an issue number is a pointer to a discussion the reader has to leave the code to follow, and it rots the moment the tracker moves.

The one exception is a test written specifically to pin that issue's behaviour: there the number belongs next to the test, because it says *why this case exists* and where the reproduction came from.

## No "how it was" comments, no unrequested backward compatibility

Don't write comments that explain previous behavior ("used to be X", "was renamed from Y", "kept for legacy callers"). Code shows the current state; history lives in git.

Don't preserve backward compatibility unless the user asks for it. No deprecation shims, no aliases, no fallbacks for the old shape. When planning a change that would break existing callers, schemas, configs, or APIs, call out the break explicitly in the plan. If the user approves it, delete the old code outright — do not leave a transition layer behind.

## Security

Databasus handles sensitive data, so security is a layered defence. When working in the repo:

- Don't disable or weaken security checks to make a build pass. For genuine false positives, suppress explicitly with a documented reason (e.g. `.trivyignore`) and explain why in the PR.
- Pin every new GitHub Action to a full commit SHA with a `# vX.Y.Z` tag comment. No floating tags like `@v4` or `@main`.
- Workflows default to top-level `permissions: contents: read`; elevate per-job only when justified.
- Keep Dockerfiles free of secrets, floating base-image tags and unjustified root. If root-at-start is required (PUID/PGID remap, volume chown, initdb), drop privileges with `gosu` before `exec`-ing the app.
- Never log secrets, tokens or credentials. Redact at the logger layer, not at call sites.

The README's `🛡️ Security & reliability engineering` section is the public-facing version of these practices — keep both consistent if substance changes. That section, like the rest of `README.md`, exists in 6 languages: a substantive edit lands in the English file and in all five `assets/readme/README.<locale>.md` copies.

---

## Documentation consistency

All project documentation agrees with itself and with the code: every `AGENTS.md`, `docs/`, the repository's own skills, the READMEs and code comments. OpenSpec specs and changes are outside this rule; the OpenSpec process keeps them consistent.

When an edit in one place affects others, update every affected place in the same task: a renamed command, a changed rule, a new path, a different order of steps. Before handing the work over, search the repository for every mention of what changed and bring them in line.

Agent instructions are the git-tracked `AGENTS.md` files and the repository's own skills under `.agents/skills`, that is every skill except the generated `openspec-*`. They hold principles and conventions that change how the agent behaves and outlive the next feature, not the implementation. An instruction changes when a rule changes, not when the code it points to changes. It may hold a table of documents with the topic of each, a table of commands with the purpose of each, the stack of a part, a link to a file as an example or a source of values, and one sentence saying that a linter enforces a rule.

Everything else lives next to what it describes:

| What | Where |
|---|---|
| Inventories of what exists: features, endpoints, checks, tools | the code, the generated API description, the program's README |
| The state of the code, "currently" and "for now" statements about it | the code and its comments, `docs/` |
| How a process, module, command or deployment works; helper signatures and options | the code and its comments, `docs/`, `README.md` |
| Why an architectural choice was made | `docs/architecture.md`, next to the rule it explains |
| How the review hooks work | the docstring and comments of `.agent-hooks/review_gate.py`, the prompts `.agent-hooks/*-prompt.md`, the specs `openspec/specs/implementation-review` and `openspec/specs/proposal-review` |
| The sections of a document | the document itself |

An ordinary feature does not change agent instructions. If a feature needs to, first check that what you add does not belong in the table above.

One rule is written in one place. Other documents link to it instead of retelling it. If you find a contradiction between documents that is unrelated to the task, do not fix it silently: report it in the response.

---

## OpenSpec

The `openspec-*` skills are generated by OpenSpec for Claude Code, plus a hand-added "Humanizer preflight" section; after regenerating them, restore that section.

A request to write a spec, write up a change or write its artifacts goes through the `openspec-propose` skill, and an edit to a finished change through `openspec-update-change`. This way the change gets every artifact in OpenSpec format at once.

When the user asks to pack a change, in any language, the agent first checks that every task in its `tasks.md` is checked; if one is not, it changes nothing and names the unchecked tasks. Otherwise it archives the change through the `openspec-archive-change` skill, and the pack request answers that skill's sync prompt: "Sync now" when the main specs need the delta specs, "Archive now" when they are already synced. Then it commits the implementation, the main specs and the archived change in one commit per the [Git commits skill](.agents/skills/git-commits/SKILL.md). The request is the user's consent to the sync and to that commit, so the agent asks about neither again. The commit leaves out changes the agent did not make, per [Other agents' changes in the working tree](#other-agents-changes-in-the-working-tree), and the agent does not push.

### Discussing a task in explore

When the agent works through a task with the user in `/opsx:explore` or the `openspec-explore` skill, it builds the answer from the essence to the mechanics:

1. What needs to be done, without technical details: what problem exists now, what changes for the Databasus user or for the team, what is out of scope.
2. How it works now. The agent explains the parts of the system the task touches in plain words. It introduces a feature, a service, a migration and other terms before their first use, following the [how-to-communicate skill](.agents/skills/humanizer/SKILL.md). It quotes no code without explaining it.
3. How it will work: an ASCII diagram and a short step-by-step story.
4. Decisions: each as a separate point with its reason and with what breaks under the other choice.
5. How to tell the task is done: tests and observable signs.
6. Which documents will change.

The agent settles technical questions that have an optimal answer itself and explains the choice in point 4. It asks the user only about what the user decides: the size of the feature, product rules, cutting the task per [The best code is unwritten](#the-best-code-is-unwritten). The question goes at the end of the answer, one at a time.

---

## Committing

Follow the [Git commits skill](.agents/skills/git-commits/SKILL.md) for commit messages and branch names.
