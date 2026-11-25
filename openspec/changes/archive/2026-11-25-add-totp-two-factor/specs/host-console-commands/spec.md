## MODIFIED Requirements

### Requirement: The binary offers a stable set of console commands

The Databasus binary SHALL accept these commands with these flags:

- the `healthcheck` subcommand checks that the running server answers the version request;
- `--test-storage` saves and deletes a probe file through local storage;
- `--new-password` with `--email` changes an account's password;
- `--create-user` with `--email`, `--password`, `--role` and an optional `--name` creates an account;
- `--disable-2fa` turns off the second factor at sign-in;
- `--reset-2fa` with `--email` removes one account's authenticator app and its recovery codes;
- `--list-admins` prints the addresses of all administrators.

The usage text of `--disable-2fa`, `--reset-2fa` and `--email` SHALL describe both second-factor methods and both commands that take an address. What each command does with accounts and settings is described by the `root-admin-account`, `two-factor-authentication` and `authenticator-app` specs. The names of the commands and flags SHALL NOT change, because the user documentation and the container startup script record them. For the same reason the binary in the image SHALL stay available as `/app/main`, that is as `./main` from the container's working directory, and as `databasus` on `PATH`.

#### Scenario: Operator resets a password following the README

- **WHEN** the operator runs `docker exec -it databasus ./main --new-password=<password> --email=<address>` for an existing account
- **THEN** the command prints that the password was reset and exits with code 0
- **AND** the account signs in with the new password

#### Scenario: Operator creates an account

- **WHEN** the operator runs `docker exec -it databasus ./main --create-user --email=<address> --password=<password> --role=member`
- **THEN** the command prints the result and exits with code 0
- **AND** the account signs in with that address and password

#### Scenario: Operator disables the second factor

- **WHEN** the operator runs `docker exec -it databasus ./main --disable-2fa` on an instance that requires a code at sign-in
- **THEN** the command prints that the second factor is off and exits with code 0
- **AND** the next password sign-in of an account without an authenticator app needs no code

#### Scenario: Operator resets an account's authenticator app

- **WHEN** the operator runs `docker exec -it databasus ./main --reset-2fa --email=<address>` for an account with a connected authenticator app
- **THEN** the command prints that the app was reset and exits with code 0
- **AND** the account's next password sign-in no longer asks for a code from that app

#### Scenario: Operator looks up an administrator's address

- **WHEN** the operator runs `docker exec -it databasus ./main --list-admins`
- **THEN** the command prints the addresses of all administrators unmasked and exits with code 0

### Requirement: A console command does only its own work and exits

A console command SHALL perform its action and end the process. It SHALL NOT open an HTTP port and SHALL NOT start background work: backups, restores, health checks, cleanups and telemetry. On success the command SHALL exit with code 0. On failure it SHALL exit with a non-zero code, report the reason and SHALL NOT change any account or any instance setting. Migrations the command applied before its action stay applied.

The binary SHALL refuse before reading the connection string and before connecting to the internal database in four cases:

- `--create-user` is passed together with `--new-password`: the message explains that the new account's password goes in `--password`;
- `--create-user` is passed without `--email`;
- `--new-password` is passed without `--email`;
- `--reset-2fa` is passed without `--email`.

The account creation command checks the values of `--password`, `--role` and `--name` as the `root-admin-account` spec requires, after connecting to the database, and on refusal also changes no account. When several command flags are passed, the binary SHALL run only the first command in the order `--test-storage`, `--list-admins`, `--create-user`, `--new-password`, `--disable-2fa`, `--reset-2fa`, and the other command flags SHALL NOT run.

#### Scenario: Command inside a running container

- **WHEN** the operator opens a shell in a running container and runs a password reset there
- **THEN** the command finds this container's internal database, resets the password and exits with code 0
- **AND** the command's process opens no listening port and starts no backups, restores, health checks, cleanups or telemetry
- **AND** the running server answers 200 on `GET /api/v1/system/version` the whole time

#### Scenario: Password reset for an unknown address

- **WHEN** the operator runs a password reset with an address the instance does not have
- **THEN** the command exits with a non-zero code, reports the error and changes no account

#### Scenario: Password reset without an address

- **WHEN** the operator runs `--new-password=<password>` without `--email`, and the connection string to the internal database is neither set nor published
- **THEN** the command exits with a non-zero code and says that the address goes in `--email`
- **AND** the output has no message about a missing connection string, because the command never got that far

#### Scenario: Authenticator reset without an address

- **WHEN** the operator runs `--reset-2fa` without `--email`, and the connection string to the internal database is neither set nor published
- **THEN** the command exits with a non-zero code, says that the address goes in `--email` and changes no account
- **AND** the output has no message about a missing connection string, because the command never got that far

#### Scenario: Disable and reset passed together

- **WHEN** the operator runs `--disable-2fa --reset-2fa --email=<address>`
- **THEN** only the second factor requirement is switched off, and the account's authenticator app stays connected

#### Scenario: Account creation with the password reset flag

- **WHEN** the operator runs `--create-user` together with `--new-password`
- **THEN** the command refuses, explains that the password goes in `--password`, exits with a non-zero code and changes no account

### Requirement: Commands write the same audit records as before

Creating an account, resetting a password, disabling the second factor and resetting an authenticator app from the console SHALL write the same audit events as these actions do outside the console: the record of the created account that the `root-admin-account` spec requires, the record of the password change, the record of disabling the second factor that the `two-factor-authentication` spec requires, and the record of the app reset that the `authenticator-app` spec requires. A console command SHALL NOT skip an audit record because it runs without the server.

#### Scenario: Account created from the console

- **WHEN** the operator creates an account from the console
- **THEN** the audit log holds a record with its address and role saying that it was created from the host console

#### Scenario: Password reset from the console

- **WHEN** the operator resets an account's password from the console
- **THEN** the audit log holds a record of that account's password change

#### Scenario: Second factor disabled from the console

- **WHEN** the operator disables the second factor from the console
- **THEN** the audit log holds a record saying that the second factor was disabled from the host

#### Scenario: Authenticator app reset from the console

- **WHEN** the operator resets an account's authenticator app from the console
- **THEN** the audit log holds a record naming that account and saying that its app was reset from the host
