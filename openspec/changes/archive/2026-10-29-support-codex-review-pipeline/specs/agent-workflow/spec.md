## ADDED Requirements

### Requirement: Apply consultations stay within the active client

During apply the agent SHALL resolve routine questions itself. When a decision is both difficult and consequential, it SHALL seek a separate consultation directly through a read-only instance of its active client, consider the advice and make the decision. A Codex session SHALL consult Codex using its review profile for full reviews; a Claude Code session SHALL consult Claude Code using its review profile for full reviews. The consultation MUST NOT require the other client, edit files, start recursive review or replace the main agent's decision-making responsibility. The consultation SHALL use the read-only access and recursion restrictions of the review-hook-launch capability; it returns advisory text rather than a review verdict. A failed consultation SHALL be reported and MUST NOT silently switch clients or be presented as advice that was received. The agent SHALL then make the decision from available evidence, asking the user only when information, a decision or authorization only the user can supply is necessary.

#### Scenario: Codex needs a consequential consultation

- **WHEN** Codex encounters a difficult and consequential decision within an agreed apply scope
- **THEN** it seeks advice directly through a separate read-only Codex instance with `high` reasoning effort, considers the advice and makes the decision without requiring Claude Code

#### Scenario: Claude Code needs a consequential consultation

- **WHEN** Claude Code encounters a difficult and consequential decision within an agreed apply scope
- **THEN** it seeks advice directly through a separate read-only Claude Code instance on Opus with `medium` effort, considers the advice and makes the decision without requiring Codex

#### Scenario: The consultation cannot run

- **WHEN** the selected consultation instance fails or times out
- **THEN** the agent makes the decision from available evidence, reports that it received no consultation and does not switch clients
- **AND** it asks the user only if the decision needs information, a decision or authorization that only the user can supply
