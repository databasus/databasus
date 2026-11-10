package containers

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgproto3"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/background"
)

type PostgresProxyEvent struct {
	ConnectionID     int
	Destination      Endpoint
	IsReplication    bool
	Kind             string
	Command          string
	SystemIdentifier string
}

type PostgresProxyCommandBarrier struct {
	Prefix       string
	Reached      chan PostgresProxyEvent
	Release      chan struct{}
	hasTriggered bool
}

type PostgresProxyRoute struct {
	SQLDestination         Endpoint
	ReplicationDestination Endpoint
}

type PostgresProxy struct {
	responseFragmentSize int
	commandBarrier       *PostgresProxyCommandBarrier

	Endpoint               Endpoint
	mu                     sync.Mutex
	sqlDestination         Endpoint
	replicationDestination Endpoint
	scriptedDestinations   []Endpoint
	events                 []PostgresProxyEvent
	connections            map[int]net.Conn
	nextConnectionID       int
	runs                   *background.Group
}

func StartPostgresProxy(t *testing.T, source Endpoint) *PostgresProxy {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	host, portText, err := net.SplitHostPort(listener.Addr().String())
	require.NoError(t, err)
	port, err := strconv.Atoi(portText)
	require.NoError(t, err)
	proxy := &PostgresProxy{
		Endpoint:       Endpoint{Host: host, Port: port},
		sqlDestination: source, replicationDestination: source,
		connections: make(map[int]net.Conn),
		runs:        background.New(background.Deps{Logger: slog.New(slog.DiscardHandler)}),
	}
	t.Cleanup(func() {
		cancel()
		_ = listener.Close()
		proxy.CutConnections()
		_ = proxy.runs.Wait(context.WithoutCancel(ctx))
	})
	proxy.runs.Go(ctx, "postgres_proxy", func(ctx context.Context) error {
		for {
			client, acceptErr := listener.Accept()
			if acceptErr != nil {
				return nil
			}
			proxy.mu.Lock()
			proxy.nextConnectionID++
			connectionID := proxy.nextConnectionID
			proxy.connections[connectionID] = client
			proxy.mu.Unlock()
			proxy.runs.Go(ctx, "postgres_proxy_connection", func(ctx context.Context) error {
				defer func() { _ = client.Close() }()
				defer func() {
					proxy.mu.Lock()
					delete(proxy.connections, connectionID)
					proxy.mu.Unlock()
				}()
				return proxy.forward(ctx, client, connectionID)
			})
		}
	})
	return proxy
}

func (p *PostgresProxy) Route(route PostgresProxyRoute) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.sqlDestination = route.SQLDestination
	p.replicationDestination = route.ReplicationDestination
}

func (p *PostgresProxy) SetResponseFragmentSize(size int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.responseFragmentSize = size
}

func (p *PostgresProxy) CutConnection(connectionID int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if connection := p.connections[connectionID]; connection != nil {
		_ = connection.Close()
	}
}

func (p *PostgresProxy) HoldCommand(prefix string) *PostgresProxyCommandBarrier {
	p.mu.Lock()
	defer p.mu.Unlock()
	barrier := &PostgresProxyCommandBarrier{
		Prefix:  prefix,
		Reached: make(chan PostgresProxyEvent, 1),
		Release: make(chan struct{}),
	}
	p.commandBarrier = barrier
	return barrier
}

func (p *PostgresProxy) RouteNext(destinations ...Endpoint) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.scriptedDestinations = append(p.scriptedDestinations, destinations...)
}

func (p *PostgresProxy) CutConnections() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, connection := range p.connections {
		_ = connection.Close()
	}
}

func (p *PostgresProxy) GetEvents() []PostgresProxyEvent {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]PostgresProxyEvent(nil), p.events...)
}

func (p *PostgresProxy) record(event PostgresProxyEvent) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.events = append(p.events, event)
}

func (p *PostgresProxy) forward(ctx context.Context, client net.Conn, connectionID int) error {
	stopClosingClient := context.AfterFunc(ctx, func() { _ = client.Close() })
	defer stopClosingClient()
	startupPacket, err := readPostgresProxyPacket(client, true)
	if err != nil {
		return err
	}
	if len(startupPacket) == 8 && binary.BigEndian.Uint32(startupPacket[4:]) == 80877103 {
		if _, err := client.Write([]byte("N")); err != nil {
			return err
		}
		startupPacket, err = readPostgresProxyPacket(client, true)
		if err != nil {
			return err
		}
	}
	var startup pgproto3.StartupMessage
	if err := startup.Decode(startupPacket[4:]); err != nil {
		return err
	}
	isReplication := startup.Parameters["replication"] != "" && startup.Parameters["replication"] != "false"
	p.mu.Lock()
	destination := p.sqlDestination
	if isReplication {
		destination = p.replicationDestination
	}
	if len(p.scriptedDestinations) > 0 {
		destination = p.scriptedDestinations[0]
		p.scriptedDestinations = p.scriptedDestinations[1:]
	}
	p.mu.Unlock()
	event := PostgresProxyEvent{ConnectionID: connectionID, Destination: destination, IsReplication: isReplication}
	defer func() { closedEvent := event; closedEvent.Kind = "closed"; p.record(closedEvent) }()
	source, err := (&net.Dialer{}).DialContext(
		ctx,
		"tcp",
		net.JoinHostPort(destination.Host, strconv.Itoa(destination.Port)),
	)
	if err != nil {
		return err
	}
	defer func() { _ = source.Close() }()
	closeSource := context.AfterFunc(ctx, func() { _ = source.Close(); _ = client.Close() })
	defer closeSource()
	if _, err := source.Write(startupPacket); err != nil {
		return err
	}
	var queryMu sync.Mutex
	lastQuery := ""
	responsesDone := make(chan struct{})
	p.runs.Go(ctx, "postgres_proxy_responses", func(context.Context) error {
		defer close(responsesDone)
		defer func() { _ = client.Close() }()
		isStreaming := false
		for {
			packet, readErr := readPostgresProxyPacket(source, false)
			if readErr != nil {
				return nil
			}
			responseEvent := event
			switch packet[0] {
			case 'R':
				if len(packet) >= 9 && binary.BigEndian.Uint32(packet[5:9]) == 0 {
					responseEvent.Kind = "authenticated"
					p.record(responseEvent)
				}
			case 'D':
				queryMu.Lock()
				command := lastQuery
				queryMu.Unlock()
				if command == "IDENTIFY_SYSTEM" ||
					command == "SELECT system_identifier::text FROM pg_control_system()" {
					var row pgproto3.DataRow
					if row.Decode(packet[5:]) == nil && (len(row.Values) == 4 || len(row.Values) == 1) {
						responseEvent.Kind = "identified"
						responseEvent.SystemIdentifier = string(row.Values[0])
						p.record(responseEvent)
					}
				}
			case 'W':
				isStreaming = true
				responseEvent.Kind = "streaming"
				p.record(responseEvent)
			}
			p.mu.Lock()
			fragmentSize := p.responseFragmentSize
			if isStreaming {
				fragmentSize = 0
			}
			p.mu.Unlock()
			for len(packet) > 0 {
				writeSize := len(packet)
				if fragmentSize > 0 && writeSize > fragmentSize {
					writeSize = fragmentSize
				}
				written, err := client.Write(packet[:writeSize])
				if err != nil {
					return nil
				}
				packet = packet[written:]
			}
		}
	})
	for {
		packet, readErr := readPostgresProxyPacket(client, false)
		if readErr != nil {
			break
		}
		command := ""
		switch packet[0] {
		case 'Q':
			command = strings.TrimSpace(strings.TrimRight(string(packet[5:]), "\x00;"))
		case 'P':
			var parsed pgproto3.Parse
			if err := parsed.Decode(packet[5:]); err != nil {
				return err
			}
			command = strings.TrimSpace(parsed.Query)
		}
		if command != "" {
			queryMu.Lock()
			lastQuery = command
			queryMu.Unlock()
			commandEvent := event
			commandEvent.Kind = "command"
			commandEvent.Command = command
			p.record(commandEvent)
			p.mu.Lock()
			barrier := p.commandBarrier
			shouldHold := barrier != nil && !barrier.hasTriggered && strings.HasPrefix(command, barrier.Prefix)
			if shouldHold {
				barrier.hasTriggered = true
			}
			p.mu.Unlock()
			if shouldHold {
				barrier.Reached <- commandEvent
				select {
				case <-barrier.Release:
				case <-ctx.Done():
					return nil
				}
			}
		}
		if _, err := source.Write(packet); err != nil {
			break
		}
	}
	_ = source.Close()
	<-responsesDone
	return nil
}

func readPostgresProxyPacket(reader io.Reader, isStartup bool) ([]byte, error) {
	headerSize := 5
	if isStartup {
		headerSize = 4
	}
	header := make([]byte, headerSize)
	if _, err := io.ReadFull(reader, header); err != nil {
		return nil, err
	}
	size := binary.BigEndian.Uint32(header[headerSize-4:])
	if size < 4 || size > 64*1024*1024 {
		return nil, fmt.Errorf("invalid PostgreSQL packet size: %d", size)
	}
	body := make([]byte, int(size)-4)
	if _, err := io.ReadFull(reader, body); err != nil {
		return nil, err
	}
	return append(header, body...), nil
}
