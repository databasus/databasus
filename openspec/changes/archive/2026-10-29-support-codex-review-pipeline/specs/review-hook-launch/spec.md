## MODIFIED Requirements

### Requirement: The environment check reports every tool the hooks need

The repository SHALL have a command, `mise run doctor [--client <codex|claude>]`, that reports the selected review client and, one line per tool, whether the hooks can find Python, git, OpenSpec and the selected client's CLI. It SHALL use the client-selection rules of this capability and MUST NOT require the other client's CLI. The command SHALL run through the same launcher as the hooks, and SHALL exit with a non-zero code when a required tool is missing or the client cannot be selected. The version manager runs that command, so when the version manager itself cannot be found, the launcher run directly with the doctor action SHALL report that and exit with a non-zero code. Finding a CLI SHALL NOT be reported as proof that its authentication or model access works.

#### Scenario: Every tool is present

- **WHEN** the user runs the environment check with a selected client and every required tool installed
- **THEN** the selected client and each required tool's path are reported and the command exits with code 0 without claiming that the client is authenticated or can reach its model

#### Scenario: Only Codex is installed

- **WHEN** the environment check selects Codex and Python, git, OpenSpec and Codex are available but Claude Code is absent
- **THEN** the command exits with code 0 without requiring Claude Code

#### Scenario: Only Claude Code is installed

- **WHEN** the environment check selects Claude Code and Python, git, OpenSpec and Claude Code are available but Codex is absent
- **THEN** the command exits with code 0 without requiring Codex

#### Scenario: No reviewer CLI

- **WHEN** the selected client's CLI is absent and the other client's CLI is installed
- **THEN** the command names the missing selected CLI, exits with a non-zero code and does not switch clients

#### Scenario: No version manager

- **WHEN** the version manager cannot be found and the user starts the launcher's doctor action directly
- **THEN** the command reports that the version manager was not found and exits with a non-zero code

### Requirement: Codex runs the same review hooks as Claude Code

Codex SHALL run the repository's review hooks from its own project configuration: the session start hook, the request snapshot and every gate that Claude Code runs when a turn finishes, each with the same timeout as in the Claude Code configuration. Codex session reviewers SHALL run in Codex; Claude Code session reviewers SHALL run in Claude Code. They SHALL use the client-specific review profiles defined by the full review model requirement of the implementation-review capability. Findings SHALL reach either agent through the same repair cycle: the hook blocks the turn, the agent resolves the findings and finishes the turn again, and review repeats until acceptance or its repeat limit. Request snapshots, findings and accepted states SHALL follow the review-state isolation requirement of this capability. The plan gate SHALL run only in Claude Code, because only Claude Code submits a plan through the intercepted plan tool. The repository's hook tests SHALL fail when a gate or timeout of a finished turn differs between the two configurations.

#### Scenario: Codex implements the last task

- **WHEN** Codex checks the last task of a change and finishes the turn
- **THEN** the full implementation review runs in Codex without starting Claude Code
- **AND** a blocking finding reaches Codex, which fixes it and finishes the turn again for review over the previous findings

#### Scenario: Claude Code implements the last task

- **WHEN** Claude Code checks the last task of a change and finishes the turn
- **THEN** the full implementation review runs in Claude Code without starting Codex

#### Scenario: A fix round in Codex

- **WHEN** a hook blocks a Codex turn and Codex continues with the review text as its next prompt
- **THEN** the gates keep comparing the working tree against the state at the user's request, not against the state at the start of the fix round

#### Scenario: A state accepted by the same client and review profile

- **WHEN** a session's request snapshot predates edits that another session of the same client and review profile made and accepted, and the first session finishes with a working tree matching that accepted state but differing from its request snapshot
- **THEN** no full review starts because the stored acceptance for that client and review profile matches the current state

#### Scenario: A state accepted in Claude Code

- **WHEN** Claude Code accepted a change and Codex finishes a turn in the same working copy without changing files
- **THEN** no full review starts because the turn has no edits, and the Claude Code acceptance is not imported into Codex state

#### Scenario: A Codex session starts or compacts its context

- **WHEN** a Codex session starts or compacts its running context
- **THEN** the agent is told to load the communication skill before its first action

#### Scenario: A plan in a Codex reply

- **WHEN** Codex shows a plan in its reply and changes no file
- **THEN** no plan gate runs

#### Scenario: The configurations drift apart

- **WHEN** a finished-turn gate or timeout changes in only one client configuration
- **THEN** the hook tests fail and name the differing gate

## ADDED Requirements

### Requirement: Review client selection is explicit and deterministic

Hook invocations SHALL declare the client that owns their hook configuration. Agent-launched manual review and environment checks SHALL select the active client without a user choosing it again. Terminal invocations SHALL accept `--client codex` or `--client claude`, with the explicit option taking precedence over the agent environment. With no explicit option, the agent's declared client SHALL be used. With neither, or with an invalid selected value, the command SHALL fail with instructions for selecting a client before starting a reviewer. The system MUST NOT infer the client from the available executables, start the other client after an error, or change clients during a repair cycle.

#### Scenario: A Codex hook inherits a Claude declaration

- **WHEN** a hook declared by the Codex configuration starts in an environment that otherwise declares Claude Code
- **THEN** the hook selects Codex

#### Scenario: A manual run from an agent

- **WHEN** either agent runs manual review or the environment check without an explicit client option
- **THEN** the command selects that agent's client

#### Scenario: An explicit terminal override

- **WHEN** the environment declares Codex and the terminal command explicitly selects Claude Code
- **THEN** the command selects Claude Code

#### Scenario: A terminal without agent context

- **WHEN** no explicit option or agent declaration selects a client
- **THEN** the command fails before running a reviewer and explains both accepted client options

#### Scenario: Invalid client

- **WHEN** the selected client value is neither `codex` nor `claude`
- **THEN** the command rejects the value before starting a reviewer

#### Scenario: Selected client fails

- **WHEN** the selected reviewer CLI is missing, unauthenticated, unable to access its model, or exits with an error
- **THEN** the review is not accepted, the agent receives the cause and the other client is not started

### Requirement: Reviewer execution cannot modify the work or recursively review itself

Every reviewer SHALL run in a separate noninteractive process with read-only repository access and without tools that can modify external systems. Its prompt SHALL carry the same role and project rules regardless of client. Reviewer processes MUST NOT start the repository's review cycle for their own work, replace the initiating request snapshot or write persistent reviewer conversations. The caller SHALL validate a structured final result against the selected review contract; progress messages MUST NOT be interpreted as the final verdict. A missing, malformed or schema-invalid result, a process failure or timeout SHALL leave the review unaccepted and pass the failure reason to the agent. Temporary output artifacts SHALL be private to one invocation and removed after success or failure. The child restrictions MUST NOT disable the main agent's hooks or change its permissions.

#### Scenario: Codex returns progress before its final result

- **WHEN** the Codex reviewer emits progress and then a valid structured final answer
- **THEN** only the final answer determines the review result

#### Scenario: Reviewer emits no usable final answer

- **WHEN** a reviewer exits successfully but omits its final result or returns a value that violates the review schema
- **THEN** the review is not accepted and the agent receives a result-format error

#### Scenario: A reviewer reaches its time limit

- **WHEN** a reviewer exceeds its allowed execution time
- **THEN** it is terminated, its temporary output is removed and the review remains unaccepted with a timeout reason

#### Scenario: A reviewer loads the repository configuration

- **WHEN** a reviewer starts in the repository that registered the main agent's hooks and external tools
- **THEN** the reviewer cannot edit repository files, call tools that modify external systems or start another review cycle
- **AND** the initiating request snapshot and main agent hook configuration remain unchanged

### Requirement: Review state belongs to a client and review profile

Request snapshots and per-request gate state SHALL remain keyed by each session's unique identifier. Full-review findings and acceptance SHALL be separated by client, review profile and change, and remain reusable across sessions of the same client and review profile. The review profile SHALL distinguish model selection, reasoning effort and the review contract. A state from a different client or review profile, or a state without that identity, MUST NOT be used to accept a review or restrict it to previous findings. Selecting another client SHALL NOT by itself start a review in a turn without edits; the existing trigger conditions continue to apply. A manual review SHALL use and update the same state as automatic review for its selected client and review profile.

#### Scenario: Acceptance belongs to another client

- **WHEN** Claude Code accepted a change and Codex later runs an eligible automatic review or an explicit manual review
- **THEN** Codex starts its own full review and does not treat the Claude Code acceptance as its own

#### Scenario: Findings belong to another review profile

- **WHEN** a review starts with a different model selection, effort or review contract from the state that contains findings
- **THEN** it starts a full review rather than a repeat restricted to those findings

#### Scenario: Existing state has no client identity

- **WHEN** an eligible review finds a stored acceptance or findings written without client and review profile identity
- **THEN** it starts a full review without importing that state

#### Scenario: Manual findings reach the next automatic review

- **WHEN** a manual review stores findings and the same client and review profile later finishes fixes
- **THEN** the automatic review checks those findings and the edits made after that review
