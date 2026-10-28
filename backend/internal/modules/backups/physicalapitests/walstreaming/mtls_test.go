package walstreaming_test

import (
	"testing"
)

func TestArchivePhysicalWalOverMtls(t *testing.T) {
	t.Run("archives segments from a PostgreSQL source authenticated with a client certificate", func(t *testing.T) {
		team := newWalTeam(t, walTeamSpec{IsMtls: true})
		team.enable(t)
		team.waitForReceiver(t)
		source := team.openSource(t)
		rotateWal(t, source)
		team.waitForWal(t, 1)
	})
}
