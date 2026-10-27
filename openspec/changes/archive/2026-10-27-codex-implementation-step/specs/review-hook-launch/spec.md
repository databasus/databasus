## ADDED Requirements

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
