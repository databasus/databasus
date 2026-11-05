## MODIFIED Requirements

### Requirement: Shared audit presentation preserves each view's meaning

Instance, user and workspace audit views SHALL retain their existing access checks, filters, newest-first paging, page sizes, author/workspace visibility, date/time conventions and distinct empty messages. Event content and captured author/workspace names SHALL be displayed literally. Changing the selected user or workspace, or leaving the view, SHALL invalidate its old loads. A common presentation SHALL not broaden which events the caller may fetch. Workspace audit SHALL retain author visibility, no redundant workspace column, short timestamps, its ten-event page size and its own scroll container.

#### Scenario: An administrator opens a user's audit drawer

- **WHEN** an administrator opens one user's audit history and then another's
- **THEN** each view requests that user's events and a late response from the first cannot replace the second
- **AND** paging, column selection and the user-specific empty message retain their meaning

#### Scenario: An event contains technical or user-entered characters

- **WHEN** an event message or captured label contains angle brackets or interpolation-like text
- **THEN** it is displayed literally without executing markup or translating the event's contents

#### Scenario: A user opens a workspace's audit view

- **WHEN** a user opens audit history for the current workspace
- **THEN** only that workspace's authorized events load in the existing order and page size
- **AND** desktop rows and mobile cards retain author visibility, short timestamps and workspace-specific empty copy

#### Scenario: Workspace changes while audit is loading

- **WHEN** an earlier workspace's audit request completes after a workspace switch
- **THEN** its events, errors and paging state cannot change the current view
