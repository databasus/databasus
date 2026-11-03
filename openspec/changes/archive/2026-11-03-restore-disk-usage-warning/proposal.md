## Why

The desktop header displays disk usage at every capacity and always uses muted text, so it no longer warns users when local backup space is running out. Restore the visibility and danger thresholds lost during the application-shell refactor before release.

## What Changes

- Show the desktop header disk summary only when used space is strictly greater than 85 percent of total space.
- Use the theme's muted text color through 95 percent and its danger text color strictly above 95 percent.
- Retain the existing one-line summary, localized numbers and explanatory tooltip.
- Delete the unused `app.diskUsage.compactSummary` key from all six dictionaries.
- Cover ordinary, elevated and critical usage, exact threshold boundaries and both themes. Preserve the mobile menu's always-available disk summary and existing danger threshold.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `frontend-application`: Specify desktop disk-warning visibility and theme-aware severity, while preserving mobile disk information.

## Impact

- Application presentation and tests: `frontend/src/app/ui/ApplicationScreen.tsx` and `ApplicationScreen.test.tsx`.
- Localization: the six dictionaries in `frontend/src/shared/i18n/locales/` and their existing completeness checks.
- Browser verification: disk-warning scenarios in `frontend/browser/scenarios/appShell.spec.ts` using the existing HTTP fixture infrastructure.
- Related verification repair: the existing enlarged-text SMTP browser scenarios wait for test-delivery feedback and dismiss its transient notice before clicking Save, so the notice cannot intercept the action.
- Existing theme roles and disk-formatting code are reused. No new dependency, HTTP contract change, schema change or caller migration is required; the deleted translation key has no callers.
- This change follows the root `AGENTS.md` and `frontend/AGENTS.md`.

## Out of Scope

- Changing backend disk measurement, instance-health thresholds, API operations or startup request handling.
- Hiding mobile disk information below the desktop warning threshold.
- Restoring the old two-line summary layout, adding warning settings or redesigning the header.
- Cleaning unrelated translations or changing architecture and agent conventions.
