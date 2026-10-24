## Why

The rewrite of the backend starts with rules and checks, and the target documents for it already sit in the working tree as an unreviewed draft: `docs/`, the trimmed module docs, the removed ADRs. Before code moves, these documents have to be final, and the review hooks have to check what the rewrite is about: architecture across a whole change, names chosen while designing, and the API contract.

Two more problems make the agent instructions cost more than they give. The root `AGENTS.md` spends about a fifth of its length on general engineering advice the model follows anyway, while the reviewers can only enforce rules that are written down, so the house rules have to stay and the generic text has to go. And the communication skill is reread in full on every turn, which repeats 235 lines in every turn of a session.

## What Changes

- Finish the draft of the target documents and agent instructions: `docs/architecture.md`, `docs/testing.md`, `docs/glossary.md`, the module docs, the `testing` and `api-design` skills, the new and amended main specs, and the removal of the ADRs together with every reference to them.
- Remove general engineering advice from the root `AGENTS.md`: the "Engineering philosophy" introduction, the task tiers and the numbered steps for non-trivial tasks, and the "Linting and formatting" section that repeats the command table. Keep every project-specific rule.
- Shorten the "Naming" and "Functions, methods and types" sections to the rules that differ from Go and TypeScript idiom, with examples from this repository. The section names stay, because the naming reviewer reads them.
- Load the communication skill once per context, that is at session start, after the context is compacted and in each subagent, instead of at the start of every turn.
- Move the rules of the checklist in `assets/tools/AGENTS.md` into a repository skill for adding and refreshing client tool bundles, and move its facts about the code and the runtime image to `assets/tools/README.md`, `docs/architecture.md` or code comments. Nothing loads that file automatically today, because the folder has no `CLAUDE.md`.
- **BREAKING** for the documented workflow: drop `just test-affected` from the target documents for good. Before handing work over, the agent runs the tests of the packages it touched and of the packages that test the same feature. The full run stays `just test`. The documents that describe the transitional substitute make it the permanent rule.
- Split the full proposal review's "proposal and design" reviewer into independent reviewers: necessity, architecture, names, and consistency of the proposal and design.
- Add an architecture reviewer to the full implementation review, with its own finding category, so architecture is checked across the whole change and not only per turn.
- Make the architecture reviewers, in the quick checks and in both full reviews, check HTTP operations and edits to `backend/api/*.yaml` against the `api-design` skill.
- Define old code as everything outside the target layout in `docs/architecture.md`, the architecture review prompts and `AGENTS.md`. Today they name only `backend/internal/features`, `backend/internal/util` and `agent/verification`, while old code also lives in `backend/cmd`, `backend/internal/config`, `backend/internal/middleware` and `backend/internal/storage`.
- Fix the comment in `.agent-hooks/review_gate.py` that says the quick reviewer runs on a weaker model than the writing agent.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `proposal-review`: the full proposal review runs separate necessity, architecture and names reviewers next to the consistency and task reviewers, and checks designs against `docs/architecture.md` instead of ADRs.
- `implementation-review`: the full implementation review runs an architecture reviewer with an `architecture` finding category, which also checks contract edits against the API design rules.
- `agent-workflow`: the quick architecture check covers the HTTP contract against the API design rules.
- `agent-instructions`: the root `AGENTS.md` names `docs/architecture.md` instead of ADRs as the place for the reasons behind architectural choices.

## Impact

- Agent instructions: `AGENTS.md`, `CLAUDE.md`, `backend/AGENTS.md`, `agent/verification/AGENTS.md`, `assets/tools/AGENTS.md` and `assets/tools/README.md`, `assets/readme/CLAUDE.md`, the one line in `frontend/AGENTS.md` that points to the removed section, `openspec/config.yaml`, `.agents/skills/testing`, `.agents/skills/api-design` and a new skill for client tool bundles.
- Documents and specs: `docs/`, `adr/` removed, the main specs edited by the draft.
- Review hooks: `.agent-hooks/review_gate.py`, `review_gate_test.py`, `architecture-prompt.md`, `proposal-prompt.md`, `implementation-prompt.md`, `implementation-result.schema.json`.
- Comments in backend code and the website FAQ that pointed to ADRs, as the draft already changed them.
- The change answers to the root `AGENTS.md`, `backend/AGENTS.md`, `website/AGENTS.md` for the FAQ copies, and `assets/readme/AGENTS.md`.

## Out of scope

- The frontend rules, including moving the Feature-Sliced Design section into a skill or adding an FSD linter.
- New linters, `archlint`, test helpers in the production binary and CI changes. The `backend-machine-checks` change does them.
- Choosing the reviewer effort by the size of a change. Effort stays tied to the kind of review.
- A record of slow and flaky tests for the reviewers.
