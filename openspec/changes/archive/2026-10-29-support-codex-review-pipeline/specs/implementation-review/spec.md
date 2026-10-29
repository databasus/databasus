## MODIFIED Requirements

### Requirement: The full implementation review starts after a change is implemented

The hook that runs when the agent finishes a turn SHALL start the full implementation review when all of these hold: every task of an active change is checked, the working tree changed during this turn, and its state differs from the last state the same review client and review profile accepted for this change. The accepted state SHALL be stored per review client, review profile and change, and SHALL remain reusable across sessions of the same review client and review profile. The full implementation review MUST NOT start while a task is unchecked, in a turn that left the working tree unchanged, or in a turn whose request carries an OpenSpec planning command. Archived changes are not reviewed.

#### Scenario: The agent checks the last task

- **WHEN** the agent checks the last task of a change and finishes the turn
- **THEN** the hook starts the full implementation review of that change

#### Scenario: Tasks remain

- **WHEN** a task of the change is unchecked and the agent finishes the turn
- **THEN** the full implementation review does not start and only the quick checks run

#### Scenario: A turn without changes

- **WHEN** every task of the change is checked but the working tree did not change during the turn
- **THEN** the full implementation review does not start

#### Scenario: New session after acceptance

- **WHEN** the full implementation review accepted a change in one session, and another session of the same review client and review profile finishes with a working tree matching that accepted state but differing from its own request snapshot
- **THEN** the full implementation review does not start again because the stored acceptance matches the current state

#### Scenario: Edit after acceptance

- **WHEN** after an accepted review the agent changes files and finishes the turn
- **THEN** the full implementation review starts again

#### Scenario: Another client's acceptance does not suppress review

- **WHEN** one client's eligible implementation review sees a working tree changed since its request snapshot that another client accepted
- **THEN** it starts its own full review because the stored acceptance belongs to a different review client

### Requirement: Full review models

The review client SHALL be selected by the review-hook-launch capability. In Claude Code, full implementation and proposal reviewers SHALL run on Opus with `medium` effort, and quick checks SHALL run on Opus with `low` effort. In Codex, full implementation and proposal reviewers SHALL use the developer's configured Codex model with `high` reasoning effort, and quick checks SHALL use that model with `medium` reasoning effort. Full-review repeats SHALL use the same client and review profile for full reviews as their initial review. Quick checks include the plan, necessity, diff, architecture and code-shape gates wherever the client supports that gate. Review tooling MUST NOT require the other client to be installed or authenticated.

#### Scenario: Full review in Claude Code

- **WHEN** the full implementation review starts from Claude Code
- **THEN** its reviewers run in Claude Code on Opus with `medium` effort

#### Scenario: Quick check in Claude Code

- **WHEN** a supported quick check starts from Claude Code
- **THEN** its reviewer runs in Claude Code on Opus with `low` effort

#### Scenario: Full review in Codex

- **WHEN** the full implementation review starts from Codex
- **THEN** its reviewers run in Codex on the configured model with `high` reasoning effort

#### Scenario: Quick check in Codex

- **WHEN** a supported quick check starts from Codex
- **THEN** its reviewer runs in Codex on the configured model with `medium` reasoning effort

#### Scenario: A Codex implementation needs fixes

- **WHEN** a full implementation review in Codex returns a blocking finding and the agent fixes it
- **THEN** the repeat review runs in Codex with `high` reasoning effort and the existing repeat limit and summary behavior apply

### Requirement: The whole review runs by hand

The repository SHALL have a command, `mise run review [<change>] [--client <codex|claude>]`, that runs the review a finished change gets from the hooks without a hook event, as a fallback when hooks did not run or another review is needed. The client SHALL be selected by the review-hook-launch capability. It SHALL run only AI reviewers: the quick checks `diff`, `architecture` and `code-shape` over the working tree, `code-shape` only when a code file changed, and the full implementation review of the named change, or of every active change whose tasks are all checked. Quick and full reviewers SHALL share one parallel pool with at most six reviewers running at a time; full reviewers MUST NOT wait for all quick reviewers to finish. The command MUST NOT run linting, tests or builds. Verification is the implementing agent's preceding step, and manual review is not a routine extra step after the automatic hooks. The full review SHALL run even when the selected client and review profile accepted this state. When an earlier review stored findings for the same client, review profile and change, the full review SHALL be the repeat over those findings, as in the hook; otherwise it SHALL start from scratch. The command SHALL read no hook payload, report the selected client, print every finding and a closing verdict, and exit with code 0 when everything accepted the change, 1 when something did not or a reviewer could not run, and 2 when there is nothing to review or client selection is invalid. It SHALL store the full review's outcome per change and selected review profile like the hook does, so the next turn of that agent repeats review over those findings. The command SHALL run through the launcher of the hooks; when the launcher cannot start it, the command SHALL print the reason and exit with a non-zero code instead of answering with a stop message for the agent.

#### Scenario: Everything accepts

- **WHEN** the user runs manual review with an implemented change and every reviewer accepts
- **THEN** the command reports its client, prints that review accepted the change in its current state and exits with code 0

#### Scenario: Parallel manual review

- **WHEN** manual review includes quick reviewers and full implementation reviewers
- **THEN** both kinds run concurrently in the same pool without a separate quick-review phase

#### Scenario: Verification already ran

- **WHEN** the implementing agent completed linting and tests and starts manual review because a hook did not run
- **THEN** the command starts AI reviewers without repeating linting, tests or builds

#### Scenario: A finding to fix

- **WHEN** a manual reviewer returns a `bug` finding and the agent then fixes it and finishes its turn
- **THEN** the command prints the finding and exits with code 1, and the same client's automatic review repeats over the finding

#### Scenario: A run after the fixes

- **WHEN** manual review stored findings, the agent fixed them and manual review runs again with the same client and review profile
- **THEN** the full review repeats over those findings and the files changed since

#### Scenario: Nothing to review

- **WHEN** no active change has every task checked, or the named change does not exist
- **THEN** the command says so, runs no reviewer and exits with code 2

#### Scenario: A Codex-only manual run

- **WHEN** Codex runs manual review without an explicit client option and Claude Code is not installed
- **THEN** all quick and full reviewers run in Codex with their corresponding review profiles

#### Scenario: A Claude-only manual run

- **WHEN** Claude Code runs manual review without an explicit client option and Codex is not installed
- **THEN** all quick and full reviewers run in Claude Code with their corresponding review profiles

#### Scenario: Selected reviewer cannot run

- **WHEN** a selected reviewer crashes, cannot authenticate or times out during manual review
- **THEN** the command reports the reason, exits with code 1 and does not start the other client

#### Scenario: Manual review has no valid client selection

- **WHEN** manual review runs without a client option or agent declaration, or with an invalid selected client value
- **THEN** it explains the accepted client options, starts no reviewer and exits with code 2
