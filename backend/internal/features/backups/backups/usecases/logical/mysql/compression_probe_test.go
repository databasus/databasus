package usecases_logical_mysql

import (
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"

	"databasus-backend/internal/config"
	mysql_engine "databasus-backend/internal/engines/mysql"
	mysqltypes "databasus-backend/internal/features/databases/databases/mysql"
	"databasus-backend/internal/util/testing/containers"
)

func Test_BuildMysqldumpArgs_ForEverySupportedVersion_OmitsCompressionFlags(t *testing.T) {
	for _, version := range []mysql_engine.Version{
		mysql_engine.Version57,
		mysql_engine.Version80,
		mysql_engine.Version84,
		mysql_engine.Version9,
	} {
		uc := &CreateMysqlBackupUsecase{}
		database := &mysqltypes.MysqlDatabase{Version: version}

		args := uc.buildMysqldumpArgs(database)

		for _, arg := range args {
			if strings.Contains(arg, "compress") {
				t.Fatalf("compression is negotiated by the probe, not the arg builder; "+
					"version %s produced %q", version, arg)
			}
		}
	}
}

func Test_GetNetworkCompressionCandidates_OnMysql80AndNewer_TriesZstdThenZlibThenUncompressed(
	t *testing.T,
) {
	for _, version := range []mysql_engine.Version{
		mysql_engine.Version80,
		mysql_engine.Version84,
		mysql_engine.Version9,
	} {
		candidates := getNetworkCompressionCandidates(version)

		if len(candidates) != 3 {
			t.Fatalf("expected 3 candidates for %s, got %v", version, candidates)
		}

		if !slices.Contains(candidates[0], "--compression-algorithms=zstd") {
			t.Fatalf("expected zstd first for %s, got %v", version, candidates[0])
		}

		if !slices.Contains(candidates[1], "--compression-algorithms=zlib") {
			t.Fatalf("expected zlib second for %s, got %v", version, candidates[1])
		}

		if len(candidates[2]) != 0 {
			t.Fatalf("expected uncompressed last for %s, got %v", version, candidates[2])
		}
	}
}

func Test_GetNetworkCompressionCandidates_OnMysql57_TriesCompressThenUncompressed(t *testing.T) {
	candidates := getNetworkCompressionCandidates(mysql_engine.Version57)

	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %v", candidates)
	}

	if !slices.Equal(candidates[0], []string{"--compress"}) {
		t.Fatalf("expected --compress first, got %v", candidates[0])
	}

	if len(candidates[1]) != 0 {
		t.Fatalf("expected uncompressed last, got %v", candidates[1])
	}
}

func Test_GetNetworkCompressionCandidates_OnMysql57_OmitsCompressionAlgorithms(t *testing.T) {
	for _, candidate := range getNetworkCompressionCandidates(mysql_engine.Version57) {
		for _, arg := range candidate {
			if strings.HasPrefix(arg, "--compression-algorithms") {
				t.Fatalf("the 5.7 client has no --compression-algorithms, got %q", arg)
			}
		}
	}
}

func Test_GetNetworkCompressionLabel_WhenCandidateIsEmpty_ReadsUncompressed(t *testing.T) {
	if label := getNetworkCompressionLabel(nil); label != "uncompressed" {
		t.Fatalf("expected \"uncompressed\", got %q", label)
	}
}

func Test_IsCompressionRejection_OnError2066_ReportsRejection(t *testing.T) {
	rejectionStderr := "mysqldump: Got error: 2066: Connection failed due to wrongly " +
		"configured compression algorithm when trying to connect"

	if !isCompressionRejection(rejectionStderr) {
		t.Fatalf("expected error 2066 to read as a compression rejection, got %q", rejectionStderr)
	}
}

func Test_IsCompressionRejection_OnUnrelatedFailure_ReportsNoRejection(t *testing.T) {
	for _, unrelatedStderr := range []string{
		"mysqldump: Got error: 1045: Access denied for user 'app'@'%' when trying to connect",
		"mysqldump: Couldn't find table: \"__databasus_compression_probe__\"",
	} {
		if isCompressionRejection(unrelatedStderr) {
			t.Fatalf("expected no compression rejection for %q", unrelatedStderr)
		}
	}
}

func probeCompressionAgainstMysql(t *testing.T, endpoint containers.Endpoint) []string {
	t.Helper()

	databaseName := containers.MysqlDatabase
	database := &mysqltypes.MysqlDatabase{
		Version:  mysql_engine.Version80,
		Host:     endpoint.Host,
		Port:     endpoint.Port,
		Username: containers.MysqlUsername,
		Database: &databaseName,
	}
	uc := &CreateMysqlBackupUsecase{logger: slog.New(slog.NewTextHandler(os.Stdout, nil))}

	myCnfFile, err := uc.createTempMyCnfFile(database, containers.MysqlPassword)
	if err != nil {
		t.Fatalf("failed to create .my.cnf: %v", err)
	}

	t.Cleanup(func() { _ = os.RemoveAll(filepath.Dir(myCnfFile)) })

	mysqldumpBin, err := mysql_engine.ClientTools{
		Root: config.GetEnv().ClientToolsDir,
	}.GetExecutable(
		mysql_engine.Version80,
		mysql_engine.ExecutableMysqldump,
	)
	if err != nil {
		t.Fatalf("failed to resolve mysqldump: %v", err)
	}

	return uc.probeNetworkCompressionArgs(t.Context(), CompressionProbeSpec{
		MysqldumpBin: mysqldumpBin,
		MyCnfFile:    myCnfFile,
		DatabaseName: databaseName,
		DatabaseID:   uuid.New(),
	}, mysql_engine.Version80)
}

func Test_ProbeNetworkCompressionArgs_AgainstStockMysql80_SelectsZstd(t *testing.T) {
	compressionArgs := probeCompressionAgainstMysql(t, containers.StartMysql(t, "mysql:8.0"))

	if !slices.Contains(compressionArgs, "--compression-algorithms=zstd") {
		t.Fatalf("expected zstd against a stock MySQL 8.0, got %v", compressionArgs)
	}
}

func Test_ProbeNetworkCompressionArgs_WhenServerRefusesCompression_FallsBackToUncompressed(
	t *testing.T,
) {
	compressionArgs := probeCompressionAgainstMysql(
		t, containers.StartMysqlWithoutCompression(t, "mysql:8.0"),
	)

	if len(compressionArgs) != 0 {
		t.Fatalf("expected the uncompressed fallback, got %v", compressionArgs)
	}
}
