package mailer

import (
	"bufio"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"io"
	"math/big"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	fakeServerUsername = "mailer"
	fakeServerPassword = "correct-horse-battery-staple"
)

type fakeServerOptions struct {
	implicitTLSConfig     *tls.Config
	startTLSConfig        *tls.Config
	authMechanisms        []string
	loginPrompts          [2]string
	isSilentAfterGreeting bool
}

type fakeServer struct {
	host          string
	port          int
	options       fakeServerOptions
	mutex         sync.Mutex
	commands      []string
	messages      []string
	sessionsEnded chan struct{}
}

func startFakeServer(t *testing.T, options fakeServerOptions) *fakeServer {
	t.Helper()

	if options.loginPrompts == [2]string{} {
		options.loginPrompts = [2]string{"Username:", "Password:"}
	}

	var listenConfig net.ListenConfig

	listener, err := listenConfig.Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = listener.Close() })

	if options.implicitTLSConfig != nil {
		listener = tls.NewListener(listener, options.implicitTLSConfig)
	}

	listenerAddress, isTCPAddress := listener.Addr().(*net.TCPAddr)
	require.True(t, isTCPAddress)
	server := &fakeServer{
		host:          listenerAddress.IP.String(),
		port:          listenerAddress.Port,
		options:       options,
		sessionsEnded: make(chan struct{}, 16),
	}

	go func() {
		for {
			conn, acceptErr := listener.Accept()
			if acceptErr != nil {
				return
			}

			go server.serve(t.Context(), conn)
		}
	}()

	return server
}

func (s *fakeServer) getConnection(security Security) Connection {
	return Connection{
		Host:                 s.host,
		Port:                 s.port,
		Security:             security,
		Username:             fakeServerUsername,
		Password:             fakeServerPassword,
		HeloName:             "backup.example.com",
		IsInsecureSkipVerify: true,
	}
}

func (s *fakeServer) waitForSessionEnd(t *testing.T) {
	t.Helper()

	select {
	case <-s.sessionsEnded:
	case <-time.After(5 * time.Second):
		t.Fatal("fake SMTP session did not end")
	}
}

func (s *fakeServer) getCommands() []string {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return append([]string(nil), s.commands...)
}

func (s *fakeServer) getMessages() []string {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return append([]string(nil), s.messages...)
}

func (s *fakeServer) getCommandVerbs() []string {
	var verbs []string
	for _, command := range s.getCommands() {
		verbs = append(verbs, strings.ToUpper(strings.Fields(command)[0]))
	}

	return verbs
}

func (s *fakeServer) record(command string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.commands = append(s.commands, command)
}

func (s *fakeServer) serve(ctx context.Context, conn net.Conn) {
	defer func() {
		_ = conn.Close()
		s.sessionsEnded <- struct{}{}
	}()

	if _, err := io.WriteString(conn, "220 fake ESMTP\r\n"); err != nil {
		return
	}

	if s.options.isSilentAfterGreeting {
		_, _ = io.Copy(io.Discard, conn)
		return
	}

	reader := bufio.NewReader(conn)
	isEncrypted := s.options.implicitTLSConfig != nil

	for {
		command, err := readLine(reader)
		if err != nil {
			return
		}

		s.record(command)

		fields := strings.Fields(command)
		if len(fields) == 0 {
			reply(conn, "500 empty command")
			continue
		}

		switch strings.ToUpper(fields[0]) {
		case "EHLO":
			s.replyToEhlo(conn, isEncrypted)
		case "STARTTLS":
			reply(conn, "220 ready to start TLS")

			tlsConn := tls.Server(conn, s.options.startTLSConfig)
			if err := tlsConn.HandshakeContext(ctx); err != nil {
				return
			}

			conn = tlsConn
			reader = bufio.NewReader(conn)
			isEncrypted = true
		case "AUTH":
			s.handleAuth(conn, reader, fields)
		case "DATA":
			reply(conn, "354 end with <CR><LF>.<CR><LF>")

			message, err := readMessageData(reader)
			if err != nil {
				return
			}

			s.mutex.Lock()
			s.messages = append(s.messages, message)
			s.mutex.Unlock()

			reply(conn, "250 queued")
		case "QUIT":
			reply(conn, "221 bye")
			return
		default:
			reply(conn, "250 OK")
		}
	}
}

func (s *fakeServer) replyToEhlo(conn net.Conn, isEncrypted bool) {
	extensions := []string{"fake"}
	if s.options.startTLSConfig != nil && !isEncrypted {
		extensions = append(extensions, "STARTTLS")
	}

	if len(s.options.authMechanisms) > 0 {
		extensions = append(extensions, "AUTH "+strings.Join(s.options.authMechanisms, " "))
	}

	for index, extension := range extensions {
		separator := "-"
		if index == len(extensions)-1 {
			separator = " "
		}

		reply(conn, "250"+separator+extension)
	}
}

func (s *fakeServer) handleAuth(conn net.Conn, reader *bufio.Reader, fields []string) {
	var password string

	switch strings.ToUpper(fields[1]) {
	case mechanismPlain:
		encodedPayload := ""
		if len(fields) > 2 {
			encodedPayload = fields[2]
		}

		payload, _ := base64.StdEncoding.DecodeString(encodedPayload)
		payloadParts := strings.Split(string(payload), "\x00")
		password = payloadParts[len(payloadParts)-1]
	case mechanismLogin:
		for index, prompt := range s.options.loginPrompts {
			reply(conn, "334 "+base64.StdEncoding.EncodeToString([]byte(prompt)))

			answer, err := readLine(reader)
			if err != nil {
				return
			}

			if answer == "*" {
				reply(conn, "501 authentication cancelled")
				return
			}

			if index == 1 {
				decodedPassword, _ := base64.StdEncoding.DecodeString(answer)
				password = string(decodedPassword)
			}
		}
	default:
		reply(conn, "504 unrecognized authentication type")
		return
	}

	if password != fakeServerPassword {
		reply(conn, "535 5.7.8 Authentication credentials invalid")
		return
	}

	reply(conn, "235 2.7.0 Authentication successful")
}

func readLine(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')

	return strings.TrimRight(line, "\r\n"), err
}

func readMessageData(reader *bufio.Reader) (string, error) {
	var message strings.Builder

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}

		if line == ".\r\n" {
			return message.String(), nil
		}

		message.WriteString(strings.TrimPrefix(line, "."))
	}
}

func reply(conn net.Conn, line string) {
	_, _ = io.WriteString(conn, line+"\r\n")
}

func newSelfSignedTLSConfig(t *testing.T) *tls.Config {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "127.0.0.1"},
		NotBefore:    time.Now().UTC().Add(-time.Hour),
		NotAfter:     time.Now().UTC().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1"), net.IPv6loopback},
		IsCA:         true,
	}

	certificate, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	require.NoError(t, err)

	return &tls.Config{
		Certificates: []tls.Certificate{{Certificate: [][]byte{certificate}, PrivateKey: key}},
	}
}
