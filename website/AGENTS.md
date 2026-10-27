# Website — Agent Rules

Next.js 16 App Router, static export (`output: "export"`, `trailingSlash: true`), deployed to GitHub Pages. The site is translated into 6 locales (`ru`, `es`, `pt`, `zh`, `fr`, `id`) by full per-language page copies, not dictionaries.

## Structure

- `app/(en)/` — the English site (root layout with `lang="en"`). URLs are unchanged by the route group. `not-found.tsx` and `error.tsx` live here; the `[lang]` tree deliberately has none (GitHub Pages serves the single `out/404.html`).
- `app/[lang]/` — translated pages. Root layout renders `<html lang>` per locale, `generateStaticParams()` returns the 6 locales, `dynamicParams = false`.
- `app/components/SiteDocument.tsx` — the shared document skeleton (head preloads, analytics script, body). Both root layouts are thin wrappers over it; never duplicate head markup in a layout.
- `app/components/LandingHeaderComponent.tsx` and `LandingFooterComponent.tsx` — the navbar and footer of the landing pages (home, `mysql-backup`, `mongodb-backup`) in every locale. Their labels live in `landingLabels.ts`; each page passes only `lang` and its own in-page anchor links. Never copy the navbar or footer markup back into a page.
- `app/i18n.ts` — single source of truth: `LOCALES`, hreflang codes, OG locales, `TRANSLATED_PATHS`, `getLocalizedUrl()` (absolute, for metadata), `getLocalizedHref()` (relative, for in-page links), `getLanguageAlternates()`, `splitLocaleFromPathname()`. Never hand-build a localized URL, absolute or relative.

## Translated page convention

Every translated page follows one shape, no exceptions:

```
app/[lang]/<path>/page.tsx        — thin dispatcher
app/[lang]/<path>/content/ru.tsx  — full page copy per locale
app/[lang]/<path>/content/es.tsx
...
```

Each `content/<locale>.tsx` exports **both** `metadata` (fully translated, `canonical` via `getLocalizedUrl`, `languages` via `getLanguageAlternates`, `openGraph.locale` from `OG_LOCALES`) and the default page component. The dispatcher imports all six and picks by `params.lang` in `generateMetadata` and in render. A translator (human or agent) owns exactly one `content/<locale>.tsx` per page and touches nothing shared.

`generateStaticParams` lives only in `app/[lang]/layout.tsx` — do not add it to pages.

## Sync rule (critical)

**If you change a page in `app/(en)/` that is listed in `TRANSLATED_PATHS`, you must apply the same change to all 6 copies in `app/[lang]/<path>/content/`.** That includes structure, links, and metadata. Check before finishing:

1. The section/element hierarchy matches the English page.
2. Internal links carry the `/{lang}/` prefix — except links to untranslated pages (`/privacy`, `/terms-of-use`, `/contribute/*`), which stay unprefixed.
3. Heading `id` anchors are identical to English in every locale (cross-links like `/faq#oss-programs` depend on them).
4. Technical strings (install commands, configs, file names, code blocks) are byte-identical to English.
5. Adding a page to the translated core or adding a locale starts in `app/i18n.ts` (`TRANSLATED_PATHS` / `LOCALES`); sitemap and hreflang follow from it automatically. The application's links to website pages depend on it too: `frontend/src/shared/i18n/websitePages.ts` marks which linked pages are translated, and a frontend test (`websitePages.test.ts`) reads `TRANSLATED_PATHS` and fails when the two disagree. Adding or removing a translated path that the application links to means updating that table in the same change.

## Translation rules

- Titles and `<h1>` target the highest-volume search phrasing of the locale, not a literal translation. Canonical head-forms: ru «резервное копирование PostgreSQL», es «copia de seguridad de PostgreSQL», pt «backup PostgreSQL», fr «sauvegarde PostgreSQL», zh «PostgreSQL 备份», id «backup PostgreSQL». Check what query the English page targets (title/keywords) and pick the frequent equivalent.
- Synonyms go in body text, canonical query forms in h1/h2 (ru «бекап», «дамп»; es «respaldo»; pt «cópia de segurança»; fr «backup»; zh «数据库备份»). Don't force synonyms into subheadings.
- Portuguese is Brazilian (pt-BR): usuário, equipe, seção, arquivo, gerund progressives («está rodando», never «está a correr»), proclisis («se conecta»). Chinese is Simplified.
- `keywords` get a plain translation, nothing more (search engines ignore them).
- URL form everywhere (canonical, hreflang, sitemap) is **with** trailing slash — Next normalizes rendered metadata URLs to that form under `trailingSlash: true`, and internal consistency of the cluster is what matters. `getLocalizedUrl` produces it; never hand-build URLs.

## Translation quality

These rules also govern the application's interface dictionaries (`frontend/src/shared/i18n/locales/<locale>.ts`). There the English plain-hyphen rule from `frontend/AGENTS.md` does not apply to the other languages; each follows its own typography below.

Translations are edited copy in the target language, not word-for-word renderings. The Russian pages were proofread by the owner — treat their current text as the tone reference before writing or editing any locale.

- **No calques.** If a sentence keeps English word order or literally renders an English construction, rewrite it («Результат отправляется обратно — включая X» → «Результат приходит вам: X»; "user experience with X" → just describe working with X). Read the sentence aloud in the target language; if it sounds translated, it is.
- **No bureaucratese or pompous filler.** Cut inflated verbs («обеспечивает возможность» → «умеет», «дает») and closing self-praise sentences ("This adoption level reflects strong community trust…"). End on the last concrete fact.
- **No repeats.** Don't restate a list or repeat the same verb in adjacent sentences ("supports X, Y, Z… X, Y, Z are also supported").
- **Everyday tech anglicisms are fine in body text** when they are what practitioners actually say (ru: бекап, дамп, стриминг, продакшен, комплаенс, «из коробки», «соло-разработчик») — don't swap them for stiff formal equivalents. The formal canonical query form still owns `title`/`h1`/`h2` (see above). Exception by owner's choice: «рабочие пространства», not «воркспейсы».
- **Follow locale conventions.** Russian: «е», never «ё»; numerals like «2-х минут», «3-х попыток», «17-й версии»; «—» stays where Russian grammar wants it. Spanish: usted; canonical «copia de seguridad» in headings, «respaldo» as a body synonym; no bare English «backup» in prose. French: vous; a space before % («99 %»); a plain ASCII space (not `&nbsp;`) before `: ; ! ?`. Chinese: 你, never 您 (titles included); full-width punctuation in prose（，。：）; a space at every CJK↔Latin boundary. Indonesian: «kamu» only when a pronoun is useful, and sparingly; natural sentences, no slang or literal English syntax; `1.800.000` and `1.234,56` (period for thousands, comma for decimals). Number formatting per locale: en `1,800,000`, fr `1 800 000`, es/pt `1.800.000`, zh `180 万` (万-format for millions in prose).
- **Interface terms.** The application's dictionaries use one word per concept, so the interface stays consistent and the docs can quote it. Russian: backup «бекап», storage «хранилище», notifier «канал уведомлений», workspace «рабочее пространство», restore «восстановление», health check «проверка доступности», restore verification «проверка восстановления», verification agent «агент проверки». Spanish: «copia de seguridad», «almacenamiento», «canal de notificación», «espacio de trabajo», «restauración», «comprobación de disponibilidad», «verificación de restauración», «agente de verificación». Portuguese: «backup», «armazenamento», «canal de notificação», «espaço de trabalho», «restauração», «verificação de disponibilidade», «verificação de restauração», «agente de verificação». French: «sauvegarde», «stockage», «canal de notification», «espace de travail», «restauration», «vérification de disponibilité», «vérification de restauration», «agent de vérification». Chinese: 备份, 存储, 通知渠道, 工作区, 恢复, 可用性检查, 恢复验证, 验证代理. Indonesian: backup «backup», database «database», restore «restore», storage «penyimpanan», notifier «kanal notifikasi», workspace «ruang kerja», health check «pemeriksaan ketersediaan», restore verification «verifikasi restore», verification agent «agen verifikasi». The interface shows counts and sizes as full numerals in every language (zh `1,800,000`); the 万 form is for prose only.
- **Product stats stay in sync.** Docker pulls, GitHub stars and similar figures must match across all 7 languages _and_ every duplicated block on a page (visible FAQ + its JSON-LD copy). Changing a number means changing it everywhere; competitors' figures are separate and only change when their reality does.
- **Some labels stay English everywhere** by design: the hero badges ("GitHub stars" / "Docker pulls"), brand-name nav items (Slack, Google Drive, "Databasus vs X"), and product UI option names quoted in docs ("After backup", "Scheduled verification", "Hourly") — the interface is English, so the docs quote it verbatim.

## Language exception

The repo-wide "English only" rule has two carve-outs here: rendered content and metadata of `content/<locale>.tsx` files, and the target-language label values in the shared locale maps of five components: `landingLabels.ts` (`LANDING_LABELS`, `LANGUAGE_SELECTOR_LABELS`, `TABLE_OF_CONTENTS_LABELS`), `copyLabels.ts` (`COPY_LABELS`, `VIDEO_PLAY_LABELS`), `DocsNavbarComponent.tsx` (`NAVBAR_LABELS`), `DocsSidebarComponent.tsx` (`SIDEBAR_TITLES`, `NAVIGATION_HEADINGS`, `NAVIGATION_ACTIONS`) and `InstallationComponent.tsx` (`INSTALLATION_TEXTS`). Those values include accessible names. The selector and video components consume these maps or receive labels through props; they hold no target-language literal of their own. Identifiers, comments, file names (locale codes aside), commit messages, and everything else stay English.

## Verification

- `npm run build` — static export must succeed; `out/index.html` and `out/404.html` must exist.
- `npm run lint` (`next lint` was removed in Next 16).
- Completeness: for every `TRANSLATED_PATHS` × locale pair, `out/{lang}/{path}/index.html` exists with the right `<html lang=`, hreflang links, and a translated `<title>`.
