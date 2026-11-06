## ADDED Requirements

### Requirement: A failed verification reports its underlying diagnostic

When the verification agent reports a failed verification, it SHALL include the available underlying failure diagnostic separately from the short failure summary. A rejected restore SHALL include the database restore tool's error output. A failure before restoration, or during the subsequent check, SHALL include the underlying error available to the agent. The agent SHALL deliver this diagnostic with the result report, before removing the temporary environment; viewing it later SHALL require neither a running agent nor that environment.

An absent diagnostic SHALL remain absent. The system SHALL NOT fabricate tool output for a failure detected by the server or for a run whose agent did not deliver a diagnostic. Diagnostics SHALL NOT change verification verdicts, exit-code classification, retry decisions or the evaluation of ignorable restore errors.

#### Scenario: Restore fails with a specific extension error

- **WHEN** the restore tool reports an unavailable extension followed by dependent type or table errors, and the agent reports the failed verification
- **THEN** the saved details include the available tool errors separately from the short failure summary, subject to diagnostic sanitization and size limits
- **AND** the verification remains failed under the existing verdict rules

#### Scenario: The agent fails before restoration starts

- **WHEN** the agent cannot download the backup or prepare its temporary environment and has an underlying error
- **THEN** its failure report includes that diagnostic without inventing a restore exit code
- **AND** the existing retry policy still applies

#### Scenario: The check fails after a successful restore

- **WHEN** restoration succeeds but the agent's subsequent database check fails with an underlying error
- **THEN** the failure diagnostic describes the check failure rather than presenting successful restore output as its cause

#### Scenario: Diagnostics are viewed after cleanup

- **WHEN** an authorized user opens a failed verification whose agent reported a diagnostic and has since removed the temporary environment or stopped
- **THEN** the saved diagnostic remains available

#### Scenario: No diagnostic was delivered

- **WHEN** a failed verification predates diagnostic reporting or its agent disappeared before delivering a report
- **THEN** the existing short failure reason remains available and the detailed diagnostic is absent

#### Scenario: The restore fails without captured stderr

- **WHEN** the restore tool fails and the agent has only a generic restore failure summary with no captured stderr
- **THEN** the report preserves that summary and omits the detailed diagnostic
- **AND** report sanitization does not manufacture diagnostic output from the summary

#### Scenario: Sanitization removes all diagnostic output

- **WHEN** the agent's captured diagnostic becomes empty after sanitization
- **THEN** the diagnostic is omitted and remains absent through subsequent report processing

### Requirement: Verification diagnostics have a bounded and explicit representation

A saved diagnostic SHALL contain at most 65,536 bytes of valid UTF-8 output. Sanitized output within this limit SHALL be preserved in full. Longer sanitized output SHALL retain its beginning and end, with an omission marker between them and an explicit truncation indicator. The marker SHALL count toward the limit, and truncation SHALL NOT split a UTF-8 character. Sanitization SHALL happen before truncation, so slicing cannot expose part of a credential that would otherwise be masked.

The report receiver SHALL reject diagnostic output exceeding this byte limit with `400` and `code` `invalid_request`, without changing the verification. Detailed diagnostics SHALL be optional, and an omitted diagnostic SHALL be accepted. A supplied diagnostic SHALL have nonempty output and an explicit truncation indicator. A completed report SHALL NOT carry a failure diagnostic; such a report SHALL be rejected without changing the verification.

#### Scenario: Short output

- **WHEN** a failed verification produces sanitized output within the byte limit
- **THEN** its saved diagnostic contains all that output and indicates that it was not truncated

#### Scenario: Long multilingual output

- **WHEN** a failed verification produces sanitized output exceeding the byte limit, including multibyte characters
- **THEN** its saved diagnostic retains the beginning and end within the limit, remains valid UTF-8 and indicates the omitted middle
- **AND** truncation does not change the verification's verdict

#### Scenario: An oversized diagnostic is submitted

- **WHEN** an agent submits diagnostic output exceeding 65,536 UTF-8 bytes
- **THEN** the response is `400` with `code` `invalid_request` and no verification fields change

#### Scenario: A completed report includes a failure diagnostic

- **WHEN** an agent submits a completed verification report with a failure diagnostic
- **THEN** the response is `400` with `code` `invalid_request` and no verification fields change

### Requirement: Verification diagnostics protect credentials and recognizable row excerpts

Before transmitting diagnostics, the agent SHALL mask credentials it knows and credential patterns covered by the application's existing log redaction. Before persisting them, the server SHALL also apply the application's redaction. Both boundaries SHALL suppress recognized PostgreSQL `DETAIL`, `CONTEXT` and `Command was:` sections, including their multiline bodies, because those sections can contain row values or executable SQL bodies. Primary error records and their object identifiers before the first sensitive section SHALL remain available after this filtering. When the text supplies no trusted section-end boundary, the rest of the stream SHALL be omitted with that section; record-looking lines within it SHALL NOT restart output. The short failure summary SHALL receive the same applicable masking and section filtering, so it cannot bypass these protections.

User interfaces SHALL render diagnostic output as literal text and SHALL NOT interpret HTML, terminal control sequences or interpolated translation markup from it. Diagnostics SHALL NOT be appended to verification lists, notifications or audit events. These rules do not claim to remove arbitrary application values embedded in primary error messages; access remains restricted to users who can view the verification's database.

#### Scenario: Output contains credentials and a row excerpt

- **WHEN** a diagnostic includes a credential-bearing connection URL, a credential known to the agent and multiline PostgreSQL sections containing sample row values or SQL
- **THEN** the agent's transmitted diagnostic and summary contain neither those credentials nor those section bodies
- **AND** a diagnostic supplied directly to the server receives the same pattern masking and section filtering before persistence
- **AND** the primary failure remains readable

#### Scenario: A sensitive body contains record-looking lines

- **WHEN** a sensitive PostgreSQL section contains SQL or row values whose lines look like errors, restore records or sanitization markers
- **THEN** those lines and the remainder of the ambiguous body are omitted at both sanitization boundaries
- **AND** primary error records preceding the section remain readable

#### Scenario: Output contains markup

- **WHEN** saved output contains angle brackets, braces or terminal control sequences
- **THEN** viewing or copying it does not execute markup, expand translation expressions or execute terminal commands

### Requirement: Verification diagnostics follow the final attempt and existing ownership rules

The server SHALL store an accepted failed report's diagnostic atomically with its verification outcome. Only the agent currently owning a running verification SHALL be able to write its diagnostic. A refused report SHALL change neither the outcome nor the diagnostic. Requeueing SHALL clear the previous attempt's diagnostic, and completion, cancellation or a server-detected failure without a diagnostic SHALL leave no diagnostic from an earlier attempt. The system SHALL NOT keep an attempt history as part of this feature. Removing a verification SHALL remove its diagnostic with it.

#### Scenario: An infrastructure failure is retried

- **WHEN** an agent reports an infrastructure failure with a diagnostic and the server requeues the verification
- **THEN** the pending run has no diagnostic from the failed attempt
- **AND** a later terminal failure exposes only the diagnostic of its final attempt, if delivered

#### Scenario: A stale agent reports after cancellation or reassignment

- **WHEN** a verification is canceled or reassigned before the previous agent's report is accepted
- **THEN** that report is refused under the existing ownership rules and cannot overwrite the verification's status or diagnostic

#### Scenario: A failed report is committed

- **WHEN** the server accepts a final failed report with a diagnostic
- **THEN** the failed outcome and diagnostic become visible together, or neither is written if the transaction fails

### Requirement: Verification diagnostic access matches detail access

Only a signed-in user authorized to view the verification's database SHALL be able to retrieve its diagnostic, through the existing verification detail operation. Verification list responses SHALL omit diagnostics. A user outside the workspace SHALL receive the existing `404` response without diagnostic content; a request without a session SHALL receive `401`.

#### Scenario: An authorized member opens details

- **WHEN** a member who can view the database requests a failed verification's details
- **THEN** the response includes the saved diagnostic when present
- **AND** listing the same verification does not include that diagnostic

#### Scenario: A user cannot view the workspace

- **WHEN** a signed-in user outside the workspace requests verification details
- **THEN** the response is `404` and reveals no diagnostic output

### Requirement: The verification sidebar separates the summary from expandable error details

For a failed verification, the sidebar SHALL keep its short failure summary visible and provide an initially collapsed "Error details" section immediately below it. Expanding the section SHALL display the saved diagnostic in readable monospace text with preserved line breaks and a copy action. Copying SHALL copy the displayed sanitized output, including an omission marker when present, and report clipboard success or failure accurately. The section SHALL explain when no diagnostic was recorded and SHALL distinguish truncated output from complete output. It SHALL NOT offer copying when there is no output.

The section SHALL support keyboard expansion and collapse, narrow screens, both themes and live language switching. Labels and feedback SHALL follow the selected interface language; diagnostic output SHALL keep its literal content. Opening another verification SHALL reset the section to collapsed and SHALL NOT show output from the previously selected verification. Verifications that are not failed SHALL NOT show this failure-diagnostic section.

#### Scenario: A user investigates a failed verification

- **WHEN** a user opens a failed verification with recorded diagnostic output
- **THEN** the short failure summary is visible and the error-details section is collapsed
- **AND** expanding the section reveals the diagnostic and allows copying it

#### Scenario: Output was truncated or not recorded

- **WHEN** the user expands the details of a failed verification
- **THEN** truncated output has a visible truncation notice, while absent output has an explanatory empty state and no copy action

#### Scenario: Language changes while details are open

- **WHEN** a user changes the interface language with error details expanded
- **THEN** labels and active feedback update without changing the diagnostic or collapsing the section

#### Scenario: The selected verification changes

- **WHEN** a user opens a different verification while an earlier detail request or copy action is pending
- **THEN** the new verification starts collapsed, and late work from the previous verification cannot replace its diagnostic or show stale copy feedback
