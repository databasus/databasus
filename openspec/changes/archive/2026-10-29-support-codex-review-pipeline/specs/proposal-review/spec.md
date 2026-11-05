## MODIFIED Requirements

### Requirement: The full proposal review starts once the proposal is written

The hook that runs when the agent finishes a turn SHALL start the full proposal review of an active change when all of these hold: the change has every artifact its schema requires, its task list has tasks and none is checked, the change's files changed during this turn, and their state differs from the last state the same review client and review profile accepted for this change. The turn SHALL be recognized by the change's files, not by the request text: whether the agent started through a command, a skill or a plain request MUST NOT matter. The accepted state SHALL be stored per review client, review profile and change, and SHALL remain reusable across sessions of the same review client and review profile. Archived changes are not reviewed.

#### Scenario: The agent writes every artifact after a plain request

- **WHEN** the user writes "write a spec for this", and the agent creates the proposal, design, delta specs and tasks and finishes the turn
- **THEN** the hook starts the full proposal review of that change

#### Scenario: Artifacts are not finished

- **WHEN** a change has a proposal but no tasks, and the agent finishes the turn
- **THEN** the full proposal review does not start and the quick checks run

#### Scenario: Implementation has started

- **WHEN** at least one task of the change is checked, and the agent edits a delta spec
- **THEN** the full proposal review does not start

#### Scenario: New session without edits

- **WHEN** the full proposal review accepted a proposal in one session, and in a new session the agent finishes a turn without touching the change's files
- **THEN** the full proposal review does not start

#### Scenario: Edit after acceptance

- **WHEN** after an accepted review the agent edits the design of a change whose tasks are not started
- **THEN** the full proposal review starts again

#### Scenario: Another client's acceptance does not suppress review

- **WHEN** one client's eligible proposal review sees files changed since its request snapshot that another client accepted
- **THEN** it starts its own full review because the stored acceptance belongs to a different review client

## ADDED Requirements

### Requirement: Proposal review stays within the selected client

Initial and repeat proposal reviews SHALL use the client selected by the review-hook-launch capability and the review profile for full reviews defined by the implementation-review capability. A complete proposal written through a command, a skill or a plain request SHALL receive the same strict OpenSpec validation, reviewer roles, finding treatment, repeat limits and summary behavior in either client. The selected client's CLI SHALL be sufficient; the other client MUST NOT be required. Proposal acceptance and findings SHALL follow the client and review profile isolation rules of the review-hook-launch capability.

#### Scenario: Codex writes a complete proposal

- **WHEN** Codex finishes a turn that makes an unstarted proposal eligible for full review
- **THEN** the existing verification-section format check and strict OpenSpec validation precede the existing parallel reviewer roles, all running in Codex with `high` reasoning effort and without starting Claude Code

#### Scenario: Claude Code writes a complete proposal

- **WHEN** Claude Code finishes a turn that makes an unstarted proposal eligible for full review
- **THEN** the existing verification-section format check and strict OpenSpec validation precede the existing parallel reviewer roles, all running in Claude Code on Opus with `medium` effort and without starting Codex

#### Scenario: A proposal fails validation

- **WHEN** either client finishes an eligible proposal whose strict OpenSpec validation fails
- **THEN** the turn is blocked with the validation reason and no AI reviewer starts

#### Scenario: Codex fixes proposal findings

- **WHEN** a full proposal review in Codex returns a blocking finding and Codex fixes the artifacts
- **THEN** the repeat runs in Codex with `high` reasoning effort over the previous findings and subsequent edits, and acceptance follows the existing summary behavior

#### Scenario: A proposal was reviewed by the other client

- **WHEN** an eligible proposal review starts in a different client from the one whose stored findings or acceptance exist
- **THEN** the selected client starts its own full review rather than reusing the other client's verdict or restricted repeat
