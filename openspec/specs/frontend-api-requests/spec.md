# Frontend API Requests Specification

## Purpose

Give frontend callers one predictable HTTP failure model while preserving successful empty operations, cancellation and explicit domain requests.

## Requirements

### Requirement: HTTP failures have stable frontend classifications

A request that cannot reach the server SHALL be classified as `network_unreachable`. A response that cannot provide required data, has unreadable JSON, or refuses an operation without a usable Problem SHALL be classified as `request_failed`, retaining its HTTP status when available. A valid Problem SHALL retain its code and status. These failures SHALL reach the interface through translated error handling. An operation whose success has no body SHALL complete successfully without inventing response data. Intentional lifecycle cancellation SHALL NOT be presented as a network or server failure.

#### Scenario: The connection fails

- **WHEN** a declared request fails before receiving a response
- **THEN** its caller receives the network failure classification and the interface can display its translated network message

#### Scenario: A proxy returns an unusable error response

- **WHEN** an operation receives an error status with an empty, non-JSON or structurally invalid Problem body
- **THEN** its caller receives the request failure classification with that status
- **AND** the interface does not display the response text as its error message

#### Scenario: Required data is missing or unreadable

- **WHEN** an operation requiring a JSON body receives an empty response or unreadable JSON
- **THEN** its caller receives the request failure classification rather than a successful empty value

#### Scenario: A bodyless success completes

- **WHEN** an operation whose successful response has no body returns that success status
- **THEN** its caller observes success

#### Scenario: A request is intentionally cancelled

- **WHEN** a request is cancelled because its screen or list lifecycle has ended
- **THEN** it does not produce a network-failure notification

#### Scenario: An accepted transfer omits required operation data

- **WHEN** a database transfer returns status 202 without its required progress data
- **THEN** its caller receives the request failure classification with status 202 rather than treating the transfer as completed

### Requirement: Request handling does not replay mutations implicitly

Failure normalization and complete-list traversal MUST NOT automatically repeat a mutation after a refusal, network failure or unreadable response. A new mutation request SHALL require a new explicit invocation by its caller.

#### Scenario: The outcome of a mutation is unknown

- **WHEN** a mutation fails while awaiting its response
- **THEN** only the original request was sent
- **AND** no automatic retry follows

### Requirement: Importing the shared API starts no request machinery

Loading shared API declarations SHALL NOT start timers, queues, HTTP requests or retry loops. Request work SHALL start only when a caller invokes an operation.

#### Scenario: A screen loads the API declarations

- **WHEN** shared API declarations are imported before any operation is invoked
- **THEN** no interval, queued request or HTTP request starts as a consequence of that import

### Requirement: New screen visits read current server data

Opening a screen or changing its workspace, resource or request filters SHALL start a fresh read for that context. A response from an earlier visit SHALL NOT substitute for this read or be presented as the completed load of the new visit. Results SHALL NOT be persisted for reuse on later visits. An open screen SHALL retain its accepted data during a background refresh according to its existing failure policy. Concurrent consumers of the same current read SHALL be allowed to share pending work, provided that each new visit still requests current data and isolates different contexts.

#### Scenario: A resource changes while its screen is closed

- **WHEN** the user leaves a resource screen, the resource changes on the server and the user returns
- **THEN** the screen requests the resource again and displays its current server values
- **AND** the previous visit's response does not complete the new load

#### Scenario: A different workspace or filter is selected

- **WHEN** the user changes the selected workspace, resource or request filters
- **THEN** the interface reads the newly selected context
- **AND** rows and failures from the preceding context do not appear as that context's results

#### Scenario: Another current consumer already has data

- **WHEN** a screen opens while another active consumer has an earlier result for the same context
- **THEN** opening the screen still starts or joins a fresh pending read for that context
- **AND** the retained result alone does not count as completing the new visit's load

#### Scenario: A new visit fails while another consumer holds an earlier result

- **WHEN** a new screen's fresh read fails while another active consumer holds an earlier result
- **THEN** the new screen shows its initial-load failure without presenting that earlier result
- **AND** the existing consumer retains accepted data only under its background-failure policy
- **AND** an explicit retry can load current data for both consumers

### Requirement: Obsolete ordinary reads are cancelled through their HTTP work

An ordinary screen read SHALL request cancellation of its pending HTTP work when its context is replaced, when the read is explicitly superseded or when its final active consumer leaves. A complete-list read SHALL propagate cancellation to its pending page and SHALL NOT request a subsequent page after cancellation. An obsolete success or failure SHALL change neither current data, loading state nor feedback. Cancellation SHALL NOT be shown as a network failure. Specialized paged-list and WAL lifecycles SHALL retain their existing request-acceptance and refresh behavior.

#### Scenario: The last consumer leaves with a read pending

- **WHEN** the last active consumer leaves an ordinary read while its HTTP request is pending
- **THEN** that request is signalled for cancellation
- **AND** its late completion produces no screen update or error notification

#### Scenario: A complete listing is cancelled on a later page

- **WHEN** a complete listing is cancelled while a page with a possible continuation is pending
- **THEN** the pending page is signalled for cancellation and no following page is requested
- **AND** a partial collection is not accepted as a complete result

#### Scenario: A newer read supersedes an earlier one

- **WHEN** an explicit reload supersedes an unfinished ordinary read and the earlier response later arrives
- **THEN** the earlier request is signalled for cancellation
- **AND** only the current read can publish data, an error or completion state

### Requirement: Successful actions reconcile active server data

A successful server action SHALL acknowledge the returned values in the affected active interface and request fresh affected data where a follow-up read is needed. A read started before the action succeeded SHALL NOT overwrite the acknowledged values. A failed follow-up read SHALL NOT be reported as a failed mutation or cause the action to run again. Acknowledgement and refresh SHALL preserve selection, open sections and newer unsaved input under the screen's existing editor rules unless the current read establishes access loss. A current workspace-access refusal SHALL clear protected content and reconcile accessible workspaces under the existing access policy. The application SHALL NOT introduce automatic retries or an offline replay queue for explicit server actions.

#### Scenario: A mutation refreshes a resource list

- **WHEN** the user successfully changes a resource shown in an active list
- **THEN** the interface acknowledges that success and requests current affected list data
- **AND** the refreshed list shows the accepted server values without a manual page reload

#### Scenario: A read started before a save finishes last

- **WHEN** a save succeeds while an earlier read is pending and that earlier read later returns older values
- **THEN** those values do not replace the acknowledged save
- **AND** newer unsaved input remains intact

#### Scenario: Refresh fails after a successful action

- **WHEN** an action succeeds and its follow-up read fails without establishing access loss
- **THEN** the action remains acknowledged as successful and is not replayed
- **AND** the screen retains accepted data and handles the read failure under its existing refresh policy

#### Scenario: A follow-up read establishes workspace access loss

- **WHEN** an action succeeds and its current follow-up read establishes that workspace access is no longer available
- **THEN** protected list and detail content is cleared before accessible-workspace reconciliation completes
- **AND** another accessible workspace is selected or the existing empty-workspace or recoverable-failure experience is shown without restoring the protected content
- **AND** the successful action is not reclassified as failed or replayed

#### Scenario: An action is submitted while connectivity is unavailable

- **WHEN** the user explicitly submits an action while the browser reports no connectivity
- **THEN** the attempt follows ordinary request failure handling rather than waiting in an offline queue
- **AND** reconnecting does not submit the action again without a new user invocation

### Requirement: Request infrastructure preserves screen refresh policy

Request infrastructure SHALL preserve each screen's refresh interval, browser-tab visibility rules, prerequisite reads and stopping conditions. Initial failures that require explicit retry SHALL remain visible until that retry. Background failures SHALL retain the screen's existing feedback and continuation policy. Merely focusing the window, reconnecting the network, or changing language or theme SHALL NOT introduce a request that the screen's existing policy does not call for. A polling tick SHALL NOT start a competing read while the same ordinary refresh is pending.

#### Scenario: Healthcheck refresh fails after a successful first load

- **WHEN** a current-day healthcheck refresh fails after attempts have loaded successfully
- **THEN** the shown attempts remain without a blocking failure and a later scheduled refresh can recover
- **AND** the refresh interval remains one minute

#### Scenario: Healthcheck initial load fails

- **WHEN** the first attempts read fails
- **THEN** the failure remains available with explicit retry
- **AND** a timer tick does not replace it or silently retry it

#### Scenario: Storage usage polling fails

- **WHEN** a storage-usage read fails after polling has started
- **THEN** automatic usage reads stop until explicit retry under the existing failure policy
- **AND** a permitted retained usage value does not overwrite editing input

#### Scenario: Window focus has no screen refresh rule

- **WHEN** the window gains focus or connectivity returns and the screen has no corresponding refresh rule
- **THEN** no extra read or mutation is sent solely because of that event

#### Scenario: A polling tick occurs during a pending refresh

- **WHEN** an ordinary refresh remains pending when its next polling tick occurs
- **THEN** that tick does not launch a competing refresh or cancel the pending one
