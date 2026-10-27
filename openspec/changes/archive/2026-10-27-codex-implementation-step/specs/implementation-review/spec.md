## MODIFIED Requirements

### Requirement: Full review models
The full review reviewers SHALL run on Opus with `medium` effort. The quick checks SHALL run on Opus with `low` effort. The reviewers SHALL run in Claude Code whichever client the agent works in, so a change implemented in Codex gets the same review as one implemented in Claude Code.

#### Scenario: Full review in Claude Code
- **WHEN** the full implementation review starts from Claude Code
- **THEN** the reviewers run on Opus with `medium` effort

#### Scenario: Quick check in Claude Code
- **WHEN** the diff check starts from Claude Code
- **THEN** the reviewer runs on Opus with `low` effort

#### Scenario: Full review in Codex
- **WHEN** the full implementation review starts from Codex
- **THEN** the reviewers run in Claude Code on Opus with `medium` effort

#### Scenario: Quick check in Codex
- **WHEN** the diff check starts from Codex
- **THEN** the reviewer runs in Claude Code on Opus with `low` effort
