# Workspace Members Specification

## Purpose

Defines how the people of a workspace are managed: adding and inviting members, changing their roles, removing them and handing the workspace to a new owner, with exactly one owner per workspace.

## Requirements

### Requirement: Adding an existing user makes them a member

Adding a member by the email address of an existing user SHALL make that user a member of the workspace with the requested role, and the answer SHALL report that the user was added. Adding a user who is already a member SHALL get `409` with `code` `already_member` and SHALL NOT change their role.

#### Scenario: User with an account

- **WHEN** an admin of a workspace adds the address of an existing user as a member with the role `WORKSPACE_MEMBER`
- **THEN** the user is listed among the workspace's members with that role, and the answer reports that the user was added

#### Scenario: User who is already a member

- **WHEN** an admin of a workspace adds the address of a user who is already a viewer there, asking for `WORKSPACE_MEMBER`
- **THEN** the response is `409` with `code` `already_member`, and the user stays a viewer

### Requirement: Adding an unknown address invites the user

Adding a member by an address that has no user SHALL invite a user with that address and make them a member with the requested role, and the answer SHALL report that the user was invited. A user who is not an instance admin SHALL be allowed to invite only while the instance settings allow member invitations. Otherwise the request SHALL get `403` with `code` `permission_denied`, and no user, member or email SHALL be created.

#### Scenario: Invitation allowed

- **WHEN** the instance settings allow member invitations and an admin of a workspace adds an address that has no user
- **THEN** the address is listed among the workspace's members as an invited user with the requested role, and the answer reports that the user was invited

#### Scenario: Invitation not allowed

- **WHEN** the instance settings do not allow member invitations and an admin of a workspace who is not an instance admin adds an address that has no user
- **THEN** the response is `403` with `code` `permission_denied`, no user with that address exists, and no email is sent

### Requirement: The owner role changes hands only through a transfer

Adding a member and changing a member's role SHALL accept only the roles `WORKSPACE_ADMIN`, `WORKSPACE_MEMBER` and `WORKSPACE_VIEWER`. A request with `WORKSPACE_OWNER` SHALL get `400` with `code` `invalid_request` and SHALL change nothing. Changing the role of the owner or removing the owner SHALL get `409` with `code` `owner_requires_transfer`.

#### Scenario: Admin makes another member the owner

- **WHEN** an admin of a workspace changes the role of a member to `WORKSPACE_OWNER`
- **THEN** the response is `400` with `code` `invalid_request`, and the workspace keeps its owner and the member keeps their role

#### Scenario: Adding a member as owner

- **WHEN** the owner of a workspace adds a user with the role `WORKSPACE_OWNER`
- **THEN** the response is `400` with `code` `invalid_request`, and the user does not become a member

#### Scenario: Demoting the owner

- **WHEN** an instance admin changes the role of a workspace's owner to `WORKSPACE_ADMIN`
- **THEN** the response is `409` with `code` `owner_requires_transfer`, and the owner keeps the role

#### Scenario: Removing the owner

- **WHEN** an instance admin removes the owner of a workspace from it
- **THEN** the response is `409` with `code` `owner_requires_transfer`, and the owner stays a member

### Requirement: Only the owner manages admins

Making a member an admin, changing the role of an admin, adding a user as an admin and removing an admin SHALL be allowed only to the owner of the workspace and to an instance admin. An admin of the workspace SHALL get `403` with `code` `permission_denied` for each of them and SHALL change nothing, also when the admin is the member being changed. The permission check SHALL come before the checks of the own role and of the owner, so an actor without the permission always gets `403`.

#### Scenario: Admin demotes another admin

- **WHEN** an admin of a workspace changes the role of another admin to `WORKSPACE_VIEWER`
- **THEN** the response is `403` with `code` `permission_denied`, and the other admin keeps the role

#### Scenario: Admin promotes a member

- **WHEN** an admin of a workspace changes the role of a member to `WORKSPACE_ADMIN`
- **THEN** the response is `403` with `code` `permission_denied`, and the member keeps the role

#### Scenario: Admin removes another admin

- **WHEN** an admin of a workspace removes another admin
- **THEN** the response is `403` with `code` `permission_denied`, and the other admin stays a member

#### Scenario: Owner promotes a member

- **WHEN** the owner of a workspace changes the role of a member to `WORKSPACE_ADMIN`
- **THEN** the member is an admin

### Requirement: A member cannot change their own role

A request allowed by the permission check that changes the role of the actor's own member SHALL get `409` with `code` `cannot_change_own_role` and SHALL change nothing. When the actor's own member is also the owner, `cannot_change_own_role` SHALL be the answer rather than `owner_requires_transfer`.

#### Scenario: Owner changes their own role

- **WHEN** the owner of a workspace changes their own role to `WORKSPACE_ADMIN`
- **THEN** the response is `409` with `code` `cannot_change_own_role`, and the owner keeps the role

#### Scenario: Admin demotes themselves

- **WHEN** an admin of a workspace changes their own role to `WORKSPACE_VIEWER`
- **THEN** the response is `403` with `code` `permission_denied`, because changing an admin's role needs the permission to manage admins, and the admin keeps the role

### Requirement: Removing a member takes away their access

Removing a member SHALL end the user's access to the workspace. The user's account SHALL remain.

#### Scenario: Removed member reads the workspace

- **WHEN** an admin of a workspace removes a member and that user then reads the workspace
- **THEN** the user gets `404` with `code` `not_found`, and the user can still sign in

### Requirement: Ownership transfer keeps exactly one owner

The owner of a workspace and an instance admin SHALL be able to transfer ownership to another member of the workspace. The new owner SHALL get the owner role and the previous owner SHALL become an admin, both at once, and the workspace SHALL have exactly one owner after every transfer. Naming a member that is not in the workspace SHALL get `404` with `code` `not_found`. Transferring to the current owner SHALL succeed and change nothing.

#### Scenario: Owner hands the workspace to an admin

- **WHEN** the owner of a workspace transfers ownership to one of its admins
- **THEN** that admin is the owner, the previous owner is an admin, and the workspace has one owner

#### Scenario: Owner transfers to themselves

- **WHEN** the owner of a workspace transfers ownership naming their own member
- **THEN** the request succeeds, the owner stays the owner, and every other member keeps their role

#### Scenario: Admin transfers ownership

- **WHEN** an admin of a workspace who is not its owner transfers ownership to a member
- **THEN** the response is `403` with `code` `permission_denied`, and the owner does not change

#### Scenario: Transfer to a member of another workspace

- **WHEN** the owner of a workspace transfers ownership naming a member of a different workspace
- **THEN** the response is `404` with `code` `not_found`, and the owner does not change

### Requirement: The member list shows who the members are

The member list SHALL show for each member the user's email address, name and workspace role, and whether the user was invited and has not finished sign-up. It SHALL list the members in the order they joined, oldest first, and SHALL page with a cursor.

#### Scenario: Invited member in the list

- **WHEN** an admin of a workspace invites an address and then lists the members
- **THEN** the list shows that address with the requested role, marked as invited, after the members who joined earlier

#### Scenario: Member list longer than one page

- **WHEN** a workspace has more members than the requested page size and a client follows the cursor to the end
- **THEN** every member is listed exactly once, in the order they joined
