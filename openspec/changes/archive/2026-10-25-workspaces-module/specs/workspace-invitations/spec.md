## ADDED Requirements

### Requirement: An invitation is recorded before its email is sent

Adding a member by an address that has no user SHALL create the invited user and the member together: either both exist after the request or neither does. The invitation email SHALL be sent only after both are recorded. A failure to send the email SHALL NOT undo them, and the request SHALL still report that the user was invited.

#### Scenario: Mail server refuses the invitation email

- **WHEN** an admin of a workspace adds an address that has no user and the mail server refuses the invitation email
- **THEN** the answer has the outcome `INVITED`, and the address is listed among the workspace's members as an invited user

#### Scenario: Adding the member fails

- **WHEN** adding a member by an address that has no user fails after the invited user was created
- **THEN** the response is an error and not the outcome `INVITED`, no user with that address exists, the address is not a member, and no invitation email is sent
