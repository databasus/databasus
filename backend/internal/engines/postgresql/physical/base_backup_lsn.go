package physical

import (
	"bufio"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
)

var pgBasebackupStartPointRegexp = regexp.MustCompile(
	`write-ahead log start point:\s+([0-9A-Fa-f]+/[0-9A-Fa-f]+)\s+on timeline\s+(\d+)`,
)

// pgBasebackupStopPointRegexp parses lines like
// "pg_basebackup: ... write-ahead log end point: 0/3000220".
var pgBasebackupStopPointRegexp = regexp.MustCompile(
	`write-ahead log end point:\s+([0-9A-Fa-f]+/[0-9A-Fa-f]+)`,
)

func parseLsnsFromStderr(stderr []byte) (walmath.LSN, walmath.LSN, int, error) {
	scanner := bufio.NewScanner(strings.NewReader(string(stderr)))

	var (
		startLSN, stopLSN walmath.LSN
		timelineID        int
		gotStart, gotStop bool
	)

	for scanner.Scan() {
		line := scanner.Text()

		if matches := pgBasebackupStartPointRegexp.FindStringSubmatch(line); matches != nil {
			parsed, err := walmath.ParseLSN(matches[1])
			if err != nil {
				return 0, 0, 0, fmt.Errorf("parse start LSN: %w", err)
			}

			tli, err := strconv.Atoi(matches[2])
			if err != nil {
				return 0, 0, 0, fmt.Errorf("parse timeline ID: %w", err)
			}

			startLSN = parsed
			timelineID = tli
			gotStart = true
		}

		if matches := pgBasebackupStopPointRegexp.FindStringSubmatch(line); matches != nil {
			parsed, err := walmath.ParseLSN(matches[1])
			if err != nil {
				return 0, 0, 0, fmt.Errorf("parse stop LSN: %w", err)
			}

			stopLSN = parsed
			gotStop = true
		}
	}

	if !gotStart || !gotStop {
		return 0, 0, 0, errors.New("pg_basebackup did not emit both start and end points")
	}

	return startLSN, stopLSN, timelineID, nil
}
