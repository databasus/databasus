package mysql

import (
	"context"
	"log/slog"

	"databasus-backend/internal/lib/sshtunnel"
)

// Tunnel is a source reached through its SSH tunnel. It belongs to the operation that opened it,
// which closes it when it is done.
type Tunnel struct {
	source    *Source
	forwarder *sshtunnel.Forwarder
}

// OpenTunnel returns the source itself without an SSH tunnel, so callers never branch. Through a
// tunnel the source connects to the local end of the tunnel.
func (s *Source) OpenTunnel(ctx context.Context, logger *slog.Logger) (*Tunnel, error) {
	if s.settings.SSHTunnel == nil {
		return &Tunnel{source: s, forwarder: nil}, nil
	}

	forwarder, err := sshtunnel.Open(ctx, sshtunnel.OpenSpec{
		Settings:         *s.settings.SSHTunnel,
		Target:           sshtunnel.Endpoint{Host: s.settings.Host, Port: s.settings.Port},
		Logger:           logger,
		HandshakeTimeout: 0,
	})
	if err != nil {
		return nil, err
	}

	localEndpoint := forwarder.GetLocalEndpoint()

	settingsThroughTunnel := s.settings
	settingsThroughTunnel.Host = localEndpoint.Host
	settingsThroughTunnel.Port = localEndpoint.Port
	settingsThroughTunnel.SSHTunnel = nil

	return &Tunnel{
		source: &Source{
			settings:         settingsThroughTunnel,
			clientTools:      s.clientTools,
			generateUsername: s.generateUsername,
		},
		forwarder: forwarder,
	}, nil
}

func (t *Tunnel) GetSource() *Source {
	return t.source
}

// GetLocalEndpoint returns the local end of the tunnel, or false when the source is reached
// directly.
func (t *Tunnel) GetLocalEndpoint() (sshtunnel.Endpoint, bool) {
	if t.forwarder == nil {
		return sshtunnel.Endpoint{}, false
	}

	return t.forwarder.GetLocalEndpoint(), true
}

func (t *Tunnel) Close() {
	if t.forwarder == nil {
		return
	}

	t.forwarder.Close()
}
