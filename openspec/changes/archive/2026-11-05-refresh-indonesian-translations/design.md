## Context

See `proposal.md` for motivation. The application already centralizes localization in `shared/i18n` (`docs/frontend-localization.md:5`), checks dictionary shape and interpolation (`docs/frontend-localization.md:16`), and routes documentation links through a locale-aware helper (`docs/frontend-localization.md:51`). The website already registers Indonesian in its locale and metadata maps (`website/app/i18n.ts:3`). Its translated route list excludes the removed agent-installation page (`website/app/i18n.ts:45`).

The approach follows `frontend/AGENTS.md` localization and architecture rules, `website/AGENTS.md` full-page translation rules, and `assets/readme/AGENTS.md:19` source-mirroring rules. It preserves the existing architecture described in `docs/architecture.md`; no new localization layer is needed.

## Goals / Non-Goals

**Goals:** Reuse existing locale integration, enforce dictionary completeness, and keep the three contribution areas independently reviewable.

**Non-Goals:** New translation infrastructure, changes to regional clock conventions, backend changes, or redesigning translated pages.

## Decisions

1. Update the Indonesian object literal against the current English dictionary. Existing type and interpolation checks cover missing keys and malformed substitutions (`docs/frontend-localization.md:16-19`). Reject spreading English into Indonesian: it would conceal untranslated copy and weaken excess-key checks.
2. Keep website translations as full page copies and use the existing route helpers (`website/app/i18n.ts:84-109`). Reconcile English source changes in the Indonesian copies and delete the orphaned agent page. Reject retaining old documentation as a compatibility page because upstream removed the route and its navigation.
3. Match README structure and technical bytes while retaining localized prose and adjusted relative links (`assets/readme/AGENTS.md:19-45`). Reject replacing the translation with an external link because the requested contribution includes an Indonesian README.
4. Verify each module with its pinned pnpm toolchain (`frontend/package.json:55`, `website/package.json:26`). Keep README, website and application file groups separate; website availability precedes application links. Reject one combined PR because the maintainer requested separate review scopes.

## Risks / Trade-offs

- Source drift after a large merge: compare current English and Indonesian content, then run existing dictionary, lint and build checks.
- A successful static build can still contain a wrong translation or broken section link: inspect representative Indonesian pages at desktop and mobile sizes and retain screenshots.
- Application documentation links can precede website publication: document website-first ordering in the handoff.
- Native review hooks reject OpenCode as a review client: the user explicitly authorized proceeding after the doctor failure. Leave hook configuration unchanged and obtain an independent review with the available agent capability.

## Migration Plan

No database or API migration is required. The website and README can be reviewed independently. Make the Indonesian website available before enabling application links to it. Reverting a contribution removes only that contribution's locale integration and content.
