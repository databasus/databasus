## Context

See `proposal.md` for motivation. This design resolves the choice between persistent help and a tooltip, including how assistive technology receives the warning.

The checkbox currently sits inside a `Form.Item` without help text (`frontend/src/features/edit-storage/ui/edit/storages/EditSFTPStorageComponent.tsx:168`). The advanced section is hidden until requested, but starts expanded when skipping verification is already configured (same file, lines 26 and 167). The editor already associates credential help text through unique IDs and `aria-describedby` (same file, lines 143-145). The existing English warning is at `frontend/src/shared/i18n/locales/en.ts:2165`; all five other dictionaries contain the same key.

Root `AGENTS.md` and `frontend/AGENTS.md` constrain this change. The existing ownership stays in `features/edit-storage`, consistent with `docs/architecture.md:29`; no architecture or shared-control contract changes are needed. `docs/frontend-design.md` governs field descriptions, and `docs/frontend-localization.md` governs translating during rendering.

## Goals / Non-Goals

**Goals:** Make the warning visible alongside the option and available as its accessible description using the existing form and localization infrastructure.

**Non-Goals:** Introduce a shared warning component, a confirmation dialog or a new form-validation rule. Product exclusions are listed in `proposal.md`.

## Decisions

### Use persistent field help with an explicit accessible description

Render `t('storages.edit.sftp.skipHostKeyTooltip')` in the checkbox's `Form.Item help`. Give the warning an ID derived from the component's existing `useId` value and reference it from the checkbox's `aria-describedby`. The warning renders whenever the advanced section is visible, regardless of the checkbox value.

A `Tooltip` would require another interaction to discover the warning. Showing help only after checking the option would hide the warning at the moment the user makes the decision. Persistent field help avoids both limitations and follows the editor's existing approach to credential descriptions.

### Reuse the existing translated sentence

Resolve the existing key through `useTranslation` during rendering. Rewriting the warning or introducing another key would expand the change without fixing an additional defect. Storing translated text in state would prevent a visible warning from following language changes.

### Cover the field through the existing screen renderer

Add `EditSFTPStorageComponent.test.tsx` beside the component and use the real `renderScreen` providers and user-event. Begin with the scenario `warns that skipping host key verification reduces security when advanced settings are expanded`: render settings that do not skip verification, expand advanced settings, and assert visible warning text and the checkbox's accessible description. Run it before the fix and confirm it fails because the warning is missing.

Add coverage for settings that already skip verification and parameterize the localized warning checks over all six dictionaries. Use dictionary values for translated assertions. Do not rely on text merely existing inside a hidden region: check visibility after expansion and resolve the checkbox's description to the actual warning. The component makes no HTTP requests, so it needs no HTTP handlers.

Testing only the translation key or inspecting JSX would miss whether users can see the text and whether it describes the checkbox. A separate browser scenario is unnecessary for this field-description change; the screen test exercises the relevant behavior without a new browser fixture.

## Risks / Trade-offs

- Persistent help increases the advanced section's height -> Use the existing form layout and allow the translated sentence to wrap naturally.
- Help can be visible without being associated with the checkbox -> Set an explicit description ID and verify that relationship in the regression test.
- A test can pass on hidden text -> Assert visibility after expanding the section and on initial rendering when the option is already enabled.

## Migration Plan

No migration is required. The change affects presentation only; rollback removes the field help and its description reference without changing saved settings.

## Verification Repair

The full `mise run test` run exposed a pre-existing failure in the hourly WAL statistics scenario (`backend/internal/modules/backups/physicalapitests/streamstatistics/statistics_test.go:190`). The test compared the last bucket with wall time, while the service uses its injected clock (`backend/internal/modules/backups/internal/physical/wal_stream_statistics_service.go:55`) and `apptest` freezes that clock during construction (`backend/internal/app/apptest/apptest.go:85`). Setup crossing a minute boundary can therefore fail the assertion without a product defect.

Set the application's clock two minutes behind its initial value to reproduce the mismatch without waiting. Assert that the last bucket starts at that clock's current minute, and retain the bucket count, width, consecutive spacing and end-time assertions. This strengthens the check and removes dependence on elapsed setup time. Widening a tolerance or retrying the test would retain the mismatch between clocks. This test-only repair follows `backend/AGENTS.md` and the standing repair authorization in root `AGENTS.md`.

The full run passed all 2370 frontend tests and the other backend packages before stopping on this failure. Repeat the affected backend package directly, as `docs/testing.md#checking-work` prescribes for a reported failure, and run the upgrade stage separately because the failed full run did not reach it. No WAL statistics product requirement or HTTP contract changes.
