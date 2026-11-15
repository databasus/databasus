package containers

import (
	"fmt"
	"strings"
	"sync"
	"testing"
)

// PostgresTransferRate is the BASE_BACKUP MAX_RATE a held transfer runs at, in kilobytes per second.
type PostgresTransferRate int

const (
	// PostgreSQL's minimum keeps a large failover payload in flight after the gate opens.
	SlowestPostgresTransferRate PostgresTransferRate = 32
	// A released backup of a fresh cluster still finishes within a test's wait limits.
	SteadyPostgresTransferRate PostgresTransferRate = 2048
)

type PostgresTransferGate struct {
	Reached      chan PostgresProxyEvent
	release      chan struct{}
	releaseOnce  sync.Once
	rate         PostgresTransferRate
	hasTriggered bool
}

func (p *PostgresProxy) HoldTransfer(t *testing.T, rate PostgresTransferRate) *PostgresTransferGate {
	t.Helper()

	p.mu.Lock()
	defer p.mu.Unlock()
	gate := &PostgresTransferGate{
		Reached: make(chan PostgresProxyEvent, 1),
		release: make(chan struct{}),
		rate:    rate,
	}
	p.transferGate = gate
	t.Cleanup(gate.Release)

	return gate
}

func (gate *PostgresTransferGate) Release() {
	gate.releaseOnce.Do(func() { close(gate.release) })
}

func (p *PostgresProxy) limitBackupTransfer(command string) (string, error) {
	p.mu.Lock()
	gate := p.transferGate
	hasPendingGate := gate != nil && !gate.hasTriggered
	p.mu.Unlock()

	if !hasPendingGate || !strings.HasPrefix(command, "BASE_BACKUP") {
		return command, nil
	}

	options, hasOptions := strings.CutSuffix(command, ")")
	if !hasOptions {
		return "", fmt.Errorf("backup transfer gate requires parenthesized BASE_BACKUP options")
	}

	return fmt.Sprintf("%s, MAX_RATE %d)", options, gate.rate), nil
}
