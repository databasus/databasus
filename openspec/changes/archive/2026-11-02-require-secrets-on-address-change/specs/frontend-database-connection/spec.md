## MODIFIED Requirements

### Requirement: The connection form asks for what its scenario uses

Adding a database SHALL require the connection fields of its engine, including the password, and a successful connection test before the user continues. Editing a saved database SHALL require the same fields and test, except that a stored secret MAY stay empty while the address that secret is used at stays the saved one, as the `secret-handling` spec defines the resource address. A restore target SHALL require the same fields as adding a database, SHALL have no connection test and SHALL offer its start once those fields are filled. Every connection option of every engine SHALL remain available when adding and editing, and the PostgreSQL logical, PostgreSQL physical, MySQL, MariaDB and MongoDB variants SHALL keep their own options.

#### Scenario: A new database without a password

- **WHEN** a user fills every other required field of a new database and leaves its password empty
- **THEN** the connection test control is disabled

#### Scenario: A saved database without a new password

- **WHEN** a user changes the user name of a saved database and leaves its password empty
- **THEN** the test control is enabled and the test uses the stored password

#### Scenario: A saved database at another port

- **WHEN** a user changes the port of a saved database and leaves its password empty
- **THEN** a message under the password field says the address changed and the password must be entered again, and the test control is disabled

#### Scenario: A filled restore target

- **WHEN** a user fills every required field of a restore target
- **THEN** the form offers the start control and shows no connection test control

#### Scenario: MariaDB options

- **WHEN** a user opens the advanced settings of a MariaDB database
- **THEN** the event exclusion and Galera options are offered, and they are not offered for a MySQL database

### Requirement: The connection form presents a value as stored only while the saved database still holds it

The form SHALL show a secret, certificate or SSH credential as stored only while editing the saved database that holds it. In every other case the form SHALL show the value the user typed. A saved client certificate SHALL appear as stored only while the form still holds that saved certificate. Turning SSL off or choosing to replace the certificates SHALL remove them from the form, and the form SHALL then show empty editable certificate fields, also after SSL is turned on again. A save SHALL send the certificates the form shows, so an empty client or root certificate is removed from the saved database. A secret left empty while editing, including the client key, SHALL be omitted from the request and keep its stored value, as long as the address that secret is used at stays the saved one. While that address differs from the saved one, as the `secret-handling` spec defines the resource address, the form SHALL require the secret where the settings use it, and SHALL show under its empty field a message that the address changed and the secret must be entered again. The SSH credentials of a tunnel that was disabled in the saved database SHALL be required without that message, because the form never showed them as stored, while the database password SHALL still get it. The SSH credentials SHALL leave their stored state for empty editable fields. A saved client certificate that the form still holds SHALL stay shown as stored, because it is not a secret and is still sent, and an empty required client key field SHALL appear with it, the root certificate unchanged. When the address returns to the saved value, the message SHALL disappear, the stored states SHALL return and the additional client key field SHALL go away while it is empty. A value the user typed SHALL stay, and a typed client key SHALL keep its field visible, so the save never sends a value the form does not show. Each secret SHALL follow its own address, so a database password follows the database's address and its SSH tunnel, and the SSH credentials follow only the tunnel. This is a security rule: a stored secret must not appear available, or be sent to a server, outside the database it belongs to, and the form must not show a certificate as kept while the save removes it.

#### Scenario: A certificate typed in the wizard

- **WHEN** a user types a client certificate for a new database, continues and goes back to the connection step
- **THEN** the form shows the typed certificate as editable text and offers no replace control

#### Scenario: A saved database with a client certificate

- **WHEN** a user edits the connection of a saved database with a client certificate
- **THEN** the form shows the certificate as stored and offers a replace control

#### Scenario: SSL is turned off and on again

- **WHEN** a user turns SSL off and on again for a saved database with a client certificate, tests the connection and saves
- **THEN** the form shows empty certificate fields with no replace control, and the update request contains no certificate and no client key

#### Scenario: Replace is left empty

- **WHEN** a user chooses to replace the certificates of a saved database and leaves the fields empty
- **THEN** the fields stay empty and editable, and the update request contains no certificate and no client key

#### Scenario: An empty secret on save

- **WHEN** a user saves a saved database with its password left empty at the same host and port
- **THEN** neither the test nor the update request contains a password

#### Scenario: The SSH tunnel moved to another host

- **WHEN** a user changes the SSH tunnel host of a saved database with stored SSH credentials
- **THEN** the form shows editable empty SSH credential fields instead of the stored state, and the test control is disabled until the credential and the database password are entered

#### Scenario: The address returned to the saved value

- **WHEN** a user changes the host of a saved database and then restores its saved value
- **THEN** the message under the password field disappears, and the test control is enabled

#### Scenario: The SSH tunnel turned off

- **WHEN** a user turns off the enabled SSH tunnel of a saved database and leaves the password empty
- **THEN** the SSH fields disappear, the message appears under the password field, and the test control is disabled

#### Scenario: The MongoDB SRV flag changed

- **WHEN** a user changes the SRV flag of a saved MongoDB database and leaves the password empty
- **THEN** the message appears under the password field, and the test control is disabled

#### Scenario: A database moved behind the same tunnel

- **WHEN** a user changes the host of a saved database behind an unchanged enabled SSH tunnel with stored credentials and leaves the password empty
- **THEN** the message appears under the password field, the SSH credentials stay in their stored state, and the test control is disabled until the password is entered

#### Scenario: A moved host with a saved client certificate

- **WHEN** a user changes the host of a saved PostgreSQL database with a client certificate and enters the password
- **THEN** the certificate stays shown as stored, an empty client key field with the message appears, and the test control is disabled until the key is entered

#### Scenario: The host returned with a saved client certificate

- **WHEN** a user changes the host of a saved PostgreSQL database with a client certificate and then restores its saved value
- **THEN** only the stored certificate with its replace control is shown

#### Scenario: The host returned with a typed client key

- **WHEN** a user changes the host of a saved PostgreSQL database with a client certificate, types a client key and then restores the saved host
- **THEN** the client key field stays visible with the typed key, and the message is not shown

#### Scenario: A disabled SSH tunnel turned on

- **WHEN** a user turns on the disabled SSH tunnel of a saved database and leaves the secrets empty
- **THEN** the SSH credential fields are empty and show no message, the message appears under the database password, and the test control is disabled

#### Scenario: Replace at a moved host

- **WHEN** a user changes the host of a saved PostgreSQL database with a client certificate, enters the password and chooses to replace the certificates, leaving them empty
- **THEN** no client key is required, and the test control is enabled
