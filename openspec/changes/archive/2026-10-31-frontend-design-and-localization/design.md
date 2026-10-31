## Context

See `proposal.md` for motivation and scope. This is milestone 2 of the frontend plan, after the foundation change. The approach is constrained by root `AGENTS.md`, `frontend/AGENTS.md`, and the downward dependencies, state ownership and test-provider rules in `docs/architecture.md:29`.

Current evidence:

| Source | Constraint or gap |
| --- | --- |
| `frontend/src/shared/theme/getAntdTheme.ts:5` | AntD configuration already lives below app; it currently selects the algorithm and brand blue. Extend this owner rather than moving it again. |
| `frontend/src/app/ui/App.tsx:44` | App composes library configuration, feedback context and routing. Shared test infrastructure must not import this composition. |
| `frontend/src/index.css:61` | Scrollbars consume custom properties without definitions in the source tree. Global font selectors also compete with component typography. |
| `frontend/index.html:8` and `:13` | The viewport disables zoom and typography depends on Google's external stylesheet. |
| `frontend/src/shared/ui/AdvancedFieldRowComponent.tsx:11` | A visible field label is an unrelated div, and the help icon has a direct named color. |
| `frontend/src/shared/ui/AdvancedSettingsToggleComponent.tsx:11` | Disclosure is a clickable div with no expanded state. |
| `frontend/src/shared/ui/ConfirmationComponent.tsx:43` | Cancellation can receive destructive styling from the inverse of the confirm color. Both actions are primary. |
| `frontend/src/shared/ui/LanguageSelectorComponent.tsx:42` | Hover and click are existing language-menu behaviors that must survive the keyboard repair. |
| `frontend/src/features/notifiers/ui/edit/EditNotifierComponent.tsx:56` | Values, dirty state and test/save flags are independent; test completion at line 101 does not check whether settings changed. Save completion at line 80 unconditionally clears dirty state. |
| `frontend/src/features/notifiers/ui/NotifiersComponent.tsx:205` | The editor's completion callback refreshes the list and closes the creation modal; it cannot acknowledge persistence while retaining newer edits. |
| `frontend/src/features/storages/ui/edit/EditStorageComponent.tsx:63` | The storage editor has the same lifecycle shape, establishing the next consumer without including its conversion here. |
| `frontend/src/shared/i18n/LocaleProvider.tsx:38` | A stable translation instance and provider tree already preserve state during language changes. |
| `frontend/src/shared/toast/ToastHelper.ts:1` | Legacy feedback creates its own DOM, colors and timers; callers outside the pilot remain for their domain milestones. |
| `frontend/src/widgets/main/MainScreenComponent.tsx:198` | Shell dimensions and raw surface colors are mixed. Its full layout conversion belongs to milestone 3. |
| `frontend/src/features/backups/physical/monitoring/lib/chartColors.ts:18` | WAL has an explicit light/dark palette with domain roles; its calculations and line meanings do not need rewriting. |
| `frontend/package.json:6`, `frontend/vitest.config.ts:19`, `.github/workflows/ci-release.yml:135` | There is a screen-test command but no browser UI runner. Vitest discovers source tests, and CI already has an independent frontend test job. |

The current tests and their retained guarantees are mapped in `coverage.md`. The request-revision failure is a code-path finding; implementation must reproduce it on the unchanged editor before fixing it.

## Goals / Non-Goals

**Goals:** establish one semantic theme, accessible shared interactions and an actual form/browser pilot that later domain changes can copy. Preserve the current visual identity, all notifier types, URLs, saved preferences and credential semantics.

**Non-Goals:** introduce a schema-driven form framework, put notifier policy in shared, restructure all notifier pages, or prove every screen accessible. Shared-contract callers outside the pilot receive only necessary mechanical updates and regression verification. `proposal.md` defines the remaining exclusions.

## Decisions

### 1. Extend AntD tokens and export a small semantic CSS bridge

`shared/theme` owns the brand seed, AntD configuration and mapping from resolved AntD tokens to application roles. Enable AntD's CSS-variable mode and expose only application roles actually consumed by CSS: page/container surfaces, primary/secondary text, borders, focus, status and scrollbar roles. Tailwind's semantic utilities refer to these properties; layout and spacing utilities remain available. Standard AntD text, borders, status and controls use its tokens directly.

A small bridge beneath `ConfigProvider` reads the resolved tokens and makes application properties available to document surfaces and portals. Production app and `renderScreen` compose the same public bridge below their own configuration providers. It must support StrictMode replay and clean up any document mutations it owns. Theme changes update values without remounting descendants. The global font family is also set in AntD configuration, and CSS stops forcing the body typeface onto technical text.

WAL's existing palette remains an explicitly approved, narrowly scoped chart-palette owner. Component colors use standard theme roles; genuinely independent series colors stay in that palette. Preserve its dash patterns and existing numeric/time/reduced-motion behavior. Do not copy the palette into shared solely to satisfy a path rule.

Alternatives: a second application-wide palette would duplicate AntD's derived colors; copying every AntD alias would create unnecessary maintenance; mode-specific raw Tailwind colors in consumers would keep the inconsistency. The theme bridge follows the shared infrastructure boundary in `docs/architecture.md:31` and is documented there with its ownership, not its token inventory.

### 2. Repair existing shared controls and keep the reusable set small

Use AntD controls and `Form.Item` for field label/help/error presentation where their behavior fits. Repair `AdvancedFieldRowComponent` with a required `controlId`, a real associated label and linked help; update every SMTP and Telegram caller with matching control IDs, including multi-editor uniqueness. Replace the disclosure div with an AntD/native button exposing `aria-expanded` and an associated region. Give the language and theme buttons explicit localized accessible names and menu keyboard behavior. Preserve the language selector's paired placement, native language names, hover and touch/click behavior.

Replace confirmation's `actionButtonColor` with `isDestructive`; cancel is an ordinary secondary action. Update every caller across notifier, storage, database, restore and backup areas in the same change. These are deliberate internal source-contract breaks, declared in the proposal. Do not keep aliases. Preserve each caller's underlying operation and existing translated description. Verify dialog focus containment/restoration through the actual application.

Use `InlineCodeComponent` for inline technical prose and keep multiline diagnostics selectable and escaped in a bounded scroll region. Standard control glyphs use `@ant-design/icons`. Existing product/engine/provider images remain assets, with decorative or meaningful alternatives as appropriate. Limit other abstractions to patterns used by shared controls and the pilot: field presentation and form actions. AntD already supplies empty/loading/error primitives; a new wrapper is justified only by an actual shared behavior.

Alternatives: wrapping every Button/Input would create another UI kit; cloning arbitrary child elements to invent IDs would obscure ownership; exempting existing controls from keyboard checks would leave the new foundation unusable.

### 3. Retain the controlled draft and derive dirty state from its baseline

Keep notifier API calls and policy in the existing notifier owner, consistent with `docs/architecture.md:36-37`. Retain `EditNotifierComponent`'s controlled name/settings values as the single draft owner, with the existing provider components receiving settings and emitting changes. Use AntD `Form.Item` for label/help/error presentation without registering a second value store or introducing form-store adapters. Keep validation as domain-owned data translated at render time. Preserve all seven notifier variants and their current provider interfaces while making SMTP the detailed pilot.

The editor owns an immutable acknowledged baseline snapshot, a saved notifier ID and an editing-session identity alongside that draft. Dirty means semantic inequality between the current draft and its baseline, replacing manually toggled dirty flags. Optional empty values are normalized consistently with the existing request adapters; saved hidden secrets remain absent, distinct from entered replacements. New-form initial values form a baseline too: blank creation is not an unsaved saved entity, and valid entered settings still require an explicit test before creation.

Apply asynchronous initial values only to the intended untouched editing session. Switching entity/workspace or reopening after cancel starts a new session and baseline. Refreshing the same entity must not overwrite dirty input. Theme/locale changes do not reset the controlled draft, reinitialize its baseline or remount provider components. Derive field errors from the current draft and render their translations in the current language while retaining values, focus and expanded state. All field updates continue through the existing controlled handlers.

Alternatives: moving seven providers into AntD's value store would require adapters and additional initialization/reset synchronization without changing user functionality. Keeping independently toggled dirty flags reproduces inconsistent states, and treating touched as dirty incorrectly offers Save after a reversed edit. A second form library or full provider rewrite adds unnecessary migration work.

### 4. Guard asynchronous work by session, request and settings revision

Introduce the smallest domain-independent request-revision unit under `shared/lib`, verified through the real notifier editor. It provides identity/invalidation only. Notifier payload normalization, delivery rules, validation, secret treatment and the decision to require a successful test stay in the notifier model. Document the boundary so the storage conversion in milestone 4 can reuse the mechanism without importing a sibling feature.

Maintain distinct identities for the editing session, draft revision, settings revision and active request. A settings revision changes for delivery settings, including entered secrets and provider selection; name, locale and theme changes do not change it. Entity/workspace changes invalidate the whole session. Each explicit test captures its session, settings revision and request identity. Only a matching completion can set current success or error. An A-to-B-to-A edit still invalidates a test started before those changes. Store the successful current settings revision in memory; never log, persist or serialize secret-bearing snapshots for diagnostics. There are no automatic test retries, including on language changes, rerenders or StrictMode replay.

Each save captures its session, request identity, submitted payload and draft revision. Ignore UI effects from a previous session. A current-session success acknowledges the submitted values as the persisted baseline, even if the user has since edited the form; it retains the newer values and recomputes dirty/test eligibility. Only a completion still matching the current draft may invoke the existing close/replace callback. Failure keeps the current values and applicable next action. Duplicate activation cannot create concurrent saves. Closing invalidates all pending UI effects; transport cancellation is only an optimization and cannot undo a delivered notification or saved mutation.

Lock provider type as soon as a create request is submitted, before awaiting its response. Disable the type selector and guard the settings-change handler against changes to the submitted type; names and fields within that type remain editable. If creation fails without a saved ID, unlock type selection for the still-unsaved draft. A successful creation fixes the type permanently, following the main requirement `A saved notifier keeps its type`. Derive saved presentation and the provider's `isSaved` context from the retained saved ID, not only from `editingNotifier`, so the selector cannot reappear after creation in the same open editor. Allowing an SMTP-to-Telegram switch during pending creation would leave a draft that the returned SMTP ID cannot accept; silently discarding that switch would lose user input.

Creation has an explicit transition to saved editing: on a current-session create success, retain the returned notifier ID before acknowledging the baseline or notifying a parent. The next explicit test and save use that ID and its fixed provider type, and the next save calls update even if the original `editingNotifier` prop remains absent. This ID promotion belongs to the same editing session and preserves newer name and same-provider settings values, focus and expanded state; it does not initialize the draft again. Retain test eligibility when only the name changed, and require another explicit test if delivery settings changed. Subsequent payloads omit credentials already acknowledged as stored unless the user changed them again. A success from an obsolete session never promotes its ID into the active session.

Add a required `onPersisted(savedNotifier)` callback for every accepted current-session save, distinct from the existing `onChanged` completion callback. Update every caller in notifier screens, database notifier selection and database transfer composition. `onPersisted` silently refreshes saved entities/options without closing, remounting or attaching a notifier to another draft; `onChanged` performs completion/selection/attachment only when no newer draft remains. Parent refreshes of the promoted ID remain same-session acknowledgements, not entity switches. Cancelling newer unsaved edits leaves the already created notifier available; it neither deletes it nor creates it again. Verify with controlled responses that the type is locked during creation, a newer name or SMTP field survives success, subsequent tests/updates retain the returned ID and SMTP type, and failure without an ID unlocks type selection. Also verify create-then-cancel in the enclosing screen.

```text
edit --> valid dirty draft --> explicit test(settings revision)
                                   |
                  current success v
                               save(submitted draft)
                                   |
                         acknowledge submitted values
                                   |
                   preserve any newer draft and its action

session/settings change --> invalidate obsolete test effects
session change ---------> invalidate obsolete save effects
```

Alternatives: a success boolean loses which settings passed; abort alone cannot guard a response already completed; comparing the whole draft would require another test message after a name change. A generalized test/save workflow in shared would move domain policy into infrastructure.

### 5. Use context-bound feedback and reactive localized errors

The pilot uses `notification.useNotification()` for context-bound AntD feedback with its holder owned by the editor. Unmounting that holder removes both visible and queued notices even while the application providers remain mounted. Field validation and persistent operation errors remain in the form. Keep error/code/translation-key data in state and translate during rendering; supported channel diagnostics remain literal technical text with translated surrounding prose. Use safe translation patterns from the existing localization spec, including supplied elements for user values in rich sentences. The localization delta modifies the main language-rendering requirement explicitly: already displayed transient success notices may retain their creation language until their original bounded lifetime ends, without restarting it; a notice emitted after a switch uses the latest language. Persistent feedback and failures are outside this exception. Obsolete or unmounted sessions emit nothing, and reopen does not inherit persistent feedback.

Remove pilot calls to `alert` and `ToastHelper`. Keep the helper only while independent callers exist, and record its remaining owners/milestones rather than pretending migration is complete. No static AntD feedback API or new feedback framework is introduced.

Alternatives: storing translated errors freezes their language; recreating the form on a switch loses input; converting every toast in this change expands into later milestones.

### 6. Preserve Jost with local assets and bound the layout pilot

Keep Jost and vendor the minimum licensed Latin/Cyrillic WOFF2 assets needed for actual interface weights, with source revision/hash and OFL text. Audit used weights/styles before selecting subsets; measure individual and total byte sizes in the change evidence. Global font definitions/assets live with application global styling and are linked from `docs/frontend-design.md`. Define an explicit CJK system fallback; provide the same CJK font environment for screenshot runs. Production must make no Google Fonts or other font-service request. Remove `maximum-scale` and `user-scalable=no` from the viewport.

Use a flex/grid layout with `min-width: 0`, `min-height: 0` and one vertical scroll owner in the pilot's enclosing notifier region. Keep shell height props for unrelated consumers. Exercise the actual pilot inside the existing shell; adapt only the minimum wrapper needed to avoid double scrolling. Preserve `useScreenHeight` and its visualViewport handling until milestone 3 proves whether the shell still needs it. Verify 320 CSS-pixel width and actual 200 percent text enlargement; pixel-density emulation does not verify text reflow.

Alternatives: system-only typography would change the existing identity; external fonts keep an unnecessary installation dependency; bundling all scripts/weights is wasteful. Removing every `contentHeight` now would pull the complete shell and unrelated pages into this milestone. Exact font file sizes are not yet measured: the exploration's remote asset request timed out. Measurement and coverage verification are implementation tasks, not an unresolved product choice.

### 7. Check style syntax and keep a separate finite migration baseline

Add `lint:styles` to `frontend:lint`. Reuse the installed TypeScript parser for TS/TSX and a declared CSS parser for stylesheets; parse Tailwind color utilities, modifiers and arbitrary color values in style-bearing strings. Inspect relevant inline style properties and chart/component color props, including referenced local color constants. Recognize hex, named colors, rgb/hsl/other functional colors, default Tailwind palette utilities, and focus-outline removal. Semantic CSS variables and inherited/currentColor values are permitted. Replacement focus styles must be explicit and visible; suppressions require a specific finding and reason rather than a file-wide exemption.

Theme definitions and the existing WAL chart-palette file are exact approved owners. Importing a color through another ordinary component does not make it an approved source. Technical strings and images are classified by their role instead of scanned as arbitrary color-like text. The checker does not claim to prove computed contrast or every possible dynamically constructed style; token-pair checks and browser verification cover the declared foundation.

Use a separate `frontend/style-baseline.json` with rule, file, normalized construct/value, count, owner and removal milestone. Reuse the existing architecture baseline's matching approach where it fits without merging the inventories. Record only findings present before migration. Shared UI/theme foundation and pilot entries must disappear in this change. The explicitly retained legacy `ToastHelper` and untouched domain entries keep their assigned milestones, with final cleanup in milestone 8. New, multiplied and stale findings fail, independent of line shifts. Fixture tests prove both rejection and accepted semantic styles, not merely a regex's matches.

Alternatives: a hex grep misses named colors and utilities; a wildcard folder allowlist permits new violations; fixing all legacy color consumers now would exceed the agreed pilot.

### 8. Run browser UI checks against the actual built application

Add a pinned Playwright runner and `@axe-core/playwright`, a focused browser-test directory outside application `src`, and `pnpm --dir frontend test:browser --project=ui`. The command builds the production frontend and serves `dist` on a run-owned ephemeral port with SPA fallback and runtime configuration matching the actual build. Drive the existing screens and routes through real providers and domain facades. Do not add a fixture gallery or a test-only production route.

Browser fixtures replace HTTP through finite method/path handlers, use generated operation types and share pure payload builders where useful. Do not import the Vitest/MSW server lifecycle into the browser runner. Independently account for unhandled API requests, unexpected external requests and application errors. A caught rejection still fails teardown. Fixture credentials and sessions are explicitly synthetic. Artifacts contain no real credentials, and all target-language labels come from existing dictionaries rather than new foreign-language literals outside them.

Begin with SMTP editing and shared controls, confirmation reached through an existing screen, and WAL chart geometry reached through its actual monitoring screen. Use deterministic monitoring fixtures with sufficient points and freeze time as necessary; preserve existing polling/chart logic. Accessibility scanning targets the shared foundation and pilot. Avoid a global waiver for unrelated legacy screens: fix defects in touched controls and use declared scan regions for the bounded pilot.

Define the separation between UI fixtures and future real-API E2E in configuration and documentation. In this change, requesting the reserved `e2e` project fails clearly as unavailable; it must not succeed with an empty suite or pretend fixtures are real E2E. Milestone 3 supplies that project's backend lifecycle.

Alternatives: happy-dom cannot measure chart layout or real focus behavior; a separate demo application can diverge from production; building the real-backend harness now expands into milestone 3.

### 9. Make visual comparisons reproducible and immediately useful in CI

The public browser command runs the UI project in the same immutable Linux/Chromium/font container on local Fedora and CI. Pin the image by digest, match the browser to the pinned runner version and use the repository-pinned Node/pnpm inside the runner. Isolate dependency/build outputs and temporary state from the host checkout and concurrent runs. Use the existing Docker daemon without pruning or touching shared test services. Any added container setup drops privileges for the test process. Its resource owner waits for readiness, propagates child failure/signals and removes only its own resources.

Start with four reference states: light English desktop SMTP, dark Russian desktop SMTP with validation, light French narrow SMTP, and dark Chinese narrow SMTP. Use fixed viewport, time, synthetic data, reduced motion and loaded local fonts. Wait for application readiness and `document.fonts.ready`, not arbitrary sleeps. The behavioral suite additionally checks keyboard interactions, text enlargement and chart dimensions. Compare declared text/background and focus/status token pairs in both resolved themes; use the documented accessibility thresholds for their role.

The ordinary command uses a non-updating snapshot mode and fails for missing baselines. A separate explicit reference-update command uses the same environment. Changes are reviewed as images; no broad diff tolerance masks a rendering problem. Store useful actual/diff images and traces on failure with synthetic data.

Add `test-frontend-browser-ui` next to frontend tests in `ci-release.yml`, using the public command and locked installation. Include it in `determine-version.needs`, preserving all existing requirements; downstream release publication remains gated by that job. Any new Action uses a full SHA. Verify workflow dependency wiring with a local automated configuration test, without a push or pipeline run.

Alternatives: screenshots from Fedora and Ubuntu can differ despite equal browser versions; system-dependent baselines multiply reference sets; postponing CI to milestone 8 leaves the new suite unenforced. Running a container costs setup time, but bounds the reference environment from the first accepted screenshot.

### 10. Separate durable rules, explanations and change evidence

`docs/frontend-design.md` describes density, spacing, typography/fallback, surfaces, action hierarchy, form/list/table states, feedback, dialogs/drawers, technical text, responsive scrolling, keyboard/focus, chart semantics and why the patterns exist. Values remain in code. `docs/frontend-localization.md` explains runtime ownership, safe whole sentences, exhaustive labels, six complete dictionaries, count wording, region-versus-language and error/notice lifetimes.

The two new skills describe how and when to apply those decisions, with links to documents and code owners. Use the skill-creator workflow when implementing the skills. Frontend AGENTS holds the triggers: design for UI/style/copy, localization for new or changed displayed copy and translations, testing for checks. Move existing detailed i18n rules to their single appropriate owner and replace their old location with links; preserve applicable typography/terminology references to website guidance. The FSD skill keeps placement rules; the new skills do not duplicate them.

Update `docs/architecture.md`, `docs/testing.md` and command descriptions for actual ownership and lifecycle. Do not change review-hook code: its existing verification grammar already permits the exact browser UI command. Record test mapping in `coverage.md`, resolved asset sizes and local verification outcomes in change evidence, and milestone completion only after implementation and required review. Search current docs/skills/config for affected names; archived historical changes stay historical.

Alternatives: token lists in skills drift from code; a growing frontend AGENTS mixes workflow with explanations; changing review infrastructure already prepared by milestone 1 adds no capability.

## Risks / Trade-offs

- Shared-control contract changes affect otherwise untouched areas. Update every caller atomically, run the complete frontend suite and build, and retain existing operation behavior.
- Controlled state can be overwritten by an asynchronous refresh or parent persistence callback. Test language switches, type changes, async initialization, same-ID acknowledgements and reopen through the real editor without a second value owner.
- A completed save can persist an earlier snapshot while a newer draft remains local. Acknowledge only that submitted snapshot and retain the newer draft; never imply the newer values were saved.
- A11y scanning may expose unrelated shell defects. Fix the foundation and the minimum controls needed to reach the pilot, explicitly bound scan regions, and leave the full shell redesign to milestone 3.
- Font subsets can miss accents, Cyrillic or technical glyphs. Check all six languages, preserve fallback and record the shipped asset sizes/license before accepting screenshots.
- Browser setup adds container and cold-install cost. Pin and cache immutable dependencies while preserving run isolation; do not add backend services to this suite.
- Static style analysis is necessarily bounded. Test the declared syntax categories, keep exceptions exact and shrinking, and complement it with resolved-token and browser checks.

## Migration Plan

1. Recheck the checkout and preserve the test map. Add each regression and observe its intended failure immediately before implementing that repair, so earlier integration checks are not left with unrelated failing tests.
2. Establish theme roles, font delivery and accessible shared controls, updating all broken internal contracts together.
3. Implement the SMTP/notifier form and request-revision pilot; migrate its feedback and prove lifecycle/secret behavior.
4. Introduce and reduce the style baseline to cover only untouched consumers. Add browser UI verification, reference images and local harness-failure checks.
5. Wire the browser job into CI and release dependencies; complete docs/skills and run the final local checks and required review.

No HTTP contract, database or persisted preference migration is required. Release rollback uses the previous application image and leaves stored settings intact. During development, remove the replaced internal props and pilot mechanisms directly; no compatibility layer is introduced. Implementation completion stops at a reviewed, verified working tree ready to commit.
