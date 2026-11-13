package cnpg_test

import (
	"testing"

	"databasus-backend/internal/app/apptest/physicaltest"
	containers "databasus-backend/internal/lib/testcontainers"
)

// postgresMajor pairs CloudNativePG's image of a major with the official image that boots its
// recovered clusters outside Kubernetes.
type postgresMajor struct {
	Name         string
	ClusterImage string
	BootImage    string
}

//nolint:gochecknoglobals // Immutable shared table of the majors every scenario family runs on.
var (
	postgres17 = postgresMajor{
		Name:         "PostgreSQL 17",
		ClusterImage: containers.CNPGPostgres17Image,
		BootImage:    "postgres:17",
	}
	postgres18 = postgresMajor{
		Name:         "PostgreSQL 18",
		ClusterImage: containers.CNPGPostgres18Image,
		BootImage:    "postgres:18",
	}
)

type cnpgScenario struct {
	name string
	run  func(*testing.T, physicaltest.CNPGManagerSpec)
}

func runCNPGScenarios(t *testing.T, major postgresMajor, scenarios []cnpgScenario) {
	t.Helper()

	for _, scenario := range scenarios {
		t.Run(scenario.name+major.Name, func(t *testing.T) {
			t.Parallel()

			scenario.run(t, physicaltest.CNPGManagerSpec{
				Pool:         sharedPool,
				ClusterImage: major.ClusterImage,
				BootImage:    major.BootImage,
			})
		})
	}
}
