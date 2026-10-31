## Why

Frontend milestone 2 needs a common visual and localization foundation before the remaining screens move to their final owners. Shared controls currently lack some keyboard and label semantics, styling bypasses the theme, and a late settings-test response can make an edited notifier draft appear safe to save.

## What Changes

- Extend the existing AntD theme with semantic roles shared by AntD, CSS and Tailwind, including defined scrollbar colors and an explicit chart-palette boundary.
- Document the existing visual language and introduce focused design-system and frontend-localization skills. Preserve the six dictionaries, domain keys, safe interpolation and the distinction between interface language and browser region.
- Repair shared field, disclosure, confirmation and language/theme controls; allow browser zoom; use context-bound AntD feedback in the pilot. Supply the necessary Jost fonts locally with their license and a CJK fallback.
- **BREAKING (frontend source contracts only):** require an explicit control ID on the shared advanced-field row and replace confirmation's color-based action prop with an explicit destructive-action flag. Update every caller and delete the old prop shapes without aliases. HTTP and persisted data are unaffected.
- Use the notifier editor with its SMTP fields as the representative form pilot. Keep its existing controlled name/settings values as the single draft owner, derive dirty state from an acknowledged baseline and add explicit initialization/reopen behavior and request-revision guards. Use AntD field presentation without introducing another value store. Preserve all existing notifier variants and their secret-omission behavior.
- **BREAKING (frontend source contracts only):** add a required persistence callback to the notifier editor, separate from its existing completion callback, and update all callers. A successful creation retains its returned ID and newer local edits; the next save updates that notifier rather than creating another one, and refreshing available saved notifiers does not close the editor. Lock provider type during pending creation and keep it immutable after success, preserving the saved-notifier type rule.
- Establish a bounded flex/grid layout pilot with an explicit scroll owner. Keep the full shell rewrite for milestone 3.
- Add style checks with a finite shrinking baseline, plus browser UI checks against the production build for themes, localization, keyboard/focus, narrow layouts and real chart geometry. Include accessibility scanning and a small visual baseline in a reproducible environment.
- Add a browser UI CI check and make its success a prerequisite for release. Real-backend browser E2E remains milestone 3.

## Capabilities

### New Capabilities

- `frontend-design-system`: consistent theme semantics, accessible shared controls, local typography and responsive form presentation.

### Modified Capabilities

- `frontend-localization`: active form feedback follows language changes, existing transient success notices have an explicit bounded language-switch exception, and language selection remains usable from the keyboard.
- `notifiers`: the notifier editor binds test/save outcomes to the active draft and preserves edits, secrets and explicit delivery actions.
- `frontend-code-checks`: new style violations fail checks while individually recorded migration residue can only shrink.
- `frontend-screen-testing`: production-build browser UI checks complement the existing screen tests with reproducible visual and accessibility coverage.
- `ci-pipeline`: the browser UI check runs the local entry point and gates release.

## Impact

The change answers to root `AGENTS.md` and `frontend/AGENTS.md`. It touches `frontend/src/shared/{theme,ui,i18n}`, the notifier editor and the minimum enclosing layout needed for its pilot, frontend style/browser tooling, `.github/workflows/ci-release.yml`, `mise.toml`, frontend documentation and the two new repository skills. All callers of changed frontend source contracts are updated together, including database composition that embeds the notifier editor; removed contracts have no compatibility aliases.

Development dependencies will include a pinned Playwright runner and accessibility scanner, and local font assets will include their source and license. The browser environment is pinned and shared between local checks and CI. HTTP contracts, stored settings, application URLs and persisted locale/theme preferences remain unchanged. New translations stay in the six existing interface dictionaries.

### Out of scope

- A new visual brand, another UI kit, a separate library of base controls, or a new form/state-management framework.
- Mass conversion of storage, database, backup or verification forms; domain moves reserved for milestones 3-7.
- Authentication/session refactoring, new routes, deep links, Google Drive OAuth or SFTP credential changes.
- A real-backend browser harness, backend engine matrices, new HTTP operations or schema migrations.
- Dictionary splitting, new languages, a new pluralization model or broad copy rewriting unrelated to the pilot.
- Global removal of every legacy toast, color or computed-height consumer; untouched occurrences remain assigned to their existing milestones.
- Committing, pushing, deploying or publishing this change.
