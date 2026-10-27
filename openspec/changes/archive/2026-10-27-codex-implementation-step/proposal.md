## Why

The user wants to explore, research and write OpenSpec changes in Claude Code, and then hand the implementation step to Codex. Today the repository is set up for Claude Code alone: Codex finishes a turn without any review hook, so a change it implements reaches the user with no lint, no change verification commands and no full implementation review. Codex also runs without the project's browser server, without the test build tag on its `go` commands, and without the module rules that Claude Code loads for each folder.

## What Changes

- Codex gets its own hook configuration that runs the same review gates as Claude Code when a turn finishes: the quick checks, the full implementation review and the full proposal review, with the same timeouts. It also gets the session start hook, which loads the communication skill, and the request snapshot that the gates compare against.
- The reviewers stay in Claude Code. A change that Codex implements is reviewed by the same Claude Code reviewers, on the same models and with the same effort, as one implemented in Claude Code. Codex receives the findings, fixes them and finishes the turn again, in the usual repeat cycle.
- Codex gets a project configuration with the browser server Claude Code uses and with the test build tag on the `go` commands it runs.
- The root `AGENTS.md` gains a rule that makes the module rules reach Codex: before changing a file under a folder with its own `AGENTS.md`, the agent reads that file. Codex does not load these files on its own the way Claude Code loads them through its per-folder `CLAUDE.md`.
- The documents that name Claude Code as the only client now name both clients where both apply: `AGENTS.md`, the `run-app` skill, the hook docstring and the browser server script.
- A hook test keeps the two hook configurations in line, so a gate or a timeout added to one cannot go missing from the other.
- Nothing changes for work done only in Claude Code: its hook configuration and the gate logic stay as they are.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `review-hook-launch`: adds the requirement that Codex runs the same review hooks as Claude Code, with Claude Code reviewers.
- `implementation-review`: the full review models requirement also covers a review started from a Codex session.
- `agent-workflow`: skills, the browser server and the module rules reach Codex sessions from the same sources Claude Code uses, and Codex `go` commands see the test build tag.

## Out of scope

- A plan review in Codex. Codex has no plan tool like `ExitPlanMode`, and in this workflow Codex does not plan. A plan shown in a Codex reply passes without the plan gate.
- Codex as a reviewer. The repository used to review a Codex session with Codex, and this change does not restore that.
- `/opsx:*` slash commands in Codex. Codex starts the same workflows through its skills, for example `$openspec-apply-change <change>`.
- Exploring and proposing in Codex as a supported flow. The proposal review would still run there, but nothing is tuned for it.
- The user-level Codex configuration in `~/.codex/config.toml`: the model, the context limit, the sandbox and the approval policy stay the user's choice.
- An LSP tool for Codex.

## Impact

- New files: `.codex/hooks.json` and `.codex/config.toml`.
- `.agent-hooks/review_gate.py`: the docstring and comments only; no gate logic changes. `.agent-hooks/run.sh`: a comment. `.agent-hooks/review_gate_test.py`: a new test for the two configurations.
- `AGENTS.md`, `.agents/skills/run-app/SKILL.md` and `scripts/browser-mcp.sh`.
- No product code, API or schema changes, and nothing is **BREAKING**.
- The change answers to the root `AGENTS.md`. It touches no folder with its own module `AGENTS.md`.
- Codex needs the `claude` CLI installed and signed in, because the reviewers run there. Codex runs project hooks only after the user trusts them.
