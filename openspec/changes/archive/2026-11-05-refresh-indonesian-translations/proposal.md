## Why

The Indonesian translation contribution predates substantial upstream changes. Its application dictionary and documentation must match the current product before maintainers can review the README, website and application independently.

## What Changes

- Add Indonesian to the application's supported languages, with complete current copy and matching component, date and number locales.
- Bring Indonesian website pages and the README into agreement with current English sources, including removal of documentation that upstream no longer publishes.
- Preserve existing language selection, accessibility and technical instructions.
- Keep README, website and application changes independently reviewable. Publish the website before or together with application links to Indonesian pages.

## Capabilities

### New Capabilities

- `indonesian-documentation`: Indonesian README and website content follows the published English documentation and offers working language navigation.

### Modified Capabilities

- `frontend-localization`: Include Indonesian among supported interface languages and native language names.

## Impact

The change follows root `AGENTS.md`, `frontend/AGENTS.md`, `website/AGENTS.md` and `assets/readme/AGENTS.md`. It touches translation dictionaries, locale registration, localized documentation and their existing checks. No backend API, storage schema or dependency changes are required.

Out of scope: backend behavior, visual redesign, new documentation features, translation changes to other languages, review-client configuration, deployment, and publishing or closing pull requests without user authorization.
