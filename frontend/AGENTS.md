# Frontend guidelines (React + TypeScript)

This document contains coding standards for the Databasus frontend (React 19 + TypeScript + Vite + Mantine + TailwindCSS).
The rules for every module are in the root `AGENTS.md`.

---

## Table of Contents

- [UI kit and icons](#ui-kit-and-icons)
- [React component structure](#react-component-structure)
- [Effects and rendering](#effects-and-rendering)
- [Vertical spacing](#vertical-spacing)
- [Clipboard operations](#clipboard-operations)
- [Forms](#forms)
- [Design and localization](#design-and-localization)
- [Architecture and skills](#architecture-and-skills)
- [Verification](#verification)

---

## UI kit and icons

- Use **Mantine** for interactive controls and overlays.
- Use semantic HTML with Tailwind for layout and spacing.
- Import Mantine components directly. Create a focused component in `shared/ui` only when shared application behavior needs one; do not create a wrapper for every library component.
- Use **`lucide-react`** for standard icons.
- Keep provider and engine illustrations as assets.

---

### Mantine skills

Read the applicable repository-local official skill before specialized work:

- [mantine-form](../.agents/skills/mantine-form/SKILL.md): `@mantine/form` values, validation, lists, controlled/uncontrolled mode or form context.
  - Existing state does not need to move to `@mantine/form` merely to use Mantine controls.
- [mantine-combobox](../.agents/skills/mantine-combobox/SKILL.md): searchable, multiple or custom selection, option rendering, filtering or `Combobox`/`ComboboxPopover` interaction.
- [mantine-custom-components](../.agents/skills/mantine-custom-components/SKILL.md): Mantine-integrated shared components, Styles API, factories, theme extensions or CSS-variable contracts.

When an upstream example differs, project conventions govern theme ownership, localization, accessibility and state lifetimes.

Documentation lookup is described in [frontend design](../docs/frontend-design.md#official-mantine-documentation).

## React component structure

- Group related code.
- Arrange declarations so their dependencies are easy to follow from top to bottom.
- There is no mandatory order for state, hooks, derived values, handlers or effects. Do not reorder code or raise review findings solely to enforce a section order.

---

## Effects and rendering

- Declare every dependency an effect reads.
- Use `useEffectEvent` for a function that only an effect calls, when the function must read current values without changing when the effect runs.
- Async handlers use `useLatestRef`, because `useEffectEvent` cannot serve them.
- A lint exemption applies to one line and includes its reason.
- Keep `react/set-state-in-effect` off. Pending flags and resets at the start of a load are intentional. Rewriting them risks the tested handling of late responses.
- Rendering follows the configured React rules for pure calls, immutable inputs and ref access.
- Component files export components and constants at runtime.
- Plain elements with click handlers need keyboard interaction and appropriate semantics.
- Images need `alt` text.

---

## Vertical spacing

Follow [Blank lines separate logical steps](../AGENTS.md#blank-lines-separate-logical-steps) in the root rules.

---

## Clipboard operations

Always use `BrowserClipboard` (`shared/lib/BrowserClipboard.ts`) for clipboard operations. Never call `navigator.clipboard` directly.

- **Copy:** call `BrowserClipboard.copyToClipboard(text)`. It uses `navigator.clipboard`, with an `execCommand('copy')` fallback for non-secure contexts such as plain HTTP.
- **Paste:**
  1. Check `BrowserClipboard.isClipboardApiAvailable()` first.
  2. If it is available, use `BrowserClipboard.readFromClipboard()`.
  3. If not, show `ClipboardPasteModalComponent` (`shared/ui`). It lets the user paste manually into a text input modal.

---

## Forms

Follow the field, dirty-state and progressive-disclosure conventions in [frontend design](../docs/frontend-design.md#density-fields-and-actions).

---

## Design and localization

Apply the [frontend-design skill](../.agents/skills/frontend-design/SKILL.md) when changing UI layout, styles, controls or visible copy. It links to the design conventions and their code owners.

Apply the [frontend-localization skill](../.agents/skills/frontend-localization/SKILL.md) when adding or changing displayed copy, translations, language-dependent formatting or feedback.

[Frontend localization](../docs/frontend-localization.md) holds the dictionary, safe interpolation and regional-format rules. They include English punctuation and the website translation-quality reference.

## Architecture and skills

The frontend architecture and the reasons behind its boundaries are in [docs/architecture.md](../docs/architecture.md#frontend).

Apply these skills:

- [feature-sliced-design](../.agents/skills/feature-sliced-design/SKILL.md) when placing or reorganizing code, defining public APIs or changing dependencies.
- [testing](../.agents/skills/testing/SKILL.md) when writing, changing, running or reviewing tests.
- [run-app](../.agents/skills/run-app/SKILL.md) before starting or viewing the app.

Types:

- API request and response types come from the generated contract.
- Keep generated schema aliases tied to their domain owner. Do not handwrite copies of the contract.
- Other independently meaningful domain or view-state types get their own named logical unit. Tiny aliases tightly coupled to that unit may stay with it.

Only `main.tsx` and `vite-env.d.ts` sit at the source root, beside the global style sheet. Every other script module belongs to a layer.

### Import paths

Use `@/` for every literal module reference from a script in `src` to a target within `src`. This includes sibling imports, type imports, re-exports, dynamic imports and side-effect stylesheet imports.

- Keep direct imports within the same slice.
- Use another slice's public API when crossing a slice boundary.

Frontend lint enforces this convention.

Exceptions:

- References from source tests to tooling outside `src` may stay relative.
- Imports originating outside `src` may stay relative.
- CSS resource URLs, `new URL(..., import.meta.url)`, package imports and source-code strings inside test fixtures retain their own path semantics. Do not rewrite them as part of an import migration.

## Verification

Before handing over a frontend change:

1. Run `mise run frontend:lint`.
2. Run `pnpm --dir frontend build`.
3. Check its tests as [`docs/testing.md`](../docs/testing.md#checking-work) says.

That document also covers isolating tests and repeating one failing test.
