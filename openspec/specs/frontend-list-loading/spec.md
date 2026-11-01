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

Unmounting or deactivating a list SHALL remove its polling and scroll subscriptions and invalidate pending callbacks. Late responses SHALL NOT publish state or show errors. Reactivating the lifecycle after cleanup SHALL permit a fresh load, including the setup-cleanup-setup sequence used by development checks.

#### Scenario: A response arrives after leaving the screen

- **WHEN** a list is unmounted with a page or refresh request pending and that request later succeeds or fails
- **THEN** no state or error notification is published for the removed list
- **AND** its timers and scroll subscriptions remain removed

#### Scenario: Setup runs again after cleanup

- **WHEN** a list lifecycle is set up, cleaned up and set up again
- **THEN** the new lifecycle can load normally and receives no result from the previous lifecycle

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
