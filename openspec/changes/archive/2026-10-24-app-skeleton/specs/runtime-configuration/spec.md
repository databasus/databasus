## ADDED Requirements

### Requirement: An empty setting value means the setting is not set

An empty value of any environment variable the application reads itself SHALL mean that the setting is not set. For an optional setting this is the default value, and startup SHALL NOT stop. For a required setting startup SHALL stop with a log message saying that a required setting is missing and with a non-zero exit code, as when the variable is absent.

#### Scenario: Empty mail server port

- **WHEN** the operator sets `SMTP_PORT` to an empty value
- **THEN** the application starts the same way as without this variable

#### Scenario: Empty connection string with a published one

- **WHEN** the operator sets `DATABASE_DSN` to an empty value, and the container startup script has published a connection string
- **THEN** the application connects with the published string and starts

#### Scenario: Empty connection string to the internal database

- **WHEN** the operator sets `DATABASE_DSN` to an empty value, and no connection string is published
- **THEN** startup stops with a log message saying that a required setting is missing, and the process exits with a non-zero code

### Requirement: The application does not change the environment of the tools it starts

The application SHALL read its settings without changing the environment variables of its process. The database client tools and other programs the application starts SHALL get the variables the operator set unchanged, including variables with an empty value. Values the application loaded for itself SHALL NOT reach the environment of its process, and so of the tools: the values from the environment file and the connection string the process read from the file published at container startup. Variables the application adds for a particular tool, such as a password file or a client encoding, are outside this requirement.

#### Scenario: Operator set a variable empty

- **WHEN** the operator passes a variable with an empty value to the container and a backup runs
- **THEN** the dump tool the backup starts sees this variable with an empty value

#### Scenario: Values from the environment file do not reach the tools

- **WHEN** a developer starts the application with an environment file that sets a variable absent from the process environment, and a backup runs
- **THEN** the dump tool does not see this variable

## MODIFIED Requirements

### Requirement: The environment file is optional

When the application starts without an environment file, it SHALL read its settings from the process environment alone and SHALL report in its log that no environment file was loaded. When an environment file exists, the application SHALL read it and SHALL NOT let it override a variable that is set in the process environment to a non-empty value. A variable set to an empty value counts as not set, so its value comes from the file. A required setting that is missing from both SHALL still stop startup with a log message saying that a required setting is missing.

#### Scenario: Container without an environment file

- **WHEN** the published image starts with only the variables the operator passed in
- **THEN** the application starts and logs that no environment file was loaded

#### Scenario: Development checkout without an environment file

- **WHEN** a developer starts the backend from a checkout that has no environment file, with neither the application mode nor the database connection set in the shell
- **THEN** the log says that no environment file was loaded, startup stops with a message saying a required setting is missing, and the process exits with a non-zero status

#### Scenario: Operator value wins over the file

- **WHEN** a variable is set in the process environment to a non-empty value and the environment file sets it to a different value
- **THEN** the application uses the value from the process environment

#### Scenario: Empty process value is filled from the file

- **WHEN** a variable is set in the process environment to an empty value and the environment file sets it to a value
- **THEN** the application uses the value from the file
