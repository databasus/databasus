package storages_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/lib/testwait"
	"databasus-backend/internal/modules/storages"
)

func TestDeleteStorage(t *testing.T) {
	t.Parallel()

	t.Run("returns 204 and no longer lists the storage when nothing references it", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		createdStorage := tm.createLocalStorage(t, "Local")

		response := deleteStorage(t, tm.owner, createdStorage.Id)

		require.Equal(t, http.StatusNoContent, response.StatusCode(), string(response.Body))
		assert.Empty(t, listStorages(t, tm.owner, tm.workspace.Id))
	})

	t.Run("returns 409 in_use and keeps the storage when a logical backup config uses it", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		createdStorage := tm.createLocalStorage(t, "Local")
		tm.app.InsertBackupConfigUsingStorage(t, apptest.StorageUse{
			DatabaseID: tm.app.InsertDatabase(t, tm.workspace.Id),
			StorageID:  createdStorage.Id,
		})

		response := deleteStorage(t, tm.owner, createdStorage.Id)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusConflict, apiclient.ProblemCodeInUse)
		abandonResponse, abandonErr := tm.owner.Client.DeleteStorageWithResponse(t.Context(), createdStorage.Id,
			&apiclient.DeleteStorageParams{ShouldAbandonCleanup: new(true)})
		require.NoError(t, abandonErr)
		requireProblem(
			t,
			abandonResponse.StatusCode(),
			abandonResponse.Body,
			http.StatusConflict,
			apiclient.ProblemCodeInUse,
		)
		assert.Equal(t, []uuid.UUID{createdStorage.Id}, getListedStorageIDs(t, tm.owner, tm.workspace.Id))
	})

	t.Run("returns 409 in_use and keeps the storage when a physical backup config uses it", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		createdStorage := tm.createLocalStorage(t, "Local")
		databaseID := tm.app.InsertDatabase(t, tm.workspace.Id)
		tm.app.InsertPhysicalBackupConfigUsingStorage(t, apptest.StorageUse{
			DatabaseID: databaseID,
			StorageID:  createdStorage.Id,
		})

		response := deleteStorage(t, tm.owner, createdStorage.Id)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusConflict, apiclient.ProblemCodeInUse)
		abandonResponse, abandonErr := tm.owner.Client.DeleteStorageWithResponse(t.Context(), createdStorage.Id,
			&apiclient.DeleteStorageParams{ShouldAbandonCleanup: new(true)})
		require.NoError(t, abandonErr)
		requireProblem(
			t,
			abandonResponse.StatusCode(),
			abandonResponse.Body,
			http.StatusConflict,
			apiclient.ProblemCodeInUse,
		)
		assert.Equal(t, []uuid.UUID{createdStorage.Id}, getListedStorageIDs(t, tm.owner, tm.workspace.Id))
	})

	t.Run("returns 409 in_use and removes no file when backups are still stored in it", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		createdStorage := tm.createLocalStorage(t, "Local")
		pendingFileName := "pending-" + uuid.NewString()
		tm.app.WriteUnpublishedFile(t, createdStorage.Id, pendingFileName)
		backupID := insertCompletedBackup(t, tm, tm.app.InsertDatabase(t, tm.workspace.Id), createdStorage.Id)

		response := deleteStorage(t, tm.owner, createdStorage.Id)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusConflict, apiclient.ProblemCodeInUse)
		abandonResponse, abandonErr := tm.owner.Client.DeleteStorageWithResponse(t.Context(), createdStorage.Id,
			&apiclient.DeleteStorageParams{ShouldAbandonCleanup: new(true)})
		require.NoError(t, abandonErr)
		requireProblem(
			t,
			abandonResponse.StatusCode(),
			abandonResponse.Body,
			http.StatusConflict,
			apiclient.ProblemCodeInUse,
		)
		assert.Equal(t, []uuid.UUID{createdStorage.Id}, getListedStorageIDs(t, tm.owner, tm.workspace.Id))
		assert.True(t, tm.app.IsLocalStorageFilePresent(t, pendingFileName))
		assert.Equal(t, 1, countPendingDeletions(t, tm.app, createdStorage.Id))
		assert.Equal(t, 1, countRows(t, tm.app, "SELECT COUNT(*) FROM backups_logical WHERE id = ?", backupID))
	})

	t.Run("refuses pending cleanup and abandons it only after explicit acknowledgement", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		createdStorage := tm.createLocalStorage(t, "Local")
		pendingFileName := "pending-" + uuid.NewString()
		tm.app.WriteUnpublishedFile(t, createdStorage.Id, pendingFileName)
		require.True(t, tm.app.IsLocalStorageFilePresent(t, pendingFileName))

		response := deleteStorage(t, tm.owner, createdStorage.Id)
		requireProblem(
			t,
			response.StatusCode(),
			response.Body,
			http.StatusConflict,
			apiclient.ProblemCodeCleanupPending,
		)
		assert.True(t, tm.app.IsLocalStorageFilePresent(t, pendingFileName))
		assert.Equal(t, 1, countPendingDeletions(t, tm.app, createdStorage.Id))

		abandoned, err := tm.owner.Client.DeleteStorageWithResponse(t.Context(), createdStorage.Id,
			&apiclient.DeleteStorageParams{ShouldAbandonCleanup: new(true)})
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, abandoned.StatusCode(), "%s", abandoned.Body)
		assert.True(t, tm.app.IsLocalStorageFilePresent(t, pendingFileName))
		assert.Zero(t, countPendingDeletions(t, tm.app, createdStorage.Id))
	})

	t.Run("deletes the storage and names the files left behind when its provider is unreachable",
		func(t *testing.T) {
			t.Parallel()

			tm := newTeam(t)
			storageID := insertStoredS3Storage(t, tm, storedS3Storage{
				bucket:             minioBucket{name: "unreachable", endpointURL: "http://127.0.0.1:1", client: nil},
				encryptedAccessKey: tm.app.EncryptSecret(t, "access"),
				encryptedSecretKey: tm.app.EncryptSecret(t, "secret"),
			})
			leftBehindFileName := "unreachable-" + uuid.NewString()
			tm.app.ExecuteSQL(t, "INSERT INTO storages_pending_file_deletions (storage_id, file_name) VALUES (?, ?)",
				storageID, leftBehindFileName)

			response, err := tm.owner.Client.DeleteStorageWithResponse(t.Context(), storageID,
				&apiclient.DeleteStorageParams{ShouldAbandonCleanup: new(true)})
			require.NoError(t, err)

			require.Equal(t, http.StatusNoContent, response.StatusCode(), string(response.Body))
			assert.Empty(t, listStorages(t, tm.owner, tm.workspace.Id))
			assert.Contains(t, listAuditMessages(t, tm, tm.workspace.Id),
				"Storage deleted: Offsite; automatic cleanup abandoned for 1 outstanding file(s): "+leftBehindFileName)
		})

	t.Run("deletes the storage without removing a file whose write is still running and names it",
		func(t *testing.T) {
			t.Parallel()

			tm := newTeam(t)
			createdStorage := tm.createLocalStorage(t, "Local")
			runningFileName := "running-" + uuid.NewString()
			contentReader, contentWriter := io.Pipe()
			writeResult := make(chan error, 1)

			go func() {
				writeResult <- tm.app.WriteStorageFile(context.WithoutCancel(t.Context()),
					storages.StoredFileReference{StorageID: createdStorage.Id, FileName: runningFileName},
					contentReader)
			}()

			testwait.Until(t, "the running write registers its file", func(context.Context) error {
				if countPendingDeletions(t, tm.app, createdStorage.Id) == 0 {
					return fmt.Errorf("no pending deletion of %s yet", runningFileName)
				}

				if !tm.app.IsLocalStorageStagingFilePresent(t, runningFileName) {
					return fmt.Errorf("no staging file of %s yet", runningFileName)
				}

				return nil
			})

			response := deleteStorage(t, tm.owner, createdStorage.Id)
			requireProblem(
				t,
				response.StatusCode(),
				response.Body,
				http.StatusConflict,
				apiclient.ProblemCodeCleanupPending,
			)
			response, err := tm.owner.Client.DeleteStorageWithResponse(t.Context(), createdStorage.Id,
				&apiclient.DeleteStorageParams{ShouldAbandonCleanup: new(true)})
			require.NoError(t, err)

			require.Equal(t, http.StatusNoContent, response.StatusCode(), string(response.Body))
			assert.Contains(t, listAuditMessages(t, tm, tm.workspace.Id),
				"Storage deleted: Local; automatic cleanup abandoned for 1 outstanding file(s): "+runningFileName)
			assert.True(t, tm.app.IsLocalStorageStagingFilePresent(t, runningFileName),
				"the deletion must leave the file of the running write in place")

			require.NoError(t, contentWriter.Close())
			assert.Error(t, <-writeResult, "a write whose storage was deleted must not hand out a receipt")
		})
}

func TestDeleteStorageAccess(t *testing.T) {
	t.Parallel()

	t.Run("lets a workspace member delete a storage", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		member := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER)
		createdStorage := tm.createLocalStorage(t, "Local")

		response := deleteStorage(t, member, createdStorage.Id)

		require.Equal(t, http.StatusNoContent, response.StatusCode(), string(response.Body))
		assert.Empty(t, listStorages(t, tm.owner, tm.workspace.Id))
	})

	t.Run("returns 403 permission_denied and keeps the storage when a viewer deletes it", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		viewer := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)
		createdStorage := tm.createLocalStorage(t, "Local")

		response := deleteStorage(t, viewer, createdStorage.Id)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusForbidden,
			apiclient.ProblemCodePermissionDenied)
		assert.Equal(t, []uuid.UUID{createdStorage.Id}, getListedStorageIDs(t, tm.owner, tm.workspace.Id))
	})

	t.Run("returns 404 not_found and keeps the storage when a stranger deletes another team's storage",
		func(t *testing.T) {
			t.Parallel()

			tm := newTeam(t)
			stranger := tm.stranger(t)
			createdStorage := tm.createLocalStorage(t, "Local")

			response := deleteStorage(t, stranger, createdStorage.Id)

			requireProblem(t, response.StatusCode(), response.Body, http.StatusNotFound,
				apiclient.ProblemCodeNotFound)
			assert.Equal(t, []uuid.UUID{createdStorage.Id}, getListedStorageIDs(t, tm.owner, tm.workspace.Id))
		})

	t.Run("returns 401 unauthenticated and keeps the storage without a session", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		createdStorage := tm.createLocalStorage(t, "Local")

		response, err := tm.app.NewClient(t).DeleteStorageWithResponse(t.Context(), createdStorage.Id, nil)

		require.NoError(t, err)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusUnauthorized,
			apiclient.ProblemCodeUnauthenticated)
		assert.Equal(t, []uuid.UUID{createdStorage.Id}, getListedStorageIDs(t, tm.owner, tm.workspace.Id))
	})
}

func deleteStorage(t *testing.T, session apptest.Session, storageID uuid.UUID) *apiclient.DeleteStorageResponse {
	t.Helper()

	response, err := session.Client.DeleteStorageWithResponse(t.Context(), storageID, nil)
	require.NoError(t, err)

	return response
}

// insertCompletedBackup writes the backup record directly, because a completed backup through the
// contract needs a reachable database.
func insertCompletedBackup(t *testing.T, tm team, databaseID, storageID uuid.UUID) uuid.UUID {
	t.Helper()

	backupID := uuid.New()
	tm.app.ExecuteSQL(t, `INSERT INTO backups_logical (id, database_id, storage_id, status, file_name)
		VALUES (?, ?, ?, 'COMPLETED', ?)`, backupID, databaseID, storageID, "backup-"+backupID.String())

	return backupID
}

func countPendingDeletions(t *testing.T, app *apptest.App, storageID uuid.UUID) int {
	t.Helper()

	return countRows(t, app, "SELECT COUNT(*) FROM storages_pending_file_deletions WHERE storage_id = ?", storageID)
}

func countRows(t *testing.T, app *apptest.App, query string, values ...any) int {
	t.Helper()

	var rowCounts []int
	app.QuerySQL(t, &rowCounts, query, values...)
	require.Len(t, rowCounts, 1)

	return rowCounts[0]
}

func listAuditMessages(t *testing.T, tm team, workspaceID uuid.UUID) []string {
	t.Helper()

	auditEvents := apptest.ListAllAuditEvents(t, tm.admin, apiclient.ListAuditEventsParams{
		WorkspaceId:  &workspaceID,
		AuthorUserId: nil,
		Limit:        nil,
		After:        nil,
	})

	messages := make([]string, 0, len(auditEvents))
	for _, auditEvent := range auditEvents {
		messages = append(messages, strings.TrimSpace(auditEvent.Message))
	}

	return messages
}

func TestAbandonStorageCleanup(t *testing.T) {
	t.Parallel()

	t.Run("makes no provider requests when refusing or abandoning cleanup", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		var providerRequests atomic.Int64
		provider := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
			providerRequests.Add(1)
			http.Error(response, "provider unavailable", http.StatusServiceUnavailable)
		}))
		t.Cleanup(provider.Close)
		createdStorage := createStorage(t, tm.owner, apiclient.StorageCreate{
			WorkspaceId: tm.workspace.Id,
			Name:        "Unavailable archive",
			Settings: newS3Settings(t, apiclient.S3StorageSettings{
				S3Bucket: "unavailable", S3Endpoint: new(provider.URL),
				S3AccessKey: new("access"), S3SecretKey: new("secret"),
			}),
		})
		storageID := createdStorage.Id
		tm.app.RequestStorageFileDeletions(t, storages.FileDeletionRequest{
			References: []storages.StoredFileReference{{StorageID: storageID, FileName: "pending.dump"}},
			Scope:      nil,
		})

		refused := deleteStorage(t, tm.owner, storageID)

		requireProblem(t, refused.StatusCode(), refused.Body, http.StatusConflict, apiclient.ProblemCodeCleanupPending)
		assert.Zero(t, providerRequests.Load())

		abandoned := abandonStorageCleanup(t, tm.owner.Client, storageID)

		require.Equal(t, http.StatusNoContent, abandoned.StatusCode(), "%s", abandoned.Body)
		assert.Zero(t, providerRequests.Load())
		assert.Zero(t, countPendingDeletions(t, tm.app, storageID))
	})

	t.Run("restores credentials and every obligation when the abandonment audit fails", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		namespace := apptest.CreateStorageNamespace(t, tm.owner, tm.workspace.Id)
		tm.app.WriteUnpublishedFile(t, namespace.StorageID, "first.dump")
		tm.app.WriteUnpublishedFile(t, namespace.StorageID, "second.dump")
		tm.app.ExecuteSQL(t, "CREATE SEQUENCE abandonment_after_cascade")
		encryptedKeys := getStoredS3Keys(t, tm.app, namespace.StorageID)
		require.Len(t, encryptedKeys, 1)

		var pendingBefore []string
		tm.app.QuerySQL(
			t,
			&pendingBefore,
			"SELECT row_to_json(pending)::text FROM storages_pending_file_deletions pending WHERE storage_id=? ORDER BY id",
			namespace.StorageID,
		)
		require.Len(t, pendingBefore, 2)

		tm.app.ExecuteSQL(t, `CREATE FUNCTION refuse_abandonment_audit() RETURNS trigger AS $$ BEGIN
   IF NEW.message LIKE 'Storage deleted:%' THEN
    IF EXISTS (SELECT 1 FROM storages WHERE id='`+namespace.StorageID.String()+`') OR
     EXISTS (SELECT 1 FROM storages_pending_file_deletions WHERE storage_id='`+namespace.StorageID.String()+`') THEN
      RAISE EXCEPTION 'storage cascade has not completed';
    END IF;
    PERFORM nextval('abandonment_after_cascade');
    RAISE EXCEPTION 'abandonment audit refused after cascade';
   END IF;
   RETURN NEW;
  END $$ LANGUAGE plpgsql`)
		tm.app.ExecuteSQL(
			t,
			"CREATE TRIGGER refuse_abandonment_audit BEFORE INSERT ON lib_audit_events FOR EACH ROW EXECUTE FUNCTION refuse_abandonment_audit()",
		)

		response := abandonStorageCleanup(t, tm.owner.Client, namespace.StorageID)

		requireProblem(
			t,
			response.StatusCode(),
			response.Body,
			http.StatusInternalServerError,
			apiclient.ProblemCodeInternal,
		)
		assert.Equal(t, namespace.StorageID, getStorage(t, tm.owner, namespace.StorageID).Id)
		var hasReachedAudit bool
		tm.app.QuerySQL(t, &hasReachedAudit, "SELECT is_called FROM abandonment_after_cascade")
		assert.True(t, hasReachedAudit)
		assert.Equal(t, encryptedKeys, getStoredS3Keys(t, tm.app, namespace.StorageID))
		var pendingAfter []string
		tm.app.QuerySQL(
			t,
			&pendingAfter,
			"SELECT row_to_json(pending)::text FROM storages_pending_file_deletions pending WHERE storage_id=? ORDER BY id",
			namespace.StorageID,
		)
		assert.Equal(t, pendingBefore, pendingAfter)
		for _, message := range listAuditMessages(t, tm, tm.workspace.Id) {
			assert.NotContains(t, message, "automatic cleanup abandoned")
		}
		namespace.RequireFile(t, "first.dump")
		namespace.RequireFile(t, "second.dump")
	})

	for _, shouldFail := range []bool{false, true} {
		t.Run(
			fmt.Sprintf("keeps obligations absent after a late provider response with failure=%t", shouldFail),
			func(t *testing.T) {
				t.Parallel()

				tm := newTeam(t)
				namespace := apptest.CreateStorageNamespace(t, tm.owner, tm.workspace.Id)
				tm.app.WriteUnpublishedFile(t, namespace.StorageID, "late.dump")
				tm.app.RequestStorageFileDeletions(t, storages.FileDeletionRequest{
					References: []storages.StoredFileReference{
						{StorageID: namespace.StorageID, FileName: "late.dump"},
					},
					Scope: nil,
				})
				deletion := namespace.HoldNextDeletion(t, tm.owner, shouldFail)
				completed := make(chan struct{})
				go func() {
					defer close(completed)
					tm.app.RunJobPass(t, "storage_file_deletion")
				}()
				deletion.WaitUntilStarted(t)

				abandoned := abandonStorageCleanup(t, tm.owner.Client, namespace.StorageID)
				require.Equal(t, http.StatusNoContent, abandoned.StatusCode(), "%s", abandoned.Body)
				assert.Zero(t, countPendingDeletions(t, tm.app, namespace.StorageID))
				deletion.Release()
				<-completed
				providerRequests := deletion.GetRequestCount()
				tm.app.Clock.Advance(48 * time.Hour)
				tm.app.RunJobPass(t, "storage_file_deletion")
				tm.app.RunJobPass(t, "storage_file_deletion")

				assert.Zero(t, countPendingDeletions(t, tm.app, namespace.StorageID))
				assert.Equal(t, providerRequests, deletion.GetRequestCount())
				if shouldFail {
					namespace.RequireFile(t, "late.dump")
				} else {
					namespace.RequireFileAbsent(t, "late.dump")
				}
			},
		)
	}
}

func TestAbandonStorageCleanupAccess(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		actor  string
		status int
		code   apiclient.ProblemCode
	}{
		{
			name:   "preserves pending files without disclosing cleanup when unauthenticated",
			actor:  "anonymous",
			status: http.StatusUnauthorized,
			code:   apiclient.ProblemCodeUnauthenticated,
		},
		{
			name:   "preserves pending files without disclosing cleanup when called by an outsider",
			actor:  "outsider",
			status: http.StatusNotFound,
			code:   apiclient.ProblemCodeNotFound,
		},
		{
			name:   "preserves pending files without disclosing cleanup when called by a viewer",
			actor:  "viewer",
			status: http.StatusForbidden,
			code:   apiclient.ProblemCodePermissionDenied,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			tm := newTeam(t)
			storage := tm.createLocalStorage(t, "Private archive")
			tm.app.WriteUnpublishedFile(t, storage.Id, "private-cleanup.dump")
			client := tm.app.NewClient(t)
			switch testCase.actor {
			case "outsider":
				client = tm.stranger(t).Client
			case "viewer":
				client = tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER).Client
			}

			response := abandonStorageCleanup(t, client, storage.Id)

			requireProblem(t, response.StatusCode(), response.Body, testCase.status, testCase.code)
			assert.NotContains(t, string(response.Body), "private-cleanup.dump")
			assert.NotContains(t, string(response.Body), "cleanup_pending")
			assert.NotContains(t, string(response.Body), "pendingFileCount")
			assert.Equal(t, storage.Id, getStorage(t, tm.owner, storage.Id).Id)
			assert.Equal(t, 1, countPendingDeletions(t, tm.app, storage.Id))
		})
	}

	t.Run(
		"abandons cleanup with an audit when an instance administrator has no workspace membership",
		func(t *testing.T) {
			t.Parallel()

			tm := newTeam(t)
			storage := tm.createLocalStorage(t, "Local")
			tm.app.WriteUnpublishedFile(t, storage.Id, "administrator.dump")

			response := abandonStorageCleanup(t, tm.admin.Client, storage.Id)

			require.Equal(t, http.StatusNoContent, response.StatusCode(), "%s", response.Body)
			assert.Empty(t, listStorages(t, tm.owner, tm.workspace.Id))
			assert.Zero(t, countPendingDeletions(t, tm.app, storage.Id))
			assert.Contains(
				t,
				listAuditMessages(t, tm, tm.workspace.Id),
				"Storage deleted: Local; automatic cleanup abandoned for 1 outstanding file(s): administrator.dump",
			)
		},
	)
}

func abandonStorageCleanup(
	t *testing.T,
	client *apiclient.ClientWithResponses,
	storageID uuid.UUID,
) *apiclient.DeleteStorageResponse {
	t.Helper()

	response, err := client.DeleteStorageWithResponse(
		t.Context(),
		storageID,
		&apiclient.DeleteStorageParams{ShouldAbandonCleanup: new(true)},
	)
	require.NoError(t, err)

	return response
}

type storageDeletionOutcome struct {
	Response *apiclient.DeleteStorageResponse
	Err      error
}

func TestDeleteStorageRegistrationRace(t *testing.T) {
	t.Parallel()

	t.Run("refuses deletion when a concurrent registration takes the storage lock first", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		storage := tm.createLocalStorage(t, "Local")
		barrier := tm.app.BeginSQLTransaction(t)
		barrier.ExecuteSQL(t, "SELECT pg_advisory_xact_lock(492001)")
		tm.app.ExecuteSQL(t, `CREATE FUNCTION hold_registration() RETURNS trigger AS $$ BEGIN
   PERFORM pg_advisory_xact_lock(492001); RETURN NEW; END $$ LANGUAGE plpgsql`)
		tm.app.ExecuteSQL(
			t,
			"CREATE TRIGGER hold_registration BEFORE INSERT ON storages_pending_file_deletions FOR EACH ROW EXECUTE FUNCTION hold_registration()",
		)
		written := make(chan error, 1)
		go func() {
			written <- tm.app.WriteStorageFile(t.Context(), storages.StoredFileReference{StorageID: storage.Id, FileName: "registered.dump"}, strings.NewReader("content"))
		}()
		waitForBlockedStorageQuery(t, tm, "INSERT INTO storages_pending_file_deletions")
		deleted := make(chan storageDeletionOutcome, 1)
		go func() {
			response, err := tm.owner.Client.DeleteStorageWithResponse(t.Context(), storage.Id, nil)
			deleted <- storageDeletionOutcome{Response: response, Err: err}
		}()
		waitForBlockedStorageQuery(t, tm, "FOR UPDATE")

		barrier.Commit(t)
		require.NoError(t, <-written)
		outcome := <-deleted
		require.NoError(t, outcome.Err)
		response := outcome.Response

		requireProblem(
			t,
			response.StatusCode(),
			response.Body,
			http.StatusConflict,
			apiclient.ProblemCodeCleanupPending,
		)
		assert.Equal(t, 1, countPendingDeletions(t, tm.app, storage.Id))
		assert.True(t, tm.app.IsLocalStorageFilePresent(t, "registered.dump"))
		assert.Equal(t, storage.Id, getStorage(t, tm.owner, storage.Id).Id)
	})

	t.Run("starts no upload when concurrent deletion takes the storage lock first", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		namespace := apptest.CreateStorageNamespace(t, tm.owner, tm.workspace.Id)
		observed := namespace.HoldNextDeletion(t, tm.owner, false)
		barrier := tm.app.BeginSQLTransaction(t)
		barrier.ExecuteSQL(t, "SELECT pg_advisory_xact_lock(492002)")
		tm.app.ExecuteSQL(t, `CREATE FUNCTION hold_storage_delete() RETURNS trigger AS $$ BEGIN
   PERFORM pg_advisory_xact_lock(492002); RETURN OLD; END $$ LANGUAGE plpgsql`)
		tm.app.ExecuteSQL(
			t,
			"CREATE TRIGGER hold_storage_delete BEFORE DELETE ON storages FOR EACH ROW EXECUTE FUNCTION hold_storage_delete()",
		)
		deleted := make(chan storageDeletionOutcome, 1)
		go func() {
			response, err := tm.owner.Client.DeleteStorageWithResponse(t.Context(), namespace.StorageID, nil)
			deleted <- storageDeletionOutcome{Response: response, Err: err}
		}()
		waitForBlockedStorageQuery(t, tm, "DELETE FROM")
		written := make(chan error, 1)
		go func() {
			written <- tm.app.WriteStorageFile(t.Context(), storages.StoredFileReference{StorageID: namespace.StorageID, FileName: "refused.dump"}, strings.NewReader("content"))
		}()
		waitForBlockedStorageQuery(t, tm, "INSERT INTO storages_pending_file_deletions")

		barrier.Commit(t)
		outcome := <-deleted
		require.NoError(t, outcome.Err)
		response := outcome.Response
		require.Equal(t, http.StatusNoContent, response.StatusCode(), "%s", response.Body)
		require.Error(t, <-written)

		assert.Zero(t, observed.GetRequestCount())
		namespace.RequireFileAbsent(t, "refused.dump")
		assert.Zero(t, countPendingDeletions(t, tm.app, namespace.StorageID))
		assert.Empty(t, listStorages(t, tm.owner, tm.workspace.Id))
	})
}

func waitForBlockedStorageQuery(t *testing.T, tm team, queryPart string) {
	t.Helper()

	testwait.Until(t, "the storage operation waits for the controlled lock", func(context.Context) error {
		if countRows(t, tm.app, `SELECT count(*) FROM pg_stat_activity WHERE datname=current_database()
   AND wait_event_type='Lock' AND query ILIKE ?`, "%"+queryPart+"%") == 0 {
			return fmt.Errorf("no blocked query containing %s", queryPart)
		}

		return nil
	})
}
