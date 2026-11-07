## ADDED Requirements

### Requirement: Storage and notifier lists offer consistent name search

Storage and notifier lists SHALL offer local, case-insensitive substring search by resource name when the loaded collection contains at least five items. The threshold SHALL use the full collection count, not the number of search matches. A list with fewer than five items and no active query SHALL omit the search control. An active query SHALL remain visible and clearable if a collection update reduces the count below five. Search SHALL filter list cards without changing the underlying collection, remembered selection or open detail/editor. Changing or clearing the query SHALL NOT issue a network request. A query with no matches SHALL show translated no-results feedback containing the literal query; clearing it SHALL restore all loaded cards. Search controls and persistent feedback SHALL follow the active language without resetting the query or repeating requests.

#### Scenario: A user searches storages by name

- **WHEN** a workspace has at least five loaded storages and the user enters part of a storage name with different letter casing
- **THEN** only matching storage cards are shown without another list request
- **AND** the search control remains available even if fewer than five cards match

#### Scenario: A user searches notifiers by name

- **WHEN** a workspace has at least five loaded notifiers and the user enters a name substring
- **THEN** matching cards appear under the same rules as storage search

#### Scenario: Search has no matches

- **WHEN** a query matches no storage or notifier names
- **THEN** translated no-results feedback shows the query as text
- **AND** clearing the query restores the loaded list

#### Scenario: Filtering hides the selected resource

- **WHEN** a query excludes the currently selected resource's card
- **THEN** its selection and open details or editing draft remain intact

#### Scenario: A small collection needs no search

- **WHEN** a storage or notifier list contains fewer than five loaded items and has no active query
- **THEN** it shows the cards without a search control

#### Scenario: A collection shrinks during search

- **WHEN** a collection update leaves fewer than five resources while a query is active
- **THEN** the query remains visible and can be cleared to reveal all remaining resources

#### Scenario: Language changes during search

- **WHEN** the user switches interface language with an active query and no-results feedback
- **THEN** the search label and feedback use the new language while the query, selection and draft remain unchanged
- **AND** no additional request is sent

### Requirement: Resource collection refresh preserves each screen's policy

Notifier lists SHALL continue refreshing in the background every five minutes while mounted. Storage lists SHALL NOT introduce periodic refresh. Both lists SHALL retain initial loading, explicit read retry and reconciliation after acknowledged creation, editing, deletion or transfer. A background read SHALL preserve an active editor's newer input. Leaving a list's workspace or session or unmounting it SHALL prevent obsolete read completions from changing visible items, selection, feedback or pending state, and SHALL stop its periodic refresh.

#### Scenario: Notifier polling refreshes the current collection

- **WHEN** five minutes elapse with the notifier list mounted in the current workspace and session
- **THEN** its collection refreshes without discarding an active editor's newer input

#### Scenario: An idle storage list stays event-driven

- **WHEN** five minutes elapse with the storage list mounted and no user action or resource mutation
- **THEN** no periodic storage list request is made

#### Scenario: Leaving a notifier list stops its polling

- **WHEN** a notifier list is unmounted and time advances beyond its refresh interval
- **THEN** the departed list issues no further polling requests

#### Scenario: An obsolete collection read completes

- **WHEN** a list read succeeds or fails after the user leaves its workspace or session or a newer list read supersedes it
- **THEN** it cannot replace current items, change selection, publish feedback or alter the current loading state
