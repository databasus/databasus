## Context

The draft of the target documents is uncommitted in the working tree: `docs/architecture.md`, `docs/testing.md`, `docs/glossary.md` and the `api-design` skill are untracked, `adr/` is deleted, and the backend comments, the website FAQ and several main specs already point to the new documents instead of ADRs. See proposal.md for why the draft has to be final before code moves.

Facts that shape the decisions:

- The quick reviewers judge against written rules only. `.agent-hooks/common-prompt.md` tells them to "report only violations of a rule written in a document you read", and `.agent-hooks/naming-prompt.md` names the sections "Naming" and "Functions, methods and types" of the root `AGENTS.md` as its standard. A rule removed from the documents stops being enforced.
- The root `AGENTS.md:112-131` holds general engineering advice: the "skeptical senior engineer" introduction, task tiers and six steps. The one project-specific point in it, preferring controller tests, is already in `.agents/skills/testing/SKILL.md:12`. `AGENTS.md:228-234` repeats the lint and format commands from the command table at `AGENTS.md:41-54`.
- `AGENTS.md:27` requires reading the communication skill completely "at the start of every turn". The skill has 235 lines. The `session-start` hook already asks for it in a fresh context.
- `assets/tools/AGENTS.md` is a 138-line checklist for adding and refreshing client tool bundles. `assets/tools/` has no `CLAUDE.md`, so Claude Code never loads that file by itself.
- `just test-affected` does not exist. The target documents describe it in `AGENTS.md:44`, `AGENTS.md:69-70`, `docs/testing.md:52-60`, `docs/testing.md:72` and `.agents/skills/testing/SKILL.md:104`. `AGENTS.md:77` describes the substitute: the packages the agent touched and the packages that test the same feature, run through `just test-package`.
- The full proposal review runs one reviewer per delta spec plus the roles `proposal and design` and `tasks` (`.agent-hooks/review_gate.py:1225`). The `proposal and design` role checks simplicity, architecture and consistency at once (`.agent-hooks/proposal-prompt.md`, "Role: proposal and design"). The quick `necessity`, `architecture`, `diff` and `naming` checks skip that turn (`review_gate.py:699-701`).
- The full implementation review runs one reviewer per delta spec plus `documentation and tasks` and `tests` (`review_gate.py:1202`). Architecture is checked only by the quick `architecture` check, which sees the paths of the last request.
- No reviewer prompt mentions the `api-design` skill, and the `naming` check reads only `.go`, `.ts` and `.tsx` files (`review_gate.py:76-77`).
- `docs/architecture.md:5`, `.agent-hooks/architecture-prompt.md:9` and the last line of "Until the migration finishes" in `AGENTS.md` name only `backend/internal/features`, `backend/internal/util` and `agent/verification` as old code. Old code also lives in `backend/cmd`, `backend/internal/config`, `backend/internal/middleware` and `backend/internal/storage`.
- `assets/tools/AGENTS.md` mixes rules of agent behavior with facts about the code and the image: how the backend resolves a binary path (lines 11-14), the runtime image's glibc and OpenSSL versions (lines 44 and 50), which bundles exist (lines 66-67), how the restore order and the compression are chosen (lines 68-74), how the startup check works (lines 131-135), and an issue number (line 102).
- The comment at `review_gate.py:95-97` says the quick reviewer runs "one step below" the writing model, while `QUICK_REVIEWER` uses `opus`, the same model as the agent.

The change answers to the root `AGENTS.md` sections "Documentation consistency" and "Review hooks", and to the `agent-instructions` capability, which limits what agent instructions may hold.

## Goals / Non-Goals

**Goals:**

- The documents in the working tree become the committed target state, with nothing that contradicts the code it marks as already moved.
- Agent instructions hold only rules specific to this repository, and every rule a reviewer enforces stays written where the reviewer reads it.
- The full proposal review checks necessity, architecture and names independently, and the full implementation review checks architecture across the whole change.

**Non-Goals:**

- Changing the quick checks' triggers, models or effort.
- Rewriting the communication skill or the testing skill beyond the test selection rule.
- Frontend rules.

## Decisions

### Remove general advice, keep every house rule in `AGENTS.md`

The root `AGENTS.md` loses the "Engineering philosophy" introduction with its task tiers and steps, and the "Linting and formatting" section. The other sections under "Engineering philosophy" stay as top-level sections of their own, because each holds a rule specific to this repository: what the agent does with review findings, the best code is unwritten, refactoring within a feature, other agents' changes, agent memory, naming, comments, issue numbers, no unrequested backward compatibility, security. The sentence "This root file holds the engineering philosophy that applies everywhere" and its echoes in `openspec/config.yaml:15` and `frontend/AGENTS.md:4` change to say that the root file holds the rules for every module.

Alternatives:

- Move the rules into the reviewer prompts. The agent would learn a rule only after a reviewer blocks the turn, so every violation costs a review round. The reviewers already read `AGENTS.md`, so nothing is gained.
- Keep the advice. It costs context in every session and adds nothing the model does not do by default. The plan and necessity reviewers still check the behaviors it described: solving the cause, and looking for a simpler option.

### Naming stays in `AGENTS.md`, cut to rules that differ from idiom

"Naming" and "Functions, methods and types" keep their headings and keep only rules a model would not follow by default: booleans with `is`/`can`/`has`/`should`, state named with its entity, no `got`/`want`/`expected` in tests, the `Get` prefix against Go idiom, predicates as questions, a name that states every effect, a struct for four or more parameters or two adjacent parameters of one type, no generic or stuttering type names, the allowed short names. General advice such as "naming is the biggest readability lever" and the list of generic placeholders goes. Examples come from this repository's code.

Alternatives:

- A separate naming skill. A skill is loaded when its description matches the task, and naming applies to every edit, so loading would be unreliable exactly where the rule matters.
- The naming hook prompt only. Same cost as above: the agent learns the rule after the block.

### The communication skill is loaded once per context

`AGENTS.md` requires loading the skill at the start of a session, after the context is compacted and at the start of each subagent, before the first message or action. The skill stays in context after that, so rereading it every turn repeats the same text.

Alternatives:

- Every turn, as now. It costs 235 lines per turn without new information.
- Only the `session-start` hook. Subagents get no such hook, so the rule in `AGENTS.md` still has to name them.

### The client tool bundle checklist becomes a skill

The rules of agent behavior in `assets/tools/AGENTS.md` move to `.agents/skills/client-tool-bundles/SKILL.md`, whose description triggers on adding an engine version or refreshing a bundle. A skill is an agent instruction, so the facts listed in Context do not move with them: how a path is resolved goes to a comment next to the mapping in the code, the runtime image's library versions and the list of bundles to `assets/tools/README.md`, how the startup check and the restore order work to the code that does it, and the issue number is dropped in favor of the constraint it illustrates. `assets/tools/AGENTS.md` is deleted, and `docs/architecture.md:27` and any other mention point to the skill. The tables of versions and download sources stay in `assets/tools/README.md`.

Alternatives:

- Add `assets/tools/CLAUDE.md` that imports the file. Adding a version starts in backend code that reads the version, so the agent may reach this folder only at the end, after the decisions the checklist exists to shape.
- Move it into `assets/tools/README.md`. The README is for humans and holds the inventory; the checklist is a rule of agent behavior.

### Drop `just test-affected`

The target documents describe the rule that is already in force as permanent: while working the agent runs the package or test it needs, and before handing the work over it runs the tests of the packages it touched and of the packages that test the same feature, with `just test-package` in `backend/`. The full run is `just test`, under a machine-wide lock. "Until the migration finishes" loses the substitute line, and `AGENTS.md`, `docs/testing.md` and the testing skill no longer mention `test-affected`. The rule that an agent never runs `just test` stays in "Until the migration finishes" until per-test databases arrive.

Alternatives:

- Build `test-affected` from the `go list -deps -test` graph. It is a tool to maintain, and its selection still misses packages that test a feature through a running app. The full implementation review already runs the change's verification commands, and its tests reviewer checks that they cover every package with changed tests. CI runs everything.

### Separate reviewers in the full proposal review

`first_roles` of `PROPOSAL_REVIEW` becomes `necessity`, `architecture`, `names`, `proposal and design`, `tasks`. `proposal-prompt.md` gets a section per role:

- `necessity` takes the simplicity check from the old `proposal and design` role and returns `simplify` or `scope`;
- `architecture` checks the design against `docs/architecture.md` and the architecture sections of the module docs that `architecture-prompt.md` lists, and an HTTP operation the change adds or changes against the `api-design` skill, and returns `mismatch`;
- `names` checks names introduced by the artifacts against "Naming", "Functions, methods and types", the naming rules of the module docs, the table naming rules in "Data" of `docs/architecture.md` and `docs/glossary.md`, and returns `mismatch` or `gap`;
- `proposal and design` keeps only consistency between decisions and the truth of facts about the code.

The categories do not change. Only the descriptions of `gap` and `mismatch` in `proposal-prompt.md` widen to the glossary and the naming rules.

Alternatives:

- Keep one reviewer and add only `names`. One reviewer that checks simplicity, architecture and consistency spends its effort on whichever it finds first, and the findings of one kind crowd out the others.
- A new `naming` category. A wrong name contradicts a written rule, which is what `mismatch` already means, and a new category would change the schema and the blocking rules for no difference in what the agent does.

### An architecture reviewer in the full implementation review

`first_roles` of `IMPLEMENTATION_REVIEW` gains `architecture`, and `implementation-prompt.md` gets "Role: architecture". The reviewer reads the same documents as `architecture-prompt.md` and reviews every path of the change, committed parts included, the way the other full review roles do. A new category `architecture` joins `implementation-result.schema.json` and the blocking categories, and the block instruction tells the agent to move the code or to change `docs/architecture.md` in the same change.

Alternatives:

- Report architecture as `spec_mismatch` or `bug`. The block instruction for `spec_mismatch` tells the agent to choose between the code and the delta spec, which is the wrong choice for an architecture rule.
- Rely on the quick `architecture` check. It reads the paths changed in one request, so a violation spread over several requests passes.

### The architecture reviewers check the HTTP contract

The rules for HTTP operations belong to the architecture reviewers, because the contract is the boundary between the backend and its clients. `architecture-prompt.md` tells the quick architecture check to read `.agents/skills/api-design/SKILL.md` and check against it when a plan adds or changes an HTTP operation or a turn edits a file under `backend/api/`. The "Role: architecture" sections of `proposal-prompt.md` and `implementation-prompt.md` say the same for the full reviews, with `mismatch` and `architecture` findings. `diff-prompt.md` already leaves architecture to that reviewer, so the contract is checked once per review.

Alternatives:

- Put the reading instruction in `common-prompt.md` for every reviewer. Every reviewer would read the skill, but none would own its rules, so a broken rule could pass with each reviewer assuming another one reports it.
- A separate quick check for the contract. The contract appears in a later milestone and changes in a fraction of turns; a linter for the contract arrives with it and takes the mechanical rules.
- Extend `naming` to `.yaml`. Names are only part of the API rules, and the error format, pagination and access marks would stay unchecked.

### Old code is everything outside the target layout

`docs/architecture.md:5`, `architecture-prompt.md:9`, the new "Role: architecture" sections in `proposal-prompt.md` and `implementation-prompt.md`, and the last line of "Until the migration finishes" in `AGENTS.md` stop listing old directories. They point to the layout block in "Backend layout" of `docs/architecture.md` instead, which also holds `backend/api` and `backend/migrations`, and say that Go code under `backend/` and `agent/` outside that layout is old code that keeps its structure until the milestone that moves it. The rule covers only `backend/` and `agent/`; the frontend rules in the architecture review are not affected. The `backend-machine-checks` change draws the same line for the linters.

Alternatives:

- Add the four missing roots to the list. Any old root missed now or added by mistake later would be held to the new layout, and the lists in the documents and in the lint configuration would drift apart.

### Reviewing the draft

The draft is read against the code and the refactor plan. A statement in `docs/` may describe the target layout, because `docs/architecture.md` names the paths not yet moved and `AGENTS.md` lists the substitutes in "Until the migration finishes". Every statement about a path that already exists has to be true for that path. Every mention of ADRs or of `adr/` outside the archive of finished changes goes.

## Risks / Trade-offs

- [The proposal review costs more: a change with two delta specs gets seven reviewers instead of four, in two waves of at most six] → The review runs once per change, and the budget in `FULL_REVIEW_BUDGET_SECONDS` already fits two waves.
- [Shorter naming rules make the naming reviewer less strict] → Every rule that differs from idiom keeps one bad and one good example, so the reviewer compares against the same standard.
- [Without `test-affected` the agent can miss a package that depends on the change] → The testing skill names the packages that test the same feature, the tests reviewer checks the verification commands, and CI runs every group.
- [The architecture reviewer may report old code that the change only touched] → Its prompt carries the same exemption as `architecture-prompt.md`: code outside the target layout that keeps its structure is not reported.
- [Removing general advice changes agent behavior in ways no test shows] → The plan and necessity checks still review every plan for the cause of the problem and for a simpler option.
