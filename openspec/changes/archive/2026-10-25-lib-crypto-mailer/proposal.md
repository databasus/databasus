## Why

Milestone 4 of the backend rewrite plan moves `users` into a module, but two pieces of shared code stand in the way. The master key lives in `features/encryption/secrets`, and the model of its legacy table lives in `features/users`. That creates the `encryption ↔ users` cycle. Instance mail lives in the feature `features/email`, while its SMTP client `util/smtp_transport` is imported even by `lib/config`. Both have no domain logic. `docs/architecture.md` assigns them to `lib/crypto` and `lib/mailer`, and the `users` module in the last change of this milestone takes them through `Deps`. This change is the first of four; `audit-in-lib`, `users-actor` and `users-module` follow it.

## What Changes

- `lib/crypto` appears with no package-level state. It holds:
  - `MoveLegacyMasterKeyToFile`, the one-time move of a key that an old release kept in the database into the key file;
  - `OpenOrCreateMasterKey`, which loads the key from `secret.key` in the data directory or creates it there once, and checks its length;
  - field encryption in the `enc:` format.
- `internal/app` calls both functions in `app.New`, where the legacy move runs today, so the key is loaded or created at startup instead of on first use. A key file that cannot be read or created, or that holds fewer than 32 bytes, stops the startup with a message that names the file. Today a short key panics at the first encryption.
- `lib/database` gets `Conn(ctx, db)` and `*database.Transactor`, which `app` builds from the pool and services take through `Deps`. Its `Transaction(ctx, fn)` puts the transaction into the context, and a repository query joins it. The legacy key move is their first consumer. `docs/architecture.md` and `backend/AGENTS.md` change from `database.Transaction(ctx, fn)` to the transactor.
- The password generator for read-only and replication users stays in `util/encryption` until `databases` moves. It no longer swallows a `crypto/rand` error by falling back to the first character of the set or by skipping a shuffle step: such an error panics, as the standard library does itself.
- `lib/mailer` appears. It holds the SMTP transport from `util/smtp_transport`, which the email notifier calls with its own settings. It also holds the instance sender from `features/email`, built with `mailer.NewInstanceSender(Deps)` from the operator's settings. `lib/config` validates the SMTP settings through `lib/mailer` instead of the old `util` package.
- The old code reaches both packages through bridges, the same way it reaches `config.GetEnv()`. `util/encryption` keeps the `FieldEncryptor` interface, the password generator and the getters `GetFieldEncryptor` and `GetMasterKey`, which return what `app` installed. `features/email` keeps only the getter, renamed to `GetInstanceSender`. In a test build with the `testhelpers` tag, a getter that `app` did not fill builds its value itself. The storage probe, which runs without `app.New`, stops calling the encryptor getter.
- The callers of the master key in `backups`, `restores` and `users` switch from `*secrets.SecretKeyService` to the master key of `lib/crypto`. The callers of SMTP types switch their imports to `lib/mailer`. The 242 references to the `FieldEncryptor` interface do not change.
- `lib/config` takes the `EnvMode` type from `util/env`, its last import of old code, and `util/env` is deleted.
- Deleted: `features/encryption/secrets`, `util/smtp_transport`, the model `users_models.SecretKey`, the body of `features/email` and the field encryptor of `util/encryption`. Their tests move with the code and are renamed per the testing skill.
- **BREAKING** for the schema: the table `secret_keys` is renamed to `lib_crypto_secret_keys`. Only `lib/crypto` reads it, once at startup, to move a key left by an old release into the file.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `secret-handling`: the master key is loaded or created when the backend starts. A key left in the database by an old release is moved into the file first. A key file that cannot be read or created, or that is shorter than 32 bytes, stops the startup.

## Out of scope

- `lib/audit`, `lib/clock` and `background.Runner`: the change `audit-in-lib`.
- The `users` module, `Actor`, the JWT tokens and the move of the HTTP operations of `users` to the contract: the changes `users-actor` and `users-module`. JWT tokens stay signed with the master key.
- Stream encryption of backup files. It imports no feature code, so it creates no cycle, and it moves into `lib/crypto` with `backups` in milestone 9.
- Moving the password generator. It moves with `databases` in milestone 8.
- Changing the field encryption format, the stream encryption format, the key derivation or the key file location. Data encrypted by earlier releases decrypts unchanged.
- Moving the decryption of secrets out of the providers' `AfterFind` hooks: milestones 6–8, together with the providers.
- Moving the mail templates of invitations, sign-in codes and password resets out of `users` and `workspaces`. They stay where the messages are built.
- `lib/ratelimit`, `lib/cache` and the Cloudflare Turnstile client: the change `users-module`.

## Impact

- Code: new `backend/internal/lib/crypto` and `backend/internal/lib/mailer`; `backend/internal/lib/database` (`Conn`, `Transaction`); `backend/internal/app` (wiring, the storage probe, `getters_test.go`); `backend/internal/lib/config`; `backend/internal/util/encryption` and `backend/internal/features/email` become bridges; deleted `backend/internal/features/encryption/secrets` and `backend/internal/util/smtp_transport`. The callers: about 15 files in `features/backups` and `features/restores` that read the master key, `features/users/services`, `features/workspaces/services`, `features/notifiers` and `cmd/seed_dev`.
- Migrations: one new migration renames `secret_keys`.
- Running instances: the key file and every encrypted value stay as they are. The rename only changes metadata. A test of `internal/app` covers the move of a key left in the renamed table into the file.
- Documents: `docs/architecture.md` and `backend/AGENTS.md` (how a service opens a transaction), `temp/refactor/plan.md` (the milestone 4 checklist and the arrival of `database.Conn` and the transactor).
- Rules the change answers to: the root `AGENTS.md` and `backend/AGENTS.md`.
