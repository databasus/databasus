## ADDED Requirements

### Requirement: Session transitions discard session-owned request state

Ending or replacing a session SHALL discard that session's reusable request results and completed action state, request cancellation of its ordinary pending reads, and stop its automatic refresh work. A newly activated session SHALL load its own data even when it belongs to the same user and selects the same workspace and resource. A pending action from the ended session SHALL NOT publish data, schedule follow-up reads, show feedback, close a dialog or navigate in the new session. Discarding frontend request state SHALL NOT be treated as proof that an already submitted server action was cancelled. Language, theme and the application-version reload cooldown SHALL retain their existing lifetimes.

#### Scenario: Cached resource data exists when the user signs out

- **WHEN** a user signs out while resource data and a pending ordinary read belong to that session
- **THEN** the results and completed action state are discarded, the read is signalled for cancellation and the session's automatic refresh work stops
- **AND** sign-in shows no resource content from the ended session

#### Scenario: The same user signs back in

- **WHEN** the same user signs back in and opens the same workspace and resource
- **THEN** a fresh read obtains data authorized by the newly activated session
- **AND** results from the previous session cannot complete that load

#### Scenario: A previous session's mutation succeeds late

- **WHEN** a mutation submitted under an earlier session succeeds after a new session activates
- **THEN** it causes no data update, follow-up read, feedback, dialog closure or navigation in the new session
- **AND** the new session remains active regardless of the old request's outcome

#### Scenario: A session transition interrupts polling

- **WHEN** a session is replaced while a polling read is pending
- **THEN** no later tick belongs to the previous session and its pending response cannot repopulate the current interface
- **AND** language, theme and the version-reload cooldown remain selected
