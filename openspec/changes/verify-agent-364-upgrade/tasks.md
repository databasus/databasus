## 1. Released binary and upgrade scenario

- [x] 1.1 Add a fixed 3.64.0 agent fixture to upgrade-test preparation: resolve and pin the published release image digest, extract the host-architecture binary into test-owned storage, and check its version independently of the newest-release server baseline. Wire it into the existing upgrade command without committing the binary. Verify with `mise run backend:test-upgrade`, including an assertion that selecting a different server baseline does not change the agent fixture version; check changed shell files with `mise run lint-shell`.
- [x] 1.2 Extend the real-server upgrade flow after its preservation assertions with an isolated agent process and forwarding gateway that holds discovery and claims. Run the actual 3.64.0 binary with only the startup update check skipped, observe a real `400` heartbeat, release discovery, and require replacement, restart and an accepted current heartbeat with the same identity and resource settings. Add the scenario `updates a 3.64.0 agent and accepts its heartbeat after the old heartbeat is rejected`. Verify with `mise run backend:test-upgrade`; retain evidence of any production defect before fixing it and rerun that scenario after the fix.
- [x] 1.3 Release claims only after the replacement heartbeat, complete verification of the known encrypted backup and assert its table and row statistics, including exclusion of the row added after backup. Preserve the existing pending-record migration assertion before agents start. Extend owned-resource cleanup and failure diagnostics for agents, gateways and restore containers without logging credentials. Capture fixture-container volume mounts and assert their removal on both normal and failure cleanup. Verify the completion and cleanup assertions with `mise run backend:test-upgrade`.

## 2. Failed replacement and manual recovery

- [x] 2.1 Add the released-agent scenario with an unprivileged process, read-only executable directory, writable working files and usable engine socket. Keep claims held; observe rejected heartbeat and failed background replacement, and require the installed executable to remain 3.64.0. Verify this is an update failure rather than engine, config, log or lock startup failure with `mise run backend:test-upgrade`.
- [x] 2.2 Extend that scenario to stop the agent, download and atomically install the current binary with installation permissions, and restart under the agent identity using its existing configuration. Require the saved configuration on restart without resupplying credentials or resource flags. Temporarily remove that configuration, require the restart helper to reject missing settings, and run the actual replacement binary without flags supplying credentials or capacity. Assert a nonzero exit and a missing-configuration diagnostic, then restore the configuration and require accepted contact and unchanged settings, release claims, and complete a known-backup verification with correct table and row statistics. Verify with `mise run backend:test-upgrade`; do not assert assignment churn before recovery, because the companion change removes it.

## 3. Contract and recovery guidance

- [x] 3.1 Correct the previous-release compatibility description in `backend/api/agent.yaml` without restoring gigabyte-only heartbeats. Run `mise run generate-api`, qualify the shared-contract claim in `docs/architecture.md` with successful self-update, and describe the fixed released-agent coverage in `docs/testing.md`. Verify regeneration and contract consistency with `mise run backend:lint`.
- [x] 3.2 Document update permissions, failure symptoms and manual recovery in `docs/architecture.md`. Cover built-in daemon and service/container supervision, the correct architecture, temporary download and replacement, saved settings, accepted server contact and a completed verification. Use the installed binary's absolute path for stop, replacement and restart while preserving the working directory. Keep website pages and translations unchanged. Verify the recovery sequence with the automated scenario from task 2.2 through `mise run backend:test-upgrade` and check the final diff with `git diff --check`.

## 4. Integration checks

- [x] 4.1 Run `mise run backend:format`, `mise run backend:lint`, `mise run lint-shell` on the final implementation. Search the affected contract and documentation, including contract-rule comments, for contradictory update claims, verify the two changes retain separate ownership, and record the full-suite results required below before handing over for mandatory review.

- [x] 4.2 Repair the WAL shutdown setup race exposed by the required full run: wait for PostgreSQL to report an active replication slot before capturing its identity in both shutdown scenarios. Keep the same-slot, slot-count and thirty-second restart assertions. Verify with `mise exec -- go test -C backend -count=1 -timeout 30m -run '^TestWalStreamShutdown$' ./internal/modules/backups/physicalapitests/walstreaming`; task 4.1 owns the subsequent full-suite check.

## Change verification

```sh
mise run test
```
