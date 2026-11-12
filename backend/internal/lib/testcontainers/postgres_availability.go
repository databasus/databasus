package containers

func (p *PostgresProxy) SetUnavailable(isUnavailable bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.isUnavailable = isUnavailable
}

func (p *PostgresProxy) SetReplicationBlocked(isBlocked bool) {
	p.mu.Lock()
	p.isReplicationBlocked = isBlocked
	p.mu.Unlock()

	if !isBlocked {
		return
	}

	for _, event := range p.GetEvents() {
		if event.IsReplication {
			p.CutConnection(event.ConnectionID)
		}
	}
}
