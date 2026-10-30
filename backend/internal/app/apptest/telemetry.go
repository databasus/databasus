package apptest

import (
	"net"
	"net/url"
)

type TelemetrySettings struct {
	ReceiverURL  string
	IdentityPath string
}

func WithTelemetry(spec TelemetrySettings) Option {
	parsed, err := url.Parse(spec.ReceiverURL)
	if err != nil || parsed.Scheme != "http" || !net.ParseIP(parsed.Hostname()).IsLoopback() {
		panic("telemetry tests require an explicit local HTTP receiver")
	}
	return func(applied *settings) {
		applied.isTelemetryEnabled = true
		applied.telemetryReceiverURL = spec.ReceiverURL
		applied.telemetryIdentityPath = spec.IdentityPath
	}
}

func WithoutTelemetry() Option {
	return func(applied *settings) { applied.isTelemetryEnabled = false }
}
