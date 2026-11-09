# secret-handling Specification

## Purpose
Defines how the backend treats the secrets it stores and uses: tokens stay out of the process output, the master encryption key exists once, and a secret that cannot be decrypted makes its operation fail visibly.

## Requirements

### Requirement: The master encryption key is generated once

On a fresh installation, the backend SHALL generate the master encryption key once and SHALL use that one key for every secret it encrypts. The backend SHALL load or create the key while it starts, before it serves requests or runs background work. This SHALL hold when several backend processes start at the same time before the key exists. A key that an older release left in the internal database SHALL be written into the key file before the key is loaded, replacing whatever the file held, and SHALL then be removed from the database.

#### Scenario: Concurrent first use

- **WHEN** several backend processes start at the same moment on an installation that has no master key yet
- **THEN** one key file exists afterwards
- **AND** every secret encrypted by those processes decrypts with that key

#### Scenario: Key already exists

- **WHEN** the backend runs with an existing master key file and no master key left in its database
- **THEN** it uses the key from that file and does not rewrite the file

#### Scenario: Key left in the database by an older release

- **WHEN** the backend starts on an internal database that still holds a master key
- **THEN** the key file contains that key
- **AND** the database no longer holds it
- **AND** secrets encrypted with that key before the upgrade decrypt

### Requirement: The backend refuses to start without a usable master key

When a legacy master key in the internal database cannot be written into the key file, or when, after any legacy key has been moved into the file, the key file cannot be read or created or holds fewer than 32 bytes, the backend SHALL stop during startup with an error that names the key file. It SHALL NOT serve requests or run background work in that state. Without a legacy key it SHALL NOT replace an existing key file that it cannot use, and a legacy key that could not be written SHALL stay in the database.

#### Scenario: Key file shorter than 32 bytes

- **WHEN** the backend starts and the master key file holds fewer than 32 bytes
- **THEN** the backend stops with an error that names the key file
- **AND** the key file is unchanged

#### Scenario: Key file cannot be read

- **WHEN** the backend starts and the master key file exists but cannot be read
- **THEN** the backend stops with an error that names the key file
- **AND** no new key file replaces it

#### Scenario: Key file cannot be created

- **WHEN** the backend starts without a master key file, and the directory of the key file does not let the backend create a file
- **THEN** the backend stops with an error that names the key file
- **AND** it serves no request and starts no background work

#### Scenario: Legacy key cannot be written into the key file

- **WHEN** the backend starts on an internal database that still holds a master key, and the key file cannot be written
- **THEN** the backend stops with an error that names the key file
- **AND** it serves no request
- **AND** the database still holds the key

### Requirement: A notifier secret that cannot be decrypted fails the notification visibly

When a webhook notifier sends a notification and its URL, its body template or one of its header values cannot be decrypted, the notification SHALL fail with an error that says decryption failed. The backend SHALL NOT send a request built from the encrypted value. The notifier SHALL still appear in the notifier list, so the user can edit it.

#### Scenario: URL cannot be decrypted

- **WHEN** a webhook notifier's stored URL cannot be decrypted with the current master key, and a notification is sent through it
- **THEN** the notification fails with an error that says the URL could not be decrypted
- **AND** no HTTP request is made

#### Scenario: Body template cannot be decrypted

- **WHEN** a webhook notifier's stored body template cannot be decrypted with the current master key, and a notification is sent through it
- **THEN** the notification fails with an error that says the body template could not be decrypted
- **AND** no HTTP request is made

#### Scenario: Header value cannot be decrypted

- **WHEN** one header value of a webhook notifier cannot be decrypted, and a notification is sent through it
- **THEN** the notification fails with an error that says a header could not be decrypted
- **AND** no HTTP request is made

#### Scenario: Notifier list with an undecryptable notifier

- **WHEN** a workspace has a webhook notifier whose secrets cannot be decrypted, and a user opens the notifier list
- **THEN** the list loads and includes that notifier

### Requirement: The verification agent keeps its token file private

The verification agent stores its token in its configuration file on the user's machine. The agent SHALL create that file readable and writable only by its owner, with mode `0600`, and SHALL keep that mode when it rewrites the file. At every start, before it reads the token, the agent SHALL set mode `0600` on an existing configuration file that has any permission bit for the group or for others, so that a file an earlier release created with mode `0644` is closed on the first start after the upgrade. A file with no group or other permission bit, such as one with mode `0400`, SHALL be left as it is.

When the agent cannot narrow the mode, for example because the file belongs to another user, it SHALL keep starting and verifying as before. It SHALL show a warning on the console of the command that starts it and SHALL write the same warning to its log. The warning SHALL be marked as a warning and SHALL name:

- the path of the configuration file;
- its current mode and its owner;
- the error the operating system returned;
- that other users of the machine can read the agent token;
- the command that sets mode `0600` on the file and, when the file belongs to another user, the command that gives it to the user the agent runs as.

The first line of the console warning SHALL start with `WARNING:`, and the log record SHALL carry a message that starts with `warning:`, because the agent's console and log lines show no level. The warning SHALL NOT contain the token. The agent SHALL show it at every start while the file still has a group or other permission bit.

#### Scenario: First save creates a private file

- **WHEN** the agent saves its configuration and no configuration file exists yet
- **THEN** the new file has mode `0600`

#### Scenario: Rewriting keeps the file private

- **WHEN** the agent rewrites a configuration file whose mode is `0600`
- **THEN** the file still has mode `0600`

#### Scenario: A file from an earlier release is narrowed

- **WHEN** the agent starts and its configuration file has mode `0644` and belongs to the user the agent runs as
- **THEN** the file has mode `0600` before the agent reads the token
- **AND** the agent starts without a warning about the file

#### Scenario: The mode cannot be narrowed

- **WHEN** the agent starts and its configuration file has mode `0644`, belongs to another user and the agent cannot change its mode
- **THEN** the agent starts and verifies backups as it would with a private file
- **AND** the console of the starting command shows a warning whose first line starts with `WARNING:`, which says that other users of the machine can read the agent token, and which names the file's path, its mode, its owner, the operating system's error, the command that sets mode `0600` and the command that changes the owner
- **AND** the agent's log holds a record whose message starts with `warning:` and that names the same path, mode, owner, error and commands
- **AND** neither the console nor the log contains the token

#### Scenario: The warning repeats until the file is fixed

- **WHEN** the previous start showed the warning because the mode could not be narrowed, and the agent starts again with the file unchanged
- **THEN** the agent shows the warning again on the console and in its log

#### Scenario: A file without group or other bits is left alone

- **WHEN** the agent starts and its configuration file has mode `0400`
- **THEN** the file keeps mode `0400` and the agent shows no warning about it

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

A secret that carries its own address, such as a Discord or Teams webhook URL, an Azure connection string or a Telegram proxy URL while the proxy stays on, and a secret whose service has a fixed address, such as a Slack bot token, SHALL keep its stored value when the request leaves it out.

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
