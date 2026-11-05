---
name: api-design
description: Design HTTP operations of Databasus in backend/api/openapi.yaml and backend/api/agent.yaml. Covers paths and methods, fields, enums and secrets, polymorphic settings, the Problem error format and status codes, access marks, long operations, cursor pagination, downloads and compatibility, and how the backend implements an operation and the frontend calls it. Load when a plan or a change adds or changes an HTTP operation or edits either contract.
---

# Designing the HTTP API

## Who calls the API

- `openapi.yaml` serves the SPA and users' own scripts. The SPA has no private endpoints and uses the same operations a script would.
- `agent.yaml` serves the verification agent. The SPA does not use it, and `openapi.yaml` holds no agent operations.

## Contract first

1. Edit the contract.
2. Run `mise run generate-api`.
3. Implement the generated strict server interface in the module's `internal/httpapi`, and call the operation from the frontend through the generated types. A module serving `agent.yaml` implements its generated strict server in `internal/agentapi`. The `system` tag is the one exception: it is not a module, and `internal/app/systemhttpapi` serves it.
4. Run `mise run backend:lint`. `vacuum` checks the contract against the rules below, and the lint fails when the generated code is out of date.

Every operation has an `operationId` and exactly one tag. The tag is the name of the module that implements the operation. Generated files are never edited by hand.

## Paths and methods

- Paths are relative to `/api/v1` and made of lowercase plural nouns, with words joined by hyphens: `/backup-configs/{backupConfigId}`. A path parameter names its entity: `{storageId}`, never `{id}`.
- `GET` reads, `POST` on a collection creates, `PUT` replaces the editable fields of a resource, and `DELETE` deletes. An action that does not change fields is a `POST` on a sub-path named by a verb: `POST /backups/{backupId}/cancel`, `POST /storages/{storageId}/test`.
- A resource that belongs to a workspace is listed with a filter, such as `GET /storages?workspaceId=...`. It is not nested under the workspace path.

## Fields

- Names are `camelCase`. Enum values are `UPPER_SNAKE_CASE`, except the error codes of `ProblemCode`, which are lower `snake_case` as the table in [Errors](#errors) shows, and the names a database engine itself uses, such as server versions and libpq SSL modes, which a schema marked `x-engine-values: true` keeps as the engine writes them. IDs are strings with `format: uuid`; an identifier that another system issues keeps that system's type, such as the integer Telegram topic `threadId`. Times are RFC 3339 in UTC with `format: date-time`. Sizes and durations carry the unit in the name: `sizeBytes`, `intervalSeconds`.
- A field that may be missing is optional. A field that is present with an empty value is `nullable`. One meaning never uses both.
- Secrets (passwords, keys, tokens) are `writeOnly`. A response never carries them. On `PUT`, omission alone keeps a secret's stored value while the settings' [resource address](../../../docs/glossary.md) stays the stored one; at another address the secret must be sent again. Other removal requires an explicit request signal. Document provider-specific retention and removal semantics in the [API contract](../../../backend/api/openapi.yaml), alongside the relevant operation or settings schema. A separate exception to the response rule is a credential an operation itself issues, such as a session token or a single-use download token: it is returned once in that operation's response and in no other.
- Some resources have settings that depend on their type: a storage, a notifier and a database. Such a resource puts its settings into a `oneOf` with the discriminator `type` and one schema per provider.

## Errors

Every operation has a `default` response with the `Problem` schema (RFC 9457) and a `code` field. The frontend chooses its text by `code`, so `detail` is meant for logs and scripts, not for people.

| Status | When | Typical `code` |
|---|---|---|
| 400 | the request breaks the contract or a validation rule; `errors` lists the fields | `invalid_request` |
| 401 | no valid session | `unauthenticated` |
| 403 | the actor sees the object but its role does not allow the action | `permission_denied` |
| 404 | the object does not exist or belongs to a workspace the actor is not a member of | `not_found` |
| 409 | the state forbids the action, for example the object is in use, the work is already running or the object would be a duplicate | `in_use`, `already_running` |
| 429 | a rate limit was hit | `rate_limited` |
| 500 | anything unexpected; the body has no internal details | `internal` |

A start refused by the target's backup config answers 400 `invalid_request` with a field problem on `databaseId` rather than 409: `startLogicalBackup` and `startPhysicalBackup` do so when the config names no storage. `startPhysicalBackup` also does so when backups are turned off, because its WAL stream has stopped, while `startLogicalBackup` accepts a manual start then.

A new `code` is added to its contract's enum in the same change. A code of `openapi.yaml` is also added to the frontend's dictionary; the frontend does not receive codes exclusive to `agent.yaml`.

## Access

- Operations of the instance admin area carry `x-access: admin`, and the `users` middleware enforces the mark. An operation without a mark requires a signed-in user.
- The module checks workspace access through `workspaces.Authorize`, which answers 404 and 403 as the table describes. The contract documents both responses.
- An operation without a session, such as sign-in or a download by token, carries `x-access: public` and documents why.

## Long operations

Starting a backup, a restore or a verification answers 202 with the resource in its in-progress status. The client follows progress with `GET` on the resource. The operation never blocks until the work ends. A cancel whose status is final when it answers responds 200 with the resource. Starting a physical backup is an exception, for the reason in [architecture.md, HTTP API](../../../docs/architecture.md#http-api).

## Pagination

A list takes `limit` and `after` and returns `items` and `nextCursor`. The cursor is opaque. `nextCursor` is absent on the last page. A list has a fixed order, documented in the operation. `OFFSET` and counts of rows for paging are not offered. A figure that is not a paging count may sit on the page.

## Downloads

A file the browser downloads is returned with `Content-Disposition` and with `application/octet-stream` or the file's real media type. A link cannot carry the session header. The browser therefore first gets a single-use token from an authenticated `POST`, then passes it as the `token` query parameter. Tokens never appear in logs.

## Compatibility

Follow the compatibility policy in [architecture.md, HTTP API](../../../docs/architecture.md#http-api), including frozen agent-discovery paths and explicit approval of request-shape breaks.

## Backend side

- The handler maps the generated request to one service call, and the call's result to the generated response. It returns errors through `lib/apierror`, so their body always matches `Problem`.
- The module's tests call the operation through the client from `apptest`, which checks every response against the contract.

## Frontend side

- Types come only from `shared/api/schema.gen.ts`. Do not write API types by hand.
- Requests go through the generated client in `shared/api`. Errors reach the user only through `translateApiError`, which chooses the text by `code`.
- A request is retried only when it failed on the network or with 502, 503 or 504, never on a 4xx.

## Before handing over

- The contract passes `mise run backend:lint`, and the generated code is up to date.
- Every operation has its tag, its `operationId`, an access mark where needed, the `default` error response and the statuses its access check can give.
- Secrets are `writeOnly`, settings that vary by type are a `oneOf`, and lists page with a cursor.
- A breaking change is marked in the proposal.
