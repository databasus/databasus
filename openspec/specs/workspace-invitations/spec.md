# Workspace Invitations Specification

## Purpose

Describes what a person invited to a workspace without a Databasus account receives by email, and where the invitation leads them, so they can create the account and reach the workspace.

## Requirements

### Requirement: The invitation email leads to the sign-up form

When the instance's public address is configured, the invitation email sent to a person without an account SHALL contain a "Sign up" link to that address exactly as the operator configured it, with no path added. Opening the link while not signed in SHALL show the sign-up form.

#### Scenario: Public address without a trailing slash

- **WHEN** the public address is `https://databasus.example.com` and a workspace owner adds a member by an email address that has no account
- **THEN** the invitation email's "Sign up" link points to `https://databasus.example.com`

#### Scenario: Public address with a trailing slash

- **WHEN** the public address is `https://databasus.example.com/` and a workspace owner adds a member by an email address that has no account
- **THEN** the invitation email's "Sign up" link points to `https://databasus.example.com/`

#### Scenario: Invited person opens the link

- **WHEN** the invited person opens the "Sign up" link in a browser that is not signed in to the instance
- **THEN** the sign-up form is shown, and completing it with the invited address gives the account the workspace membership from the invitation

### Requirement: An unknown address in the web app leads to the main screen

The web app SHALL send any address it has no page for to its root, replacing the unknown address in the browser history. It SHALL never render a blank page for such an address. This includes the `/sign-up` address that earlier invitation emails link to.

#### Scenario: Link from an earlier invitation email

- **WHEN** a visitor who is not signed in opens `<public address>/sign-up`
- **THEN** the browser ends up at the root of the web app and the sign-up form is shown

#### Scenario: Unknown address while signed in

- **WHEN** a signed-in user opens an address the web app has no page for
- **THEN** the browser ends up at the root of the web app and the user's main screen is shown

#### Scenario: Back navigation after the redirect

- **WHEN** a visitor is sent from an unknown address to the root and then presses the browser's back button
- **THEN** the browser returns to the page before the unknown address instead of to the unknown address again

### Requirement: An invitation is recorded before its email is sent

Adding a member by an address that has no user SHALL create the invited user and the member together: either both exist after the request or neither does. The invitation email SHALL be sent only after both are recorded. A failure to send the email SHALL NOT undo them, and the request SHALL still report that the user was invited.

#### Scenario: Mail server refuses the invitation email

- **WHEN** an admin of a workspace adds an address that has no user and the mail server refuses the invitation email
- **THEN** the answer has the outcome `INVITED`, and the address is listed among the workspace's members as an invited user

#### Scenario: Adding the member fails

- **WHEN** adding a member by an address that has no user fails after the invited user was created
- **THEN** the response is an error and not the outcome `INVITED`, no user with that address exists, the address is not a member, and no invitation email is sent
