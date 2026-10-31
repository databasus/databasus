# Frontend design

The interface uses AntD components and icons with Tailwind for layout and spacing. Shared controls and the SMTP notifier editor establish the presentation foundation; the remaining domain screens migrate in their own milestones.

## Theme and typography

`frontend/src/shared/theme/getAntdTheme.ts` owns the light/dark AntD algorithms and contrast-adjusted aliases. `ThemeCssVariables.tsx` derives application surface, text, border, focus and status roles from those resolved tokens. `frontend/src/index.css` exposes semantic Tailwind utilities and document/scrollbar styles. Use these roles or AntD tokens in consumers; keep raw values in the exact theme owner. The independent WAL chart palette owns series colors and dash patterns.

Use `bg-page`, `bg-surface` or `bg-raised` for their corresponding depth; `text-content` for ordinary text and `text-muted` for secondary text. Keep status words and icons alongside status colors. Text links and status text have at least 4.5:1 contrast on the declared surface in both modes. Focus uses a visible outline; never remove it without a visible keyboard replacement.

Jost variable normal fonts are shipped in `src/app/fonts` for Latin, extended Latin and Cyrillic. Their source manifest records the package, hashes and sizes; the adjacent license travels with the assets. The explicit system CJK fallback renders Chinese. No font service is needed at runtime. Commands, paths and diagnostics keep a monospace stack; `InlineCodeComponent` also provides a distinct background. Do not force the body family onto every element.

## Density, fields and actions

Prefer AntD's existing sizes to a separate control scale. Keep related fields close, separate groups with consistent spacing and allow translated labels to wrap. Use a compact field size for configuration forms without shrinking readable help text. `Form.Item` can present controlled values without registering another form store.

A visible label identifies the actual control. Shared advanced fields require a caller-owned unique `controlId`; the input receives that ID and the corresponding help/error descriptions. A disclosure is a keyboard button with expanded state and a controlled region. Hide optional fields until requested while retaining meaningful configured advanced values when a session opens.

Submit actions appear only for a dirty draft. Dependent fields appear only when their prerequisite has a value. An acknowledged baseline defines dirty state; toggling a value back removes the actions. A successful settings test survives name, language and theme changes. Changing settings invalidates it, including a change away and back.

Group actions in a wrapping row. Give the main operation primary emphasis; cancellation is ordinary. Destructive styling identifies the destructive action itself. Pending buttons expose loading and prevent duplicate activation. An unsuccessful operation preserves the draft. Do not automatically retry operations that deliver messages or persist entities.

## Feedback and containers

Use field errors beside fields and themed AntD error/loading/empty primitives for operation or collection states. Preserve a clear distinction between no records, filtered no results, loading and a failed load. Tables and lists retain their existing selection and permission rules. Use context-bound AntD feedback; [localization](frontend-localization.md#errors-and-notice-lifetimes) defines language and notice lifetimes.

Dialogs and drawers have an accessible title, trap focus while open and restore focus to an available invoker. `ConfirmationComponent` also restores focus when its owner unmounts it immediately. Standard glyphs use AntD icons; provider/engine illustrations stay assets. Give meaningful images an alternative and decorative images an empty one.

A bounded region has one vertical scroll owner. Use `min-h-0` and `min-w-0` along its flex/grid chain; do not infer a child's dimensions from an unrelated viewport subtraction. The notifier region consumes its existing shell height, while the detail panel scrolls internally. The shell's overall height and visualViewport lifecycle remain a separate migration. Keep technical horizontal scrolling inside its own region and allow ordinary translated text to wrap. Support a 320 CSS-pixel viewport and actual 200 percent text enlargement; never disable browser zoom.

Charts use labelled numeric/time axes, non-color distinctions and positive measured dimensions. Reduced motion disables chart entry transitions. A DOM-only render cannot establish plot geometry; the browser suite exercises the production WAL route.

## Enforcement and migration

`pnpm --dir frontend lint:styles` parses TypeScript/TSX and CSS, resolves presentation constants and checks color utilities, arbitrary values and focus removal. It accepts semantic references, layout utilities and technical data. Only the exact theme and WAL palette files define independent colors. The finite `frontend/style-baseline.json` records remaining constructs and counts with an owner and removal milestone. Repairing an occurrence requires removing its stale baseline entry; copying one fails lint.

Shared UI and the SMTP pilot have no baseline entries. Other domain screens, the legacy toast implementation and shell presentation remain assigned to their migration milestones. Avoid expanding those exceptions. [Testing](testing.md#browser-ui-tests) describes the browser environment, failure artifacts and explicit reference updates.
