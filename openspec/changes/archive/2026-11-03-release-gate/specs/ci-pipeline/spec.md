## ADDED Requirements

### Requirement: Release requires the security scans

CI SHALL select a version, publish an image or create a release for a commit on `main` only after secret scanning, rule-based security scanning and static analysis (CodeQL) have passed for that commit. A scan that fails, is cancelled or times out SHALL stop the release. A running scan of a commit on `main` SHALL NOT be cancelled because a newer commit arrived or a scheduled scan started.

#### Scenario: Secret scan finds a secret on main

- **WHEN** every test and image check passes on a push to `main` but the secret scan reports a finding that is not allowlisted
- **THEN** version selection does not run, and CI publishes no image and creates no release for that commit

#### Scenario: Rule-based scan fails on main

- **WHEN** every other check passes on a push to `main` but the rule-based security scan fails
- **THEN** version selection does not run, and CI publishes no image and creates no release for that commit

#### Scenario: Static analysis fails on main

- **WHEN** every other check passes on a push to `main` but the static analysis fails or times out
- **THEN** version selection does not run, and CI publishes no image and creates no release for that commit

#### Scenario: Static analysis is cancelled on main

- **WHEN** a maintainer cancels the static analysis of a commit on `main`
- **THEN** version selection does not run, and CI publishes no image and creates no release for that commit

#### Scenario: A newer commit arrives during the scans

- **WHEN** a commit is pushed to `main` while the scans of the previous commit run
- **THEN** the scans of the previous commit run to completion

#### Scenario: The scheduled scan starts during a release scan

- **WHEN** the weekly scheduled scan starts while the release scan of a commit on `main` runs
- **THEN** the release scan runs to completion and its result decides the release

#### Scenario: A queued commit is replaced by a newer one

- **WHEN** a run for a commit on `main` is still queued behind an active run and a newer commit arrives
- **THEN** the queued run does not release, and the run of the newer commit passes the same checks before its own release, which includes both commits

### Requirement: CodeQL alerts gate a release by severity

The release analysis is the static analysis that runs as part of the release gate for a commit on `main`. The severity gate SHALL stop the release of that commit when the release analysis reports an open critical CodeQL alert on `main`. Open high CodeQL alerts that the release analysis reports SHALL appear as a warning and SHALL NOT stop the release. A dismissed CodeQL alert SHALL NOT count. A CodeQL alert that the release analysis no longer reports SHALL NOT count, even while a scheduled analysis still shows it as open. If the severity gate cannot read the CodeQL alerts, or finds no release analysis of the released commit, the release SHALL stop. The severity gate SHALL be repeatable for the same commit, so a dismissal takes effect without a new commit.

#### Scenario: An open critical CodeQL alert exists

- **WHEN** the static analysis of a commit on `main` completes and reports an open critical CodeQL alert
- **THEN** version selection does not run and the severity gate output lists the alert

#### Scenario: Only high CodeQL alerts are open

- **WHEN** the static analysis completes and high CodeQL alerts are open but no critical one is
- **THEN** the severity gate passes and reports the high alerts as a warning

#### Scenario: The owner dismisses the only critical CodeQL alert

- **WHEN** the owner dismisses the only open critical CodeQL alert and reruns the failed severity gate for the same commit
- **THEN** the severity gate passes and the release continues

#### Scenario: A commit fixes a critical CodeQL alert

- **WHEN** a commit on `main` fixes a critical CodeQL alert that the last scheduled analysis still shows as open
- **THEN** the severity gate passes without waiting for the next scheduled analysis

#### Scenario: The release analysis of the commit is not found

- **WHEN** the severity gate finds no release analysis of the released commit
- **THEN** the severity gate fails and the release stops, whatever the CodeQL alerts show

#### Scenario: CodeQL alerts cannot be read

- **WHEN** the severity gate cannot read the CodeQL alerts or the service returns an error
- **THEN** the severity gate fails and the release stops

### Requirement: The secret scan that gates a release reads the released history

The secret scan that gates a release SHALL read only commits reachable from the released commit. The scheduled secret scan SHALL read every branch and SHALL NOT affect a release.

#### Scenario: A finding exists only on another branch

- **WHEN** a branch not merged into `main` holds a secret scan finding and a commit on `main` is otherwise clean
- **THEN** the secret scan that gates the release of that commit passes

#### Scenario: The scheduled scan reads every branch

- **WHEN** the scheduled secret scan runs and a branch not merged into `main` holds a finding
- **THEN** the scheduled scan fails and reports the finding

### Requirement: The application image is checked on every published architecture

Each run of the CI and Release workflow, except a run on `develop` or a pull request from `develop`, SHALL build the application image on amd64 once the lint jobs and the Dockerfile scan have passed. Each such run on the `main` branch SHALL also build it on arm64, and no other run SHALL. The same smoke checks and the same image vulnerability scan SHALL run on each built architecture before any image is published. A failure on one architecture SHALL stop the release even if the other passes. A critical vulnerability with an available fix SHALL fail the image check. A high vulnerability SHALL NOT fail it.

#### Scenario: The arm64 image fails its smoke checks

- **WHEN** the amd64 image passes its image check but the arm64 image fails its smoke checks
- **THEN** on `main` CI selects no version and publishes no image

#### Scenario: A pull request is checked on amd64 only

- **WHEN** CI runs for a pull request from a branch other than `develop`
- **THEN** the amd64 image check runs and no arm64 image is built

#### Scenario: A manual run on main checks arm64

- **WHEN** a maintainer starts the workflow by hand on `main`
- **THEN** the arm64 image check runs before version selection

#### Scenario: A bundled client tool is broken on arm64

- **WHEN** the arm64 image starts but reports that a bundled client tool cannot run
- **THEN** the arm64 image check fails

#### Scenario: The arm64 image has a fixable critical vulnerability

- **WHEN** the image vulnerability scan of the arm64 image finds a critical vulnerability with an available fix
- **THEN** the arm64 image check fails and CI publishes no image

#### Scenario: The image has only high vulnerabilities

- **WHEN** the image vulnerability scan finds high vulnerabilities and no critical one with an available fix
- **THEN** the image check passes

### Requirement: Version selection waits for the pre-release image checks

CI SHALL select a version only after the application image checks and the verification image build have passed. A failure in either SHALL leave every image without a tag for that version. This requirement SHALL NOT cover a failure in the image rebuild after version selection, which can still leave some images tagged.

#### Scenario: The application image check fails on main

- **WHEN** every test and scan passes on a push to `main` and the verification image builds, but the application image check fails
- **THEN** version selection does not run and no image gets a tag for a new version

#### Scenario: The verification image build fails on main

- **WHEN** every test, scan and application image check passes on a push to `main` but the verification image build fails
- **THEN** version selection does not run and no image gets a tag for a new version

### Requirement: Build inputs are pinned

Every base image of the published application image SHALL be referenced by an immutable digest of its multi-architecture manifest. The image vulnerability scanner and the version selection tool SHALL be installed at exact versions, and the scanner download SHALL be verified against its published checksum. CI and a developer's machine SHALL take the scanner version from one pin in the repository.

#### Scenario: A base image tag moves

- **WHEN** a publisher moves a tag of a base image to new content
- **THEN** a build of the same commit uses the same base image content as before

#### Scenario: The scanner publishes a new release

- **WHEN** a new scanner version appears, including a compromised one
- **THEN** CI keeps using the pinned version until a commit changes it

#### Scenario: The scanner download does not match its checksum

- **WHEN** the downloaded scanner does not match its published checksum
- **THEN** the job that installs it fails and no scan runs with that download

#### Scenario: CI and a developer use the same scanner

- **WHEN** CI and a developer's machine install the repository's tools
- **THEN** both resolve the same scanner version from the same pin

#### Scenario: The version selection tool publishes a new release

- **WHEN** a new version of the version selection tool appears
- **THEN** version selection keeps using the exact version the workflow names

### Requirement: Workflow write permissions are granted per job

A workflow SHALL default to read-only repository contents at its top level. A write permission SHALL be granted only to the job that needs it.

#### Scenario: The dependency review comments on a pull request

- **WHEN** the dependency review finds a problem in a pull request from a branch of this repository
- **THEN** its job can post the comment, and the top-level permissions of the workflow stay read-only

### Requirement: Workflow files are linted

The repository's lint command SHALL lint the workflow files. A CI check SHALL do the same on each run of the CI and Release workflow except a pull request from `develop`, so pushes to `develop` are covered. A failure of the check on `main` SHALL stop version selection.

#### Scenario: A workflow names a job that does not exist

- **WHEN** a workflow file has a `needs` reference to a job that does not exist
- **THEN** the repository's lint command and the CI check fail

#### Scenario: A commit is pushed to develop

- **WHEN** a commit is pushed to `develop`
- **THEN** the CI check lints the workflow files of that commit

#### Scenario: The check fails on main

- **WHEN** every other check passes on a push to `main` but the workflow lint fails
- **THEN** version selection does not run, and CI publishes no image and creates no release for that commit
