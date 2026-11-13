package containers

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// On the 32-vCPU, 64-GB reference host, four durable two-node clusters started together on one
// k3s were ready in 51 s, using 1.6 GiB, while listing all pods took 0.09 s. This capacity keeps
// the shared API server responsive; extra scenarios lease another k3s instead of waiting.
const CNPGClustersPerK3s = 4

// CNPGPool shares k3s containers between the scenarios of one package run. It limits how many
// CloudNativePG clusters share a k3s, never how many scenarios run.
type CNPGPool struct {
	mu        sync.Mutex
	capacity  int
	authority *TestCertificateAuthority
	members   []*pooledK3s
	logf      func(format string, arguments ...any)
}

type pooledK3s struct {
	k3s    *K3s
	leases int
	ready  chan struct{}
	err    error
}

type CNPGPoolSpec struct {
	Capacity int
	Logf     func(format string, arguments ...any)
}

func NewCNPGPool(spec CNPGPoolSpec) (*CNPGPool, error) {
	authority, err := NewTestCertificateAuthority()
	if err != nil {
		return nil, err
	}

	return &CNPGPool{capacity: spec.Capacity, authority: authority, logf: spec.Logf}, nil
}

// StartCluster leases a place on a k3s with free capacity, starting a new k3s when every running one
// is full, and starts a CloudNativePG cluster there. The lease ends after the cluster's namespace is
// deleted.
func (pool *CNPGPool) StartCluster(t *testing.T, spec CNPGClusterSpec) *CNPGCluster {
	t.Helper()

	member := pool.lease(t)
	spec.K3s = member.k3s
	spec.Authority = pool.authority

	return StartCNPGCluster(t, spec)
}

func (pool *CNPGPool) GetAuthority() *TestCertificateAuthority {
	return pool.authority
}

func (pool *CNPGPool) GetK3sCount() int {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	return len(pool.members)
}

func (pool *CNPGPool) Close(ctx context.Context) error {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	var errs []error
	for _, member := range pool.members {
		<-member.ready
		if member.k3s != nil {
			errs = append(errs, member.k3s.Terminate(ctx))
		}
	}

	pool.members = nil
	if err := errors.Join(errs...); err != nil {
		return fmt.Errorf("close the k3s pool: %w", err)
	}

	return nil
}

func (pool *CNPGPool) lease(t *testing.T) *pooledK3s {
	t.Helper()

	pool.mu.Lock()
	var member *pooledK3s
	for _, candidate := range pool.members {
		if candidate.leases < pool.capacity && !candidate.hasFailed() {
			member = candidate
			break
		}
	}

	isNew := member == nil
	if isNew {
		member = &pooledK3s{ready: make(chan struct{})}
		pool.members = append(pool.members, member)
	}

	member.leases++
	pool.mu.Unlock()
	t.Cleanup(func() { pool.release(member) })

	if isNew {
		// The k3s outlives the scenario that started it; the package's TestMain closes the pool.
		member.k3s, member.err = StartK3s(context.WithoutCancel(t.Context()), K3sSpec{Logf: pool.logf})
		close(member.ready)
	}

	select {
	case <-member.ready:
	case <-t.Context().Done():
		t.Fatalf("wait for a pooled k3s: %v", t.Context().Err())
	}

	require.NoError(t, member.err, "start a pooled k3s")

	return member
}

func (pool *CNPGPool) release(member *pooledK3s) {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	member.leases--
}

func (member *pooledK3s) hasFailed() bool {
	select {
	case <-member.ready:
		return member.err != nil
	default:
		return false
	}
}
