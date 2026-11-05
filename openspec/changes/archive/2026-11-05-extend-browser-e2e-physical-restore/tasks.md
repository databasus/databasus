## 1. Correct the displayed recovery and startup commands

- [x] 1.1 Add focused regressions for restore paths containing spaces, relative and absolute Docker mount sources, a custom Host bin directory and a digest-pinned Docker image. Cover field changes through the real restore form and both preparation and startup instructions. Run `pnpm --dir frontend test src/pages/databases/lib/physical-backups/restoreCommands.test.ts src/pages/databases/ui/physical-backups/PhysicalRestoreComponent.test.tsx` and observe the relevant failures before repairing the builders.
- [x] 1.2 Repair the existing command builders and their callers so the form's selected path, tools and image are used consistently and quoted as shell arguments. Preserve the existing version-specific layouts. If instruction wording must change, update all six dictionaries together. Verify with the same focused frontend command, `mise run frontend:lint` and `pnpm --dir frontend build`.

## 2. Prepare recovery resources owned by one browser run

- [x] 2.1 Configure the pinned PostgreSQL 18 source for physical replication and WAL summarization. Add the idle recovery runner, digest-pinned private Docker daemon, private socket and run-owned data volumes to the existing Compose project. Prepare curl, zstd, PostgreSQL tools and the Docker CLI without building another image or mounting the host engine socket. Add a public lifecycle probe that proves runner readiness, common application-address reachability, matching bind-mount paths and a usable private engine. Verify with `mise run e2e:unit` and `mise run e2e:lifecycle` before building the physical backup scenario.
- [x] 2.2 Add execution of unchanged page commands through standard input, under the required user and working directory, with run-owned regular log files and bounded process lifetimes. Extend masking for physical restore-token query values and encoded diagnostics, and collect Host PostgreSQL and nested Docker logs before teardown. Verify with `mise run e2e:unit` and lifecycle execution/masking probes through `mise run e2e:lifecycle`.

## 3. Extend the existing critical path

- [x] 3.1 Continue after the logical restore on the existing instance, account, workspace and storage. Create an independent physical marker table, add the physical connection and backup settings through the UI, and complete one full and one incremental backup. Wait for source WAL summaries instead of sleeping. Through read-only public operations, require the completed incremental's `rootFullBackupId` to equal the completed full backup's ID and `parentIncrementalBackupId` to be absent. Prevent unrelated scheduled full backups during the scenario. Verify with `mise run e2e:browser`.
- [x] 3.2 Add the two post-incremental write stages, record different WAL filenames and commit positions, rotate both segments and wait for archived coverage through the final write. Require the UI to show the completed WAL range without assuming one row per segment. Verify with `mise run e2e:browser` and the independent expected-row checks.
- [x] 3.3 Restore through Script/Host into `./host restore` with the explicit PostgreSQL bin directory outside PATH. Read and execute the active tab's recovery, ownership and startup commands unchanged; check the downloaded chain and WAL, wait for recovery completion and compare all four write stages. Verify with `mise run e2e:browser`.
- [x] 3.4 Close and reopen the dialog for a new token, then restore through Script/Docker into `/restore-work/docker restore` with the source's complete image reference. Execute both displayed commands unchanged, verify the running image and recovered rows, and recheck the source. Verify with `mise run e2e:browser`.

- [x] 3.5 Repair the explicit physical backup menu to send the public `FULL` and `INCREMENTAL` kinds. Reproduce both lowercase requests in screen tests before the fix, then run the affected screen tests, frontend lint/build and the browser critical path.

## 4. Prove failure handling and preserve the CI contract

- [x] 4.1 Extend lifecycle probes with nested recovery resources for failure, interruption, crashed-run sweeping and overlapping runs. Add an invalid Host tools-path case entered through the UI that executes the resulting command and fails with the intended diagnostic. Prove that all run-owned recovery resources disappear, another live run and the sentinel survive, and logs mask restore tokens. Verify with `mise run e2e:unit`, `mise run e2e:lifecycle` and `uv run --project e2e/browser pytest e2e/browser/lifecycle_interruption`.
- [x] 4.2 Keep the full physical path under the existing `test-e2e-browser` job and preserve explicit candidate-image selection and current workflow triggers. Update `docs/testing.md` with the exact coverage, recovery tools, privileged fixture requirement and cleanup behavior; search other descriptions of this suite and update affected statements only. Measure the local public run against the current job budget and adjust the timeout only for demonstrated legitimate work. Verify the existing named-image lifecycle probe with `mise run e2e:lifecycle`, the full path with `mise run e2e:browser`, and workflow changes, if any, with `mise run lint-workflows`.
- [x] 4.3 Preserve completed and partial masked log streams independently when recovery log collection times out. Reproduce the loss through the public command with an unresponsive run-owned private engine, then verify preserved outer logs, partial recovery output, masked secrets and complete cleanup with the same lifecycle regression and `mise run e2e:lifecycle`.

## Change verification

```sh
mise run test
mise run e2e:browser
```


## Verification results

- `mise run test` passed. After repairing the explicit backup menu, its 33 screen tests, frontend lint and the production build passed as well.
- The focused recovery command and form tests passed (42 tests); shellcheck passed for both fixture preparation scripts.
- `mise run e2e:unit` passed (28 tests).
- The latest `mise run e2e:lifecycle` finished with 26 passes and one setup failure in 989.63 seconds, including successful fresh-checkout and overlapping full critical paths. The failed case did not reach its intended scenario because the nested image pull timed out waiting for Docker Hub's authentication endpoint. Its isolated repeat passed in 58.49 seconds. An earlier setup failure's cause was not established. Setup failures remain failures; the suite adds no retries.
- The unresponsive-private-engine regression reproduced the loss of completed and partial logs before the fix and passed afterward, both alone and in the full lifecycle run. It verifies masked outer logs, partial recovery output, the timeout diagnostic, complete removal and an unaffected unrelated container.
- `uv run --project e2e/browser pytest e2e/browser/lifecycle_interruption` passed (3 tests).
- The final `mise run e2e:browser` passed (4 scenarios; 5 lifecycle-only probes skipped), including both physical recoveries. Total supervisor time was 88.0 seconds, including setup and cleanup; scenario time was 65.76 seconds. The existing CI job timeout and triggers are unchanged.
- [GitHub/Namespace validation](https://github.com/databasus/databasus/actions/runs/42056073549) passed using the existing candidate-build and browser jobs. The `namespace-profile-4vcpu-8ram-ubuntu-26-x64` test runner passed 28 unit tests, all 4 browser scenarios and 22 lifecycle cases; the 5 image-building lifecycle cases were skipped as configured for CI. The public browser run took 68.4 seconds including setup and cleanup. The temporary branch, workflow file and candidate artifact were removed; the workflow was disabled and its run history retained.
