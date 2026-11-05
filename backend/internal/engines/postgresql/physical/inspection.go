package physical

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
)

type Inspection struct{ conn *pgx.Conn }

func (s *Source) OpenInspection(ctx context.Context) (*Inspection, error) {
	conn, err := s.connect(ctx)
	if err != nil {
		return nil, fmt.Errorf("open inspection: %w", err)
	}

	return &Inspection{conn: conn}, nil
}

func (i *Inspection) Close(ctx context.Context) error { return i.conn.Close(ctx) }

func (i *Inspection) Ping(ctx context.Context) error {
	var response int

	if err := i.conn.QueryRow(ctx, "SELECT 1").Scan(&response); err != nil {
		return fmt.Errorf("ping source: %w", err)
	}

	return nil
}

func closeInspection(ctx context.Context, inspection *Inspection, logger *slog.Logger) {
	if err := inspection.Close(ctx); err != nil {
		logger.WarnContext(ctx, "failed to close inspection", slog.Any("error", err))
	}
}
