package physical

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	containers "databasus-backend/internal/lib/testcontainers"
)

type replicationResponseProxy struct {
	endpoint          containers.Endpoint
	command           string
	ctx               context.Context
	cancel            context.CancelFunc
	isResponsePending atomic.Bool
	responseHeld      chan struct{}
	clientClosed      chan struct{}
	done              chan struct{}
	failure           chan error
}

type replicationProxySpec struct {
	Source  containers.Endpoint
	Command string
}

type replicationProxyConnections struct {
	Client net.Conn
	Source net.Conn
}

func holdReplicationResponse(t *testing.T, spec replicationProxySpec) *replicationResponseProxy {
	t.Helper()

	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	host, portText, err := net.SplitHostPort(listener.Addr().String())
	require.NoError(t, err)
	port, err := strconv.Atoi(portText)
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(t.Context())
	proxy := &replicationResponseProxy{
		endpoint: containers.Endpoint{Host: host, Port: port}, command: spec.Command,
		ctx: ctx, cancel: cancel, responseHeld: make(chan struct{}), clientClosed: make(chan struct{}),
		done: make(chan struct{}), failure: make(chan error, 1),
	}
	t.Cleanup(func() {
		cancel()
		_ = listener.Close()
	})

	go func() {
		defer close(proxy.done)
		defer func() { _ = listener.Close() }()
		proxy.failure <- proxy.serve(listener, spec.Source)
	}()

	return proxy
}

func (proxy *replicationResponseProxy) serve(listener net.Listener, sourceEndpoint containers.Endpoint) error {
	client, err := listener.Accept()
	if err != nil {
		return err
	}

	_ = listener.Close()
	defer func() { _ = client.Close() }()
	closeClient := context.AfterFunc(proxy.ctx, func() { _ = client.Close() })
	defer closeClient()

	source, err := (&net.Dialer{}).DialContext(
		proxy.ctx,
		"tcp",
		net.JoinHostPort(sourceEndpoint.Host, strconv.Itoa(sourceEndpoint.Port)),
	)
	if err != nil {
		return err
	}
	defer func() { _ = source.Close() }()
	closeSource := context.AfterFunc(proxy.ctx, func() { _ = source.Close() })
	defer closeSource()

	connections := replicationProxyConnections{Client: client, Source: source}
	startup, err := readReplicationPacket(client, true)
	if err != nil {
		return err
	}

	if _, err := io.Copy(source, bytes.NewReader(startup)); err != nil {
		return err
	}

	responsesDone := make(chan struct{})
	go func() {
		defer close(responsesDone)
		_ = proxy.forwardResponses(connections)
		_ = client.Close()
	}()

	clientErr := proxy.forwardRequests(connections)
	_ = client.Close()
	close(proxy.clientClosed)
	proxy.cancel()
	_ = source.Close()
	<-responsesDone

	return clientErr
}

func (proxy *replicationResponseProxy) forwardRequests(connections replicationProxyConnections) error {
	for {
		packet, err := readReplicationPacket(connections.Client, false)
		if err != nil {
			return err
		}

		if packet[0] == 'Q' && string(packet[5:]) == proxy.command+"\x00" {
			proxy.isResponsePending.Store(true)
		}

		if _, err := io.Copy(connections.Source, bytes.NewReader(packet)); err != nil {
			return err
		}
	}
}

func (proxy *replicationResponseProxy) forwardResponses(connections replicationProxyConnections) error {
	for {
		packet, err := readReplicationPacket(connections.Source, false)
		if err != nil {
			return err
		}

		if proxy.isResponsePending.Load() {
			close(proxy.responseHeld)
			<-proxy.ctx.Done()
			return proxy.ctx.Err()
		}

		if _, err := io.Copy(connections.Client, bytes.NewReader(packet)); err != nil {
			return err
		}
	}
}

func readReplicationPacket(reader io.Reader, isStartup bool) ([]byte, error) {
	headerSize := 5
	if isStartup {
		headerSize = 4
	}
	header := make([]byte, headerSize)
	if _, err := io.ReadFull(reader, header); err != nil {
		return nil, err
	}

	packetSize := binary.BigEndian.Uint32(header[headerSize-4:])
	if packetSize < 4 || packetSize > 1024*1024 {
		return nil, fmt.Errorf("invalid replication proxy packet size: %d", packetSize)
	}

	body := make([]byte, int(packetSize)-4)
	if _, err := io.ReadFull(reader, body); err != nil {
		return nil, err
	}

	return append(header, body...), nil
}
