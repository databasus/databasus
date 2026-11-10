package app

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"databasus-backend/internal/lib/testwait"
)

func serveTestApp(t *testing.T, application *App) (string, context.CancelFunc, <-chan error) {
	t.Helper()

	addresses := make(chan string, 1)
	application.logger = slog.New(slog.NewJSONHandler(serverAddressWriter{addresses: addresses}, nil))
	serveContext, stopServing := context.WithCancel(t.Context())
	serveErrors := make(chan error, 1)
	serveDone := make(chan struct{})

	go func() {
		serveErrors <- application.Serve(serveContext, "127.0.0.1:0")
		close(serveDone)
	}()
	t.Cleanup(func() {
		stopServing()

		select {
		case <-serveDone:
		case <-time.After(10 * time.Second):
			t.Error("application did not finish shutdown")
		}
	})

	select {
	case address := <-addresses:
		return address, stopServing, serveErrors
	case err := <-serveErrors:
		t.Fatalf("application stopped before listening: %v", err)
	case <-time.After(testwait.DefaultLimit):
		t.Fatal("application did not start listening")
	}

	return "", stopServing, serveErrors
}

type serverAddressWriter struct {
	addresses chan<- string
}

func (w serverAddressWriter) Write(record []byte) (int, error) {
	var logRecord struct {
		Message string `json:"msg"`
		Address string `json:"address"`
	}
	if err := json.Unmarshal(record, &logRecord); err != nil {
		return 0, err
	}

	if logRecord.Message == "http server listening" {
		w.addresses <- logRecord.Address
	}

	return len(record), nil
}
