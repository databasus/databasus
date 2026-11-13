package containers

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgproto3"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/testwait"
)

func TestPostgresProxy(t *testing.T) {
	t.Parallel()

	authority, err := NewTestCertificateAuthority()
	require.NoError(t, err)
	certificate, err := authority.IssueServerCertificate(
		ServerCertificateSpec{DNSNames: []string{"localhost"}, IPAddresses: []string{loopbackIP}},
	)
	require.NoError(t, err)
	source := StartPhysicalPostgres(t, "postgres:17", WithServerCertificate(certificate))

	t.Run("returns after cancellation interrupts TLS startup", func(t *testing.T) {
		t.Parallel()

		proxy := StartPostgresProxy(t, source, WithProxyTLS(authority, certificate))
		client, server := net.Pipe()
		defer func() { _ = client.Close(); _ = server.Close() }()
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		done := make(chan error, 1)

		go func() { done <- proxy.forward(ctx, server, 0) }()
		_, err := client.Write([]byte{0, 0, 0, 8, 4, 210, 22, 47})
		require.NoError(t, err)
		response := make([]byte, 1)
		_, err = io.ReadFull(client, response)
		require.NoError(t, err)
		require.Equal(t, []byte{'S'}, response)

		cancel()

		testwait.Until(t, "the interrupted TLS startup returns", func(context.Context) error {
			select {
			case err := <-done:
				require.Error(t, err)
				return nil
			default:
				return fmt.Errorf("TLS startup is still running")
			}
		})
	})

	t.Run("returns after cancellation while the upstream withholds its SSL response", func(t *testing.T) {
		t.Parallel()

		listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP(loopbackIP)})
		require.NoError(t, err)
		defer func() { _ = listener.Close() }()
		require.NoError(t, listener.SetDeadline(time.Now().Add(testwait.DefaultLimit)))
		sourceAddress, hasTCPAddress := listener.Addr().(*net.TCPAddr)
		require.True(t, hasTCPAddress)
		destination := Endpoint{Host: loopbackIP, Port: sourceAddress.Port}
		proxy := StartPostgresProxy(t, destination, WithProxyTLS(authority, certificate))
		client, server := net.Pipe()
		defer func() { _ = client.Close(); _ = server.Close() }()
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		done := make(chan error, 1)

		proxy.runs.Go(ctx, "stalled_upstream", func(ctx context.Context) error {
			done <- proxy.forward(ctx, server, 0)
			return nil
		})
		sslRequest := []byte{0, 0, 0, 8, 4, 210, 22, 47}
		_, err = client.Write(sslRequest)
		require.NoError(t, err)
		response := make([]byte, 1)
		_, err = io.ReadFull(client, response)
		require.NoError(t, err)
		require.Equal(t, []byte{'S'}, response)
		secured := tls.Client(client, authority.GetClientTLS(loopbackIP))
		require.NoError(t, secured.HandshakeContext(ctx))
		startup, err := (&pgproto3.StartupMessage{
			ProtocolVersion: pgproto3.ProtocolVersionNumber,
			Parameters:      map[string]string{"user": PostgresUsername, "database": "postgres"},
		}).Encode(nil)
		require.NoError(t, err)
		_, err = secured.Write(startup)
		require.NoError(t, err)

		upstream, err := listener.AcceptTCP()
		require.NoError(t, err)
		defer func() { _ = upstream.Close() }()
		require.NoError(t, upstream.SetReadDeadline(time.Now().Add(testwait.DefaultLimit)))
		request := make([]byte, len(sslRequest))
		_, err = io.ReadFull(upstream, request)
		require.NoError(t, err)
		require.Equal(t, sslRequest, request)

		cancel()

		testwait.Within(5*time.Second).
			Until(t, "forwarding exits without an upstream SSL response", func(context.Context) error {
				select {
				case err := <-done:
					require.Error(t, err)
					return nil
				default:
					return fmt.Errorf("forwarding still waits for the upstream SSL response")
				}
			})
		require.NoError(t, upstream.SetReadDeadline(time.Now().Add(5*time.Second)))
		_, err = upstream.Read(response)
		require.ErrorIs(t, err, io.EOF, "cancellation must close the upstream socket")
	})

	t.Run("holds a TLS backup transfer at its first data when the gate is set", func(t *testing.T) {
		t.Parallel()

		proxy := StartPostgresProxy(t, source, WithProxyTLS(authority, certificate))
		gate := proxy.HoldTransfer(t)
		connection := connectThroughProxy(t, proxyConnection{Proxy: proxy, Authority: authority, IsReplication: true})
		connection.Frontend().SendQuery(&pgproto3.Query{String: "BASE_BACKUP (CHECKPOINT 'fast')"})
		require.NoError(t, connection.Frontend().Flush())

		var transfer PostgresProxyEvent
		testwait.Until(t, "the backup reaches the transfer gate", func(context.Context) error {
			select {
			case transfer = <-gate.Reached:
				return nil
			default:
				return fmt.Errorf("the transfer has not reached the gate")
			}
		})

		require.Positive(t, transfer.TransferredBytes)
		require.Positive(t, transfer.BackendPID)
		require.Contains(t, transfer.Command, "BASE_BACKUP")

		require.NoError(t, connection.Conn().SetReadDeadline(time.Now().Add(testwait.DefaultLimit)))
		for {
			message, err := connection.Frontend().Receive()
			require.NoError(t, err)
			copyData, hasData := message.(*pgproto3.CopyData)
			if hasData && len(copyData.Data) > 0 && copyData.Data[0] == 'd' {
				break
			}
		}

		require.NoError(t, connection.Conn().SetReadDeadline(time.Now().Add(time.Second)))
		_, err := connection.Frontend().Receive()
		var blockedRead net.Error
		require.ErrorAs(t, err, &blockedRead, "backup data must remain blocked before release")
		require.True(t, blockedRead.Timeout())

		gate.Release()
		require.NoError(t, connection.Conn().SetReadDeadline(time.Now().Add(testwait.DefaultLimit)))
		message, err := connection.Frontend().Receive()
		require.NoError(t, err, "backup data must resume after release")
		require.IsType(t, &pgproto3.CopyData{}, message)
	})

	t.Run("refuses TLS replication and keeps SQL when replication is blocked", func(t *testing.T) {
		t.Parallel()

		proxy := StartPostgresProxy(t, source, WithProxyTLS(authority, certificate))
		proxy.SetReplicationBlocked(true)

		_, err := pgconn.ConnectConfig(
			t.Context(),
			getProxyConfig(t, proxyConnection{Proxy: proxy, Authority: authority, IsReplication: true}),
		)
		connection := connectThroughProxy(t, proxyConnection{Proxy: proxy, Authority: authority})

		require.Error(t, err)
		require.NoError(t, connection.Exec(t.Context(), "SELECT 1").Close())
		requireProxyEvent(t, proxy, "unavailable")
	})

	t.Run("refuses every TLS connection when an outage is active", func(t *testing.T) {
		t.Parallel()

		proxy := StartPostgresProxy(t, source, WithProxyTLS(authority, certificate))
		proxy.SetUnavailable(true)

		_, err := pgconn.ConnectConfig(
			t.Context(),
			getProxyConfig(t, proxyConnection{Proxy: proxy, Authority: authority}),
		)

		require.Error(t, err)
		requireProxyEvent(t, proxy, "unavailable")
	})

	t.Run("refuses the upstream when its certificate comes from another authority", func(t *testing.T) {
		t.Parallel()

		foreignAuthority, err := NewTestCertificateAuthority()
		require.NoError(t, err)
		foreignCertificate, err := foreignAuthority.IssueServerCertificate(
			ServerCertificateSpec{DNSNames: []string{"localhost"}, IPAddresses: []string{loopbackIP}},
		)
		require.NoError(t, err)
		proxy := StartPostgresProxy(t, source, WithProxyTLS(foreignAuthority, foreignCertificate))

		_, err = pgconn.ConnectConfig(
			t.Context(),
			getProxyConfig(t, proxyConnection{Proxy: proxy, Authority: foreignAuthority}),
		)

		require.Error(t, err)
		requireProxyEvent(t, proxy, "refused")
	})
}

type proxyConnection struct {
	Proxy         *PostgresProxy
	Authority     *TestCertificateAuthority
	IsReplication bool
}

func connectThroughProxy(t *testing.T, spec proxyConnection) *pgconn.PgConn {
	t.Helper()

	connection, err := pgconn.ConnectConfig(t.Context(), getProxyConfig(t, spec))
	require.NoError(t, err)
	t.Cleanup(func() { _ = connection.Close(context.WithoutCancel(t.Context())) })

	return connection
}

func getProxyConfig(t *testing.T, spec proxyConnection) *pgconn.Config {
	t.Helper()

	config, err := pgconn.ParseConfig(fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=postgres sslmode=disable connect_timeout=5",
		spec.Proxy.Endpoint.Host,
		spec.Proxy.Endpoint.Port,
		PostgresUsername,
		PostgresPassword,
	))
	require.NoError(t, err)
	config.TLSConfig = spec.Authority.GetClientTLS(spec.Proxy.Endpoint.Host)
	config.Fallbacks = nil
	if spec.IsReplication {
		config.RuntimeParams["replication"] = "true"
	}

	return config
}

func requireProxyEvent(t *testing.T, proxy *PostgresProxy, kind string) {
	t.Helper()

	testwait.Until(t, "the proxy records a "+kind+" connection", func(context.Context) error {
		for _, event := range proxy.GetEvents() {
			if event.Kind == kind {
				return nil
			}
		}

		return fmt.Errorf("no %s event in %+v", kind, proxy.GetEvents())
	})
}
