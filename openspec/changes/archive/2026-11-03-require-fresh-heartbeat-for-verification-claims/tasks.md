## 1. Reproduce assignment to ineligible agents

- [x] 1.1 Add isolated HTTP scenarios in the verification module for a fitting pending verification claimed by an agent with no accepted heartbeat and by an agent more than five minutes past its accepted heartbeat. Use the injected clock and raw claim client. Require empty `204`, pending status, absent owner, unchanged attempt count and unchanged last-seen/capacity. Run the new `TestVerificationClaimHeartbeat` scenarios against the unfixed code with `mise exec -- go test -C backend -count=1 -run '^TestVerificationClaimHeartbeat$' ./internal/modules/verification` and record that they fail because the server assigns work.

## 2. Define and implement claim admission

- [x] 2.1 Document accepted-heartbeat eligibility and the additional empty `204` condition in `backend/api/agent.yaml`, preserving request shapes, error codes and validation/authentication precedence. Run `mise run generate-api` and verify the contract and generated artifacts with `mise run backend:lint`.
- [x] 2.2 Add the freshness decision in the verification claim service after existing capacity validation and before assignment queries or pending-row locks, using the injected clock and existing five-minute threshold. Reuse a private predicate with stale-run recovery only if needed to keep the existing boundary consistent. Add no persisted flag or bypass. Verify that task 1.1 passes with `mise exec -- go test -C backend -count=1 -run '^TestVerificationClaimHeartbeat$' ./internal/modules/verification`.

## 3. Admission, recovery and unaffected behavior

- [x] 3.1 Update existing successful-claim setups to send explicit valid heartbeats where required, including protocol clients outside the shared helper. Preserve owned verification IDs in repeated setup heartbeats and leave negative/admission clients raw. Do not auto-heartbeat from agent creation. Complete this setup repair before the broader checks in tasks 3.3 and 3.4. Verify module coverage with `mise exec -- go test -C backend -count=1 ./internal/modules/verification`; verify real-agent startup polling and any companion upgrade scenario through the full-suite command below.
- [x] 3.2 Extend `TestVerificationClaimHeartbeat` for fresh accepted contact, exactly five minutes, just beyond five minutes, repeated denied claims, recovery from absent and expired contact using the same credentials, and a healthy second agent claiming the withheld job. Verify every case with `mise exec -- go test -C backend -count=1 -run '^TestVerificationClaimHeartbeat$' ./internal/modules/verification`.
- [x] 3.3 Cover gigabyte-only rejected contact before any accepted heartbeat, rejected contact after a success without extending its original expiration, and rejected contact within the still-valid window without early revocation. Assert the stored capacity and timestamp remain unchanged. Preserve malformed/negative claim validation and authentication/rate-limit outcomes for ineligible agents. Verify through the protocol HTTP clients with `mise exec -- go test -C backend -count=1 -run 'TestVerification(ClaimHeartbeat|HeartbeatMemory|AgentAuthentication|AgentCapacityValidation|Protocol)' ./internal/modules/verification`.
- [x] 3.4 Add regression assertions that never-seen and expired agents remain listed and counted, users can enqueue while no agent is eligible, and an expired owner can still stream/report an owned running verification before the reaper runs. Verify with `mise exec -- go test -C backend -count=1 ./internal/modules/verification`.

## 4. Documentation and integration checks

- [x] 4.1 Add Assignment eligibility to the Product table in `docs/glossary.md`, defining it separately from registration and resource capacity and relating it to the existing `lastSeenAt` and `claimVerification` contract names. Describe new-assignment admission and the exact five-minute boundary in `docs/architecture.md`, distinguishing it from registered-agent counts and existing running-job recovery. Preserve the companion change's compatibility clarification when present. Verify terminology consistency with `rg -n -i 'assignment eligibility|lastSeenAt|claimVerification' docs/glossary.md docs/architecture.md backend/api/agent.yaml`, run `mise run backend:format` and `mise run backend:lint`, then record the full-suite results required below before handing over for mandatory review.

## Change verification

```sh
mise run test
```
