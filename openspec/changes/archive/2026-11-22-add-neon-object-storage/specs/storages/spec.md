## ADDED Requirements

### Requirement: S3 objects stay within 5 GiB

Every object Databasus writes to an S3 storage SHALL be no larger than 5 GiB (5 × 1024³ bytes), whatever the size of the saved file and whichever S3-compatible provider holds the bucket. A file larger than that SHALL be stored as several objects described by a manifest, as large files already are. Files saved before this rule, whose objects may be larger, SHALL still read back intact and delete completely.

#### Scenario: A large backup is saved

- **WHEN** a backup of about 12 GiB is saved to an S3 storage
- **THEN** it is stored as several objects, none larger than 5 GiB, and reads back byte for byte

#### Scenario: A file saved with larger objects is read

- **WHEN** a member restores a backup that an earlier version stored as one object of about 15 GiB
- **THEN** the restore reads it back intact, and deleting the backup removes all its objects

### Requirement: Neon Object Storage is offered as an S3 storage choice

The storage type list of the storage form SHALL offer Neon Object Storage next to the storage types. Choosing it SHALL create an ordinary S3 storage: the saved storage's type SHALL be S3 and its settings SHALL be S3 settings, with path-style addressing and no storage class unless the member sets them. The Neon choice SHALL show the S3 fields with Neon examples, an endpoint of the form `https://br-<branch>.storage.<cell>.<region>.aws.neon.tech` and the region `us-east-2`, and a link to the Neon Object Storage guide in the interface language. Switching a new storage between S3 and Neon Object Storage SHALL keep the values already entered.

#### Scenario: Creating a Neon Object Storage storage

- **WHEN** a member chooses Neon Object Storage, fills the bucket, region, keys and a Neon endpoint, and saves
- **THEN** the saved storage has type S3 with exactly those settings, path-style addressing and no storage class

#### Scenario: Neon examples are shown

- **WHEN** a member chooses Neon Object Storage for a new storage
- **THEN** the empty endpoint and region fields show the Neon endpoint form and `us-east-2` as examples

#### Scenario: Switching between S3 and Neon

- **WHEN** a member enters a bucket under S3 and then chooses Neon Object Storage
- **THEN** the bucket stays filled

#### Scenario: The guide is linked

- **WHEN** a member views the Neon Object Storage form in Russian
- **THEN** the form links to the Russian page of the Neon Object Storage guide

### Requirement: Neon credentials can be pasted from the clipboard

The Neon Object Storage form SHALL offer to read the credentials block that Neon Console gives for object storage from the clipboard, above an "or enter the details manually" divider and the S3 fields. The block holds lines of the form `NAME="value"`, with or without quotes and with comment lines. Reading it SHALL fill the endpoint from `AWS_ENDPOINT_URL_S3`, the region from `AWS_REGION`, the access key from `AWS_ACCESS_KEY_ID` and the secret key from `AWS_SECRET_ACCESS_KEY`, and SHALL leave the bucket and the other fields as they were. A successful read SHALL confirm that the credentials were filled. When the block holds no endpoint at Neon, the form SHALL report that the text is not a Neon credentials block and change nothing. When the block lacks either key or holds a placeholder instead of it, the form SHALL fill what the block holds, leave that key's field as it was and warn that a credential has to be generated in Neon Console. When the browser does not offer the clipboard API, the form SHALL offer a field to paste the text into instead; when the API exists but reading is refused, the form SHALL report that the clipboard could not be read.

#### Scenario: A complete block is pasted

- **WHEN** a member pastes a block with a Neon endpoint, a region and both keys
- **THEN** the endpoint, region, access key and secret key fields hold the block's values and the bucket field is unchanged

#### Scenario: The block has no generated keys

- **WHEN** a member pastes a block that holds the endpoint and the region but placeholders instead of the keys
- **THEN** the endpoint and region are filled, both key fields are unchanged and the form warns that a credential has to be generated

#### Scenario: Unrelated text is pasted

- **WHEN** a member pastes text that holds no Neon endpoint
- **THEN** the form reports that the text is not a Neon credentials block and every field is unchanged

#### Scenario: The clipboard API is unavailable

- **WHEN** the browser does not offer the clipboard API and the member pastes the block into the offered field
- **THEN** the fields are filled as if the clipboard had been read

#### Scenario: Reading the clipboard is refused

- **WHEN** the browser offers the clipboard API but refuses to read it
- **THEN** the form reports that the clipboard could not be read and every field is unchanged

### Requirement: Pasted Neon credentials stay in the form

Reading the Neon credentials block SHALL happen in the browser and SHALL send no request; the credentials SHALL leave the browser only when the member tests or saves the storage, as any typed credential does. The secret key SHALL stay masked in its field after the paste, and no notification or error message SHALL show either key's value.

#### Scenario: A block is pasted

- **WHEN** a member pastes a complete Neon credentials block
- **THEN** no request is sent, the secret key field is masked, and the confirmation names no key value

### Requirement: A saved Neon Object Storage storage opens with the Neon form

When a member opens a saved S3 storage whose endpoint host is `neon.tech` or ends with `.neon.tech`, the form SHALL show the Neon credentials paste and the Neon guide link, as for a new Neon Object Storage storage. The storage's type SHALL still show as S3. A saved S3 storage with any other endpoint, or without one, SHALL open with the plain S3 form.

#### Scenario: Opening a Neon storage

- **WHEN** a member opens a saved S3 storage whose endpoint is `https://br-example.storage.c-9.us-east-2.aws.neon.tech`
- **THEN** the form shows the Neon credentials paste and the Neon guide link

#### Scenario: Opening another S3 storage

- **WHEN** a member opens a saved S3 storage whose endpoint is at Cloudflare R2
- **THEN** the form shows the plain S3 fields without the Neon paste

### Requirement: Public materials describe Neon Object Storage setup

The website SHALL publish a Neon Object Storage guide at `/storages/neon-object-storage` in every supported language. The guide SHALL walk through creating a bucket and a credential in Neon Console and adding the storage in Databasus, with screenshots, and SHALL answer in short direct answers: whether Neon Object Storage is S3-compatible, which endpoint and region to use, that it uses path-style addressing, how its object size limits relate to Databasus backups, that the service is in beta, and that backups belong in a different Neon project than the databases they protect, because deleting a project removes both. Its title, description and headings SHALL target the search phrasing for backing up to Neon Object Storage in each language, and the page SHALL carry step-by-step and question-and-answer structured data that matches its visible text. The storages page, the documentation navigation, the Neon database guide, the sitemap, the language alternates and the machine-readable page listing SHALL link to it. The README in every supported language SHALL name Neon Object Storage as plain text, without a link, at the end of its list of cloud storage destinations, after Cloudflare R2.

#### Scenario: Reading the guide in any language

- **WHEN** a reader opens `/storages/neon-object-storage` or its translated page in any supported language
- **THEN** the page shows the setup steps with screenshots and the answers above in that language

#### Scenario: A machine reads the guide

- **WHEN** a search engine or answer engine reads the page
- **THEN** it finds a translated title and description, language alternates for every locale, and structured data whose steps and answers match the visible text

#### Scenario: Finding the guide

- **WHEN** a reader opens the storages page, the documentation navigation, the Neon database guide, the sitemap or the machine-readable page listing in any supported language
- **THEN** each links to the Neon Object Storage guide

#### Scenario: Reading the README

- **WHEN** a reader opens the README in any supported language
- **THEN** its cloud storage list ends with Cloudflare R2 and Neon Object Storage, named without a link
