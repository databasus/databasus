# Frontend design and localization coverage

This map preserves the existing test guarantees while adding the milestone-2 pilot. Paths are relative to `frontend/`. Existing files stay in place unless a unit is actually extracted; implementation records any move and its reason here. No existing scenario is removed merely because its control, feedback mechanism or selector changes.

## Existing guarantees

| Current test | Owner after this change | Guarantee and necessary adjustment |
| --- | --- | --- |
| `src/features/notifiers/ui/edit/EditNotifierComponent.test.tsx` | Same notifier editor test | Preserve saved-type presentation, omitted Slack token, omitted stored webhook header value and failed-test diagnostics. Replace alert spying with visible context-bound feedback assertions and use accessible field names. |
| `src/features/notifiers/ui/edit/notifiers/EditTelegramNotifierComponent.test.tsx` | Same provider test | Preserve initial advanced disclosure for new notifiers, configured proxy and custom Bot API URL. Add keyboard/state assertions after the shared disclosure repair. |
| `src/features/notifiers/ui/NotifierComponent.test.tsx` | Same notifier screen test | Retain existing notifier actions and permissions; mechanically updated confirmation props must not change their operations. |
| `src/shared/lib/test/renderScreen.test.tsx` | Same harness test | Preserve real locale/theme configuration, draft retention, subscription/timer cleanup and isolation. Add the shared theme bridge to the same composition. |
| `src/shared/i18n/dictionaries.test.ts` | Same dictionary test | Preserve identical placeholders, formatting tags, technical code spans and nonempty values across all six dictionaries. |
| `src/shared/i18n/createI18n.security.test.tsx` | Same security test | Preserve hostile names and unsupported-tag handling. Add pilot coverage using the same safe interpolation constraints. |
| Other `src/shared/i18n/*.test.*` | Same formatter/error/locale/link tests | Preserve first language selection, region distinctions, safe API errors and localized website links. |
| `src/features/storages/ui/edit/EditStorageComponent.test.tsx` | Same storage test | Preserve saved-type display, omitted S3 credentials, custom class and technical failure detail. No storage lifecycle conversion in this change. |
| `src/features/backups/physical/monitoring/ui/WalStreamMonitoringSections.test.tsx` and monitoring `lib/*.test.ts` | Same monitoring owners | Preserve existing text, rates, timeline, polling and chart calculations; add browser geometry without replacing these tests. |
| Existing database/backup/restore/storage confirmation consumers and their tests | Same owners | Update `actionButtonColor` callers to `isDestructive` and retain underlying confirmation actions. Run the complete frontend suite to include those areas. |
| `src/app/ui/App.test.tsx` | Same app test | Preserve application-provider and authenticated-profile behavior after theme integration. |

## Added guarantees

| Surface | New scenarios | Verification layer |
| --- | --- | --- |
| Shared field/disclosure/confirmation controls | Associated unique IDs, help/error, keyboard and expanded state, ordinary cancel styling, pending/destructive presentation | Shared UI screen tests; browser focus checks |
| Language/theme controls | Hover/click behavior retained, keyboard selection/dismissal, accessible compact names, selected state, no form reset | Shared UI tests and browser UI |
| Theme | Shared token roles and scrollbar properties in both modes, system changes, document/portal bridge cleanup, declared contrast pairs | Theme/provider tests plus browser UI |
| Notifier values | Async initial values, untouched versus dirty refresh, new/saved forms, reversed edits, conditional fields, cancel/reopen, provider changes | Notifier editor screen tests |
| Test/save lifecycle | Settings A pending then B, A-B-A, late failure, name-only change, locale/theme change, duplicate actions, failed save, newer edit during save, entity/workspace change, unmount and StrictMode replay | Controlled HTTP responses through the real editor |
| Creation-to-update transition | Block type switching during controlled pending creation; keep newer name/same-provider edits; retain the created ID and immutable type even with creation props unchanged; send exactly one create and later tests/updates to that ID/type; unlock type after failure without an ID; silently refresh parents without closing/remounting; preserve the created notifier after cancelling newer edits | Controlled-response editor tests and enclosing notifier screen tests, plus database notifier-selection/transfer regression tests |
| Secrets | Omitted saved SMTP password, supplied replacement, retained Slack token/header omission, no placeholder credentials | Request inspection in editor screen tests |
| Active errors | Validation and request failure translated after switching language; diagnostics/user values remain literal; no automatic delivery | Editor screen tests and one browser scenario |
| Transient success notices | An already displayed notice may retain its creation language only until its original expiration; switching language does not restart that lifetime, persistent feedback updates and later notices use the selected language | Production browser scenario with controlled time |
| Style checker | TSX/CSS/Tailwind direct/named/functional colors, focus removal, semantic references, exact owners, baseline counts/new/stale findings, unrelated line movement | Node fixture tests under frontend Vitest |
| Browser resource owner | Valid build/fixture run, caught unexpected request, scenario failure, interruption, child exit and resource cleanup | Isolated runner failure fixtures with bounded completion and owned-resource inspection |
| Actual browser UI | SMTP editing, confirmation, light/dark, English/Russian/French/Chinese states, keyboard focus, 320 CSS-pixel reflow, real 200 percent text enlargement, positive WAL plot geometry | `test:browser` UI project |
| Typography/visual baseline | Local fonts only, Cyrillic/Latin/CJK readability, loaded-font readiness, stable four-state reference matrix, missing/changed baseline failure without automatic update | Browser UI and runner failure fixtures |
| CI configuration | Browser job uses local command and immutable environment; release version selection requires success; existing requirements remain | Local workflow-configuration regression test |

## Completion evidence

Implementation tasks and local verification are complete. Full implementation review accepted the change in its current state. Milestone 2 is marked complete in the uncommitted frontend plan.

### Final test owners

- `src/shared/theme/ThemeCssVariables.test.tsx` checks document properties, system media changes and StrictMode cleanup. `getAntdTheme.test.ts` checks the resolved token contrast pairs without a document environment. `ThemeAssets.test.ts` checks vendored asset hashes, license and removal of external font/zoom restrictions.
- `src/shared/ui/AdvancedFieldRowComponent.test.tsx`, `AdvancedSettingsToggleComponent.test.tsx`, `ConfirmationComponent.test.tsx` and `LanguageThemeControlComponent.test.tsx` check associated labels/help, keyboard disclosure, ordinary cancellation and language-menu interactions. Existing Telegram tests retain configured advanced disclosure.
- `src/features/notifiers/ui/edit/EditNotifierComponent.test.tsx` retains the original four scenarios and adds the three regressions below. `NotifierLifecycle.test.tsx` adds 18 controlled-response/session cases, including visible field-error translation, failed creation unlocking type and SMTP omission/replacement in both test and update payloads.
- `src/features/notifiers/ui/NotifiersComponent.test.tsx` checks that persistence refresh retains the newer input node and create-then-cancel leaves the saved notifier available. `src/features/databases/ui/NotifierComposition.test.tsx` checks that same behavior through database notifier selection and database transfer composition.
- The pure WAL response builder moved from `features/backups/physical/monitoring/lib/testWalStreamStatistics.ts` to `shared/lib/test/buildWalStreamStatistics.ts`, so browser fixtures and existing monitoring tests share data without importing domain services. All four existing consumers import that focused entry directly; no compatibility re-export remains, and no scenario was removed.
- `tooling/styleChecks.test.ts` covers 122 source-analysis fixtures, including referenced-object counts, technical-object classification, directional borders, outline shorthands/longhands with declaration ordering and importance, conditional CSS scope, shorthand properties, isolated conditional values/objects/classes, absent replacement values, template expressions and imperative branches, invisible focus replacements, stylesheet `@apply` (including arbitrary divide/placeholder colors), short-circuit object spreads and imperative DOM style mutations. `tooling/styleBaseline.test.ts` independently checks baseline counts and stale/new findings across source-line changes. `tooling/runBrowser.test.ts` and `tooling/HttpFixture.test.ts` cover six resource/fixture cases. `tooling/verifyBrowserProbes.ts` runs isolated Chromium failure probes before the production UI suite.
- `browser/scenarios/smtp.spec.ts` covers fourteen production-route scenarios, including four visual references, explicit test/save, literal diagnostics, locale/theme retention, live validation, scoped axe, confirmation focus, nested menu dismissal and language selection inside the narrow sidebar with translated navigation, retained drafts and restored focus, plus success-notice lifetime. `browser/scenarios/wal.spec.ts` checks measured positive SVG dimensions and actual plotted curves with deterministic series.

### Reproduced defects and repairs

Before replacing the notifier lifecycle, the three added editor regressions failed on the unchanged editor: reversed input still offered an action; an earlier test could authorize changed settings; and a late save cleared newer edits. The repaired tests pass through the real editor and HTTP boundary.

Browser checks also exposed insufficient light-theme help-text contrast, missing focus restoration when a confirmation owner immediately unmounts, Chinese button-name spacing, and a success notice covering the language menu until expiry. Resolved theme aliases, shared focus restoration, explicit button spacing policy and bottom-right context-bound notices repair those cases. The editor owns its notification holder so queued notices cannot outlive an unmounted editor, and language/theme menus expose the selected option. Retained-provider tests verify that successful late test/save completions emit neither notices nor persistence/completion callbacks after editor unmount. The outline-longhand, important utility, conditional CSS, shorthand property, conditional value, unbraced imperative branch and stylesheet `@apply` fixtures failed on the uncorrected checker and pass with explicit visible-replacement cases. Conditional-class and absent-replacement regressions also failed before repair. Branch resolution now retains absent values through references, templates, short-circuit expressions and conditional spreads, and overwritten object properties contribute only their final value to focus checks. The non-API request probe also reproduced an incorrectly successful teardown before the fixture boundary was restricted by method and resource type. A browser regression reproduced sidebar dismissal from nested language/theme menus; Escape now dismisses only the menu and restores its control focus, including dismissal from the open menu trigger. The style fixture first failed for imperative DOM colors; the parser now detects them and records only the existing legacy toast occurrences for milestone 8.

### Browser environment and references

The immutable source image is `mcr.microsoft.com/playwright:v1.64.1-noble@sha256:47b510dd9230bf0b9bfc9b20102241e577ea7a99334ab4c3ca19f19770e9b8ea` on `linux/amd64`, with Playwright 1.64.1, Node 24.21.0 and pnpm 9.15.9. `browser/environment.json` pins the Node archive hash; the Dockerfile verifies it and installs the frozen frontend lockfile. The dependency cache identity is computed from that metadata, Dockerfile, package and lockfile. Local and CI commands use the same runtime and fonts.

| Reference | Theme / locale | CSS viewport | State |
| --- | --- | --- | --- |
| `browser/references/smtp-light-en-desktop.png` | Light / English | 1280 x 1200 | Saved SMTP settings |
| `browser/references/smtp-dark-ru-desktop-validation.png` | Dark / Russian | 1280 x 1200 | Visible associated email validation |
| `browser/references/smtp-light-fr-narrow.png` | Light / French | 320 x 1200 | Wrapped help and actions |
| `browser/references/smtp-dark-zh-narrow.png` | Dark / Chinese | 320 x 1200 | Explicit system CJK fallback |

All four references were generated with the explicit update command and inspected as images. Ordinary runs compare exact pixels and leave their SHA-256 hashes unchanged. The narrow cases additionally open advanced settings, double each computed font size, assert actual input text enlargement, check label/control overlap and reach cancellation without document-wide horizontal overflow. Separate dirty French/Chinese scenarios retain a long translated validation error under enlargement, correct it, and reach both explicit Test and Save while checking overlap and document width. Keyboard-only browser navigation selects a language, checks its selected menu state and restores focus after selection and Escape. The DOM menu tests wait until opening actually transfers focus before sending the next action, removing a race exposed by the complete suite. CDP rendered-font inspection proves Jost glyph use in Latin/Cyrillic labels and system fallback glyph use in Chinese; font request accounting proves local-only WOFF2 loading.

Scenario assertion failure, independently caught undeclared API/non-API fetch and XHR (including a static-looking fetch URL), page errors, missing/changed references, child exit and interruption all produce nonzero results in isolated probes. These probes import the production Playwright configuration and the actual extended browser fixture; undeclared-request/page-error failures come from its automatic teardown. Navigation and static resource loading pass through only as same-origin GET requests of those resource types; fetch/XHR remains accounted for at every path. Following runs start clean. Unique runtime containers and temporary workspaces disappear after success/failure/interruption; the dependency image remains cached. No shared test services are stopped. Real-API E2E remains a clearly unavailable separate project.

### Migration remainder

The style baseline contains 686 exact entries and 1,207 occurrences: milestone 3 has 341 shell/user/shared occurrences; milestone 4 has 234 workspace/storage/notifier occurrences; milestone 5 has 183 database occurrences; milestone 6 has 99 logical-backup/restore occurrences; milestone 7 has 346 physical-backup/verification occurrences; milestone 8 has four legacy toast occurrences. Shared UI, the editor shell and SMTP pilot have none. Architecture retains its unchanged 43 individually recorded findings. Domain toast callers and the full shell height/visualViewport conversion remain in their agreed later milestones.

### Local verification

- `mise run frontend:lint`: passed formatting, source lint, architecture/style baselines, types and generated API consistency.
- `pnpm --dir frontend test`: 106 files and 870 tests passed. The existing happy-dom chart-size warnings remain; browser plot geometry is checked separately and passes.
- `pnpm --dir frontend build`: passed with the existing large-chunk and runtime-config-script warnings.
- `pnpm --dir frontend test:browser:update`: generated the four inspected reference images in the pinned environment.
- `pnpm --dir frontend test:browser --project=ui`: 15 scenarios passed, together with the isolated harness failure probes; reference hashes remained unchanged and owned containers were removed.
- `python -m unittest discover -s scripts -p frontend_ci_test.py`: four tests passed. No hosted pipeline was needed.
- Both new skills passed the skill-creator validator.
- `git diff --check` and `openspec validate frontend-design-and-localization --strict`: passed.

## Starting evidence

- `mise run frontend:lint`: passed; 43 individually recorded architecture findings.
- `pnpm --dir frontend test`: 92 files and 707 tests passed in 16.32 seconds.
- `pnpm --dir frontend build`: passed; existing large-chunk and runtime-config-script warnings remain.
- Typography audit found normal-style weight utilities; the vendored variable faces cover weights 100-900. Local Jost variable normal fonts: Cyrillic 10,140 bytes, Latin extended 17,104 bytes, Latin 26,576 bytes; 53,820 font bytes total, plus the 4,497-byte OFL license. Package source and SHA-256 identities are recorded with the assets.
