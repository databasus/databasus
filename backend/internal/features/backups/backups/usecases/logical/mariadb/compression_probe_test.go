package usecases_logical_mariadb

import (
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"

	"databasus-backend/internal/config"
	mariadb_engine "databasus-backend/internal/engines/mariadb"
	mariadbtypes "databasus-backend/internal/features/databases/databases/mariadb"
	"databasus-backend/internal/util/testing/containers"
)

func Test_BuildMariadbDumpArgs_WhenCalled_OmitsCompressionFlags(t *testing.T) {
	uc := &CreateMariadbBackupUsecase{}
	database := &mariadbtypes.MariadbDatabase{Version: mariadb_engine.Version1011}

	args := uc.buildMariadbDumpArgs(database)

	for _, arg := range args {
		if strings.Contains(arg, "compress") {
			t.Fatalf("compression is negotiated by the probe, not the arg builder, got %q", arg)
		}
	}
}

func Test_NetworkCompressionCandidates_WhenRead_TryCompressThenUncompressed(t *testing.T) {
	if len(networkCompressionCandidates) != 2 {
		t.Fatalf("expected 2 candidates, got %v", networkCompressionCandidates)
	}

	if !slices.Equal(networkCompressionCandidates[0], []string{"--compress"}) {
		t.Fatalf("expected --compress first, got %v", networkCompressionCandidates[0])
	}

	if len(networkCompressionCandidates[1]) != 0 {
		t.Fatalf("expected uncompressed last, got %v", networkCompressionCandidates[1])
	}
}

func Test_GetNetworkCompressionLabel_WhenCandidateIsEmpty_ReadsUncompressed(t *testing.T) {
	if label := getNetworkCompressionLabel(nil); label != "uncompressed" {
		t.Fatalf("expected \"uncompressed\", got %q", label)
	}
}

func Test_IsCompressionRejection_OnError2066_ReportsRejection(t *testing.T) {
	rejectionStderr := "mariadb-dump: Got error: 2066: Connection failed due to wrongly " +
		"configured compression algorithm when trying to connect"

	if !isCompressionRejection(rejectionStderr) {
		t.Fatalf("expected error 2066 to read as a compression rejection, got %q", rejectionStderr)
	}
}

func Test_IsCompressionRejection_OnUnrelatedFailure_ReportsNoRejection(t *testing.T) {
	for _, unrelatedStderr := range []string{
		"mariadb-dump: Got error: 1045: Access denied for user 'app'@'%' when trying to connect",
		"mariadb-dump: Couldn't find table: \"__databasus_compression_probe__\"",
	} {
		if isCompressionRejection(unrelatedStderr) {
			t.Fatalf("expected no compression rejection for %q", unrelatedStderr)
		}
	}
}

func Test_ProbeNetworkCompressionArgs_AgainstStockMariadb_SelectsCompress(t *testing.T) {
	endpoint := containers.StartMariadb(t, "mariadb:10.11")

	databaseName := containers.MariadbDatabase
	database := &mariadbtypes.MariadbDatabase{
		Version:  mariadb_engine.Version1011,
		Host:     endpoint.Host,
		Port:     endpoint.Port,
		Username: containers.MariadbUsername,
		Database: &databaseName,
	}
	uc := &CreateMariadbBackupUsecase{logger: slog.New(slog.NewTextHandler(os.Stdout, nil))}

	myCnfFile, err := uc.createTempMyCnfFile(database, containers.MariadbPassword)
	if err != nil {
		t.Fatalf("failed to create .my.cnf: %v", err)
	}

	t.Cleanup(func() { _ = os.RemoveAll(filepath.Dir(myCnfFile)) })

	mariadbDumpBin, err := mariadb_engine.ClientTools{Root: config.GetEnv().ClientToolsDir}.GetExecutable(
		mariadb_engine.Version1011, mariadb_engine.ExecutableMariadbDump,
	)
	if err != nil {
		t.Fatalf("failed to resolve mariadb-dump: %v", err)
	}

	compressionArgs := uc.probeNetworkCompressionArgs(t.Context(), CompressionProbeSpec{
		MariadbDumpBin: mariadbDumpBin,
		MyCnfFile:      myCnfFile,
		DatabaseName:   databaseName,
		DatabaseID:     uuid.New(),
	})

	if !slices.Contains(compressionArgs, "--compress") {
		t.Fatalf("expected --compress against a stock MariaDB, got %v", compressionArgs)
	}
}
