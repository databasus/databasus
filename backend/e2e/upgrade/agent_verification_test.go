package upgrade

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/testwait"
)

type assignedVerification struct {
	verificationID string
	agentID        string
}

func getOrCreateAgentUpgradeVerification(t *testing.T, api *apiClient, written writtenUpgradeData) string {
	t.Helper()

	var verification currentVerification

	require.NoError(t, api.get(t.Context(), "/verifications/"+written.verificationID, &verification))

	if verification.Status == "PENDING" {
		return verification.ID
	}

	require.NoError(
		t,
		api.post(t.Context(), "/verifications", map[string]string{"backupId": written.backupID}, &verification),
	)

	return verification.ID
}

func checkAgentBackupVerification(t *testing.T, api *apiClient, assignment assignedVerification) {
	t.Helper()

	var verification struct {
		Status      string `json:"status"`
		AgentID     string `json:"agentId"`
		FailMessage string `json:"failMessage"`
		TableStats  []struct {
			SchemaName string `json:"schemaName"`
			Name       string `json:"name"`
			RowCount   int64  `json:"rowCount"`
		} `json:"tableStats"`
	}

	testwait.Until(t, "the updated agent to finish verification", func(ctx context.Context) error {
		if err := api.get(ctx, "/verifications/"+assignment.verificationID, &verification); err != nil {
			return err
		}

		if verification.Status == "FAILED" || verification.Status == "CANCELED" {
			t.Fatalf("verification %s: %s", verification.Status, verification.FailMessage)
		}

		if verification.Status != "COMPLETED" {
			return fmt.Errorf("verification is %s", verification.Status)
		}

		return nil
	})

	require.Equal(t, assignment.agentID, verification.AgentID)

	for _, table := range verification.TableStats {
		if table.Name == "upgrade_rows" {
			require.Equal(t, sourceSchemaName, table.SchemaName)
			require.Equal(t, int64(len(getSourceRows())), table.RowCount)

			return
		}
	}

	t.Fatal("verification did not report the backed-up table")
}
