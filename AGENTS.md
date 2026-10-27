# Databasus — Agent Rules and Guidelines

This document contains project-wide coding standards and best practices for Databasus.
This is NOT a strict set of rules — it is a set of recommendations to help write better, more consistent code.

Per-folder rules live next to the code they govern:

- [`backend/AGENTS.md`](backend/AGENTS.md) — backend code rules, Go + Gin + GORM + PostgreSQL
- [`agent/verification/AGENTS.md`](agent/verification/AGENTS.md) — verification agent code rules, Go CLI
- [`frontend/AGENTS.md`](frontend/AGENTS.md) — frontend code rules, React + TypeScript + Vite + Ant Design + Tailwind
- [`website/AGENTS.md`](website/AGENTS.md) — marketing site rules and its translations, Next.js
- [`assets/readme/AGENTS.md`](assets/readme/AGENTS.md) — translated copies of the root `README.md`
- [`adr/AGENTS.md`](adr/AGENTS.md) — how to write architecture decision records

This root file holds the engineering philosophy that applies everywhere.

---

## Mandatory communication skill

At the start of every turn, before sending any message or taking any action, read the [how-to-communicate skill](.agents/skills/humanizer/SKILL.md) completely. Load it explicitly regardless of automatic skill selection. This applies to every request and workflow, for all agents working in this repository, including Claude Code and its subagents.

Apply the skill in embedded mode to all agent-authored prose, including chat responses, progress updates, plans, documentation, OpenSpec artifacts, review findings, commit messages and pull request text. Prioritize the reader's understanding, factual accuracy and necessary context before style and brevity. Run its draft, audit and final pass internally, then emit only the final text. Preserve exact code, commands, paths, identifiers, schemas, required templates, quotations and user-provided text unless the user asks to edit them. Select relevant information for original answers without hiding material limitations; preserve substantive claims when editing supplied text unless summarization is requested. Editing must not change facts, behavior, scope or technical meaning.

Reply to the user in the language they write in, and keep everything committed to the repository in English, with only the exceptions [Language in code](#language-in-code) lists. Build sentences a reader parses on the first pass, and match an answer's size to the question's: a one-line question needs neither headings nor a list.

---

## Development environment

Work happens on a Fedora host; any other host distro is on its own. Tool versions are pinned in [`mise.toml`](mise.toml): run `mise install` once and activate mise in your shell, so every command below resolves to the pinned version. In a fresh worktree, copy `.env.example` to `.env`.

The repository's skills live in `.agents/skills/`; `.claude/skills` is a symlink to that directory, so each skill has one source file.

| Command | Purpose |
|---|---|
| `just run` | Start the app locally: dev database, test fixtures, WAL backup source with its load generator, backend and frontend; seeds a dev account with databases and restarts the backend when its code changes |
| `just test` | Run the full backend suite |
| `just test-package <packages>`, run in `backend/` | Run the given backend test packages |
| `just test-filesystem` | Run the Docker filesystem suite, mostly left to CI |
| `just pull-testcontainers` | Pull the database images the backend tests use |
| `just lint` | Check every module without changing files |
| `just format` | Format every module |
| `just doctor` | Check that the review hooks find the tools they need |
| `goose -dir backend/migrations create <name> sql` | Create a migration |
| `goose -dir backend/migrations up` (or `down`), with `.env` exported | Apply or roll back migrations on the dev database; goose reads `GOOSE_DRIVER` and `GOOSE_DBSTRING` from the environment |
| `swag init -d . -g cmd/main.go -o swagger`, run in `backend/` | Regenerate the API docs |

### Code search and navigation

Search text with `rg`. For definitions, references, callers, implementations and type errors, use the LSP tool: the project enables the gopls and typescript-language-server plugins in `.claude/settings.json`, and `mise.toml` pins both servers. The TypeScript server takes TypeScript from `frontend/node_modules`, so it needs `pnpm install` in `frontend/`. It answers "who calls this" and "what breaks if I rename this" more reliably than a text search, and reports compile errors right after an edit. Check shell scripts with `shellcheck`.

### Running commands

Run tests, linters and builds as ordinary foreground commands, so the user sees which command runs and that the agent is working. A backend, frontend or other long-lived process the agent started to check its work is stopped before the task ends. Long-lived processes the user wants running are started by the user.

### Several agents, one set of containers

Several agents may work on this repo at once, in different worktrees, sharing one Docker daemon, one set of test containers and one pool of test metadata databases. Don't let that stop you from running tests — simultaneous runs are rare and usually harmless. Two habits keep them that way:

- Lint and format whenever you want: the linters, the formatters and the frontend's `pnpm test` touch nothing shared.
- Test the packages you touched, and the packages that test the same feature, with `just test-package ./internal/features/<feature>/...` in `backend/`; the database slot it claims is described in [`backend/internal/config/config.go`](backend/internal/config/config.go). An agent never runs `just test`, not even at the end of a change, and never makes finishing a task depend on it: it resets the test containers and slot databases every other agent uses. The full suite is for the developer and CI. A fresh environment needs one full run first, which the developer starts.

When a failure makes no sense — a container that vanished, a database that emptied itself — a parallel run is worth suspecting in maybe one case in twenty. Re-run before hunting the bug.

---

## Tests

When you write, change, run or review tests, load the [testing skill](.agents/skills/testing/SKILL.md) first. It holds the rules every module shares; the module docs add the rules of their stack.

---

## Checking the app in a browser

When you start the app or look at its pages, load the [run-app skill](.agents/skills/run-app/SKILL.md) first. It says when to reuse, start and stop the app and how to sign in.

---

## Language in code

**English only in code, comments, identifiers, log messages, API strings, test assertions, and commit messages.** No other language inside `backend/` or `agent/` at all, and none inside `frontend/src/` outside the interface dictionaries named below — not even for user-facing fallback copy or error messages.

Translated end-user content is allowed in six places and nowhere else: the website page copies (`website/app/[lang]/*/content/<locale>.tsx`), the README translations (`assets/readme/README.<locale>.md`), the seven language-switcher labels in the root `README.md`, the frontend interface dictionaries (`frontend/src/shared/i18n/locales/<locale>.ts`), the shared website locale maps listed in [`website/AGENTS.md` - Language exception](website/AGENTS.md#language-exception), and the quoted example pairs in the [how-to-communicate skill](.agents/skills/humanizer/SKILL.md), which model conversation output rather than documentation prose. Even there, file names (locale codes aside), identifiers, dictionary keys and commit messages stay English.

The dictionaries are the only target-language text anywhere under `frontend/`, `index.html` included. Every other file there stays English, fallback copy and error messages included: a component renders a translation by looking up a key, never by holding the translated string itself.

---

## Engineering philosophy

**Think like a skeptical senior engineer and code reviewer. Don't just do what was asked — also think about what should have been asked. Catch real issues, not theoretical ones.**

### Task tiers (scale your response to the task)

- **Trivial** (typos, formatting, single-field adds): apply directly. Steps 5 only.
- **Standard** (CRUD, typical features): steps 1, 5.
- **Complex** (architecture, security, performance-critical): all steps.
- **Unclear** (ambiguous requirements): steps 1 and 4 are mandatory.

### Steps for non-trivial tasks

1. **Restate the objective**, list explicit + inferred assumptions, flag shaky ones.
2. **Propose solutions** — for complex tasks, 2–3 approaches including a simpler baseline; recommend one with tradeoffs (complexity, maintainability, performance, extensibility).
3. **Identify risks** — edge cases, security/privacy, performance, operational concerns (deployment, observability, rollback). Before finalizing, ask "what could go wrong?" and patch.
4. **Handle ambiguity** — pick a reasonable default, label it, note what changes under alternative assumptions.
5. **Deliver quality** — correct, testable, maintainable code with minimal tests/validation. Prefer controller tests over unit tests.
6. **Fix root causes, not symptoms** — ask "why did this happen?" and address the underlying issue.

### Review hooks

Every change answers to this document **and** the module doc for each area it touches. The module docs add stack-specific rules on top of this one; they never replace it, so a change under `backend/` answers to both.

Which review hooks exist, when they run and which model reviews are described in the docstring and comments of [`.agent-hooks/review_gate.py`](.agent-hooks/review_gate.py), the reviewer prompts in `.agent-hooks/*-prompt.md`, and the specs `openspec/specs/proposal-review` and `openspec/specs/implementation-review`.

What the agent does with findings:

- A blocking finding is resolved by the agent itself, without asking the user, and then the agent finishes the turn again. It fixes the code, the documentation or the change's artifacts; when the implementation is right, it fixes the delta spec. A false finding is rejected with the reason in the response.
- A `minor` finding is optional.
- A `scope` finding proposes cutting the feature. The agent does not apply it; the summary shows the user what they lose and what they save, and the user decides.
- After a full review blocks the turn, whether a reviewer found problems or a check before the reviewers failed, the agent finishes the turn again with one short line about its fixes that names every rejected finding with the reason. The repeat review reads that reply, and the summary for the user waits until the review accepts the result. After a quick check blocks a finished change, the next reply is that summary; after it blocks a plan, the agent revises the plan and shows it again.
- The summary for the user describes the final state as if there had been no review: what was done, which decisions were made, which checks passed, what remains and the next step. It leaves out review rounds, the findings and the fixes made for them.
- Run the linter and formatter for each directory the diff touches before finishing.

### The best code is unwritten

Before doing a task, check whether it is needed in this form. If the feature can be skipped or done much more simply, offer that to the user before implementing: what they lose and what they save.

- Product: an existing feature, a manual process or an external service already covers the need; a narrower option covers the same need; a setting, screen or scenario is rarely needed and costs more than it gives.
- Code: what is needed already exists in the repository or its dependencies; a small refactoring lets existing code be reused instead of writing new code; the standard library or a ready tool is enough.

The user decides; the task is never cut without their consent. Simplify code within the task yourself, and agree a large rework first per [Refactoring within a feature](#refactoring-within-a-feature).

### Refactoring within a feature

The system improves gradually, along with ordinary tasks. If code the task touches stands in the way of the task or breaks the rules of the module docs or an accepted ADR, the refactoring is part of the same task; no separate task for later. The refactoring is limited to the code the task works with and does not touch other agents' changes in the working tree. The agent refactors the touched code while implementing, without asking the user, and lists no refactoring suggestions for it. Its summary names only a refactoring that goes beyond the change, together with its cost, because such a rework needs the user's agreement first.

### Other agents' changes in the working tree

Several agents sometimes work in one branch at once. A file or change the agent did not create within the current request belongs to someone else. Never delete, revert, format, stage or commit someone else's changes. If they must be moved aside with `git stash` before a commit, ask the user first.

### Agent memory

The agent writes no rules, preferences or facts to its local memory outside the repository, such as `~/.claude/projects/*/memory`. Other agents and people cannot read that memory, and it drifts from the repository rules. A rule or preference that must outlive the session goes, with the user's consent, into an `AGENTS.md` file or a repository skill.

### Naming

Name variables and functions for **intent**, not mechanism. Naming is the biggest readability lever — avoid generic placeholders (`data`, `handle`, `process`, `tmp`, `helper`, `manager`), type-suffix noise (`nameStr`, `agentList`, `tokenObj`), and mechanism-flavored names (`tickNow`, `hbResp`, `dataObj`).

Booleans take an `is` / `can` / `has` / `should` prefix (`isAllowed`, `canAccess`, `hasItems`, `shouldRetry`) — never bare nouns/verbs like `allowed` or `touches`.

State that holds "the entity currently being X-ed" must include the entity: `createdAgent`, not `created`; `listedAgents`, not `listed`; `deletingAgentId`, not `deletingId`. Pair related state explicitly: `revealedToken` + `revealedTokenAgentName`, not `tokenAgentName`. Loop variables get the domain word — `for _, agent := range listedAgents` (Go) — single letters only inside trivial one-line lambdas (`.map((a) => a.id)`).

Test-mechanism names (`got`, `want`, `expected`) are not acceptable — use the domain noun: `persistedAgent`, `firstRotation`, `secondRotation`. Match domain language across the wire — if the API says `agent`, don't rename to `worker` or `node` in client code.

Idiomatic short names stay where the convention is well-established: Go receivers (`s *AgentService`, `r *AgentRepository`, `ctx *gin.Context`) and JS/TS error catches (`} catch (e) {`). When no good name exists, the abstraction is wrong — extract or rename it, don't reach for `helper` / `tmp` / `manager`.

Examples:

```go
// bad
var hbResp HeartbeatResponse
var got *Agent
for _, a := range listed { ... }

// good
var heartbeatResponse HeartbeatResponse
var persistedAgent *Agent
for _, agent := range listedAgents { ... }
```

```ts
// bad
const [tickNow, setTickNow] = useState(Date.now());
const [deletingId, setDeletingId] = useState<string | null>(null);
const [tokenAgentName, setTokenAgentName] = useState('');

// good
const [currentTimeMs, setCurrentTimeMs] = useState(Date.now());
const [deletingAgentId, setDeletingAgentId] = useState<string | null>(null);
const [revealedTokenAgentName, setRevealedTokenAgentName] = useState('');
```

### Functions, methods and types

The intent rule applies to callables and types too — this is where it's violated most:

- **A name that needs a "what" comment is a naming bug.** Doc comments are for *why*, ordering, and hidden constraints — never to restate behavior. If you wrote `// Foo does X` above `func Foo`, rename `Foo` until that comment is redundant, then delete it. Keep only the sentence a name *cannot* carry (e.g. "must be called before the container exists").
- **Predicate methods read as a question**, same as boolean vars: `IsAborted(id)`, not `AbortedContains(id)`; `HasCapacity()`, not `CapacityCheck()`.
- **The name states everything the unit does.** A function that records *and* cancels is `recordAndCancelAborts` — or it's two functions. A name that hides a second effect is a lie.
- **Getters take a `Get` prefix and name the entity.** `GetRunningVerificationIDs()`, not `Active()`, `RunningVerificationIDs()`, or `DiskUsageBytes()`. This is a deliberate house style: prefer the explicit `Get` *even though vanilla Go idiom omits it* — it keeps accessors visually distinct from actions and matches the backend (`GetAuditLogService`, `GetGlobalAuditLogs`). Constructors stay `New...`; a getter that returns a bool is still a predicate, so it stays `Is/Has...`, not `Get...`.
- **Long positional parameter lists become a struct.** ~4+ params, or two same-typed params adjacent → a named parameter/DTO struct (the input counterpart to the result type, e.g. `spawnPlan` → `SpawnSpec`). Kills call-site ordering bugs and the comment explaining argument order.
- **Type names: avoid generic *and* avoid package stutter.** Not `Manager` / `Provisioner` / `Handler` (generic), not `container.ContainerManager` (stutters — `revive` fails CI). If the only honest name stutters, put the noun on the *variable*, not the type: keep the type `container.Manager`, name the variable `containerManager`. If no precise type name exists, the abstraction is wrong — split it.

```go
// bad — comment restates the name; name hides the second effect; predicate isn't a question
// applyAborts records the abort set and cancels each registered job.
func (h *Heartbeater) applyAborts(ids []uuid.UUID) { ... }
func (h *Heartbeater) AbortedContains(id uuid.UUID) bool { ... }

// good — the names carry it; no comment needed
func (h *Heartbeater) recordAndCancelAborts(ids []uuid.UUID) { ... }
func (h *Heartbeater) IsAborted(id uuid.UUID) bool { ... }
```

### Linting and formatting

After each change run linting and formatting depending on folder you are working it.
- backend and agent have `just lint` and `just format` in their folder
- frontend has `pnpm lint`, `pnpm typecheck` and `pnpm format`
- `just lint` and `just format` at the root cover all three; the root `just lint` also runs shellcheck on every shell script

### Comments are a last resort

The default is **no comment**. Before writing one, reach for a clearer name or a smaller function first — self-explanatory code beats commented code every time. A comment is justified only when it carries something the code cannot: a *why* (business rule, hidden cross-system constraint, non-obvious algorithm or optimisation, ADR reference). Never write a comment that restates *what* the code does or narrates the obvious — if a `// Foo does X` comment sits above the code, that's a naming bug: rename until the comment is redundant, then delete it. This applies everywhere, tests included — a well-named test needs no header comment explaining what it checks.

### No issue numbers in production code

Fixing an issue does not put its number in the code. Production files, comments and error messages state the constraint itself — an issue number is a pointer to a discussion the reader has to leave the code to follow, and it rots the moment the tracker moves.

The one exception is a test written specifically to pin that issue's behaviour: there the number belongs next to the test, because it says *why this case exists* and where the reproduction came from.

### No "how it was" comments, no unrequested backward compatibility

Don't write comments that explain previous behavior ("used to be X", "was renamed from Y", "kept for legacy callers"). Code shows the current state; history lives in git.

Don't preserve backward compatibility unless the user asks for it. No deprecation shims, no aliases, no fallbacks for the old shape. When planning a change that would break existing callers, schemas, configs, or APIs, call out the break explicitly in the plan. If the user approves it, delete the old code outright — do not leave a transition layer behind.

### Security

Databasus handles sensitive data, so security is a layered defence. When working in the repo:

- Don't disable or weaken security checks to make a build pass. For genuine false positives, suppress explicitly with a documented reason (e.g. `.trivyignore`) and explain why in the PR.
- Pin every new GitHub Action to a full commit SHA with a `# vX.Y.Z` tag comment. No floating tags like `@v4` or `@main`.
- Workflows default to top-level `permissions: contents: read`; elevate per-job only when justified.
- Keep Dockerfiles free of secrets, floating base-image tags and unjustified root. If root-at-start is required (PUID/PGID remap, volume chown, initdb), drop privileges with `gosu` before `exec`-ing the app.
- Never log secrets, tokens or credentials. Redact at the logger layer, not at call sites.

The README's `🛡️ Security & reliability engineering` section is the public-facing version of these practices — keep both consistent if substance changes. That section, like the rest of `README.md`, exists in 7 languages: a substantive edit lands in the English file and in all six `assets/readme/README.<locale>.md` copies.

---

## Documentation consistency

All project documentation agrees with itself and with the code: every `AGENTS.md`, `adr/`, `docs/`, the repository's own skills, the READMEs and code comments. OpenSpec specs and changes are outside this rule; the OpenSpec process keeps them consistent.

When an edit in one place affects others, update every affected place in the same task: a renamed command, a changed rule, a new path, a different order of steps. Before handing the work over, search the repository for every mention of what changed and bring them in line.

Agent instructions are the git-tracked `AGENTS.md` files and the repository's own skills under `.agents/skills`, that is every skill except the generated `openspec-*`. They hold principles and conventions that change how the agent behaves and outlive the next feature, not the implementation. An instruction changes when a rule changes, not when the code it points to changes. It may hold a table of documents with the topic of each, a table of commands with the purpose of each, the stack of a part, a link to a file as an example or a source of values, and one sentence saying that a linter enforces a rule.

Everything else lives next to what it describes:

| What | Where |
|---|---|
| Inventories of what exists: features, endpoints, checks, tools | the code, the generated API description, the program's README |
| The state of the code, "currently" and "for now" statements about it | the code and its comments, `docs/` |
| How a process, module, command or deployment works; helper signatures and options | the code and its comments, `docs/`, `README.md` |
| Why an architectural choice was made | `adr/` |
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
