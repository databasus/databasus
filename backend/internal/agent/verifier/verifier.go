// Package verifier runs the post-restore check tiers against the restored
// container over the host-published port. It returns stats and an error; it
// does NOT classify the failure — the runner decides (a dead connection is
// agent infra, a reachable DB with an impossible tier-1 result is a bad
// backup).
package verifier

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"

	"databasus-backend/internal/agent/dbconn"
)

const (
	queryTimeout          = 30 * time.Second
	probeTimeout          = 5 * time.Second
	minimumAnalyzeTimeout = 60 * time.Second
	analyzeTimeoutMax     = 10 * time.Minute

	excludedSchemas = "'pg_catalog','information_schema','pg_toast'"
	// information_schema.schemata on PG 12 exposes per-session pg_temp_N and
	// pg_toast_temp_N namespaces that PG 13+ hides; the regex filters them so
	// SchemaCount / TableCount / tableStats stay consistent across majors.
	excludedSchemaRegex = `^pg_(temp|toast_temp)_`

	dbSizeSQL      = `SELECT pg_database_size(current_database())`
	schemaCountSQL = `SELECT count(*) FROM information_schema.schemata ` +
		`WHERE schema_name NOT IN (` + excludedSchemas + `) ` +
		`AND schema_name !~ '` + excludedSchemaRegex + `'`
	tableCountSQL = `SELECT count(*) FROM information_schema.tables ` +
		`WHERE table_schema NOT IN (` + excludedSchemas + `) ` +
		`AND table_schema !~ '` + excludedSchemaRegex + `' ` +
		`AND table_type = 'BASE TABLE'`
	tableStatsSQL = `SELECT n.nspname, c.relname, c.reltuples::bigint ` +
		`FROM pg_class c JOIN pg_namespace n ON c.relnamespace = n.oid ` +
		`WHERE c.relkind = 'r' ` +
		`AND n.nspname NOT IN (` + excludedSchemas + `) ` +
		`AND n.nspname !~ '` + excludedSchemaRegex + `' ` +
		`ORDER BY c.reltuples DESC LIMIT 100`
	fkCountSQL = `SELECT count(*) FROM information_schema.referential_constraints`
)

type TableStat struct {
	SchemaName string
	Name       string
	RowCount   int64
}

type Stats struct {
	DBSizeBytes int64
	SchemaCount int
	TableCount  int
	TableStats  []TableStat
	FKCount     int
}

type Verifier struct {
	analyzeTimeoutMax time.Duration
	log               *slog.Logger
}

func NewVerifier(log *slog.Logger) *Verifier {
	return &Verifier{analyzeTimeoutMax: analyzeTimeoutMax, log: log}
}

func (v *Verifier) CollectStats(ctx context.Context, conn dbconn.Conn) (Stats, error) {
	pgConn, err := pgx.Connect(ctx, conn.GetDSN())
	if err != nil {
		return Stats{}, fmt.Errorf("connect restored db: %w", err)
	}
	defer func() { _ = pgConn.Close(ctx) }()

	var stats Stats

	if err := collectRequiredStat(ctx, pgConn, dbSizeSQL, &stats.DBSizeBytes); err != nil {
		return Stats{}, fmt.Errorf("tier 1 database size: %w", err)
	}

	collectOptionalStat(ctx, v.log, pgConn, schemaCountSQL, &stats.SchemaCount, "tier 2 schema count unavailable")
	collectOptionalStat(ctx, v.log, pgConn, tableCountSQL, &stats.TableCount, "tier 3 table count unavailable")

	stats.TableStats = v.collectTableStats(ctx, pgConn, stats.DBSizeBytes)

	collectOptionalStat(ctx, v.log, pgConn, fkCountSQL, &stats.FKCount, "tier 5 fk metadata unavailable")

	return stats, nil
}

// Best-effort: any failure yields nil, never an error — a large DB whose
// ANALYZE exceeds the budget is expected, not a verification failure.
func (v *Verifier) collectTableStats(
	ctx context.Context,
	pgConn *pgx.Conn,
	dbSizeBytes int64,
) []TableStat {
	analyzeCtx, cancel := context.WithTimeout(
		ctx, computeAnalyzeTimeout(dbSizeBytes, v.analyzeTimeoutMax))
	defer cancel()

	if _, err := pgConn.Exec(analyzeCtx, "ANALYZE"); err != nil {
		v.log.WarnContext(ctx, "tier 4 skipped: ANALYZE failed or timed out", slog.Any("error", err))
		return nil
	}

	queryCtx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	rows, err := pgConn.Query(queryCtx, tableStatsSQL)
	if err != nil {
		v.log.WarnContext(ctx, "tier 4 table stats unavailable", slog.Any("error", err))
		return nil
	}
	defer rows.Close()

	var tableStats []TableStat
	for rows.Next() {
		var stat TableStat
		if err := rows.Scan(&stat.SchemaName, &stat.Name, &stat.RowCount); err != nil {
			v.log.WarnContext(ctx, "tier 4 row scan failed", slog.Any("error", err))
			return nil
		}

		tableStats = append(tableStats, stat)
	}

	if err := rows.Err(); err != nil {
		v.log.WarnContext(ctx, "tier 4 row iteration failed", slog.Any("error", err))
		return nil
	}

	return tableStats
}

// ProbeConnAlive is the runner's disambiguator: a failed tier with a dead
// connection is agent infra (retry), a failed tier on a live connection is a
// broken restored DB (terminal).
func ProbeConnAlive(ctx context.Context, conn dbconn.Conn) bool {
	probeCtx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()

	pgConn, err := pgx.Connect(probeCtx, conn.GetDSN())
	if err != nil {
		return false
	}
	defer func() { _ = pgConn.Close(probeCtx) }()

	return pgConn.Ping(probeCtx) == nil
}

func collectRequiredStat[T any](ctx context.Context, pgConn *pgx.Conn, sql string, dest *T) error {
	queryCtx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	return pgConn.QueryRow(queryCtx, sql).Scan(dest)
}

func collectOptionalStat[T any](
	ctx context.Context,
	log *slog.Logger,
	pgConn *pgx.Conn,
	sql string,
	dest *T,
	label string,
) {
	if err := collectRequiredStat(ctx, pgConn, sql, dest); err != nil {
		log.WarnContext(ctx, "optional statistic unavailable", slog.Any("statistic", label), slog.Any("error", err))
	}
}

// computeAnalyzeTimeout scales the ANALYZE budget with database size: a flat
// 30s makes tier-4 stats reliably absent on exactly the large databases they
// matter for.
func computeAnalyzeTimeout(dbSizeBytes int64, maxTimeout time.Duration) time.Duration {
	const bytesPerGB = 1024 * 1024 * 1024
	const perGB = 30 * time.Second

	gb := float64(dbSizeBytes) / bytesPerGB

	scaled := minimumAnalyzeTimeout + time.Duration(gb*float64(perGB))
	if scaled < minimumAnalyzeTimeout {
		return minimumAnalyzeTimeout
	}

	if scaled > maxTimeout {
		return maxTimeout
	}

	return scaled
}
