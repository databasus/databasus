## ADDED Requirements

### Requirement: Automatic list refresh follows browser-tab visibility

The logical-backup, physical-backup, restore, verification and verification-agent lists SHALL send no automatic refresh request while the browser tab is hidden. When it becomes visible, a ready list SHALL request its first page immediately without waiting for its next polling tick, unless a refresh is already pending. Subsequent automatic refreshes SHALL use the list's existing interval. Initial loads, explicit reloads and refreshes requested by user actions SHALL remain independent of visibility. A request sent before the tab became hidden SHALL be allowed to finish under the existing context and lifecycle rules. Returning to the tab SHALL preserve loaded rows until the refreshed page is accepted under the existing merge rules. A failed first load SHALL still require an explicit retry.

#### Scenario: A ready list stays in a hidden browser tab

- **WHEN** a ready list's browser tab becomes hidden and several polling intervals pass
- **THEN** no automatic refresh request is sent
- **AND** the loaded rows remain available

#### Scenario: The browser tab becomes visible between ticks

- **WHEN** a ready list's hidden browser tab becomes visible before its next polling tick
- **THEN** its first page is requested immediately
- **AND** later automatic refreshes use the existing interval

#### Scenario: Returning while a refresh is pending

- **WHEN** the browser tab becomes visible while a refresh sent before it was hidden is still pending
- **THEN** no competing refresh request is started
- **AND** the pending response remains subject to the current list's context and lifecycle rules

#### Scenario: A list first mounts in a hidden browser tab

- **WHEN** a list first mounts while the browser tab is hidden
- **THEN** its initial page loads normally
- **AND** no automatic refresh is sent until the tab becomes visible

#### Scenario: Returning after an initial load failed

- **WHEN** a list's first load failed and its browser tab becomes visible
- **THEN** no automatic retry is sent
- **AND** the failure and explicit retry remain available

#### Scenario: An explicit reload starts while hidden

- **WHEN** an explicit reload is requested while the browser tab is hidden
- **THEN** the first page loads under the existing reload rules
- **AND** subsequent automatic refreshes remain paused until the tab becomes visible

### Requirement: An unchanged list refresh publishes no change

A successful refresh of a polled list SHALL publish no list-state change when the merged result preserves all row values and their order, the effective continuation cursor, page metadata and loading state. Changed values SHALL still be accepted even when row identities are unchanged. This suppression SHALL preserve the existing rules for concurrent next-page requests, cursor invalidation and refresh failures.

#### Scenario: A refresh returns the same values

- **WHEN** a refreshed page merges into exactly the current rows, order, continuation cursor and metadata, without changing loading state
- **THEN** the shown list remains intact and no list-state change is published

#### Scenario: Only page metadata changes

- **WHEN** a physical-backup refresh changes total storage usage while the merged rows and continuation cursor remain the same
- **THEN** the updated storage usage is published

#### Scenario: A row changes without changing identity

- **WHEN** a refreshed row keeps its identity but changes its status or another value
- **THEN** the new row values are published

#### Scenario: Only the effective continuation cursor changes

- **WHEN** a refresh preserves row values but changes the continuation cursor used by the loaded list
- **THEN** the accepted list uses the new cursor for subsequent scrolling

#### Scenario: An unchanged refresh overlaps the next page

- **WHEN** an unchanged first-page refresh finishes while a compatible next-page request is pending
- **THEN** no list-state change is published by that refresh
- **AND** the pending request keeps its loading indicator and appends normally when it finishes

## MODIFIED Requirements

### Requirement: Leaving a list ends its observable work

Unmounting or deactivating a list SHALL remove its polling, visibility and scroll subscriptions and invalidate pending callbacks. Late responses SHALL NOT publish state or show errors. Reactivating the lifecycle after cleanup SHALL permit a fresh load, including the setup-cleanup-setup sequence used by development checks.

#### Scenario: A response arrives after leaving the screen

- **WHEN** a list is unmounted with a page or refresh request pending and that request later succeeds or fails
- **THEN** no state or error notification is published for the removed list
- **AND** its timers, visibility subscriptions and scroll subscriptions remain removed

#### Scenario: Setup runs again after cleanup

- **WHEN** a list lifecycle is set up, cleaned up and set up again
- **THEN** the new lifecycle can load normally and receives no result from the previous lifecycle

#### Scenario: The browser tab becomes visible after leaving the screen

- **WHEN** a list is unmounted and the browser tab subsequently becomes visible
- **THEN** no refresh request is sent for the removed list
