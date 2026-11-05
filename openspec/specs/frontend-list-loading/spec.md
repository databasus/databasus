# Frontend List Loading Specification

## Purpose

Keep complete selections and visible paged lists correct when cursor responses arrive late, filters change or the user leaves a screen.

## Requirements

### Requirement: Complete lists follow every cursor with unchanged domain filters

Complete workspace, database, storage and notifier listings SHALL request successive pages until the server omits the next cursor. Each page SHALL retain the listing's domain filters and page limit. The caller SHALL receive items in page order, including when the final page is empty. Failure of any page SHALL fail the complete listing without returning a successful partial list or automatically retrying the failed page.

#### Scenario: A complete listing spans several pages

- **WHEN** a listing receives successive next cursors and then an empty page with no next cursor
- **THEN** it requests each cursor in order with the same domain filters and limit
- **AND** it returns all preceding items in page order

#### Scenario: A later page fails

- **WHEN** the second or later page cannot be loaded
- **THEN** the complete listing fails rather than reporting the earlier pages as a complete result
- **AND** the failed request is not retried automatically

### Requirement: Reset and reload isolate the new list from old requests

Changing a list's selection or filters, or explicitly reloading it, SHALL invalidate preceding requests and their cursor. Until the new first page succeeds, scrolling and polling SHALL NOT issue next-page or refresh requests against the old list. A superseded request SHALL NOT publish rows, errors, page metadata or loading-state changes into the new list. Aggregate metadata such as physical-backup total usage SHALL be accepted under the same generation as the first-page rows. Failure of the current first page SHALL end its loading state and permit a new explicit reload without reviving the old cursor.

#### Scenario: Scrolling while a filter reset is pending

- **WHEN** the user changes a filter and scrolls before the new first page finishes
- **THEN** no request uses the previous filter's next cursor

#### Scenario: Polling overlaps the first page

- **WHEN** a polling tick occurs while the initial or reloaded first page is pending
- **THEN** it does not start a competing refresh

#### Scenario: An old next-page request finishes during a new load

- **WHEN** a request from a replaced list finishes while the new list is loading
- **THEN** it changes neither the new rows nor the new loading indicators and reports no stale error

#### Scenario: A physical-backup refresh finishes after the new list loads

- **WHEN** an old physical-backup refresh returns after a newer list reload has succeeded
- **THEN** its total usage does not replace the accepted list's usage
- **AND** its rows and loading indicators remain ignored

#### Scenario: A reload fails

- **WHEN** the current first-page request fails
- **THEN** its loading indicator ends, the current error is reported once and an explicit reload can start a fresh request
- **AND** the previous cursor remains unusable

### Requirement: Refresh and scrolling preserve a valid loaded tail

A first-page refresh SHALL update the current list without discarding its already loaded tail when the refreshed page's last row occurs in the loaded rows and anchors a tail below it. A next page SHALL append without duplicating existing items or replacing a newer loaded copy with the page's repeated copy. If the refreshed page's last row cannot anchor a loaded tail, or the page is empty or complete, the visible list SHALL adopt that page's extent and cursor, as specified by the backup-list refresh requirement. Concurrent requests SHALL own their own loading state; stale completion MUST NOT release or retain another generation's operation lock.

#### Scenario: A refresh updates rows above an already loaded tail

- **WHEN** a refreshed first page's last row anchors a loaded tail and the user has scrolled to later pages
- **THEN** the changed first-page rows appear and the aligned loaded tail and its next cursor remain

#### Scenario: The next page repeats a refreshed item

- **WHEN** a next-page response repeats an item already updated in the visible list
- **THEN** the list contains that item once and keeps its loaded copy

#### Scenario: The refreshed page cannot align with the old list

- **WHEN** the refreshed first page's last row cannot anchor a loaded tail, the page is empty or it contains the complete list
- **THEN** it replaces the old extent and the list uses its cursor

#### Scenario: An unchanged refresh finishes while the next page is pending

- **WHEN** a first-page refresh returns the same continuation cursor while the next page is pending
- **THEN** the pending next page and its loading indicator remain active
- **AND** its eventual response appends normally without a duplicate request

#### Scenario: An old refresh remains pending after reload

- **WHEN** a new first page has loaded while a refresh of the replaced list is still pending
- **THEN** the new list can refresh independently
- **AND** completion of the old refresh cannot alter its rows or request state

#### Scenario: Refresh and the next page finish in either order

- **WHEN** a refresh and a next-page request for the current list overlap and finish in either order
- **THEN** the final list follows the refresh alignment and deduplication rules
- **AND** each loading indicator reflects only its current request

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

### Requirement: A failed refresh gives no feedback

A failed first-page refresh of a polled list SHALL keep the shown rows, their cursor and their loading state, and SHALL show no blocking alert, message or error. This SHALL also hold for a refresh that follows a successful action, such as a cancel, because the action already succeeded and the next refresh updates the row. The next refresh SHALL run on its own schedule. A failed first page or next page SHALL still be reported, once per failure, and the report SHALL identify which of the two failed. A failed first page SHALL stay available to the screen until an explicit reload clears it.

#### Scenario: A refresh fails

- **WHEN** a polled list is shown, possibly with a next page pending, and its refresh fails
- **THEN** the rows stay unchanged, no error is reported, the cursor still loads the next page, a pending next page keeps its loading indicator and the next refresh still runs

#### Scenario: A refresh after a successful action fails

- **WHEN** a cancel succeeds and the refresh that follows it fails
- **THEN** no error is reported and the next refresh still runs

#### Scenario: The first page fails

- **WHEN** the first page of a list fails
- **THEN** loading ends, the list has no rows, the failure is reported once as a first-page failure and stays available to the screen, and an explicit reload clears it and starts a fresh request

#### Scenario: A next page fails

- **WHEN** the next page of a list fails
- **THEN** the failure is reported once as a next-page failure, the loaded rows and the cursor stay, and a later scroll requests that page again

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
