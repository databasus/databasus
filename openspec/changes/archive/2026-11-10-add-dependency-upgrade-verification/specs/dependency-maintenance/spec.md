## Purpose

Defines a maintainer-led dependency upgrade procedure and bounded automated checks for known pins, coordinated versions and local artifact integrity.

## ADDED Requirements

### Requirement: Maintainers have one upgrade procedure

The repository SHALL provide a guide usable by people and agents covering application and website packages, development tools, test dependencies, workflow actions, container inputs, database clients and verification extensions. It SHALL identify authoritative manifests and official sources, explain release selection and coordinated changes, require a reviewable plan before implementation, and describe functional checks and the final report. The updater SHALL review release notes, compatibility, security and end-of-life status, including newly introduced dependency surfaces. Host prerequisites, first-party images and intentionally historical fixtures SHALL be distinguished from current third-party inputs.

#### Scenario: An agent prepares an update

- **WHEN** a maintainer asks an agent to update dependencies using the guide
- **THEN** it identifies affected inputs, explains selected versions and compatibility effects, and presents a plan before changing dependencies
- **AND** the procedure does not authorize publication or edits to another contributor's work

### Requirement: Release age is reviewed during update selection

The procedure SHALL use a default waiting period of seven complete days for packages, development tools and actions, and fourteen complete days for images, database clients and standalone verification extensions. Related updates SHALL use the longer applicable period. Native age filters and ordinary automated-update cooldowns SHALL be used where supported. The updater SHALL review official publication information for selected releases, including exact pins and security updates, and report missing age information or a departure from the normal period with its reason. The local verification command SHALL NOT claim to prove the publication age of every selected artifact or transitive dependency.

#### Scenario: An exact pin is not covered by a native age filter

- **WHEN** the updater selects an exact version outside the native filter's coverage
- **THEN** it reviews the release age and records a too-fresh or uncertain selection in the update report
- **AND** the local command does not invent publication-history evidence

#### Scenario: An automated security update bypasses an ordinary cooldown

- **WHEN** a security update is proposed without the ordinary waiting period
- **THEN** the updater reviews its urgency, age and compatibility and explains any departure from the normal period
- **AND** existing vulnerability and functional checks still apply

### Requirement: Package installations use native frozen graphs

Application and development-tool installations SHALL consume their authoritative native locks through frozen or locked installation where supported. Missing or stale locks and downloaded-content integrity failures SHALL fail the corresponding native installation or verification step without silently repairing committed inputs. The local verifier SHALL NOT implement a separate dependency resolver, transitive graph interpreter or isolated package-installation service.

#### Scenario: A manifest changes without its lock

- **WHEN** a changed manifest is inconsistent with its native lock
- **THEN** the native frozen installation fails rather than selecting a new graph during CI

#### Scenario: A downloaded package has changed

- **WHEN** package content does not match its native recorded integrity
- **THEN** the native installation or verification step fails without replacing the expected integrity value

### Requirement: Local verification checks known declarations without network access

The repository SHALL expose one offline dependency-check command callable from the root and application subdirectories. Its documented scope SHALL consist of known dependency declarations and recorded local artifacts. It SHALL check exact tool pins and required lock presence, full third-party action commit pins, third-party image digests, related version consistency and local vendored-file hashes and executable permissions. It SHALL reject malformed known records, missing required platform records, checksum mismatches and conflicting declarations with actionable diagnostics and a nonzero exit. It SHALL NOT install tools, query publication services, download artifacts or modify source files, locks or the index.

#### Scenario: Local verification runs from a subdirectory

- **WHEN** the command runs from an application subdirectory without network access and with its normal runtime available
- **THEN** it checks the same known declarations and local artifacts as a root invocation and leaves the working tree and index unchanged

#### Scenario: A browser version disagrees with its image

- **WHEN** a known package, runtime or browser-image declaration conflicts with its related pins
- **THEN** verification fails and identifies the conflicting declarations

#### Scenario: A local vendored client changes

- **WHEN** a recorded executable or private library has different bytes or an executable loses its required permission
- **THEN** verification fails and identifies the affected file without rewriting the expected record

#### Scenario: A new dependency surface is introduced

- **WHEN** an update introduces a dependency outside the command's documented scope
- **THEN** the updater reviews it and extends the focused checks where appropriate
- **AND** a successful run over known declarations is not described as proof of universal dependency discovery

### Requirement: Direct downloads are verified before consumption

Client refreshes and standalone binary downloads SHALL verify content against a recorded expected cryptographic checksum before extraction, installation or execution. The updater SHALL obtain checksum information from official sources, review available vendor signatures through standard tools, and preserve source URLs and release identities. A local hash alone SHALL NOT be presented as publisher authentication. A mismatch SHALL fail without automatically replacing the expected digest. Existing source-distribution equality requirements for vendored clients SHALL remain in force.

#### Scenario: A release archive is replaced

- **WHEN** downloaded client or extension bytes differ from the expected checksum
- **THEN** extraction or installation fails without accepting the replacement

### Requirement: Related client inputs stay coordinated across architectures

Client refreshes SHALL keep sibling tools and supported architectures on the same recorded release/build family and retain executable permissions. Known client records SHALL cover shipped executables and private libraries. A documented historical architecture absence SHALL remain explicit and SHALL NOT excuse a missing record on a supported architecture.

#### Scenario: Only one supported architecture is refreshed

- **WHEN** the recorded client builds disagree between supported architectures
- **THEN** local verification rejects the partial refresh

### Requirement: Verification reports its actual guarantees

The command SHALL identify its checked scope and independently detectable failures, return nonzero for any required local check it cannot complete, and keep credentials out of diagnostics. Success SHALL describe only the configured local checks. The guide SHALL explain that release-age judgment, dependency discovery and upstream review also depend on the updater, and that added OS packages remain subject to distribution repository changes. Functional, compatibility, vulnerability and secret checks SHALL remain separate requirements.

#### Scenario: Local verification passes

- **WHEN** all configured local checks pass
- **THEN** the command reports that result without claiming universal publication-age enforcement, publisher safety, successful application tests or complete image reproducibility
