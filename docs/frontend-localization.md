# Frontend localization

The interface supports six complete dictionaries. The selected language changes words and number grouping immediately without remounting providers, screens or drafts. Region-dependent clock and date conventions still follow the browser region.

`shared/i18n/LocaleProvider.tsx` owns the i18next instance, selected locale, AntD locale and dayjs language. The document language and title follow the same selection. English defines the key structure; dictionary typing and placeholder/tag tests enforce completeness. New copy belongs in all six dictionaries in the same change.

## User-facing copy

In the English dictionary (`shared/i18n/locales/en.ts`), use a plain hyphen `-` in any string the user will see — labels, descriptions, notifications, modal bodies, error messages. Reserve em dashes (`—`) and en dashes (`–`) for markdown docs and code comments only.

The other dictionaries follow their language's own typography instead, as set out in [`website/AGENTS.md` - Translation quality](../website/AGENTS.md#translation-quality): Russian keeps «—» where its grammar wants it, French puts a space before `: ; ! ?` and `%`, Chinese uses full-width punctuation. The same section lists the canonical interface terms per language.

---

## Interface languages (i18n)

The interface ships in English, Russian, Spanish, Portuguese (Brazilian), Chinese (Simplified) and French, through `i18next` and `react-i18next`. Everything lives in `shared/i18n`.

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
- API errors reach the user only through `translateApiError(error, t)`, which picks the text by the error's `code`. Every `code` the contract declares has a translation. Never show `error.message` or the backend's text directly.
- Links to databasus.com pages go through `getWebsitePageUrl(pageId, locale)` with a page from `WEBSITE_PAGES`, which opens the page in the selected language when the website publishes it.

### Clock format and date order

`shared/time/getUserTimeFormat.ts` and `shared/time/utils.ts` read `navigator.language` on purpose. The 12- or 24-hour clock and the day/month order are regional conventions, not properties of a language: English is written with both clocks. The browser reports the user's region, while the language selection only says which words to use. Those reads stay; a bare `toLocaleString()` does not, because digit grouping follows the selected language.

---

## Errors and notice lifetimes

Keep persistent errors as API errors, translation keys with parameters, or supported delivery diagnostics. Translate them during rendering, so switching language updates a visible error. Render diagnostic values as React text; angle brackets, braces, hostnames and commands retain their literal characters.

Context-bound AntD feedback inherits the active theme. Transient success notices capture their language when created and expire after their original bounded lifetime. A locale switch does not recreate a notice or restart its deadline. A later success uses the current language. Closing an editing session destroys its own notice, and late results from an obsolete session cannot create one.

Authentication/profile field errors retain validity flags and render their descriptions beside associated controls. Account API errors remain data until rendering. Profile and settings retain newer edits across language/theme changes and pending saves; async success notices use the language active when completion is accepted.

SMTP field errors are derived from the controlled settings and touched fields. Changing locale updates the label, help and error without clearing values, expanded settings or successful testing of unchanged settings.

## Verification

Run dictionary tests, the affected screen tests and frontend lint. For shared localization behavior, also run the production browser UI command described in [testing.md](testing.md#browser-ui-tests). Obtain non-English assertions from the dictionaries; keep every other file under `frontend/` in English.
