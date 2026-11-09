## REMOVED Requirements

### Requirement: OAuth tokens never reach the process output

**Reason**: The built-in Google Drive storage client and its token refresh operations are removed. General secret masking and the security requirements of other integrations remain in force.

**Migration**: Remove the retired provider and its refresh-specific diagnostics and tests. Use 3.67.0 to remove Google Drive storages before upgrading, or remain on that version.

## MODIFIED Requirements

### Requirement: A stored secret is used only at the resource address it was saved for

The resource address of a saved storage, notifier or database is the set of its settings that decide where its stored secrets go, as listed below. When a request names a saved resource and a resource address other than the saved one, a stored secret SHALL NOT fill a secret the request leaves out. This SHALL hold for a change, for a test of unsaved settings, and for checking whether a limited user is advisable and creating one.

If the settings in the request use the secret, the request SHALL get `400` with `code` `invalid_request` and a field error for a secret it still needs, nothing SHALL be saved and nothing SHALL leave the backend. If the settings no longer use the secret, a change SHALL remove it from the saved settings, so a later request cannot bring it back at the new address. A secret with no stored value SHALL be treated as a secret the request left out always was: only the rules for a required setting apply. A secret the request supplies SHALL be used as supplied, and a stored secret SHALL keep working while the resource address stays the same. This SHALL hold for every workspace role. This is a security rule: a user who may not see a stored secret must not be able to send it to a server of their choice.

The resource address SHALL be compared exactly as the request and the saved settings hold it, so a difference in letter case, in a trailing slash or in a scheme counts as a change. Only these settings SHALL count, and each secret SHALL count as used only under its condition:

- an S3 storage: the endpoint, for the access key and the secret key, which are always used;
- an Azure Blob storage: the endpoint and the account name, for the account key, which is used with account key authentication; a request that leaves out the account name keeps the stored one, and this is not a change;
- an FTP or NAS storage: the host and the port, for the password, which is always used;
- an SFTP storage: the host and the port, for the password and the private key, each used while its authentication method is selected;
- an email notifier: the SMTP host and port, for the SMTP password, which is used while an SMTP user is set;
- a Telegram notifier: the Bot API URL and the proxy, for the bot token, which is always used; a proxy URL present in the request and turning the proxy on or off each count as a change;
- a Mattermost notifier: the server URL, for the bot token, which is used in the bot mode;
- a webhook notifier: the URL, for the value of each header, which is used while the request lists that header;
- a database: the host, the port and the address of its SSH tunnel, and also the SRV flag for MongoDB, for the password, which is always used, and for the PostgreSQL client key, which is used while the request sends a client certificate;
- the SSH tunnel of a database: whether it is enabled, and its host and port while it is enabled, for the SSH password and the private key, each used while the tunnel is enabled and its authentication type is selected, and for the key passphrase, which is never filled in after a change and is removed with the old address.

A secret that carries its own address, such as a Discord or Teams webhook URL, an Azure connection string, a Telegram proxy URL while the proxy stays on or an rclone configuration, and a secret whose service has a fixed address, such as a Slack bot token, SHALL keep its stored value when the request leaves it out.

#### Scenario: Endpoint changed without the keys

- **WHEN** a member changes the endpoint of a saved S3 storage and leaves out both keys
- **THEN** the response is `400` with `code` `invalid_request` and a field error for a key, and the storage keeps its endpoint and its keys

#### Scenario: Address changed with the secret supplied

- **WHEN** a member changes the host of a saved FTP storage and sends a new password
- **THEN** the change succeeds, and the storage has the new host and the new password

#### Scenario: Test at another address reaches nothing

- **WHEN** a member tests unsaved SMTP settings that name a saved email notifier with an SMTP user and another SMTP host, without the password
- **THEN** the response is `400` with `code` `invalid_request`, and no connection reaches that host

#### Scenario: A limited user at another address

- **WHEN** a member checks whether a limited user is advisable, or creates one, naming a saved database and another host, without the password
- **THEN** the response is `400` with `code` `invalid_request`, and no connection reaches that host

#### Scenario: A workspace owner follows the same rule

- **WHEN** the owner of a workspace changes the Bot API URL of a saved Telegram notifier and leaves out the bot token
- **THEN** the response is `400` with `code` `invalid_request`, and the notifier keeps its Bot API URL

#### Scenario: A difference in letter case counts as a change

- **WHEN** a member changes the host of a saved database from `db.example.com` to `DB.example.com` and leaves out the password
- **THEN** the response is `400` with `code` `invalid_request`

#### Scenario: A trailing slash counts as a change

- **WHEN** a member adds a trailing slash to the URL of a saved webhook notifier and sends its header `Authorization` without a value
- **THEN** the response is `400` with `code` `invalid_request`

#### Scenario: A change outside the resource address keeps the secret

- **WHEN** a member changes the storage class of a saved S3 storage and leaves out both keys
- **THEN** the change succeeds, and the next backup to it authenticates with the stored keys

#### Scenario: A login change at the same address keeps the secret

- **WHEN** a member changes the user name of a saved database at the same host and port and leaves out the password
- **THEN** the backend connects with the new user name and the stored password

#### Scenario: An omitted Azure account name is not a change

- **WHEN** a member changes the container of a saved Azure Blob storage with account key authentication and leaves out the account name and the account key
- **THEN** the change succeeds, and the storage keeps both

#### Scenario: A new proxy needs the token again

- **WHEN** a member sends a proxy URL for a saved Telegram notifier and leaves out the bot token
- **THEN** the response is `400` with `code` `invalid_request`, and no request leaves the backend

#### Scenario: A tunnel moved to another bastion

- **WHEN** a member changes the SSH tunnel host of a saved database and leaves out the database password and the SSH private key
- **THEN** the response is `400` with `code` `invalid_request` and a field error for one of them, and no connection reaches that host

#### Scenario: A database moved behind the same bastion keeps the SSH secrets

- **WHEN** a member changes the host of a saved database behind an unchanged enabled SSH tunnel, sends a new password and leaves out the SSH private key
- **THEN** the backend opens the tunnel with the stored private key and connects to the new host with the new password

#### Scenario: A tunnel turned on again

- **WHEN** a member enables the disabled SSH tunnel of a saved database with the host it held before and leaves out the SSH secrets
- **THEN** the response is `400` with `code` `invalid_request`, and no connection leaves the backend

#### Scenario: A tunnel turned off without the database password

- **WHEN** a member disables the enabled SSH tunnel of a saved database and leaves out the database password
- **THEN** the response is `400` with `code` `invalid_request` and a field error for the password, and the database keeps its tunnel and its secrets

#### Scenario: A tunnel turned off with the database password

- **WHEN** a member disables the enabled SSH tunnel of a saved database, sends the database password and leaves out the SSH secrets
- **THEN** the change succeeds, and the stored SSH secrets are removed without a field error for them

#### Scenario: A new private key drops the old passphrase

- **WHEN** a member moves the SSH tunnel of a saved database, sends a new private key and the database password, and leaves out the passphrase
- **THEN** the change succeeds, and the stored passphrase is removed

#### Scenario: A secret out of use cannot return at the new address

- **WHEN** a member changes the server URL of a saved Mattermost notifier in the webhook mode and leaves out the bot token, then in a later request switches it to the bot mode at the same server URL, again without the bot token
- **THEN** the first request succeeds and removes the stored bot token, and the second gets `400` with `code` `invalid_request`, and the token never reaches that server

#### Scenario: A secret with its own address

- **WHEN** a member changes the name of a saved Azure Blob storage that uses a connection string and leaves out the connection string
- **THEN** the change succeeds, and the storage keeps its connection string
