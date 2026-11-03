## 1. Reproduce the header regression

- [x] 1.1 Extend `frontend/src/app/ui/ApplicationScreen.test.tsx` with shared typed HTTP setup and scenarios named `hides the disk summary at 50 percent`, `shows the disk summary in the muted color at 90 percent` and `shows the disk summary in the danger color at 96 percent`. Add exact 85/95 percent boundaries, 86 percent and 85.01/95.01 percent rounding cases, using light/dark renderer options. Preserve capacity/percentage coverage, put the environment directive first and remove the redundant inner theme provider. Complete the disk response and flush its updates before asserting absence. Run `pnpm --dir frontend test src/app/ui/ApplicationScreen.test.tsx` against the unfixed code and confirm failures expose the low-usage visibility and missing danger color.

## 2. Restore warning behavior and remove the unused translation

- [x] 2.1 Update the header auxiliary content in `ApplicationScreen.tsx` to require usage strictly above 85 percent and use `text-danger` strictly above 95 percent, otherwise `text-muted`. Compare raw byte ratios, retaining the current formatter, summary and tooltip. Verify that every scenario passes with `pnpm --dir frontend test src/app/ui/ApplicationScreen.test.tsx`.
- [x] 2.2 Remove `app.diskUsage.compactSummary` from all six locale dictionaries, without a replacement key or alias. Verify dictionary completeness and placeholders with `pnpm --dir frontend test src/shared/i18n/dictionaries.test.ts` and validate callers with `pnpm --dir frontend typecheck`.

## 3. Verify rendered colors and responsive behavior

- [x] 3.1 Add automated scenarios in `frontend/browser/scenarios/appShell.spec.ts` using explicit disk-response fixtures: check computed muted/danger colors at 90/96 percent against active theme roles, theme changes while the warning is visible, header reflow at 769 pixels with 200 percent text, and mobile information at 50/95/96 percent. Retain finite method/path fixtures without duplicates. Verify with `pnpm --dir frontend test:browser --project=ui`.
- [x] 3.2 Repair the delivery-notice race exposed by the enlarged-text French/Chinese SMTP browser scenarios: wait for the delivery-success notice, close it and verify its absence before clicking Save. Retain existing reflow and request-count assertions. Verify with `pnpm --dir frontend test:browser --project=ui`.
- [x] 3.3 Format only the changed frontend files with `pnpm --dir frontend format` followed by their paths relative to `frontend/`, then run `mise run frontend:lint` and `pnpm --dir frontend build`. Run the commands in Change verification before finishing for mandatory review, resolve blocking findings, and update task completion and verification evidence in this change. Leave the working tree ready to commit.

## Change verification

```sh
mise run test
pnpm --dir frontend test:browser --project=ui
```
