// Package recoverypause holds whether this installation is paused after a recovery. A recovered
// installation may run beside the lost one it was restored from, so until root admin confirms the
// old one is stopped, nothing may reach a remote storage, a source database or a notifier. The
// owner of the persisted pause sets the gate at startup and clears it on activation; every module
// asks the gate before such an effect.
package recoverypause

import "sync/atomic"

type Gate struct {
	isPaused atomic.Bool
}

func NewGate(isPaused bool) *Gate {
	gate := &Gate{isPaused: atomic.Bool{}}
	gate.isPaused.Store(isPaused)

	return gate
}

func (g *Gate) IsPaused() bool {
	return g.isPaused.Load()
}

// Check returns ErrRecoveryPaused while paused, for an operation with an external effect.
func (g *Gate) Check() error {
	if g.IsPaused() {
		return ErrRecoveryPaused
	}

	return nil
}

// Resume opens the gate after the persisted pause was cleared.
func (g *Gate) Resume() {
	g.isPaused.Store(false)
}
