package containers

import (
	"strings"

	"github.com/testcontainers/testcontainers-go"
)

func summarizerInitScript(shouldSummarizeWal bool) testcontainers.ContainerFile {
	value := "off"
	if shouldSummarizeWal {
		value = "on"
	}
	return testcontainers.ContainerFile{
		Reader: strings.NewReader(
			"ALTER SYSTEM SET summarize_wal='" + value + "';\nSELECT pg_reload_conf();\n",
		),
		ContainerFilePath: "/docker-entrypoint-initdb.d/00_summarizer.sql",
		FileMode:          0o644,
	}
}
