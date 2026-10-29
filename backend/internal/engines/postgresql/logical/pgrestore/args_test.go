package pgrestore

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildArgs(t *testing.T) {
	t.Parallel()
	for _, testCase := range []struct {
		name                string
		parallelJobs        int
		isTimescaleDB       bool
		isRestoreOwnership  bool
		isRestorePrivileges bool
		isVerbose           bool
		tocListPath         string
		archivePath         string
		args                []string
	}{
		{
			name:         "builds a stream restore with ownership and privileges disabled",
			parallelJobs: 1, isVerbose: true,
			args: []string{"-Fc", "--no-password", "-h", "db.example.com", "-p", "5433", "-U", "backup_user", "-d", "orders", "--verbose", "--clean", "--if-exists", "--no-owner", "--no-acl"},
		},
		{
			name:         "caps file restore jobs when sixteen jobs are requested",
			parallelJobs: 16, isVerbose: true, isRestoreOwnership: true, isRestorePrivileges: true, archivePath: "/restore/backup.dump",
			args: []string{"-Fc", "-j", "8", "--no-password", "-h", "db.example.com", "-p", "5433", "-U", "backup_user", "-d", "orders", "--verbose", "--clean", "--if-exists", "/restore/backup.dump"},
		},
		{
			name:         "places the table of contents before the archive with one job",
			parallelJobs: 1, isVerbose: true, tocListPath: "/restore/toc.list", archivePath: "/restore/backup.dump",
			args: []string{"-Fc", "--no-password", "-h", "db.example.com", "-p", "5433", "-U", "backup_user", "-d", "orders", "--verbose", "--clean", "--if-exists", "--no-owner", "--no-acl", "-L", "/restore/toc.list", "/restore/backup.dump"},
		},
		{
			name:         "omits cleaning and parallel jobs when restoring TimescaleDB",
			parallelJobs: 16, isTimescaleDB: true, isVerbose: true, archivePath: "/restore/backup.dump",
			args: []string{"-Fc", "--no-password", "-h", "db.example.com", "-p", "5433", "-U", "backup_user", "-d", "orders", "--verbose", "--no-owner", "--no-acl", "/restore/backup.dump"},
		},
		{
			name:         "omits verbose and parallel jobs with verification settings",
			parallelJobs: 1, archivePath: "/restore/backup.dump",
			args: []string{"-Fc", "--no-password", "-h", "db.example.com", "-p", "5433", "-U", "backup_user", "-d", "orders", "--clean", "--if-exists", "--no-owner", "--no-acl", "/restore/backup.dump"},
		},
		{
			name:         "keeps ownership when privileges are disabled",
			parallelJobs: 1, isRestoreOwnership: true,
			args: []string{"-Fc", "--no-password", "-h", "db.example.com", "-p", "5433", "-U", "backup_user", "-d", "orders", "--clean", "--if-exists", "--no-acl"},
		},
		{
			name:         "keeps privileges when ownership is disabled",
			parallelJobs: 1, isRestorePrivileges: true,
			args: []string{"-Fc", "--no-password", "-h", "db.example.com", "-p", "5433", "-U", "backup_user", "-d", "orders", "--clean", "--if-exists", "--no-owner"},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			args := BuildArgs(ArgsSpec{
				Host: "db.example.com", Port: 5433, User: "backup_user", Database: "orders",
				ParallelJobs: testCase.parallelJobs, IsTimescaleDB: testCase.isTimescaleDB,
				IsRestoreOwnership: testCase.isRestoreOwnership, IsRestorePrivileges: testCase.isRestorePrivileges,
				IsVerbose: testCase.isVerbose, TocListPath: testCase.tocListPath, ArchivePath: testCase.archivePath,
			})
			assert.Equal(t, testCase.args, args)
		})
	}
}

func TestGetParallelJobs(t *testing.T) {
	t.Parallel()
	for _, testCase := range []struct {
		name          string
		requestedJobs int
		isTimescaleDB bool
		parallelJobs  int
	}{
		{name: "returns one when the request is negative", requestedJobs: -1, parallelJobs: 1},
		{name: "returns one when no jobs are requested", requestedJobs: 0, parallelJobs: 1},
		{name: "keeps the request when it is within the limit", requestedJobs: 4, parallelJobs: 4},
		{name: "caps jobs when the request exceeds eight", requestedJobs: 16, parallelJobs: 8},
		{name: "returns one when the target uses TimescaleDB", requestedJobs: 16, isTimescaleDB: true, parallelJobs: 1},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, testCase.parallelJobs, GetParallelJobs(testCase.requestedJobs, testCase.isTimescaleDB))
		})
	}
}
