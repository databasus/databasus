package containers

import (
	"fmt"
	"strings"
	"sync"
	"testing"
)

type PostgresTransferGate struct {
	Reached      chan PostgresProxyEvent
	release      chan struct{}
	releaseOnce  sync.Once
	hasTriggered bool
}

func (p *PostgresProxy) HoldTransfer(t *testing.T) *PostgresTransferGate {
	t.Helper()

	p.mu.Lock()
	defer p.mu.Unlock()
	gate := &PostgresTransferGate{Reached: make(chan PostgresProxyEvent, 1), release: make(chan struct{})}
	p.transferGate = gate
	t.Cleanup(gate.Release)

	return gate
}

func (gate *PostgresTransferGate) Release() {
	gate.releaseOnce.Do(func() { close(gate.release) })
}

func (p *PostgresProxy) limitBackupTransfer(command string) (string, error) {
	p.mu.Lock()
	hasPendingGate := p.transferGate != nil && !p.transferGate.hasTriggered
	p.mu.Unlock()

	if !hasPendingGate || !strings.HasPrefix(command, "BASE_BACKUP") {
		return command, nil
	}

	options, hasOptions := strings.CutSuffix(command, ")")
	if !hasOptions {
		return "", fmt.Errorf("backup transfer gate requires parenthesized BASE_BACKUP options")
	}

	// PostgreSQL's minimum rate keeps the large failover payload in flight after the gate opens.
	return options + ", MAX_RATE 32)", nil
}
