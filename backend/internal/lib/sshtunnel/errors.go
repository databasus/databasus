package sshtunnel

import "errors"

var (
	// Returned when a connection reaches the forwarder after Close: an expected shutdown race, not a
	// tunnel failure, so callers can tell it apart from a bastion that is genuinely unreachable.
	ErrForwarderClosed = errors.New("ssh tunnel forwarder is closed")
	ErrUnknownAuthType = errors.New("invalid SSH tunnel auth type")
)
