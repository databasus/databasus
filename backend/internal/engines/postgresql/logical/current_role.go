package logical

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func getCurrentRoleName(ctx context.Context, conn *pgx.Conn) (string, error) {
	var roleName string
	if err := conn.QueryRow(ctx, "SELECT current_user").Scan(&roleName); err != nil {
		return "", fmt.Errorf("failed to read the current role: %w", err)
	}

	return roleName, nil
}
