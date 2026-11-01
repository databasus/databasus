## ADDED Requirements

### Requirement: Recorded violations form a finite shrinking baseline

The checks SHALL permit only individually recorded existing findings, each with an owner and a removal milestone. Baseline matching SHALL distinguish the rule, source, dependency or structural location, and occurrence count. The baseline MUST NOT exempt entire layers, slices or files. New findings, increased occurrence counts and stale entries SHALL fail verification. Unchanged violations SHALL remain recognizable after unrelated line movement.

#### Scenario: A new violation appears beside an accepted one

- **WHEN** a file containing a recorded violation adds another forbidden dependency or another occurrence of the same dependency
- **THEN** frontend lint fails

#### Scenario: A violation is repaired

- **WHEN** a recorded violation no longer exists but its baseline entry remains
- **THEN** frontend lint fails until the stale entry is removed

#### Scenario: An unrelated edit moves a violation

- **WHEN** an unchanged recorded dependency moves to a different line
- **THEN** the baseline still recognizes the same finding without authorizing any additional finding

## REMOVED Requirements

### Requirement: Existing violations form a finite shrinking baseline

**Reason**: Its scenario about nested domain groups no longer applies, because frontend lint treats every folder under a layer as one slice and no nested group remains.

**Migration**: "Recorded violations form a finite shrinking baseline" keeps the other three scenarios and the same rules for the baseline, without the nested groups.
