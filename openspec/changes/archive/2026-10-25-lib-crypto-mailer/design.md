## Context

Why the change is needed is in proposal.md, section Why. This document covers only what shapes the approach.

**The master key.** `SecretKeyService` (`internal/features/encryption/secrets/service.go`) has two jobs:

- `MigrateKeyFromDbToFileIfExist` (`service.go:23`) moves a key that an old release kept in the `secret_keys` table into the file. If the table holds a row, it writes the file even when one exists (`service.go:43`) and then deletes the row. `app.New` calls it after `old_config.Install` and `storage.Install` and before the old code is wired (`internal/app/app.go:79`).
- `GetSecretKey` (`service.go:54`) loads the key lazily under a process mutex. If the file is missing, it generates two UUIDs and writes them with `os.WriteFile` (`service.go:71`). Two processes that start at once can each write their own key, and the one that writes first loses its key.

The service reads the table through `storage.GetDb()` and the model `users_models.SecretKey` (`internal/features/users/models/secret_key.go`). That model is the `encryption ↔ users` cycle in `temp/refactor/backend-analysis.md`. Both the field encryption (`internal/util/encryption/secret_key_field_encryptor.go:35`, `:90`) and the stream encryption take the first 32 bytes of the key, `[]byte(masterKey)[:32]`. A shorter key panics at the first encryption.

**Who reads the key.** The `FieldEncryptor` interface (`internal/util/encryption/field_encryptor.go`) is referenced 242 times in production code, from 22 packages through `GetFieldEncryptor()`. The notifiers' test helper `PassthroughEncryptor` (`internal/features/notifiers/models/testing.go`) implements it. The raw key is read through `GetSecretKey()` by about 15 files:

- the logical backup use cases of every engine, for example `usecases/logical/postgresql/create_backup_uc.go:449`;
- the restore use cases, for example `restores/usecases/postgresql/restore_backup_uc.go:225`;
- the physical backuper and the WAL supervisor (`backuping/physical/backuper.go:313`, `wal_supervisor.go:360`);
- `backups/services/logical_service.go:312` and `physical_service.go:632`;
- `users/services/user_services.go:216` and `:275`, which sign and check JWT tokens.

The backup and restore files pass the key to the stream encryption in `internal/features/backups/backups/encryption`. That package imports nothing from the features, so it is not part of any cycle.

**The storage probe.** `ProbeStorage` (`internal/app/storage_probe.go:27`) runs at container start before the internal database exists, without `app.New`. It calls `encryption.GetFieldEncryptor()` (`storage_probe.go:46`) and hands the result to local storage. Local storage takes the encryptor as a parameter but never calls it (`internal/features/storages/models/local/model.go`), because it stores no secret.

**The password generator.** `GenerateComplexPassword` (`internal/util/encryption/password_generator.go:14`) builds the passwords of read-only and replication users. Five engine models call it, for example `databases/databases/postgresql/logical/model.go:503`. On a `crypto/rand` error it returns the first character of the set (`password_generator.go:45`) or skips a swap of the shuffle (`:54`). `users` does not use it.

**Mail.** `EmailSMTPSender` (`internal/features/email/email.go`) builds a connection from `config.GetEnv()` in `di.go` and sends through `util/smtp_transport`. `users/services/di.go:20`, `:33` and `workspaces/services/di.go:42` take it through `GetEmailSMTPSender()` and declare their own `EmailSender` interfaces with `IsConfigured` and `SendEmail`. `smtp_transport` is imported by:

- the email notifier, which calls `Send` with the user's settings (`notifiers/models/email_notifier`);
- `users/services/settings_service.go:237`, which parses the test email recipient;
- `features/email`;
- the tests of `features/notifiers`;
- `lib/config`, for the `Security` type and the validation of `SMTP_SECURITY`, `SMTP_HELO_NAME` and `SMTP_FROM` (`internal/lib/config/config.go:18`, `:218`). So a `lib` package depends on old code.

**Bridges already in use.** `internal/storage/storage.go` shows the pattern: `Install` stores what `app` built, the getter returns it, and `loadWithoutApp` either panics (production) or builds the value itself (a build with the `testhelpers` tag). `internal/app/getters_test.go:219` calls every old getter in a test build.

**Rules in force.** `internal/lib` is in the target layout. Code that lands there passes every linter and `archlint` at once, its tests included: no package-level dependencies, `*gorm.DB` only in a `*_repository.go`, errors in `errors.go`, logs through the `...Context` methods with attributes only. `lib` imports no `internal/features` package. Names follow "Functions, methods and types" in the root `AGENTS.md`: a name states everything the unit does, and a type name does not stutter. The change answers to the root `AGENTS.md` and `backend/AGENTS.md`.

## Goals / Non-Goals

**Goals:**

- `lib/crypto` and `lib/mailer` hold the code and the tests listed in the proposal, built in `internal/app`.
- The old code changes only where it reads the raw key or imports SMTP types. The 242 references to `FieldEncryptor` stay.
- No `lib` package imports `internal/util` or `internal/features`.

**Non-Goals:**

- Moving the stream encryption or the password generator. Neither is part of a cycle, and `users` uses neither. They move with `backups` and `databases`, whose code changes their callers anyway.
- Removing the bridges. Each one goes away when the last old caller moves into a module.

## Decisions

### 1. The master key is loaded once in `app.New` by two functions, and its value is a plain string afterwards

`app.New` calls two functions at the place where `MigrateKeyFromDbToFileIfExist` runs today: after the migrations, `old_config.Install` and `storage.Install`, and before the bridges of decision 4 are installed and the old code is wired.

1. `crypto.MoveLegacyMasterKeyToFile(ctx, deps)` moves a key left in `lib_crypto_secret_keys` into the file and deletes the row. It does nothing when the table is empty.
2. `crypto.OpenOrCreateMasterKey(path)` reads the file, or creates it when it is missing, and checks that the key holds at least 32 bytes.

The second function returns a `*crypto.MasterKey` whose `GetSecret()` returns the string without an error. `crypto.NewFieldEncryptor(masterKey)` and the old stream encryption callers take it. Any failure fails `app.New` with an error that names the file. The console commands go through `app.New` too, so they refuse an unusable key the same way.

*Rejected: one function `OpenMasterKey` that moves, creates and checks.* Its name would not say that it also moves a row out of the database, which the naming rule forbids, and the legacy move needs the database while the open does not. *Rejected: keep the lazy loading under a mutex.* The error then surfaces in the middle of a backup or a request instead of at startup. Every reader also has to handle an error that only the first call can produce, which is 15 `if err != nil` branches around a value that never changes.

### 2. The key file is created exclusively, and the legacy move overrides it

When the file is missing, `OpenOrCreateMasterKey` writes the new key into a temporary file in the same directory and links it into place with `os.Link`, which fails when the target already exists. On that failure it reads the file the other process wrote and uses that key. The key keeps its current form, two UUIDs, so its length stays 72 bytes.

The legacy move keeps today's behavior: a row with an empty secret counts as no key and is left alone, and a row with a key is written into the file even when the file exists, and it does not read or check the file it replaces. The table holds a row only on an installation whose release never wrote a key file, so a file next to such a row is not a key in use. The 32-byte check and the refusal to replace an unusable file apply only after the move, to the file `OpenOrCreateMasterKey` reads. The `secret-handling` delta states both rules in that order.

*Rejected: `O_CREATE|O_EXCL` on the target with a direct write.* A crash between the creation and the write leaves an empty key file, which the next start refuses as shorter than 32 bytes. Linking a fully written temporary file never exposes a partial key.

### 3. `database.Conn` and `database.Transactor` arrive here, and the legacy table gets a repository

The legacy move needs a query, and `docs/architecture.md` lets only a repository hold `*gorm.DB`, running each query through `database.Conn(ctx, r.db)`. `lib/database` gets both halves of the mechanism now:

- `database.NewTransactor(db)` returns a `*database.Transactor`, built in `internal/app` from the pool and handed to services through `Deps`. Its method `Transaction(ctx, fn)` begins a transaction on that pool and puts it into the context that `fn` receives.
- `Conn(ctx, db)` returns the transaction from the context when present and `db.WithContext(ctx)` otherwise.

A service therefore opens a transaction without holding `*gorm.DB`, which `forbidigo` allows only in repositories, `lib` and `app`. `docs/architecture.md` ("Transactions travel in the context") and `backend/AGENTS.md` say `database.Transaction(ctx, fn)` today. A package-level function cannot know the pool without a package-level variable, which the architecture forbids, so both documents change to the transactor from `Deps` in this change.

*Rejected: `database.Transaction(ctx, db, fn)` with an explicit pool.* Only `lib` and `app` may hold the pool, so a module service could not call it.

The legacy move is their first consumer: `MoveLegacyMasterKeyToFile` takes the transactor and the repository through its `Deps`. Inside one transaction it locks the row with `SELECT ... FOR UPDATE`, writes the file and deletes the row. If the commit fails after the file is written, the row stays and the next start writes the same key again, so the step can repeat safely.

*Rejected: a plain `db.WithContext(ctx)` in the repository until `audit-in-lib` (4b).* That writes the one repository of `lib` against the rule and rewrites it one change later. *Rejected: introduce only `Conn` now.* Conn has to know the context key that the transactor sets. Splitting one small file across two changes gains nothing.

The plan placed both functions in the same milestone, with `users` and `lib/audit` as the first consumers. This change moves them forward by one change, and `temp/refactor/plan.md` records it.

### 4. The old code gets bridges, and `FieldEncryptor` stays in the old package

`util/encryption` keeps the interface `FieldEncryptor` and the password generator, and adds `Install(masterKey, fieldEncryptor)`, `GetFieldEncryptor()` and `GetMasterKey()`. `features/email` keeps `Install(instanceSender)` and the getter, renamed to `GetInstanceSender()` after the type it returns. Its callers in `users/services/di.go`, `workspaces/services/di.go` and the old tests change with it. `app.New` installs both bridges after `OpenOrCreateMasterKey` returns and before `setUpDependencies`, as decision 1 orders the steps. In a test build, `loadWithoutApp` opens the key from `config.GetEnv().SecretKeyPath` and builds the sender from `config.GetEnv()`. It does not do the legacy move: a test database never holds a legacy key.

The storage probe passes no encryptor to local storage, a nil `FieldEncryptor`, instead of calling the bridge getter. Local storage stores no secret and never calls the encryptor, so the probe keeps working without `app.New`, and it does not create a key file before the legacy move had its chance. The existing probe tests in `internal/app` and `cmd/databasus` run the probe without `app.New` and cover it.

*Rejected: move the interface to `lib/crypto` and change 242 references.* By the module rules an interface belongs to its consumer. Each module that moves declares the interface it needs, or takes `*crypto.FieldEncryptor`, so the shared interface would be thrown away anyway. Moving it now adds a 22-package diff with nothing to review.

*Rejected: give every old caller the `lib` object without a bridge.* The old objects are built lazily in `di.go` and cannot receive `Deps`, which is the reason the skeleton introduced bridges.

*Rejected: let the probe open the key itself.* On an installation whose key is still in the database, the probe would create a new key file before `app.New` moves the real one. The legacy move would then overwrite it, but the probe would have encrypted nothing with a correct key.

### 5. `lib/mailer` holds both the transport and the instance sender

The package has two parts:

- **The transport.** `mailer.Send(ctx, connection, message)` with the `Connection`, `Message` and `Security` types and the address and greeting helpers, moved from `util/smtp_transport` with their behavior. It has no state, so the email notifier keeps calling it with its own settings.
- **The instance sender.** `mailer.NewInstanceSender(mailer.Deps{Settings, Logger})` returns a `*mailer.InstanceSender` with `IsConfigured()` and `SendEmail(ctx, to, subject, htmlBody)`. `Settings` holds the operator's values: host, port, security, user, password, sender, greeting name and the skip-verify flag. `config.Config.GetInstanceMailSettings()` maps the SMTP variables to `Settings` and fills the default security and greeting name the way `features/email/di.go` does today. `lib/config` already imports `lib/mailer` to validate those variables, so `app` and the test-build bridge of decision 4 share one mapping instead of two copies.

The type is named after what sets it apart from `mailer.Send`: it sends the instance's own mail with the operator's server. The method names stay `IsConfigured` and `SendEmail`, so the interfaces of `users` and `workspaces` and their test fakes do not change. `lib/config` validates the SMTP settings through `mailer.ParseSender`, `mailer.ValidateHeloName` and `Security.IsValid`.

The tests of the instance sender run against the fake SMTP server the transport tests already use, so `lib/mailer` imports no container helper of the old code.

*Rejected: separate `lib/smtp` and `lib/mailer` packages.* The outgoing-email spec requires one delivery behavior for instance mail and notification channels. Two packages would invite the two paths to drift apart, and only the sender would import the transport.

*Rejected: let the notifier send through the instance sender.* A notifier carries the user's server, not the operator's, so it needs the stateless function.

### 6. Moved code follows the target rules when it moves, and the generator stops degrading in place

The field encryptor, the transport, the instance sender and their tests land in the target layout and must pass every linter at once. They get errors in `errors.go` and test names per the testing skill. Their behavior does not change, and the tests that pin it move with them. Among them is `Test_Decrypt_LegacyDeterministicNonceCiphertext_StillDecrypts`, which proves that values from older releases decrypt.

The password generator keeps its signature and its place. Its two fallbacks are deleted, and an error from `crypto/rand` panics. Since Go 1.24, a read from `crypto/rand.Reader` does not return an error, and a failure ends the program instead, so the panic is unreachable today. If the reader ever changed, a panic would stop the operation instead of producing a weaker password. The five callers do not change.

*Rejected: give the generator an error return.* It would change five engine models for an error path that never runs. *Rejected: move the generator into `lib/crypto` now.* Its callers are the engine models that milestone 8 moves, and moving it twice adds churn for nothing.

## Risks / Trade-offs

- [The 32-byte check refuses an installation that runs today with a shorter hand-written key] → Such a key panics at the first encryption today, so the instance cannot hold an encrypted secret. Refusing to start with the file's name is the better failure. The release notes mention it.
- [Old test packages run in parallel processes and share a configuration directory] → The exclusive link from decision 2 gives them one key, where today two processes can each write a different one.
- [The storage probe passes a nil encryptor] → A future change that makes local storage encrypt something would panic in the probe. Local storage moves into the `storages` module in milestone 7, where the probe gets its encryptor through `Deps`.
- [Moving tests into the target layout turns on test linting for them] → The change budgets the cleanup of the moved tests. No old test outside the moved packages is touched beyond imports.

## Migration Plan

One migration renames `secret_keys` to `lib_crypto_secret_keys` and renames its index `secret_keys_secret_idx` to `lib_crypto_secret_keys_secret_idx`. Both are metadata changes. On the next start, `MoveLegacyMasterKeyToFile` finds a legacy key, if any, in the renamed table. An upgrade is one-way, as `docs/architecture.md` states: an older binary refuses to start on the new schema before it reads anything.
