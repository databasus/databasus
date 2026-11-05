package testdb

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var testDatabaseNamePattern = regexp.MustCompile(`^databasus_test_([0-9]+)_[0-9a-f]{8}$`)

func newTestDatabaseName(tb testing.TB) string {
	tb.Helper()

	databaseName, err := generateTestDatabaseName(time.Now())
	require.NoError(tb, err)

	return databaseName
}

// pg_database stores no creation time, so the name carries it for the abandoned-database cleanup.
func generateTestDatabaseName(createdAt time.Time) (string, error) {
	randomBytes := make([]byte, 4)

	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", fmt.Errorf("generate a test database name: %w", err)
	}

	return testDatabasePrefix + strconv.FormatInt(createdAt.Unix(), 10) + "_" + hex.EncodeToString(randomBytes), nil
}

func getTestDatabaseCreationTime(databaseName string) (time.Time, bool) {
	nameMatch := testDatabaseNamePattern.FindStringSubmatch(databaseName)
	if nameMatch == nil {
		return time.Time{}, false
	}

	createdAtSeconds, err := strconv.ParseInt(nameMatch[1], 10, 64)
	if err != nil {
		return time.Time{}, false
	}

	return time.Unix(createdAtSeconds, 0), true
}
