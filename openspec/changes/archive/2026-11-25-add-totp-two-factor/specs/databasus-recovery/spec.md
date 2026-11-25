## MODIFIED Requirements

### Requirement: UI replacement requires current root authorization and explicit confirmation

Upload, preparation status, cancellation and apply SHALL require the target installation's current root admin; other administrators and workspace members SHALL be refused. The owner SHALL explicitly confirm replacement after seeing the snapshot date, source version and warning that current users, settings and backup catalog will be replaced. While the instance requires two-factor sign-in, by either method, applying SHALL also require a single-use confirmation by root admin's second factor; while it does not, the session and the replacement confirmation SHALL be enough. When root admin has a connected authenticator app, the confirmation SHALL be a current app code or an unused recovery code, checked under the rules of the `authenticator-app` spec; no mail SHALL be sent, and requesting a confirmation SHALL tell the client that the app code is expected. Otherwise the confirmation SHALL be a single-use code emailed to root admin and bound to the current root account/session, that recovery operation and the immutable archive content. The account password SHALL NOT be accepted as a confirmation. A code SHALL be emailed only while the instance requires two-factor sign-in, root admin has no connected app, a system mail server is configured and root admin's address is usable; while the instance does not require two-factor sign-in, or the mail server or the address is missing for a root admin without an app, a code request SHALL be refused with 409 `identity_proof_unavailable`. Possession of the source key alone SHALL NOT authorize replacement. While a code is required, a missing, invalid, expired, reused or wrong-purpose code SHALL NOT apply an import. If the emailed code cannot be sent, the interface SHALL show the sending error; the owner can still recover from mounted files into a separate fresh target, and the interface SHALL NOT imply that mounted files can replace the current account-bearing target.

#### Scenario: Another administrator imports an archive

- **WHEN** an authenticated administrator other than root admin attempts upload, preparation inspection or application
- **THEN** the request is refused with 403 and no recovery data or capability is exposed

#### Scenario: Owner cancels replacement confirmation

- **WHEN** root admin cancels confirmation of a prepared import
- **THEN** the active installation and its key remain unchanged and no publication begins

#### Scenario: An export proof is submitted for recovery

- **WHEN** a caller supplies a master-key export proof or a proof for another prepared import
- **THEN** recovery apply is refused without replacing the active installation

#### Scenario: Two-factor sign-in is off

- **WHEN** the instance does not require two-factor sign-in and root admin confirms replacement of a prepared import
- **THEN** apply is accepted without a code or a password
- **AND** a request for an emailed code is refused with 409 `identity_proof_unavailable` without sending mail

#### Scenario: Two-factor sign-in is on

- **WHEN** the instance requires two-factor sign-in and root admin confirms replacement without a code or with a wrong one
- **THEN** apply is refused with 400 `identity_proof_invalid` and the prepared recovery stays ready

#### Scenario: Two-factor sign-in is on and root admin has an app

- **WHEN** the instance requires two-factor sign-in, root admin has a connected app, and root admin confirms replacement with a current app code
- **THEN** apply is accepted without sending mail, even when the instance has no mail server

#### Scenario: Requesting a confirmation when root admin has an app

- **WHEN** the instance requires two-factor sign-in, with or without a mail server, root admin has a connected app, and root admin requests a confirmation for a prepared recovery
- **THEN** the answer names the app method and carries no expiry time, no mail is sent, and the request is not refused with 409 `identity_proof_unavailable`

#### Scenario: Confirming replacement with a recovery code

- **WHEN** the instance requires the authenticator app method and has no mail server, and root admin with a connected app confirms replacement with an unused recovery code
- **THEN** apply is accepted and that recovery code stops working

#### Scenario: An app code already spent

- **WHEN** root admin confirms replacement with an app code that was already accepted in the same 30-second period, for example for a master-key export
- **THEN** apply is refused with 400 `identity_proof_invalid` and the prepared recovery stays ready

