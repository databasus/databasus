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
2. Run `just generate-api` at the root.
3. Implement the generated strict server interface in the module's `internal/httpapi`, and call the operation from the frontend through the generated types.
4. Run `just lint` in `backend/`. `vacuum` checks the contract against the rules below, and the lint fails when the generated code is out of date.

Every operation has an `operationId` and exactly one tag. The tag is the name of the module that implements the operation. Generated files are never edited by hand.

## Paths and methods

- Paths are relative to `/api/v1` and made of lowercase plural nouns, with words joined by hyphens: `/backup-configs/{backupConfigId}`. A path parameter names its entity: `{storageId}`, never `{id}`.
- `GET` reads, `POST` on a collection creates, `PUT` replaces the editable fields of a resource, and `DELETE` deletes. An action that does not change fields is a `POST` on a sub-path named by a verb: `POST /backups/{backupId}/cancel`, `POST /storages/{storageId}/test`.
- A resource that belongs to a workspace is listed with a filter, such as `GET /storages?workspaceId=...`. It is not nested under the workspace path.

## Fields

- Names are `camelCase`. Enum values are `UPPER_SNAKE_CASE`. IDs are strings with `format: uuid`. Times are RFC 3339 in UTC with `format: date-time`. Sizes and durations carry the unit in the name: `sizeBytes`, `intervalSeconds`.
- A field that may be missing is optional. A field that is present with an empty value is `nullable`. One meaning never uses both.
- Secrets (passwords, keys, tokens) are `writeOnly`. A response never carries them. On `PUT`, an omitted secret keeps its stored value.
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

A new `code` is added to the contract's enum and to the frontend's dictionary in the same change.

## Access

- Operations of the instance admin area carry `x-access: admin`, and the `users` middleware enforces the mark. An operation without a mark requires a signed-in user.
- The module checks workspace access through `workspaces.Authorize`, which answers 404 and 403 as the table describes. The contract documents both responses.
- An operation without a session, such as sign-in or a download by token, carries `x-access: public` and documents why.

## Long operations

Starting a backup, a restore or a verification answers 202 with the resource in its in-progress status. The client follows progress with `GET` on the resource. The operation never blocks until the work ends.

## Pagination

A list takes `limit` and `after` and returns `items` and `nextCursor`. The cursor is opaque. `nextCursor` is absent on the last page. A list has a fixed order, documented in the operation. `OFFSET` and totals are not offered.

## Downloads

A file the browser downloads is returned with `Content-Disposition` and with `application/octet-stream` or the file's real media type. A link cannot carry the session header. The browser therefore first gets a single-use token from an authenticated `POST`, then passes it as the `token` query parameter. Tokens never appear in logs.

## Compatibility

- `openapi.yaml` may change incompatibly in a release when the frontend changes in the same change. The change's proposal marks it as breaking, so the release notes mention it for users' scripts.
- `/api/v1/system/version` and `/api/v1/system/verification-agent` never change. Installed agents update themselves through them.
- An operation in `agent.yaml` stays compatible with the agent of the previous release, because an agent keeps running until its next update check.

## Backend side

- The handler maps the generated request to one service call, and the call's result to the generated response. It returns errors through `lib/apierror`, so their body always matches `Problem`.
- The module's tests call the operation through the client from `apptest`, which checks every response against the contract.

## Frontend side

- Types come only from `shared/api/schema.gen.ts`. Do not write API types by hand.
- Requests go through the generated client in `shared/api`. Errors reach the user only through `translateApiError`, which chooses the text by `code`.
- A request is retried only when it failed on the network or with 502, 503 or 504, never on a 4xx.

## Before handing over

- The contract passes `just lint`, and the generated code is up to date.
- Every operation has its tag, its `operationId`, an access mark where needed, the `default` error response and the statuses its access check can give.
- Secrets are `writeOnly`, settings that vary by type are a `oneOf`, and lists page with a cursor.
- A breaking change is marked in the proposal.
