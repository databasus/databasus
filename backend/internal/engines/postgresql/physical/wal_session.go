package physical

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgproto3"

	"databasus-backend/internal/lib/background"
)

type walSession struct {
	directory string
	listener  net.Listener
	upstream  *pgconn.HijackedConn
	cancel    context.CancelFunc
	runs      *background.Group
}

func (s *Source) openWalSession(ctx context.Context, spec WalReceiverSpec) (*walSession, error) {
	if spec.ExpectedSystemIdentifier == "" {
		return nil, ErrSourceIdentityMissing
	}
	connection, err := s.connectReplication(ctx, spec.ApplicationName)
	if err != nil {
		return nil, fmt.Errorf("connect WAL source: %w", err)
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		_ = connection.Close(closeCtx)
	}()
	results, err := connection.Exec(ctx, "IDENTIFY_SYSTEM").ReadAll()
	if err != nil {
		return nil, fmt.Errorf("identify WAL source: %w", err)
	}
	row, err := getReplicationResultRow(results, 4)
	if err != nil {
		return nil, err
	}
	if err := verifySourceIdentity(
		sourceIdentityComparison{Expected: spec.ExpectedSystemIdentifier, Actual: string(row[0])},
	); err != nil {
		return nil, err
	}
	if err := connection.SyncConn(ctx); err != nil {
		return nil, fmt.Errorf("synchronize verified WAL connection: %w", err)
	}
	upstream, err := connection.Hijack()
	if err != nil {
		return nil, fmt.Errorf("transfer verified WAL connection: %w", err)
	}
	directory, err := os.MkdirTemp("", "db-wal-")
	if err != nil {
		_ = upstream.Conn.Close()
		return nil, err
	}
	listener, err := (&net.ListenConfig{}).Listen(ctx, "unix", filepath.Join(directory, ".s.PGSQL.5432"))
	if err != nil {
		_ = upstream.Conn.Close()
		_ = os.RemoveAll(directory)
		return nil, err
	}
	sessionCtx, cancel := context.WithCancel(ctx)
	session := &walSession{
		directory: directory,
		listener:  listener,
		upstream:  upstream,
		cancel:    cancel,
		runs:      background.New(background.Deps{Logger: spec.Logger}),
	}
	session.runs.Go(
		sessionCtx,
		"verified_wal_session",
		func(ctx context.Context) error { return session.serve(ctx, spec) },
	)
	return session, nil
}

func (s *walSession) Close(ctx context.Context) {
	s.cancel()
	_ = s.listener.Close()
	_ = s.upstream.Conn.Close()
	_ = s.runs.Wait(context.WithoutCancel(ctx))
	_ = os.RemoveAll(s.directory)
}

func (s *walSession) serve(ctx context.Context, spec WalReceiverSpec) error {
	stopClosing := context.AfterFunc(ctx, func() { _ = s.listener.Close(); _ = s.upstream.Conn.Close() })
	defer stopClosing()
	downstream, err := s.listener.Accept()
	if err != nil {
		return nil
	}
	_ = s.listener.Close()
	defer func() { _ = downstream.Close() }()
	defer func() { _ = s.upstream.Conn.Close() }()
	stopDownstream := context.AfterFunc(ctx, func() { _ = downstream.Close() })
	defer stopDownstream()
	// No second client can attach to the authenticated upstream session.
	if err := s.startLocalClient(downstream); err != nil {
		return fmt.Errorf("initialize local WAL client: %w", err)
	}
	copied := make(chan struct{})
	s.runs.Go(ctx, "verified_wal_server_messages", func(context.Context) error {
		defer close(copied)
		defer func() { _ = downstream.Close() }()
		defer func() { _ = s.upstream.Conn.Close() }()
		for {
			packet, err := readWalProtocolPacket(s.upstream.Conn, false)
			if err != nil {
				return nil
			}
			if packet[0] == 'W' && spec.OnStarted != nil {
				spec.OnStarted()
			}
			if _, err := downstream.Write(packet); err != nil {
				return nil
			}
		}
	})
	_, _ = io.Copy(s.upstream.Conn, downstream)
	_ = s.upstream.Conn.Close()
	_ = downstream.Close()
	<-copied
	return nil
}

func (s *walSession) startLocalClient(downstream net.Conn) error {
	packet, err := readWalProtocolPacket(downstream, true)
	if err != nil {
		return err
	}
	var startup pgproto3.StartupMessage
	if err := startup.Decode(packet[4:]); err != nil {
		return err
	}
	if startup.Parameters["replication"] != "true" {
		return fmt.Errorf("local WAL client must use physical replication")
	}
	messages := []pgproto3.BackendMessage{&pgproto3.AuthenticationOk{}}
	for name, value := range s.upstream.ParameterStatuses {
		messages = append(messages, &pgproto3.ParameterStatus{Name: name, Value: value})
	}
	messages = append(
		messages,
		&pgproto3.BackendKeyData{ProcessID: s.upstream.PID, SecretKey: s.upstream.SecretKey},
		&pgproto3.ReadyForQuery{TxStatus: s.upstream.TxStatus},
	)
	for _, message := range messages {
		encoded, err := message.Encode(nil)
		if err != nil {
			return err
		}
		if _, err := downstream.Write(encoded); err != nil {
			return err
		}
	}
	return nil
}

func readWalProtocolPacket(connection net.Conn, isStartup bool) ([]byte, error) {
	headerSize := 5
	if isStartup {
		headerSize = 4
	}
	header := make([]byte, headerSize)
	if _, err := io.ReadFull(connection, header); err != nil {
		return nil, err
	}
	lengthOffset := headerSize - 4
	length := int(binary.BigEndian.Uint32(header[lengthOffset:]))
	if length < 4 || length > 64*1024*1024 {
		return nil, fmt.Errorf("invalid WAL protocol message length %d", length)
	}
	packet := make([]byte, length+lengthOffset)
	copy(packet, header)
	_, err := io.ReadFull(connection, packet[headerSize:])
	return packet, err
}
