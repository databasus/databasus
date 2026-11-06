package walstreaming_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	"databasus-backend/internal/lib/testwait"
)

func TestPhysicalWalSamples(t *testing.T) {
	t.Parallel()

	t.Run(
		"records source and archive values and clears its failing time after the first healthy sample",
		func(t *testing.T) {
			t.Parallel()

			team := newWalTeam(t, walTeamSpec{})
			team.App.InsertPhysicalWalStreamer(
				t,
				apptest.PhysicalWalStreamerSeed{
					DatabaseID:   team.Database.Id,
					Status:       "FAILED",
					FailingSince: new(time.Now().UTC().Add(-time.Hour)),
				},
			)

			team.enable(t)
			team.waitForReceiver(t)
			source := team.openSource(t)

			var first streamStatistics
			testwait.Until(
				t,
				"the first sample is visible and its streamer failure has cleared",
				func(context.Context) error {
					first = team.statistics(t)
					if first.Current == nil {
						return fmt.Errorf("no current sample")
					}
					if first.Streamer.FailingSince != nil {
						return fmt.Errorf("the streamer's failing time has not cleared")
					}
					return nil
				},
			)

			assert.Equal(t, "REACHABLE", first.Current.SourceState)
			assert.Nil(t, first.Current.SourceWrittenBytes)
			assert.NotNil(t, first.Current.SourceCurrentLSN)
			assert.NotNil(t, first.Current.SlotWalStatus)
			assert.NotNil(t, first.Current.SlotLagBytes)
			assert.Positive(t, first.Current.LocalQueuePauseBytes)
			assert.Nil(t, first.Streamer.FailingSince)

			generateWal(t, source, 4096)
			rotateWal(t, source)
			team.waitForWal(t, 1)
			var archived streamStatistics
			testwait.Until(t, "a sample reports the committed archive", func(context.Context) error {
				archived = team.statistics(t)
				if archived.Current == nil || !archived.Current.SampledAt.After(first.Current.SampledAt) ||
					archived.Current.ArchivedSegments == 0 {
					return fmt.Errorf("no archive sample")
				}
				return nil
			})

			assert.Positive(t, archived.Current.ArchivedSegments)
			assert.Positive(t, archived.Current.ArchivedWalBytes)
			assert.Positive(t, archived.Current.ArchivedStoredBytes)
			assert.Positive(t, archived.Current.HighestConcurrentUploads)
			assert.NotNil(t, archived.Current.SegmentWriteAvgMillis)
			assert.NotNil(t, archived.Current.SegmentWriteMaxMillis)

			generateWal(t, source, 4096)
			testwait.Until(t, "the next sample reports new source writes", func(context.Context) error {
				next := team.statistics(t)
				if next.Current == nil || !next.Current.SampledAt.After(archived.Current.SampledAt) ||
					next.Current.SourceWrittenBytes == nil ||
					*next.Current.SourceWrittenBytes <= 0 {
					return fmt.Errorf("no later write sample")
				}
				return nil
			})
		},
	)
	t.Run(
		"preserves the first unreachable time across sessions when source login fails again before the next sample",
		func(t *testing.T) {
			t.Parallel()

			team := newWalTeam(t, walTeamSpec{})
			role := team.useReplicationRole(t)
			team.enable(t)
			team.waitForReceiver(t)
			source := team.openSource(t)

			setLogin := func(isAllowed bool) {
				command := " NOLOGIN"
				if isAllowed {
					command = " LOGIN"
				}
				_, err := source.Exec(t.Context(), "ALTER ROLE "+pgx.Identifier{role}.Sanitize()+command)
				require.NoError(t, err)
			}

			getActiveWalSlot(t, source)
			setLogin(false)
			var first streamStatistics
			testwait.Until(t, "the source outage appears in a persisted sample", func(context.Context) error {
				first = team.statistics(t)
				if first.Current == nil || first.Current.SourceState != "UNREACHABLE" ||
					first.Current.SourceUnreachableSince == nil {
					return fmt.Errorf("no unreachable sample")
				}
				return nil
			})

			assert.Nil(t, first.Current.SourceCurrentLSN)
			assert.Nil(t, first.Current.SlotLagBytes)
			assert.Nil(t, first.Current.SourceWrittenBytes)

			setLogin(true)
			team.App.RestartModules(t)
			team.waitForReceiver(t)
			getActiveWalSlot(t, source)
			setLogin(false)
			testwait.Until(t, "the new session keeps the original outage time", func(context.Context) error {
				next := team.statistics(t)
				if next.Current == nil || !next.Current.SessionStartedAt.After(first.Current.SessionStartedAt) ||
					next.Current.SourceState != "UNREACHABLE" ||
					next.Current.SourceUnreachableSince == nil {
					return fmt.Errorf("no new unreachable session sample")
				}
				assert.True(t, first.Current.SourceUnreachableSince.Equal(*next.Current.SourceUnreachableSince))
				assert.Nil(t, next.Current.SlotWalStatus)
				assert.Nil(t, next.Current.SlotLagBytes)
				assert.Nil(t, next.Current.SourceWrittenBytes)
				return nil
			})

			setLogin(true)
		},
	)
}
