package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"
)

const (
	logFileName    = "databasus-verification.log"
	oldLogFileName = "databasus-verification.log.old"
	maxLogFileSize = 5 * 1024 * 1024 // 5MB
)

type rotatingWriter struct {
	mu          sync.Mutex
	file        *os.File
	currentSize int64
	maxSize     int64
	logPath     string
	oldLogPath  string
}

func (w *rotatingWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.currentSize+int64(len(p)) > w.maxSize {
		if err := w.rotate(); err != nil {
			return 0, fmt.Errorf("failed to rotate log file: %w", err)
		}
	}

	n, err := w.file.Write(p)
	w.currentSize += int64(n)

	return n, err
}

func New() (*slog.Logger, io.Closer, error) {
	writer, closer, err := buildWriter()
	log := slog.New(slog.NewTextHandler(writer, &slog.HandlerOptions{
		Level: slog.LevelInfo,
		ReplaceAttr: func(_ []string, attribute slog.Attr) slog.Attr {
			if attribute.Key == slog.TimeKey {
				attribute.Value = slog.StringValue(time.Now().Format("2006/01/02 15:04:05"))
			}
			if attribute.Key == slog.LevelKey {
				return slog.Attr{}
			}
			return attribute
		},
	}))
	return log, closer, err
}

func (w *rotatingWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.file.Close()
}

func buildWriter() (io.Writer, io.Closer, error) {
	f, err := os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return os.Stdout, nil, fmt.Errorf("failed to open %s for logging: %w", logFileName, err)
	}

	var currentSize int64
	if info, err := f.Stat(); err == nil {
		currentSize = info.Size()
	}

	rw := &rotatingWriter{
		file:        f,
		currentSize: currentSize,
		maxSize:     maxLogFileSize,
		logPath:     logFileName,
		oldLogPath:  oldLogFileName,
	}

	return io.MultiWriter(os.Stdout, rw), rw, nil
}

func (w *rotatingWriter) rotate() error {
	if err := w.file.Close(); err != nil {
		return fmt.Errorf("failed to close %s: %w", w.logPath, err)
	}

	if err := os.Remove(w.oldLogPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove %s: %w", w.oldLogPath, err)
	}

	if err := os.Rename(w.logPath, w.oldLogPath); err != nil {
		return fmt.Errorf("failed to rename %s to %s: %w", w.logPath, w.oldLogPath, err)
	}

	f, err := os.OpenFile(w.logPath, os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("failed to create new %s: %w", w.logPath, err)
	}

	w.file = f
	w.currentSize = 0

	return nil
}
