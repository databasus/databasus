package walstreaming_test

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	"databasus-backend/internal/app/apptest/enginetest"
	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testwait"
)

func TestPhysicalTimelineHistoryConcurrency(t *testing.T) {
	t.Parallel()

	t.Run("publishes one encrypted history when full completion races the receiver", func(t *testing.T) {
		t.Parallel()

		_, standby := containers.StartPhysicalPrimaryWithStandby(t, "postgres:17")
		observer := historyObserver{Handler: slog.DiscardHandler, events: make(chan string, 1024)}
		team := newWalTeam(t, walTeamSpec{SourceEndpoint: &standby, LogSink: observer})
		source := team.openSource(t)

		var isPromoted bool
		require.NoError(t, source.QueryRow(t.Context(), "SELECT pg_promote()").Scan(&isPromoted))
		require.True(t, isPromoted)

		engineTeam := enginetest.Team{App: team.App, Owner: team.Owner}
		user := engineTeam.CreateLimitedUser(t, team.Database)
		settings, err := team.Database.Settings.AsPostgresqlPhysicalSettings()
		require.NoError(t, err)
		settings.Username = user.Username
		settings.Password = &user.Password
		team.Database = engineTeam.UpdateDatabaseSettings(t, team.Database, settings)

		registration := team.App.BeginSQLTransaction(t)
		lockKey := "concurrent-history-" + uuid.NewString()
		registration.ExecuteSQL(t, "SELECT pg_advisory_xact_lock(hashtextextended(?,0))", lockKey)
		var lockOwner int
		registration.QuerySQL(t, &lockOwner, "SELECT pg_backend_pid()")
		team.App.ExecuteSQL(t, "CREATE TABLE history_upload_attempts(file_name text NOT NULL)")
		team.App.ExecuteSQL(t, fmt.Sprintf(`CREATE FUNCTION hold_concurrent_history() RETURNS trigger AS $$ BEGIN
   IF NEW.file_name LIKE '%s-HIST-%%' AND right(NEW.file_name,9)<>'.metadata' AND NEW.not_before>NEW.created_at THEN
    PERFORM pg_advisory_xact_lock(hashtextextended('%s',0));
    INSERT INTO history_upload_attempts VALUES (NEW.file_name);
   END IF; RETURN NEW; END $$ LANGUAGE plpgsql`, team.Database.Id, lockKey))
		team.App.ExecuteSQL(
			t,
			"CREATE TRIGGER hold_concurrent_history BEFORE INSERT ON storages_pending_file_deletions FOR EACH ROW EXECUTE FUNCTION hold_concurrent_history()",
		)

		historyPath := team.App.GetWalQueueFilePath(team.Database.Id, "00000002.history")
		require.NoError(t, os.MkdirAll(filepath.Dir(historyPath), 0o700))
		require.NoError(t, os.WriteFile(historyPath, []byte("notification only\n"), 0o600))

		engineTeam.EnableBackups(t, team.Database.Id, team.StorageID, apiclient.BackupEncryptionENCRYPTED)
		testwait.Until(t, "both history publishers reach independent artifact writes", func(context.Context) error {
			var waiting int
			team.App.QuerySQL(
				t,
				&waiting,
				`SELECT count(*) FROM pg_locks waiting JOIN pg_locks held USING (locktype,database,classid,objid,objsubid)
    WHERE NOT waiting.granted AND held.granted AND held.pid=? AND waiting.locktype='advisory'`,
				lockOwner,
			)
			if waiting < 2 {
				return fmt.Errorf("%d history writers are waiting", waiting)
			}

			return nil
		})
		registration.Commit(t)
		team.App.WaitForPhysicalBackups(t)
		testwait.Until(t, "the receiver acknowledges the concurrent history winner", func(context.Context) error {
			if _, err := os.Stat(historyPath); !os.IsNotExist(err) {
				return fmt.Errorf("notification is still present")
			}

			return nil
		})

		observer.RequireNoFailures(t)

		histories := team.getPublishedHistory(t)
		require.Len(t, histories, 1)

		var attempts []string
		team.App.QuerySQL(t, &attempts, "SELECT DISTINCT file_name FROM history_upload_attempts")
		require.Len(t, attempts, 2)
		assert.Contains(t, attempts, histories[0].FileName)

		metadataBody, err := os.ReadFile(team.App.GetLocalStorageFilePath(histories[0].FileName + ".metadata"))
		require.NoError(t, err)
		var metadata struct {
			Encryption     string `json:"encryption"`
			EncryptionSalt string `json:"encryptionSalt"`
			EncryptionIV   string `json:"encryptionIv"`
		}
		require.NoError(t, json.Unmarshal(metadataBody, &metadata))

		assert.Equal(t, "ENCRYPTED", metadata.Encryption)
		assert.NotEmpty(t, metadata.EncryptionSalt)
		assert.NotEmpty(t, metadata.EncryptionIV)

		rotateWal(t, source)
		rotateWal(t, source)
		testwait.Until(t, "archived WAL extends the promoted full", func(context.Context) error {
			var hasReplay bool
			team.App.QuerySQL(
				t,
				&hasReplay,
				`SELECT EXISTS(SELECT 1 FROM backups_physical b JOIN backups_physical_wal_segments w
    ON w.database_id=b.database_id AND w.timeline_id=b.timeline_id WHERE b.database_id=? AND b.status='COMPLETED'
    AND w.start_lsn<=b.stop_lsn AND w.end_lsn>b.stop_lsn AND w.file_name IS NOT NULL)`,
				team.Database.Id,
			)
			if !hasReplay {
				return fmt.Errorf("no committed WAL beyond the full")
			}

			return nil
		})

		var token struct {
			Token string `json:"token"`
		}
		team.call(t, apptest.PhysicalPitrTokenRequest(team.Database.Id, nil), http.StatusCreated, &token)

		status, bundle := team.App.CallPhysicalAPI(t, apptest.Session{}, apptest.PhysicalDownloadRequest(token.Token))
		require.Equal(t, http.StatusOK, status, "%s", bundle)

		reader := tar.NewReader(bytes.NewReader(bundle))
		var historyBody []byte

		for {
			header, err := reader.Next()
			if errors.Is(err, io.EOF) {
				break
			}

			require.NoError(t, err)

			if header.Name == "wal/00000002.history" {
				historyBody, err = io.ReadAll(reader)
				require.NoError(t, err)
			}
		}

		var sourceHistory []byte
		require.NoError(
			t,
			source.QueryRow(t.Context(), "SELECT pg_read_binary_file('pg_wal/00000002.history')").Scan(&sourceHistory),
		)
		require.Equal(t, sourceHistory, historyBody)

		team.App.ExecuteSQL(t, "DROP TRIGGER hold_concurrent_history ON storages_pending_file_deletions")

		for _, attempt := range attempts {
			if attempt == histories[0].FileName {
				continue
			}

			testwait.Until(t, "cleanup removes the losing history artifact and metadata", func(context.Context) error {
				team.App.RunJobPass(t, "storage_file_deletion")

				for _, fileName := range []string{attempt, attempt + ".metadata"} {
					if _, err := os.Stat(team.App.GetLocalStorageFilePath(fileName)); !os.IsNotExist(err) {
						return fmt.Errorf("losing file remains: %s", fileName)
					}
				}

				return nil
			})
		}

		require.FileExists(t, team.App.GetLocalStorageFilePath(histories[0].FileName))
		require.FileExists(t, team.App.GetLocalStorageFilePath(histories[0].FileName+".metadata"))
	})
}
