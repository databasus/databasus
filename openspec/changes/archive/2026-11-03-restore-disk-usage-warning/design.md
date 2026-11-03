## Context

See [proposal.md](proposal.md) for motivation and [the delta spec](specs/frontend-application/spec.md) for observable behavior. This design records the exact threshold semantics and the choice to remove the unused compact translation rather than restore its layout.

- `frontend/src/app/ui/ApplicationScreen.tsx:161` renders the header summary whenever `diskUsage` exists and assigns a permanent `text-muted` class at line 164.
- `frontend/src/widgets/app-shell/ui/AppShell.tsx:47` limits desktop header actions to nonmobile viewports and renders the supplied auxiliary slot at line 65.
- `frontend/src/widgets/app-shell/ui/SidebarComponent.tsx:78` already compares byte usage with `0.95`; its mobile section renders available disk information at line 162 and switches between semantic text colors at line 166.
- `frontend/src/shared/theme/ThemeCssVariables.tsx:14` derives the muted role from the theme; line 18 derives the danger role from `colorErrorText`.
- `frontend/src/shared/i18n/locales/en.ts:84` defines `compactSummary`, whose only remaining occurrences are definitions in the six dictionaries. Current header and mobile consumers use `summary`.
- `frontend/src/app/ui/ApplicationScreen.test.tsx:15` checks the returned capacity and percentage only at 90 percent. It neither asserts color nor exercises ordinary usage.
- `frontend/src/app/model/useApplicationStartup.ts:25` starts the disk request separately from essential startup requests. Waiting for essential startup alone therefore cannot prove that a hidden disk summary has processed its response.
- `v3.64.0`'s `frontend/src/widgets/main/MainScreenComponent.tsx` used strict `> 0.85` and `> 0.95` comparisons. The replacement introduced in `8ab92e21` lacks these header checks.

The approach follows root `AGENTS.md`, `frontend/AGENTS.md`, and the ownership and theme rules in `docs/architecture.md:35`, `docs/architecture.md:39` and `docs/architecture.md:42`. Existing app composition, widget slots and theme ownership remain sufficient; no architecture documentation change is needed.

## Goals / Non-Goals

**Goals:** Apply the warning policy at the existing header composition point, reuse semantic theme colors, and make tests distinguish a completed low-usage response from an unprocessed response.

**Non-Goals:** Move request ownership, add a reusable warning abstraction, alter the shell's layout or change mobile information availability. Product scope is listed in the proposal.

## Decisions

### Keep the header policy in ApplicationScreen

Derive visibility and severity directly from available byte counts in `ApplicationScreen`, then supply the existing auxiliary slot. Guard visibility with `diskUsage` and a strict ratio comparison above `0.85`; choose danger above `0.95` and muted otherwise. Keep comparisons independent of the formatted percentage, so values such as 85.01 and 95.01 percent retain the correct behavior when displayed with one decimal place.

Putting the policy in `AppShell` would give a generic layout widget knowledge of disk data. Changing startup to suppress low usage would also remove information needed by the mobile menu. A shared policy module would add another unit for two trivial comparisons without reducing the scope of this repair.

### Reuse semantic colors and the existing single-line summary

Use `text-muted` and `text-danger`, as the mobile summary already does. Keep `getDiskUsageValues`, `app.diskUsage.summary` and the tooltip. Semantic roles follow the current theme and avoid introducing independent color values in a consumer.

Raw red/gray values duplicate theme ownership. Reinstating the old two-line `Trans` rendering would change geometry beyond the warning regression. Delete `compactSummary` from `en`, `ru`, `es`, `pt`, `fr` and `zh` together rather than retaining an unused key or an alias. No callers require migration.

### Verify the real screen and rendered browser colors

Extend `ApplicationScreen.test.tsx` with a common HTTP setup and the requested scenarios:

- `hides the disk summary at 50 percent`
- `shows the disk summary in the muted color at 90 percent`
- `shows the disk summary in the danger color at 96 percent`

Also cover 85, 86 and 95 percent and fractions just above 85 and 95 that round to the boundary. Use the existing real screen renderer with explicit light/dark options and typed disk responses. Select the desktop header to avoid confusing its summary with drawer text. For hidden-summary assertions, settle the disk response inside the renderer's async update boundary before checking absence; waiting for another startup response or merely observing that the disk request started is insufficient. Do not use fixed sleeps.

Remove the test's redundant inner `ThemeProvider` and move its environment directive to the first line while editing it, following the existing renderer contract. Keep the original capacity/percentage assertion in a visible-summary scenario.

Add focused production-browser checks to `frontend/browser/scenarios/appShell.spec.ts`: verify muted/danger computed colors against the active theme roles at 90/96 percent, switch themes while the summary is visible, and verify mobile information at 50/95/96 percent. Replace the explicit disk fixture for each scenario rather than adding a duplicate method/path entry. Browser checks also exercise header reflow with the visible warning at 769 pixels and 200 percent text. Existing application fixtures and the browser runner remain the test infrastructure.

Only checking the presence of `text-danger` cannot establish the rendered color because DOM screen tests do not load the production stylesheet. Only checking the text at 90 percent repeats the existing coverage gap.

## Risks / Trade-offs

- A low-usage test can pass before disk data arrives. Mitigate by completing that specific response and flushing its React updates before asserting absence; prove that the test fails on the unfixed screen.
- Rounded percentages can hide a comparison error near a boundary. Mitigate with byte counts representing 85.01 and 95.01 percent as well as exact-boundary cases.
- A visible one-line warning can consume header space. Mitigate with the production browser reflow assertion just above the mobile breakpoint and enlarged text.
- Removing a key from only some dictionaries creates drift. Delete it in all six dictionaries and run dictionary placeholder/completeness tests and frontend type checks.
- The required browser run exposed a race in the existing enlarged-text French/Chinese SMTP scenarios: the asynchronous delivery-success notice can appear during the Save click, and hovering it pauses dismissal while it overlaps the control. Wait for the observed success notice, dismiss it through its close control, and assert its absence before continuing the existing Save and request-count checks. This repairs test synchronization without weakening geometry assertions, forcing clicks or changing production behavior.

## Migration Plan

No persisted data or API migration is needed. Ship the frontend changes and all six dictionary deletions together. If rollback is needed, revert the frontend change as a unit; no compatibility layer is required.
