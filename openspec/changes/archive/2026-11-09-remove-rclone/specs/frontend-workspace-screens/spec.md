## MODIFIED Requirements

### Requirement: Resource operations retain provider and access policy

All supported storage and notifier provider variants, validation, secret omission, saved-type restrictions, connection or delivery tests, deletion conflicts, transfers and storage migration progress SHALL remain available under their existing permissions. A refusal SHALL retain the last acknowledged resource and provide translated correction or recovery. An accepted transfer or migration SHALL continue to show its actual progress rather than imply immediate completion.

#### Scenario: A resource in use cannot be deleted

- **WHEN** deleting a storage or notifier is refused because another resource uses it
- **THEN** the item and selection remain available and the relevant translated conflict is shown

#### Scenario: A viewer opens a resource

- **WHEN** a workspace viewer opens storage or notifier details
- **THEN** existing read and saved-test capabilities remain available and prohibited mutation controls are unavailable
