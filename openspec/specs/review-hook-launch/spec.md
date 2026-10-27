# review-hook-launch Specification

## Purpose

Makes the review hooks start the same way in every environment and refuse to stay silent: a hook that cannot run stops the agent with a reason, so a review is never skipped without the user knowing.

## Requirements

### Requirement: Hooks run with the repository's pinned tools
Every review hook SHALL start through one launcher that runs the hook with the Python and OpenSpec versions pinned in the repository's tool versions file. The launcher SHALL find the version manager even when the client was started outside a login shell, for example from an IDE. The launcher MUST NOT install a missing tool itself, because an installation would not fit the hook timeout.

#### Scenario: A client started from an IDE
- **WHEN** Claude Code was started from an IDE whose environment does not activate the version manager, and the agent finishes a turn
- **THEN** the hooks run with the pinned Python and OpenSpec versions

#### Scenario: A pinned version is not installed
- **WHEN** the version manager is present but the pinned OpenSpec version is not installed, and the agent finishes a turn
- **THEN** no installation starts, the agent stops, and the user sees that the hooks did not start, that `mise install` installs the pinned tools and that `mise run doctor` checks the environment

### Requirement: A hook that cannot run stops the agent
When the launcher cannot find the version manager, when the hook exits with an error, or when a tool a gate needs is missing, the hook SHALL stop the agent and show the user the reason and the command that checks the environment. The hook MUST NOT let the turn pass as if the review had run.

#### Scenario: OpenSpec is missing
- **WHEN** the OpenSpec CLI cannot be found and a turn finishes that wrote the artifacts of a change
- **THEN** the agent stops, and the user sees that the hooks could not find `openspec` and that `mise run doctor` checks the environment

#### Scenario: The version manager is missing
- **WHEN** the launcher cannot find the version manager
- **THEN** the agent stops, and the user sees that the version manager was not found, that it has to be installed and that `mise run doctor` then checks the environment

#### Scenario: The hook crashes
- **WHEN** the hook process exits with a non-zero code
- **THEN** the agent stops, and the user sees which hook action failed, its exit code, and that `mise run doctor` checks the environment

### Requirement: The environment check reports every tool the hooks need
The repository SHALL have a command, `mise run doctor`, that reports, one line per tool, whether the hooks can find Python, git, OpenSpec and the Claude Code CLI that runs the reviewers. The command SHALL run through the same launcher as the hooks, and SHALL exit with a non-zero code when any of them is missing. The version manager runs that command, so when the version manager itself cannot be found, the launcher run directly as `.agent-hooks/run.sh doctor` SHALL report that and exit with a non-zero code.

#### Scenario: Every tool is present
- **WHEN** the user runs `mise run doctor` with every tool installed
- **THEN** each tool is reported with its path and the command exits with code 0

#### Scenario: No reviewer CLI
- **WHEN** the Claude Code CLI is not installed and the user runs `mise run doctor`
- **THEN** the command reports `claude` as not found and exits with a non-zero code

#### Scenario: No version manager
- **WHEN** the version manager cannot be found and the user runs `.agent-hooks/run.sh doctor`
- **THEN** the command reports that the version manager was not found and exits with a non-zero code

### Requirement: Codex runs the same review hooks as Claude Code
Codex SHALL run the repository's review hooks from its own project configuration: the session start hook, the request snapshot and every gate that Claude Code runs when a turn finishes, each with the same timeout as in the Claude Code configuration. The reviewers of a Codex session SHALL run in Claude Code, with the same read-only permissions, models and effort as the reviewers of a Claude Code session. Their findings SHALL reach Codex the way they reach Claude Code: the hook blocks the turn, Codex resolves the findings and finishes the turn again, and the review repeats until it accepts the result or reaches its repeat limit. The review state SHALL be shared by both clients working in one working copy. The plan gate SHALL run only in Claude Code, because only Claude Code submits a plan through a tool the hook can intercept. The repository's hook tests SHALL fail when a gate or a timeout of a finished turn differs between the two configurations.

#### Scenario: Codex implements the last task
- **WHEN** Codex checks the last task of a change and finishes the turn
- **THEN** the full implementation review runs with reviewers in Claude Code
- **AND** when a reviewer returns a blocking finding, Codex gets the findings, fixes them and finishes the turn again, and the review repeats over the previous findings

#### Scenario: A fix round in Codex
- **WHEN** the hook blocks a Codex turn and Codex continues with the review text as its next prompt
- **THEN** the gates keep comparing the working tree against the state at the user's request, not against the state at the start of the fix round

#### Scenario: A state accepted in Claude Code
- **WHEN** the full implementation review accepted a change in a Claude Code session, and Codex finishes a turn in the same working copy without changing the working tree
- **THEN** the full implementation review does not start again

#### Scenario: A Codex session starts or compacts its context
- **WHEN** a Codex session starts, or Codex compacts the context of a running session
- **THEN** the agent is told to load the communication skill before its first action

#### Scenario: A plan in a Codex reply
- **WHEN** Codex shows a plan in its reply and changes no file
- **THEN** no plan gate runs

#### Scenario: The configurations drift apart
- **WHEN** a gate or its timeout is added to or changed in the Claude Code configuration of a finished turn and not in the Codex configuration
- **THEN** the hook tests fail and name the gate
