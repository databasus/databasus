package walstreaming_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	containers "databasus-backend/internal/lib/testcontainers"
)

type historyAcquisitionBarrier struct {
	endpoint    containers.Endpoint
	started     chan struct{}
	released    chan struct{}
	releaseOnce sync.Once
	isHeld      atomic.Bool
}

func holdNextHistoryAcquisition(t *testing.T, source containers.Endpoint) *historyAcquisitionBarrier {
	t.Helper()

	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	host, portText, err := net.SplitHostPort(listener.Addr().String())
	require.NoError(t, err)
	port, err := strconv.Atoi(portText)
	require.NoError(t, err)
	barrier := &historyAcquisitionBarrier{
		endpoint: containers.Endpoint{Host: host, Port: port},
		started:  make(chan struct{}), released: make(chan struct{}),
	}
	ctx, cancel := context.WithCancel(t.Context())
	closeListener := context.AfterFunc(ctx, func() { _ = listener.Close() })
	var connections sync.WaitGroup

	connections.Go(func() {
		for {
			client, acceptErr := listener.Accept()
			if acceptErr != nil {
				return
			}

			connections.Go(func() {
				_ = barrier.forward(ctx, client, source)
			})
		}
	})

	t.Cleanup(func() {
		cancel()
		_ = listener.Close()
		closeListener()
		barrier.Release()
		connections.Wait()
	})

	return barrier
}

func (barrier *historyAcquisitionBarrier) Release() {
	barrier.releaseOnce.Do(func() { close(barrier.released) })
}

func (barrier *historyAcquisitionBarrier) forward(
	ctx context.Context,
	client net.Conn,
	endpoint containers.Endpoint,
) error {
	defer func() { _ = client.Close() }()
	closeClient := context.AfterFunc(ctx, func() { _ = client.Close() })
	defer closeClient()

	startup, err := readHistoryProxyPacket(client, true)
	if err != nil {
		return err
	}

	for len(startup) == 8 {
		if _, err := client.Write([]byte{'N'}); err != nil {
			return err
		}

		startup, err = readHistoryProxyPacket(client, true)
		if err != nil {
			return err
		}
	}

	source, err := (&net.Dialer{}).DialContext(ctx, "tcp", net.JoinHostPort(endpoint.Host, strconv.Itoa(endpoint.Port)))
	if err != nil {
		return err
	}
	defer func() { _ = source.Close() }()
	closeSource := context.AfterFunc(ctx, func() { _ = source.Close() })
	defer closeSource()

	if _, err := io.Copy(source, bytes.NewReader(startup)); err != nil {
		return err
	}

	responsesDone := make(chan struct{})
	go func() {
		defer close(responsesDone)
		_, _ = io.Copy(client, source)
		_ = client.Close()
	}()
	defer func() {
		_ = source.Close()
		_ = client.Close()
		<-responsesDone
	}()

	isReceiver := bytes.Contains(startup, []byte("databasus_wal_receiver_"))
	for {
		packet, err := readHistoryProxyPacket(client, false)
		if err != nil {
			return err
		}

		if !isReceiver && packet[0] == 'Q' && string(packet[5:]) == "TIMELINE_HISTORY 2\x00" &&
			barrier.isHeld.CompareAndSwap(false, true) {
			close(barrier.started)
			select {
			case <-barrier.released:
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		if _, err := io.Copy(source, bytes.NewReader(packet)); err != nil {
			return err
		}
	}
}

func readHistoryProxyPacket(reader io.Reader, isStartup bool) ([]byte, error) {
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
		return nil, fmt.Errorf("invalid history proxy packet size: %d", packetSize)
	}

	body := make([]byte, int(packetSize)-4)
	if _, err := io.ReadFull(reader, body); err != nil {
		return nil, err
	}

	return append(header, body...), nil
}
