package catalog

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_PhysicalBackupFailoverErrorReasons_SerializeAsStableValues(t *testing.T) {
	reasons := []BackupErrorReason{
		BackupErrorTimelineSwitchDetected,
		BackupErrorFailoverDuringBackup,
	}

	serializedReasons, err := json.Marshal(reasons)
	require.NoError(t, err)
	require.JSONEq(t, `["TIMELINE_SWITCH_DETECTED","FAILOVER_DURING_BACKUP"]`, string(serializedReasons))
}
