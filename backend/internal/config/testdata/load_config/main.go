//go:build testhelpers

// load_config loads the test configuration from a binary that is not a go test binary, as
// cmd/cleanup_test_db does, and prints the database it got.
package main

import (
	"fmt"

	"databasus-backend/internal/config"
	"databasus-backend/internal/lib/testdb"
)

func main() {
	fmt.Println("loading")
	fmt.Println("loaded " + testdb.GetDatabaseName(config.GetEnv().DatabaseDsn))
}
