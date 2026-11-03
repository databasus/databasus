## ADDED Requirements

### Requirement: New assignments require a fresh accepted heartbeat

For an authenticated, non-deleted agent with an otherwise valid claim request, the server SHALL assign new verification work only if it has previously accepted a heartbeat from that agent and that heartbeat is no more than five minutes old. A heartbeat exactly five minutes old SHALL remain eligible; one older than five minutes SHALL be expired. An agent without an accepted heartbeat or with an expired heartbeat SHALL receive an empty `204` response, even when pending work fits the capacity in its claim.

Withholding an assignment SHALL leave pending verification status, ownership and attempt count unchanged and SHALL NOT refresh the agent's last-seen time or capacity. Another eligible agent SHALL be able to claim that pending work. An agent SHALL regain eligibility when the server accepts a new heartbeat, using the same registration and current token.

This rule SHALL govern new assignments only. An agent SHALL remain listed and included in the registered-agent count while it exists, regardless of assignment eligibility. A user SHALL still be able to enqueue a verification while no agent is eligible. Existing ownership, cancellation and lifecycle rules SHALL continue to govern streams and reports for previously assigned work.

#### Scenario: A registered agent has never sent an accepted heartbeat

- **WHEN** an authenticated agent with no accepted heartbeat claims work and a pending verification fits its claimed capacity
- **THEN** the response is an empty `204`
- **AND** the verification stays pending with no assigned agent and the same attempt count
- **AND** the agent's last-seen time and stored capacity do not change

#### Scenario: A fresh accepted heartbeat allows an assignment

- **WHEN** an agent claims fitting pending work less than five minutes after an accepted heartbeat
- **THEN** it receives the assignment through the normal claim response

#### Scenario: The accepted heartbeat is exactly five minutes old

- **WHEN** an agent claims fitting pending work exactly five minutes after its last accepted heartbeat
- **THEN** it remains eligible and receives the assignment

#### Scenario: The accepted heartbeat has expired

- **WHEN** an agent claims fitting pending work more than five minutes after its last accepted heartbeat
- **THEN** the response is an empty `204`
- **AND** the verification stays pending with no assigned agent and the same attempt count
- **AND** repeated claims do not refresh the accepted heartbeat or consume attempts

#### Scenario: A healthy agent takes work withheld from an expired agent

- **WHEN** an expired agent receives no assignment and another agent with a fresh accepted heartbeat claims the same fitting pending verification
- **THEN** the healthy agent receives the assignment with no attempt consumed by the refused claim

#### Scenario: A recovered agent sends a valid heartbeat

- **WHEN** an agent without an accepted heartbeat or with an expired heartbeat sends a valid heartbeat and then claims fitting pending work
- **THEN** the heartbeat is accepted and the claim receives an assignment without new registration or token rotation

### Requirement: Rejected heartbeats do not establish or extend assignment eligibility

A rejected heartbeat SHALL NOT establish or refresh the last accepted heartbeat used for assignment eligibility. Its rejection SHALL preserve the previously accepted timestamp and stored capacity. If there is no previously accepted heartbeat, the agent SHALL remain ineligible. If one exists, its original five-minute window SHALL continue to determine eligibility: a rejected request SHALL neither extend that window nor revoke it early.

#### Scenario: Only an incompatible heartbeat has been sent

- **WHEN** the server rejects an agent's heartbeat that supplies memory only in gigabytes and the agent has no previously accepted heartbeat
- **THEN** a subsequent otherwise valid claim receives an empty `204` and fitting pending work remains unassigned

#### Scenario: Rejected heartbeats do not renew an earlier accepted heartbeat

- **WHEN** an agent sends an accepted heartbeat, later sends rejected heartbeats, and claims fitting pending work more than five minutes after the accepted heartbeat
- **THEN** the response is an empty `204`, even if a rejected heartbeat was sent recently
- **AND** the stored capacity and last-seen timestamp remain those of the accepted heartbeat

#### Scenario: A rejected request does not revoke a fresh accepted heartbeat

- **WHEN** an agent sends an accepted heartbeat, later sends a rejected heartbeat and claims fitting pending work within the original five-minute window
- **THEN** the agent remains eligible based on the accepted heartbeat
- **AND** the rejected heartbeat changes neither stored capacity nor the accepted timestamp
