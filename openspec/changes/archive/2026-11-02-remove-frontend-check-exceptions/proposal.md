## Why

Frontend checks still allow old problems instead of catching them. Lint keeps a baseline, a list of tolerated findings, and four of its entries protect `ToastHelper`, whose fixed white box ignores the dark theme. Eleven browser `alert` dialogs remain on database settings, healthcheck and notifier screens, where the healthcheck view repeats one every minute while the server is unavailable. Icon buttons have no names, and the database tabs cannot be reached from the keyboard. `exhaustive-deps`, `refs`, `purity`, `immutability` and `set-state-in-effect` are off for the whole frontend, so a stale effect goes unnoticed.

Milestone 8 ends the migration. Once these problems are repaired, the exceptions that hide them can go: any dependency or style finding fails lint, and a React rule allows only a commented exception at a single site.

## What Changes

- Replace all `alert` calls and `ToastHelper` with themed feedback, show failed healthcheck and notifier loads on the screen with a retry, and show a failed database action's reason on the screen until the next action, until the user closes it, or until the user leaves the settings tab or selects another database. Delete `shared/toast`.
- Keep the list and selection correct when a copy, rename or delete answers after another tab opened or another database was selected.
- Give icon-only actions, tabs, selection cards, edit icons, logo links and the controls that lack one an accessible name and keyboard operation.
- Delete `fsd-baseline.json`, `style-baseline.json` and the code that reads them, so every finding fails lint.
- Reject source files outside the layers in `lint:fsd`.
- Enable `react-hooks/exhaustive-deps`, `react/refs`, `react/purity` and `react/immutability` as errors, raise `react/only-export-components` to an error, and add the `jsx-a11y` plugin with four rules as errors, after repairing their findings. `react/set-state-in-effect` and the `jsx-a11y` rules the plugin would otherwise warn about stay off, each with its reason recorded.
- Add the `useRequestRevision` and `useLatestRef` hooks that these repairs need.
- **BREAKING (contributors only)**: a contributor can no longer record a tolerated dependency or style finding. Screens and addresses stay the same, except for the feedback above. HTTP requests stay the same unless a reproduced defect requires otherwise; `coverage.md` names each such site.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `frontend-code-checks`: the two baseline requirements are removed. New requirements say that no finding can be recorded, that source files live in a layer, and that effect and interaction rules are enforced.
- `frontend-database-screens`: healthcheck and notifier loads end in content or a recoverable failure, and database actions report their outcome through themed feedback.
- `frontend-design-system`: controls on database screens have names and keyboard operation.

## Impact

This is the second of three changes of frontend milestone 8. It depends on `clean-frontend-residue` and comes before `close-frontend-behavior-matrix`. The root `AGENTS.md` and `frontend/AGENTS.md` apply. The work uses the frontend-design, frontend-localization, feature-sliced-design and testing skills.

Affected areas:

- `frontend/src/pages/databases`, for feedback, controls and most rule repairs;
- `frontend/src/shared/toast`, which is deleted, and `frontend/src/shared/hooks`, which gains the two hooks;
- other slices with rule findings: `app`, `pages/{auth,instance-settings,notifiers,profile,storages,users,workspace-settings}`, `features/{database-connection,edit-notifier,edit-storage}` and `entities/backup-configs`;
- `frontend/src/shared/i18n/locales`, for new names in six dictionaries;
- `frontend/src/widgets/app-shell` and `frontend/src/pages/auth`, for logo links;
- `frontend/lint`, `frontend/oxlint.config.ts`, `frontend/tooling/styleBaseline.test.ts` and `frontend/src/shared/lib/test/frontendArchitecture.test.ts`;
- `docs/architecture.md`, `docs/frontend-design.md`, `docs/testing.md`, `frontend/AGENTS.md`, the feature-sliced-design and frontend-design skills, `.agent-hooks/architecture-prompt.md` and `.agent-hooks/review_gate_test.py`.

There is no backend, HTTP contract, generated code or dependency change.

### Out of Scope

- Tests for the behavior matrix unrelated to these repairs, which `close-frontend-behavior-matrix` adds.
- Changing polling intervals. Request counts or order change only for a reproduced defect pinned by a test.
- Rewriting effect loads to avoid `setState` in effects.
- A shared component for the `Alert` with a retry, which many files already write inline.
- `jsx-a11y/control-has-associated-label`. Of its 17 findings, 15 are links and buttons whose translated text the rule cannot see, and the 2 real ones are repaired here.
