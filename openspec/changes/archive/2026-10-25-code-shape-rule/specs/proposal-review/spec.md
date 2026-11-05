## MODIFIED Requirements

### Requirement: Quick checks yield to the full proposal review
When only files of changes that have every artifact and no checked task changed during the turn, the `necessity`, `architecture`, `diff` and `code-shape` checks SHALL skip the turn. This way the quick checks duplicate neither the review turn nor the summary turn after acceptance. When other files changed too, the quick checks SHALL run as usual.

#### Scenario: A turn with only artifacts
- **WHEN** the agent finished every artifact of a change and changed nothing else
- **THEN** the quick checks skip the turn and only the full proposal review runs

#### Scenario: An unfinished proposal
- **WHEN** the agent wrote only the proposal and finishes the turn with a question to the user
- **THEN** the full proposal review does not start, and the quick necessity check reviews the proposal
