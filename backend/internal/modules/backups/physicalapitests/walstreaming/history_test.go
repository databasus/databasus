package walstreaming_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/klauspost/compress/zstd"
	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testwait"
)

type historyRow struct {
	ID              uuid.UUID
	StorageID       uuid.UUID
	FileName        string
	HistoryFilename string
}

func (team *walTeam) getPublishedHistory(t *testing.T) []historyRow {
	t.Helper()

	var histories []historyRow
	team.App.QuerySQL(
		t,
		&histories,
		"SELECT id,storage_id,file_name,history_filename FROM backups_physical_wal_history_files WHERE database_id=? AND timeline_id=2",
		team.Database.Id,
	)

	return histories
}

func TestPhysicalTimelineHistory(t *testing.T) {
	t.Parallel()

	for _, failure := range []string{"login", "storage", "metadata", "cluster"} {
		t.Run(
			"retains the notification and publishes once after a "+failure+" failure is repaired",
			func(t *testing.T) {
				t.Parallel()

				primary, standby := containers.StartPhysicalPrimaryWithStandby(t, "postgres:17")
				observer := historyObserver{Handler: slog.DiscardHandler, events: make(chan string, 1024)}
				team := newWalTeam(t, walTeamSpec{SourceEndpoint: &standby, LogSink: observer})
				var namespace *apptest.StorageNamespace
				if failure == "storage" || failure == "metadata" {
					storage := apptest.CreateStorageNamespace(t, team.Owner, team.Workspace.Id)
					namespace = &storage
					team.StorageID = storage.StorageID
				}

				if failure == "cluster" {
					team.App.ExecuteSQL(
						t,
						"UPDATE databases_postgresql_physical SET system_identifier='another-cluster' WHERE database_id=?",
						team.Database.Id,
					)
				}

				team.enable(t)
				team.waitForReceiver(t)
				primaryTeam := *team
				primaryTeam.Source = primary
				primarySource := primaryTeam.openSource(t)
				rotateWal(t, primarySource)
				if failure != "cluster" {
					team.App.WaitForPhysicalBackups(t)
					team.App.PausePhysicalJobs(t)
				}

				source := team.openSource(t)

				if failure == "login" {
					_, err := primarySource.Exec(t.Context(), "ALTER ROLE testuser NOLOGIN")
					require.NoError(t, err)

					var roleLSN string
					require.NoError(
						t,
						primarySource.QueryRow(t.Context(), "SELECT pg_current_wal_insert_lsn()::text").Scan(&roleLSN),
					)
					testwait.Until(t, "the login failure replays before promotion", func(ctx context.Context) error {
						var hasReplayed bool
						if err := source.QueryRow(ctx, "SELECT pg_last_wal_replay_lsn()>=$1::pg_lsn", roleLSN).
							Scan(&hasReplayed); err != nil {
							return err
						}

						if !hasReplayed {
							return fmt.Errorf("the role change has not replayed")
						}

						return nil
					})
				}

				if failure == "storage" {
					namespace.MakeUnavailable(t, team.Owner)
				}

				var metadataFailure apptest.StorageMetadataFailure
				if failure == "metadata" {
					metadataFailure = namespace.FailMetadataUploads(t, team.Owner, team.Database.Id.String()+"-HIST-")
				}

				var isPromoted bool
				require.NoError(t, source.QueryRow(t.Context(), "SELECT pg_promote()").Scan(&isPromoted))
				require.True(t, isPromoted)

				historyPath := team.App.GetWalQueueFilePath(team.Database.Id, "00000002.history")
				require.NoError(t, os.WriteFile(historyPath, []byte("untrusted local bytes\n"), 0o600))

				var abandonedArtifact string
				if failure == "metadata" {
					testwait.Until(
						t,
						"metadata upload starts after the artifact is stored",
						func(context.Context) error {
							select {
							case fileName := <-metadataFailure.Started:
								abandonedArtifact = strings.TrimSuffix(fileName, ".metadata")

								return nil
							default:
								return fmt.Errorf("no metadata upload yet")
							}
						},
					)
					namespace.RequireFile(t, abandonedArtifact)
					namespace.RequireFileAbsent(t, abandonedArtifact+".metadata")
					require.Empty(t, team.getPublishedHistory(t))
					require.FileExists(t, historyPath)

					metadataFailure.Fail()
				}

				observer.Wait(t, "history upload failed; will retry next tick")
				require.FileExists(t, historyPath)
				require.Empty(t, team.getPublishedHistory(t))

				if failure == "metadata" {
					testwait.Until(
						t,
						"the abandoned history artifact is deleted before retry succeeds",
						func(ctx context.Context) error {
							team.App.RunJobPass(t, "storage_file_deletion")
							_, err := namespace.Client.StatObject(
								ctx,
								namespace.Bucket,
								abandonedArtifact,
								minio.StatObjectOptions{},
							)
							if err == nil {
								return fmt.Errorf("abandoned artifact still exists")
							}

							if minio.ToErrorResponse(err).StatusCode != http.StatusNotFound {
								return fmt.Errorf("check abandoned artifact: %w", err)
							}

							return nil
						},
					)
					namespace.RequireFileAbsent(t, abandonedArtifact+".metadata")
					require.Empty(t, team.getPublishedHistory(t))
					require.FileExists(t, historyPath)

					metadataFailure.Repair()
				}

				if failure == "login" {
					_, err := source.Exec(t.Context(), "ALTER ROLE testuser LOGIN")
					require.NoError(t, err)
				}

				if failure == "storage" {
					namespace.MakeAvailable(t, team.Owner)
				}

				if failure == "cluster" {
					team.disable(t)
					testwait.Until(t, "the rejected source snapshot stops", func(context.Context) error {
						team.App.RunJobPass(t, "backups_physical_wal_stream_manager")
						if team.live(t).IsStreamerRunning {
							return fmt.Errorf("streamer still running")
						}

						return nil
					})
					var identifier string
					require.NoError(
						t,
						source.QueryRow(t.Context(), "SELECT system_identifier::text FROM pg_control_system()").
							Scan(&identifier),
					)
					team.App.ExecuteSQL(
						t,
						"UPDATE databases_postgresql_physical SET system_identifier=? WHERE database_id=?",
						identifier,
						team.Database.Id,
					)
					team.enable(t)
					team.App.RunJobPass(t, "backups_physical_wal_stream_manager")
					team.waitForReceiver(t)
					require.NoError(t, os.WriteFile(historyPath, []byte("untrusted local bytes\n"), 0o600))
				}

				observer.Wait(t, "timeline switch observed via .history")
				testwait.Until(t, "the verified history notification is removed", func(context.Context) error {
					if _, err := os.Stat(historyPath); !os.IsNotExist(err) {
						return fmt.Errorf("notification is still present")
					}

					return nil
				})

				histories := team.getPublishedHistory(t)
				require.Len(t, histories, 1)
				require.Equal(t, team.StorageID, histories[0].StorageID)
				require.Equal(t, "00000002.history", histories[0].HistoryFilename)

				if failure == "metadata" {
					require.NotEqual(t, abandonedArtifact, histories[0].FileName)
					namespace.RequireFile(t, histories[0].FileName)
					namespace.RequireFile(t, histories[0].FileName+".metadata")
					namespace.RequireFileAbsent(t, abandonedArtifact)
				}

				var sourceHistory []byte
				require.NoError(
					t,
					source.QueryRow(t.Context(), "SELECT pg_read_binary_file('pg_wal/00000002.history')").
						Scan(&sourceHistory),
				)
				readArtifact := func(fileName string) []byte {
					if namespace == nil {
						body, err := os.ReadFile(team.App.GetLocalStorageFilePath(fileName))
						require.NoError(t, err)

						return body
					}

					object, err := namespace.Client.GetObject(
						t.Context(),
						namespace.Bucket,
						fileName,
						minio.GetObjectOptions{},
					)
					require.NoError(t, err)
					defer func() { _ = object.Close() }()
					body, err := io.ReadAll(object)
					require.NoError(t, err)

					return body
				}
				decoder, err := zstd.NewReader(nil)
				require.NoError(t, err)
				body, err := decoder.DecodeAll(readArtifact(histories[0].FileName), nil)
				decoder.Close()
				require.NoError(t, err)
				assert.Equal(t, sourceHistory, body)

				var metadata struct {
					WalHistoryFileID uuid.UUID `json:"walHistoryFileId"`
					DatabaseID       uuid.UUID `json:"databaseId"`
					TimelineID       int       `json:"timelineId"`
					HistoryFilename  string    `json:"historyFilename"`
				}
				require.NoError(t, json.Unmarshal(readArtifact(histories[0].FileName+".metadata"), &metadata))
				assert.Equal(t, histories[0].ID, metadata.WalHistoryFileID)
				assert.Equal(t, team.Database.Id, metadata.DatabaseID)
				assert.Equal(t, 2, metadata.TimelineID)
				assert.Equal(t, "00000002.history", metadata.HistoryFilename)

				require.NoError(t, os.WriteFile(historyPath, []byte("different notification bytes\n"), 0o600))
				observer.Wait(t, "timeline switch observed via .history")
				assert.Equal(t, histories, team.getPublishedHistory(t))
				testwait.Until(
					t,
					"failed history attempts leave the cleanup queue after repair",
					func(context.Context) error {
						team.App.RunJobPass(t, "storage_file_deletion")
						var pendingHistory int
						team.App.QuerySQL(
							t,
							&pendingHistory,
							"SELECT count(*) FROM storages_pending_file_deletions WHERE storage_id=? AND file_name LIKE ?",
							team.StorageID,
							team.Database.Id.String()+"-HIST-%",
						)
						if pendingHistory != 0 {
							return fmt.Errorf("%d history files await cleanup", pendingHistory)
						}

						return nil
					},
				)
			},
		)
	}
}
