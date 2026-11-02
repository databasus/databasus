## Context

See `proposal.md` for the motivation. Forms never receive secrets back, so a request about a saved resource leaves them empty and the backend fills them from the stored row.

Every such path in the three families goes through one merge, and `Validate` runs right after it:

| Family | Merge | Called by |
|---|---|---|
| Notifiers | `notifiers/internal/notifier/service.go:316`, provider call at `:331` | change at `:182`, unsaved test at `test_send_service.go:120` |
| Storages | `storages/internal/storage/service.go:306`, provider call at `:325` | change at `:211`, unsaved test at `connection_test_service.go:115` |
| Databases | `databases/internal/userdatabase/service.go:424`, provider call at `:437` | change at `:246`; unsaved test and both limited-user operations at `settings_target_service.go:64`, through `limiteduser/service.go:46` and `:79` |

Each provider's merge fills every empty secret without looking at the address, for example `storages/internal/providers/s3/provider.go:70`, `notifiers/internal/providers/telegram/provider.go:104` and `notifiers/internal/providers/webhook/provider.go:85`. `TestSettings` of notifiers authorizes with `ActionView` (`test_send_service.go:84`), while storages and databases already need `ActionEditContent` (`connection_test_service.go:68`, `settings_target_service.go:40`).

Restores take their target's secrets only from the request, and tests of saved resources use only stored settings, so neither reaches the merge.

The approach is constrained by these rules: a domain rule stays in the module that owns the data, a service checks access first, and only the family's registry maps a type to its implementation (`backend/AGENTS.md`). On the frontend, code stays inside its feature-sliced layer and displayed copy lives in the six dictionaries (`frontend/AGENTS.md`). `docs/architecture.md:280` allows a release to break the contract when the same change updates the frontend, so this break needs no shim. The architecture does not change, and no package is added.

## Goals / Non-Goals

**Goals:**

- Close the path at the merge, so a save, a test and both limited-user operations share one rule.
- Refuse before anything is written or leaves the backend.
- Close the two-step path through a secret that is temporarily out of use.

**Non-Goals:**

- Fields other than addresses. The user decided this: changing a login, a TLS mode or certificate verification keeps the secret.
- A way to detect an address field that a future provider forgets to declare. That is the accepted cost of the decision above; see Risks.
- Listing every refused secret in one answer. A script gets one per request, as it does today for any other missing required setting.

## Decisions

### Each provider's merge fills a bound secret only while its address is unchanged

The provider already owns its secret policy, including the fixed prefix (`s3/provider.go:81`), and it alone knows which fields are its address. Its merge computes whether the address changed and fills a bound secret only when it did not. Everything else follows from the existing `Validate` call:

| Secret after the merge | Settings use it | Outcome |
|---|---|---|
| empty, address changed | yes | `Validate` answers its existing field error, nothing is saved or sent |
| empty, address changed | no | saved empty, which removes the stored value |
| filled, address unchanged | either | as today, including the existing SFTP and SSH selection guards |

`Validate` already refuses a missing required secret: S3 keys (`s3/provider.go:59`), FTP and NAS passwords, the selected SFTP credential (`sftp/provider.go:83`), the Azure account key in account key mode (`azureblob/provider.go:68`), the SMTP password while a user is set (`email/provider.go:68`), the Telegram token, the Mattermost token in bot mode (`mattermost/provider.go:139`), every database password, the SSH credential of the selected type (`ssh_tunnel_settings.go:51`) and a PostgreSQL client certificate sent without its key (`engines/postgresql/shared/tls.go:38`, reported under `sslMode`). A disabled tunnel passes it (`ssh_tunnel_settings.go:37`), so turning a tunnel off removes the SSH secrets and refuses only a missing database password.

Only a listed webhook header has no such rule (`webhook/provider.go:61`), so its merge refuses a header without a value explicitly with `NewInvalidSettingError` under `headers[<index>].value`, and only when a stored value exists.

The helpers change shape accordingly: `MergeStoredPostgresqlTLS` takes whether the database's address changed, and `MergeStoredSSHTunnel` compares the tunnel's own address itself. Neither returns an error, because `Validate` answers. An exported `IsSSHTunnelAddressChanged` lets each database provider include the tunnel in its own comparison.

Rejected:

- **A shared library that decides each field and reports all refused fields at once.** It would re-implement `Validate`, add an "in use" declaration for about twenty fields and add a package, for a benefit only a script could see.
- **Check only in the test operations.** A member would save the new address and then test the saved resource.
- **A `GetAddress` method on the provider interface with a generic check in the service.** Secrets inside one provider follow different addresses: a database's password follows the host, the port and the tunnel, while its SSH secrets follow only the tunnel.

### Resource addresses per provider

| Provider | Resource address | Bound secrets |
|---|---|---|
| S3 | `Endpoint` | `AccessKey`, `SecretKey` |
| Azure Blob | `Endpoint`, `AccountName` | `AccountKey` |
| FTP, NAS | `Host`, `Port` | `Password` |
| SFTP | `Host`, `Port` | `Password`, `PrivateKey` |
| Email | `SMTPHost`, `SMTPPort` | `SMTPPassword` |
| Telegram | `BotAPIURL`, `IsProxyEnabled`, a `ProxyURL` in the request | `BotToken` |
| Mattermost | `ServerURL` | `BotToken` |
| Webhook | `URL` | each listed header's `Value` |
| Databases | `Host`, `Port`, tunnel address; also `IsSrv` for MongoDB | `Password`, PostgreSQL `ClientKey` |
| SSH tunnel | `IsEnabled`, and `Host` and `Port` while enabled | `Password`, `PrivateKey`, `PrivateKeyPassphrase` |

Unbound, because the secret carries its own address or the service's address is fixed: Discord, Teams and the Mattermost webhook URL, the Azure connection string, the Telegram proxy URL, rclone, Slack (`SlackPostMessageURL` comes from `Deps`), Google Drive and Local.

`AccountName` counts because the default endpoint is built from it (`azureblob/blob_container.go:428`). The Azure form sends no account name on an unrelated edit (`azureblob/provider.go:83`), so its merge fills the stored name first and compares afterwards.

`Bucket` and `Region` of S3 do not count. Minio takes the AWS host from a fixed region map and falls back to a default (`minio-go/v7@v7.2.1/endpoints.go:267`), and it validates the bucket name before a request (`CheckValidBucketName`, `api-put-object.go:395`). So neither can leave the endpoint's service.

A disabled tunnel has no address, so moving its host keeps its secrets. Turning it on always counts as a change, so those secrets cannot reach the moved host. This keeps today's behavior of `databases/secrets_test.go:190`, which edits a disabled tunnel.

Rejected: a fail-closed list of safe fields, where any other field counts as an address. It catches a forgotten field, but the user chose to ask for secrets again only when an address changes.

### Addresses are compared exactly

The comparison is plain equality of the stored and the incoming value, so `Example.com` against `example.com` asks for the secret again.

Stored addresses are in clear (`webhook/settings.go:21`, `mysql/settings_repository.go:72`), so they compare directly. The Telegram `ProxyURL` is stored encrypted, and its presence in a request counts as a change by choice: decrypting to compare would also tell a caller whether a guessed proxy URL matches the stored one. A webhook URL that cannot be decrypted loads empty and compares as changed, which is the safe direction.

Rejected: normalizing hosts and URLs. A normalizer bug would reopen the leak, while exact comparison costs at most one extra prompt.

### A secret out of use is removed, not kept

Without this, a user could park a secret and move it in two requests. For example, they could switch a Mattermost notifier to the webhook mode with an attacker's `ServerURL` and then return to the bot mode at that URL with the address now unchanged. Filling only at an unchanged address gives this removal without extra code.

Rejected: refusing with `400` whenever a secret is left out. That would block a legitimate move, such as to an SMTP server without authentication or an Azure connection string.

### A refusal is `400 invalid_request` with a field error

The contract already has this shape, and the form prevents the request in normal use.

Rejected: a new `ProblemCode`. A form never needs to tell this refusal apart, and a script reads the field in `errors`.

### A viewer is refused in `TestSettings` of notifiers

`Authorize` moves to `ActionEditContent`, the same check storages use. It still comes first, so a stranger keeps getting `404`, and nothing is loaded before the refusal.

Rejected: letting a viewer test settings without a saved notifier's ID. The roles and the two other families would then differ, and a viewer has no edit form to test from: `pages/notifiers/ui/NotifierComponent.tsx:262` already hides editing behind `isCanManageNotifiers`.

### Forms decide with a pure function per family, and the server stays the authority

Each family gets one pure function that compares the draft with the saved settings:

- `features/edit-notifier/lib`, against the acknowledged `baseline` of `useNotifierDraft`;
- `features/edit-storage/lib`, against `savedSettings` of `useStorageDraft` (`useStorageDraft.ts:40`);
- `features/database-connection/lib`, against `draft.savedSettings` (`isConnectionDraftFilled.ts:8`).

The result feeds the existing fill checks (`isNotifierSettingsFilled.ts:14`, `isStorageSettingsFilled.ts:4`, `isConnectionDraftFilled.ts:19`) and replaces `isSaved` wherever `isSaved` decides whether a secret is kept. The storage forms need a separate flag, because `isSaved` also locks the S3 prefix (`EditS3StorageComponent.tsx:171`).

What the user sees follows what each form already has. One new dictionary entry, in all six languages, says the address changed and the secret must be entered again, and it appears as help text under each required empty secret. Beyond that:

- Notifier forms lose `secretKeptPlaceholder`, and the webhook form hides `savedHeadersHidden` (`EditWebhookNotifierComponent.tsx:151`).
- The SFTP form replaces its saved-credential text (`EditSFTPStorageComponent.tsx:94`).
- The S3, Azure, FTP, NAS and database password fields have no kept state today (`ConnectionCredentialFields.tsx:93`), and none is added.
- The SSH tunnel form leaves its stored state (`EditSshTunnelComponent.tsx:115`).
- The PostgreSQL certificate mask stays, because the certificate is not a secret and is still sent, and an empty client key field appears under it (`PostgresqlCertificateFields.tsx:45`). The replace control keeps clearing the certificate, which takes the key out of use.

Rejected:

- **Relying on the server's error alone.** `translateApiError` shows generic text by `code` and names no field (`translateApiError.ts:54`), so the user would not learn what to re-enter.
- **A new operation that tells the form what is required.** This would add a request on every keystroke to avoid one small comparison.
- **A new kept-secret hint on forms that have none.** Nothing in this change needs it.

The address lists exist twice as a result. If the frontend's list goes stale, the server still refuses, and the user sees the generic error.

## Risks / Trade-offs

- [A future provider or address field is not declared, and its secret stays reusable at a new address] → The rule is stated in the `secret-handling` spec and in the provider interface's comment, and the table tests in each family pin every current address field. Nothing detects a new field automatically; this is the accepted cost of the address-only decision.
- [A member can still weaken the transport at the same address, for example by turning off TLS or certificate verification and intercepting] → Accepted by the user's decision and named as out of scope in the proposal.
- [Scripts that change a host without a secret start getting `400`] → The break is marked in the proposal, and the answer names the field.
- [A script that omits several secrets learns of them one request at a time] → Accepted; this matches every other missing required setting.
- [Turning a Telegram proxy off or a tunnel off asks for a secret, although no new party receives it] → Accepted, to keep one exact rule. The prompt costs little.
- [An SMTP user cleared at an unchanged address still gets `400` from `Validate`, because the stored password fills in] → This is existing behavior that this change leaves alone. Only a changed address removes the password.
- [The form cannot tell whether a saved webhook header holds a value, because the value is `writeOnly`] → After a URL change the form requires a value for every saved header, even in the rare case where the stored value is empty. The server would accept such a header without a value, so only the form is stricter.

## Migration Plan

No data migration: stored settings and secrets stay as they are, and the rule applies only to later requests. Rollback is a revert of the change, and it needs no data repair.
