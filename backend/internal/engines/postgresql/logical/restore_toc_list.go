package logical

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	pgshared "databasus-backend/internal/engines/postgresql/shared"
)

type tocListRun struct {
	pgRestore   string
	archivePath string
	tempDir     string
	credentials *pgshared.CredentialTempFiles
}

// writeFilteredTocList writes the archive's table of contents without the entries the spec skips,
// for pg_restore -L, and returns the list's path.
func (s *Source) writeFilteredTocList(ctx context.Context, spec RestoreSpec, run tocListRun) (string, error) {
	spec.Logger.InfoContext(ctx, "generating filtered TOC list", slog.String("backup_file", run.archivePath))

	var tocList bytes.Buffer

	err := s.runClientTool(ctx, clientToolRun{
		path:        run.pgRestore,
		args:        []string{"-l", run.archivePath},
		credentials: run.credentials,
		stdin:       nil,
		stdout:      &tocList,
		stderr:      nil,
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate TOC list: %w", err)
	}

	filteredLines := filterTocList(ctx, spec, tocList.String())

	tocListPath := filepath.Join(run.tempDir, "pg_restore_toc.list")

	err = os.WriteFile(tocListPath, []byte(strings.Join(filteredLines, "\n")), 0o600)
	if err != nil {
		return "", fmt.Errorf("failed to write TOC list file: %w", err)
	}

	spec.Logger.InfoContext(ctx, "generated filtered TOC list file",
		slog.String("toc_file", tocListPath),
		slog.Int("original_lines", len(strings.Split(tocList.String(), "\n"))),
		slog.Int("filtered_lines", len(filteredLines)),
	)

	return tocListPath, nil
}

func filterTocList(ctx context.Context, spec RestoreSpec, tocList string) []string {
	var filteredLines []string

	for line := range strings.SplitSeq(tocList, "\n") {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" {
			continue
		}

		upperLine := strings.ToUpper(trimmedLine)

		if spec.IsExcludeExtensions && isExtensionTocEntry(upperLine) {
			spec.Logger.InfoContext(
				ctx,
				"excluding extension-related entry from restore",
				slog.String("toc_line", trimmedLine),
			)

			continue
		}

		if spec.IsSkipUserMappings && isUserMappingTocEntry(upperLine) {
			spec.Logger.InfoContext(
				ctx,
				"excluding user mapping entry from restore",
				slog.String("toc_line", trimmedLine),
			)

			continue
		}

		filteredLines = append(filteredLines, line)
	}

	return filteredLines
}

// " EXTENSION " catches both CREATE EXTENSION ("3420; 0 0 EXTENSION - uuid-ossp") and
// COMMENT ON EXTENSION ("3462; 0 0 COMMENT - EXTENSION "uuid-ossp"") entries.
func isExtensionTocEntry(upperLine string) bool {
	return strings.Contains(upperLine, " EXTENSION ")
}

// " USER MAPPING " catches CREATE USER MAPPING entries
// ("18239; 0 0 USER MAPPING - platform SERVER oracle_tb_server").
func isUserMappingTocEntry(upperLine string) bool {
	return strings.Contains(upperLine, " USER MAPPING ")
}
