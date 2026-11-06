# Databasus — Agent Rules and Guidelines

This document contains project-wide coding standards and best practices for Databasus.
This is NOT a strict set of rules — it is a set of recommendations to help write better, more consistent code.

How the system is built is described in `docs/`:

- [`docs/architecture.md`](docs/architecture.md) — the layout, the module boundaries, data, the API contract, long-running work, and the reasons behind each rule
- [`docs/testing.md`](docs/testing.md) — the kinds of tests, isolation, test selection and runs
- [`docs/glossary.md`](docs/glossary.md) — one term per concept, in prose and in code
- [`docs/machine-setup.md`](docs/machine-setup.md) — storage recommendations for faster local tests
- [`docs/frontend-design.md`](docs/frontend-design.md) — frontend design conventions and their code owners
- [`docs/frontend-localization.md`](docs/frontend-localization.md) — dictionaries, safe interpolation and regional formats
- [`docs/google-drive-storage-authorization.md`](docs/google-drive-storage-authorization.md) — how a storage editor authorizes with Google Drive

Product behavior is specified in `openspec/specs`.

Per-folder rules live next to the code they govern:

- [`backend/AGENTS.md`](backend/AGENTS.md) — backend and verification agent code rules, Go + Gin + GORM + PostgreSQL
- [`frontend/AGENTS.md`](frontend/AGENTS.md) — frontend code rules, React + TypeScript + Vite + Ant Design + Tailwind
- [`website/AGENTS.md`](website/AGENTS.md) — marketing site rules and its translations, Next.js
- [`assets/readme/AGENTS.md`](assets/readme/AGENTS.md) — translated copies of the root `README.md`

This root file holds the rules for every module.

Before changing a file under a folder with its own `AGENTS.md`, read that file unless it is already in context.

---

## Mandatory communication skill

Read the [how-to-communicate skill](.agents/skills/humanizer/SKILL.md) completely once per context, before the first message or action: at the start of a session, after the context is compacted, and at the start of each subagent. Load it explicitly regardless of automatic skill selection. The skill then stays in context, so it is not reread on later turns. This applies to all agents working in this repository, including Claude Code, Codex and their subagents.

Apply the skill in embedded mode to all agent-authored prose, including chat responses, progress updates, plans, documentation, OpenSpec artifacts, review findings, commit messages and pull request text. Prioritize the reader's understanding, factual accuracy and necessary context before style and brevity. Run its draft, audit and final pass internally, then emit only the final text. Preserve exact code, commands, paths, identifiers, schemas, required templates, quotations and user-provided text unless the user asks to edit them. Select relevant information for original answers without hiding material limitations; preserve substantive claims when editing supplied text unless summarization is requested. Editing must not change facts, behavior, scope or technical meaning.

Reply to the user in the language they write in, and keep everything committed to the repository in English, with only the exceptions [Language in code](#language-in-code) lists. Build sentences a reader parses on the first pass, and match an answer's size to the question's: a one-line question needs neither headings nor a list.

---

## Development environment

Work happens on a Fedora host; any other host distro is on its own. Tool versions are pinned in [`mise.toml`](mise.toml), and the `mise run` commands below are tasks from the same file and `mise-tasks/` that run from any directory of the repository. Run `mise install` once and activate mise in your shell, so every command resolves to the pinned version. Once per clone, run `mise exec lefthook@2.2.1 -- lefthook uninstall` if an older checkout installed lefthook, then `hk install --mise`, so the commit hook from [`hk.pkl`](hk.pkl) checks formatting and lint without changing files; worktrees share the clone's hook. In a fresh worktree, copy `.env.example` to `.env`.

The repository's skills live in `.agents/skills/`, which Codex reads directly; Claude Code reads them through `.claude/skills`, a symlink to that directory, so each skill has one source file.

| Command | Purpose |
|---|---|
| `mise run run` | Start the app locally: dev database, test fixtures, WAL backup source with its load generator, backend and frontend; seeds a dev account with databases and restarts the backend when its code changes |
| `mise run test` | Check the work: the backend packages without the end-to-end ones and the frontend tests |
| `mise run test-full` | Run all tests for the usual Docker development environment; [testing.md](docs/testing.md#checking-work) lists the included suites and the Podman/filesystem exceptions |
| `mise run lint` | Check every module without changing files; `backend:lint`, `frontend:lint` and `python:lint` check one area |
| `mise run format` | Format every module; `backend:format`, `frontend:format` and `python:format` format one area |
| `mise run doctor` | Check review tools and whether Codex project hooks are enabled and trusted |
| `mise run review [<change>]` | Manually rerun parallel AI reviews of a finished OpenSpec change when hooks did not run or another review is needed |
| `goose -dir backend/migrations create <name> sql` | Create a migration |
| `goose -dir backend/migrations up` (or `down`), with `.env` exported | Apply or roll back migrations on the dev database; goose reads `GOOSE_DRIVER` and `GOOSE_DBSTRING` from the environment. The server also applies migrations itself at startup |
| `mise run generate-api` | Regenerate the servers, the test clients and the frontend types from `backend/api/*.yaml` |

### Code search and navigation

Search text with `rg`. For definitions, references, callers, implementations and type errors, use the LSP tool: the project enables the gopls and typescript-language-server plugins in `.claude/settings.json`, and `mise.toml` pins both servers. The TypeScript server takes TypeScript from `frontend/node_modules`, so it needs `pnpm install` in `frontend/`. It answers "who calls this" and "what breaks if I rename this" more reliably than a text search, and reports compile errors right after an edit. Check shell scripts with `shellcheck`.

For a client without the LSP tool, such as Codex, use the `gopls` command line for Go navigation and diagnostics, and `pnpm --dir frontend typecheck` to check TypeScript types.

### Running commands

Run tests, linters and builds as ordinary foreground commands, so the user sees which command runs and that the agent is working. A backend, frontend or other long-lived process the agent started to check its work is stopped before the task ends. Long-lived processes the user wants running are started by the user.

### Several agents, one machine

Several agents may work on this repo at once, in different worktrees, sharing one Docker daemon and one test PostgreSQL server. Every test takes its own database and gives its containers unique names, as [`docs/testing.md`](docs/testing.md) describes, so runs from different worktrees do not disturb each other.

- The main agent, the one the user talks to, works in the checkout or worktree its session starts in. It creates a new git worktree only when the user explicitly asks for one, and it uses a worktree the user created as it is.
- A background agent, one the main agent starts, may create a worktree of its own when its work needs one or when the main agent asks it to. The main agent may ask for that.
- Lint and format whenever you want: they touch nothing shared.
- [`docs/testing.md`](docs/testing.md#checking-work) says which test run a change owes. `mise run test` and `mise run test-full` take turns through a machine-wide lock, so one may wait for another; repeating one failing test directly takes no lock.
- Never stop the shared test PostgreSQL server or prune Docker, because other runs use them. The one exception is the recovery of that server after a crash, while no test run is active, which [`docs/testing.md`](docs/testing.md#isolation) describes.

---

## Tests

Choose checks in proportion to the change. Run `mise run test` for substantial features or broad changes to application behavior, or when the user explicitly requests it. Do not run it automatically for small, localized fixes such as button spacing, styling, copy edits or an isolated bug fix; run only the checks relevant to the affected code and behavior. Documentation-only changes need no test run. [`docs/testing.md`](docs/testing.md#checking-work) describes how to select the relevant checks, including long and end-to-end tests.

An agent never runs `mise run test-full` unless the user directly asks for that run.

When you write, change, run or review tests, load the [testing skill](.agents/skills/testing/SKILL.md) first. It says how to write a test; [`docs/testing.md`](docs/testing.md) says which kinds of tests exist and how they are isolated.

During implementation, immediately fix flaky tests, insufficient test timeouts and other important defects exposed by the implementation or its checks. This includes the code changes needed to repair the defect and updates to the OpenSpec artifacts; no separate approval is needed. Fix the cause rather than weakening assertions, bypassing checks or hiding failures with retries. Increase a timeout when the operation legitimately needs more time, and explain the repair and its verification in the final report.

When you design or change an HTTP operation, load the [api-design skill](.agents/skills/api-design/SKILL.md) first.

---

## Checking the app in a browser

When you start the app or look at its pages, load the [run-app skill](.agents/skills/run-app/SKILL.md) first. It says when to reuse, start and stop the app and how to sign in.

---

## Language in code

**English only in code, comments, identifiers, log messages, API strings, test assertions, and commit messages.** No other language inside `backend/` at all, and none inside `frontend/src/` outside the interface dictionaries named below — not even for user-facing fallback copy or error messages.

Translated end-user content is allowed in six places and nowhere else: the website page copies (`website/app/[lang]/*/content/<locale>.tsx`), the README translations (`assets/readme/README.<locale>.md`), the six language-switcher labels in the root `README.md`, the frontend interface dictionaries (`frontend/src/shared/i18n/locales/<locale>.ts`), the shared website locale maps listed in [`website/AGENTS.md` - Language exception](website/AGENTS.md#language-exception), and the quoted example pairs in the [how-to-communicate skill](.agents/skills/humanizer/SKILL.md), which model conversation output rather than documentation prose. Even there, file names (locale codes aside), identifiers, dictionary keys and commit messages stay English.

The dictionaries are the only target-language text anywhere under `frontend/`, `index.html` included. Every other file there stays English, fallback copy and error messages included: a component renders a translation by looking up a key, never by holding the translated string itself.

A temporary working document the agent writes for the user outside code, such as a plan, an analysis or draft notes, may be in the language the user speaks while it stays uncommitted. It is translated to English before it is committed.

---

## Review hooks

Every change answers to this document **and** the module doc for each area it touches. The module docs add stack-specific rules on top of this one; they never replace it, so a change under `backend/` answers to both.

Which review hooks exist, when they run and which model reviews are described in the docstring and comments of [`.agent-hooks/review_gate.py`](.agent-hooks/review_gate.py), the reviewer prompts in `.agent-hooks/*-prompt.md`, and the specs `openspec/specs/proposal-review` and `openspec/specs/implementation-review`.

Before finishing an implementation, the agent runs the relevant formatters, linters and tests itself, then finishes the turn so the review hooks run. Implementation review runs only AI reviewers; it does not rerun linting, tests or builds. `mise run review` is a manual fallback when the hooks did not run or a review needs repeating, not a routine step after implementation.

What the agent does with findings:

- A blocking finding is resolved by the agent itself, without asking the user, and then the agent finishes the turn again. It fixes the code, the documentation or the change's artifacts; when the implementation is right, it fixes the delta spec. A false finding is rejected with the reason in the response.
- A `minor` finding is optional.
- A `scope` finding proposes cutting the feature. The agent does not apply it; the summary shows the user what they lose and what they save, and the user decides.
- After a full review blocks the turn, whether a reviewer found problems or a check before the reviewers failed, the agent finishes the turn again with one short line about its fixes that names every rejected finding with the reason. The repeat review reads that reply, and the summary for the user waits until the review accepts the result. After a quick check blocks a finished change, the next reply is that summary; after it blocks a plan, the agent revises the plan and shows it again.
- The summary for the user describes the final state: what was done, which decisions were made, which checks passed, what remains and the next step. It does not retell the review rounds; `openspec/config.yaml` adds what the summary of an OpenSpec change says about its review.
- `mise run review` runs as a background command, because it can outlast a foreground one. The agent resolves its findings like those of a blocking full review.
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

The agent writes no rules, preferences or facts to its local memory outside the repository, such as `~/.claude/projects/*/memory` or Codex's `~/.codex/memories`. Other agents and people cannot read that memory, and it drifts from the repository rules. A rule or preference that must outlive the session goes, with the user's consent, into an `AGENTS.md` file or a repository skill.

## Naming

Names state intent, not mechanism. The rules below go beyond Go and TypeScript idiom, so the reviewers hold code to them:

- Booleans take an `is`, `can`, `has` or `should` prefix: `isSidebarOpen`, not `sidebarOpen`.
- State that holds the entity currently being acted on names that entity: `selectedDatabaseId`, not `selectedId`.
- Test variables are named after the domain, never `got`, `want` or `expected`: `persistedAgent`, not `got`.
- A loop variable takes the domain word: `for _, storage := range listedStorages`, not `for _, s := range listed`. Single letters stay only inside trivial one-line lambdas.
- Client code keeps the domain words of the API: if the API says `agent`, the client does not call it `worker`.

Short names stay where the convention is well established: Go receivers (`s *NotifierService`), `ctx`, and the JS/TS error catch `catch (e)`.

## Functions, methods and types

Python named functions and methods declare parameter and return types, including tests, fixtures and callbacks. Use concrete collection values and structured records; validate external dynamic values before exposing their typed shape. Do not bypass checking with unrestricted dynamic annotations or blanket suppressions. A necessary suppression names a specific rule and explains the external limitation. [Python quality checks](docs/testing.md#python-quality-checks) describes verification.

- Getters take a `Get` prefix and name the entity, even though Go idiom omits it: `GetWorkspace`, not `Workspace`. Constructors stay `New...`, and a getter that returns a bool is a predicate.
- A predicate reads as a question: `IsAborted(id)`, not `AbortedContains(id)`.
- The name states everything the unit does, or the unit is split: `repairAndSaveConfigForBackupType`, not `saveConfigForBackupType` for a method that also repairs the row.
- Four or more parameters, or two adjacent parameters of one type, become a named struct: `SpawnSpec`, not `spawn(name, image string, ...)`.
- A type name is neither generic nor stuttering: not `Helper` or `Processor`, not `container.ContainerManager`. When the only honest name stutters, the variable carries the noun: `containerManager := container.NewManager(...)`.

## Code is split into logical units

Code is split into units that each hold one concept: a type with its methods, a function with its private helpers, or a few types and functions that only make sense together. What changes together sits together; what changes for a different reason lives apart. A file holds one such unit, so the file tree of a package tells what the package does without opening a file.

- A part that can be reused, tested or changed on its own is a unit of its own and gets its own file. A private helper serving one unit stays with it.
- If the honest name of a file or a function needs "and", it holds two units and is split.
- Split by concept, not by length: one long concept stays whole, two short unrelated concepts are split.
- A file is named after its unit in domain words, never `utils`, `helpers`, `common`, `types` or `misc`. A test file mirrors the file it tests.

Where a unit belongs across packages, modules and layers is decided by [`docs/architecture.md`](docs/architecture.md). A file whose content `docs/architecture.md` or a module doc fixes follows that rule instead: `module.go` holds the whole public contract of a module, `errors.go` holds every error of its package, an enum sits next to the entity whose field it types, and a role file holds its role. The module docs add the other conventions of their stack.

## Blank lines separate logical steps

Use a single blank line to separate logically distinct steps inside a function or other code block: setup, validation, main work and return. Separate independent branches and guards from the surrounding steps. Keep statements that form one step together, including a call and its immediate error check. This applies to all code, including tests, where setup, execution and assertions form distinct steps.

Do not put a blank line after every statement, at the start or end of a body, or inside a single statement split across lines. Never stack two blank lines. Keep a short (≤ 5-line) function together. If blank lines are not enough to make a body easy to follow, extract a function rather than adding comments to label its steps.

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

The README's `🛡️ Security & reliability engineering` section is the public-facing version of these practices — keep both consistent if substance changes. That section, like the rest of `README.md`, exists in 7 languages: a substantive edit lands in the English file and in all six `assets/readme/README.<locale>.md` copies.

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

The `openspec-*` skills are generated by OpenSpec for Claude Code, plus hand-added "Humanizer preflight" and "Environment preflight" sections; after regenerating them, restore those sections.

A request to write a spec, write up a change or write its artifacts goes through the `openspec-propose` skill, and an edit to a finished change through `openspec-update-change`. This way the change gets every artifact in OpenSpec format at once.

When the user asks to pack a change, in any language, the agent first checks that every task in its `tasks.md` is checked; if one is not, it changes nothing and names the unchecked tasks. Otherwise it archives the change through the `openspec-archive-change` skill, and the pack request answers that skill's sync prompt: "Sync now" when the main specs need the delta specs, "Archive now" when they are already synced. Then it commits the implementation, the main specs and the archived change in one commit per the [Git commits skill](.agents/skills/git-commits/SKILL.md). The request is the user's consent to the sync and to that commit, so the agent asks about neither again. The commit leaves out changes the agent did not make, per [Other agents' changes in the working tree](#other-agents-changes-in-the-working-tree), and the agent does not push.

In a linked worktree, the pack also merges the commit into the local `develop` branch, and the request is consent to that too:

- Run the merge in the checkout that holds `develop`; Git cannot move a branch checked out elsewhere.
- Resolve conflicts by your own judgment, then rerun the tests of the packages they touched.
- Ask the user only about a critical conflict that the code and specs do not settle, or when another agent's uncommitted changes block the merge. Abort the merge before asking.

### Environment check before a change

Before writing or editing the artifacts of a change, and before applying one, the agent running the workflow runs `mise run doctor` without `--client`, so it checks that agent's own client. It runs the check once for each such request and before any other step of the workflow. A broken agent or broken hooks then show up before the work starts, not at the review that follows it.

In Codex the agent asks to run the command outside the sandbox, because the check cannot read the hook state from inside it.

If the command exits with a non-zero code, or the agent cannot run it, the agent stops without creating or changing anything, shows the user the output and waits. The agent does not switch clients or work around the failure, and continues only after the check passes or the user tells it to go on.

### Decisions during apply

The standing repair authorization in [Tests](#tests) applies even when a repair requires updating the change's artifacts.

During `/opsx:apply` or the `openspec-apply-change` skill, the agent resolves questions within the agreed change using its own judgment and continues working. It investigates uncertainty and fixes routine problems itself. When a decision is both difficult and consequential, it seeks advice through a separate read-only instance of its active review client, using that client's review profile for full reviews and the [native invocation and child restrictions](.agent-hooks/review_gate.py) documented with the review hooks. It considers the advice and makes the decision. If consultation fails, it decides from available evidence, reports that no advice was received and never switches clients.

The agent asks the user only when it cannot proceed without a decision, information or authorization that only the user can provide. It first checks whether the existing conversation and repository already supply what it needs. Human intervention during apply is exceptional; uncertainty alone is not a reason to stop and ask.

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
