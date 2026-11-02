# frontend-database-connection Specification

## Purpose
Make the database connection form predictable in every scenario, so it asks only for what the scenario uses, never treats a secret as stored unless it is, and lets a database be saved only with settings a test approved.

## Requirements

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

### Requirement: A connection test approves only the settings it tested

When adding or editing a database, the form SHALL offer saving or continuing only after a successful test of the current connection settings. Any change to a connection setting SHALL withdraw that approval, even if the value later returns to the tested one. A setting that does not affect the connection, such as a schema or table selection, SHALL keep the approval. An answer to a test SHALL be ignored if the connection settings changed after the test began, if the form closed or if the session changed. A test already pending SHALL prevent another test.

#### Scenario: A password changes during a test

- **WHEN** a user changes the password while a test is pending and the test then succeeds
- **THEN** the form does not offer saving and asks for a new test

#### Scenario: A failure answers after a change

- **WHEN** a user changes the password while a test is pending and the test then fails
- **THEN** the form shows no failure from that test

#### Scenario: A value returns to the tested one

- **WHEN** a test succeeds and the user changes the port and changes it back
- **THEN** the form asks for a new test

#### Scenario: A table selection changes after a test

- **WHEN** a test succeeds and the user adds an excluded table
- **THEN** the form still offers saving

#### Scenario: Leaving during a test

- **WHEN** the form closes while a test is pending and the test then answers
- **THEN** no feedback appears and the answer changes nothing

#### Scenario: The session changes during a test

- **WHEN** the session changes while a test is pending and the test then answers
- **THEN** no feedback appears and the form does not offer saving

#### Scenario: A repeated test click

- **WHEN** a user clicks the test control twice while the first test is pending
- **THEN** exactly one test request is sent

### Requirement: A connection failure keeps the input and explains itself in the form

A failed test or refused submission SHALL keep every entered value and SHALL show its reason inside the form, without a browser dialog. A failure the server classifies for a physical PostgreSQL database SHALL show its steps. A failed test of any other database variant SHALL also show the hint about the allowed IP list. Changing any connection setting, including an SSH setting, SHALL remove the feedback of a failed test. While a submission is pending, the form SHALL accept no input and SHALL prevent another submission. The form's own failure text and classified steps SHALL follow the selected language at render time, and the server's reason SHALL stay as the server wrote it.

#### Scenario: A test cannot reach the server

- **WHEN** a connection test answers that the database could not be reached
- **THEN** the form shows the server's reason inline with the allowed IP list hint, every value stays and no browser dialog opens

#### Scenario: A physical test fails and the SSH host changes

- **WHEN** a physical database's test shows a classified failure and the user changes the SSH host
- **THEN** the failure and its steps disappear

#### Scenario: The language changes while a failure is visible

- **WHEN** a failed test is visible and the user changes the language
- **THEN** the form's text around the failure changes language, the server's reason stays literal and every value stays

#### Scenario: A save is refused

- **WHEN** the server refuses a save of the connection section
- **THEN** the section stays open with every value and shows the translated reason

#### Scenario: A repeated click during a pending save

- **WHEN** a user clicks the save control twice while the first save is pending
- **THEN** exactly one update request is sent

### Requirement: The SSL mode follows the host until the user chooses it

For a new PostgreSQL database or restore target, the form SHALL derive the SSL mode from the host as it is typed, until the user chooses a mode or imports a connection string. A host starting with `https://` SHALL give `require`. A host starting with `http://`, the host `localhost` and an IPv4 address SHALL give `disable`. Any other host SHALL leave the mode unchanged. A choice SHALL remain while the user moves between wizard steps of the same database type, and a new database type SHALL derive its mode again. For a saved database the form SHALL treat the stored mode as chosen, and only a host entered with `https://` SHALL change it, by turning on SSL if it is off. Turning SSL off, whether chosen or derived, SHALL clear the entered certificates.

#### Scenario: A new database on localhost

- **WHEN** a user types `localhost` as the host of a new database without choosing an SSL mode
- **THEN** the SSL mode is `disable`

#### Scenario: A new database on an IPv4 address

- **WHEN** a user types `10.0.0.5` as the host of a new database without choosing an SSL mode
- **THEN** the SSL mode is `disable`

#### Scenario: A new database on a remote hostname

- **WHEN** a user types `db.example.com` as the host of a new database whose SSL mode is `require` without having chosen it
- **THEN** the SSL mode stays `require`

#### Scenario: A choice survives going back

- **WHEN** a user chooses an SSL mode and certificates, continues, goes back and changes the host to `localhost`
- **THEN** the chosen SSL mode and the certificates stay

#### Scenario: A saved database with a new host

- **WHEN** a user changes the host of a saved database to `localhost`
- **THEN** the stored SSL mode stays

#### Scenario: A saved database with an https host

- **WHEN** a user enters a host with `https://` for a saved database whose SSL mode is `disable`
- **THEN** the SSL mode becomes `require`

#### Scenario: A derived mode turns SSL off

- **WHEN** a user has typed certificates for a new database without choosing an SSL mode and then types `localhost` as the host
- **THEN** the SSL mode is `disable` and the certificates are cleared

### Requirement: A connection string fills the form of its own engine

The form SHALL import a connection string from the clipboard. When the clipboard API is unavailable, the form SHALL offer a manual paste. When reading the clipboard fails, the form SHALL show a translated error and change nothing. An accepted string SHALL fill its engine's fields and withdraw test approval. A string the engine does not accept SHALL show a translated error and SHALL change nothing. MariaDB SHALL accept `mariadb://`, `mysql://`, `jdbc:mariadb://`, `jdbc:mysql://` and the key-value format. MySQL SHALL accept `mysql://`, `jdbc:mysql://` and the key-value format, and SHALL refuse the MariaDB schemes.

#### Scenario: A MariaDB string in a MariaDB form

- **WHEN** a user imports `mariadb://user:secret@db.example.com:3306/shop` into a MariaDB form
- **THEN** the host, port, user, password and database name are filled and a new test is required

#### Scenario: A MariaDB string in a MySQL form

- **WHEN** a user imports `mariadb://user:secret@db.example.com:3306/shop` into a MySQL form
- **THEN** the form shows a format error and every field keeps its value

#### Scenario: The clipboard API is unavailable

- **WHEN** a user starts the import where the clipboard API is unavailable
- **THEN** the form offers a manual paste, and a pasted string is imported

#### Scenario: Reading the clipboard fails

- **WHEN** a user starts the import and reading the clipboard fails
- **THEN** the form shows a translated error, opens no manual paste and every field keeps its value

### Requirement: The connection form works by label and keyboard

Every field of the form SHALL be identified by its visible label, and the connection string import SHALL be reachable and usable by keyboard.

#### Scenario: Finding the password field

- **WHEN** an assistive technology lists the form's controls
- **THEN** the password field is announced with its visible label

#### Scenario: Importing by keyboard

- **WHEN** a keyboard user moves focus to the import control and activates it
- **THEN** the import starts as it does with a pointer
