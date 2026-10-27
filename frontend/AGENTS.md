# Frontend guidelines (React + TypeScript)

This document contains coding standards for the Databasus frontend (React 19 + TypeScript + Vite + Ant Design + TailwindCSS).
For project-wide engineering philosophy, see the root `AGENTS.md`.

---

## Table of Contents

- [UI kit and icons](#ui-kit-and-icons)
- [React component structure](#react-component-structure)
- [Vertical spacing](#vertical-spacing)
- [Clipboard operations](#clipboard-operations)
- [Forms](#forms)
- [User-facing copy](#user-facing-copy)
- [Interface languages (i18n)](#interface-languages-i18n)
- [FSD (Feature-Sliced Design)](#fsd-feature-sliced-design)
- [Refactoring](#refactoring)

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
   - **Hooks** — `useRef` + ref mutation, `useCallback`, `useMemo`, `useEffect`. A function wrapped in `useCallback` is a hook — it lives here, not in the functions section.
   - **Calculated values** — Derived data computed inline (e.g. an AntD `columns` array).
   - **Return** — JSX markup

**All hooks (including every `useEffect`) come below every plain function definition.** If a `useEffect` reads a handler, the handler must be defined above it — don't reorder by putting handlers below the effects.

---

## Vertical spacing

- Structure function bodies with vertical rhythm. Put a blank line between logically distinct steps (setup / main work / return, independent branches, before and after a guard). Do not leave blank lines at the start or end of a body, never stack two in a row. Do not insert blanks inside a tight expression, a single statement split across lines, or a short (≤ 5-line) function. If blank lines alone aren't enough to navigate a body, extract — don't add comments.

---

## Clipboard operations

Always use `ClipboardHelper` (`shared/lib/ClipboardHelper.ts`) for clipboard operations — never call `navigator.clipboard` directly.

- **Copy:** `ClipboardHelper.copyToClipboard(text)` — uses `navigator.clipboard` with `execCommand('copy')` fallback for non-secure contexts (HTTP).
- **Paste:** Check `ClipboardHelper.isClipboardApiAvailable()` first. If available, use `ClipboardHelper.readFromClipboard()`. If not, show `ClipboardPasteModalComponent` (`shared/ui`) which lets the user paste manually via a text input modal.

---

## Forms

### Progressive disclosure

- Submit buttons (`Save`, `Update password`, etc.) render **only when the form is dirty**.
- Dependent fields (e.g. `Confirm password`) render **only after the field they depend on has a value**.

---

## User-facing copy

In the English dictionary (`shared/i18n/locales/en.ts`), use a plain hyphen `-` in any string the user will see — labels, descriptions, notifications, modal bodies, error messages. Reserve em dashes (`—`) and en dashes (`–`) for markdown docs and code comments only.

The other dictionaries follow their language's own typography instead, as set out in [`website/AGENTS.md` - Translation quality](../website/AGENTS.md#translation-quality): Russian keeps «—» where its grammar wants it, French puts a space before `: ; ! ?` and `%`, Chinese uses full-width punctuation. The same section lists the canonical interface terms per language.

---

## Interface languages (i18n)

The interface ships in English, Russian, Spanish, Portuguese (Brazilian), Chinese (Simplified), French and Indonesian, through `i18next` and `react-i18next`. Everything lives in `shared/i18n`.

### Dictionaries

- English (`locales/en.ts`) is the source of truth for the set of keys. Every other dictionary is typed `typeof en`, so a missing or leftover key fails `pnpm build`. A language cannot ship partly translated.
- Each dictionary is one object literal in one file. TypeScript's check for leftover keys only works on a literal written in place; a nested object passed in through a variable or a spread escapes it.
- Keys are grouped by domain (`backups`, `databases`, `storages`, `status`, `errors`, `common`, ...), not by file path, so moving a component does not rename its keys.
- `dictionaries.test.ts` checks that every translation uses the same `{{placeholders}}` and `Trans` tags as English.
- Translations are written in the target language, not rendered word for word. Follow [`website/AGENTS.md` - Translation quality](../website/AGENTS.md#translation-quality).

### Rules for code

- No user-facing string literal in `src`. The `i18next/no-literal-string` lint rule enforces it, SCREAMING_SNAKE constants and default parameter values included. Technical strings stay literal: commands, code blocks, connection strings, cron expressions, engine and tool names, file paths, `localStorage` keys. A technical string the rule reports takes `// eslint-disable-next-line i18next/no-literal-string -- <reason>`; in JSX, make it an expression on its own line (`{'pg_dump'}`) so the directive applies to it. A file that holds nothing but technical strings, such as a shell command builder, may disable the rule once at the top with the reason.
- Components read text with `const { t } = useTranslation()` from `react-i18next`.
- Code outside components holds keys, not text. A module-level constant, a label table or a parser that called `t()` would freeze the language that was active when it ran. Such code stores a `TranslationKey`, or a `LocalizedText` (`{ key, params }`) when the message has parameters, and the component translates it while rendering (`t(key)`, `translateLocalizedText(text, t)`). The same goes for state: keep an error or a key in state, not a translated string.
- Every enum shown to the user has a label table `<TYPE>_LABEL_KEYS: Record<Enum, TranslationKey>` next to the type it labels, exported through the slice's `index.ts`. Never build a key from a value (``t(`status.${value}`)``): adding an enum member must fail the build.
- Sentences stay whole. A sentence with inline markup is one key rendered through `<Trans i18nKey="..." components={{ code: <InlineCodeComponent />, bold: <strong />, docsLink: <a href={...} /> }} />` (`<TransByKey>` from `shared/i18n` when the key is held in a variable: `Trans` cannot type-check the whole key union), with named tags in the dictionary (`Run <code>pg_dump</code> first`). Values are interpolated into a whole sentence (`'Delete {{name}}?'`), never concatenated with translated fragments. When the inserted word would inflect in other languages, each value gets its own sentence key.
- A user-entered value (a workspace, database or user name) in a `Trans` sentence is a self-closing tag whose supplied element holds the value: dictionary `Delete <workspaceName/>?`, component `components={{ workspaceName: <strong>{name}</strong> }}`. Never pass it through `values`: `Trans` would expand placeholder syntax inside it.
- Inline code in prose, inside a `Trans` sentence or not, goes through `InlineCodeComponent` from `shared/ui`, not a bare `<code>`: a monospace font alone doesn't set code apart from the text around it.
- Counts are phrased so one string fits every number: `Members: {{count}}`, not `{{count}} members`. The dictionaries have no plural forms, because Russian's `_few` would be a stray key under `typeof en`.
- Translated text is never rendered as HTML. No `dangerouslySetInnerHTML`.
- Locale-dependent output comes from hooks, so a language switch re-renders it: `useTranslation` for text, `useLocale()` for `formatNumber` (instead of `toLocaleString()`) and `formatRelativeTime` (instead of dayjs `fromNow()`).
- API errors reach the user only through `translateApiError(error, t)`: a known error code gives its translation, otherwise the backend's message, otherwise a general message. Never show `error.message` directly.
- Links to databasus.com pages go through `getWebsitePageUrl(pageId, locale)` with a page from `WEBSITE_PAGES`, which opens the page in the selected language when the website publishes it.

### Clock format and date order

`shared/time/getUserTimeFormat.ts` and `shared/time/utils.ts` read `navigator.language` on purpose. The 12- or 24-hour clock and the day/month order are regional conventions, not properties of a language: English is written with both clocks. The browser reports the user's region, while the language selection only says which words to use. Those reads stay; a bare `toLocaleString()` does not, because digit grouping follows the selected language.

---

## FSD (Feature-Sliced Design)

The project follows FSD v2.1. Layers in use: `app/`, `pages/`, `widgets/`, `features/`, `entity/`, `shared/`.

### Import direction

`app → pages → widgets → features → entity → shared`

A module may only import from layers strictly **below** it. Cross-imports between slices on the **same** layer are forbidden.

Paths below are written from `src/` for readability. Real imports are relative; the project has no path alias.

```text
✅ Allowed
  a widget       imports  features/backups, entity/backups, shared/ui
  a feature      imports  entity/databases, shared/api
  an entity      imports  shared/lib

❌ Violations
  an entity      imports  features/databases      (upward: entity sits below features)
  a feature      imports  pages/AuthPageComponent (upward: pages sits above features)
  features/users imports  features/backups        (same layer: cross-slice import)
  shared/lib     imports  entity/users            (upward: shared is the bottom layer)
```

### Where new code goes

- Used in only one page → keep it in that `pages/` slice.
- Reusable infrastructure, **no business logic** → `shared/` (UI kit, utils, API client, route constants, auth tokens, CRUD helpers).
- User interaction reused in 2+ pages → `features/`.
- Domain model reused in 2+ pages/features → `entity/`.
- App-wide providers, router, theme → `app/`.

**When in doubt — keep it in `pages/`.** Extract only when a second real consumer appears.

### Quick placement table

| Scenario                 | Single use                            | Reused in 2+ places                   |
| ------------------------ | ------------------------------------- | ------------------------------------- |
| Profile form             | `pages/profile/ui/ProfileForm.tsx`    | `features/profile-form/`              |
| Database card            | `pages/databases/ui/DatabaseCard.tsx` | `entity/database/ui/DatabaseCard.tsx` |
| Data fetching for backup | `pages/backup/api/fetch-backup.ts`    | `entity/backup/api/`                  |
| Auth token / session     | `shared/auth/` (always)               | `shared/auth/` (always)               |
| Login form               | `pages/login/ui/LoginForm.tsx`        | `features/auth/`                      |
| CRUD helpers             | `shared/api/` (always)                | `shared/api/` (always)                |
| Date formatting util     | —                                     | `shared/lib/format-date.ts`           |
| Modal content            | `pages/[page]/ui/SomeModal.tsx`       | —                                     |

### MUST rules

1. **Downward-only imports.** No upward imports, no same-layer cross-imports.
2. **Public API via `index.ts`.** External consumers import only from a slice's `index.ts`, never its internal files.
   ```text
   ✅  features/users
   ❌  features/users/ui/AuthNavbarComponent
   ```
3. **Domain-based file names.** Name files by the domain they represent, not their technical role.
   ```text
   // ❌  model/types.ts, model/utils.ts, lib/helpers.ts
   // ✅  model/user.ts, model/backup.ts, api/fetch-database.ts
   ```
4. **No business logic in `shared/`.** Shared holds only infrastructure. Domain calculations live in `entity/` or higher.
5. **One type per file in `model/` / DTOs.** Each `interface`, `class`, or `enum` that represents a domain entity, DTO, request body, or response shape gets its own file in `model/` (or `models/`), named after the type. Don't co-locate sibling types like `Foo` + `FooStatus` + `FooResponse` in a single `Foo.ts` — split them. Re-export each from the slice `index.ts` so consumers still import from the slice's public API.
   ```text
   // ❌  model/RestoreVerification.ts — enums + table-stat + main interface all in one file
   // ✅  model/RestoreVerification.ts        — interface RestoreVerification
   // ✅  model/RestoreVerificationTableStat.ts — interface RestoreVerificationTableStat
   // ✅  model/VerificationStatus.ts         — enum VerificationStatus
   // ✅  model/VerificationTrigger.ts        — enum VerificationTrigger
   ```
   Tiny shape-only helper types tightly coupled to a parent type (a literal-union alias, a `Pick<>`) may stay in the parent's file. Splitting applies to anything with its own identity — anything you'd reasonably import on its own elsewhere.

### Segments inside a slice

- `ui/` — components, styles
- `model/` — state, types, domain logic, validation
- `api/` — backend calls, request functions, API-specific types
- `lib/` — internal helpers for this slice only
- `config/` — slice-level config / feature flags

`app/` and `shared/` have **segments only, no slices**. Segments within them may import from each other.

### AVOID

- Creating an entity prematurely (single consumer → keep it in the page).
- Putting CRUD inside `entity/` — CRUD is infrastructure, goes to `shared/api/`.
- Creating a `user` entity just for auth tokens/DTOs — those belong in `shared/auth/` or `shared/api/`.
- Extracting single-use code "for future reuse."
- God slices (`user-management/` covering auth + profile + password) — split by focused responsibility.
- Importing UI segments of one entity from another entity. Entity UI may be imported only from features/widgets/pages.
- Abusing the `@x` cross-import pattern — it is a last resort, not a tool.

### Cross-imports between same-layer slices

When two slices on the same layer need to share code, try **in order**:

1. **Merge** — if they always change together, they are one slice.
2. **Extract shared logic down to `entity/`** — keep UI in the features/widgets.
3. **Compose in a higher layer (IoC)** — the parent page/widget imports both and wires them together via props/slots.
4. **`@x` notation** — explicit, documented cross-import between entities only. Last resort.

---

## Refactoring

When applying changes, **do not forget to refactor old code**. You can shortify, make more readable, improve code quality, etc. Common logic can be extracted to functions, constants, files, etc.

**After each large change with more than ~50-100 lines of code:**

- Run `pnpm format` (from `frontend/` root folder)
- Run `pnpm lint` to verify the change
