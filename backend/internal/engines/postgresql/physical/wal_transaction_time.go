package physical

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/lib/process"
)

type WalTransactionTimeSpec struct {
	ClientTools pgshared.ClientTools
	Version     pgshared.Version
	LocalPath   string
	EndLSN      walmath.LSN
}

func GetWalTransactionTime(ctx context.Context, spec WalTransactionTimeSpec) (*time.Time, error) {
	transactions := &walTransactionTimeOutput{}
	decodeErr := transactions.read(ctx, spec)
	if transactions.candidateLSN != nil {
		verification := &walTransactionTimeOutput{requiredLSN: transactions.candidateLSN}
		verificationErr := verification.read(ctx, spec)
		if verificationErr == nil && verification.latest != nil &&
			(transactions.latest == nil || verification.latest.After(*transactions.latest)) {
			transactions.latest = verification.latest
		}

		decodeErr = errors.Join(decodeErr, verificationErr)
	}

	return transactions.latest, decodeErr
}

type walTransactionTimeOutput struct {
	line                     []byte
	latest                   *time.Time
	candidateLSN             *walmath.LSN
	requiredLSN              *walmath.LSN
	hasUntrustedDescriptions bool
	hasReadFirstLine         bool
}

func (output *walTransactionTimeOutput) Write(body []byte) (int, error) {
	for _, character := range body {
		if character == '\n' {
			if err := output.readLine(); err != nil {
				return 0, err
			}

			output.line = output.line[:0]
		} else if len(output.line) < 512 {
			output.line = append(output.line, character)
		}
	}

	return len(body), nil
}

func (output *walTransactionTimeOutput) read(ctx context.Context, spec WalTransactionTimeSpec) error {
	var diagnostics bytes.Buffer
	args := []string{"--rmgr=Transaction", "--end=" + spec.EndLSN.String()}
	if output.requiredLSN != nil {
		args = append(args, "--start="+output.requiredLSN.String(), "--limit=1")
	}
	args = append(args, filepath.Base(spec.LocalPath))

	err := process.Run(ctx, process.Spec{
		Path: spec.ClientTools.GetExecutable(spec.Version, pgshared.ExecutablePgWaldump),
		Args: args,
		Dir:  filepath.Dir(spec.LocalPath), Env: []string{"LC_ALL=C", "TZ=UTC"},
		Stdout: output, Stderr: &diagnostics,
	})
	if err != nil {
		return fmt.Errorf(
			"read wal transaction timestamps: %w: %s",
			err,
			strings.TrimSpace(diagnostics.String()),
		)
	}

	return nil
}

func (output *walTransactionTimeOutput) readLine() error {
	if output.requiredLSN != nil && output.hasReadFirstLine {
		return nil
	}

	output.hasReadFirstLine = true
	header, description, hasDescription := strings.Cut(string(output.line), ", desc: ")
	if !hasDescription || !strings.HasPrefix(header, "rmgr: Transaction ") {
		return nil
	}

	_, location, hasLocation := strings.Cut(header, ", lsn: ")
	location, _, _ = strings.Cut(location, ", prev ")
	if !hasLocation {
		return nil
	}

	locationLSN, err := walmath.ParseLSN(location)
	if err != nil {
		return nil
	}

	if output.requiredLSN != nil && locationLSN != *output.requiredLSN {
		return fmt.Errorf("wal decoder did not start at the requested record")
	}

	kind, timestamp, _ := strings.Cut(description, " ")
	switch kind {
	case "PREPARE":
		output.hasUntrustedDescriptions = true
		if output.requiredLSN != nil {
			return fmt.Errorf("wal record is not a completed transaction")
		}

		return nil
	case "COMMIT_PREPARED", "ABORT_PREPARED":
		_, timestamp, _ = strings.Cut(timestamp, ": ")
	case "COMMIT", "ABORT":
	default:
		return nil
	}

	// PREPARE descriptions contain unescaped GIDs, including forged record headers.
	if output.hasUntrustedDescriptions {
		output.candidateLSN = &locationLSN
		return nil
	}

	fields := strings.Fields(timestamp)
	if len(fields) < 3 {
		return fmt.Errorf("incomplete wal transaction timestamp")
	}

	fields[2] = strings.TrimSuffix(fields[2], ";")
	transactionAt, err := time.Parse("2006-01-02 15:04:05.999999 MST", strings.Join(fields[:3], " "))
	if err != nil {
		return fmt.Errorf("parse wal transaction timestamp: %w", err)
	}

	if output.latest == nil || transactionAt.After(*output.latest) {
		output.latest = new(transactionAt.UTC())
	}

	return nil
}
