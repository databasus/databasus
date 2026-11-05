package container

import "github.com/google/uuid"

type SpawnRequest struct {
	PgMajor            string
	CPUPerJob          int
	RAMMbPerJob        int
	VerificationID     uuid.UUID
	TimescaledbVersion string
}

type spawnPlan struct {
	verificationID uuid.UUID
	image          string
	password       string
	cpuPerJob      int
	ramMbPerJob    int
	networkID      string
	labels         map[string]string
}

// SpawnSpec is the security-hardened container request. Every clause is a
// load-bearing control: the container is the only boundary around a restore
// that runs attacker-controlled code as the DB superuser. Constructed by the
// Manager (unit-testable) and translated 1:1 to Docker by DockerEngine.
type SpawnSpec struct {
	Name        string
	Image       string
	Cmd         []string
	Env         []string
	Labels      map[string]string
	NanoCPUs    int64
	MemoryBytes int64
	PidsLimit   int64
	NetworkID   string

	// The hardening invariants, explicit so a unit test asserts them and the
	// engine translation cannot silently drop one.
	CapDropAll     bool
	ReadonlyRootfs bool
	User           string
}
