package disk

import (
	"context"
	"path/filepath"

	"databasus-backend/internal/config"
	"databasus-backend/internal/lib/diskusage"
)

type DiskService struct{}

func (s *DiskService) GetDiskUsage(_ context.Context) (*DiskUsage, error) {
	// The data folder is /databasus-data/backups, and the volume is mounted at /databasus-data.
	volumeUsage, err := diskusage.Get(filepath.Dir(config.GetEnv().DataFolder))
	if err != nil {
		return nil, err
	}

	return &DiskUsage{
		TotalSpaceBytes: volumeUsage.TotalBytes,
		UsedSpaceBytes:  volumeUsage.UsedBytes,
		FreeSpaceBytes:  volumeUsage.FreeBytes,
	}, nil
}
