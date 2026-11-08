## Context

See `proposal.md` for the reported loss of historical restore points and the agreed scope.

- `backend/internal/lib/gfs/gfs.go:29` sorts by creation time; `:41-47` derives rolling cutoffs from the newest item; `:49-85` caps a level's window by a coarser level's window. At `:109-118`, the first item fills the single weekly and monthly slots even when both levels are enabled alongside daily retention.
- `backend/internal/lib/gfs/gfs.go:142-150` already resolves timestamp ties by ascending ID and leaves the caller's slice unchanged. `backend/internal/lib/gfs/tier.go:5` allows one item to carry several retention levels.
- `backend/internal/modules/backups/internal/logical/retention_policy.go:84-117` filters completed backups, delegates selection and drops only unselected completed backups. The one-hour protection is applied separately at `:32-38`.
- `backend/internal/modules/backups/internal/physical/retention/cleaner.go:398-433` calls the same selector for completed full backups. Physical cleanup retains its own chain and dependency rules at `:319-380`; combined retention unions chain and full selections at `:439`.
- `frontend/src/pages/databases/ui/backup-config/GfsRetentionHint.tsx:5-28` renders the existing help used by both editors, currently through the logical dictionary key. The physical editor also has its own full-backup tooltip. Existing help tests live in `PhysicalBackupConfigFields.test.tsx:56`.
- `backend/api/openapi.yaml:5775-5868` describes both sets of GFS fields without distinguishing completed periods from current candidates.

The design follows root `AGENTS.md`, `backend/AGENTS.md` and `frontend/AGENTS.md`. The backend layout, module ownership and shared-code rules in `docs/architecture.md` keep generic calendar arithmetic in `lib/gfs`, and backup status and deletion decisions in the backups module. The HTTP API section requires contract-first generation. No dependency direction, persistence model or architecture rule changes.

## Goals / Non-Goals

**Goals:** Make selection survive creation followed by immediate cleanup across calendar boundaries. Keep it a deterministic pure calculation over eligible IDs, creation times and configured counts. Use the same calculation for logical backups and physical full backups.

**Non-Goals:** Add persisted assignments, a stateful promotion service, a separate scheduler or a generic retention framework. The public behavior and broader exclusions are in the proposal and delta specs.

## Decisions

### 1. Anchor periods to the newest successful eligible backup

Preserve the existing anchor: after filtering in each consumer, the selector takes the newest item's creation time. Convert timestamps to UTC before forming calendar boundaries or period keys. A failed, cancelled or running backup cannot move that anchor. Physical incrementals also cannot move the full-backup anchor.

This preserves the newest usable restore point during an outage or a pause in successful backups. Calendar rollover becomes effective when a successful backup enters a new period. Help and API descriptions must make that definition of "current" explicit.

Alternative rejected: using the cleanup clock would age out the final successful backups while a source remains broken and would change the existing pause behavior. Using the newest attempted backup has the same safety problem when attempts fail.

### 2. Compute independent calendar ranges

For a positive count N, derive the start of the current UTC period from the anchor:

| Level | Range selecting one newest item per period |
| --- | --- |
| Hourly | Current hour and N minus one preceding hours |
| Daily | Current day and N minus one preceding days |
| Weekly | N completed Monday-start weeks, plus the current week |
| Monthly | N completed calendar months, plus the current month |
| Yearly | N completed calendar years, plus the current year |

The current bucket of each weekly/monthly/yearly level is its candidate and does not consume a completed-period slot. Use half-open calendar ranges and calendar-month/year arithmetic from a period's first day, never from the reference's day of month. Empty periods consume a calendar position without reaching farther back to fill a quota. A level with a nonpositive count contributes nothing. Keep the consumers' existing no-effective-GFS-configuration guards; do not turn invalid persisted settings into delete-everything behavior.

Select each level's first eligible item per bucket from the sorted items, then union the choices while retaining all tier reasons. Keep the existing tie order and input immutability. Remove rolling duration cutoffs, cross-level caps, `laterOf` if it has no remaining use, and counters that treat a current long period as a completed-period slot. Replace the selector's positional counts with a named selection input and update both callers, as root naming rules require for four or more parameters. Keep helpers local to the selection concept instead of creating a general calendar abstraction.

Alternatives rejected: merely increasing weekly/monthly counts hides the defect in settings and still mixes candidates with completed history. A minimum of seven daily backups or thirty-one daily backups changes the user's storage budget. Filling the last N nonempty periods can retain arbitrarily old history across gaps. Hierarchical caps let an unrelated count silently shorten another level's retention.

### 3. Protect the candidate before the period ends

The selector always sees the newest successful candidate of each enabled long period. A newer success replaces it for that level; the old item remains if another level still needs it. At the next successful period transition, the previous candidate is already present and becomes the completed representative. No database field is needed: chronological selection and the kept files are sufficient.

```text
successful backup
        |
        v
select newest per calendar bucket
        |
        +--> recent hours/days
        +--> completed weeks/months/years
        +--> current long-period candidates
        |
        v
union selected IDs --> existing deletion protections --> cleanup
```

For daily successes at 12:00 UTC, settings `days=2, weeks=1, months=1`, and cleanup after each success, the November 5 selection is November 5, November 4, November 1 and October 31. Continue cleanup into December to prove October's monthly representative eventually expires.

Alternatives rejected: selecting only completed periods loses the candidate before a period closes when shorter levels are disabled. Keeping the first backup of a period would change which restore point represents that period. Persisted labels and promotion jobs add state and failure modes that timestamp selection does not need.

### 4. Keep deletion policy with each consumer

Logical retention still filters successful backups, keeps failed/cancelled rows, applies the one-hour shield, and checks running backups and restores before deletion. Physical retention still calculates its full-backup selection from completed full backups, then applies grace, chain, combined-policy, incremental and WAL rules. A selected physical full is not a promise to keep its dependents. No selection branch is duplicated into the consumers.

Alternative rejected: repairing only the logical caller would leave the same public GFS setting with different meanings in physical retention and duplicate calendar rules. Changing physical dependency cleanup along with this fix would expand the change beyond selection.

### 5. Explain the semantics at existing API and interface surfaces

Add the following product terms beside Retention in `docs/glossary.md`. Use the same terms in the API descriptions, documentation and English interface help; translate their meanings consistently in the other dictionaries. These concepts introduce no stored tier assignments or API fields. The shared GFS specification owns their selection rules.

| Term | Meaning |
| --- | --- |
| GFS candidate | The newest successfully completed eligible backup protected for the current week, month or year of an enabled GFS level. It does not consume a completed-period slot and can be replaced by a newer success in that period. |
| GFS representative | The newest successfully completed eligible backup selected for a calendar period. A completed weekly, monthly or yearly representative occupies a completed-period slot; the protected selection for a current long period is called a GFS candidate instead. |
| GFS reference instant | The creation time of the newest successfully completed eligible backup, which determines the current UTC calendar periods. Only logical backups qualify for logical retention, and only physical full backups qualify for physical full-backup retention. Failed, cancelled, running and, for physical full-backup retention, incremental backups do not advance it. |

Update descriptions of `LogicalRetention`, `FullBackupsRetention` and their GFS properties in `backend/api/openapi.yaml` before implementation, then run `mise run generate-api`. Retain field names, required fields, types, defaults and validation. Document the anchor, UTC boundaries, short-period inclusion, completed long periods, candidates and overlaps without introducing an API flag.

Update existing GFS policy descriptions, help and tooltips in all seven dictionaries. Allow the shared `GfsRetentionHint` to choose the appropriate logical or physical explanation through a typed domain choice, with both callers explicit, so physical help consistently says full backups. Keep its existing disclosure and draft behavior. Use the localization provider rather than storing translated text in state. Short labels can stay brief, but must not contradict the detailed help. No new controls, changed form behavior or shared localization machinery are needed.

Alternatives rejected: leaving the existing "last N" wording would preserve the misunderstanding that caused the report. One generic explanation for all backups would obscure the full-only rule in physical retention. Defining the new terms only in help would leave the glossary incomplete and allow API wording to drift. A new dedicated documentation page or preview screen is unnecessary for this repair.

### 6. Verify the sequence of deletions and the consumer boundaries

Start with a regression that fails on the existing selector for the reported configuration. A pure test adds one backup, calculates the keep set and physically removes unselected items from its next iteration. Repeat through multiple weeks, months and years. Testing a fully populated list alone cannot prove that a candidate survived earlier cleanup.

Use unit tests for the pure calendar matrix: hourly/daily inclusive ranges, N completed long periods, single-level policies, disabled levels, gaps, UTC offsets, exact boundaries, leap February, ISO-year transitions, ties, shuffled inputs, input immutability, overlaps, independent windows and expiry. Preserve checks for no eligible items and no active levels.

Use a logical module HTTP test with real backups, the test clock and explicit retention job passes. Configure the policy through the API, create daily successful backups, run cleanup after each, then assert the exact listed IDs and local storage files/metadata. Advance beyond the one-hour shield, process pending file deletions, and assert an expired representative's files are removed. Exercise newer failed/cancelled attempts with valid test setup and prove they do not shift the successful selection. Reuse existing running/restore and other-policy coverage instead of rebuilding those features.

Extend the physical catalog tests to perform repeated GFS cleanup while seeding successive completed full backups using the existing catalog fixture for states the API cannot create. Verify selected historical fulls survive and unselected fulls disappear after grace; include full-only and combined retention, with existing dependency and WAL assertions preserved. Failed/incremental/running records must not advance the full-backup anchor.

Use the existing logical configuration and physical fields screen tests to assert explanatory content and unchanged draft values/dirty state. Run dictionary placeholder checks and test a non-English explanation through the real locale provider. A copy-only update does not require new visual references, browser scenarios or a test of unchanged locale-switch infrastructure.

Alternative rejected: exhaustive real backup runs for every calendar combination would be slow without increasing confidence over pure arithmetic tests plus consumer integration coverage.

## Risks / Trade-offs

- Existing settings can retain more successful files because completed long-period slots are now additional to their candidates. Explain the behavior change in the proposal and help; keep overlapping IDs deduplicated. A monthly-only count of one can keep two successful backups.
- Calendar ranges replace duration windows, so behavior near boundaries and gaps changes even for hourly/daily settings. Specify exact half-open ranges and test both edges; do not claim that the new keep set is always a superset of the old one.
- An outage can keep old successful backups longer because the anchor is frozen. This preserves existing behavior and is explicit in the API, help and tests.
- New semantics cannot recover history already removed. Start building completed-period history from available successes; do not synthesize missing backups.
- A shared selector can affect physical restore points. Verify physical full and combined policies while preserving dependency cleanup rules and safety protections.

## Migration Plan

No data migration or config rewrite is required. The next retention pass after an upgrade uses the new selection for existing GFS settings. Do not add a compatibility flag or retain the old algorithm. No deleted files are recreated.

A binary rollback reinstates the old retention behavior and can delete the historical points this fix retained. If an operator needs to preserve those points while rolling back, they must prevent the old GFS cleanup from running until an appropriate retention configuration is in place. Rollback does not reconstruct files removed by either algorithm.
