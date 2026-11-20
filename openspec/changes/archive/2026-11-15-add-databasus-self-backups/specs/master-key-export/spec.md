## Purpose

Helps the instance owner retain the existing master key separately and controls its deliberate export without exposing it through ordinary account or settings responses.

## ADDED Requirements

### Requirement: Only root admin can deliberately export the existing master key

Export SHALL require an active root-admin session and a recent identity confirmation bound to that account and this purpose. A password confirmation SHALL be supported; when a password is unavailable, a single-use confirmation code through the configured system mail server SHALL be supported. If neither method is available, the interface SHALL explain how to retain the key directly from the server. Failed, expired or reused proofs SHALL not yield a key. Ordinary instance administrators and workspace members SHALL receive no key or export capability.

#### Scenario: Root admin confirms identity and downloads

- **WHEN** root admin supplies a valid identity confirmation and explicitly downloads the key
- **THEN** the file contains exactly the existing key, without regenerating it or changing its bytes

#### Scenario: Another administrator requests a download

- **WHEN** another administrator directly requests key export
- **THEN** the request is refused with 403 and no key is transferred

#### Scenario: External-provider account has no password or configured mail

- **WHEN** root admin has no local password and cannot receive a confirmation code
- **THEN** the interface provides the host-file saving instructions without exposing the key

### Requirement: HTTP risk is explained before the key is transferred

When the application page or key-download connection uses unencrypted HTTP, the interface SHALL warn before requesting key material that the key would travel unencrypted and could be intercepted. It SHALL show the server-side key location and guidance for retrieving it through a protected host connection. Downloading over HTTP SHALL require an explicit continue action followed by identity confirmation. Opening or dismissing the warning SHALL NOT fetch the key. HTTPS SHALL NOT be described as protection against response-body logging by infrastructure that terminates that connection.

#### Scenario: Owner cancels the HTTP warning

- **WHEN** root admin opens export over HTTP and cancels
- **THEN** the browser receives neither key material nor a usable key-download credential

### Requirement: Exported key material is never an ordinary response field

The key SHALL leave the server only through the explicitly authorized file download. It SHALL NOT appear in ordinary settings, account or status responses, URLs, logs, telemetry, browser persistent storage or cacheable responses. Download authorization SHALL be short-lived, single-use and purpose-bound. The audit log SHALL record the actor and export event without key material or download credentials.

#### Scenario: A download credential is reused

- **WHEN** a caller reuses an already consumed key-download credential
- **THEN** the download is refused and the key is not returned again

### Requirement: Saving the key is an explicit persistent acknowledgment

Root admin SHALL be prompted on their first authenticated visit without a saved-key acknowledgment, independently of workspace or database setup. This SHALL apply to existing installations after the feature is introduced. The owner SHALL be able to download, acknowledge a key saved directly from the host, or defer. A download alone SHALL NOT count as acknowledgment. Acknowledgment SHALL persist for the current key across browsers and sessions; deferral SHALL not reopen the large dialog on each navigation and SHALL leave a smaller reminder until a later sign-in or acknowledgment.

#### Scenario: Owner downloads but has not confirmed saving

- **WHEN** root admin downloads the file but does not acknowledge storing it safely
- **THEN** the key remains marked as requiring acknowledgment

#### Scenario: Owner saved the host file separately

- **WHEN** root admin explicitly confirms saving the current key directly from the host
- **THEN** future visits in another browser do not repeat initial key onboarding for that key
