## Verification evidence

- Before the fix, `pnpm --dir frontend test src/app/ui/ApplicationScreen.test.tsx` ran 16 scenarios: 8 passed and 8 failed. The failures showed visible summaries at 50/85 percent and missing danger color at 95.01/96 percent in both themes.
- After the fix, the same command passed all 16 scenarios, including exact and rounded threshold boundaries.
- `pnpm --dir frontend test src/shared/i18n/dictionaries.test.ts` passed all 20 dictionary checks (run together with the screen tests).
- `pnpm --dir frontend test` passed all 2378 tests in 183 files.
- `mise run frontend:lint` passed formatting, source lint, architecture, semantic styles, types and generated API freshness checks.
- `pnpm --dir frontend build` passed. Vite reported its existing runtime-config script and large-chunk warnings.
- `pnpm --dir frontend test:browser --project=ui` passed all 122 scenarios after the SMTP synchronization repair, including the eight added disk scenarios and both enlarged-text SMTP cases.
- `mise run test` passed all selected backend packages, all 2378 frontend tests and the upgrade test from `v3.64.0` (upgrade package: 86.114 seconds). The run acquired the machine-wide full-test lock after waiting for another run.

## Review execution

`mise run doctor --client codex` reported that project hooks were not loaded in this session. Native quick hooks were invoked manually with the Codex client. Architecture accepted the change; code-shape accepted it after the requested blank-line separation was applied; diff initially reported pending verification. All required verification passed. `mise run review restore-disk-usage-warning --client codex` completed successfully: diff, architecture and code-shape quick checks and the full implementation review accepted the change with no remaining findings.

## Related test repair

The browser suite exposed a delivery-success notice racing the enlarged-text French/Chinese SMTP Save click. The scenarios now wait for the visible success notice, dismiss it through its normal Close control and assert its absence before retaining the existing Save, reflow and request-count checks. Production notifier behavior and test timeouts are unchanged.
