# Frontend design

The interface uses Mantine controls and overlays, Lucide icons, and semantic HTML with Tailwind for layout and spacing. Colors and other presentation values come from semantic roles, so both themes stay readable and one place controls how the application looks. This document covers the theme, forms, feedback and layout, and it ends with the automatic checks.

## Theme and typography

- **Owners.** `frontend/src/shared/theme/themeTokens.ts` owns the light and dark application tokens. `mantineTheme.ts` binds Mantine defaults and public CSS variables to those roles. `ThemeCssVariables.tsx` publishes surface, text, border, focus and status roles, and `frontend/src/index.css` exposes them as semantic Tailwind utilities.
- **Consumers** use these roles or the corresponding Mantine configuration, and raw values stay in the theme owner. The independent WAL chart palette owns its series colors and dash patterns.
- **Roles.** Use `bg-page`, `bg-surface` or `bg-raised` for depth, `text-content` for ordinary text and `text-muted` for secondary text.
- **Status.** Keep status words and icons beside status colors. Color contrast is a visual design choice, without an enforced minimum ratio.
- **Focus** uses a visible outline. Never remove it without a visible keyboard replacement.
- **Fonts.** Jost ships in `src/app/fonts` for Latin and Cyrillic, a system CJK fallback renders Chinese and no font service is needed at runtime. Commands, paths and diagnostics keep a monospace stack, and `InlineCodeComponent` adds a distinct background.

## Density, fields and actions

- **Density.** Use the shared Mantine theme sizes: compact controls use `xs`, ordinary controls use `sm`, and prominent actions use `md`. Keep related fields close and let translated labels wrap. Configuration forms use a compact field size without shrinking readable help text.
- **Labels.** A visible label identifies the actual control. Shared advanced fields require a caller-owned unique `controlId`. The input and its help and error descriptions receive it.
- **Disclosure.** A disclosure is a keyboard button with expanded state and a controlled region. Optional fields stay hidden until requested, and meaningful configured advanced values stay visible when a session opens.
- **Dirty state.** A draft is dirty when it differs from its acknowledged baseline. Submit actions appear only for a dirty draft, so toggling a value back removes them. Dependent fields appear only when their prerequisite has a value.
- **Backup configuration forms** are the exception and keep Save visible, because [`recurring-schedules`](../openspec/specs/recurring-schedules/spec.md) requires it disabled for an incomplete schedule. The logical and physical settings sections also disable it for an unchanged draft, as [`frontend-database-screens`](../openspec/specs/frontend-database-screens/spec.md) says.
- **Settings tests.** A successful test survives name, language and theme changes. Changing settings invalidates it, including a change away and back.
- **Actions.** Group them in a wrapping row. The main operation gets primary emphasis, and destructive styling marks the destructive action itself. Pending buttons expose loading and prevent duplicate activation.
- **Failure.** An unsuccessful operation preserves the draft. Do not automatically retry operations that deliver messages or persist entities.

## Feedback and containers

- **Errors and states.** Put field errors beside fields, and use themed Mantine alerts and loaders with explicit empty-state text for operation or collection states. Keep four states clearly distinct: no records, no results after filtering, loading and a failed load.
- **Feedback.** Use `useFeedback` from `shared/lib/feedback`, backed by the application notification provider. [Localization](frontend-localization.md#errors-and-notice-lifetimes) defines language and notice lifetimes.
- **Dialogs and drawers** have an accessible title, trap focus while open and restore focus to an available invoker. `ConfirmationComponent` also restores focus when its owner unmounts it immediately.
- **Icons and images.** Standard glyphs use `lucide-react`, and provider and engine illustrations stay assets. Meaningful images get an alternative, and decorative images an empty one.
- **Scrolling.** A bounded region has one vertical scroll owner.
  - Use `min-h-0` and `min-w-0` along its flex or grid chain, and do not infer a child's dimensions from an unrelated viewport subtraction.
  - The shell owns the visualViewport height. Pages own their scrolling region, and list-detail pages allocate it through `ResourceListDetail`.
  - Keep technical horizontal scrolling inside its own region, and let ordinary translated text wrap.
- **Small screens.** Support a 320 CSS-pixel viewport and actual 200 percent text enlargement, and never disable browser zoom.
- **Charts** use labelled numeric and time axes, non-color distinctions and positive measured dimensions. Reduced motion disables entry transitions. A DOM-only render cannot establish plot geometry, so the browser suite exercises the production WAL route.

## Enforcement

The style check included in `pnpm --dir frontend lint` parses TypeScript, TSX and CSS, resolves presentation constants and checks color utilities, arbitrary values and focus removal. It accepts semantic references, layout utilities and technical data. Only the theme and WAL palette files define independent colors, and any finding fails lint.

[Testing](testing.md#browser-ui-tests) describes the browser environment, failure artifacts and explicit reference updates.

## Official Mantine documentation

Use the [official LLM guide](https://mantine.dev/guides/llms/) and [documentation index](https://mantine.dev/llms.txt) to find linked per-page Markdown. Read the pages relevant to the control being changed, match the installed major version, and verify minor-specific APIs against the installed package. The index follows the current documentation, so a newer example may require a different installed version.

Repository-local official skills supply reusable patterns; [frontend agent rules](../frontend/AGENTS.md#mantine-skills) own their loading triggers. Keep project conventions when examples differ. Do not copy the full documentation into the repository or change machine-global agent settings; the linked pages provide the required API context.
