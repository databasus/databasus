package physical

import (
	"context"
	"log/slog"

	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/lib/sshtunnel"
)

// Tunnel is a source reached through its SSH tunnel. It belongs to the operation that opened it,
// which closes it when it is done.
type Tunnel struct {
	source    *Source
	forwarder *sshtunnel.Forwarder
}

// OpenTunnel refuses the embedded internal cluster first. Without an SSH tunnel it returns the
// source itself, so callers never branch. Through a tunnel Host stays the real one, because
// sslmode=verify-full matches the certificate against it and libpq looks up .pgpass by host and
// port, while the connection goes to the local end of the tunnel.
func (s *Source) OpenTunnel(ctx context.Context, logger *slog.Logger) (*Tunnel, error) {
	if err := s.ValidateNotEmbeddedTarget(); err != nil {
		return nil, err
	}

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
	settingsThroughTunnel.HostAddr = localEndpoint.Host
	settingsThroughTunnel.Port = localEndpoint.Port
	settingsThroughTunnel.SSHTunnel = nil

	return &Tunnel{
		source:    &Source{settings: settingsThroughTunnel, generateUsername: s.generateUsername},
		forwarder: forwarder,
	}, nil
}

// ValidateNotEmbeddedTarget refuses Databasus's own internal cluster, unless it is reached through
// a bastion on another host.
func (s *Source) ValidateNotEmbeddedTarget() error {
	bastionHost := ""
	if s.settings.SSHTunnel != nil {
		bastionHost = s.settings.SSHTunnel.Host
	}

	return pgshared.ValidateNotEmbeddedTarget(pgshared.EmbeddedTargetSpec{
		Host:               s.settings.Host,
		Port:               s.settings.Port,
		DatabaseName:       "",
		IsPhysical:         true,
		IsSSHTunnelEnabled: s.settings.SSHTunnel != nil,
		SSHBastionHost:     bastionHost,
	})
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

// IsBastionReachable reports true without a tunnel: with no bastion between us and the cluster,
// nothing about the transport can be at fault, and the caller's failure is the source's own.
func (t *Tunnel) IsBastionReachable(ctx context.Context) bool {
	if t.forwarder == nil {
		return true
	}

	return t.forwarder.IsBastionReachable(ctx)
}

func (t *Tunnel) Close() {
	if t.forwarder == nil {
		return
	}

	t.forwarder.Close()
}
