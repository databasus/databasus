## Context

See proposal.md, "Why", for the motivation. This section covers only the current state that shapes the approach.

- **The hooks.** Claude Code wires the review hooks in `.claude/settings.json`: `session-start` on SessionStart, `snapshot` on UserPromptSubmit, the plan gates on `PreToolUse` for `ExitPlanMode`, and six gates on Stop (`diff`, `necessity`, `architecture`, `code-shape`, `implementation`, `proposal`). Every command resolves the repository through `CLAUDE_PROJECT_DIR` (`.claude/settings.json:14`). All of them start `.agent-hooks/run.sh`, which runs `review_gate.py` with the pinned Python and OpenSpec.
- **The reviewer.** It is always a separate `claude --print` process in plan permission mode, with hooks disabled and a read-only tool list (`.agent-hooks/review_gate.py:588`). The gate logic reads only common payload fields: `session_id`, `cwd`, `prompt`, `last_assistant_message` and `hook_event_name`.
- **The old Codex support.** The repository supported Codex until commit `c6c543ff`. That version started each gate with `--client codex` and reviewed a Codex session with a Codex reviewer (`codex exec` on `gpt-6.1-sol`).
- **The Codex hook format.** Codex reads project hooks from `.codex/hooks.json` with the same event names and the same output fields as Claude Code. A Stop hook that returns `"decision": "block"` continues the turn, and its `reason` becomes the next prompt. `"continue": false` with `stopReason` stops the turn and shows the reason as hook output. Hooks of one event start concurrently, and every command runs in the session's `cwd`. SessionStart also fires after a compaction, with the source `compact`. Codex loads a project's `.codex/` layer, hooks included, only when the project is trusted, and it runs a new or changed hook only after the user trusts it. `codex exec --dangerously-bypass-hook-trust` runs the hooks of one invocation without that trust. Codex spawns a hook command directly as the user's process in the session's `cwd`, outside its command sandbox (`codex-rs/hooks/src/engine/command_runner.rs` in the Codex source), so the hook's checks reach Docker and the reviewer reaches the network as they do under Claude Code.
- **What Codex reads by itself.** Codex discovers skills in `.agents/skills/` in every directory from its start directory up to the repository root. That is where the repository keeps every skill already; `.claude/skills` is a symlink to it (`AGENTS.md`, "Development environment"). Codex loads only the `AGENTS.md` files on the path from the repository root to its start directory. Claude Code loads a module's rules through the module's `CLAUDE.md`, for example `backend/CLAUDE.md` imports `backend/AGENTS.md`, and `agent/verification/CLAUDE.md` imports both `agent/verification/AGENTS.md` and `backend/AGENTS.md`.
- **The test build tag.** `.claude/settings.json` sets `GOFLAGS=-tags=testhelpers` for every process of a Claude Code session. The mise tasks and `backend/.golangci.yml` pass the tag themselves, so the hooks' checks do not depend on that variable. Only the agent's own `go` commands do (`AGENTS.md`, "Until the migration finishes").
- **The browser server.** `.mcp.json` registers `chrome-devtools` through `scripts/browser-mcp.sh` for Claude Code.

The change answers to the root `AGENTS.md` only: it touches no folder that has a module `AGENTS.md`, and `docs/architecture.md` does not cover the agent tooling.

## Goals / Non-Goals

**Goals:**

- Codex passes the same gates as Claude Code when it finishes a turn, with Claude Code reviewers.
- The gate logic does not branch on the client, so a fix to a gate reaches both clients.
- A Codex session gets the skills, the browser server, the test build tag and the module rules from the sources Claude Code uses.

**Non-Goals:**

- The Codex user configuration: the model, the context limit, the sandbox and the approval policy.
- A way to grant Codex trust to the project hooks from the repository. Trust is the user's decision, recorded in their configuration.

## Decisions

### 1. The reviewers stay in Claude Code, and the hook takes no client argument

`review_gate.py` keeps calling `claude` for every reviewer. `.codex/hooks.json` starts the same actions with the same arguments as `.claude/settings.json`, and no `--client` flag returns.

Alternatives:

- **Restore the `--client` switch and review Codex with Codex, as before `c6c543ff`.** The user wants Claude Code to review the work Codex does. Two reviewer models would also accept different things, so the result of a change would depend on the client that implemented it.
- **Take the `--client` switch from `cloud`'s hooks, where it lets a plan in a Codex reply pass through the plan gates.** Codex does not plan in this workflow, so the switch would only carry code with no use here.

### 2. Codex wires exactly the Stop gates of Claude Code, and a test holds them equal

`.codex/hooks.json` holds `session-start` on SessionStart, `snapshot` on UserPromptSubmit and the six Stop gates with the timeouts from `.claude/settings.json`: 600 seconds for the quick gates and 7200 for `implementation` and `proposal`. The plan gates are left out because Codex has no `ExitPlanMode`. A new test in `.agent-hooks/review_gate_test.py`, next to `FullReviewBudgetTest` (`.agent-hooks/review_gate_test.py:625`), reads both files. It checks that the SessionStart, UserPromptSubmit and Stop actions with their timeouts are the same. It also runs each Codex command through `bash -c` from a subdirectory of a test repository without `CLAUDE_PROJECT_DIR`, the way Codex starts it. `FullReviewBudgetTest` also checks the full review budget against the Codex timeouts.

Alternatives:

- **Generate one configuration from the other.** That adds a generator and a step to forget. The two files have different formats only in the command string, and the test catches a drift as cheaply.
- **Leave the parity to review.** A forgotten gate in one file would silently drop a review in one client.

### 3. The Codex command finds the repository from its working directory

The command is `"$(git rev-parse --show-toplevel)/.agent-hooks/run.sh" <action>`. Codex runs it in the session's `cwd`, which lies inside the repository or a worktree of it. Claude Code's commands keep `CLAUDE_PROJECT_DIR`.

Alternative: **a fixed relative path such as `.agent-hooks/run.sh`.** It breaks when Codex starts in a subdirectory.

### 4. A fix round needs no change to the snapshot

A Codex block reason becomes the next prompt, so UserPromptSubmit may fire for it. Every block reason starts with `AUTOMATIC REVIEW:`, and `snapshot` keeps the request's baseline for such a prompt (`.agent-hooks/review_gate.py:443`). If Codex does not fire the hook for a continuation, the baseline stays anyway. The review state lives in the Git directory and is keyed by session and by change, not by client, so a state accepted in one client stays accepted in the other.

Alternative: **mark the fix round from the Stop payload's `stop_hook_active`.** Claude Code and Codex both handle the round already through the prefix, and a second signal would have to agree with it.

### 5. A project `.codex/config.toml` carries the browser server, the build tag and the instruction size

- `[mcp_servers.chrome-devtools]` starts `scripts/browser-mcp.sh` with the command `.mcp.json` uses. It replaces the user's global `chrome-devtools` entry inside this repository. That entry connects to the user's own Chrome with `--autoConnect`, while the spec asks for a temporary profile per session.
- `shell_environment_policy.set.GOFLAGS = "-tags=testhelpers"` gives Codex's shell commands the tag, as the `env` of `.claude/settings.json` does for Claude Code.
- `project_doc_max_bytes = 65536`. Codex truncates its project instructions at this limit, and the root `AGENTS.md` already holds 25.7 KB of the default 32 KiB. With `frontend/AGENTS.md` on the path the total is 41.6 KB.

Alternatives:

- **Put these keys in `~/.codex/config.toml`.** They are facts of this repository, and another clone or another developer would not get them.
- **Use the user's global `chrome-devtools` entry.** It drives the user's own browser with their sign-ins and shares one profile between sessions, which the browser requirement forbids.

### 6. The root `AGENTS.md` makes the agent read a module's rules before an edit

A new sentence after the table of per-folder docs says: before changing a file under one of these folders, read its `AGENTS.md` unless it is already in context, and for `agent/verification/` also `backend/AGENTS.md`. The rule binds every agent and costs Claude Code nothing, because its `CLAUDE.md` imports have already put the file in context.

Alternatives:

- **Start Codex in the module folder.** One change often spans `backend/` and `frontend/`.
- **Inline the module docs into the root `AGENTS.md`.** Every session would load every module's rules, and the root file would pass the size limit.
- **A `CLAUDE.md`-style import for Codex.** Codex has no import syntax for `AGENTS.md`.

### 7. The skills need no new link

Codex reads `.agents/skills/` directly, so the directory stays the single source. The documents that name `.claude/skills` as the way in add that Codex reads the directory itself. The `openspec-*` skills stay generated for Claude Code: Codex reads the same files, and OpenSpec's Codex output writes prompts to the user's `~/.codex/prompts`, outside the repository.

Alternative: **a `.codex/skills` symlink.** Codex does not read that path.

### 8. Documents name both clients where both apply

The following documents name both clients where the behavior is shared:

- `AGENTS.md`: "Mandatory communication skill", "Development environment", "Code search and navigation", "Until the migration finishes" and "Agent memory";
- the docstring and the reviewer comment of `review_gate.py` (`.agent-hooks/review_gate.py:2`, `:45`, `:105`);
- the comment of `.agent-hooks/run.sh`;
- the `run-app` skill (`.agents/skills/run-app/SKILL.md:8`);
- the header of `scripts/browser-mcp.sh`.

"Code search and navigation" keeps the LSP tool as the first choice. For a client without it, the section names the `gopls` command line and `pnpm --dir frontend typecheck`. "Agent memory" names Codex's memory next to `~/.claude/projects/*/memory`. The docstring of `review_gate.py` also says that Codex runs the hooks only after the user trusts them, and that `claude` has to be installed and signed in for a Codex session. A person learns how the hooks work there, per the documentation table of `AGENTS.md`.

Alternative: **a separate Codex guide.** It would retell the hook docstring and `AGENTS.md`, against the rule that one rule lives in one place.

## Risks / Trade-offs

- [Codex records hook trust per file path, so every new worktree asks for trust again, and an untrusted hook is skipped without a review. Codex may also treat a worktree under `.claude/worktrees/` as a project the user has not trusted, and then it loads neither the hooks nor `.codex/config.toml`.] → Task 3.2 asks Codex, from the change's own worktree, which command starts `chrome-devtools`, which shows whether Codex loads the project layer there without starting a model. The docstring of `review_gate.py` tells the user to trust the project and its hooks when Codex asks, and states what task 3.2 found about worktrees.
- [In Codex 0.164.0 a command-line project trust override does not make an untrusted clone load its project configuration.] → The pilot temporarily registers its clone as trusted in the user's Codex configuration and removes only that entry afterwards. The main project's trust and the user's other settings remain intact. Mise trust stays as requested by the user.
- The main checkout loads the project browser launcher with and without a command-line trust override. The existing linked worktree `.claude/worktrees/logical-backups-module` starts Codex without a project trust prompt or its own trust entry. It has no `.codex/` configuration, so `codex mcp get chrome-devtools` lists the global launcher there; a local `config/read` query confirms that only user and system layers are present. This distinguishes inherited repository trust from the configuration files that each checkout must contain. These checks changed no files in that worktree.
- [A project `mcp_servers` entry does not replace the user's global entry of the same name.] → Task 3.2 shows which entry wins. If the global entry wins, the project entry takes another name, and the `run-app` skill names it.
- [After the repeat limit the user does not see the remaining findings in Codex.] → Codex turns `stopReason` into a hook output entry of the kind `Stop` (`codex-rs/hooks/src/events/stop.rs`), and the hook sets `systemMessage`, which Codex shows as a warning. The gate logic stays as it is.
- [The Codex hook format changes in a later Codex release.] → The parity test covers only the configuration. A format change surfaces as a hook that does not block, and the pilot of task 3.1 is the check to repeat after a Codex upgrade.
- [The pilot costs a Codex run and a few quick Claude reviews, and a blocked turn makes Codex act on the findings.] → The pilot runs in a temporary clone that holds the change's diff, so whatever Codex edits there never reaches the worktree. It writes one requested scratch file, so only the quick gates run, never a full review.

## Migration Plan

Nothing migrates. The new files take effect when the user starts Codex in the repository and trusts its hooks. Removing `.codex/` turns the setup off, and Claude Code is unaffected either way.
