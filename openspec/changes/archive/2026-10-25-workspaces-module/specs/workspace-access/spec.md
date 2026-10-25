## Purpose

Defines who may see and change a workspace and what is inside it: what each workspace role allows, what an instance admin may do, and how the API answers a request the actor may not make.

## ADDED Requirements

### Requirement: Each workspace role allows a fixed set of actions

A member's workspace role SHALL decide what the member may do in the workspace, and the roles SHALL build on each other:

- a viewer MAY see the workspace, its members and its content;
- a member MAY also create, change and delete the workspace's databases, storages and notifiers and everything attached to them;
- an admin MAY also rename the workspace and add, change and remove members who are not admins or the owner;
- the owner MAY also add, change and remove admins, transfer ownership and delete the workspace.

The operations on workspaces and their members SHALL apply these rules with the answers of the requirements below. The operations on a workspace's databases, storages and notifiers SHALL apply the same rules, and SHALL keep their current error answers until those parts of the product move behind the API contract.

#### Scenario: Viewer reads the workspace

- **WHEN** a viewer of a workspace reads it and lists its members
- **THEN** both requests succeed

#### Scenario: Viewer renames the workspace

- **WHEN** a viewer of a workspace renames it
- **THEN** the response is `403` with `code` `permission_denied`, and the name does not change

#### Scenario: Member renames the workspace

- **WHEN** a user with the role `WORKSPACE_MEMBER` renames the workspace
- **THEN** the response is `403` with `code` `permission_denied`, and the name does not change

#### Scenario: Member adds a member

- **WHEN** a user with the role `WORKSPACE_MEMBER` adds another user to the workspace
- **THEN** the response is `403` with `code` `permission_denied`, and the user does not become a member

#### Scenario: Member and viewer on the workspace's content

- **WHEN** the content operations ask whether a user with the role `WORKSPACE_MEMBER` and a viewer of the workspace may edit its content
- **THEN** the member may and the viewer may not

#### Scenario: Admin renames the workspace

- **WHEN** an admin of a workspace renames it
- **THEN** the workspace has the new name

#### Scenario: Admin deletes the workspace

- **WHEN** an admin of a workspace who is not its owner deletes it
- **THEN** the response is `403` with `code` `permission_denied`, and the workspace remains

### Requirement: An instance admin may do everything in every workspace

An instance admin SHALL be allowed every action in every workspace, whether or not they are a member of it.

#### Scenario: Instance admin outside the workspace

- **WHEN** an instance admin who is not a member of a workspace reads it, renames it and lists its members
- **THEN** every request succeeds

#### Scenario: Instance admin deletes a workspace they are not a member of

- **WHEN** an instance admin who is not a member of an empty workspace deletes it
- **THEN** the workspace is deleted

### Requirement: A user outside a workspace learns nothing about it

An operation on a workspace or on one of its members, asked by a user who is not a member and not an instance admin, SHALL get `404` with `code` `not_found`, the same answer as for a workspace that does not exist. The response SHALL NOT reveal the workspace's name or members.

#### Scenario: Reading another team's workspace

- **WHEN** a user who is not a member of a workspace reads it by its ID
- **THEN** the response is `404` with `code` `not_found`

#### Scenario: Same answer as a missing workspace

- **WHEN** a user who is not a member reads an existing workspace and then a workspace ID that does not exist
- **THEN** both responses are `404` with `code` `not_found` and the same body apart from the ID

#### Scenario: Changing a member of another team's workspace

- **WHEN** a user who is not a member of a workspace changes the role of one of its members by the member's ID
- **THEN** the response is `404` with `code` `not_found`, and the role does not change

### Requirement: A role too low for the action is refused without a change

An operation on a workspace or on one of its members, asked by a member whose role does not allow it, SHALL get `403` with `code` `permission_denied` and SHALL change nothing.

#### Scenario: Viewer adds a member

- **WHEN** a viewer of a workspace adds another user as a member
- **THEN** the response is `403` with `code` `permission_denied`, and the user does not become a member

### Requirement: A user sees only their own workspaces

The list of workspaces SHALL hold the workspaces the signed-in user is a member of, each with the user's role in it. For an instance admin it SHALL hold every workspace of the instance, with the admin's role in the workspaces they are a member of and no role in the others.

#### Scenario: Member lists workspaces

- **WHEN** a user who is a member of one of two workspaces lists workspaces
- **THEN** the list holds that one workspace with the user's role

#### Scenario: Instance admin lists workspaces

- **WHEN** an instance admin who is the owner of one workspace and not a member of another lists workspaces
- **THEN** the list holds both workspaces, the first with the role `WORKSPACE_OWNER` and the second without a role

### Requirement: The instance settings decide who creates workspaces

An instance admin SHALL be allowed to create a workspace. A user who is not an instance admin SHALL be allowed only while the instance settings let members create workspaces, and otherwise SHALL get `403` with `code` `permission_denied`. The creator SHALL become the workspace's owner.

#### Scenario: Member creates a workspace when the setting allows it

- **WHEN** the instance settings let members create workspaces and a user who is not an instance admin creates one
- **THEN** the workspace exists and the user is its owner

#### Scenario: Member creates a workspace when the setting forbids it

- **WHEN** the instance settings do not let members create workspaces and a user who is not an instance admin creates one
- **THEN** the response is `403` with `code` `permission_denied`, and no workspace is created
