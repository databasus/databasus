## Context

See [proposal.md](proposal.md) for motivation and scope. This is a complete implementation replacement with unchanged product requirements, so delta specs are deliberately skipped.

The current application installs Ant Design configuration and feedback above the router in `frontend/src/app/ui/App.tsx:1`. Its session-generation effect destroys programmatic dialogs. Screen tests reproduce that context separately in `frontend/src/shared/lib/test/ScreenConfiguration.tsx:1`; both paths must migrate together.

`frontend/src/shared/theme/getAntdTheme.ts:5` owns library tokens. `frontend/src/shared/theme/ThemeCssVariables.tsx:10` derives the application's semantic CSS roles from them, and `frontend/src/index.css:52` exposes those roles to Tailwind. `frontend/lint/styleChecks.ts:20` names the approved color owners. Removing the library therefore requires moving token ownership and updating its checks, not just replacing imports.

`frontend/src/shared/i18n/localeSettings.ts:2` couples language settings to library locale objects. The actual locale table includes English and six translations. Date and time conventions also depend on the browser's region; `frontend/src/entities/schedules/ui/ScheduleDetailsFields.tsx:32` uses the shared clock-format helper rather than deriving the clock from the selected interface language.

There are direct dependencies on internal markup in production and tests: `frontend/src/shared/ui/wrappingSelectClassName.ts:1`, `frontend/src/shared/ui/useDropdownFocus.ts:12`, `frontend/src/shared/ui/ResourceTransferDialog.tsx:157`, and `frontend/src/entities/schedules/ui/ScheduleEditor.test.tsx:33`. These include real narrow-layout repairs, so deleting them requires equivalent behavior in the replacement.

The existing production browser suite already supplies operation-typed HTTP fixtures and explicitly controlled visual references (`frontend/browser/HttpFixture.ts:1`, `frontend/playwright.config.ts:1`). Its eight committed images cover SMTP editing and storage cleanup rather than the full interface. The real-backend suite has separate selectors to update (`e2e/browser/scenarios/application_screens.py:71`, `e2e/browser/scenarios/physical_backup_chain.py:63`).

The change follows root `AGENTS.md` and `frontend/AGENTS.md`. It intentionally replaces the latter's Ant Design and icon conventions. Layer boundaries, public APIs, state ownership and screen-test composition remain as specified in `docs/architecture.md:332`. No backend HTTP contract changes are needed.

## Goals / Non-Goals

**Goals:**

- Finish with one component library and no Ant Design dependency, direct or transitive: `antd` and every `@ant-design/*` package must be absent from project manifests, lockfiles, the resolved dependency graph and the production build. Remove their runtime, icon, type, style and test-selector dependencies.
- Preserve the interface's visual identity and every existing interaction, including asynchronous response acceptance, permission restrictions and session invalidation.
- Keep layout readable as semantic HTML with Tailwind; centralize shared control appearance without creating a parallel component framework.
- Make every affected surface independently traceable to automated coverage and before/after browser evidence during migration; retain that history in this change after test consolidation.
- Give supported agents one maintained copy of the official skills and clear rules for loading them.

**Non-Goals:**

- See the proposal's exclusions. In particular, form-state, request-state, routing and chart architecture are not migration targets.
- Browser evidence does not claim that synthetic provider responses establish successful delivery to an external account. Real integrations remain covered by their existing suites.

## Decisions

### 1. Complete one migration, with dependency-ordered implementation

Capture the baseline before changing the application providers. Then replace infrastructure, shared interactions and each consumer family, and finally remove the old packages and dead files. Ant Design may remain temporarily while consumers are being edited inside the unfinished change; it is absent from the completed result. No pilot release, optional follow-up migration or compatibility layer remains.

The user chose the full scope. A pilot-only change would leave most of that scope undone. Replacing every file in an unstructured pass would make regressions harder to locate; dependency ordering preserves the full scope while keeping verification useful.

### 2. Keep existing layer boundaries and use HTML for layout

Pages and features compose ordinary `main`, `section`, `div`, `form`, `label` and table elements with Tailwind. Mantine supplies controls and interactions where its behavior is useful. Direct Mantine imports are permitted; an application-wide alias for each vendor component is unnecessary.

Keep the current shared controls when they own a repeated interaction, such as confirmation, transfer, clipboard paste or advanced-field presentation. Add a focused shared control only when real consumers require the same behavior. Domain models and save policy stay with their existing owner. This follows `docs/architecture.md:340` and the FSD skill's extraction rule.

Replacing every layout element with `Box`, `Stack`, `Group`, `Text` or a local equivalent would add library vocabulary without solving the task. A universal vendor-neutral UI API would duplicate the vendor API and still have to represent vendor-specific behaviors. Writing all interactive primitives ourselves would add keyboard, focus, positioning and accessibility maintenance that the application does not need to own.

```text
app --> pages --> widgets --> features --> entities --> shared
                consumers compose:
                  HTML + Tailwind
                  Mantine controls
                  focused shared/ui interactions

shared/theme tokens --> Mantine configuration
                    --> semantic Tailwind roles
```

### 3. Pin a compatible package set and remove the old icon dependency

The planning-time registry check on 2026-11-07 found Mantine `9.7.5` compatible with the repository's React 19.2 line. Use matching exact versions of `@mantine/core`, `@mantine/hooks`, `@mantine/dates` and `@mantine/notifications`. Use `lucide-react` `1.70.0` for standard icons. Record the resolved versions in the lockfile; do not mix Mantine package versions.

Do not add `@mantine/form` merely to replace `Form.Item`: current field values and save policies already belong to controlled React state. Vendor its official skill for tasks that actually use that package. Keep dayjs and the existing schedule/time calculations; adapt only their UI boundaries.

Lucide supplies standard icons and permits complete removal of `@ant-design/icons`. Retaining the old icon package would leave the migration incomplete. Adding another icon system as well would provide no benefit. Preserve existing provider, engine, brand and sidebar assets; match glyph size, alignment, meaning and accessible names when standard icons change.

### 4. Make application tokens the source of truth

Keep theme infrastructure in `shared/theme`, as `docs/architecture.md:366` requires. Introduce an application-owned token sheet and Mantine theme configuration there. Preserve existing semantic role names in Tailwind and their light/dark values, local Jost fonts, CJK fallback, focus indicators, compact control geometry, surfaces and scrollbar roles.

Use Mantine theme defaults, public CSS variables and `classNames` for component parts. CSS modules or application classes own hover, focus, responsive and wrapping rules. Define CSS layer order explicitly so Mantine styles sit below application component rules and Tailwind utilities. Preserve Databasus token values while configuring these layers and component defaults.

Retain the existing Tailwind Preflight in the base layer and put Mantine styles above that layer. This preserves native-element defaults while allowing library component styles to win over the reset. Preserve existing `.dark` consumers and system-theme synchronization while binding Mantine to the same resolved mode. Keep providers mounted during theme and language changes.

Update the color-owner list and fixtures rather than exempting `shared/theme` or CSS broadly. Delete `getAntdTheme` and the Ant Design token bridge after their consumers move. Keeping token derivation tied to the removed library is impossible; changing the application palette would constitute a redesign. Global `unstyled` mode would require restyling every primitive and is unnecessary for the agreed scope.

### 5. Preserve form and selection contracts at the boundary

Replace decorative `Form.Item` with Mantine input labels/descriptions/errors or focused existing field composition. Keep a single actual `form` where a submission interaction exists, avoid nested forms in embedded editors, and make non-submit actions explicit buttons. Preserve validation timing, trimmed values, dirty baselines, conditional fields, tested-settings approval, stored-secret omission and protection against duplicate requests.

Use ready-made `Select`, `MultiSelect`, `Autocomplete`, `TagsInput` or `NativeSelect` when their behavior fits. For long selected labels, first use Mantine's button-target `ComboboxPopover` with a wrapping trigger. Use `Combobox` primitives only when required content or interaction cannot be expressed by that ready component. The public primitives handle option navigation and selection; do not rebuild keyboard handling around DOM searches. Keep model value types intact, converting component-specific empty values only at the control boundary.

Resource menus keep one vertical scroll owner in the options list, capped at 240px and allowed to shrink inside a popup bounded by the dynamic viewport with 8px edge clearance. Search and creation stay outside that scroll owner. Public positioning middleware may shift the popup along either axis, and placement can adapt while it is open. Focusing search does not scroll the page. A viewport bound without internal scrolling would leave the final options unreachable; scrolling the complete popup would hide search and creation. Extra application measurements and resize listeners are unnecessary.

Native selects remain suitable for simple choices but cannot provide every existing searchable, tagged or richly rendered selection. Forcing every choice through an ordinary select would remove current functionality. Moving all forms into a new state manager would risk draft and stale-response behavior without helping the library replacement.

Dates and schedules retain the existing regional order, 12/24-hour clock and local/UTC conversions. Mantine date strings and component empty values must be adapted explicitly. Cover weekday/month-day rollover, daylight-saving boundaries, calendar clearing and time-of-day entry without rewriting the schedule algorithms.

### 6. Preserve overlay and feedback lifecycles

Use Mantine `Modal`, `Drawer`, `Menu`, `Tooltip` and `Popover` while preserving current placement, dimensions, close restrictions, pending states and scrolling. Existing anchored confirmations remain anchored; modal confirmations remain modal. Retain declarative ownership where it already exists, and replace the two programmatic confirmation flows with state owned by their current screen. Destroy or close overlays when their owning session becomes obsolete.

Nested creation dialogs must return focus to their parent, and menus/calendar dropdowns must remain reachable above the correct dialog. Escape closes the active popup before its parent where the current interaction calls for it. Preserve focus restoration when a caller unmounts an overlay immediately, rather than assuming a closing animation will always run.

Introduce a small shared feedback hook for the existing repeated success/error notice calls, backed by Mantine notifications. Preserve notice ownership, original deadlines and creation-time language; render persistent errors from keys or API error data under the current locale. Account changes clear obsolete notices and prevent late callbacks from creating new ones. Do not remount the provider to implement locale changes.

A global replacement modal service would add lifecycle machinery for only two callers. Browser-native confirmations would lose the existing presentation and translated interaction. Using raw Mantine notification calls everywhere would leave cleanup and language-lifetime policy distributed across consumers.

### 7. Keep current table behavior without a new table framework

Use Mantine table presentation or semantic table markup and page-owned state for the capabilities actually present: chronological sorting, filters, paging/load-more behavior, row actions, details and narrow rendering. Preserve sort direction, selected filters and date limits. Map table column models locally where they remain useful; remove Ant Design column types.

The current logical table has client sorting and external pagination (`frontend/src/pages/databases/ui/logical-backups/LogicalBackupsTable.tsx:48`, `:107`). Mantine's table is not a one-for-one data-grid replacement, so swapping tags alone would lose behavior. A new generic grid or another data-table dependency is unnecessary for the observed requirements.

### 8. Vendor official skills once and document their activation

Vendor `mantine-form`, `mantine-combobox` and `mantine-custom-components` from `https://github.com/mantinedev/skills` at revision `607846dbc328b6ddc85106d8aad3c080f9f57a1d`, verified during planning. Preserve each `SKILL.md` and all referenced resources verbatim. Include the upstream MIT license and an `upstream.json` in each directory with source paths and SHA-256 hashes, following the repository's existing FSD provenance approach. Check referenced files, discovery paths and hashes with local assertions during vendoring; a permanent test that only repeats those manifests is unnecessary.

Store the skills under `.agents/skills/<name>/`. Codex discovers that directory directly. Retain and verify `.claude/skills -> ../.agents/skills`, which already exposes every new skill to Claude Code. Do not add a second Codex copy or a symlink inside `.codex` that its discovery does not require. This preserves `agent-workflow`'s single-source requirement.

`frontend/AGENTS.md` defines when to load each skill:

| Skill                       | Load before                                                                                                                     |
| --------------------------- | ------------------------------------------------------------------------------------------------------------------------------- |
| `mantine-form`              | Work involving `@mantine/form` values, validation, lists, controlled/uncontrolled mode or form context                          |
| `mantine-combobox`          | Searchable/multiple/custom selection, option rendering, filtering or `Combobox`/`ComboboxPopover` interaction                   |
| `mantine-custom-components` | Creating Mantine-integrated shared components or changing their Styles API, factories, theme extension or CSS-variable contract |

The repository-owned frontend-design skill continues to govern ordinary presentation and points to the applicable Mantine skill when needed. Project requirements take precedence over upstream examples: translated literals, direct palette colors, unconditional uncontrolled forms and factory-based wrappers are not adopted merely because an example uses them.

Keep official documentation URLs and version guidance in the frontend design documentation: `https://mantine.dev/guides/llms/`, `https://mantine.dev/llms.txt` and linked per-page Markdown. Skills describe patterns; the documentation index supplies current API details. Consult documentation matching the installed major and verify minor-specific APIs. Do not vendor `llms-full.txt` or install a separate documentation MCP server for this migration: the required pages are available directly and the existing browser MCP already covers inspection.

Local copies per agent could drift, global installs would not travel with the repository, and copying only `SKILL.md` would break relative references.

### 9. Preserve behavior assertions while removing vendor DOM coupling

Update the production providers and real test configuration together. During the unfinished migration, add Mantine alongside the actual Ant Design providers so unmigrated consumers remain functional; remove the Ant Design providers after the last consumer moves. This temporary composition is not a compatibility API and must not ship. Keep finite typed HTTP fixtures, request recording, controlled responses, strict lifecycle cleanup and actual application screens. Implement only browser-platform shims the document emulator lacks. Disable transitions through supported test configuration when necessary; do not replace Mantine components with fakes or remove browser portal coverage.

Prefer accessible labels and roles, scoped within named regions/dialogs. Assert selected values, `aria-expanded`, `aria-selected`, pending/disabled state, visible errors and outgoing requests. Where a visual property matters, check geometry/computed presentation in the browser rather than a vendor class. Remove `.ant-*` selectors; do not translate them into `.mantine-*` selectors. An application-owned test ID is reserved for a surface without a meaningful accessible query, with the reason stated locally.

Keep race, late-response, cleanup, locale-switch, permission and secret-handling scenarios. Rewriting tests only until they pass would hide regressions; mocking our components or services would stop testing the actual integration.

### 10. Establish before/after evidence for every surface

The matrix below is a coverage ledger, not a replacement for tests. Before provider/theme changes, add or extend production browser scenarios with deterministic fixtures for all its rows. Assign each row a stable ID in scenario titles until migration acceptance is complete; decision 11 governs their later removal. Exercise the same scenario and data before and after migration, and retain reviewable reference/diff evidence under the existing ignored browser-results location. Use actual screenshot assertions with unique surface, variant, state and context names, and the existing explicit reference-update mechanism. Before replacing a consumer, preserve its source, fixtures, lockfile and capture configuration alongside the original evidence so the view can be reproduced. Classify failures as confirmed product defects, test mistakes or unresolved causes. Correct test mistakes and investigate unresolved causes before migrating that consumer. Confirmed old defects may be repaired in the replacing Mantine implementation, with their failing original evidence retained. Independent geometry assertions can accumulate failures while later views are captured; they must still fail the run. This changes implementation order only: every assertion and inspection remains a completion requirement.

For each row, inspect desktop light and dark presentations and a 320 CSS-pixel layout with a long supported translation. Check 200 percent actual text enlargement on every distinct screen/dialog layout; shared provider variants may reuse the same layout assertion but must each be opened and inspected. Exercise Chinese fallback and every supported locale in the relevant shared control tests. Also check intermediate desktop/tablet widths when a layout changes ownership or column structure. Do not multiply all locales, states and widths into an unnecessary full Cartesian product.

The implementing agent operates Chrome DevTools MCP against the checkout's local app according to the run-app skill. For external callbacks, permission states and provider outcomes that cannot be reached with local seed data, or when the shared development ports belong to another checkout, the agent uses the same built production app in the isolated Playwright environment with finite fixtures and inspects the captured screen; no external account, user intervention or test-only production route is required. Every row needs visual inspection, and every interactive row needs a keyboard/interaction pass. Browser automation supplements that inspection with repeatable assertions.

For overlays check title, focus containment/restoration, Escape/outside-click rules, pending closure policy, nested stacking and scroll ownership. For fields check labels/help/errors and long values; for lists check initial loading, empty, filtered-empty, data and recoverable failure states where applicable. Include language/theme changes with drafts, visible errors and pending work. A failed load must not silently look empty.

Keep baseline and final images distinct during review. Small changes to standard icon strokes or unavoidable library rendering may be accepted only after inspecting the actual difference and confirming that geometry, meaning and behavior are preserved. Fix unintended differences before updating references. Never raise screenshot thresholds, automatically accept the whole suite, or treat a new baseline as proof that the design was preserved.

At apply time, expand the ledger if new surfaces are found. Record a reason if a row is truly the same reachable surface as another; do not silently skip it. A row is complete only when its automated scenario passes and its before/after evidence has been inspected. Evidence paths, context and outcome are recorded during implementation, not fabricated in this proposal.

### 11. Consolidate tests after migration acceptance

The full matrix is a migration acceptance requirement. It does not require permanently retaining one screenshot or executable scenario for every row. Finish tasks 2.1–2.13 and 18.5–18.6, including passing assertions and inspection of every required before/after view, before starting section 19. Do not remove or consolidate an unresolved failing scenario to make acceptance pass.

Review tests added or expanded by this change and record their disposition in this document's implementation evidence: the behavior checked, its retained test file and descriptive scenario title, and the reason for each removal or consolidation. Preserve unique behavior coverage, including regressions for defects discovered during migration. Merge repeated assertions into existing scenarios when they establish the same behavior. Screen tests can own draft, validation and request semantics; keep production-browser assertions for actual keyboard interaction, portal stacking, focus, layout, text enlargement, scroll ownership and chart geometry. A document emulator cannot replace those browser checks. Provider-specific fields, secret handling and conditional behavior remain covered even when their screenshots share a layout.

Retain a small visual regression set chosen by distinct layout risk: an application shell/list, a complex form, nested overlays, narrow layouts with long translations and enlarged text, and light/dark styling. Record the retained scenarios and the reason each needs a screenshot. Avoid multiplying equivalent provider layouts or every state across all contexts. Prefer observable assertions for overflow, visibility, retained input, focus and requests where they express the requirement; use screenshots for appearance that those assertions do not establish. Preserve unique existing regression coverage and the existing browser runner's explicit update, missing-reference and zero-pixel failure rules on both supported architectures. Remove redundant screenshots and their unused references together; never raise thresholds or accept unexplained differences to shorten the suite.

After acceptance, remove matrix IDs from executable test titles, fixtures, capture names and reference filenames. Use domain names and behavior descriptions instead. Preserve the completed matrix, original scenario identities and evidence paths here, and map each row to its retained scenarios or the rationale for consolidating its temporary checks. Before renaming tests, retain the pre-cleanup scenario sources and capture configuration with the existing evidence so the historical captures remain reproducible. Rename retained references without changing their pixels when only identity changes; if regeneration is necessary, inspect the differences and use the explicit update command. The historical ID-presence check in task 18.6 is a pre-cleanup check, not a permanent test requirement.

Remove helpers, fixtures and capture/context collection code used only for migration evidence when they have no remaining consumers. Keep shared browser setup, typed HTTP fixtures and useful failure diagnostics. Scope cleanup to work owned by this change; do not format, delete or rewrite another agent's concurrent edits. Record a concise final coverage summary here before running the final checks on the cleaned suite.

Keeping the entire migration capture matrix would make routine copy and style changes update many equivalent images. Deleting all added tests would lose meaningful regression coverage. Removing IDs before acceptance would break traceability, and weakening screenshot comparisons would hide differences. Consolidation after acceptance preserves the evidence and useful checks while reducing ongoing maintenance.

## Risks / Trade-offs

- **Control geometry differs:** calibrate the central theme against the baseline before correcting consumers; verify wrapping labels and action rows, not only resting screenshots.
- **The selected-value input cannot wrap:** use a ready button-target selection before introducing a custom primitive; preserve full value visibility at narrow widths.
- **Overlay behavior differs:** explicitly cover nested creation, calendar/menu portals, immediate unmount and session replacement. Visual similarity alone does not establish focus correctness.
- **Date APIs represent values differently:** keep model/timezone logic unchanged and assert actual submitted values under regional and daylight-saving cases.
- **Feedback leaks across accounts:** own notification IDs and confirmation state by the current editing session and account generation.
- **The baseline misses a hidden branch:** enumerate provider variants, permission states and overlays before migration and add discovered rows before declaring completion.
- **Reference updates hide a regression:** inspect old/new/diff evidence and retain strict ordinary-run behavior; explain accepted differences.
- **The change is large:** keep work within the current checkout and migrate in consumer families. Preserve existing behavior coverage and complete all migration assertions before consolidating tests under decision 11. Do not split remaining consumers into an unapproved follow-up.
- **Cleanup can erase evidence or unique coverage:** begin only after migration acceptance, record retained coverage and removal reasons, preserve reproducible historical evidence, and verify the cleaned suite before review.
- **Instructions drift from code:** update the owning conventions and search affected documentation in the same change. Archived OpenSpec history remains historical.
- **Only one host architecture is available locally:** report local browser evidence accurately; existing CI provides the other native architecture's evidence. Do not claim an ARM run from an amd64 result or change the supported matrix.

## Migration Plan

1. Inventory and cover the matrix against the unchanged application. Capture deterministic automated baselines and agent-operated before views.
2. Vendor and verify the skills and update the intended component conventions. Introduce the matching dependency set, tokens, CSS layers, locale/date configuration and providers.
3. Migrate shared interactions, schedules and connection controls, then authentication/shell, provider editors, database operations and administration. Keep each consumer family's behavior and browser assertions intact.
4. Update the real-backend browser selectors, delete all old providers/theme bridges/selectors/packages and reconcile documentation. Verify that `antd` and every `@ant-design/*` package are absent from direct and transitive dependencies, project manifests and lockfiles, the resolved dependency graph and the production build. No server API or persisted data migration is required.
5. Complete every ledger row, inspect each visual difference, deliberately update only reviewed references and pass the full migration browser suite. Preserve the completed evidence and scenario identities before cleanup.
6. Consolidate tests under decision 11, remove migration IDs and unused capture infrastructure, and record the final coverage summary. Run formatting, lint, build and the final verification commands from `tasks.md` on the cleaned result. Stop any local app process started for inspection; leave a reused user-owned app running according to the run-app skill.
7. Resolve mandatory review findings and leave the complete implementation ready to commit. This change does not commit or publish itself.

Deployment uses the normal frontend build. A rollback restores the complete previous frontend artifact and dependency lock together; no database rollback is needed. Do not ship the intermediate mixed-library state.

## Browser Verification Matrix

The rows below are pending. Source paths identify owners, not a mandate to create new components. Reuse existing browser scenario files by owner and add narrowly scoped files when a surface has no coverage. During apply, record evidence and inspection outcomes for each row in this section. After section 19 cleanup, keep these historical entries and their mapping to retained coverage or consolidation reasons; executable tests no longer need the row IDs. Shared presentation checks use real consumers rather than a new production showcase route.

### Shared presentation

Existing browser owners: `appShell.spec.ts`. Add scenarios there or in a focused file beside them.

| ID   | Surface and required variants                                     | Additional checks                                                            | Source owner under frontend/src               | Evidence                              |
| ---- | ----------------------------------------------------------------- | ---------------------------------------------------------------------------- | --------------------------------------------- | ------------------------------------- |
| UI01 | Language and theme controls, including their menus                | All supported locales; system/light/dark; menu focus and current choice      | `shared/ui/LanguageThemeControlComponent.tsx` | Inspected; see `visual-evidence.json` |
| UI02 | Advanced disclosure, labelled field, help tooltip and inline code | Expanded state; help/error associations; long labels; technical text         | `shared/ui/AdvancedFieldRowComponent.tsx`     | Inspected; see `visual-evidence.json` |
| UI03 | Shared destructive and ordinary confirmation dialogs              | Cancel/confirm/Escape; pending action; immediate-unmount focus restoration   | `shared/ui/ConfirmationComponent.tsx`         | Inspected; see `visual-evidence.json` |
| UI04 | Clipboard paste fallback dialog                                   | HTTP/non-secure fallback; empty/filled/cancel; focus restoration             | `shared/ui/ClipboardPasteModalComponent.tsx`  | Inspected; see `visual-evidence.json` |
| UI05 | Error, retry, loading, empty and transient feedback surfaces      | Distinct states; translated persistent errors; notice expiration and cleanup | `shared/ui/LoadFailure.tsx`                   | Inspected; see `visual-evidence.json` |

### Authentication and shell

Existing browser owners: `auth.spec.ts / appShell.spec.ts`. Add scenarios there or in a focused file beside them.

| ID   | Surface and required variants                           | Additional checks                                                                         | Source owner under frontend/src                   | Evidence                              |
| ---- | ------------------------------------------------------- | ----------------------------------------------------------------------------------------- | ------------------------------------------------- | ------------------------------------- |
| AU01 | Sign-in screen                                          | Password visibility; validation; pending/error; OAuth choices; Turnstile enabled/disabled | `pages/auth/ui/SignInComponent.tsx`               | Inspected; see `visual-evidence.json` |
| AU02 | Registration and first-owner claim screens              | Both entry modes; validation; disabled registration; pending/failure                      | `pages/auth/ui/SignUpComponent.tsx`               | Inspected; see `visual-evidence.json` |
| AU03 | Sign-in code screen                                     | Invalid/expired code; resend; pending; return to password; changed attempt                | `pages/auth/ui/SignInCodeComponent.tsx`           | Inspected; see `visual-evidence.json` |
| AU04 | Password-reset request screen                           | Validation; delivery success/failure; pending; navigation                                 | `pages/auth/ui/RequestResetPasswordComponent.tsx` | Inspected; see `visual-evidence.json` |
| AU05 | Password-reset completion screen                        | Code/password/confirmation errors; pending; completed session transition                  | `pages/auth/ui/ResetPasswordComponent.tsx`        | Inspected; see `visual-evidence.json` |
| AU06 | OAuth sign-in callback screen                           | Pending, success and refused/obsolete attempt                                             | `pages/auth-callback/ui/AuthCallbackPage.tsx`     | Inspected; see `visual-evidence.json` |
| AU07 | Google Drive callback screen                            | Pending, accepted, failed, missing and obsolete authorization attempt                     | `pages/storage-oauth/ui/StorageOAuthPage.tsx`     | Inspected; see `visual-evidence.json` |
| SH01 | Authentication startup loading and retry screen         | No users/users; failed options load and retry; local fonts                                | `pages/auth/ui/AuthPage.tsx`                      | Inspected; see `visual-evidence.json` |
| SH02 | Application startup and empty-workspace screen          | Loading; essential failure/retry; empty; remembered inaccessible workspace                | `app/ui/ApplicationScreen.tsx`                    | Inspected; see `visual-evidence.json` |
| SH03 | Desktop application shell                               | Current navigation; workspace choice; version and disk tooltips; role restrictions        | `widgets/app-shell/ui/AppShell.tsx`               | Inspected; see `visual-evidence.json` |
| SH04 | Mobile navigation drawer                                | Open/close; menu scrolling; theme/language draft preservation; focus return               | `widgets/app-shell/ui/SidebarComponent.tsx`       | Inspected; see `visual-evidence.json` |
| SH05 | Workspace creation dialog and permission-denied variant | Allowed/denied; name validation; pending/error; created selection; focus                  | `app/ui/CreateWorkspaceDialogComponent.tsx`       | Inspected; see `visual-evidence.json` |

### Storage collection and provider forms

Existing browser owners: `storageResources.spec.ts / resourceCollections.spec.ts`. Add scenarios there or in a focused file beside them.

| ID   | Surface and required variants                         | Additional checks                                                                 | Source owner under frontend/src                                              | Evidence                              |
| ---- | ----------------------------------------------------- | --------------------------------------------------------------------------------- | ---------------------------------------------------------------------------- | ------------------------------------- |
| ST01 | Storage list and detail navigation                    | Search; selected resource; permission/refusal; loading/empty/filtered-empty/error | `pages/storages/ui/StoragesComponent.tsx`                                    | Inspected; see `visual-evidence.json` |
| ST02 | Storage creation dialog and type picker               | Name/type; provider changes; closing/reopening; pending test/save                 | `pages/storages/ui/StoragesComponent.tsx`                                    | Inspected; see `visual-evidence.json` |
| ST03 | Local storage creation/editing and read-only details  | Create/edit/detail; tested-settings approval; save/error                          | `features/edit-storage/ui/edit/storages/EditLocalStorageComponent.tsx`       | Inspected; see `visual-evidence.json` |
| ST04 | S3 storage creation/editing and read-only details     | Endpoint/region/bucket; omitted secret; advanced settings; copy                   | `features/edit-storage/ui/edit/storages/EditS3StorageComponent.tsx`          | Inspected; see `visual-evidence.json` |
| ST05 | Azure Blob creation/editing and read-only details     | Authentication fields; omitted key; configured advanced values                    | `features/edit-storage/ui/edit/storages/EditAzureBlobStorageComponent.tsx`   | Inspected; see `visual-evidence.json` |
| ST06 | FTP storage creation/editing and read-only details    | Host/port/path; password omission; test invalidation                              | `features/edit-storage/ui/edit/storages/EditFTPStorageComponent.tsx`         | Inspected; see `visual-evidence.json` |
| ST07 | SFTP storage creation/editing and read-only details   | Password and key authentication; stored-secret omission; address-change notice    | `features/edit-storage/ui/edit/storages/EditSFTPStorageComponent.tsx`        | Inspected; see `visual-evidence.json` |
| ST08 | NAS storage creation/editing and read-only details    | SMB host/port/share; share-name warning; SSL/domain/path; credential handling     | `features/edit-storage/ui/edit/storages/EditNASStorageComponent.tsx`         | Inspected; see `visual-evidence.json` |
| ST09 | Google Drive creation/editing and read-only details   | Credentials; authorize/reauthorize; cancelling; retained draft and token omission | `features/edit-storage/ui/edit/storages/EditGoogleDriveStorageComponent.tsx` | Inspected; see `visual-evidence.json` |
| ST10 | Rclone storage creation/editing and read-only details | Configuration text; advanced values; literal technical characters                 | `features/edit-storage/ui/edit/storages/EditRcloneStorageComponent.tsx`      | Inspected; see `visual-evidence.json` |
| ST11 | Storage transfer dialog                               | Long workspace names; blocked/empty/error states; confirmation and focus          | `pages/storages/ui/StorageTransferDialogComponent.tsx`                       | Inspected; see `visual-evidence.json` |
| ST12 | Storage deletion confirmation                         | Usage restrictions; destructive action; failure and draft/selection retention     | `pages/storages/ui/StorageComponent.tsx`                                     | Inspected; see `visual-evidence.json` |
| ST13 | Storage cleanup warning dialog                        | Current cleanup counts; wait action focus; deletion pending/error                 | `pages/storages/ui/StorageComponent.tsx`                                     | Inspected; see `visual-evidence.json` |

### Notifier collection and provider forms

Existing browser owners: `notifierResources.spec.ts / smtp.spec.ts / resourceCollections.spec.ts`. Add scenarios there or in a focused file beside them.

| ID   | Surface and required variants                              | Additional checks                                                                  | Source owner under frontend/src                                                | Evidence                              |
| ---- | ---------------------------------------------------------- | ---------------------------------------------------------------------------------- | ------------------------------------------------------------------------------ | ------------------------------------- |
| NT01 | Notifier list and detail navigation                        | Search; selection; permission/refusal; loading/empty/filtered-empty/error          | `pages/notifiers/ui/NotifiersComponent.tsx`                                    | Inspected; see `visual-evidence.json` |
| NT02 | Notifier creation dialog and type picker                   | Name/type; provider changes; close/reopen; test/save state                         | `pages/notifiers/ui/NotifiersComponent.tsx`                                    | Inspected; see `visual-evidence.json` |
| NT03 | SMTP notifier creation/editing and read-only details       | Stored password omission; advanced settings; validation; successful test retention | `features/edit-notifier/ui/edit/notifiers/EditEmailNotifierComponent.tsx`      | Inspected; see `visual-evidence.json` |
| NT04 | Telegram notifier creation/editing and read-only details   | Custom endpoint; bot token; destinations; address change; literal errors           | `features/edit-notifier/ui/edit/notifiers/EditTelegramNotifierComponent.tsx`   | Inspected; see `visual-evidence.json` |
| NT05 | Webhook notifier creation/editing and read-only details    | Methods; dynamic headers; validation; payload and secret preservation              | `features/edit-notifier/ui/edit/notifiers/EditWebhookNotifierComponent.tsx`    | Inspected; see `visual-evidence.json` |
| NT06 | Slack notifier creation/editing and read-only details      | Webhook/destination fields; test/save; secret omission                             | `features/edit-notifier/ui/edit/notifiers/EditSlackNotifierComponent.tsx`      | Inspected; see `visual-evidence.json` |
| NT07 | Discord notifier creation/editing and read-only details    | Webhook fields; test/save; secret omission                                         | `features/edit-notifier/ui/edit/notifiers/EditDiscordNotifierComponent.tsx`    | Inspected; see `visual-evidence.json` |
| NT08 | Teams notifier creation/editing and read-only details      | Webhook fields; validation; test/save; secret omission                             | `features/edit-notifier/ui/edit/notifiers/EditTeamsNotifierComponent.tsx`      | Inspected; see `visual-evidence.json` |
| NT09 | Mattermost notifier creation/editing and read-only details | Endpoint/channel fields; validation; test/save; secret omission                    | `features/edit-notifier/ui/edit/notifiers/EditMattermostNotifierComponent.tsx` | Inspected; see `visual-evidence.json` |
| NT10 | Notifier transfer dialog                                   | Long workspace names; blocked/empty/error states; current target and focus         | `pages/notifiers/ui/NotifierTransferDialogComponent.tsx`                       | Inspected; see `visual-evidence.json` |
| NT11 | Notifier deletion confirmation                             | Usage restrictions; destructive action; pending/failure and retained selection     | `pages/notifiers/ui/NotifierComponent.tsx`                                     | Inspected; see `visual-evidence.json` |

### Database creation and connection settings

Existing browser owners: `databaseResources.spec.ts / databaseConnection.spec.ts`. Add scenarios there or in a focused file beside them.

| ID   | Surface and required variants                               | Additional checks                                                             | Source owner under frontend/src                                                 | Evidence                              |
| ---- | ----------------------------------------------------------- | ----------------------------------------------------------------------------- | ------------------------------------------------------------------------------- | ------------------------------------- |
| DB01 | Database collection and selected-detail shell               | Search; permission/refusal; data/empty/error/loading; tab selection           | `pages/databases/ui/DatabasesComponent.tsx`                                     | Inspected; see `visual-evidence.json` |
| DB02 | Database creation dialog and engine/backup-method choices   | Engine and logical/physical cards; wizard widths; close restrictions          | `pages/databases/ui/CreateDatabaseComponent.tsx`                                | Inspected; see `visual-evidence.json` |
| DB03 | PostgreSQL logical connection creation and editing          | Connection string; TLS/certificates; schema/table selection; credentials      | `features/database-connection/ui/EditPostgresqlLogicalConnectionComponent.tsx`  | Inspected; see `visual-evidence.json` |
| DB04 | PostgreSQL physical connection creation and editing         | TLS/certificates; privileges; credential and connection-test state            | `features/database-connection/ui/EditPostgresqlPhysicalConnectionComponent.tsx` | Inspected; see `visual-evidence.json` |
| DB05 | MySQL connection creation and editing                       | Version/options; connection string; credential omission; test state           | `features/database-connection/ui/EditMySqlCompatibleConnectionComponent.tsx`    | Inspected; see `visual-evidence.json` |
| DB06 | MariaDB connection creation and editing                     | Engine identity; options; connection string; credential omission              | `features/database-connection/ui/EditMySqlCompatibleConnectionComponent.tsx`    | Inspected; see `visual-evidence.json` |
| DB07 | MongoDB connection creation and editing                     | URI/replica/auth options; credentials; test diagnostics                       | `features/database-connection/ui/EditMongodbConnectionComponent.tsx`            | Inspected; see `visual-evidence.json` |
| DB08 | SSH tunnel fields in creation, settings and restore targets | Password/key modes; passphrase; advanced host/port fields; clipboard fallback | `features/database-connection/ui/EditSshTunnelComponent.tsx`                    | Inspected; see `visual-evidence.json` |
| DB09 | Limited-user creation view and settings dialog              | Privileges disclosure; credentials; check/create states and failure feedback  | `pages/databases/ui/CreateLimitedUserComponent.tsx`                             | Inspected; see `visual-evidence.json` |
| DB10 | Limited-user skip confirmation dialog                       | Current user kind; cancel/confirm; return focus and retained wizard state     | `pages/databases/ui/CreateLimitedUserComponent.tsx`                             | Inspected; see `visual-evidence.json` |
| DB11 | Read-only database settings and name editor                 | All five engine views; inline name editing; copy actions; literal diagnostics | `pages/databases/ui/DatabaseConfigComponent.tsx`                                | Inspected; see `visual-evidence.json` |
| DB12 | Database notifier selection in creation and settings        | Search/multi-select; long labels; no notifiers; failed load/retry             | `pages/databases/ui/DatabaseNotifiersFields.tsx`                                | Inspected; see `visual-evidence.json` |
| DB13 | Nested notifier creation from database settings/wizard      | Parent draft retained; test/save; new selection; nested focus and Escape      | `pages/databases/ui/DatabaseNotifiersFields.tsx`                                | Inspected; see `visual-evidence.json` |
| DB14 | Database copy confirmation                                  | Ordinary action styling; pending/error; copied selection                      | `pages/databases/ui/DatabaseConfigComponent.tsx`                                | Inspected; see `visual-evidence.json` |
| DB15 | Database deletion confirmation                              | Usage/pending behavior; destructive styling; error and focus restoration      | `pages/databases/ui/DatabaseConfigComponent.tsx`                                | Inspected; see `visual-evidence.json` |

### Schedules and backup configuration

Existing browser owners: `logicalBackupConfig.spec.ts / physicalBackupConfig.spec.ts / databaseCleanup.spec.ts`. Add scenarios there or in a focused file beside them.

| ID   | Surface and required variants                                     | Additional checks                                                                     | Source owner under frontend/src                                         | Evidence                              |
| ---- | ----------------------------------------------------------------- | ------------------------------------------------------------------------------------- | ----------------------------------------------------------------------- | ------------------------------------- |
| CF01 | Hourly/daily/weekly/monthly/custom schedule controls              | Selected labels; day/time rollover; 12/24-hour entry; cron errors/help                | `entities/schedules/ui/ScheduleEditor.tsx`                              | Inspected; see `visual-evidence.json` |
| CF02 | Logical backup configuration wizard step and editor               | Validation; dirty Save policy; retry settings; notification triggers; test state      | `pages/databases/ui/backup-config/LogicalBackupConfigFields.tsx`        | Inspected; see `visual-evidence.json` |
| CF03 | Logical retention settings and help popups                        | Count/time/GFS modes; numeric emptiness and bounds; dependent fields                  | `pages/databases/ui/backup-config/LogicalRetentionFields.tsx`           | Inspected; see `visual-evidence.json` |
| CF04 | Physical backup configuration wizard step and editor              | Full/incremental schedules; chain retention; WAL threshold; notification triggers     | `pages/databases/ui/backup-config/PhysicalBackupConfigFields.tsx`       | Inspected; see `visual-evidence.json` |
| CF05 | Logical and physical read-only backup configuration               | Formatted schedules/retention; storage choice; edit actions; cleanup summary          | `pages/databases/ui/backup-config/ShowLogicalBackupConfigComponent.tsx` | Inspected; see `visual-evidence.json` |
| CF06 | Backup storage selection and nested storage creation dialog       | Long selected labels; add provider; preserved parent draft; nested focus              | `pages/databases/ui/BackupStorageSelection.tsx`                         | Inspected; see `visual-evidence.json` |
| CF07 | Storage-switch warning confirmation                               | Cancel retains draft; confirmation submits once; pending and unchanged Save           | `pages/databases/ui/backup-config/EditBackupConfigSection.tsx`          | Inspected; see `visual-evidence.json` |
| CF08 | Cleanup notice and failed-summary retry in database configuration | Logical/physical states; counts refresh; failure retains values; retry placement      | `pages/databases/ui/backup-config/BackupConfigNotice.tsx`               | Inspected; see `visual-evidence.json` |
| CF09 | Healthcheck read-only and edit sections                           | Enablement/interval; validation; Save policy; failed request and retained input       | `pages/databases/ui/healthcheck/EditHealthcheckConfigComponent.tsx`     | Inspected; see `visual-evidence.json` |
| CF10 | Healthcheck attempts/history surface                              | Loading/empty/failure/data; availability marks; timestamp tooltips and period choices | `pages/databases/ui/healthcheck/HealthcheckAttemptsComponent.tsx`       | Inspected; see `visual-evidence.json` |

### Database transfer and nested dialogs

Existing browser owners: `databaseTransfer.spec.ts`. Add scenarios there or in a focused file beside them.

| ID   | Surface and required variants                     | Additional checks                                                             | Source owner under frontend/src                          | Evidence                              |
| ---- | ------------------------------------------------- | ----------------------------------------------------------------------------- | -------------------------------------------------------- | ------------------------------------- |
| TR01 | Database transfer dialog                          | Target workspace; storage/notifier mapping; blockers; long labels; save/error | `pages/databases/ui/DatabaseTransferDialogComponent.tsx` | Inspected; see `visual-evidence.json` |
| TR02 | Storage creation nested inside database transfer  | Correct target workspace; provider form; parent mapping retained; focus       | `pages/databases/ui/DatabaseTransferDialogComponent.tsx` | Inspected; see `visual-evidence.json` |
| TR03 | Notifier creation nested inside database transfer | Correct target workspace; provider form; selected notifier; parent focus      | `pages/databases/ui/DatabaseTransferDialogComponent.tsx` | Inspected; see `visual-evidence.json` |

### Logical backups and restores

Existing browser owners: `backupUsage.spec.ts / databaseResources.spec.ts`. Add scenarios there or in a focused file beside them.

| ID   | Surface and required variants                     | Additional checks                                                                           | Source owner under frontend/src                                              | Evidence                              |
| ---- | ------------------------------------------------- | ------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------- | ------------------------------------- |
| LB01 | Logical backup table and narrow presentation      | Chronological sorting; loading/empty/error/data; size/duration; actions                     | `pages/databases/ui/logical-backups/LogicalBackupsComponent.tsx`             | Inspected; see `visual-evidence.json` |
| LB02 | Logical backup filters and calendar popup         | Status/date/clear; calendar keyboard; locale words and regional order                       | `pages/databases/ui/logical-backups/LogicalBackupsFiltersPanelComponent.tsx` | Inspected; see `visual-evidence.json` |
| LB03 | Logical backup delete confirmation                | Pending/failure; destructive styling; selected row and focus                                | `pages/databases/ui/logical-backups/LogicalBackupsComponent.tsx`             | Inspected; see `visual-evidence.json` |
| LB04 | Logical backup failure-details dialog             | Long diagnostic text; technical characters; scroll; close/focus                             | `pages/databases/ui/logical-backups/LogicalBackupsComponent.tsx`             | Inspected; see `visual-evidence.json` |
| LB05 | Logical restore dialog and target connection form | All offered target engines; stored-secret restrictions; submit/pending/error                | `pages/databases/ui/restores/RestoresComponent.tsx`                          | Inspected; see `visual-evidence.json` |
| LB06 | Restore list/cards within the restore dialog      | Running/completed/failed/canceled; elapsed duration; refresh; cancel and diagnostic actions | `pages/databases/ui/restores/RestoresComponent.tsx`                          | Inspected; see `visual-evidence.json` |
| LB07 | Restore cancellation confirmation                 | In-flight cancellation; error preservation; focus returns to parent                         | `pages/databases/ui/restores/RestoresComponent.tsx`                          | Inspected; see `visual-evidence.json` |
| LB08 | Restore failure-details dialog                    | Diagnostic sections; copy; long text; nested focus and close                                | `pages/databases/ui/restores/RestoreErrorDialog.tsx`                         | Inspected; see `visual-evidence.json` |
| LB09 | Manual verification confirmation                  | Current backup; translated explanation; submit once; obsolete session                       | `pages/databases/ui/logical-backups/LogicalBackupsComponent.tsx`             | Inspected; see `visual-evidence.json` |

### Physical backups and WAL monitoring

Existing browser owners: `physicalBackups.spec.ts / wal.spec.ts`. Add scenarios there or in a focused file beside them.

| ID   | Surface and required variants                    | Additional checks                                                                        | Source owner under frontend/src                                                | Evidence                              |
| ---- | ------------------------------------------------ | ---------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------ | ------------------------------------- |
| PB01 | Physical chain table and narrow cards            | Full/incremental/WAL type distinctions; statuses; sizes; durations; actions              | `pages/databases/ui/physical-backups/PhysicalBackupsComponent.tsx`             | Inspected; see `visual-evidence.json` |
| PB02 | Physical backup filters and calendar popup       | Type/status/date; clear; calendar keyboard; preserved filter state                       | `pages/databases/ui/physical-backups/PhysicalBackupsFiltersPanelComponent.tsx` | Inspected; see `visual-evidence.json` |
| PB03 | Full-chain deletion confirmation                 | Dependents warning; destructive/pending/error; focus restoration                         | `pages/databases/ui/physical-backups/PhysicalBackupsComponent.tsx`             | Inspected; see `visual-evidence.json` |
| PB04 | Incremental backup deletion confirmation         | Correct backup; destructive/pending/error; retained chain                                | `pages/databases/ui/physical-backups/PhysicalBackupsComponent.tsx`             | Inspected; see `visual-evidence.json` |
| PB05 | Physical backup failure-details dialog           | Literal multiline diagnostics; wrapping; scroll; focus restoration                       | `pages/databases/ui/physical-backups/PhysicalBackupsComponent.tsx`             | Inspected; see `visual-evidence.json` |
| PB06 | Physical restore dialog                          | Script/manual methods; host/Docker tabs; generated link and commands; copy; reopen reset | `pages/databases/ui/physical-backups/PhysicalRestoreComponent.tsx`             | Inspected; see `visual-evidence.json` |
| PB07 | Point-in-time restore dialog and date/time popup | Latest or chosen target time; local/UTC boundary; method tabs; commands; calendar focus  | `pages/databases/ui/physical-backups/PhysicalRestoreComponent.tsx`             | Inspected; see `visual-evidence.json` |
| WM01 | WAL monitoring overview and charts               | Positive plot geometry; theme palette; tooltips; reduced motion; loading/error           | `pages/databases/ui/wal-stream-monitoring/WalStreamMonitoringComponent.tsx`    | Inspected; see `visual-evidence.json` |
| WM02 | WAL technical details, tips and restore timeline | Always-visible technical values and tips; timeline tooltips; long values and scrolling   | `pages/databases/ui/wal-stream-monitoring/WalStreamMonitoringComponent.tsx`    | Inspected; see `visual-evidence.json` |

### Backup verification

Existing browser owners: `verificationConfig.spec.ts / verificationScreens.spec.ts`. Add scenarios there or in a focused file beside them.

| ID   | Surface and required variants                 | Additional checks                                                                      | Source owner under frontend/src                                       | Evidence                              |
| ---- | --------------------------------------------- | -------------------------------------------------------------------------------------- | --------------------------------------------------------------------- | ------------------------------------- |
| VF01 | Verification read-only and edit configuration | Enablement; after-backup/timed scheduling; notifications; dirty/invalid/pending states | `pages/databases/ui/verification/EditVerificationConfigComponent.tsx` | Inspected; see `visual-evidence.json` |
| VF02 | Verification run table, cards and actions     | Status/reason; load more; no-agent warning; empty/error/loading; cancellation          | `pages/databases/ui/verification/VerificationsComponent.tsx`          | Inspected; see `visual-evidence.json` |
| VF03 | Verification detail drawer                    | Current run; progress; logs; diagnostic copy; failure sections; close/reopen focus     | `pages/databases/ui/verification/VerificationDetailDrawer.tsx`        | Inspected; see `visual-evidence.json` |

### Workspace and account administration

Existing browser owners: `workspaceSettings.spec.ts / profile.spec.ts / users.spec.ts / instanceSettings.spec.ts`. Add scenarios there or in a focused file beside them.

| ID   | Surface and required variants                      | Additional checks                                                                          | Source owner under frontend/src                               | Evidence                              |
| ---- | -------------------------------------------------- | ------------------------------------------------------------------------------------------ | ------------------------------------------------------------- | ------------------------------------- |
| WS01 | Workspace settings and name editor                 | Owner/admin/member/viewer; dirty Save; API failure; switching workspace                    | `pages/workspace-settings/ui/WorkspaceSettingsComponent.tsx`  | Inspected; see `visual-evidence.json` |
| WS02 | Workspace member list, table and narrow cards      | Role selection; owner/admin restrictions; pending/failure; invitation marks and join dates | `pages/workspace-settings/ui/WorkspaceMembersComponent.tsx`   | Inspected; see `visual-evidence.json` |
| WS03 | Add-member dialog                                  | Existing account/new invite; role picker; validation; pending/error                        | `pages/workspace-settings/ui/WorkspaceMembersComponent.tsx`   | Inspected; see `visual-evidence.json` |
| WS04 | Invitation-result dialog                           | Translated result; acknowledgement; retained member state; focus                           | `pages/workspace-settings/ui/WorkspaceMembersComponent.tsx`   | Inspected; see `visual-evidence.json` |
| WS05 | Ownership-transfer dialog                          | Eligible member selection; validation; pending/error; permission refresh                   | `pages/workspace-settings/ui/WorkspaceMembersComponent.tsx`   | Inspected; see `visual-evidence.json` |
| WS06 | Desktop and narrow member-removal confirmations    | Both invokers; anchored placement; cancel/confirm; destructive action and focus            | `pages/workspace-settings/ui/WorkspaceMembersComponent.tsx`   | Inspected; see `visual-evidence.json` |
| WS07 | Workspace audit log                                | Filters; paging; literal metadata; loading/empty/error; reflow                             | `pages/workspace-settings/ui/WorkspaceAuditLogsComponent.tsx` | Inspected; see `visual-evidence.json` |
| WS08 | Workspace-deletion confirmation                    | Usage/counts; translated name; pending closure policy; stale workspace/session             | `pages/workspace-settings/ui/WorkspaceSettingsComponent.tsx`  | Inspected; see `visual-evidence.json` |
| PR01 | Profile personal-details screen                    | Name/email and role display; logout; validation; overlapping save and locale switch        | `pages/profile/ui/ProfilePage.tsx`                            | Inspected; see `visual-evidence.json` |
| PR02 | Profile password-change screen                     | Visibility; confirmation errors; pending; success notice across sign-out                   | `pages/profile/ui/ProfilePage.tsx`                            | Inspected; see `visual-evidence.json` |
| AD01 | Instance settings screen                           | All policy switches; dirty/pending/error; changed draft during save                        | `pages/instance-settings/ui/InstanceSettingsPage.tsx`         | Inspected; see `visual-evidence.json` |
| AD02 | Instance mail-server information and test surface  | Configured/unconfigured; diagnostics; permission and pending/error state                   | `pages/instance-settings/ui/MailServer.tsx`                   | Inspected; see `visual-evidence.json` |
| AD03 | Instance audit log                                 | Filters; paging; literal metadata; loading/empty/error; reflow                             | `pages/instance-settings/ui/InstanceAuditLogs.tsx`            | Inspected; see `visual-evidence.json` |
| AD04 | Users administration table and narrow presentation | Search/paging; role and active-status changes; pending/error states                        | `pages/users/ui/UsersPage.tsx`                                | Inspected; see `visual-evidence.json` |
| AD05 | User audit-log drawer                              | Selected account; filters/paging; width/scroll; loading/failure; focus return              | `pages/users/ui/UserAuditLogsSidebar.tsx`                     | Inspected; see `visual-evidence.json` |

### Verification agent administration

Existing browser owners: `verificationScreens.spec.ts`. Add scenarios there or in a focused file beside them.

| ID   | Surface and required variants                    | Additional checks                                                                 | Source owner under frontend/src                                                  | Evidence                              |
| ---- | ------------------------------------------------ | --------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- | ------------------------------------- |
| AG01 | Verification agent table and narrow presentation | Online/stale/deleted states; name/status; actions; loading/empty/error            | `pages/instance-settings/ui/verification-agents/VerificationAgentsComponent.tsx` | Inspected; see `visual-evidence.json` |
| AG02 | Create-agent dialog                              | Name validation; pending/error; cancel/reopen reset and focus                     | `pages/instance-settings/ui/verification-agents/CreateAgentDialog.tsx`           | Inspected; see `visual-evidence.json` |
| AG03 | Revealed agent-token dialog                      | One-time visibility; copy; saved acknowledgement; no unintended persistence       | `pages/instance-settings/ui/verification-agents/RevealedAgentTokenDialog.tsx`    | Inspected; see `visual-evidence.json` |
| AG04 | Agent installation-commands dialog               | Architecture choice; literal install/launch commands; copy; long lines and scroll | `pages/instance-settings/ui/verification-agents/AgentInstallCommandsDialog.tsx`  | Inspected; see `visual-evidence.json` |
| AG05 | Rotate-agent-token dialog                        | Destructive confirmation; pending/error; revealed replacement token and focus     | `pages/instance-settings/ui/verification-agents/RotateAgentTokenDialog.tsx`      | Inspected; see `visual-evidence.json` |
| AG06 | Agent deletion confirmation                      | Anchored confirmation; pending/error; selected agent and focus return             | `pages/instance-settings/ui/verification-agents/VerificationAgentsTable.tsx`     | Inspected; see `visual-evidence.json` |

## Implementation evidence

The unchanged browser suite passed all 160 original scenarios in run `2558ddd5-063e-477f-b325-31ff820a9fd5`. The final expanded original-interface replay, `6ce65f89-7627-4d3f-bd57-c8b834a48c82`, completed 290 scenarios: 271 passed and 19 exposed original-interface defects. Its images, contexts and traces remain under `frontend/browser-results/antd-replay-source/frontend/browser-results/6ce65f89-7627-4d3f-bd57-c8b834a48c82/`. The original production source is unchanged. The replay provenance records fixture corrections, including hovering the original hover-only WAL tooltip after focus; the replacement separately requires keyboard access.

The preserved failures cover enlarged configuration and SSH forms, backup and restore cards, S3 credential/control widths, agent actions and table scrolling, header/menu focus and reflow, and the narrow WAL overview. They are historical reproductions, not accepted exceptions. The replacement passes their behavior and geometry assertions. Additional replacement defects found during inspection were repaired: selected navigation colors, compact control and overlay geometry, readable status tables, Azure credential widths, translated action wrapping, fresh portal text enlargement, and clipped TimePicker/password text. Removing migration captures exposed competing initial-focus requests during rapid repeated menu use. Language and theme menus now delegate initial selected-item focus to Mantine through `data-autofocus`, while application code retains immediate focus restoration on close. Dialog fields use the same single initial-focus owner; the database-wizard keyboard scenario waits until the dialog has initial focus before filling fields or selecting an engine. The agent-table keyboard test waits for the native `scrollend` event before resetting its position; otherwise an unfinished key scroll changes the background behind the next dialog. Real input heights are checked against their computed font size; locally scrollable labelled keyboard regions remain allowed while document overflow still fails.

[visual-evidence.json](visual-evidence.json) maps all 111 matrix rows to source owners, historical/final scenarios and inspected image pairs. Eight historical inspection sets contain 1172 pairs across successive corrections; this counts revisions, not unique screens. The supplemental audit adds 183 accepted comparisons on 54 viewed sheets, retaining excluded diagnostics separately. [visual-context-audit.json](visual-context-audit.json) explicitly names desktop light/dark, narrow translated and actual 200-percent evidence for every row. SH03 and SH04 name the applicable responsive counterpart where the desktop header or mobile drawer cannot exist. Each index records image hashes, actual contexts and inspection notes. Misframed sections, closed anchored confirmations and mismatched locales do not establish acceptance; focused replays replace them. All eight original-reference comparisons remain separately inspected. Library icon strokes and focus outlines may differ; unexplained layout differences were repaired before final acceptance.

The tracked ledger identifies preserved source archives and SHA-256 hashes. Original source, fixtures, lockfiles and capture configuration remain reproducible. Named PNG/context evidence records origin, theme, locale, browser region, timezone, fixed time, viewport, text enlargement and scroll ownership. The browser preview uses the stable private origin `http://127.0.0.1:4173`; it publishes no host port. Inspection used both the running development application through Chrome DevTools MCP and the production build through isolated Playwright fixtures. The tested host architecture is Linux x64; no arm64 execution is claimed.

### Test inventory and consolidation

[test-inventory.json](test-inventory.json) inventories every test owner added or expanded by this change. New declarations include renames; parameterized declarations expand into multiple runtime scenarios. The inventory preserves pre-cleanup identities so removal of migration labels cannot look like removed product coverage.

| Retained owners                                                                                                                      | Unique checks retained                                                                                                                                             | Cleanup decision                                                                                                                                                     |
| ------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Screen tests under `frontend/src`, including shared providers, lifecycle, connection, storage, notifier and administration consumers | Requests, secret omission, validation, dirty baselines, pending operations, permission restrictions, stale sessions/responses, translations and feedback deadlines | Retain all unique assertions; use actual Mantine providers and semantic queries.                                                                                     |
| `appShell.spec.ts`, `smtp.spec.ts`, `authSurfaces.spec.ts`                                                                           | Keyboard menus, active navigation, draft preservation, nested Escape, focus restoration, authentication callbacks and feedback                                     | Combine the duplicate mobile SMTP setup into the existing draft-preservation scenario; preserve both Escape cycles, first-item/Home behavior and language selection. |
| Connection, provider and resource scenarios                                                                                          | Every engine/provider, credentials and conditional fields, nested creation, SSH/paste, transfer, permissions, refusal, deletion and retained drafts                | Retain behavior across all existing contexts; remove equivalent appearance captures.                                                                                 |
| Configuration, logical/physical backup, restore and verification scenarios                                                           | Schedules, retention, calendar/region entry, pending/errors, nested dialogs, diagnostic scrolling and request payloads                                             | Retain browser interaction and geometry assertions; share representative appearance references.                                                                      |
| `wal.spec.ts`, `verificationScreens.spec.ts`, administration scenarios                                                               | Chart geometry/palette, timeline and keyboard help, labelled table scrolling, tokens, commands, drawer focus and 200-percent text                                  | Retain all distinct behavior and reflow regressions.                                                                                                                 |
| Browser fixture, source-alias and runner tooling tests                                                                               | Finite HTTP accounting, stable preview origin, explicit reference updates and failure cleanup                                                                      | Retain shared setup and diagnostics; remove migration-only PNG/context collection.                                                                                   |
| `e2e/browser/scenarios`                                                                                                              | Real backend navigation, resource changes, logical restore, physical full/incremental/WAL backup and Script/Host/Docker restore                                    | Preserve application assertions and replace vendor DOM queries with accessible state.                                                                                |

The permanent visual set has 17 references. Light/dark shell selection owns two; SMTP owns four locale/theme/width/error combinations; logical cleanup owns two theme variants; connection paste owns three contexts; narrow enlarged SSH owns one; narrow enlarged transfer owns one; WAL overview owns two themes; the physical target calendar and narrow enlarged verification diagnostics own one each. These cover selection contrast, dense forms, nested overlays, long translations, enlarged controls, charts, calendars and scrollable diagnostics. Physical cleanup images are byte-identical to logical cleanup in each theme, so their screenshot assertions are redundant; their behavior assertions remain. Reference renaming preserves pixels and is recorded in `retained-references.json` after acceptance.

The final scenario identities and current image hashes are recorded in [retained-references.json](retained-references.json). Six reference names remain unchanged, nine references are renamed, and two inspected final-state captures become shell references; 643 redundant old references are removed. The following set retains distinct appearance risks.

| Reference                                      | Layout risk                                                            |
| ---------------------------------------------- | ---------------------------------------------------------------------- |
| `application-shell-dark.png`                   | Selected navigation contrast and shell/list geometry                   |
| `application-shell-light.png`                  | Selected navigation contrast and shell/list geometry                   |
| `cleanup-logical-dark.png`                     | Compact cleanup status, warning wrapping and theme contrast            |
| `cleanup-logical-light.png`                    | Compact cleanup status, warning wrapping and theme contrast            |
| `connection-paste-dark-en-1280.png`            | Nested manual-paste dialog, parent context and compact controls        |
| `connection-paste-light-en-1280.png`           | Nested manual-paste dialog, parent context and compact controls        |
| `connection-paste-light-fr-320.png`            | Nested manual-paste dialog, parent context and compact controls        |
| `connection-ssh-fr-narrow-200.png`             | Long SSH fields and footer with doubled French text                    |
| `database-transfer-fr-narrow-200.png`          | Nested transfer mapping with doubled French text                       |
| `physical-restore-target-calendar.png`         | Regional calendar, time segments and nested popup geometry             |
| `smtp-dark-ru-desktop-validation.png`          | SMTP field density, long copy and validation across locale/theme/width |
| `smtp-dark-zh-narrow.png`                      | SMTP field density, long copy and validation across locale/theme/width |
| `smtp-light-en-desktop.png`                    | SMTP field density, long copy and validation across locale/theme/width |
| `smtp-light-fr-narrow.png`                     | SMTP field density, long copy and validation across locale/theme/width |
| `verification-diagnostics-dark-narrow-200.png` | Drawer scroll ownership and enlarged literal diagnostics               |
| `wal-overview-dark.png`                        | Chart palette, measured plots and overview spacing                     |
| `wal-overview-light.png`                       | Chart palette, measured plots and overview spacing                     |

Migration-only capture calls, automatic final-state attachments, context collection and matrix IDs are removed only after full migration acceptance. The source archive `frontend/browser-results/mantine-pre-cleanup/` preserves the complete accepted matrix suite and all its references. The ordinary runner continues to reject missing references and any changed pixel, with no retry or threshold relaxation.

### Verification status

Reference update run `f883c3eb-a88f-4a98-8ada-5f8357098b56` passed 290 scenarios. The ordinary migration run `0779498a-be21-4067-8463-079a00ca5a9a` passed 290 scenarios against 658 approved references. Cleanup consolidated only duplicate SMTP setup, retaining 289 cases and 17 references. The supplemental context audit completed missing inspections using preserved original production source and the preserved migration harness. Its capture adapters remain in ignored source archives; capture-only replay failures are classified in `visual-evidence.json` and are not reported as ordinary acceptance. Final acceptance required the completed context audit and a fresh ordinary run.

The final ordinary browser run `06b9f38a-f433-4c20-a6b4-1928ae253ebb` passed all 323 cases against the 17 retained references. The final complete frontend suite passed all 2736 tests in 192 files. No unique behavior case was removed. The previously recorded `mise run test` passed; backend runtime code is unchanged, and a library-specific package comment was corrected during final cleanup. The real-backend browser suite passed four application scenarios and skipped five opt-in lifecycle probes. Frontend, backend and Python lint, the production build and strict OpenSpec validation passed. Linux x64 was verified; no native ARM execution is claimed.

All 1015 frontend source, test, scenario and reference files match `frontend/browser-results/mantine-release-final-source-21/source.tar.gz`, SHA-256 `e0e19c2ac09e1dc6fabca8abf1c9983a663e69145566391515731889d1a2b66e`. This is the exact source of the final browser and frontend runs. The current verification no longer depends on intermediate whitespace-only source exceptions. The inspected calendar reference reflects focus moving into its popup; the other sixteen images retain their approved hashes.

Permanent tests use domain names without migration matrix prefixes, and the 17 reference images all have retained owners. Temporary capture infrastructure and duplicate migration checks are absent from the commit candidate. Final cleanup removed 63 unreferenced generated-result directories, freeing 5.33 GiB. Referenced original/final comparison evidence stays under ignored browser results, outside the commit. The style linter no longer exempts a deleted theme owner. Exact cleanup decisions and checks are recorded under `releaseCleanup` in `visual-evidence.json`.

Supplemental resource-menu inspection compared 24 before/after pairs on 12 sheets, covering two and sixty generated resources for workspaces, storages and notifiers. Workspace menus also include the current workspace. Both desktop themes and both narrow French 200-percent contexts were inspected. The comparison baseline is the prior Mantine implementation; the original Ant Design migration evidence remains intact. Short menus preserve their typography and controls. Popup placement may change to keep the complete menu visible; long lists scroll internally while search and creation remain reachable. The comparison index, exact source archives and observations are recorded under `resourcePickerScrolling` in the final ledger.

Supplemental field inspection compared eight before/after pairs for workspace-name validation and Telegram advanced URLs in both desktop themes and both narrow French 200-percent contexts. Supported Mantine `error` and `description` props retain the actual accessible descriptions; hidden help adds no visible spacing. All affected field regions match the prior Mantine presentation pixel for pixel. The ordinary database-creation browser scenario also dismisses notifier search without closing its wizard, retains the chosen storage after going back and completes creation. The shared picker consumes Escape from search even when Popover has already closed during event capture; a second Escape from the restored trigger reaches the parent. Source archives, inspected sheets and final results are recorded under `fieldAssociationsAndEscape` in the ledger.

The pending OAuth callback has separate before/after evidence from its refusal state. Six inspected comparisons cover desktop light/dark and narrow French light/dark at ordinary and actual 200-percent text. All 850 original production files are unchanged in the before replay. A valid callback exchange stays pending through the capture and resolves only when its controlled response is released. Loader and label geometry match the original; the loader occupies 20px normally and 40px at 200-percent text. Three added permanent browser cases retain the valid request, pending state, scrubbed URL, consumed attempt, translated refusal, single exchange and geometry checks. Their source and temporary inspection index are recorded under `oauthCallbackPending` in the final ledger and `stateEvidence.AU06.pending` in the context audit. The pending-state comparison leaves the 17-reference set unchanged.

The point-in-time calendar receives and traps focus within its parent dialog. A production browser scenario opens it with ArrowDown, navigates to a date, enters hours/minutes/seconds and PM using only keyboard events, submits the UTC target in America/New_York, checks Tab/Shift+Tab containment and restores focus after submission and Escape. The first Escape dismisses only the calendar; the next dismisses the parent. Six inspected prior/current Mantine comparisons cover both desktop themes and narrow French light/dark at ordinary and actual 200-percent text. Geometry is unchanged; the visible keyboard outline moves into the calendar. The inspected pointer reference update removes only the target field outline. Sources, images, red/green results and the reference review are recorded under `calendarKeyboardFocus` in the final ledger.
