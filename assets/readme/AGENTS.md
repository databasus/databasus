# README translations — Agent Rules

Translated copies of the root `README.md`, one file per locale:
`README.ru.md`, `README.es.md`, `README.pt.md`, `README.zh.md`, `README.fr.md`, `README.id.md`.
English stays at the repo root; only the translations live here.

## Sync rule (critical)

**Any change to `/README.md` must be applied to all 6 copies here** — structure,
headings, links, code blocks, badges. A translation that lags behind the English
file is worse than no translation: readers act on install commands.

The `## 🛡️ Security & reliability engineering` section is the sharpest case. The
root `AGENTS.md` requires it to stay consistent with the project's actual
security practices, so a change there now lands in 7 files.

## Structure

Each file mirrors the English README exactly: same sections in the same order,
same heading levels, same emoji in headings, same bullet structure, same
horizontal rules, same HTML blocks. The `### 📦 Installation` teaser inside
Features duplicates the real `## 📦 Installation` in English too — keep the
duplication.

## Byte-identical to English

Code blocks (commands, YAML, comments inside them), shields.io badge URLs and
their alt texts, inline code, image `alt`/`width` attributes, product and tool
names, version numbers, ports, and English UI labels quoted in the text
(`"New Database"` in the Usage steps — the interface is English, so the docs
quote it verbatim).

## Paths

These files sit one directory deeper than the English README:

- images: `../logo.svg`, `../dashboard-dark.svg`, `../dashboard.svg`, `../healthchecks.svg`
- repo files: `../../LICENSE`, `../../deploy/helm/README.md`
- the English README: `../../README.md`

## Website links

Links to `databasus.com` carry the locale prefix and a trailing slash:
`https://databasus.com/<locale>/storages/`, `https://databasus.com/<locale>/faq/#backup-databasus`.
The fragment ids (such as `#backup-databasus`) are identical in every
locale — never translate them. `https://databasus.com/contribute` has no
translation on the site and stays unprefixed.

## In-page anchors

Unlike the website, anchors here are **not** identical across locales: GitHub
derives them from the heading text, so translated headings produce translated
anchors. Nothing outside these files links into their sections, so each file's
mini-TOC and in-body links simply point at its own slugs. Rule of thumb for the
slug: lowercase, punctuation dropped, emoji dropped but the space around it
becomes a hyphen, remaining spaces become hyphens, Cyrillic and CJK preserved.
`## ✨ Features` gives `#-features`; `## 🛡️ Security & reliability engineering`
gives `#️-security--reliability-engineering` (the variation selector survives).

## Language

Translation content is in the target language; the seven language-switcher labels
(`English`, `Русский`, `Español`, `Português`, `中文`, `Français`, `Bahasa Indonesia`) appear in the
root `README.md` as well. Everything else in this directory stays English,
including commit messages and this document; file names carry only the locale
code. The root `AGENTS.md` holds the full list of places translated content is
allowed.

## Translation quality

The rules in [`website/AGENTS.md`](../../website/AGENTS.md) under "Translation
quality" apply here in full: edited copy rather than word-for-word, no calques,
no bureaucratese, no marketing filler, per-locale conventions (ru «е» not «ё»;
es usted with «copia de seguridad» as the canonical term; pt-BR; zh 你 with
full-width punctuation and CJK↔Latin spacing; fr vous with a space before %).
The proofread website pages of each locale are the tone reference.
