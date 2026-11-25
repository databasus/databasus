package instancebackups_test

import (
	"testing"
	"time"
)

// TestMain runs every scenario on a server west of UTC. Schedules are saved in UTC, so a slot or a
// deadline worked out on the server's calendar lands on the wrong hour only when the zones differ.
func TestMain(m *testing.M) {
	time.Local = time.FixedZone("UTC-3", -3*60*60)

	m.Run()
}
