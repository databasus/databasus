# Frontend localization

The interface ships in English, Russian, Spanish, Portuguese (Brazilian), Chinese (Simplified), French and Indonesian, through `i18next` and `react-i18next`. A language switch changes words and number grouping immediately, without remounting providers, screens or drafts. Clock and date conventions follow the browser region instead.

The [shared localization layer](../frontend/src/shared/i18n) owns the selected language, dictionaries, library integration and translated control labels. The document language and title follow the same selection. Updating language never remounts controls; date order and clock format still follow the browser region.

## User-facing copy

- **English.** In `shared/i18n/locales/en.ts`, use a plain hyphen `-` in any string the user will see. Em dashes (`—`) and en dashes (`–`) are for markdown docs and code comments only.
- **Other languages** follow their own typography, as [`website/AGENTS.md` - Translation quality](../website/AGENTS.md#translation-quality) sets out: Russian keeps «—» where its grammar wants it, French puts a space before `: ; ! ?` and `%`, and Chinese uses full-width punctuation. The same section lists the canonical interface terms per language.
- **Translations** are edited copy in the target language.

## Dictionaries

- New copy goes into all seven dictionaries in the same change.
- English (`locales/en.ts`) defines the keys. Every other dictionary is typed `typeof en`, so a missing or leftover key fails `pnpm build`, and a language cannot ship partly translated.
- Each dictionary is one object literal in one file. TypeScript's check for leftover keys works only on a literal written in place, so a nested object passed through a variable or a spread escapes it.
- Keys are grouped by domain (`backups`, `databases`, `storages`, `status`, `errors`, `common`, ...), so moving a component does not rename its keys.
- Dictionary checks require every translation to use the same placeholders and markup tags as English.

## Rules for code

### Literals

- `src` holds no user-facing string literals. The `i18next/no-literal-string` lint rule enforces this, including SCREAMING_SNAKE constants and default parameter values.
- Technical strings stay literal: commands, code blocks, connection strings, cron expressions, engine and tool names, file paths, `localStorage` keys.
- A technical string the rule reports takes `// eslint-disable-next-line i18next/no-literal-string -- <reason>`. In JSX, make it an expression on its own line (`{'pg_dump'}`) so the directive applies to it.
- A file with nothing but technical strings, such as a shell command builder, may disable the rule once at the top with the reason.

### Reading text

- Translate during rendering through the active language context. Code outside rendering and persistent state hold typed keys, error data or keys with parameters; translating them early would freeze the language.
- Displayed enums have exhaustive, typed label-key tables beside their domain type, exposed through the slice's public API. Never build a key dynamically from a value: adding an enum member must reveal a missing label at build time.

### Sentences

- Sentences stay whole. A sentence with inline markup is one dictionary key with named tags, such as `Run <code>pg_dump</code> first`. Render it through typed shared localization primitives; variable keys need the same type safety as literal keys.
- Values are interpolated into a whole sentence (`'Delete {{name}}?'`), never concatenated with translated fragments. When the inserted word would inflect in other languages, each value gets its own sentence key.
- A user-entered value in a `Trans` sentence, such as a workspace name, is a self-closing tag, and the supplied element holds the value: dictionary `Delete <workspaceName/>?`, component `components={{ workspaceName: <strong>{name}</strong> }}`. Never pass it through `values`, because `Trans` would expand placeholder syntax inside it.
- Inline code uses the shared presentation primitive. A monospace font alone does not set code apart from surrounding text.
- Counts are phrased so one string fits every number: `Members: {{count}}`, not `{{count}} members`. The dictionaries have no plural forms, because Russian's `_few` would be a stray key under `typeof en`.
- Translated text is never rendered as HTML. No `dangerouslySetInnerHTML`.

### Formatting, errors and links

- Text, number and relative-time formatting use the active shared locale context, so a language switch re-renders them. Bare browser or date-library formatting can select the wrong language.
- API errors use the shared translator and the error's machine-readable code. Every contract-declared code has a translation; raw backend messages never become interface copy.
- Website links use the shared page registry and locale-aware URL builder, selecting the published translation when available.

## Clock format and date order

The browser's regional locale determines the 12- or 24-hour clock and date order. The interface language determines words and number grouping. Keep those choices separate through the shared time and localization layers rather than using a bare `toLocaleString()`.

## Errors and notice lifetimes

- **Persistent errors** stay API errors, translation keys with parameters or supported delivery diagnostics. They are translated during rendering, so switching language updates a visible error.
- **Diagnostic values** render as React text, so angle brackets, braces, hostnames and commands keep their literal characters.
- **Success notices** capture their language when created and expire after their original lifetime. A locale switch leaves a visible notice and its deadline as they are.
- **Editing sessions.** Closing one destroys its own notice, and late results from an obsolete session cannot create one.
- **Forms.** Field errors render beside their controls and derive from the current state. A language or theme change keeps values, newer edits, expanded settings and successful testing of unchanged settings.

## Verification

Run dictionary tests, the affected screen tests and frontend lint. For shared localization behavior, also run the production browser UI command from [testing.md](testing.md#browser-ui-tests). Non-English assertions read the dictionaries, and every other file under `frontend/` stays in English.
