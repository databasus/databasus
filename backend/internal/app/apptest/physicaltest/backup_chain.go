package physicaltest

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/testwait"
)

type incrementalSpec struct {
	ParentStopLSN uint64
	Count         int
}

func (team Team) listChain(t *testing.T) []backupRow {
	t.Helper()

	chain := slices.Concat(team.listBackups(t, "FULL"), team.listBackups(t, "INCREMENTAL"))
	slices.SortFunc(chain, func(first, second backupRow) int {
		if byTime := first.CreatedAt.Compare(second.CreatedAt); byTime != 0 {
			return byTime
		}
		return cmp.Compare(first.ID.String(), second.ID.String())
	})
	return chain
}

func (team Team) waitForChain(t *testing.T, incrementalCount int) []backupRow {
	t.Helper()

	var chain []backupRow
	testwait.Until(t, "the physical backup chain completes", func(context.Context) error {
		chain = team.listChain(t)

		hasCompletedFull := false
		completedIncrementals := 0
		for _, backup := range chain {
			if backup.Status == "ERROR" || backup.Status == "CHAIN_BROKEN" {
				failureMessage := ""
				if backup.FailMessage != nil {
					failureMessage = *backup.FailMessage
				}
				t.Fatalf("backup %s (%s) failed with %s: %s", backup.ID, backup.Type, backup.Status, failureMessage)
			}
			if backup.Status != "COMPLETED" {
				continue
			}
			if backup.Type == "FULL" {
				hasCompletedFull = true
			} else {
				completedIncrementals++
			}
		}

		if !hasCompletedFull || completedIncrementals != incrementalCount {
			return fmt.Errorf(
				"chain has completed FULL=%v and %d of %d incrementals: %+v",
				hasCompletedFull,
				completedIncrementals,
				incrementalCount,
				chain,
			)
		}
		return nil
	})
	return chain
}

func chainTipStopLSN(t *testing.T, chain []backupRow) uint64 {
	t.Helper()

	require.NotEmpty(t, chain)
	return parseLSN(t, chain[len(chain)-1].StopLSN)
}

func (team Team) buildIncremental(t *testing.T, conn *pgx.Conn, spec incrementalSpec) []backupRow {
	t.Helper()

	generateWalActivity(t, conn, 32*1024*1024)
	_, err := conn.Exec(t.Context(), "CHECKPOINT")
	require.NoError(t, err)
	forceWalRotation(t, conn)
	waitForWalSummaries(t, conn, spec.ParentStopLSN)

	team.triggerIncremental(t)
	return team.waitForChain(t, spec.Count)
}
