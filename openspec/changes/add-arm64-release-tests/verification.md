# Verification evidence

Local execution uses native Linux amd64. Selection tests exercise ARM inputs without emulation; they do not establish native ARM execution.

## Completed local checks

| Check                                                                                                | Result                                                                                                                         |
| ---------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------ |
| `mise run doctor`                                                                                    | Passed; Codex project hooks enabled and trusted                                                                                |
| `CI=1 mise run backend:test-upgrade` with the current checkout's candidate                           | Reproduced v3.65.0's 404 before repair; repaired complete suite passed (160.765s)                                              |
| `go test -count=1 -run '^TestWorkspacesMigration$' ./migrations`                                     | Passed; preserves historical multiple-owner repair independently of newest-release setup                                       |
| MongoDB SSH test, `-race -count=10`                                                                  | Passed before changes (54.423s), and with failure diagnostics (79.887s); original hosted timeout not reproduced                |
| Four logical e2e packages together, `-race -p=8 -count=1`                                            | Passed: MySQL, MariaDB, MongoDB, PostgreSQL                                                                                    |
| MySQL and MariaDB engines, `-race -count=1`                                                          | Passed                                                                                                                         |
| Test-container package and container-free MySQL version parsing                                      | Passed                                                                                                                         |
| `go test -race -count=1 -run '^TestMongodbBastionFailureDiagnostics$' ./internal/lib/testcontainers` | Passed; expected child failure prints source and bastion diagnostics without fixture passwords                                 |
| Physical PostgreSQL e2e (17 and 18), `-race -count=1`                                                | Passed                                                                                                                         |
| Physical API packages, `-race -count=1`                                                              | Exposed an empty notification inbox after terminal status; other packages passed                                               |
| Physical notification scenarios, `-race -count=20`                                                   | Passed with a completed-run barrier; webhook counts and contents retained                                                      |
| Physical API backuping package and app harness, `-race -count=1`                                     | Passed after repair; WAL readiness retains catalog-only waiting                                                                |
| `pnpm --dir frontend test -- tooling/runBrowser.test.ts`                                             | 8 passed, including both native selections, cache separation, build-checksum failure and unsupported architecture              |
| `pnpm --dir frontend test:browser --project=ui`                                                      | 136 passed with strict existing references; failure probes include missing and changed references                              |
| `mise run e2e:unit`                                                                                  | 29 passed                                                                                                                      |
| `mise run e2e:browser` with the current checkout's candidate                                         | 4 passed; 5 probe-only cases intentionally skipped in the ordinary run; logical and physical Host/Docker recovery data checked |
| `mise run agent:e2e`                                                                                 | Passed (551.104s)                                                                                                              |
| `mise run test`                                                                                      | Passed; frontend 187 files / 2561 tests, backend routine packages passed                                                       |
| Repository script tests                                                                              | 154 passed, including negative matrix/gate/candidate mutations, image exclusions, NFS startup and architecture mismatch cases  |
| Backend, frontend and Python lint; shellcheck and actionlint                                         | Passed at the first implementation snapshot; changed files are checked again before completion                                 |
| Frontend production build                                                                            | Passed                                                                                                                         |
| Native NFS fixture image build                                                                       | Passed on amd64                                                                                                                |
| Trivy NFS Dockerfile configuration scan                                                              | Passed with the documented fixture-root exception                                                                              |
| Semgrep NFS Dockerfile scan                                                                          | Passed after an exact rule suppression for required privileged kernel NFS                                                      |

The finite physical backup helpers wait for the actual backup-run group before callers wait for notification deliveries. The general catalog-readiness helper remains usable while long-lived WAL streaming runs.

## Hosted execution

The user-authorized temporary branch is `feature/arm64-release-tests-verification`. Its first verification snapshot is `4064d324`; manual [run 42119788060](https://github.com/databasus/databasus/actions/runs/42119788060) does not select a release version or publish artifacts. Subsequent local repairs are not part of that snapshot until explicitly recorded by another run.

The first run completed. Both architectures passed upgrade, agent Docker/Podman e2e, frontend unit tests, browser UI against the same strict screenshot references, all three physical engine shards, and physical API shards one and three. ARM application/candidate/verification image builds and script tests also passed. Filesystem lifecycle and upgrade groups passed on both architectures.

Observed failures and follow-up:

- ARM `other`: unavailable FTP/SFTP images. Native pinned Debian fixtures replace them; native execution of the replacements remains pending.
- ARM logical: MariaDB write-protection, MySQL inspection and MongoDB SSH restore operation deadlines under concurrent load. Existing eight-CPU logical profiles and an explicit parallel-scenario budget are selected for the follow-up. Product deadlines are unchanged; source/bastion failure diagnostics and resource snapshots gather further evidence. The original timeout cause is not yet established.
- ARM physical API shard two: WAL restart exceeded the test's thirty-second wall-clock inference. The revised regression directly requires shutdown release while heartbeat expiry is impossible, then checks the resumed session and same single slot. Execution remains pending.
- Both browser e2e legs: the Cron selection clicked an already-closing dropdown. A single accessible selection replaces the repeated interaction; native recheck remains pending.
- Both filesystem network legs: prerequisites pulled the local NFS tag before build. The repaired prerequisite behavior passed its command-level regression; actual mounts remain pending.
- ARM filesystem permissions: all three candidate download attempts timed out before any scenario ran. This is pending transport verification.
- Semgrep: required root NFS entry point. The exact documented suppression passed locally.

The local browser lifecycle run finished with 20 passed, 5 build-only skips and 2 failures. Both failures were nested-registry pull timeouts before their intended scenarios. Recovery now loads the host's pinned image archive and exact content ID into the private engine. Unit coverage passes both successful loading without registry access and failed loading without readiness; complete lifecycle/interruption rechecks remain pending.

After resuming on November 6, `mise run doctor` passed again. Backend lint and compilation, Python lint, shellcheck, actionlint, 155 script tests and 31 browser-suite unit tests passed after the fixture/recovery edits. Trivy passed both new FTP/SFTP Dockerfiles. The local Docker daemon is unavailable because containerd panics on corrupt Bolt metadata; shared Docker data has not been modified. Further container checks therefore require hosted execution or restoration of the shared daemon. The latest WAL test edit still awaits its lint and execution results.
