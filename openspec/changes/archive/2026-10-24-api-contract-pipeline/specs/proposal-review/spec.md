## MODIFIED Requirements

### Requirement: OpenSpec validation runs first
Before starting reviewers the hook SHALL check the format of the change's "Change verification" section by the rules the full implementation review applies to it, without running its commands, and then run strict OpenSpec validation of the change. When the section's format is wrong or validation fails, the hook SHALL block the turn and pass the agent the reason or the end of the validation output. Reviewers MUST NOT start in that case.

#### Scenario: A requirement without a scenario
- **WHEN** a delta spec has a requirement without a scenario and validation fails
- **THEN** the turn is blocked, the agent gets the validation output, and no reviewer starts

#### Scenario: A forbidden command in the verification section
- **WHEN** the change's "Change verification" section contains `mise run test`
- **THEN** the turn is blocked with the reason and the list of allowed forms, the command does not run, and no reviewer starts
