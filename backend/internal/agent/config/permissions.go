package config

import (
	"fmt"
	"os"
	"os/user"
	"strconv"
	"syscall"
)

type PermissionWarning struct {
	Path      string
	Mode      os.FileMode
	Owner     string
	OwnerUID  int
	AgentUser string
	AgentUID  int
	Err       error
}

func EnsurePrivateFile(path string, chmod func(string, os.FileMode) error) (*PermissionWarning, error) {
	fileInfo, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read config file permissions: %w", err)
	}

	if fileInfo.Mode().Perm()&0o077 == 0 {
		return nil, nil
	}

	if err := chmod(path, 0o600); err != nil {
		owner := "unknown"
		ownerUID := -1
		if stat, isUnixStat := fileInfo.Sys().(*syscall.Stat_t); isUnixStat {
			ownerUID = int(stat.Uid)
			owner = getUsername(strconv.FormatUint(uint64(stat.Uid), 10))
		}
		return &PermissionWarning{
			Path: path, Mode: fileInfo.Mode().Perm(), Owner: owner,
			OwnerUID: ownerUID, AgentUID: os.Getuid(),
			AgentUser: getUsername(strconv.Itoa(os.Getuid())), Err: err,
		}, nil
	}
	return nil, nil
}

func getUsername(uid string) string {
	account, err := user.LookupId(uid)
	if err != nil {
		return uid
	}
	return account.Username
}
