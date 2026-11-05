## 1. README

- [x] 1.1 Reconcile the Indonesian README with the current English source, preserving technical bytes and relative links; verify the delivered diff with `git diff --check` and the repository documentation checker.

## 2. Website

- [x] 2.1 Update Indonesian FAQ, installation, database landing pages and security guidance to match current English content, and remove the orphaned agent-installation copy; verify with `pnpm --dir website lint`.
- [x] 2.2 Verify locale registration, translated routes and metadata remain complete after reconciliation with `pnpm --dir website build` and an automated inspection of the static export.

## 3. Application

- [x] 3.1 Reconcile Indonesian dictionary keys, substitutions and locale integration with current English sources; verify dictionary and localization tests using the existing frontend test runner.
- [x] 3.2 Update localization documentation to include Indonesian and verify `mise run frontend:lint` and `pnpm --dir frontend build`.

## 4. Integrated verification

- [x] 4.1 Run relevant existing localization screen tests and verify the three contribution areas contain no unrelated behavior changes; record exact commands and results before independent review.

## Change verification

```sh
mise run test
```

On macOS, the aggregate command stops before running suites because the system
Bash does not support its dynamic file-descriptor lock syntax. The frontend suite
was run directly instead: 187 files and 2,516 tests passed. Backend suites were not
run. Focused localization checks passed (102 tests), followed by 30 dictionary
tests after correcting saved webhook-header guidance. Frontend lint, typecheck and
build passed; website lint and build passed, exporting 184 pages. Static inspection
verified language, title, canonical and hreflang metadata on 144 translated pages.

Browser checks covered Indonesian sign-in rendering, language persistence after
reload and mobile overflow with a sign-in-options HTTP fixture. Website desktop
and mobile captures verified navigation and the one-line copy label. Full signed-in
browser flows remain unverified because the Fedora development script cannot start
on this host (`ldconfig` is unavailable). Native review-client doctor was waived by
the user; an independent read-only review found and prompted correction of the
webhook-header guidance. It did not replace the native review hooks.
