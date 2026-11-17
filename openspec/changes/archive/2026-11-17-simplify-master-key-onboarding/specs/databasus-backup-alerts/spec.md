## MODIFIED Requirements

### Requirement: Removing the last destination immediately reports stopped protection

A committed configuration change from at least one selected destination to none SHALL immediately create a stopped-protection incident. It SHALL bypass initial onboarding eligibility and the normal overdue grace period. A failed or rolled-back configuration save SHALL create no such incident. An initially unconfigured installation SHALL not send stopped-protection email. Re-adding a destination SHALL request an immediate backup and keep a resuming incident until at least one new copy succeeds; failures of remaining selected destinations SHALL continue to be reported independently.

#### Scenario: Final selected destination is removed

- **WHEN** root admin successfully removes the last selected storage
- **THEN** stopped protection becomes visible immediately and an email is eligible for delivery when mail prerequisites exist

#### Scenario: Final removal cannot be saved

- **WHEN** saving the removal of the last storage fails
- **THEN** the previous configuration remains active and no stopped-protection notification is created
