---
name: testing
description: Write, change, run and review tests in Databasus. Covers how to name a test, which layer it belongs to, how to prepare data through the API, when a unit test or a fake is allowed, how to wait for a condition, and which tests to run before handing work over. Use when writing, changing, running or reviewing tests, when fixing a bug, and when choosing which test commands a change runs.
---

# Testing

[`docs/testing.md`](../../../docs/testing.md) describes the kinds of tests, how they are isolated and how runs work. This skill says how to write one test.

## The main rules

- Test what a caller observes: an HTTP operation, an engine against a real server, or a screen. Internal functions are covered through that surface.
- Never mock our own database, modules or services. A module's test builds the real modules it depends on.
- A bug fix starts with a test that reproduces the bug. Run it on the unfixed code and see it fail for the reason the bug describes, then fix the code and see it pass.
- Run every test you wrote or changed and see it pass. A package that only compiled has not passed; say so in the report.

## Names

A test name has two parts: what is tested and the scenario.

- The Go function names what is tested, in Go style: `TestCreateStorage`, `TestRestoreBackup`.
- Each scenario is a sentence in `t.Run`, even when the function has only one scenario. The sentence is lowercase, has no period and uses the present tense. It states the result first, then the condition, which starts with `when`, `with`, `without`, `after` or `before`.

```go
func TestDeleteStorage(t *testing.T) {
	t.Parallel()

	t.Run("deletes storage when no backup uses it", func(t *testing.T) {
		t.Parallel()
		// ...
	})

	t.Run("returns 409 when a backup is stored in it", func(t *testing.T) {
		t.Parallel()
		// ...
	})
}
```

A table test puts the sentence into the `name` field. In Vitest, `describe` names the component or page and `it` holds the sentence. Variables inside a test are named after the domain, such as `createdStorage` or `secondMember`. Never name them `got`, `want`, `result` or `data`. `expectedStatus` in a case table is fine.

## Module API tests

A module's tests live in its root directory in package `<module>_test` and reach the module only through its HTTP operations.

```go
func TestGetStorage(t *testing.T) {
	t.Parallel()

	t.Run("returns 404 when storage belongs to another workspace", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t)
		owner := app.SignUp(t)
		stranger := app.SignUp(t)
		storage := owner.CreateLocalStorage(t, owner.CreateWorkspace(t))

		response, err := stranger.Client.GetStorageWithResponse(t.Context(), storage.Id)

		require.NoError(t, err)
		require.Equal(t, http.StatusNotFound, response.StatusCode())
	})
}
```

- Call operations through the client from `apptest`, which checks every response against the contract. Use a raw `http.Client` only for a request the contract cannot describe, such as malformed JSON.
- Create data through the API. Direct SQL is allowed only for a state the API cannot create, for data the API does not return, and to clean up outside the test's own database.
- Extract a helper as soon as a second test repeats the same setup.
- Cover every operation for these cases:
  - the successful case;
  - invalid input;
  - a request without a session;
  - a request from a member of another workspace, or from a member whose role is too low;
  - conflicts of state, such as deleting a storage that holds backups or creating a duplicate.
- A background job does not run on its own in the test application. Run its pass explicitly, and move the clock instead of waiting for the schedule.

## Unit tests and fakes

Write a unit test without the application only for pure logic that has no HTTP entry point or is too slow to enumerate through HTTP. [`docs/testing.md`](../../../docs/testing.md#layers) lists examples.

A fake replaces only:

- an external system we cannot run locally, behind an interface in `Deps`, such as a third-party HTTP API;
- a failure that is unsafe to cause for real, such as a full disk or a connection cut in the middle of an upload.

Test a client of an external system against a real local substitute where one exists. Use Mailpit for SMTP, MinIO, Azurite and SFTP containers for storages, and an `httptest.Server` for HTTP APIs.

Write fakes as small hand-written structs. Do not use mock generators, `testify/mock` or `testify/suite`. Use `require` for what the rest of the test depends on and `assert` for independent fields.

## Waiting

- Never wait for a fixed time. Wait for the condition with `testwait.Until`, up to the test's overall timeout. The linter rejects `time.Sleep` and `Eventually`.
- For a negative check, such as "no notification was sent", first wait for a sign that the action was processed. Then watch the check's window.
- When the behavior depends on time, move the test clock.

## Isolation

- Call `testdb.Open(t)` once per test. A helper that opens one database for several subtests makes that database shared again.
- On a shared engine server, give every object a unique name: database, role, schema and replication slot.
- Take ports from `127.0.0.1:0` and files from `t.TempDir()`. Remove every container or process the test starts in `t.Cleanup`.
- Mark independent tests with `t.Parallel()`.

## Which tests to run

While working, run the package or the single test you need. Before handing the work over, run `just test-affected` and include its output in the report. Run the full `just test` when the user asks for it. When a change needs a full run, `just test-affected` says so and runs it itself.

After you interrupt a run, remove only what that run left behind. Never prune Docker or stop the shared test PostgreSQL server.

## Before handing the work over

- Each test names what it tests and describes its scenario in a sentence.
- New behavior is covered at the right layer.
- Data is prepared through the API. Direct SQL appears only where nothing else works.
- Nothing of ours is mocked, and no test sleeps.
- The tests you touched ran and passed, and the report shows the commands.
