// Package diskusage reads how much space the filesystem that holds a path has.
package diskusage

import (
	"fmt"

	"github.com/shirou/gopsutil/v4/disk"
)

type Usage struct {
	TotalBytes int64
	UsedBytes  int64
	FreeBytes  int64
}

// Get reads the usage of the filesystem that holds path.
func Get(path string) (Usage, error) {
	pathUsage, err := disk.Usage(path)
	if err != nil {
		return Usage{}, fmt.Errorf("read disk usage of %s: %w", path, err)
	}

	return Usage{
		TotalBytes: int64(pathUsage.Total),
		UsedBytes:  int64(pathUsage.Used),
		FreeBytes:  int64(pathUsage.Free),
	}, nil
}
