// Package httpserver holds the HTTP server and the middleware that knows nothing of the domain.
package httpserver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"
)

const (
	readHeaderTimeout = 10 * time.Second
	idleTimeout       = 120 * time.Second
)

type Server struct {
	httpServer *http.Server
	listener   net.Listener
}

// Listen opens the port before anything else starts, so a busy port fails startup instead of
// leaving a process that looks alive but answers nothing. The server sets no ReadTimeout or
// WriteTimeout: both cover the whole request or response and would cut long backup uploads and
// downloads.
func Listen(ctx context.Context, address string, handler http.Handler) (*Server, error) {
	var listenConfig net.ListenConfig

	listener, err := listenConfig.Listen(ctx, "tcp", address)
	if err != nil {
		return nil, fmt.Errorf("listen on %s: %w", address, err)
	}

	return &Server{
		httpServer: newHTTPServer(handler),
		listener:   listener,
	}, nil
}

func (s *Server) GetAddress() string {
	return s.listener.Addr().String()
}

// Serve blocks until Shutdown is called.
func (s *Server) Serve() error {
	err := s.httpServer.Serve(s.listener)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}

	return fmt.Errorf("serve http: %w", err)
}

// Shutdown stops accepting connections and waits for started requests until ctx ends.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func newHTTPServer(handler http.Handler) *http.Server {
	return &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		IdleTimeout:       idleTimeout,
	}
}
