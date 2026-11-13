package cnpg_test

import (
	"context"
	"testing"
	"time"

	containers "databasus-backend/internal/lib/testcontainers"
)

// The package's scenarios share k3s containers through one pool, which TestMain removes after the
// run.
//
//nolint:gochecknoglobals // Process-level fixture synchronization for the package's scenarios.
var sharedPool *containers.CNPGPool

func TestMain(m *testing.M) {
	pool, err := containers.NewCNPGPool(containers.CNPGPoolSpec{Capacity: containers.CNPGClustersPerK3s, Logf: nil})
	if err != nil {
		panic(err)
	}

	sharedPool = pool
	defer closePool(pool)

	m.Run()
}

func closePool(pool *containers.CNPGPool) {
	//nolint:forbidigo // TestMain has no testing.T context; removing the pool has a bounded lifetime.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := pool.Close(ctx); err != nil {
		panic(err)
	}
}
