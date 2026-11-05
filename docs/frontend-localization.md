# Frontend localization

The interface ships in English, Russian, Spanish, Portuguese (Brazilian), Chinese (Simplified), French and Indonesian, through `i18next` and `react-i18next`. A language switch changes words and number grouping immediately, without remounting providers, screens or drafts. Clock and date conventions follow the browser region instead.

Everything lives in `shared/i18n`. `LocaleProvider.tsx` owns the i18next instance, the selected locale, the AntD locale and the dayjs language, and the document language and title follow the same selection.

## User-facing copy

- **English.** In `shared/i18n/locales/en.ts`, use a plain hyphen `-` in any string the user will see. Em dashes (`—`) and en dashes (`–`) are for markdown docs and code comments only.
- **Other languages** follow their own typography, as [`website/AGENTS.md` - Translation quality](../website/AGENTS.md#translation-quality) sets out: Russian keeps «—» where its grammar wants it, French puts a space before `: ; ! ?` and `%`, and Chinese uses full-width punctuation. The same section lists the canonical interface terms per language.
- **Translations** are edited copy in the target language.

## Dictionaries

- New copy goes into all seven dictionaries in the same change.
- English (`locales/en.ts`) defines the keys. Every other dictionary is typed `typeof en`, so a missing or leftover key fails `pnpm build`, and a language cannot ship partly translated.
- Each dictionary is one object literal in one file. TypeScript's check for leftover keys works only on a literal written in place, so a nested object passed through a variable or a spread escapes it.
- Keys are grouped by domain (`backups`, `databases`, `storages`, `status`, `errors`, `common`, ...), so moving a component does not rename its keys.
- `dictionaries.test.ts` checks that every translation uses the same `{{placeholders}}` and `Trans` tags as English.

## Rules for code

### Literals

- `src` holds no user-facing string literals. The `i18next/no-literal-string` lint rule enforces this, including SCREAMING_SNAKE constants and default parameter values.
- Technical strings stay literal: commands, code blocks, connection strings, cron expressions, engine and tool names, file paths, `localStorage` keys.
- A technical string the rule reports takes `// eslint-disable-next-line i18next/no-literal-string -- <reason>`. In JSX, make it an expression on its own line (`{'pg_dump'}`) so the directive applies to it.
- A file with nothing but technical strings, such as a shell command builder, may disable the rule once at the top with the reason.

### Reading text

- Components read text with `const { t } = useTranslation()` from `react-i18next`.
- Code outside components holds keys. A constant or parser that called `t()` would freeze the language active when it ran. It stores a `TranslationKey`, or a `LocalizedText` (`{ key, params }`) when the message has parameters. The component translates while rendering (`t(key)`, `translateLocalizedText(text, t)`).
- State follows the same rule: it holds an error or a key, and the text is translated while rendering.
- Every enum shown to the user has a label table `<TYPE>_LABEL_KEYS: Record<Enum, TranslationKey>` next to its type, exported through the slice's `index.ts`. Never build a key from a value (``t(`status.${value}`)``), because adding an enum member must fail the build.

### Sentences

- Sentences stay whole. A sentence with inline markup is one key with named tags in the dictionary (`Run <code>pg_dump</code> first`), rendered through `<Trans i18nKey="..." components={{ code: <InlineCodeComponent />, bold: <strong />, docsLink: <a href={...} /> }} />`.
- Use `<TransByKey>` from `shared/i18n` when the key is held in a variable, because `Trans` cannot type-check the whole key union.
- Values are interpolated into a whole sentence (`'Delete {{name}}?'`), never concatenated with translated fragments. When the inserted word would inflect in other languages, each value gets its own sentence key.
- A user-entered value in a `Trans` sentence, such as a workspace name, is a self-closing tag, and the supplied element holds the value: dictionary `Delete <workspaceName/>?`, component `components={{ workspaceName: <strong>{name}</strong> }}`. Never pass it through `values`, because `Trans` would expand placeholder syntax inside it.
- Inline code in prose goes through `InlineCodeComponent` from `shared/ui`. A monospace font alone does not set code apart from the text around it.
- Counts are phrased so one string fits every number: `Members: {{count}}`, not `{{count}} members`. The dictionaries have no plural forms, because Russian's `_few` would be a stray key under `typeof en`.
- Translated text is never rendered as HTML. No `dangerouslySetInnerHTML`.

### Formatting, errors and links

- Locale-dependent output comes from hooks, so a language switch re-renders it: `useTranslation` for text, and `useLocale()` for `formatNumber` (instead of `toLocaleString()`) and `formatRelativeTime` (instead of dayjs `fromNow()`).
- API errors reach the user only through `translateApiError(error, t)`. It picks the text by the error's `code`. Every `code` the contract declares has a translation. Never show `error.message` or the backend's text directly.
- Links to databasus.com pages go through `getWebsitePageUrl(pageId, locale)` with a page from `WEBSITE_PAGES`. The link opens the page in the selected language when the website publishes it.

## Clock format and date order

`shared/time/getUserTimeFormat.ts` reads `navigator.language` on purpose. The 12- or 24-hour clock and the day and month order are regional conventions. The browser reports the user's region, and the language selection only sets the words. A bare `toLocaleString()` is not allowed, because digit grouping follows the selected language.

## Errors and notice lifetimes

- **Persistent errors** stay API errors, translation keys with parameters or supported delivery diagnostics. They are translated during rendering, so switching language updates a visible error.
- **Diagnostic values** render as React text, so angle brackets, braces, hostnames and commands keep their literal characters.
- **Success notices** capture their language when created and expire after their original lifetime. A locale switch leaves a visible notice and its deadline as they are.
- **Editing sessions.** Closing one destroys its own notice, and late results from an obsolete session cannot create one.
- **Forms.** Field errors render beside their controls and derive from the current state. A language or theme change keeps values, newer edits, expanded settings and successful testing of unchanged settings.

## Verification

Run dictionary tests, the affected screen tests and frontend lint. For shared localization behavior, also run the production browser UI command from [testing.md](testing.md#browser-ui-tests). Non-English assertions read the dictionaries, and every other file under `frontend/` stays in English.
