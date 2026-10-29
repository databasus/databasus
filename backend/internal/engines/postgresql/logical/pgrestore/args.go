package pgrestore

import "strconv"

type ArgsSpec struct {
	Host                string
	Port                int
	User                string
	Database            string
	ParallelJobs        int
	IsTimescaleDB       bool
	IsRestoreOwnership  bool
	IsRestorePrivileges bool
	IsVerbose           bool
	TocListPath         string
	ArchivePath         string
}

func BuildArgs(spec ArgsSpec) []string {
	args := []string{"-Fc"}
	parallelJobs := GetParallelJobs(spec.ParallelJobs, spec.IsTimescaleDB)
	if parallelJobs > 1 {
		args = append(args, "-j", strconv.Itoa(parallelJobs))
	}

	args = append(args, "--no-password", "-h", spec.Host, "-p", strconv.Itoa(spec.Port),
		"-U", spec.User, "-d", spec.Database)

	if spec.IsVerbose {
		args = append(args, "--verbose")
	}

	// --clean would drop the TimescaleDB catalog that pre_restore needs.
	if !spec.IsTimescaleDB {
		args = append(args, "--clean", "--if-exists")
	}

	if !spec.IsRestoreOwnership {
		args = append(args, "--no-owner")
	}

	if !spec.IsRestorePrivileges {
		args = append(args, "--no-acl")
	}

	if spec.TocListPath != "" {
		args = append(args, "-L", spec.TocListPath)
	}

	if spec.ArchivePath != "" {
		args = append(args, spec.ArchivePath)
	}

	return args
}

func GetParallelJobs(requestedJobs int, isTimescaleDB bool) int {
	// Parallel pg_restore loads dependent _timescaledb_catalog rows before the chunk rows
	// they reference, violating the catalog's foreign keys. TimescaleDB needs one job.
	if isTimescaleDB {
		return 1
	}

	return max(1, min(requestedJobs, 8))
}
