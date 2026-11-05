# Frontend guidelines (React + TypeScript)

This document contains coding standards for the Databasus frontend (React 19 + TypeScript + Vite + Ant Design + TailwindCSS).
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

- **AntD 5 only** for components. Don't pull in Mantine, MUI, Chakra, shadcn, or Radix directly. Use AntD primitives (`Button`, `Input`, `Modal`, `Form`, `Table`, `Menu`, `Tabs`, etc.) plus Tailwind utility classes for layout and spacing.
- **`@ant-design/icons` only** for icons. Don't add `lucide-react`, `@heroicons/react`, `react-icons`, or FontAwesome.

---

## React component structure

Write React components with the following structure:

```typescript
interface Props {
   someValue: SomeValue;
}

const someHelperFunction = () => {
    ...
}

export const ReactComponent = ({ someValue }: Props): JSX.Element => {
    // First put states
    const [someState, setSomeState] = useState<...>(...)

    // Then place functions
    const loadSomeData = async () => {
        ...
    }

    // Then hooks
    useEffect(() => {
       loadSomeData();
    });

    // Then calculated values
    const calculatedValue = someValue.calculate();

    return <div> ... </div>
}
```

### Structure order

1. **Props interface** — Define component props
2. **Helper functions** (outside component) — Pure utility functions
3. **Component declaration**
   - **States** — `useState` declarations
   - **Plain functions** — Event handlers, async operations, in-component formatters. Anything that does _not_ call a React hook.
   - **Hooks** — `useRef`, `useLatestRef`, `useEffectEvent`, `useCallback`, `useMemo`, `useEffect`. A function wrapped in `useCallback` is a hook — it lives here, not in the functions section.
   - **Calculated values** — Derived data computed inline (e.g. an AntD `columns` array).
   - **Return** — JSX markup

**All hooks (including every `useEffect`) come below every plain function definition.** If a `useEffect` reads a handler, the handler must be defined above it — don't reorder by putting handlers below the effects.

---

## Effects and rendering

- Declare every dependency an effect reads. Use `useEffectEvent` for a function called only by an effect that must read current values without changing when the effect runs. Async handlers use `useLatestRef`, because `useEffectEvent` cannot serve them.
- A lint exemption applies to one line and includes its reason. Keep `react/set-state-in-effect` off: pending flags and resets at the start of a load are intentional, and rewriting them risks the tested handling of late responses.
- Rendering follows the configured React rules for pure calls, immutable inputs and ref access. Component files export components and constants at runtime.
- Plain elements with click handlers need keyboard interaction and appropriate semantics; images need `alt` text.

---

## Vertical spacing

Follow [Blank lines separate logical steps](../AGENTS.md#blank-lines-separate-logical-steps) in the root rules.

---

## Clipboard operations

Always use `BrowserClipboard` (`shared/lib/BrowserClipboard.ts`) for clipboard operations — never call `navigator.clipboard` directly.

- **Copy:** `BrowserClipboard.copyToClipboard(text)` — uses `navigator.clipboard` with `execCommand('copy')` fallback for non-secure contexts (HTTP).
- **Paste:** Check `BrowserClipboard.isClipboardApiAvailable()` first. If available, use `BrowserClipboard.readFromClipboard()`. If not, show `ClipboardPasteModalComponent` (`shared/ui`) which lets the user paste manually via a text input modal.

---

## Forms

Follow the field, dirty-state and progressive-disclosure conventions in [frontend design](../docs/frontend-design.md#density-fields-and-actions).

---

## Design and localization

Apply the [frontend-design skill](../.agents/skills/frontend-design/SKILL.md) when changing UI layout, styles, controls or visible copy. It links to the design conventions and their code owners.

Apply the [frontend-localization skill](../.agents/skills/frontend-localization/SKILL.md) when adding or changing displayed copy, translations, language-dependent formatting or feedback. [Frontend localization](../docs/frontend-localization.md) holds the dictionary, safe interpolation and regional-format rules, including English punctuation and the website translation-quality reference.

## Architecture and skills

The frontend architecture and the reasons behind its boundaries are in [docs/architecture.md](../docs/architecture.md#frontend). Apply the [feature-sliced-design skill](../.agents/skills/feature-sliced-design/SKILL.md) when placing or reorganizing code, defining public APIs or changing dependencies. Apply the [testing skill](../.agents/skills/testing/SKILL.md) when writing, changing, running or reviewing tests, and the [run-app skill](../.agents/skills/run-app/SKILL.md) before starting or viewing the app.

API request and response types come from the generated contract. Keep generated schema aliases tied to their domain owner; do not handwrite copies of the contract. Other independently meaningful domain or view-state types get their own named logical unit; tiny aliases tightly coupled to that unit may stay with it.

Only `main.tsx` and `vite-env.d.ts` sit at the source root beside the global style sheet; every other script module belongs to a layer.

## Verification

Run `mise run frontend:lint` and `pnpm --dir frontend build` before handing over a frontend change, and check its tests as [`docs/testing.md`](../docs/testing.md#checking-work) says. That document also covers isolating tests and repeating one failing test.
